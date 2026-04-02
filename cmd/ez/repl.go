package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/marshallburns/ez/internal/ezc"
	"github.com/marshallburns/ez/pkg/lineeditor"
)

const (
	PROMPT      = ">> "
	MULTI_LINE  = ".. "
	REPL_BANNER = "EZ REPL (compiler-backed) — type 'exit' to quit, 'help' for commands"
)

// replState holds the accumulated source across REPL turns
type replState struct {
	imports      []string // import lines
	declarations []string // function/struct/enum declarations
	statements   []string // statements inside main()
}

func (s *replState) buildSource() string {
	var b strings.Builder

	// Imports
	for _, imp := range s.imports {
		b.WriteString(imp)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Top-level declarations
	for _, decl := range s.declarations {
		b.WriteString(decl)
		b.WriteString("\n")
	}

	// Wrap statements in main()
	b.WriteString("do main() {\n")
	for _, stmt := range s.statements {
		b.WriteString("    ")
		b.WriteString(stmt)
		b.WriteString("\n")
	}
	b.WriteString("}\n")

	return b.String()
}

func startREPL() {
	fmt.Println(REPL_BANNER)
	fmt.Println()

	editor := lineeditor.New(100)
	state := &replState{
		imports: []string{},
	}

	// Create a temp directory for REPL files
	tmpDir, err := os.MkdirTemp("", "ez-repl-*")
	if err != nil {
		fmt.Printf("Error creating temp directory: %v\n", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "repl.ez")

	for {
		line, err := editor.ReadLine(PROMPT)
		if err != nil {
			fmt.Println()
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle REPL commands
		switch line {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return
		case "clear":
			fmt.Print("\033[H\033[2J")
			continue
		case "reset":
			state = &replState{
				imports: []string{},
			}
			fmt.Println("Session reset.")
			continue
		case "help":
			printReplHelp()
			continue
		}

		// Multi-line input: if the line ends with { and braces are unbalanced
		if needsMoreInput(line) {
			for {
				more, err := editor.ReadLine(MULTI_LINE)
				if err != nil {
					break
				}
				line += "\n" + more
				if !needsMoreInput(line) {
					break
				}
			}
		}

		// Classify the input and try adding it
		category := classifyInput(line)

		// Save state for rollback
		prevImports := make([]string, len(state.imports))
		copy(prevImports, state.imports)
		prevDecls := make([]string, len(state.declarations))
		copy(prevDecls, state.declarations)
		prevStmts := make([]string, len(state.statements))
		copy(prevStmts, state.statements)

		switch category {
		case "import":
			state.imports = append(state.imports, line)
		case "declaration":
			state.declarations = append(state.declarations, line)
		default:
			state.statements = append(state.statements, line)
		}

		// Write and compile
		source := state.buildSource()
		if err := os.WriteFile(tmpFile, []byte(source), 0644); err != nil {
			fmt.Printf("Error writing temp file: %v\n", err)
			continue
		}

		ezcPath, err := ezc.Find()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		cmd := exec.Command(ezcPath, "run", tmpFile)
		output, err := cmd.CombinedOutput()
		outStr := string(output)

		if err != nil {
			// Compilation or runtime error — rollback and show error
			state.imports = prevImports
			state.declarations = prevDecls
			state.statements = prevStmts

			// Show a cleaned-up error (strip the temp file path)
			cleaned := strings.ReplaceAll(outStr, tmpFile, "<repl>")
			fmt.Print(cleaned)
		} else if outStr != "" {
			fmt.Print(outStr)
		}
	}
}

// classifyInput determines if input is an import, declaration, or statement
func classifyInput(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "using ") {
		return "import"
	}
	if strings.HasPrefix(trimmed, "do ") ||
		strings.HasPrefix(trimmed, "private do ") ||
		strings.HasPrefix(trimmed, "const ") && (strings.Contains(trimmed, " struct ") || strings.Contains(trimmed, " enum ")) {
		return "declaration"
	}
	return "statement"
}

// needsMoreInput checks if braces are unbalanced
func needsMoreInput(input string) bool {
	depth := 0
	for _, ch := range input {
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
		}
	}
	return depth > 0
}

func printReplHelp() {
	fmt.Println("REPL Commands:")
	fmt.Println("  exit, quit  — Exit the REPL")
	fmt.Println("  clear       — Clear the screen")
	fmt.Println("  reset       — Reset session (clear all variables and imports)")
	fmt.Println("  help        — Show this help message")
	fmt.Println()
	fmt.Println("You can define variables, functions, structs, and enums.")
	fmt.Println("Each line is compiled and executed via the EZ compiler.")
}
