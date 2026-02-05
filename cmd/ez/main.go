package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/interpreter"
	"github.com/marshallburns/ez/pkg/lexer"
	"github.com/marshallburns/ez/pkg/parser"
	"github.com/marshallburns/ez/pkg/tokenizer"
	"github.com/marshallburns/ez/pkg/typechecker"
)

// Version information - injected at build time via ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
)

const asciiBanner = `
 _____ ____
| ____|__  |
|  _|   / /
| |___ / /_
|_____/____|
Programming made EZ
`

// parsedFile holds a parsed module file for two-pass type checking
type parsedFile struct {
	path     string
	source   string
	program  *ast.Program
	parser   *parser.Parser
	lexer    *lexer.Lexer
	modPath  string
	numFiles int
}

// moduleInternals stores declarations shared across files in a multi-file module
type moduleInternals struct {
	types map[string]*typechecker.Type
	funcs map[string]*typechecker.FunctionSignature
	vars  map[string]string
}

// formatLexerErrors converts lexer errors to a formatted error list.
// Returns nil if there are no errors.
func formatLexerErrors(l *lexer.Lexer, source, filename string) *errors.EZErrorList {
	if len(l.Errors()) == 0 {
		return nil
	}
	errList := errors.NewErrorList()
	for _, lexErr := range l.Errors() {
		code := errors.LookupErrorCode(lexErr.Code)
		sourceLine := errors.GetSourceLine(source, lexErr.Line)
		ezErr := errors.NewErrorWithSource(code, lexErr.Message, filename, lexErr.Line, lexErr.Column, sourceLine)
		errList.AddError(ezErr)
	}
	return errList
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getVersionString() string {
	buf := bytes.Buffer{}
	buf.WriteString(asciiBanner)
	fmt.Fprintf(&buf, "\n%s%s%s\n", errors.Bold, Version, errors.Reset)
	fmt.Fprintf(&buf, "Built: %s\n", BuildTime)

	// Always fetch fresh version info when user explicitly runs 'ez version'
	var latestVersion string

	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	release, err := fetchLatestRelease(ctx)
	if err != nil {
		// Network error - fall back to cached state if available
		state, _ := readUpdateState()
		if state != nil && state.LatestVersion != "" {
			latestVersion = state.LatestVersion
		}
	} else {
		// Update cache with fresh data
		newState := &UpdateState{
			LastCheck:     time.Now().Format("2006-01-02"),
			LatestVersion: release.TagName,
		}
		writeUpdateState(newState)
		latestVersion = release.TagName
	}

	// Always display latest version if we have it
	if latestVersion != "" {
		fmt.Fprintf(&buf, "Latest: %s\n", latestVersion)
		if isNewerVersion(Version, latestVersion) {
			fmt.Fprintf(&buf, "\nUpdate available! Run `ez update` to upgrade.\n")
		}
	}

	fmt.Fprintf(&buf, "Copyright (c) 2025-Present Marshall A Burns")
	return buf.String()
}

func checkFile(filename string) {
	// Resolve to absolute path
	absFile, err := filepath.Abs(filename)
	if err != nil {
		fmt.Printf("Error resolving path: %v\n", err)
		return
	}
	absDir := filepath.Dir(absFile)

	data, err := os.ReadFile(absFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	source := string(data)
	l := lexer.NewLexer(source)
	p := parser.NewWithSource(l, source, absFile)
	program := p.ParseProgram()

	// Check for lexer errors
	if errList := formatLexerErrors(l, source, absFile); errList != nil {
		fmt.Print(errors.FormatErrorList(errList))
		os.Exit(1)
	}

	// Check for parser errors
	if p.EZErrors().HasErrors() {
		fmt.Print(errors.FormatErrorList(p.EZErrors()))
		os.Exit(1)
	}

	// Display parser warnings
	if p.EZErrors().HasWarnings() {
		fmt.Print(errors.FormatErrorList(p.EZErrors()))
	}

	// Initialize the module loader to check imported modules
	loader := interpreter.NewModuleLoader(absDir)

	// PHASE 1: Load and typecheck all imported modules FIRST
	// This allows us to gather module function signatures before typechecking main file
	filesChecked := 0
	hasErrors := false
	importsWithAliases := collectImportsWithAliases(program, absDir, absFile)
	checked := make(map[string]bool)
	checked[absFile] = true

	// Map from module path to the alias used by the importing file
	pathToAlias := make(map[string]string)
	var modulesToCheck []string
	for _, imp := range importsWithAliases {
		modulesToCheck = append(modulesToCheck, imp.Path)
		if imp.Alias != "" {
			pathToAlias[imp.Path] = imp.Alias
		}
	}

	// Store module function signatures: alias/moduleName -> funcName -> signature
	moduleSignatures := make(map[string]map[string]*typechecker.FunctionSignature)
	// Store module type definitions: alias/moduleName -> typeName -> type
	moduleTypes := make(map[string]map[string]*typechecker.Type)
	// Store module variable/constant definitions: alias/moduleName -> varName -> type (#677)
	moduleVariables := make(map[string]map[string]string)

	// Two-pass approach for checking module files (#709, #851):
	// Pass 1: Parse ALL files from ALL modules and extract type/function declarations
	// Pass 2: Full type check with all declarations available
	// This fixes #851 where nested imports weren't available during type checking
	var allParsedFiles []parsedFile
	moduleInternalsMap := make(map[string]*moduleInternals)

	// Pass 1: Parse all module files and extract declarations
	for len(modulesToCheck) > 0 {
		// Pop first module
		modPath := modulesToCheck[0]
		modulesToCheck = modulesToCheck[1:]

		if checked[modPath] {
			continue
		}
		checked[modPath] = true

		// Load the module using the loader
		loader.SetCurrentFile(absFile) // Set context for relative imports
		mod, err := loader.Load(modPath)
		if err != nil {
			fmt.Printf("Error loading module %s: %v\n", modPath, err)
			hasErrors = true
			continue
		}

		// Initialize module-internal declarations for multi-file modules
		if len(mod.Files) > 1 {
			moduleInternalsMap[modPath] = &moduleInternals{
				types: make(map[string]*typechecker.Type),
				funcs: make(map[string]*typechecker.FunctionSignature),
				vars:  make(map[string]string),
			}
		}

		// Parse all files in this module
		for _, filePath := range mod.Files {
			if checked[filePath] {
				continue
			}

			fileData, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error reading %s: %v\n", filePath, err)
				hasErrors = true
				continue
			}

			fileSource := string(fileData)
			fileLex := lexer.NewLexer(fileSource)
			fileParser := parser.NewWithSource(fileLex, fileSource, filePath)
			fileProgram := fileParser.ParseProgram()

			// Check for lexer errors
			if errList := formatLexerErrors(fileLex, fileSource, filePath); errList != nil {
				fmt.Print(errors.FormatErrorList(errList))
				hasErrors = true
				continue
			}

			// Check for parser errors
			if fileParser.EZErrors().HasErrors() {
				fmt.Print(errors.FormatErrorList(fileParser.EZErrors()))
				hasErrors = true
				continue
			}

			allParsedFiles = append(allParsedFiles, parsedFile{
				path:     filePath,
				source:   fileSource,
				program:  fileProgram,
				parser:   fileParser,
				lexer:    fileLex,
				modPath:  modPath,
				numFiles: len(mod.Files),
			})

			// Extract types and functions from this file (lightweight pass)
			if fileProgram.Module != nil && fileProgram.Module.Name != nil {
				moduleName := fileProgram.Module.Name.Value
				if alias, hasAlias := pathToAlias[modPath]; hasAlias {
					moduleName = alias
				}
				if moduleSignatures[moduleName] == nil {
					moduleSignatures[moduleName] = make(map[string]*typechecker.FunctionSignature)
				}
				if moduleTypes[moduleName] == nil {
					moduleTypes[moduleName] = make(map[string]*typechecker.Type)
				}
				if moduleVariables[moduleName] == nil {
					moduleVariables[moduleName] = make(map[string]string)
				}

				// Extract type declarations without full type checking
				extractTc := typechecker.NewTypeChecker(fileSource, filePath)
				extractTc.SetSkipMainCheck(true)
				extractTc.RegisterDeclarations(fileProgram) // Register types/functions only

				for funcName, sig := range extractTc.GetFunctions() {
					moduleSignatures[moduleName][funcName] = sig
				}
				for typeName, t := range extractTc.GetTypes() {
					if t.Kind == typechecker.StructType || t.Kind == typechecker.EnumType {
						moduleTypes[moduleName][typeName] = t
					}
				}
				for varName, varType := range extractTc.GetVariables() {
					moduleVariables[moduleName][varName] = varType
				}

				// For multi-file modules, also store in module-internal declarations (#722)
				if len(mod.Files) > 1 {
					internals := moduleInternalsMap[modPath]
					for typeName, t := range extractTc.GetTypes() {
						if t.Kind == typechecker.StructType || t.Kind == typechecker.EnumType {
							internals.types[typeName] = t
						}
					}
					for funcName, sig := range extractTc.GetFunctions() {
						internals.funcs[funcName] = sig
					}
					for varName, varType := range extractTc.GetVariables() {
						internals.vars[varName] = varType
					}
				}
			}

			// Collect imports from this file (including aliases for nested imports) (#851)
			newImportsWithAliases := collectImportsWithAliases(fileProgram, absDir, filePath)
			for _, imp := range newImportsWithAliases {
				modulesToCheck = append(modulesToCheck, imp.Path)
				if imp.Alias != "" {
					pathToAlias[imp.Path] = imp.Alias
				}
			}
		}
	}

	// Pass 2: Full type check with ALL declarations from ALL modules available
	for _, pf := range allParsedFiles {
		if checked[pf.path] {
			continue
		}
		checked[pf.path] = true
		filesChecked++

		if pf.parser.EZErrors().HasWarnings() {
			fmt.Print(errors.FormatErrorList(pf.parser.EZErrors()))
		}

		// Type check with all module types registered
		fileTc := typechecker.NewTypeChecker(pf.source, pf.path)
		fileTc.SetSkipMainCheck(true)

		// Set current module name for same-module symbol lookup
		if pf.program.Module != nil && pf.program.Module.Name != nil {
			fileTc.SetCurrentModule(pf.program.Module.Name.Value)
		}

		// Register all collected module types for cross-module checking (#709)
		for modName, types := range moduleTypes {
			for typeName, t := range types {
				fileTc.RegisterModuleType(modName, typeName, t)
			}
		}
		for modName, funcs := range moduleSignatures {
			for funcName, sig := range funcs {
				fileTc.RegisterModuleFunction(modName, funcName, sig)
			}
		}
		for modName, vars := range moduleVariables {
			for varName, varType := range vars {
				fileTc.RegisterModuleVariable(modName, varName, varType)
			}
		}

		// For multi-file modules, register module-internal types/functions/variables (#722)
		if pf.numFiles > 1 {
			if internals, ok := moduleInternalsMap[pf.modPath]; ok {
				// Get module name for self-referencing types (e.g., testmod.Hero within testmod)
				var selfModuleName string
				if pf.program.Module != nil && pf.program.Module.Name != nil {
					selfModuleName = pf.program.Module.Name.Value
				}

				for typeName, t := range internals.types {
					fileTc.RegisterType(typeName, t)
					// Also register as module-prefixed type for self-referencing (bug #463)
					if selfModuleName != "" {
						fileTc.RegisterModuleType(selfModuleName, typeName, t)
					}
				}
				for funcName, sig := range internals.funcs {
					fileTc.RegisterFunction(funcName, sig)
				}
				for varName, varType := range internals.vars {
					fileTc.RegisterVariable(varName, varType)
				}
			}
		}

		fileTc.CheckProgram(pf.program)

		if fileTc.Errors().HasErrors() {
			fmt.Print(errors.FormatErrorList(fileTc.Errors()))
			hasErrors = true
			continue
		}

		if fileTc.Errors().HasWarnings() {
			fmt.Print(errors.FormatErrorList(fileTc.Errors()))
		}
	}

	// Print module loader warnings
	for _, warning := range loader.GetWarnings() {
		fmt.Print(errors.FormatError(warning))
	}

	if hasErrors {
		return
	}

	// PHASE 2: Type check main file with knowledge of module signatures and types
	tc := typechecker.NewTypeChecker(source, absFile)

	// Register all module function signatures for cross-module type checking
	for moduleName, funcs := range moduleSignatures {
		for funcName, sig := range funcs {
			tc.RegisterModuleFunction(moduleName, funcName, sig)
		}
	}

	// Register all module type definitions for cross-module type checking
	for moduleName, types := range moduleTypes {
		for typeName, t := range types {
			tc.RegisterModuleType(moduleName, typeName, t)
		}
	}

	// Register all module variable/constant definitions for cross-module type checking (#677)
	for moduleName, vars := range moduleVariables {
		for varName, varType := range vars {
			tc.RegisterModuleVariable(moduleName, varName, varType)
		}
	}

	tc.CheckProgram(program)

	// Check for type errors
	if tc.Errors().HasErrors() {
		fmt.Print(errors.FormatErrorList(tc.Errors()))
		os.Exit(1)
	}

	// Display type checker warnings
	if tc.Errors().HasWarnings() {
		fmt.Print(errors.FormatErrorList(tc.Errors()))
	}

	filesChecked++ // Count main file

	fmt.Printf("Check successful: %s (%d file(s) checked)\n", filename, filesChecked)
}

// checkProject checks an entire EZ project starting from main.ez
func checkProject(dir string) {
	// Resolve to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Printf("Error resolving path: %v\n", err)
		return
	}

	// Check if directory exists
	info, err := os.Stat(absDir)
	if err != nil {
		fmt.Printf("Error: directory not found: %s\n", dir)
		return
	}
	if !info.IsDir() {
		fmt.Printf("Error: not a directory: %s\n", dir)
		return
	}

	// Look for main.ez in the directory
	mainFile := filepath.Join(absDir, "main.ez")
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		fmt.Printf("Error: no main.ez found in %s\n", dir)
		fmt.Println("  = help: create a main.ez file with a main() function")
		return
	}

	// Read main.ez
	data, err := os.ReadFile(mainFile)
	if err != nil {
		fmt.Printf("Error reading main.ez: %v\n", err)
		return
	}

	source := string(data)
	l := lexer.NewLexer(source)
	p := parser.NewWithSource(l, source, mainFile)
	program := p.ParseProgram()

	// Check for lexer errors
	if errList := formatLexerErrors(l, source, mainFile); errList != nil {
		fmt.Print(errors.FormatErrorList(errList))
		return
	}

	// Check for parser errors
	if p.EZErrors().HasErrors() {
		fmt.Print(errors.FormatErrorList(p.EZErrors()))
		return
	}

	// Display parser warnings
	if p.EZErrors().HasWarnings() {
		fmt.Print(errors.FormatErrorList(p.EZErrors()))
	}

	// Type checking for main.ez
	tc := typechecker.NewTypeChecker(source, mainFile)
	tc.CheckProgram(program)

	if tc.Errors().HasErrors() {
		fmt.Print(errors.FormatErrorList(tc.Errors()))
		return
	}

	if tc.Errors().HasWarnings() {
		fmt.Print(errors.FormatErrorList(tc.Errors()))
	}

	// Initialize the module loader
	loader := interpreter.NewModuleLoader(absDir)

	// Track all modules that need to be checked
	var modulesChecked []string
	modulesChecked = append(modulesChecked, mainFile)

	// Find and check all imported modules
	hasErrors := false
	modulesToCheck := collectImports(program, absDir, mainFile)
	checked := make(map[string]bool)
	checked[mainFile] = true

	for len(modulesToCheck) > 0 {
		// Pop first module
		modPath := modulesToCheck[0]
		modulesToCheck = modulesToCheck[1:]

		if checked[modPath] {
			continue
		}
		checked[modPath] = true

		// Load the module using the loader
		loader.SetCurrentFile(mainFile) // Set context for relative imports
		mod, err := loader.Load(modPath)
		if err != nil {
			fmt.Printf("Error loading module %s: %v\n", modPath, err)
			hasErrors = true
			continue
		}

		// Check each file in the module
		for _, filePath := range mod.Files {
			if checked[filePath] {
				continue
			}
			checked[filePath] = true
			modulesChecked = append(modulesChecked, filePath)

			// Read and check the file
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error reading %s: %v\n", filePath, err)
				hasErrors = true
				continue
			}

			fileSource := string(fileData)
			fileLex := lexer.NewLexer(fileSource)
			fileParser := parser.NewWithSource(fileLex, fileSource, filePath)
			fileProgram := fileParser.ParseProgram()

			// Check for lexer errors
			if errList := formatLexerErrors(fileLex, fileSource, filePath); errList != nil {
				fmt.Print(errors.FormatErrorList(errList))
				hasErrors = true
				continue
			}

			// Check for parser errors
			if fileParser.EZErrors().HasErrors() {
				fmt.Print(errors.FormatErrorList(fileParser.EZErrors()))
				hasErrors = true
				continue
			}

			if fileParser.EZErrors().HasWarnings() {
				fmt.Print(errors.FormatErrorList(fileParser.EZErrors()))
			}

			// Type check (skip main() requirement for module files)
			fileTc := typechecker.NewTypeChecker(fileSource, filePath)
			fileTc.SetSkipMainCheck(true) // Module files don't need main()
			fileTc.CheckProgram(fileProgram)

			if fileTc.Errors().HasErrors() {
				fmt.Print(errors.FormatErrorList(fileTc.Errors()))
				hasErrors = true
				continue
			}

			if fileTc.Errors().HasWarnings() {
				fmt.Print(errors.FormatErrorList(fileTc.Errors()))
			}

			// Collect more imports from this file
			newImports := collectImports(fileProgram, absDir, filePath)
			modulesToCheck = append(modulesToCheck, newImports...)
		}
	}

	// Print module loader warnings
	for _, warning := range loader.GetWarnings() {
		fmt.Print(errors.FormatError(warning))
	}

	if hasErrors {
		fmt.Println("\nCheck failed.")
		return
	}

	fmt.Printf("Check successful: %d file(s) checked\n", len(modulesChecked))
}

// ImportInfo holds information about an import
type ImportInfo struct {
	Path  string
	Alias string
}

// collectImports extracts import paths from a program's AST
func collectImports(program *ast.Program, rootDir string, currentFile string) []string {
	importsWithAliases := collectImportsWithAliases(program, rootDir, currentFile)
	paths := make([]string, len(importsWithAliases))
	for i, imp := range importsWithAliases {
		paths[i] = imp.Path
	}
	return paths
}

// collectImportsWithAliases extracts import paths and their aliases from a program's AST
func collectImportsWithAliases(program *ast.Program, rootDir string, currentFile string) []ImportInfo {
	var imports []ImportInfo

	for _, stmt := range program.Statements {
		if importStmt, ok := stmt.(*ast.ImportStatement); ok {
			for _, item := range importStmt.Imports {
				// Skip stdlib imports (start with @)
				if strings.HasPrefix(item.Module, "@") || item.Module != "" {
					continue
				}
				// User module import (has a path)
				if item.Path != "" {
					// Resolve the path relative to current file
					basePath := filepath.Dir(currentFile)
					absPath, err := filepath.Abs(filepath.Join(basePath, item.Path))
					if err == nil {
						imports = append(imports, ImportInfo{
							Path:  absPath,
							Alias: item.Alias,
						})
					}
				}
			}
		}
	}

	return imports
}

func lexFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	l := lexer.NewLexer(string(data))

	for {
		tok := l.NextToken()
		fmt.Printf("%+v\n", tok)
		if tok.Type == tokenizer.EOF {
			break
		}
	}
}

func parse_file(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	l := lexer.NewLexer(string(data))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Parser errors:")
		for _, e := range p.Errors() {
			fmt.Printf("  %s\n", e)
		}
		return
	}

	fmt.Printf("Parsed %d statements\n", len(program.Statements))
	for i, stmt := range program.Statements {
		fmt.Printf("%d: %T - %s\n", i+1, stmt, stmt.TokenLiteral())
	}
}

func runFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// Get absolute path for module loading
	absPath, err := filepath.Abs(filename)
	if err != nil {
		fmt.Printf("Error resolving path: %v\n", err)
		return
	}

	source := string(data)
	l := lexer.NewLexer(source)
	p := parser.NewWithSource(l, source, filename)
	program := p.ParseProgram()

	// Check for lexer errors first
	if errList := formatLexerErrors(l, source, filename); errList != nil {
		fmt.Print(errors.FormatErrorList(errList))
		os.Exit(1)
	}

	// Check for parser errors using new error system
	if p.EZErrors().HasErrors() {
		fmt.Print(errors.FormatErrorList(p.EZErrors()))
		os.Exit(1)
	}

	// Display parser warnings (even if no errors)
	if p.EZErrors().HasWarnings() {
		fmt.Print(errors.FormatErrorList(p.EZErrors()))
	}

	// Initialize the module loader with the directory of the main file
	rootDir := filepath.Dir(absPath)
	loader := interpreter.NewModuleLoader(rootDir)

	// Load imported modules and extract their type definitions for cross-module type checking
	importsWithAliases := collectImportsWithAliases(program, rootDir, absPath)
	pathToAlias := make(map[string]string)
	var modulesToCheck []string
	for _, imp := range importsWithAliases {
		modulesToCheck = append(modulesToCheck, imp.Path)
		if imp.Alias != "" {
			pathToAlias[imp.Path] = imp.Alias
		}
	}

	moduleSignatures := make(map[string]map[string]*typechecker.FunctionSignature)
	moduleTypes := make(map[string]map[string]*typechecker.Type)
	moduleVariables := make(map[string]map[string]string) // (#677)
	checked := make(map[string]bool)
	checked[absPath] = true

	// Two-pass approach for checking module files (#709, #851):
	// Pass 1: Parse ALL files from ALL modules and extract type/function declarations
	// Pass 2: Full type check with all declarations available
	// This fixes #851 where nested imports weren't available during type checking
	var allParsedFiles []parsedFile
	moduleInternalsMap := make(map[string]*moduleInternals)

	// Pass 1: Parse all module files and extract declarations
	for len(modulesToCheck) > 0 {
		modPath := modulesToCheck[0]
		modulesToCheck = modulesToCheck[1:]

		if checked[modPath] {
			continue
		}
		checked[modPath] = true

		loader.SetCurrentFile(absPath)
		mod, err := loader.Load(modPath)
		if err != nil {
			// Check if this is a module error with parse errors (#726)
			if modErr, ok := err.(*interpreter.ModuleError); ok && modErr.EZErrors != nil && modErr.EZErrors.HasErrors() {
				fmt.Print(errors.FormatErrorList(modErr.EZErrors))
				os.Exit(1)
			}
			// For other errors (module not found, etc.), let them be caught at runtime
			continue
		}

		// Initialize module-internal declarations for multi-file modules
		if len(mod.Files) > 1 {
			moduleInternalsMap[modPath] = &moduleInternals{
				types: make(map[string]*typechecker.Type),
				funcs: make(map[string]*typechecker.FunctionSignature),
				vars:  make(map[string]string),
			}
		}

		// Parse all files in this module
		for _, filePath := range mod.Files {
			if checked[filePath] {
				continue
			}

			fileData, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			fileSource := string(fileData)
			fileLex := lexer.NewLexer(fileSource)
			fileParser := parser.NewWithSource(fileLex, fileSource, filePath)
			fileProgram := fileParser.ParseProgram()

			// Check for lexer errors in imported module
			if errList := formatLexerErrors(fileLex, fileSource, filePath); errList != nil {
				fmt.Print(errors.FormatErrorList(errList))
				os.Exit(1)
			}

			// Check for parser errors in imported module
			if fileParser.EZErrors().HasErrors() {
				fmt.Print(errors.FormatErrorList(fileParser.EZErrors()))
				os.Exit(1)
			}

			allParsedFiles = append(allParsedFiles, parsedFile{
				path:     filePath,
				source:   fileSource,
				program:  fileProgram,
				modPath:  modPath,
				numFiles: len(mod.Files),
			})

			// Extract types and functions from this file (lightweight pass)
			if fileProgram.Module != nil && fileProgram.Module.Name != nil {
				moduleName := fileProgram.Module.Name.Value
				if alias, hasAlias := pathToAlias[modPath]; hasAlias {
					moduleName = alias
				}
				if moduleSignatures[moduleName] == nil {
					moduleSignatures[moduleName] = make(map[string]*typechecker.FunctionSignature)
				}
				if moduleTypes[moduleName] == nil {
					moduleTypes[moduleName] = make(map[string]*typechecker.Type)
				}
				if moduleVariables[moduleName] == nil {
					moduleVariables[moduleName] = make(map[string]string)
				}

				// Extract type declarations without full type checking
				extractTc := typechecker.NewTypeChecker(fileSource, filePath)
				extractTc.SetSkipMainCheck(true)
				extractTc.RegisterDeclarations(fileProgram)

				for funcName, sig := range extractTc.GetFunctions() {
					moduleSignatures[moduleName][funcName] = sig
				}
				for typeName, t := range extractTc.GetTypes() {
					if t.Kind == typechecker.StructType || t.Kind == typechecker.EnumType {
						moduleTypes[moduleName][typeName] = t
					}
				}
				for varName, varType := range extractTc.GetVariables() {
					moduleVariables[moduleName][varName] = varType
				}

				// For multi-file modules, also store in module-internal declarations
				if len(mod.Files) > 1 {
					internals := moduleInternalsMap[modPath]
					for typeName, t := range extractTc.GetTypes() {
						if t.Kind == typechecker.StructType || t.Kind == typechecker.EnumType {
							internals.types[typeName] = t
						}
					}
					for funcName, sig := range extractTc.GetFunctions() {
						internals.funcs[funcName] = sig
					}
					for varName, varType := range extractTc.GetVariables() {
						internals.vars[varName] = varType
					}
				}
			}

			// Collect imports from this file (including aliases for nested imports) (#851)
			newImportsWithAliases := collectImportsWithAliases(fileProgram, rootDir, filePath)
			for _, imp := range newImportsWithAliases {
				modulesToCheck = append(modulesToCheck, imp.Path)
				if imp.Alias != "" {
					pathToAlias[imp.Path] = imp.Alias
				}
			}
		}
	}

	// Pass 2: Full type check with ALL declarations from ALL modules available
	for _, pf := range allParsedFiles {
		checked[pf.path] = true

		fileTc := typechecker.NewTypeChecker(pf.source, pf.path)
		fileTc.SetSkipMainCheck(true)

		// Register all collected module types for cross-module checking (#709)
		for modName, types := range moduleTypes {
			for typeName, t := range types {
				fileTc.RegisterModuleType(modName, typeName, t)
			}
		}
		for modName, funcs := range moduleSignatures {
			for funcName, sig := range funcs {
				fileTc.RegisterModuleFunction(modName, funcName, sig)
			}
		}
		for modName, vars := range moduleVariables {
			for varName, varType := range vars {
				fileTc.RegisterModuleVariable(modName, varName, varType)
			}
		}

		// For multi-file modules, register module-internal types/functions/variables (#722)
		if pf.numFiles > 1 {
			if internals, ok := moduleInternalsMap[pf.modPath]; ok {
				// Get module name for self-referencing types (e.g., testmod.Hero within testmod)
				var selfModuleName string
				if pf.program.Module != nil && pf.program.Module.Name != nil {
					selfModuleName = pf.program.Module.Name.Value
				}

				for typeName, t := range internals.types {
					fileTc.RegisterType(typeName, t)
					// Also register as module-prefixed type for self-referencing (bug #463)
					if selfModuleName != "" {
						fileTc.RegisterModuleType(selfModuleName, typeName, t)
					}
				}
				for funcName, sig := range internals.funcs {
					fileTc.RegisterFunction(funcName, sig)
				}
				for varName, varType := range internals.vars {
					fileTc.RegisterVariable(varName, varType)
				}
			}
		}

		fileTc.CheckProgram(pf.program)

		// Check for type errors in module files (#720, #722)
		if fileTc.Errors().HasErrors() {
			fmt.Print(errors.FormatErrorList(fileTc.Errors()))
			os.Exit(1)
		}

		// Display type checker warnings from module files
		if fileTc.Errors().HasWarnings() {
			fmt.Print(errors.FormatErrorList(fileTc.Errors()))
		}
	}

	// Type checking phase with module type information
	tc := typechecker.NewTypeChecker(source, filename)

	// Register module function signatures and types
	for moduleName, funcs := range moduleSignatures {
		for funcName, sig := range funcs {
			tc.RegisterModuleFunction(moduleName, funcName, sig)
		}
	}
	for moduleName, types := range moduleTypes {
		for typeName, t := range types {
			tc.RegisterModuleType(moduleName, typeName, t)
		}
	}
	// Register module variables/constants (#677)
	for moduleName, vars := range moduleVariables {
		for varName, varType := range vars {
			tc.RegisterModuleVariable(moduleName, varName, varType)
		}
	}

	tc.CheckProgram(program)

	// Check for type errors
	if tc.Errors().HasErrors() {
		fmt.Print(errors.FormatErrorList(tc.Errors()))
		os.Exit(1)
	}

	// Display type checker warnings (even if no errors)
	if tc.Errors().HasWarnings() {
		fmt.Print(errors.FormatErrorList(tc.Errors()))
	}

	// Set up the evaluation context
	ctx := &interpreter.EvalContext{
		Loader:      loader,
		CurrentFile: absPath,
	}

	env := interpreter.NewEnvironment()
	result := interpreter.Eval(program, env, ctx)

	// Print any module loading warnings
	if ctx.Loader != nil {
		for _, warning := range ctx.Loader.GetWarnings() {
			fmt.Print(errors.FormatError(warning))
		}
	}

	// Check for evaluation errors
	if errObj, ok := result.(*interpreter.Error); ok {
		printRuntimeError(errObj, source, filename)
		os.Exit(1)
	}

	// Look for main function and call it
	if mainFn, ok := env.Get("main"); ok {
		if fn, ok := mainFn.(*interpreter.Function); ok {
			fnEnv := interpreter.NewEnclosedEnvironment(env)
			mainResult := interpreter.Eval(fn.Body, fnEnv, ctx)

			// Execute ensure statements before returning (LIFO order)
			ensures := fnEnv.ExecuteEnsures()
			for _, ensureCall := range ensures {
				interpreter.Eval(ensureCall, fnEnv, ctx)
				// Note: We ignore errors from ensure statements to ensure cleanup always runs
			}
			fnEnv.ClearEnsures()

			// Check for errors from main
			if errObj, ok := mainResult.(*interpreter.Error); ok {
				printRuntimeError(errObj, source, filename)
				os.Exit(1)
			}
		}
	}
}

// printRuntimeError formats and prints a runtime error
func printRuntimeError(errObj *interpreter.Error, source, filename string) {
	// If we have location info, use formatted output
	if errObj.Code != "" && errObj.Line > 0 {
		// Find the error code - special cases for panic/assert, otherwise use lookup
		var code errors.ErrorCode
		switch errObj.Code {
		case "E5021":
			code = errors.ErrorCode{Code: "PANIC", Name: "panic", Description: "panicked here"}
		case "E5022":
			code = errors.ErrorCode{Code: "ASSERT", Name: "assertion-failed", Description: "assertion failed here"}
		default:
			code = errors.LookupErrorCode(errObj.Code)
		}

		// Use error's file if available, otherwise fall back to main file
		errorFile := filename
		errorSource := source
		if errObj.File != "" {
			errorFile = errObj.File
			// Read source from the error's file
			if data, err := os.ReadFile(errObj.File); err == nil {
				errorSource = string(data)
			}
		}
		sourceLine := errors.GetSourceLine(errorSource, errObj.Line)
		ezErr := errors.NewErrorWithSource(code, errObj.Message, errorFile, errObj.Line, errObj.Column, sourceLine)
		if errObj.Help != "" {
			ezErr.Help = errObj.Help
		}

		errList := errors.NewErrorList()
		errList.AddError(ezErr)
		fmt.Print(errors.FormatErrorList(errList))
	} else {
		// Fall back to simple output
		fmt.Println(errObj.Inspect())
	}
}
