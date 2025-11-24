package main

import (
	"fmt"
	"os"

	"github.com/marshallburns/ez/pkg/interpreter"
	"github.com/marshallburns/ez/pkg/lexer"
	"github.com/marshallburns/ez/pkg/parser"
	"github.com/marshallburns/ez/pkg/tokenizer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("EZ Language Interpreter")
		fmt.Println("Usage: ez <command> [file]")
		fmt.Println("Commands:")
		fmt.Println("  run <file>    Run an EZ program")
		fmt.Println("  lex <file>    Tokenize a file (debug)")
		return
	}

	command := os.Args[1]

	switch command {
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
		fmt.Printf("Unknown command: %s\n", command)
	}
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

	env := interpreter.NewEnvironment()
	result := interpreter.Eval(program, env)

	// Look for main function and call it
	if mainFn, ok := env.Get("main"); ok {
		if fn, ok := mainFn.(*interpreter.Function); ok {
			fnEnv := interpreter.NewEnclosedEnvironment(env)
			interpreter.Eval(fn.Body, fnEnv)
		}
	}

	// Check for errors
	if errObj, ok := result.(*interpreter.Error); ok {
		fmt.Println(errObj.Inspect())
	}
}
