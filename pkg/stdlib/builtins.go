package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/marshallburns/ez/pkg/object"
)

// Global stdin reader to maintain buffering across multiple input() calls
var stdinReader = bufio.NewReader(os.Stdin)

// getEZTypeName returns the EZ language type name for an object
func getEZTypeName(obj object.Object) string {
	switch v := obj.(type) {
	case *object.Integer:
		return v.GetDeclaredType()
	case *object.Float:
		return "float"
	case *object.String:
		return "string"
	case *object.Boolean:
		return "bool"
	case *object.Char:
		return "char"
	case *object.Array:
		return "array"
	case *object.Struct:
		if v.TypeName != "" {
			return v.TypeName
		}
		return "struct"
	case *object.Nil:
		return "nil"
	default:
		return string(obj.Type())
	}
}

// StdBuiltins contains the core standard library functions
var StdBuiltins = map[string]*object.Builtin{
	// Prints values to standard output followed by a newline
	"std.println": {
		Fn: func(args ...object.Object) object.Object {
			for i, arg := range args {
				if i > 0 {
					fmt.Print(" ")
				}
				// For strings, print raw value without quotes
				// Other types use Inspect() for proper representation
				if str, ok := arg.(*object.String); ok {
					fmt.Print(str.Value)
				} else {
					fmt.Print(arg.Inspect())
				}
			}
			fmt.Println()
			return object.NIL
		},
	},

	// Prints values to standard output WITHOUT a newline
	"std.print": {
		Fn: func(args ...object.Object) object.Object {
			for i, arg := range args {
				if i > 0 {
					fmt.Print(" ")
				}
				// For strings, print raw value without quotes
				// Other types use Inspect() for proper representation
				if str, ok := arg.(*object.String); ok {
					fmt.Print(str.Value)
				} else {
					fmt.Print(arg.Inspect())
				}
			}
			return object.NIL
		},
	},

	// Reads an integer from standard input
	"read_int": {
		Fn: func(args ...object.Object) object.Object {
			reader := bufio.NewReader(os.Stdin)
			text, readErr := reader.ReadString('\n')
			if len(text) > 0 && text[len(text)-1] == '\n' {
				text = text[:len(text)-1]
			}

			if readErr != nil {
				errObj := &object.Struct{
					TypeName: "Error",
					Fields: map[string]object.Object{
						"message": &object.String{Value: "failed to read input"},
						"code":    &object.Integer{Value: 1},
					},
				}
				return &object.ReturnValue{Values: []object.Object{&object.Integer{Value: 0}, errObj}}
			}

			val, parseErr := strconv.ParseInt(text, 10, 64)
			if parseErr != nil {
				errObj := &object.Struct{
					TypeName: "Error",
					Fields: map[string]object.Object{
						"message": &object.String{Value: "invalid integer input"},
						"code":    &object.Integer{Value: 2},
					},
				}
				return &object.ReturnValue{Values: []object.Object{&object.Integer{Value: 0}, errObj}}
			}

			return &object.ReturnValue{Values: []object.Object{&object.Integer{Value: val}, object.NIL}}
		},
	},

	// Returns the length of a string or array
	"len": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("len() takes exactly 1 argument, got %d", len(args))}
			}

			switch arg := args[0].(type) {
			case *object.String:
				return &object.Integer{Value: int64(len(arg.Value))}
			case *object.Array:
				return &object.Integer{Value: int64(len(arg.Elements))}
			default:
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("len() not supported for %s", args[0].Type())}
			}
		},
	},

	// Returns the type of the given object as a string
	"typeof": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "typeof() takes exactly 1 argument"}
			}
			return &object.String{Value: getEZTypeName(args[0])}
		},
	},

	// Converts a value to an integer
	"int": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "int() takes exactly 1 argument"}
			}
			switch arg := args[0].(type) {
			case *object.Integer:
				return arg
			case *object.Float:
				return &object.Integer{Value: int64(arg.Value)}
			case *object.String:
				cleanedValue := strings.ReplaceAll(arg.Value, "_", "")
				val, err := strconv.ParseInt(cleanedValue, 10, 64)
				if err != nil {
					return &object.Error{
						Code: "E7005",
						Message: fmt.Sprintf("cannot convert %q to int: invalid integer format\n\n"+
							"The string must contain only digits (0-9), optionally with:\n"+
							"  - A leading + or - sign\n"+
							"  - Underscores for readability (e.g., \"100_000\")\n\n"+
							"Examples of valid integers:\n"+
							"  \"42\", \"-123\", \"1_000_000\"", arg.Value),
					}
				}
				return &object.Integer{Value: val}
			case *object.Char:
				return &object.Integer{Value: int64(arg.Value)}
			default:
				if args[0].Type() == object.ARRAY_OBJ {
					return &object.Error{
						Code: "E7005",
						Message: "cannot convert ARRAY to int\n\n" +
							"Arrays cannot be directly converted to integers.\n" +
							"Hint: If you want the array length, use len() instead:\n" +
							"    len(myArray)  // returns the number of elements",
					}
				}
				return &object.Error{Code: "E7005", Message: fmt.Sprintf("cannot convert %s to int", args[0].Type())}
			}
		},
	},

	// Converts a value to a float
	"float": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "float() takes exactly 1 argument"}
			}
			switch arg := args[0].(type) {
			case *object.Float:
				return arg
			case *object.Integer:
				return &object.Float{Value: float64(arg.Value)}
			case *object.String:
				cleanedValue := strings.ReplaceAll(arg.Value, "_", "")
				val, err := strconv.ParseFloat(cleanedValue, 64)
				if err != nil {
					return &object.Error{
						Code: "E7006",
						Message: fmt.Sprintf("cannot convert %q to float: invalid float format\n\n"+
							"The string must be a valid floating-point number with:\n"+
							"  - Optional leading + or - sign\n"+
							"  - Digits with optional decimal point\n"+
							"  - Underscores for readability (e.g., \"3.14_159\")\n"+
							"  - Optional scientific notation (e.g., \"1.5e10\")\n\n"+
							"Examples of valid floats:\n"+
							"  \"3.14\", \"-2.5\", \"1_000.50\", \"1.5e10\"", arg.Value),
					}
				}
				return &object.Float{Value: val}
			default:
				if args[0].Type() == object.ARRAY_OBJ {
					return &object.Error{
						Code: "E7006",
						Message: "cannot convert ARRAY to float\n\n" +
							"Arrays cannot be directly converted to floating-point numbers.\n" +
							"Hint: If you want the array length, use len() instead:\n" +
							"    len(myArray)  // returns the number of elements",
					}
				}
				return &object.Error{Code: "E7006", Message: fmt.Sprintf("cannot convert %s to float", args[0].Type())}
			}
		},
	},

	// Converts a value to a string
	"string": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "string() takes exactly 1 argument"}
			}
			return &object.String{Value: args[0].Inspect()}
		},
	},

	// Reads a line of input from standard input
	"input": {
		Fn: func(args ...object.Object) object.Object {
			text, _ := stdinReader.ReadString('\n')
			text = strings.TrimRight(text, "\r\n")
			return &object.String{Value: text}
		},
	},
}
