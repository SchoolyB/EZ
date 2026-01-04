package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"os"
	"strings"

	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/interpreter"
	"github.com/marshallburns/ez/pkg/lexer"
	"github.com/marshallburns/ez/pkg/lineeditor"
	"github.com/marshallburns/ez/pkg/parser"
)

const (
	PROMPT      = ">> "
	CONTINUE    = ".. "
	REPL_SOURCE = "<repl>"
)

// startREPL starts the interactive REPL
func startREPL() {
	fmt.Printf("EZ Language REPL %s\n", Version)
	fmt.Println("Type 'help' for commands, 'exit' or 'quit' to exit")
	fmt.Println()

	env := interpreter.NewEnvironment()
	editor := lineeditor.New(100)
	defer editor.Close()

	for {
		line, err := editor.ReadLine(PROMPT)
		if err != nil {
			if err == lineeditor.ErrInterrupted {
				continue // Ctrl+C just cancels current line
			}
			if err == lineeditor.ErrEOF {
				fmt.Println("Goodbye!")
				break
			}
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			break
		}

		line = strings.TrimSpace(line)

		// Handle empty lines
		if line == "" {
			continue
		}

		// Handle REPL commands
		if newEnv := handleReplCommand(line, env); newEnv != nil {
			env = newEnv
			continue
		}

		// Check if we need multi-line input
		if needsMoreInput(line) {
			line = readMultiLineInputWithEditor(editor, line)
			if line == "" {
				continue // User cancelled or error
			}
		}

		// Parse and evaluate
		evaluateLine(line, env)
	}
}

// handleReplCommand handles special REPL commands
// Returns a new environment if the environment should be reset, nil otherwise
func handleReplCommand(line string, env *interpreter.Environment) *interpreter.Environment {
	switch line {
	case "exit", "quit":
		fmt.Println("Goodbye!")
		os.Exit(0)
		return nil

	case "clear":
		// Clear the terminal screen only
		fmt.Print("\033[H\033[2J")
		return nil

	case "reset":
		// Clear the terminal screen AND reset the environment
		fmt.Print("\033[H\033[2J")
		fmt.Printf("EZ Language REPL %s\n", Version)
		fmt.Println("Type 'help' for commands, 'exit' or 'quit' to exit")
		fmt.Println()
		return interpreter.NewEnvironment()

	case "help":
		printReplHelp()
		return nil
	}

	return nil
}

// printReplHelp prints REPL help information
func printReplHelp() {
	fmt.Println("REPL Commands:")
	fmt.Println("  exit, quit    Exit the REPL")
	fmt.Println("  clear         Clear the terminal screen")
	fmt.Println("  reset         Clear screen and reset environment")
	fmt.Println("  help          Show this help message")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  - Type any EZ expression or statement")
	fmt.Println("  - Expressions are automatically printed")
	fmt.Println("  - Variables and functions persist across lines")
	fmt.Println("  - Use imports like: import @std")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  >> temp x int = 5")
	fmt.Println("  >> x + 10")
	fmt.Println("  15")
	fmt.Println("  >> import @std")
	fmt.Println("  >> std.println(\"Hello, REPL!\")")
	fmt.Println("  Hello, REPL!")
}

// needsMoreInput checks if the line needs continuation
// (for function definitions, blocks, etc.)
func needsMoreInput(line string) bool {
	// Simple heuristic: if line starts with 'do' and doesn't have closing }
	// or if it has unmatched braces
	if strings.HasPrefix(line, "do ") {
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")
		return openBraces > closeBraces
	}

	// Check for other statements that might need continuation
	if strings.HasPrefix(line, "if ") || strings.HasPrefix(line, "for ") ||
		strings.HasPrefix(line, "for_each ") || strings.HasPrefix(line, "as_long_as ") {
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")
		return openBraces > closeBraces
	}

	return false
}

// readMultiLineInputWithEditor continues reading input for multi-line statements using the line editor
func readMultiLineInputWithEditor(editor *lineeditor.Editor, initial string) string {
	result, err := editor.ReadMultiLine(PROMPT, CONTINUE, initial)
	if err != nil {
		return "" // EOF, interrupt, or error
	}
	return result
}

// evaluateLine parses and evaluates a single line/statement
func evaluateLine(line string, env *interpreter.Environment) {
	// Lexer
	l := lexer.NewLexer(line)

	// Parser
	p := parser.NewWithSource(l, line, REPL_SOURCE)
	stmt := p.ParseLine()

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
			sourceLine := errors.GetSourceLine(line, lexErr.Line)
			ezErr := errors.NewErrorWithSource(code, lexErr.Message, REPL_SOURCE, lexErr.Line, lexErr.Column, sourceLine)
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

	// No statement parsed (comment or empty)
	if stmt == nil {
		return
	}

	// Evaluate
	result := interpreter.Eval(stmt, env)

	// Check for runtime errors
	if errObj, ok := result.(*interpreter.Error); ok {
		printRuntimeError(errObj, line, REPL_SOURCE)
		return
	}

	// Display result if it's an expression
	if shouldDisplayResult(stmt, result) {
		fmt.Println(result.Inspect())
	}
}

// shouldDisplayResult determines if we should print the result
func shouldDisplayResult(stmt ast.Statement, result interpreter.Object) bool {
	// Don't display nil
	if result == interpreter.NIL {
		return false
	}

	// Display expression statements (like "x + 5", "foo()")
	if _, ok := stmt.(*ast.ExpressionStatement); ok {
		return true
	}

	// Don't display declarations
	return false
}
