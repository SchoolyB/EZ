package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import "strings"

var stringsBuiltins = map[string]*Builtin{
	"strings.len": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("strings.len() takes exactly 1 argument")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("strings.len() requires a string argument")
			}
			return &Integer{Value: int64(len(str.Value))}
		},
	},
	"strings.upper": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("strings.upper() takes exactly 1 argument")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("strings.upper() requires a string argument")
			}
			return &String{Value: strings.ToUpper(str.Value)}
		},
	},
	"strings.lower": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("strings.lower() takes exactly 1 argument")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("strings.lower() requires a string argument")
			}
			return &String{Value: strings.ToLower(str.Value)}
		},
	},
	"strings.trim": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("strings.trim() takes exactly 1 argument")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("strings.trim() requires a string argument")
			}
			return &String{Value: strings.TrimSpace(str.Value)}
		},
	},
	"strings.contains": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("strings.contains() takes exactly 2 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("strings.contains() requires string arguments")
			}
			substr, ok := args[1].(*String)
			if !ok {
				return newError("strings.contains() requires string arguments")
			}
			if strings.Contains(str.Value, substr.Value) {
				return TRUE
			}
			return FALSE
		},
	},
	"strings.split": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("strings.split() takes exactly 2 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("strings.split() requires string arguments")
			}
			sep, ok := args[1].(*String)
			if !ok {
				return newError("strings.split() requires string arguments")
			}
			parts := strings.Split(str.Value, sep.Value)
			elements := make([]Object, len(parts))
			for i, p := range parts {
				elements[i] = &String{Value: p}
			}
			return &Array{Elements: elements}
		},
	},
	"strings.join": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("strings.join() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return newError("strings.join() requires an array as first argument")
			}
			sep, ok := args[1].(*String)
			if !ok {
				return newError("strings.join() requires a string separator")
			}
			parts := make([]string, len(arr.Elements))
			for i, el := range arr.Elements {
				parts[i] = el.Inspect()
			}
			return &String{Value: strings.Join(parts, sep.Value)}
		},
	},
	"strings.replace": {
		Fn: func(args ...Object) Object {
			if len(args) != 3 {
				return newError("strings.replace() takes exactly 3 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("strings.replace() requires string arguments")
			}
			old, ok := args[1].(*String)
			if !ok {
				return newError("strings.replace() requires string arguments")
			}
			newStr, ok := args[2].(*String)
			if !ok {
				return newError("strings.replace() requires string arguments")
			}
			return &String{Value: strings.ReplaceAll(str.Value, old.Value, newStr.Value)}
		},
	},
	"strings.index": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("strings.index() takes exactly 2 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("strings.index() requires string arguments")
			}
			substr, ok := args[1].(*String)
			if !ok {
				return newError("strings.index() requires string arguments")
			}
			return &Integer{Value: int64(strings.Index(str.Value, substr.Value))}
		},
	},
	"strings.starts_with": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("strings.starts_with() takes exactly 2 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("strings.starts_with() requires string arguments")
			}
			prefix, ok := args[1].(*String)
			if !ok {
				return newError("strings.starts_with() requires string arguments")
			}
			if strings.HasPrefix(str.Value, prefix.Value) {
				return TRUE
			}
			return FALSE
		},
	},
	"strings.ends_with": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("strings.ends_with() takes exactly 2 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("strings.ends_with() requires string arguments")
			}
			suffix, ok := args[1].(*String)
			if !ok {
				return newError("strings.ends_with() requires string arguments")
			}
			if strings.HasSuffix(str.Value, suffix.Value) {
				return TRUE
			}
			return FALSE
		},
	},
}

func init() {
	for name, builtin := range stringsBuiltins {
		builtins[name] = builtin
	}
}
