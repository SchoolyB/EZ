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
	// Standard library print functions (with std. prefix)
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
	"std.read_int": {
		Fn: func(args ...Object) Object {
			reader := bufio.NewReader(os.Stdin)
			text, _ := reader.ReadString('\n')
			// Trim newline
			text = text[:len(text)-1]
			val, err := strconv.ParseInt(text, 10, 64)
			if err != nil {
				return newError("invalid integer input")
			}
			return &Integer{Value: val}
		},
	},

	// Global built-ins (no prefix required)
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
	"typeof": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("typeof() takes exactly 1 argument")
			}
			return &String{Value: string(args[0].Type())}
		},
	},
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
	"string": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("string() takes exactly 1 argument")
			}
			return &String{Value: args[0].Inspect()}
		},
	},
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

// formatString is a simple printf-style formatter
func formatString(format string, args []Object) string {
	result := ""
	argIdx := 0

	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			if argIdx >= len(args) {
				result += string(format[i])
				continue
			}

			switch format[i+1] {
			case 's':
				result += args[argIdx].Inspect()
				argIdx++
				i++
			case 'd':
				if intObj, ok := args[argIdx].(*Integer); ok {
					result += fmt.Sprintf("%d", intObj.Value)
				} else {
					result += args[argIdx].Inspect()
				}
				argIdx++
				i++
			case 'f':
				if floatObj, ok := args[argIdx].(*Float); ok {
					result += fmt.Sprintf("%f", floatObj.Value)
				} else if intObj, ok := args[argIdx].(*Integer); ok {
					result += fmt.Sprintf("%f", float64(intObj.Value))
				} else {
					result += args[argIdx].Inspect()
				}
				argIdx++
				i++
			case '%':
				result += "%"
				i++
			default:
				result += string(format[i])
			}
		} else if format[i] == '\\' && i+1 < len(format) {
			switch format[i+1] {
			case 'n':
				result += "\n"
				i++
			case 't':
				result += "\t"
				i++
			case '\\':
				result += "\\"
				i++
			default:
				result += string(format[i])
			}
		} else {
			result += string(format[i])
		}
	}

	return result
}
