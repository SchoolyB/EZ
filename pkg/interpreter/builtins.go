package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
)

var builtins = map[string]*Builtin{
	// BASIC STANDARD LIBRARY BUILTINS

	// Prints values to standard output followed by a newline. Any number of arguments can be provided.
	"std.println": {
		Fn: func(args ...Object) Object {
			for i, arg := range args {
				if i > 0 {
					fmt.Print(" ")
				}
				fmt.Print(arg.Inspect())
			}
			fmt.Println()
			return NIL
		},
	},
	// Prints values to standard output WITHOUT a newline. Any number of arguments can be provided.
	"std.print": {
		Fn: func(args ...Object) Object {
			for i, arg := range args {
				if i > 0 {
					fmt.Print(" ")
				}
				fmt.Print(arg.Inspect())
			}
			return NIL
		},
	},

	// START OF FULL BUILTINS

	// Reads an integer from standard input. Returns the integer and an error object (nil if no error).
	// Recomended to just use input() and convert to type: `int` manually.
	"read_int": {
		Fn: func(args ...Object) Object {
			reader := bufio.NewReader(os.Stdin)
			text, readErr := reader.ReadString('\n')
			// Trim newline
			if len(text) > 0 && text[len(text)-1] == '\n' {
				text = text[:len(text)-1]
			}

			// Handle read error
			if readErr != nil {
				errObj := &Struct{
					TypeName: "Error",
					Fields: map[string]Object{
						"message": &String{Value: "failed to read input"},
						"code":    &Integer{Value: 1},
					},
				}
				return &ReturnValue{Values: []Object{&Integer{Value: 0}, errObj}}
			}

			// Parse integer
			val, parseErr := strconv.ParseInt(text, 10, 64)
			if parseErr != nil {
				errObj := &Struct{
					TypeName: "Error",
					Fields: map[string]Object{
						"message": &String{Value: "invalid integer input"},
						"code":    &Integer{Value: 2},
					},
				}
				return &ReturnValue{Values: []Object{&Integer{Value: 0}, errObj}}
			}

			// Success - return value and nil error
			return &ReturnValue{Values: []Object{&Integer{Value: val}, NIL}}
		},
	},

	// Returns the length of a string or array as an integer.
	"len": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("len() takes exactly 1 argument, got %d", len(args))
			}

			switch arg := args[0].(type) {
			case *String:
				return &Integer{Value: int64(len(arg.Value))}
			case *Array:
				return &Integer{Value: int64(len(arg.Elements))}
			default:
				return newError("len() not supported for %s", args[0].Type())
			}
		},
	},
	// Returns the type of the given object as a string.
	"typeof": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("typeof() takes exactly 1 argument")
			}
			return &String{Value: string(args[0].Type())}
		},
	},

	// Converts a value to an integer if possible.
	"int": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("int() takes exactly 1 argument")
			}
			switch arg := args[0].(type) {
			case *Integer:
				return arg
			case *Float:
				return &Integer{Value: int64(arg.Value)}
			case *String:
				val, err := strconv.ParseInt(arg.Value, 10, 64)
				if err != nil {
					return newError("cannot convert %q to int", arg.Value)
				}
				return &Integer{Value: val}
			case *Char:
				return &Integer{Value: int64(arg.Value)}
			default:
				return newError("cannot convert %s to int", args[0].Type())
			}
		},
	},
	// Converts a value to a float if possible.
	"float": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("float() takes exactly 1 argument")
			}
			switch arg := args[0].(type) {
			case *Float:
				return arg
			case *Integer:
				return &Float{Value: float64(arg.Value)}
			case *String:
				val, err := strconv.ParseFloat(arg.Value, 64)
				if err != nil {
					return newError("cannot convert %q to float", arg.Value)
				}
				return &Float{Value: val}
			default:
				return newError("cannot convert %s to float", args[0].Type())
			}
		},
	},

	// Converts a value to a string.
	"string": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("string() takes exactly 1 argument")
			}
			return &String{Value: args[0].Inspect()}
		},
	},

	// Reads a line of input from standard input and returns it as a string.
	"input": {
		Fn: func(args ...Object) Object {
			reader := bufio.NewReader(os.Stdin)
			text, _ := reader.ReadString('\n')
			// Trim newline
			if len(text) > 0 && text[len(text)-1] == '\n' {
				text = text[:len(text)-1]
			}
			return &String{Value: text}
		},
	},
}
