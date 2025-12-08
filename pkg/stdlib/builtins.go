package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"bufio"
	"fmt"
	"math/big"
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
	case *object.Byte:
		return "byte"
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

// deepCopy creates a deep copy of an object
// Primitives return themselves (they're immutable)
// Structs, arrays, and maps are recursively copied
func deepCopy(obj object.Object) object.Object {
	switch v := obj.(type) {
	case *object.Nil:
		return v
	case *object.Integer:
		return &object.Integer{Value: v.Value, DeclaredType: v.DeclaredType}
	case *object.Float:
		return &object.Float{Value: v.Value}
	case *object.String:
		return &object.String{Value: v.Value}
	case *object.Boolean:
		return &object.Boolean{Value: v.Value}
	case *object.Char:
		return &object.Char{Value: v.Value}
	case *object.Byte:
		return &object.Byte{Value: v.Value}
	case *object.Array:
		newElements := make([]object.Object, len(v.Elements))
		for i, elem := range v.Elements {
			newElements[i] = deepCopy(elem)
		}
		// Copied arrays are mutable by default to allow modification
		// (const enforcement happens at variable declaration level)
		return &object.Array{
			Elements:    newElements,
			Mutable:     true,
			ElementType: v.ElementType,
		}
	case *object.Map:
		newMap := object.NewMap()
		for _, pair := range v.Pairs {
			// Keys are immutable (hashable), but values may need deep copy
			newMap.Set(pair.Key, deepCopy(pair.Value))
		}
		// Copied maps are mutable by default to allow modification
		newMap.Mutable = true
		return newMap
	case *object.Struct:
		newFields := make(map[string]object.Object)
		for key, val := range v.Fields {
			newFields[key] = deepCopy(val)
		}
		// Copied structs are mutable by default to allow modification
		// (const enforcement happens at variable declaration level)
		return &object.Struct{
			TypeName: v.TypeName,
			Fields:   newFields,
			Mutable:  true,
		}
	case *object.EnumValue:
		// Enum values are immutable references, return as-is
		return v
	case *object.Function:
		// Functions are not copied (they're code references)
		return v
	case *object.Builtin:
		// Builtins are not copied
		return v
	default:
		// For any other types, return as-is
		return obj
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

	// Prints values to standard output WITHOUT a newline (like C's printf)
	// User must explicitly add \n for newlines
	"std.printf": {
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
						"code":    &object.Integer{Value: big.NewInt(1)},
					},
				}
				return &object.ReturnValue{Values: []object.Object{&object.Integer{Value: big.NewInt(0)}, errObj}}
			}

			val, parseErr := strconv.ParseInt(text, 10, 64)
			if parseErr != nil {
				errObj := &object.Struct{
					TypeName: "Error",
					Fields: map[string]object.Object{
						"message": &object.String{Value: "invalid integer input"},
						"code":    &object.Integer{Value: big.NewInt(2)},
					},
				}
				return &object.ReturnValue{Values: []object.Object{&object.Integer{Value: big.NewInt(0)}, errObj}}
			}

			return &object.ReturnValue{Values: []object.Object{&object.Integer{Value: big.NewInt(val)}, object.NIL}}
		},
	},

	// Returns the length of a string or array
	// For strings, returns character count (not byte count) to properly support UTF-8
	"len": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("len() takes exactly 1 argument, got %d", len(args))}
			}

			switch arg := args[0].(type) {
			case *object.String:
				// Use []rune to count characters, not bytes (proper UTF-8 support)
				return &object.Integer{Value: big.NewInt(int64(len([]rune(arg.Value))))}
			case *object.Array:
				return &object.Integer{Value: big.NewInt(int64(len(arg.Elements)))}
			case *object.Map:
				return &object.Integer{Value: big.NewInt(int64(len(arg.Pairs)))}
			default:
				return &object.Error{Code: "E7015", Message: fmt.Sprintf("len() not supported for %s", args[0].Type())}
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
				return &object.Integer{Value: big.NewInt(int64(arg.Value))}
			case *object.String:
				cleanedValue := strings.ReplaceAll(arg.Value, "_", "")
				val, err := strconv.ParseInt(cleanedValue, 10, 64)
				if err != nil {
					return &object.Error{
						Code: "E7014",
						Message: fmt.Sprintf("cannot convert %q to int: invalid integer format\n\n"+
							"The string must contain only digits (0-9), optionally with:\n"+
							"  - A leading + or - sign\n"+
							"  - Underscores for readability (e.g., \"100_000\")\n\n"+
							"Examples of valid integers:\n"+
							"  \"42\", \"-123\", \"1_000_000\"", arg.Value),
					}
				}
				return &object.Integer{Value: big.NewInt(val)}
			case *object.Char:
				return &object.Integer{Value: big.NewInt(int64(arg.Value))}
			case *object.Byte:
				return &object.Integer{Value: big.NewInt(int64(arg.Value))}
			case *object.EnumValue:
				// Extract the underlying value from the enum
				switch v := arg.Value.(type) {
				case *object.Integer:
					return v
				case *object.Float:
					return &object.Integer{Value: big.NewInt(int64(v.Value))}
				case *object.String:
					cleanedValue := strings.ReplaceAll(v.Value, "_", "")
					val, err := strconv.ParseInt(cleanedValue, 10, 64)
					if err != nil {
						return &object.Error{
							Code:    "E7005",
							Message: fmt.Sprintf("cannot convert enum value %q to int: underlying value is not numeric", v.Value),
						}
					}
					return &object.Integer{Value: big.NewInt(val)}
				default:
					return &object.Error{
						Code:    "E7005",
						Message: fmt.Sprintf("cannot convert enum %s.%s to int: underlying type %s is not convertible", arg.EnumType, arg.Name, arg.Value.Type()),
					}
				}
			default:
				if args[0].Type() == object.ARRAY_OBJ {
					return &object.Error{
						Code: "E7014",
						Message: "cannot convert ARRAY to int\n\n" +
							"Arrays cannot be directly converted to integers.\n" +
							"Hint: If you want the array length, use len() instead:\n" +
							"    len(myArray)  // returns the number of elements",
					}
				}
				return &object.Error{Code: "E7014", Message: fmt.Sprintf("cannot convert %s to int", args[0].Type())}
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
				f, _ := new(big.Float).SetInt(arg.Value).Float64()
				return &object.Float{Value: f}
			case *object.String:
				cleanedValue := strings.ReplaceAll(arg.Value, "_", "")
				val, err := strconv.ParseFloat(cleanedValue, 64)
				if err != nil {
					return &object.Error{
						Code: "E7014",
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
						Code: "E7014",
						Message: "cannot convert ARRAY to float\n\n" +
							"Arrays cannot be directly converted to floating-point numbers.\n" +
							"Hint: If you want the array length, use len() instead:\n" +
							"    len(myArray)  // returns the number of elements",
					}
				}
				return &object.Error{Code: "E7014", Message: fmt.Sprintf("cannot convert %s to float", args[0].Type())}
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

	// Converts a value to a char (Unicode code point)
	"char": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "char() takes exactly 1 argument"}
			}
			switch arg := args[0].(type) {
			case *object.Char:
				return arg
			case *object.Integer:
				intVal := arg.Value.Int64()
				if intVal < 0 || intVal > 0x10FFFF {
					return &object.Error{
						Code:    "E7014",
						Message: fmt.Sprintf("cannot convert %s to char: value must be a valid Unicode code point (0 to 0x10FFFF)", arg.Value.String()),
					}
				}
				return &object.Char{Value: rune(intVal)}
			case *object.Float:
				intVal := int64(arg.Value)
				if intVal < 0 || intVal > 0x10FFFF {
					return &object.Error{
						Code:    "E7014",
						Message: fmt.Sprintf("cannot convert %v to char: value must be a valid Unicode code point (0 to 0x10FFFF)", arg.Value),
					}
				}
				return &object.Char{Value: rune(intVal)}
			case *object.Byte:
				return &object.Char{Value: rune(arg.Value)}
			case *object.String:
				runes := []rune(arg.Value)
				if len(runes) != 1 {
					return &object.Error{
						Code:    "E7014",
						Message: fmt.Sprintf("cannot convert %q to char: string must be exactly one character", arg.Value),
					}
				}
				return &object.Char{Value: runes[0]}
			default:
				return &object.Error{Code: "E7014", Message: fmt.Sprintf("cannot convert %s to char", args[0].Type())}
			}
		},
	},

	// Converts a value to a byte (0-255)
	"byte": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "byte() takes exactly 1 argument"}
			}
			switch arg := args[0].(type) {
			case *object.Byte:
				return arg
			case *object.Integer:
				intVal := arg.Value.Int64()
				if intVal < 0 || intVal > 255 {
					return &object.Error{
						Code:    "E7014",
						Message: fmt.Sprintf("cannot convert %s to byte: value must be between 0 and 255", arg.Value.String()),
					}
				}
				return &object.Byte{Value: uint8(intVal)}
			case *object.Float:
				intVal := int64(arg.Value)
				if intVal < 0 || intVal > 255 {
					return &object.Error{
						Code:    "E7014",
						Message: fmt.Sprintf("cannot convert %v to byte: value must be between 0 and 255", arg.Value),
					}
				}
				return &object.Byte{Value: uint8(intVal)}
			case *object.Char:
				if arg.Value < 0 || arg.Value > 255 {
					return &object.Error{
						Code:    "E7014",
						Message: fmt.Sprintf("cannot convert char %q to byte: value must be between 0 and 255", arg.Value),
					}
				}
				return &object.Byte{Value: uint8(arg.Value)}
			case *object.String:
				cleanedValue := strings.ReplaceAll(arg.Value, "_", "")
				val, err := strconv.ParseInt(cleanedValue, 10, 64)
				if err != nil {
					return &object.Error{
						Code:    "E7014",
						Message: fmt.Sprintf("cannot convert %q to byte: invalid integer format", arg.Value),
					}
				}
				if val < 0 || val > 255 {
					return &object.Error{
						Code:    "E7014",
						Message: fmt.Sprintf("cannot convert %q to byte: value must be between 0 and 255", arg.Value),
					}
				}
				return &object.Byte{Value: uint8(val)}
			default:
				return &object.Error{Code: "E7014", Message: fmt.Sprintf("cannot convert %s to byte", args[0].Type())}
			}
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

	// Creates a deep copy of a value
	// Primitives return themselves, structs/arrays/maps are recursively copied
	"copy": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("copy() takes exactly 1 argument, got %d", len(args))}
			}
			return deepCopy(args[0])
		},
	},

	// Creates a user-defined error with the given message
	// Returns an Error struct with .message set to the argument and .code set to empty string
	// This is different from object.Error which is a runtime error that halts execution
	"error": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("error() takes exactly 1 argument, got %d", len(args))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("error() argument must be a string, got %s", getEZTypeName(args[0]))}
			}
			// Return an Error struct (not object.Error which is a runtime error)
			return &object.Struct{
				TypeName: "Error",
				Fields: map[string]object.Object{
					"message": &object.String{Value: str.Value},
					"code":    &object.String{Value: ""},
				},
			}
		},
	},
}
