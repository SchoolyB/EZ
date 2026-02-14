//go:build js && wasm

package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"os"
	"syscall/js"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/interpreter"
	"github.com/marshallburns/ez/pkg/lexer"
	"github.com/marshallburns/ez/pkg/parser"
	"github.com/marshallburns/ez/pkg/typechecker"
)

// executeEZ runs EZ code and returns the result
// Called from JavaScript: executeEZ(code) -> {success: bool}
// Output is captured via console.log interception in JavaScript
func executeEZ(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "No code provided")
		return map[string]interface{}{
			"success": false,
		}
	}

	code := args[0].String()
	success := execute(code)

	return map[string]interface{}{
		"success": success,
	}
}

func execute(source string) bool {
	filename := "playground.ez"

	// Lexer
	l := lexer.NewLexer(source)
	p := parser.NewWithSource(l, source, filename)
	program := p.ParseProgram()

	// Check for lexer errors
	if len(l.Errors()) > 0 {
		errList := errors.NewErrorList()
		for _, lexErr := range l.Errors() {
			code := errors.ErrorCode{Code: lexErr.Code, Name: "lexer-error", Description: "Lexer error"}
			sourceLine := errors.GetSourceLine(source, lexErr.Line)
			ezErr := errors.NewErrorWithSource(code, lexErr.Message, filename, lexErr.Line, lexErr.Column, sourceLine)
			errList.AddError(ezErr)
		}
		fmt.Fprint(os.Stderr, errors.FormatErrorList(errList))
		return false
	}

	// Check for parser errors
	if p.EZErrors().HasErrors() {
		fmt.Fprint(os.Stderr, errors.FormatErrorList(p.EZErrors()))
		return false
	}

	// Display parser warnings
	if p.EZErrors().HasWarnings() {
		fmt.Fprint(os.Stderr, errors.FormatErrorList(p.EZErrors()))
	}

	// Type checking
	tc := typechecker.NewTypeChecker(source, filename)
	tc.CheckProgram(program)

	// Check for type errors
	if tc.Errors().HasErrors() {
		fmt.Fprint(os.Stderr, errors.FormatErrorList(tc.Errors()))
		return false
	}

	// Display type checker warnings
	if tc.Errors().HasWarnings() {
		fmt.Fprint(os.Stderr, errors.FormatErrorList(tc.Errors()))
	}

	// Evaluation
	env := interpreter.NewEnvironment()
	ctx := &interpreter.EvalContext{
		CurrentFile: filename,
	}
	result := interpreter.Eval(program, env, ctx)

	// Check for evaluation errors
	if errObj, ok := result.(*interpreter.Error); ok {
		printRuntimeError(errObj, source, filename)
		return false
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
			}
			fnEnv.ClearEnsures()

			// Check for errors from main
			if errObj, ok := mainResult.(*interpreter.Error); ok {
				printRuntimeError(errObj, source, filename)
				return false
			}
		}
	}

	return true
}

// printRuntimeError formats and prints a runtime error
func printRuntimeError(errObj *interpreter.Error, source, filename string) {
	if errObj.Code != "" && errObj.Line > 0 {
		code := errors.ErrorCode{Code: errObj.Code, Name: "error", Description: "error occurred here"}
		sourceLine := errors.GetSourceLine(source, errObj.Line)
		ezErr := errors.NewErrorWithSource(code, errObj.Message, filename, errObj.Line, errObj.Column, sourceLine)
		if errObj.Help != "" {
			ezErr.Help = errObj.Help
		}

		errList := errors.NewErrorList()
		errList.AddError(ezErr)
		fmt.Fprint(os.Stderr, errors.FormatErrorList(errList))
	} else {
		fmt.Fprintln(os.Stderr, errObj.Inspect())
	}
}

func main() {
	// Register the executeEZ function for JavaScript
	js.Global().Set("executeEZ", js.FuncOf(executeEZ))

	// Keep the program running
	select {}
}
