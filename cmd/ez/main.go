package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/interpreter"
	"github.com/marshallburns/ez/pkg/lexer"
	"github.com/marshallburns/ez/pkg/parser"
	"github.com/marshallburns/ez/pkg/tokenizer"
	"github.com/marshallburns/ez/pkg/typechecker"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	command := os.Args[1]

	switch command {
	case "help", "-h", "--help":
		printHelp()
	case "version", "-v", "--version":
		printVersion()
	case "repl":
		startREPL()
	case "build":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ez build <file>")
			return
		}
		buildFile(os.Args[2])
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
	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ez run <file>")
			return
		}
		runFile(os.Args[2])
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
	fmt.Println("  run <file>     Run an EZ program")
	fmt.Println("  build <file>   Check syntax and types without running")
	fmt.Println("  repl           Start interactive REPL mode")
	fmt.Println("  lex <file>     Tokenize a file (debug)")
	fmt.Println("  parse <file>   Parse a file (debug)")
	fmt.Println("  version        Show version information")
	fmt.Println("  help           Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ez myProgram.ez")
	fmt.Println("  ez run examples/hello.ez")
	fmt.Println("  ez build myProgram.ez")
	fmt.Println("  ez repl")
}

func printVersion() {
	fmt.Println("EZ Language v0.1.0")
	fmt.Println("Copyright (c) 2025-Present Marshall A Burns")
}

func buildFile(filename string) {
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

	fmt.Printf("Build successful: %s\n", filename)
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
		return
	}

	// Check for parser errors using new error system
	if p.EZErrors().HasErrors() {
		fmt.Print(errors.FormatErrorList(p.EZErrors()))
		return
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
		return
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

	// Check for evaluation errors
	if errObj, ok := result.(*interpreter.Error); ok {
		printRuntimeError(errObj, source, filename)
		return
	}

	// Look for main function and call it
	if mainFn, ok := env.Get("main"); ok {
		if fn, ok := mainFn.(*interpreter.Function); ok {
			fnEnv := interpreter.NewEnclosedEnvironment(env)
			mainResult := interpreter.Eval(fn.Body, fnEnv)

			// Check for errors from main
			if errObj, ok := mainResult.(*interpreter.Error); ok {
				printRuntimeError(errObj, source, filename)
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

		sourceLine := errors.GetSourceLine(source, errObj.Line)
		ezErr := errors.NewErrorWithSource(code, errObj.Message, filename, errObj.Line, errObj.Column, sourceLine)
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
