package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/interpreter"
	"github.com/marshallburns/ez/pkg/lexer"
	"github.com/marshallburns/ez/pkg/parser"
	"github.com/marshallburns/ez/pkg/typechecker"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:               "watch [file.ez | directory]",
	Short:             "Watch files and re-run on changes",
	Long:              `Watch a file or directory for changes and automatically re-run.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: filterEzFiles,
	Run:               runWatch,
}

func runWatch(cmd *cobra.Command, args []string) {
	target := args[0]

	// Resolve to absolute path
	absTarget, err := filepath.Abs(target)
	if err != nil {
		fmt.Printf("Error resolving path: %v\n", err)
		os.Exit(1)
	}

	// Check if target exists
	info, err := os.Stat(absTarget)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if info.IsDir() {
		watchDirectory(absTarget)
	} else {
		if !strings.HasSuffix(absTarget, ".ez") {
			fmt.Println("Error: file must have .ez extension")
			os.Exit(1)
		}
		watchFile(absTarget)
	}
}

// watchFile watches a single file and its imports for changes
func watchFile(filepath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("Error creating watcher: %v\n", err)
		os.Exit(1)
	}
	defer watcher.Close()

	// Get the list of files to watch (main file + imports)
	filesToWatch := collectFilesToWatch(filepath)

	// Add all files to the watcher
	for _, f := range filesToWatch {
		if err := watcher.Add(f); err != nil {
			fmt.Printf("Error watching %s: %v\n", f, err)
		}
	}

	// Print initial message
	importCount := len(filesToWatch) - 1
	if importCount > 0 {
		fmt.Printf("Watching %s (+ %d imports)\n", shortPath(filepath), importCount)
	} else {
		fmt.Printf("Watching %s\n", shortPath(filepath))
	}
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Run initially
	executeFile(filepath)

	// Set up debouncing
	var mu sync.Mutex
	var timer *time.Timer
	debounceInterval := 100 * time.Millisecond

	// Watch for changes
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				mu.Lock()
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounceInterval, func() {
					// Refresh the watch list in case imports changed
					newFilesToWatch := collectFilesToWatch(filepath)

					// Add any new files to the watcher
					for _, f := range newFilesToWatch {
						watcher.Add(f) // Ignore errors for already-watched files
					}

					executeFile(filepath)
				})
				mu.Unlock()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Watcher error: %v\n", err)
		}
	}
}

// watchDirectory watches all .ez files in a directory and runs the file with main()
func watchDirectory(dirPath string) {
	// Find the main file
	mainFile, err := findMainFile(dirPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("Error creating watcher: %v\n", err)
		os.Exit(1)
	}
	defer watcher.Close()

	// Collect all .ez files in the directory (recursively)
	filesToWatch := collectEzFilesInDir(dirPath)

	// Add all files to the watcher
	for _, f := range filesToWatch {
		if err := watcher.Add(f); err != nil {
			fmt.Printf("Error watching %s: %v\n", f, err)
		}
	}

	// Print initial message
	fmt.Printf("Watching %s (%d files, entry: %s)\n", shortPath(dirPath), len(filesToWatch), shortPath(mainFile))
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Run initially
	executeFile(mainFile)

	// Set up debouncing
	var mu sync.Mutex
	var timer *time.Timer
	debounceInterval := 100 * time.Millisecond

	// Watch for changes
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				// Only react to .ez files
				if !strings.HasSuffix(event.Name, ".ez") {
					continue
				}
				mu.Lock()
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounceInterval, func() {
					// Refresh the watch list in case new files were added
					newFilesToWatch := collectEzFilesInDir(dirPath)
					for _, f := range newFilesToWatch {
						watcher.Add(f)
					}

					executeFile(mainFile)
				})
				mu.Unlock()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Watcher error: %v\n", err)
		}
	}
}

// collectFilesToWatch returns a list of files to watch: the main file plus all its imports
func collectFilesToWatch(mainFile string) []string {
	files := []string{mainFile}

	// Parse the main file to get imports
	data, err := os.ReadFile(mainFile)
	if err != nil {
		return files
	}

	source := string(data)
	l := lexer.NewLexer(source)
	p := parser.NewWithSource(l, source, mainFile)
	program := p.ParseProgram()

	if p.EZErrors().HasErrors() {
		return files
	}

	// Collect imports
	rootDir := filepath.Dir(mainFile)
	imports := collectImportsWithAliases(program, rootDir, mainFile)
	checked := make(map[string]bool)
	checked[mainFile] = true

	// Initialize the module loader
	loader := interpreter.NewModuleLoader(rootDir)

	// Process all imports (including transitive)
	var toProcess []ImportInfo
	toProcess = append(toProcess, imports...)

	for len(toProcess) > 0 {
		imp := toProcess[0]
		toProcess = toProcess[1:]

		if checked[imp.Path] {
			continue
		}
		checked[imp.Path] = true

		// Load the module to get its files
		loader.SetCurrentFile(mainFile)
		mod, err := loader.Load(imp.Path)
		if err != nil {
			continue
		}

		for _, filePath := range mod.Files {
			if checked[filePath] {
				continue
			}
			files = append(files, filePath)
			checked[filePath] = true

			// Parse this file to get its imports
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			fileSource := string(fileData)
			fileLex := lexer.NewLexer(fileSource)
			fileParser := parser.NewWithSource(fileLex, fileSource, filePath)
			fileProgram := fileParser.ParseProgram()

			if fileParser.EZErrors().HasErrors() {
				continue
			}

			newImports := collectImportsWithAliases(fileProgram, rootDir, filePath)
			toProcess = append(toProcess, newImports...)
		}
	}

	return files
}

// collectEzFilesInDir recursively collects all .ez files in a directory
func collectEzFilesInDir(dir string) []string {
	var files []string

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ez") {
			files = append(files, path)
		}
		return nil
	})

	return files
}

// findMainFile finds the .ez file containing the main() function in a directory
func findMainFile(dir string) (string, error) {
	ezFiles := collectEzFilesInDir(dir)

	if len(ezFiles) == 0 {
		return "", fmt.Errorf("Error: no .ez files found in %s", dir)
	}

	var mainFiles []string

	for _, filePath := range ezFiles {
		if hasMainFunction(filePath) {
			mainFiles = append(mainFiles, filePath)
		}
	}

	if len(mainFiles) == 0 {
		return "", fmt.Errorf("Error: no main() function found in %s\n  = help: create a file with a main() function", dir)
	}

	if len(mainFiles) > 1 {
		var fileList strings.Builder
		for _, f := range mainFiles {
			fileList.WriteString(fmt.Sprintf("\n    - %s", shortPath(f)))
		}
		return "", fmt.Errorf("Error: multiple main() functions found in %s:%s\n  = help: only one file should contain main()", dir, fileList.String())
	}

	return mainFiles[0], nil
}

// hasMainFunction checks if a file contains a main() function declaration
func hasMainFunction(filePath string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	source := string(data)
	l := lexer.NewLexer(source)
	p := parser.NewWithSource(l, source, filePath)
	program := p.ParseProgram()

	if p.EZErrors().HasErrors() {
		return false
	}

	for _, stmt := range program.Statements {
		if funcDecl, ok := stmt.(*ast.FunctionDeclaration); ok {
			if funcDecl.Name.Value == "main" {
				return true
			}
		}
	}

	return false
}

// executeFile runs the given file and prints output with formatting
func executeFile(filename string) {
	// Print timestamp header
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] Running %s...\n", timestamp, shortPath(filename))

	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		printSeparator()
		return
	}

	// Get absolute path for module loading
	absPath, err := filepath.Abs(filename)
	if err != nil {
		fmt.Printf("Error resolving path: %v\n", err)
		printSeparator()
		return
	}

	source := string(data)
	l := lexer.NewLexer(source)
	p := parser.NewWithSource(l, source, filename)
	program := p.ParseProgram()

	// Check for lexer errors
	if errList := formatLexerErrors(l, source, filename); errList != nil {
		printWatchError()
		fmt.Print(errors.FormatErrorList(errList))
		printSeparator()
		return
	}

	// Check for parser errors
	if p.EZErrors().HasErrors() {
		printWatchError()
		fmt.Print(errors.FormatErrorList(p.EZErrors()))
		printSeparator()
		return
	}

	// Display parser warnings
	if p.EZErrors().HasWarnings() {
		fmt.Print(errors.FormatErrorList(p.EZErrors()))
	}

	// Initialize the module loader
	rootDir := filepath.Dir(absPath)
	loader := interpreter.NewModuleLoader(rootDir)

	// Load imported modules and extract their type definitions
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
	moduleVariables := make(map[string]map[string]string)
	checked := make(map[string]bool)
	checked[absPath] = true

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
			if modErr, ok := err.(*interpreter.ModuleError); ok && modErr.EZErrors != nil && modErr.EZErrors.HasErrors() {
				printWatchError()
				fmt.Print(errors.FormatErrorList(modErr.EZErrors))
				printSeparator()
				return
			}
			continue
		}

		if len(mod.Files) > 1 {
			moduleInternalsMap[modPath] = &moduleInternals{
				types: make(map[string]*typechecker.Type),
				funcs: make(map[string]*typechecker.FunctionSignature),
				vars:  make(map[string]string),
			}
		}

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

			if errList := formatLexerErrors(fileLex, fileSource, filePath); errList != nil {
				printWatchError()
				fmt.Print(errors.FormatErrorList(errList))
				printSeparator()
				return
			}

			if fileParser.EZErrors().HasErrors() {
				printWatchError()
				fmt.Print(errors.FormatErrorList(fileParser.EZErrors()))
				printSeparator()
				return
			}

			allParsedFiles = append(allParsedFiles, parsedFile{
				path:     filePath,
				source:   fileSource,
				program:  fileProgram,
				modPath:  modPath,
				numFiles: len(mod.Files),
			})

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

			newImportsWithAliases := collectImportsWithAliases(fileProgram, rootDir, filePath)
			for _, imp := range newImportsWithAliases {
				modulesToCheck = append(modulesToCheck, imp.Path)
				if imp.Alias != "" {
					pathToAlias[imp.Path] = imp.Alias
				}
			}
		}
	}

	// Pass 2: Full type check
	for _, pf := range allParsedFiles {
		checked[pf.path] = true

		fileTc := typechecker.NewTypeChecker(pf.source, pf.path)
		fileTc.SetSkipMainCheck(true)

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

		if pf.numFiles > 1 {
			if internals, ok := moduleInternalsMap[pf.modPath]; ok {
				var selfModuleName string
				if pf.program.Module != nil && pf.program.Module.Name != nil {
					selfModuleName = pf.program.Module.Name.Value
				}

				for typeName, t := range internals.types {
					fileTc.RegisterType(typeName, t)
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
			printWatchError()
			fmt.Print(errors.FormatErrorList(fileTc.Errors()))
			printSeparator()
			return
		}

		if fileTc.Errors().HasWarnings() {
			fmt.Print(errors.FormatErrorList(fileTc.Errors()))
		}
	}

	// Type checking for main file
	tc := typechecker.NewTypeChecker(source, filename)

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
	for moduleName, vars := range moduleVariables {
		for varName, varType := range vars {
			tc.RegisterModuleVariable(moduleName, varName, varType)
		}
	}

	tc.CheckProgram(program)

	if tc.Errors().HasErrors() {
		printWatchError()
		fmt.Print(errors.FormatErrorList(tc.Errors()))
		printSeparator()
		return
	}

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
		printWatchError()
		printRuntimeError(errObj, source, filename)
		printSeparator()
		return
	}

	// Look for main function and call it
	if mainFn, ok := env.Get("main"); ok {
		if fn, ok := mainFn.(*interpreter.Function); ok {
			fnEnv := interpreter.NewEnclosedEnvironment(env)
			mainResult := interpreter.Eval(fn.Body, fnEnv, ctx)

			// Execute ensure statements
			ensures := fnEnv.ExecuteEnsures()
			for _, ensureCall := range ensures {
				interpreter.Eval(ensureCall, fnEnv, ctx)
			}
			fnEnv.ClearEnsures()

			// Check for errors from main
			if errObj, ok := mainResult.(*interpreter.Error); ok {
				printWatchError()
				printRuntimeError(errObj, source, filename)
				printSeparator()
				return
			}
		}
	}

	printSeparator()
}

// printWatchError prints the error header for watch mode
func printWatchError() {
	fmt.Println("Error")
}

// printSeparator prints a visual separator between runs
func printSeparator() {
	fmt.Println("────────────────────────────")
}

// shortPath returns a shorter version of the path for display
func shortPath(path string) string {
	// Try to make it relative to current directory
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Base(path)
	}

	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return filepath.Base(path)
	}

	// If the relative path is simpler, use it
	if len(rel) < len(path) {
		return rel
	}

	return filepath.Base(path)
}
