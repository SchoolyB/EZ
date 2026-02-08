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

// StringsBuiltins contains the strings module functions
var StringsBuiltins = map[string]*object.Builtin{
	// ============================================================================
	// Case Conversion
	// ============================================================================

	// upper converts a string to uppercase.
	// Takes a string. Returns string.
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

	// lower converts a string to lowercase.
	// Takes a string. Returns string.
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

	// ============================================================================
	// Trimming
	// ============================================================================

	// trim removes leading and trailing whitespace from a string.
	// Takes a string. Returns string.
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

	// ============================================================================
	// Search Operations
	// ============================================================================

	// contains checks if a string contains a substring.
	// Takes string and substring. Returns bool.
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

	// ============================================================================
	// String Manipulation
	// ============================================================================

	// split divides a string into an array by a separator.
	// Takes string and separator. Returns array of strings.
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
			return &object.Array{Elements: elements, ElementType: "string"}
		},
	},

	// join concatenates array elements into a string with a separator.
	// Takes array and separator. Returns string.
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

	// replace replaces all occurrences of old with new in a string.
	// Takes string, old, and new. Returns string.
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

	// index finds the first occurrence of a substring.
	// Takes string and substring. Returns int (-1 if not found).
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

	// starts_with checks if a string starts with a prefix.
	// Takes string and prefix. Returns bool.
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

	// ends_with checks if a string ends with a suffix.
	// Takes string and suffix. Returns bool.
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

	// repeat repeats a string n times.
	// Takes string and count. Returns string.
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

	// slice extracts a portion of a string by index.
	// Takes string, start, and optional end. Returns string.
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

	// trim_left removes leading whitespace from a string.
	// Takes a string. Returns string.
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

	// trim_right removes trailing whitespace from a string.
	// Takes a string. Returns string.
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

	// pad_left pads a string on the left to reach a target width.
	// Takes string, width, and optional pad char. Returns string.
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

	// pad_right pads a string on the right to reach a target width.
	// Takes string, width, and optional pad char. Returns string.
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

	// reverse reverses the characters in a string.
	// Takes a string. Returns string.
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

	// count counts non-overlapping occurrences of a substring.
	// Takes string and substring. Returns int.
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

	// ============================================================================
	// String Checks
	// ============================================================================

	// is_empty checks if a string is empty or contains only whitespace.
	// Takes a string. Returns bool.
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

	// ============================================================================
	// Character Operations
	// ============================================================================

	// chars splits a string into an array of characters.
	// Takes a string. Returns array of chars.
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
			return &object.Array{Elements: elements, Mutable: true, ElementType: "char"}
		},
	},

	// from_chars creates a string from an array of characters.
	// Takes array of chars. Returns string.
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

	// last_index finds the last occurrence of a substring.
	// Takes string and substring. Returns int (-1 if not found).
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

	// capitalize uppercases the first character of a string.
	// Takes a string. Returns string.
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

	// title converts a string to title case (capitalize each word).
	// Takes a string. Returns string.
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

	// replace_n replaces the first n occurrences of old with new.
	// Takes string, old, new, and count. Returns string.
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

	// is_numeric checks if a string contains only digits.
	// Takes a string. Returns bool.
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

	// is_alpha checks if a string contains only letters.
	// Takes a string. Returns bool.
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

	// truncate shortens a string to a max length with a suffix.
	// Takes string, max length, and suffix. Returns string.
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

	// compare compares two strings lexicographically.
	// Takes two strings. Returns int (-1, 0, or 1).
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

	// ============================================================================
	// Type Conversion
	// ============================================================================

	// to_int parses a string as an integer.
	// Takes a string. Returns int or Error.
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

	// to_float parses a string as a floating-point number.
	// Takes a string. Returns float or Error.
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

	// to_bool parses a string as a boolean.
	// Takes a string ("true"/"false"/"1"/"0"/etc). Returns bool or Error.
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
