package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"math/big"
	"strconv"
	"strings"
	"unicode"

	"github.com/marshallburns/ez/pkg/object"
)

// StringsBuiltins contains the strings module functions for string manipulation.
// All functions are prefixed with "strings." and operate on string values.
var StringsBuiltins = map[string]*object.Builtin{
	// Converts a string to uppercase
	// strings.upper(s string) -> string
	"strings.upper": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.upper() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.upper() requires a string argument"}
			}
			return &object.String{Value: strings.ToUpper(str.Value)}
		},
	},

	// Converts a string to lowercase
	// strings.lower(s string) -> string
	"strings.lower": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.lower() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.lower() requires a string argument"}
			}
			return &object.String{Value: strings.ToLower(str.Value)}
		},
	},

	// Removes leading and trailing whitespace from a string
	// strings.trim(s string) -> string
	"strings.trim": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.trim() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.trim() requires a string argument"}
			}
			return &object.String{Value: strings.TrimSpace(str.Value)}
		},
	},

	// Checks if a string contains a substring
	// strings.contains(s string, substr string) -> bool
	"strings.contains": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "strings.contains() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.contains() requires string arguments"}
			}
			substr, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.contains() requires string arguments"}
			}
			if strings.Contains(str.Value, substr.Value) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// Splits a string by a separator into an array of strings
	// strings.split(s string, sep string) -> [string]
	"strings.split": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "strings.split() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.split() requires string arguments"}
			}
			sep, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.split() requires string arguments"}
			}
			parts := strings.Split(str.Value, sep.Value)
			elements := make([]object.Object, len(parts))
			for i, p := range parts {
				elements[i] = &object.String{Value: p}
			}
			return &object.Array{Elements: elements}
		},
	},

	// Joins an array of strings with a separator
	// strings.join(arr [string], sep string) -> string
	"strings.join": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "strings.join() takes exactly 2 arguments"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "strings.join() requires an array as first argument"}
			}
			sep, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.join() requires a string separator"}
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

	// Replaces all occurrences of old with new in the string
	// strings.replace(s string, old string, new string) -> string
	"strings.replace": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: "strings.replace() takes exactly 3 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.replace() requires string arguments"}
			}
			old, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.replace() requires string arguments"}
			}
			newStr, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.replace() requires string arguments"}
			}
			return &object.String{Value: strings.ReplaceAll(str.Value, old.Value, newStr.Value)}
		},
	},

	// Returns the index of the first occurrence of substr, or -1 if not found
	// strings.index(s string, substr string) -> int
	"strings.index": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "strings.index() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.index() requires string arguments"}
			}
			substr, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.index() requires string arguments"}
			}
			return &object.Integer{Value: big.NewInt(int64(strings.Index(str.Value, substr.Value)))}
		},
	},

	// Returns true if the string starts with the prefix
	// strings.starts_with(s string, prefix string) -> bool
	"strings.starts_with": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "strings.starts_with() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.starts_with() requires string arguments"}
			}
			prefix, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.starts_with() requires string arguments"}
			}
			if strings.HasPrefix(str.Value, prefix.Value) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// Returns true if the string ends with the suffix
	// strings.ends_with(s string, suffix string) -> bool
	"strings.ends_with": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "strings.ends_with() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.ends_with() requires string arguments"}
			}
			suffix, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.ends_with() requires string arguments"}
			}
			if strings.HasSuffix(str.Value, suffix.Value) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// Repeats a string n times
	// strings.repeat(s string, count int) -> string
	"strings.repeat": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "strings.repeat() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.repeat() requires a string as first argument"}
			}
			count, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "strings.repeat() requires an integer as second argument"}
			}
			if count.Value.Sign() < 0 {
				return &object.Error{Code: "E10001", Message: "strings.repeat() count cannot be negative"}
			}
			return &object.String{Value: strings.Repeat(str.Value, int(count.Value.Int64()))}
		},
	},

	// Returns a substring from start index to optional end index (supports negative indices)
	// strings.slice(s string, start int, end int?) -> string
	"strings.slice": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: "strings.slice() takes 2 or 3 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.slice() requires a string as first argument"}
			}
			start, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "strings.slice() requires integer indices"}
			}

			// Convert to runes for proper UTF-8 character handling
			runes := []rune(str.Value)
			runeLen := len(runes)

			startIdx := int(start.Value.Int64())
			if startIdx < 0 {
				startIdx = runeLen + startIdx
			}
			if startIdx < 0 {
				startIdx = 0
			}

			endIdx := runeLen
			if len(args) == 3 {
				end, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "strings.slice() requires integer indices"}
				}
				endIdx = int(end.Value.Int64())
				if endIdx < 0 {
					endIdx = runeLen + endIdx
				}
			}

			if startIdx > runeLen {
				startIdx = runeLen
			}
			if endIdx > runeLen {
				endIdx = runeLen
			}
			if startIdx > endIdx {
				return &object.String{Value: ""}
			}

			return &object.String{Value: string(runes[startIdx:endIdx])}
		},
	},

	// Removes leading whitespace from a string
	// strings.trim_left(s string) -> string
	"strings.trim_left": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.trim_left() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.trim_left() requires a string argument"}
			}
			return &object.String{Value: strings.TrimLeftFunc(str.Value, unicode.IsSpace)}
		},
	},

	// Removes trailing whitespace from a string
	// strings.trim_right(s string) -> string
	"strings.trim_right": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.trim_right() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.trim_right() requires a string argument"}
			}
			return &object.String{Value: strings.TrimRightFunc(str.Value, unicode.IsSpace)}
		},
	},

	// Pads a string on the left to reach target width
	// strings.pad_left(s string, width int, pad_char string?) -> string
	"strings.pad_left": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: "strings.pad_left() takes 2 or 3 arguments (string, width, [pad_char])"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.pad_left() requires a string as first argument"}
			}
			width, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "strings.pad_left() requires an integer width"}
			}
			padChar := " "
			if len(args) == 3 {
				pad, ok := args[2].(*object.String)
				if !ok {
					return &object.Error{Code: "E7003", Message: "strings.pad_left() requires a string as pad character"}
				}
				if len(pad.Value) > 0 {
					// Use first rune of pad string for proper UTF-8 handling
					padRunes := []rune(pad.Value)
					padChar = string(padRunes[0])
				}
			}
			targetWidth := int(width.Value.Int64())
			// Use rune count instead of byte length for proper UTF-8 handling
			strRuneLen := len([]rune(str.Value))
			if strRuneLen >= targetWidth {
				return str
			}
			padding := strings.Repeat(padChar, targetWidth-strRuneLen)
			return &object.String{Value: padding + str.Value}
		},
	},

	// Pads a string on the right to reach target width
	// strings.pad_right(s string, width int, pad_char string?) -> string
	"strings.pad_right": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: "strings.pad_right() takes 2 or 3 arguments (string, width, [pad_char])"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.pad_right() requires a string as first argument"}
			}
			width, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "strings.pad_right() requires an integer width"}
			}
			padChar := " "
			if len(args) == 3 {
				pad, ok := args[2].(*object.String)
				if !ok {
					return &object.Error{Code: "E7003", Message: "strings.pad_right() requires a string as pad character"}
				}
				if len(pad.Value) > 0 {
					// Use first rune of pad string for proper UTF-8 handling
					padRunes := []rune(pad.Value)
					padChar = string(padRunes[0])
				}
			}
			targetWidth := int(width.Value.Int64())
			// Use rune count instead of byte length for proper UTF-8 handling
			strRuneLen := len([]rune(str.Value))
			if strRuneLen >= targetWidth {
				return str
			}
			padding := strings.Repeat(padChar, targetWidth-strRuneLen)
			return &object.String{Value: str.Value + padding}
		},
	},

	// Reverses a string (handles UTF-8 correctly)
	// strings.reverse(s string) -> string
	"strings.reverse": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.reverse() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.reverse() requires a string argument"}
			}
			runes := []rune(str.Value)
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			return &object.String{Value: string(runes)}
		},
	},

	// Counts occurrences of substr in the string
	// strings.count(s string, substr string) -> int
	"strings.count": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "strings.count() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.count() requires string arguments"}
			}
			substr, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.count() requires string arguments"}
			}
			return &object.Integer{Value: big.NewInt(int64(strings.Count(str.Value, substr.Value)))}
		},
	},

	// Returns true if the string is empty or contains only whitespace
	// strings.is_empty(s string) -> bool
	"strings.is_empty": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.is_empty() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.is_empty() requires a string argument"}
			}
			if len(strings.TrimSpace(str.Value)) == 0 {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// Splits a string into an array of characters
	// strings.chars(s string) -> [char]
	"strings.chars": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.chars() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.chars() requires a string argument"}
			}
			runes := []rune(str.Value)
			elements := make([]object.Object, len(runes))
			for i, r := range runes {
				elements[i] = &object.Char{Value: r}
			}
			return &object.Array{Elements: elements, Mutable: true}
		},
	},

	// Creates a string from an array of characters
	// strings.from_chars(arr [char]) -> string
	"strings.from_chars": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.from_chars() takes exactly 1 argument"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "strings.from_chars() requires an array argument"}
			}
			runes := make([]rune, len(arr.Elements))
			for i, el := range arr.Elements {
				switch v := el.(type) {
				case *object.Char:
					runes[i] = v.Value
				case *object.String:
					if len(v.Value) > 0 {
						runes[i] = []rune(v.Value)[0]
					}
				default:
					return &object.Error{Code: "E7003", Message: "strings.from_chars() requires an array of chars"}
				}
			}
			return &object.String{Value: string(runes)}
		},
	},

	// Returns the index of the last occurrence of substr, or -1 if not found
	// strings.last_index(s string, substr string) -> int
	"strings.last_index": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "strings.last_index() takes exactly 2 arguments"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.last_index() requires string arguments"}
			}
			substr, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.last_index() requires string arguments"}
			}
			return &object.Integer{Value: big.NewInt(int64(strings.LastIndex(str.Value, substr.Value)))}
		},
	},

	// Capitalizes the first character of a string
	// strings.capitalize(s string) -> string
	"strings.capitalize": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.capitalize() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.capitalize() requires a string argument"}
			}
			if len(str.Value) == 0 {
				return str
			}
			runes := []rune(str.Value)
			runes[0] = unicode.ToUpper(runes[0])
			return &object.String{Value: string(runes)}
		},
	},

	// Capitalizes the first character of each word
	// strings.title(s string) -> string
	"strings.title": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.title() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.title() requires a string argument"}
			}
			// Title case: capitalize first letter of each word
			runes := []rune(str.Value)
			capitalizeNext := true
			for i, r := range runes {
				if unicode.IsSpace(r) {
					capitalizeNext = true
				} else if capitalizeNext {
					runes[i] = unicode.ToUpper(r)
					capitalizeNext = false
				}
			}
			return &object.String{Value: string(runes)}
		},
	},

	// Replaces the first n occurrences of old with new
	// strings.replace_n(s string, old string, new string, n int) -> string
	"strings.replace_n": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 4 {
				return &object.Error{Code: "E7001", Message: "strings.replace_n() takes exactly 4 arguments (string, old, new, n)"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.replace_n() requires a string as first argument"}
			}
			old, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.replace_n() requires a string as second argument"}
			}
			newStr, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.replace_n() requires a string as third argument"}
			}
			n, ok := args[3].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "strings.replace_n() requires an integer as fourth argument"}
			}
			return &object.String{Value: strings.Replace(str.Value, old.Value, newStr.Value, int(n.Value.Int64()))}
		},
	},

	// Returns true if the string contains only numeric digits
	// strings.is_numeric(s string) -> bool
	"strings.is_numeric": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.is_numeric() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.is_numeric() requires a string argument"}
			}
			if len(str.Value) == 0 {
				return object.FALSE
			}
			for _, r := range str.Value {
				if !unicode.IsDigit(r) {
					return object.FALSE
				}
			}
			return object.TRUE
		},
	},

	// Returns true if the string contains only alphabetic characters
	// strings.is_alpha(s string) -> bool
	"strings.is_alpha": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.is_alpha() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.is_alpha() requires a string argument"}
			}
			if len(str.Value) == 0 {
				return object.FALSE
			}
			for _, r := range str.Value {
				if !unicode.IsLetter(r) {
					return object.FALSE
				}
			}
			return object.TRUE
		},
	},

	// Truncates a string to max length with a suffix
	// strings.truncate(s string, max_len int, suffix string) -> string
	"strings.truncate": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: "strings.truncate() takes exactly 3 arguments (string, length, suffix)"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.truncate() requires a string as first argument"}
			}
			maxLen, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "strings.truncate() requires an integer as second argument"}
			}
			suffix, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.truncate() requires a string as third argument"}
			}

			if maxLen.Value.Sign() < 0 {
				return &object.Error{Code: "E10001", Message: "strings.truncate() length cannot be negative"}
			}

			runes := []rune(str.Value)
			targetLen := int(maxLen.Value.Int64())

			if len(runes) <= targetLen {
				return str
			}

			suffixRunes := []rune(suffix.Value)
			if targetLen <= len(suffixRunes) {
				// If max length is less than or equal to suffix length, just return truncated suffix
				return &object.String{Value: string(suffixRunes[:targetLen])}
			}

			truncateAt := targetLen - len(suffixRunes)
			return &object.String{Value: string(runes[:truncateAt]) + suffix.Value}
		},
	},

	// Lexicographically compares two strings: returns -1, 0, or 1
	// strings.compare(a string, b string) -> int
	"strings.compare": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "strings.compare() takes exactly 2 arguments"}
			}
			a, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.compare() requires string arguments"}
			}
			b, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.compare() requires string arguments"}
			}
			return &object.Integer{Value: big.NewInt(int64(strings.Compare(a.Value, b.Value)))}
		},
	},

	// Parses a string as an integer
	// strings.to_int(s string) -> int
	"strings.to_int": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.to_int() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.to_int() requires a string argument"}
			}
			// Trim whitespace before parsing
			trimmed := strings.TrimSpace(str.Value)
			val, ok := new(big.Int).SetString(trimmed, 10)
			if !ok {
				return &object.Error{Code: "E7014", Message: "strings.to_int() cannot parse \"" + str.Value + "\" as integer"}
			}
			return &object.Integer{Value: val}
		},
	},

	// Parses a string as a float
	// strings.to_float(s string) -> float
	"strings.to_float": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.to_float() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.to_float() requires a string argument"}
			}
			// Trim whitespace before parsing
			trimmed := strings.TrimSpace(str.Value)
			val, err := strconv.ParseFloat(trimmed, 64)
			if err != nil {
				return &object.Error{Code: "E7014", Message: "strings.to_float() cannot parse \"" + str.Value + "\" as float"}
			}
			return &object.Float{Value: val}
		},
	},

	// Parses a string as a boolean (accepts: true/false, 1/0, yes/no, on/off)
	// strings.to_bool(s string) -> bool
	"strings.to_bool": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "strings.to_bool() takes exactly 1 argument"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "strings.to_bool() requires a string argument"}
			}
			// Trim whitespace and lowercase before parsing
			trimmed := strings.ToLower(strings.TrimSpace(str.Value))
			switch trimmed {
			case "true", "1", "yes", "on":
				return object.TRUE
			case "false", "0", "no", "off":
				return object.FALSE
			default:
				return &object.Error{Code: "E7014", Message: "strings.to_bool() cannot parse \"" + str.Value + "\" as boolean"}
			}
		},
	},
}
