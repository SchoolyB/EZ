package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	// Check for updates in background (non-blocking, once per day)
	CheckForUpdateAsync()

	command := os.Args[1]

	switch command {
	case "help", "-h", "--help":
		printHelp()
	case "version", "-v", "--version":
		printVersion()
	case "update":
		runUpdate()
	case "repl":
		startREPL()
	case "check", "build":
		if len(os.Args) < 3 {
			// No argument: check project in current directory
			checkProject(".")
		} else {
			arg := os.Args[2]
			// Check if it's a .ez file or a directory
			if strings.HasSuffix(arg, ".ez") {
				checkFile(arg)
			} else {
				checkProject(arg)
			}
		}
	case "lex":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ez lex <file>")
			return
		}
		lexFile(os.Args[2])
	case "parse":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ez parse <file>")
			return
		}
		parse_file(os.Args[2])
	default:
		// If it's not a known command, treat it as a file to run
		// This allows: ez myProgram.ez
		if len(command) > 3 && command[len(command)-3:] == ".ez" {
			runFile(command)
		} else {
			fmt.Printf("Unknown command: %s\n", command)
			fmt.Println("Run 'ez help' for usage information")
		}
	}
}

func printHelp() {
	fmt.Println("EZ Language Interpreter")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ez <file>           Run an EZ program")
	fmt.Println("  ez <command> [args] Run a specific command")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  check          Check syntax and types in current directory (requires main.ez)")
	fmt.Println("  check <dir>    Check syntax and types in specified directory")
	fmt.Println("  check <file>   Check syntax and types for a single file")
	fmt.Println("  repl           Start interactive REPL mode")
	fmt.Println("  update         Check for updates and upgrade EZ")
	fmt.Println("  version        Show version information")
	fmt.Println("  help           Show this help message")
	fmt.Println()
	fmt.Println("Debug Commands:")
	fmt.Println("  lex <file>     Tokenize a file")
	fmt.Println("  parse <file>   Parse a file")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ez myProgram.ez")
	fmt.Println("  ez check                    # Check project in current directory")
	fmt.Println("  ez check ./myproject        # Check project in myproject/")
	fmt.Println("  ez check utils.ez           # Check single file")
	fmt.Println("  ez repl")
}

func printVersion() {
	fmt.Printf("EZ Language %s\n", Version)
	fmt.Printf("Built: %s\n", BuildTime)
	fmt.Println("Copyright (c) 2025-Present Marshall A Burns")
}

func checkFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	source := string(data)
	l := lexer.NewLexer(source)
	p := parser.NewWithSource(l, source, filename)
	program := p.ParseProgram()

	// Check for lexer errors
	if len(l.Errors()) > 0 {
		errList := errors.NewErrorList()
		for _, lexErr := range l.Errors() {
			var code errors.ErrorCode
			switch lexErr.Code {
			case "E1005":
				code = errors.E1005
			default:
				code = errors.ErrorCode{Code: lexErr.Code, Name: "lexer-error", Description: "Lexer error"}
			}
			sourceLine := errors.GetSourceLine(source, lexErr.Line)
			ezErr := errors.NewErrorWithSource(code, lexErr.Message, filename, lexErr.Line, lexErr.Column, sourceLine)
			errList.AddError(ezErr)
		}
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

	// Type checking
	tc := typechecker.NewTypeChecker(source, filename)
	tc.CheckProgram(program)

	// Check for type errors
	if tc.Errors().HasErrors() {
		fmt.Print(errors.FormatErrorList(tc.Errors()))
		return
	}

	// Display type checker warnings
	if tc.Errors().HasWarnings() {
		fmt.Print(errors.FormatErrorList(tc.Errors()))
	}

	fmt.Printf("Check successful: %s\n", filename)
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
	if len(l.Errors()) > 0 {
		errList := errors.NewErrorList()
		for _, lexErr := range l.Errors() {
			var code errors.ErrorCode
			switch lexErr.Code {
			case "E1005":
				code = errors.E1005
			default:
				code = errors.ErrorCode{Code: lexErr.Code, Name: "lexer-error", Description: "Lexer error"}
			}
			sourceLine := errors.GetSourceLine(source, lexErr.Line)
			ezErr := errors.NewErrorWithSource(code, lexErr.Message, mainFile, lexErr.Line, lexErr.Column, sourceLine)
			errList.AddError(ezErr)
		}
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
			if len(fileLex.Errors()) > 0 {
				errList := errors.NewErrorList()
				for _, lexErr := range fileLex.Errors() {
					var code errors.ErrorCode
					switch lexErr.Code {
					case "E1005":
						code = errors.E1005
					default:
						code = errors.ErrorCode{Code: lexErr.Code, Name: "lexer-error", Description: "Lexer error"}
					}
					sourceLine := errors.GetSourceLine(fileSource, lexErr.Line)
					ezErr := errors.NewErrorWithSource(code, lexErr.Message, filePath, lexErr.Line, lexErr.Column, sourceLine)
					errList.AddError(ezErr)
				}
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
		fmt.Println(warning)
	}

	if hasErrors {
		fmt.Println("\nCheck failed.")
		return
	}

	fmt.Printf("Check successful: %d file(s) checked\n", len(modulesChecked))
}

// collectImports extracts import paths from a program's AST
func collectImports(program *ast.Program, rootDir string, currentFile string) []string {
	var imports []string

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
						imports = append(imports, absPath)
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
	if len(l.Errors()) > 0 {
		errList := errors.NewErrorList()
		for _, lexErr := range l.Errors() {
			var code errors.ErrorCode
			switch lexErr.Code {
			case "E1005":
				code = errors.E1005
			default:
				code = errors.ErrorCode{Code: lexErr.Code, Name: "lexer-error", Description: "Lexer error"}
			}
			sourceLine := errors.GetSourceLine(source, lexErr.Line)
			ezErr := errors.NewErrorWithSource(code, lexErr.Message, filename, lexErr.Line, lexErr.Column, sourceLine)
			errList.AddError(ezErr)
		}
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

	// Type checking phase
	tc := typechecker.NewTypeChecker(source, filename)
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

	// Initialize the module loader with the directory of the main file
	rootDir := filepath.Dir(absPath)
	loader := interpreter.NewModuleLoader(rootDir)

	// Set up the evaluation context
	ctx := &interpreter.EvalContext{
		Loader:      loader,
		CurrentFile: absPath,
	}
	interpreter.SetEvalContext(ctx)

	env := interpreter.NewEnvironment()
	result := interpreter.Eval(program, env)

	// Print any module loading warnings
	if ctx := interpreter.GetEvalContext(); ctx != nil && ctx.Loader != nil {
		for _, warning := range ctx.Loader.GetWarnings() {
			fmt.Println(warning)
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
			mainResult := interpreter.Eval(fn.Body, fnEnv)

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
		// Find the error code
		var code errors.ErrorCode
		switch errObj.Code {
		case "E2001":
			code = errors.E2001
		case "E2002":
			code = errors.E2002
		case "E2003":
			code = errors.E2003
		case "E2004":
			code = errors.E2004
		case "E2005":
			code = errors.E2005
		case "E3001":
			code = errors.E3001
		case "E3002":
			code = errors.E3002
		case "E3003":
			code = errors.E3003
		case "E3004":
			code = errors.E3004
		case "E3005":
			code = errors.E3005
		case "E3007":
			code = errors.E3007
		case "E4001":
			code = errors.E4001
		case "E4002":
			code = errors.E4002
		case "E4003":
			code = errors.E4003
		case "E4004":
			code = errors.E4004
		case "E4005":
			code = errors.E4005
		case "E4006":
			code = errors.E4006
		case "E4007":
			code = errors.E4007
		default:
			// Unknown code, use generic
			code = errors.ErrorCode{Code: errObj.Code, Name: "error", Description: "error occurred here"}
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
