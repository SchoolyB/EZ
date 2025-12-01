package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"strings"

	"github.com/marshallburns/ez/pkg/object"
)

// StringsBuiltins contains the strings module functions
var StringsBuiltins = map[string]*object.Builtin{
	"strings.len": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E10001", Message: "strings.len() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.len() requires a string argument"}
			}
			return &object.Integer{Value: int64(len(str.Value))}
		},
	},

	"strings.upper": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E10001", Message: "strings.upper() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.upper() requires a string argument"}
			}
			return &object.String{Value: strings.ToUpper(str.Value)}
		},
	},

	"strings.lower": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E10001", Message: "strings.lower() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.lower() requires a string argument"}
			}
			return &object.String{Value: strings.ToLower(str.Value)}
		},
	},

	"strings.trim": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E10001", Message: "strings.trim() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.trim() requires a string argument"}
			}
			return &object.String{Value: strings.TrimSpace(str.Value)}
		},
	},

	"strings.contains": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E10001", Message: "strings.contains() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.contains() requires string arguments"}
			}
			substr, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.contains() requires string arguments"}
			}
			if strings.Contains(str.Value, substr.Value) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	"strings.split": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E10001", Message: "strings.split() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.split() requires string arguments"}
			}
			sep, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.split() requires string arguments"}
			}
			parts := strings.Split(str.Value, sep.Value)
			elements := make([]object.Object, len(parts))
			for i, p := range parts {
				elements[i] = &object.String{Value: p}
			}
			return &object.Array{Elements: elements}
		},
	},

	"strings.join": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E10001", Message: "strings.join() takes exactly 2 arguments"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E10004", Message: "strings.join() requires an array as first argument"}
			}
			sep, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E10005", Message: "strings.join() requires a string separator"}
			}
			parts := make([]string, len(arr.Elements))
			for i, el := range arr.Elements {
				// Extract raw string value without quotes
				if str, ok := el.(*object.String); ok {
					parts[i] = str.Value
				} else {
					parts[i] = el.Inspect()
				}
			}
			return &object.String{Value: strings.Join(parts, sep.Value)}
		},
	},

	"strings.replace": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E10001", Message: "strings.replace() takes exactly 3 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.replace() requires string arguments"}
			}
			old, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.replace() requires string arguments"}
			}
			newStr, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.replace() requires string arguments"}
			}
			return &object.String{Value: strings.ReplaceAll(str.Value, old.Value, newStr.Value)}
		},
	},

	"strings.index": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E10001", Message: "strings.index() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.index() requires string arguments"}
			}
			substr, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.index() requires string arguments"}
			}
			return &object.Integer{Value: int64(strings.Index(str.Value, substr.Value))}
		},
	},

	"strings.starts_with": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E10001", Message: "strings.starts_with() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.starts_with() requires string arguments"}
			}
			prefix, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.starts_with() requires string arguments"}
			}
			if strings.HasPrefix(str.Value, prefix.Value) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	"strings.ends_with": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E10001", Message: "strings.ends_with() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.ends_with() requires string arguments"}
			}
			suffix, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.ends_with() requires string arguments"}
			}
			if strings.HasSuffix(str.Value, suffix.Value) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	"strings.repeat": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E10001", Message: "strings.repeat() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E10003", Message: "strings.repeat() requires a string as first argument"}
			}
			count, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E10006", Message: "strings.repeat() requires an integer as second argument"}
			}
			if count.Value < 0 {
				return &object.Error{Code: "E10007", Message: "strings.repeat() count cannot be negative"}
			}
			return &object.String{Value: strings.Repeat(str.Value, int(count.Value))}
		},
	},
}
