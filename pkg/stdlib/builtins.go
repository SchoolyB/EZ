package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/marshallburns/ez/pkg/object"
)

// Global stdin reader to maintain buffering across multiple input() calls
var stdinReader = bufio.NewReader(os.Stdin)

// Track if stdin EOF has been encountered - subsequent reads will exit gracefully
var stdinEOFReached = false

// getEZTypeName returns the EZ language type name for an object
// Returns the full type as it would appear in EZ code (e.g., "[int]", "map[string:int]")
func getEZTypeName(obj object.Object) string {
	switch v := obj.(type) {
	case *object.Integer:
		return v.GetDeclaredType()
	case *object.Float:
		return v.GetDeclaredType()
	case *object.String:
		return "string"
	case *object.Boolean:
		return "bool"
	case *object.Char:
		return "char"
	case *object.Byte:
		return "byte"
	case *object.Array:
		if v.ElementType != "" {
			return "[" + v.ElementType + "]"
		}
		return "array"
	case *object.Map:
		if v.KeyType != "" && v.ValueType != "" {
			return "map[" + v.KeyType + ":" + v.ValueType + "]"
		}
		return "map"
	case *object.Struct:
		if v.TypeName != "" {
			return v.TypeName
		}
		return "struct"
	case *object.EnumValue:
		if v.EnumType != "" {
			return v.EnumType
		}
		return "enum"
	case *object.Range:
		return "Range<int>"
	case *object.FileHandle:
		return "File"
	case *object.Reference:
		// Get the inner type by dereferencing
		if inner, ok := v.Deref(); ok {
			return "Ref<" + getEZTypeName(inner) + ">"
		}
		return "Ref<unknown>"
	case *object.Nil:
		return "nil"
	case *object.Function:
		return "function"
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
		return &object.Float{Value: v.Value, DeclaredType: v.DeclaredType}
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
		newMap.KeyType = v.KeyType
		newMap.ValueType = v.ValueType
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

// extractIntValue extracts a big.Int value from various EZ types for sized integer conversions
func extractIntValue(arg object.Object, targetType string) (*big.Int, *object.Error) {
	switch v := arg.(type) {
	case *object.Integer:
		return v.Value, nil
	case *object.Float:
		// Check for NaN and Inf
		if math.IsNaN(v.Value) {
			return nil, &object.Error{
				Code:    "E7033",
				Message: fmt.Sprintf("cannot convert NaN to %s", targetType),
			}
		}
		if math.IsInf(v.Value, 0) {
			return nil, &object.Error{
				Code:    "E7033",
				Message: fmt.Sprintf("cannot convert Inf to %s", targetType),
			}
		}
		return big.NewInt(int64(v.Value)), nil
	case *object.String:
		cleanedValue := strings.ReplaceAll(v.Value, "_", "")
		val, ok := new(big.Int).SetString(cleanedValue, 10)
		if !ok {
			return nil, &object.Error{
				Code:    "E7014",
				Message: fmt.Sprintf("cannot convert %q to %s: invalid integer format", v.Value, targetType),
			}
		}
		return val, nil
	case *object.Byte:
		return big.NewInt(int64(v.Value)), nil
	case *object.Char:
		return big.NewInt(int64(v.Value)), nil
	default:
		return nil, &object.Error{
			Code:    "E7014",
			Message: fmt.Sprintf("cannot convert %s to %s", arg.Type(), targetType),
		}
	}
}

// extractFloatValue extracts a float64 value from various EZ types for float conversions
func extractFloatValue(arg object.Object, targetType string) (float64, *object.Error) {
	switch v := arg.(type) {
	case *object.Float:
		return v.Value, nil
	case *object.Integer:
		f, _ := new(big.Float).SetInt(v.Value).Float64()
		return f, nil
	case *object.String:
		cleanedValue := strings.ReplaceAll(v.Value, "_", "")
		val, err := strconv.ParseFloat(cleanedValue, 64)
		if err != nil {
			return 0, &object.Error{
				Code:    "E7014",
				Message: fmt.Sprintf("cannot convert %q to %s: invalid float format", v.Value, targetType),
			}
		}
		return val, nil
	case *object.Byte:
		return float64(v.Value), nil
	case *object.Char:
		return float64(v.Value), nil
	default:
		return 0, &object.Error{
			Code:    "E7014",
			Message: fmt.Sprintf("cannot convert %s to %s", arg.Type(), targetType),
		}
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
			// If EOF was already reached, exit gracefully to prevent infinite loops
			if stdinEOFReached {
				os.Exit(0)
			}

			text, readErr := stdinReader.ReadString('\n')
			if len(text) > 0 && text[len(text)-1] == '\n' {
				text = text[:len(text)-1]
			}

			// Handle EOF separately from other errors
			if readErr == io.EOF {
				// Mark that EOF has been reached
				stdinEOFReached = true

				// If we got some text before EOF, try to parse it
				if len(text) > 0 {
					val, parseErr := strconv.ParseInt(text, 10, 64)
					if parseErr == nil {
						return &object.ReturnValue{Values: []object.Object{&object.Integer{Value: big.NewInt(val)}, object.NIL}}
					}
				}
				// Return a distinguishable EOF error
				errObj := &object.Struct{
					TypeName: "Error",
					Fields: map[string]object.Object{
						"message": &object.String{Value: "end of input"},
						"code":    &object.Integer{Value: big.NewInt(3)},
					},
				}
				return &object.ReturnValue{Values: []object.Object{&object.Integer{Value: big.NewInt(0)}, errObj}}
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
				// Check for NaN and Inf
				if math.IsNaN(arg.Value) {
					return &object.Error{
						Code:    "E7033",
						Message: "cannot convert NaN to int",
					}
				}
				if math.IsInf(arg.Value, 0) {
					return &object.Error{
						Code:    "E7033",
						Message: "cannot convert Inf to int",
					}
				}
				// Check for overflow: float64 can exceed int64 range
				if arg.Value > float64(math.MaxInt64) || arg.Value < float64(math.MinInt64) {
					return &object.Error{
						Code:    "E7033",
						Message: fmt.Sprintf("float-to-int overflow: %g exceeds int64 range", arg.Value),
					}
				}
				return &object.Integer{Value: big.NewInt(int64(arg.Value))}
			case *object.String:
				cleanedValue := strings.ReplaceAll(arg.Value, "_", "")
				val, err := strconv.ParseInt(cleanedValue, 10, 64)
				if err != nil {
					// Check if this is a range error (overflow) vs syntax error (invalid format)
					if errors.Is(err, strconv.ErrRange) {
						return &object.Error{
							Code:    "E7033",
							Message: fmt.Sprintf("integer overflow: %q exceeds int64 range (-9223372036854775808 to 9223372036854775807)", arg.Value),
						}
					}
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
					// Check for NaN and Inf
					if math.IsNaN(v.Value) {
						return &object.Error{
							Code:    "E7033",
							Message: "cannot convert enum with NaN value to int",
						}
					}
					if math.IsInf(v.Value, 0) {
						return &object.Error{
							Code:    "E7033",
							Message: "cannot convert enum with Inf value to int",
						}
					}
					// Check for overflow
					if v.Value > float64(math.MaxInt64) || v.Value < float64(math.MinInt64) {
						return &object.Error{
							Code:    "E7033",
							Message: fmt.Sprintf("float-to-int overflow: enum value %g exceeds int64 range", v.Value),
						}
					}
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
					// Check if this is a range error (overflow) vs syntax error (invalid format)
					if errors.Is(err, strconv.ErrRange) {
						return &object.Error{
							Code:    "E7014",
							Message: fmt.Sprintf("cannot convert %q to byte: value must be between 0 and 255", arg.Value),
						}
					}
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

	// ============================================================================
	// Sized Integer Conversion Functions
	// ============================================================================

	// Converts a value to an i8 (signed 8-bit integer, -128 to 127)
	"i8": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "i8() takes exactly 1 argument"}
			}
			val, err := extractIntValue(args[0], "i8")
			if err != nil {
				return err
			}
			if val.Cmp(big.NewInt(-128)) < 0 || val.Cmp(big.NewInt(127)) > 0 {
				return &object.Error{
					Code:    "E3022",
					Message: fmt.Sprintf("value %s out of i8 range (-128 to 127)", val.String()),
				}
			}
			return &object.Integer{Value: new(big.Int).Set(val), DeclaredType: "i8"}
		},
	},

	// Converts a value to an i16 (signed 16-bit integer, -32768 to 32767)
	"i16": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "i16() takes exactly 1 argument"}
			}
			val, err := extractIntValue(args[0], "i16")
			if err != nil {
				return err
			}
			if val.Cmp(big.NewInt(-32768)) < 0 || val.Cmp(big.NewInt(32767)) > 0 {
				return &object.Error{
					Code:    "E3022",
					Message: fmt.Sprintf("value %s out of i16 range (-32768 to 32767)", val.String()),
				}
			}
			return &object.Integer{Value: new(big.Int).Set(val), DeclaredType: "i16"}
		},
	},

	// Converts a value to an i32 (signed 32-bit integer)
	"i32": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "i32() takes exactly 1 argument"}
			}
			val, err := extractIntValue(args[0], "i32")
			if err != nil {
				return err
			}
			minI32 := big.NewInt(-2147483648)
			maxI32 := big.NewInt(2147483647)
			if val.Cmp(minI32) < 0 || val.Cmp(maxI32) > 0 {
				return &object.Error{
					Code:    "E3022",
					Message: fmt.Sprintf("value %s out of i32 range (-2147483648 to 2147483647)", val.String()),
				}
			}
			return &object.Integer{Value: new(big.Int).Set(val), DeclaredType: "i32"}
		},
	},

	// Converts a value to an i64 (signed 64-bit integer)
	"i64": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "i64() takes exactly 1 argument"}
			}
			val, err := extractIntValue(args[0], "i64")
			if err != nil {
				return err
			}
			minI64 := new(big.Int).SetInt64(-9223372036854775808)
			maxI64 := new(big.Int).SetInt64(9223372036854775807)
			if val.Cmp(minI64) < 0 || val.Cmp(maxI64) > 0 {
				return &object.Error{
					Code:    "E3022",
					Message: fmt.Sprintf("value %s out of i64 range", val.String()),
				}
			}
			return &object.Integer{Value: new(big.Int).Set(val), DeclaredType: "i64"}
		},
	},

	// Converts a value to an i128 (signed 128-bit integer)
	"i128": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "i128() takes exactly 1 argument"}
			}
			val, err := extractIntValue(args[0], "i128")
			if err != nil {
				return err
			}
			// -2^127 to 2^127-1
			minI128 := new(big.Int).Lsh(big.NewInt(-1), 127)
			maxI128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1))
			if val.Cmp(minI128) < 0 || val.Cmp(maxI128) > 0 {
				return &object.Error{
					Code:    "E3022",
					Message: fmt.Sprintf("value %s out of i128 range", val.String()),
				}
			}
			return &object.Integer{Value: new(big.Int).Set(val), DeclaredType: "i128"}
		},
	},

	// Converts a value to an i256 (signed 256-bit integer)
	"i256": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "i256() takes exactly 1 argument"}
			}
			val, err := extractIntValue(args[0], "i256")
			if err != nil {
				return err
			}
			// -2^255 to 2^255-1
			minI256 := new(big.Int).Lsh(big.NewInt(-1), 255)
			maxI256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1))
			if val.Cmp(minI256) < 0 || val.Cmp(maxI256) > 0 {
				return &object.Error{
					Code:    "E3022",
					Message: fmt.Sprintf("value %s out of i256 range", val.String()),
				}
			}
			return &object.Integer{Value: new(big.Int).Set(val), DeclaredType: "i256"}
		},
	},

	// Converts a value to a u8 (unsigned 8-bit integer, 0 to 255)
	"u8": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "u8() takes exactly 1 argument"}
			}
			val, err := extractIntValue(args[0], "u8")
			if err != nil {
				return err
			}
			if val.Sign() < 0 || val.Cmp(big.NewInt(255)) > 0 {
				return &object.Error{
					Code:    "E3022",
					Message: fmt.Sprintf("value %s out of u8 range (0 to 255)", val.String()),
				}
			}
			return &object.Integer{Value: new(big.Int).Set(val), DeclaredType: "u8"}
		},
	},

	// Converts a value to a u16 (unsigned 16-bit integer, 0 to 65535)
	"u16": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "u16() takes exactly 1 argument"}
			}
			val, err := extractIntValue(args[0], "u16")
			if err != nil {
				return err
			}
			if val.Sign() < 0 || val.Cmp(big.NewInt(65535)) > 0 {
				return &object.Error{
					Code:    "E3022",
					Message: fmt.Sprintf("value %s out of u16 range (0 to 65535)", val.String()),
				}
			}
			return &object.Integer{Value: new(big.Int).Set(val), DeclaredType: "u16"}
		},
	},

	// Converts a value to a u32 (unsigned 32-bit integer, 0 to 4294967295)
	"u32": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "u32() takes exactly 1 argument"}
			}
			val, err := extractIntValue(args[0], "u32")
			if err != nil {
				return err
			}
			maxU32 := new(big.Int).SetUint64(4294967295)
			if val.Sign() < 0 || val.Cmp(maxU32) > 0 {
				return &object.Error{
					Code:    "E3022",
					Message: fmt.Sprintf("value %s out of u32 range (0 to 4294967295)", val.String()),
				}
			}
			return &object.Integer{Value: new(big.Int).Set(val), DeclaredType: "u32"}
		},
	},

	// Converts a value to a u64 (unsigned 64-bit integer)
	"u64": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "u64() takes exactly 1 argument"}
			}
			val, err := extractIntValue(args[0], "u64")
			if err != nil {
				return err
			}
			maxU64 := new(big.Int).SetUint64(18446744073709551615)
			if val.Sign() < 0 || val.Cmp(maxU64) > 0 {
				return &object.Error{
					Code:    "E3022",
					Message: fmt.Sprintf("value %s out of u64 range (0 to 18446744073709551615)", val.String()),
				}
			}
			return &object.Integer{Value: new(big.Int).Set(val), DeclaredType: "u64"}
		},
	},

	// Converts a value to a u128 (unsigned 128-bit integer)
	"u128": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "u128() takes exactly 1 argument"}
			}
			val, err := extractIntValue(args[0], "u128")
			if err != nil {
				return err
			}
			// 0 to 2^128-1
			maxU128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
			if val.Sign() < 0 || val.Cmp(maxU128) > 0 {
				return &object.Error{
					Code:    "E3022",
					Message: fmt.Sprintf("value %s out of u128 range", val.String()),
				}
			}
			return &object.Integer{Value: new(big.Int).Set(val), DeclaredType: "u128"}
		},
	},

	// Converts a value to a u256 (unsigned 256-bit integer)
	"u256": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "u256() takes exactly 1 argument"}
			}
			val, err := extractIntValue(args[0], "u256")
			if err != nil {
				return err
			}
			// 0 to 2^256-1
			maxU256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
			if val.Sign() < 0 || val.Cmp(maxU256) > 0 {
				return &object.Error{
					Code:    "E3022",
					Message: fmt.Sprintf("value %s out of u256 range", val.String()),
				}
			}
			return &object.Integer{Value: new(big.Int).Set(val), DeclaredType: "u256"}
		},
	},

	// Converts a value to a float32
	"f32": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "f32() takes exactly 1 argument"}
			}
			val, err := extractFloatValue(args[0], "f32")
			if err != nil {
				return err
			}
			// Convert to float32 and back to truncate precision
			f32 := float32(val)
			return &object.Float{Value: float64(f32), DeclaredType: "f32"}
		},
	},

	// Converts a value to a float64
	"f64": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "f64() takes exactly 1 argument"}
			}
			val, err := extractFloatValue(args[0], "f64")
			if err != nil {
				return err
			}
			return &object.Float{Value: val, DeclaredType: "f64"}
		},
	},

	// Reads a line of input from standard input
	"input": {
		Fn: func(args ...object.Object) object.Object {
			// If EOF was already reached, exit gracefully to prevent infinite loops
			if stdinEOFReached {
				os.Exit(0)
			}

			text, readErr := stdinReader.ReadString('\n')
			text = strings.TrimRight(text, "\r\n")

			// Mark EOF if reached (even if we got some text)
			if readErr == io.EOF {
				stdinEOFReached = true
			}

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

	// EXIT_SUCCESS constant - returns 0 (global builtin, no import needed)
	"EXIT_SUCCESS": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(0)}
		},
		IsConstant: true,
	},

	// EXIT_FAILURE constant - returns 1 (global builtin, no import needed)
	"EXIT_FAILURE": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(1)}
		},
		IsConstant: true,
	},

	// Exits the program with the given exit code (global builtin, no import needed)
	// Example:
	//   exit(0)            // exit successfully
	//   exit(EXIT_SUCCESS) // same as above
	//   exit(EXIT_FAILURE) // exit with error code 1
	"exit": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("exit() takes exactly 1 argument, got %d", len(args))}
			}
			code, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("exit() argument must be an integer, got %s", getEZTypeName(args[0]))}
			}
			os.Exit(int(code.Value.Int64()))
			return nil // Unreachable, but required by Go's type system
		},
	},

	// Sleeps for the given number of seconds
	"std.sleep_seconds": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("sleep_seconds() takes exactly 1 argument, got %d", len(args))}
			}
			seconds, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("sleep_seconds() argument must be an integer, got %s", getEZTypeName(args[0]))}
			}
			if seconds.Value.Sign() < 0 {
				return &object.Error{Code: "E7032", Message: "sleep_seconds() duration cannot be negative"}
			}
			time.Sleep(time.Duration(seconds.Value.Int64()) * time.Second)
			return object.NIL
		},
	},

	// Sleeps for the given number of milliseconds
	"std.sleep_milliseconds": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("sleep_milliseconds() takes exactly 1 argument, got %d", len(args))}
			}
			ms, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("sleep_milliseconds() argument must be an integer, got %s", getEZTypeName(args[0]))}
			}
			if ms.Value.Sign() < 0 {
				return &object.Error{Code: "E7032", Message: "sleep_milliseconds() duration cannot be negative"}
			}
			time.Sleep(time.Duration(ms.Value.Int64()) * time.Millisecond)
			return object.NIL
		},
	},

	// Sleeps for the given number of nanoseconds
	"std.sleep_nanoseconds": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("sleep_nanoseconds() takes exactly 1 argument, got %d", len(args))}
			}
			ns, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("sleep_nanoseconds() argument must be an integer, got %s", getEZTypeName(args[0]))}
			}
			if ns.Value.Sign() < 0 {
				return &object.Error{Code: "E7032", Message: "sleep_nanoseconds() duration cannot be negative"}
			}
			time.Sleep(time.Duration(ns.Value.Int64()) * time.Nanosecond)
			return object.NIL
		},
	},

	// Terminates the program with an error message (global builtin, no import needed)
	// Use panic() when something truly unrecoverable happens.
	// Example:
	//   panic("something went terribly wrong")
	"panic": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("panic() takes exactly 1 argument, got %d", len(args))}
			}
			msg, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("panic() argument must be a string, got %s", getEZTypeName(args[0]))}
			}
			return &object.Error{Code: "E5021", Message: fmt.Sprintf("panic: %s", msg.Value)}
		},
	},

	// Prints values to standard error (stderr) followed by a newline.
	//
	// Why use eprintln() instead of println()?
	// - println() writes to "stdout" (standard output) - for normal program output
	// - eprintln() writes to "stderr" (standard error) - for error messages and diagnostics
	//
	// Both display in your terminal by default, but they can be separated:
	//   ./ez myprogram.ez > output.txt    # stdout goes to file, stderr still shows
	//   ./ez myprogram.ez 2> errors.txt   # stderr goes to file, stdout still shows
	//
	// Example:
	//   eprintln("Error: file not found")
	//   eprintln("Warning:", "value is", 0)
	"std.eprintln": {
		Fn: func(args ...object.Object) object.Object {
			for i, arg := range args {
				if i > 0 {
					fmt.Fprint(os.Stderr, " ")
				}
				if str, ok := arg.(*object.String); ok {
					fmt.Fprint(os.Stderr, str.Value)
				} else {
					fmt.Fprint(os.Stderr, arg.Inspect())
				}
			}
			fmt.Fprintln(os.Stderr)
			return object.NIL
		},
	},

	// Prints values to standard error (stderr) WITHOUT a newline.
	// Same as eprintln() but doesn't add a newline at the end.
	// See eprintln() documentation for when to use stderr vs stdout.
	//
	// Example:
	//   eprintf("Loading...")
	//   // do work
	//   eprintln(" done!")  // Output: "Loading... done!"
	"std.eprintf": {
		Fn: func(args ...object.Object) object.Object {
			for i, arg := range args {
				if i > 0 {
					fmt.Fprint(os.Stderr, " ")
				}
				if str, ok := arg.(*object.String); ok {
					fmt.Fprint(os.Stderr, str.Value)
				} else {
					fmt.Fprint(os.Stderr, arg.Inspect())
				}
			}
			return object.NIL
		},
	},

	// Asserts that a condition is true, otherwise terminates with an error (global builtin, no import needed)
	// Use assert() to verify assumptions during development and catch bugs early.
	// Example:
	//   assert(x > 0, "x must be positive")
	//   assert(len(items) > 0, "items cannot be empty")
	"assert": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("assert() takes exactly 2 arguments, got %d", len(args))}
			}
			cond, ok := args[0].(*object.Boolean)
			if !ok {
				return &object.Error{Code: "E7008", Message: fmt.Sprintf("assert() first argument must be a boolean, got %s", getEZTypeName(args[0]))}
			}
			msg, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("assert() second argument must be a string, got %s", getEZTypeName(args[1]))}
			}
			if !cond.Value {
				return &object.Error{Code: "E5022", Message: fmt.Sprintf("assertion failed: %s", msg.Value)}
			}
			return object.NIL
		},
	},
}
