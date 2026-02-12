package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode"

	"github.com/marshallburns/ez/pkg/errors"
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.upper()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.upper()"), errors.TypeExpected("string"))}
			}
			return &object.String{Value: strings.ToUpper(str.Value)}
		},
	},

	// lower converts a string to lowercase.
	// Takes a string. Returns string.
	"strings.lower": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.lower()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.lower()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.trim()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.trim()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.contains()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.contains()"), errors.TypeExpected("string"))}
			}
			substr, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.contains()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.split()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.split()"), errors.TypeExpected("string"))}
			}
			sep, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.split()"), errors.TypeExpected("string"))}
			}
			parts := strings.Split(str.Value, sep.Value)
			elements := make([]object.Object, len(parts))
			for i, p := range parts {
				elements[i] = &object.String{Value: p}
			}
			return &object.Array{Elements: elements, ElementType: "string"}
		},
	},

	// lines splits a string by newlines (handles both \n and \r\n).
	// Takes a string. Returns array of strings.
	"strings.lines": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.lines()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.lines()"), errors.TypeExpected("string"))}
			}
			// Normalize \r\n to \n before splitting
			normalized := strings.ReplaceAll(str.Value, "\r\n", "\n")
			parts := strings.Split(normalized, "\n")
			elements := make([]object.Object, len(parts))
			for i, p := range parts {
				elements[i] = &object.String{Value: p}
			}
			return &object.Array{Elements: elements, ElementType: "string"}
		},
	},

	// words splits a string by whitespace.
	// Takes a string. Returns array of strings (empty words are excluded).
	"strings.words": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.words()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.words()"), errors.TypeExpected("string"))}
			}
			parts := strings.Fields(str.Value)
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.join()"))}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: fmt.Sprintf("%s requires an %s as first argument", errors.Ident("strings.join()"), errors.TypeExpected("array"))}
			}
			sep, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s separator", errors.Ident("strings.join()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 3 arguments", errors.Ident("strings.replace()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.replace()"), errors.TypeExpected("string"))}
			}
			old, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.replace()"), errors.TypeExpected("string"))}
			}
			newStr, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.replace()"), errors.TypeExpected("string"))}
			}
			return &object.String{Value: strings.ReplaceAll(str.Value, old.Value, newStr.Value)}
		},
	},

	// char_at returns the character at the given index.
	// Takes string and index (supports negative indices, -1 = last char).
	// Returns string (single character) or error if out of bounds.
	"strings.char_at": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.char_at()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("strings.char_at()"), errors.TypeExpected("string"))}
			}
			idx, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires an %s as second argument", errors.Ident("strings.char_at()"), errors.TypeExpected("integer"))}
			}

			runes := []rune(str.Value)
			runeLen := len(runes)
			index := int(idx.Value.Int64())

			// Handle negative indices
			if index < 0 {
				index = runeLen + index
			}

			// Check bounds
			if index < 0 || index >= runeLen {
				return &object.Error{Code: "E7005", Message: fmt.Sprintf("%s index out of bounds", errors.Ident("strings.char_at()"))}
			}

			return &object.String{Value: string(runes[index])}
		},
	},

	// insert inserts a substring at the given position.
	// Takes string, position, and substring. Returns string.
	// Position is clamped to valid range.
	"strings.insert": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 3 arguments (string, position, substring)", errors.Ident("strings.insert()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("strings.insert()"), errors.TypeExpected("string"))}
			}
			pos, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires an %s as second argument", errors.Ident("strings.insert()"), errors.TypeExpected("integer"))}
			}
			substr, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as third argument", errors.Ident("strings.insert()"), errors.TypeExpected("string"))}
			}

			runes := []rune(str.Value)
			runeLen := len(runes)
			position := int(pos.Value.Int64())

			// Clamp position to valid range
			if position < 0 {
				position = 0
			}
			if position > runeLen {
				position = runeLen
			}

			result := string(runes[:position]) + substr.Value + string(runes[position:])
			return &object.String{Value: result}
		},
	},

	// center pads a string on both sides to center it within the given width.
	// Takes string, width, and optional pad character (default space).
	// Returns unchanged string if already wider than target.
	"strings.center": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes 2 or 3 arguments (string, width, [pad_char])", errors.Ident("strings.center()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("strings.center()"), errors.TypeExpected("string"))}
			}
			width, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires an %s width", errors.Ident("strings.center()"), errors.TypeExpected("integer"))}
			}
			padChar := " "
			if len(args) == 3 {
				pad, ok := args[2].(*object.String)
				if !ok {
					return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as pad character", errors.Ident("strings.center()"), errors.TypeExpected("string"))}
				}
				if len(pad.Value) > 0 {
					padRunes := []rune(pad.Value)
					padChar = string(padRunes[0])
				}
			}

			targetWidth := int(width.Value.Int64())
			strRuneLen := len([]rune(str.Value))

			if strRuneLen >= targetWidth {
				return str
			}

			totalPadding := targetWidth - strRuneLen
			leftPadding := totalPadding / 2
			rightPadding := totalPadding - leftPadding

			result := strings.Repeat(padChar, leftPadding) + str.Value + strings.Repeat(padChar, rightPadding)
			return &object.String{Value: result}
		},
	},

	// remove removes the first occurrence of a substring.
	// Takes string and substring. Returns string.
	"strings.remove": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.remove()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.remove()"), errors.TypeExpected("string"))}
			}
			substr, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.remove()"), errors.TypeExpected("string"))}
			}
			return &object.String{Value: strings.Replace(str.Value, substr.Value, "", 1)}
		},
	},

	// remove_all removes all occurrences of a substring.
	// Takes string and substring. Returns string.
	"strings.remove_all": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.remove_all()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.remove_all()"), errors.TypeExpected("string"))}
			}
			substr, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.remove_all()"), errors.TypeExpected("string"))}
			}
			return &object.String{Value: strings.ReplaceAll(str.Value, substr.Value, "")}
		},
	},

	// index finds the first occurrence of a substring.
	// Takes string and substring. Returns int (-1 if not found).
	"strings.index": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.index()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.index()"), errors.TypeExpected("string"))}
			}
			substr, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.index()"), errors.TypeExpected("string"))}
			}
			return &object.Integer{Value: big.NewInt(int64(strings.Index(str.Value, substr.Value)))}
		},
	},

	// starts_with checks if a string starts with a prefix.
	// Takes string and prefix. Returns bool.
	"strings.starts_with": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.starts_with()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.starts_with()"), errors.TypeExpected("string"))}
			}
			prefix, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.starts_with()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.ends_with()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.ends_with()"), errors.TypeExpected("string"))}
			}
			suffix, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.ends_with()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.repeat()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("strings.repeat()"), errors.TypeExpected("string"))}
			}
			count, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires an %s as second argument", errors.Ident("strings.repeat()"), errors.TypeExpected("integer"))}
			}
			if count.Value.Sign() < 0 {
				return &object.Error{Code: "E10001", Message: fmt.Sprintf("%s count cannot be negative", errors.Ident("strings.repeat()"))}
			}
			return &object.String{Value: strings.Repeat(str.Value, int(count.Value.Int64()))}
		},
	},

	// slice extracts a portion of a string by index.
	// Takes string, start, and optional end. Returns string.
	"strings.slice": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes 2 or 3 arguments", errors.Ident("strings.slice()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("strings.slice()"), errors.TypeExpected("string"))}
			}
			start, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires %s indices", errors.Ident("strings.slice()"), errors.TypeExpected("integer"))}
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
					return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires %s indices", errors.Ident("strings.slice()"), errors.TypeExpected("integer"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.trim_left()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.trim_left()"), errors.TypeExpected("string"))}
			}
			return &object.String{Value: strings.TrimLeftFunc(str.Value, unicode.IsSpace)}
		},
	},

	// trim_right removes trailing whitespace from a string.
	// Takes a string. Returns string.
	"strings.trim_right": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.trim_right()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.trim_right()"), errors.TypeExpected("string"))}
			}
			return &object.String{Value: strings.TrimRightFunc(str.Value, unicode.IsSpace)}
		},
	},

	// pad_left pads a string on the left to reach a target width.
	// Takes string, width, and optional pad char. Returns string.
	"strings.pad_left": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes 2 or 3 arguments (string, width, [pad_char])", errors.Ident("strings.pad_left()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("strings.pad_left()"), errors.TypeExpected("string"))}
			}
			width, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires an %s width", errors.Ident("strings.pad_left()"), errors.TypeExpected("integer"))}
			}
			padChar := " "
			if len(args) == 3 {
				pad, ok := args[2].(*object.String)
				if !ok {
					return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as pad character", errors.Ident("strings.pad_left()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes 2 or 3 arguments (string, width, [pad_char])", errors.Ident("strings.pad_right()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("strings.pad_right()"), errors.TypeExpected("string"))}
			}
			width, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires an %s width", errors.Ident("strings.pad_right()"), errors.TypeExpected("integer"))}
			}
			padChar := " "
			if len(args) == 3 {
				pad, ok := args[2].(*object.String)
				if !ok {
					return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as pad character", errors.Ident("strings.pad_right()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.reverse()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.reverse()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.count()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.count()"), errors.TypeExpected("string"))}
			}
			substr, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.count()"), errors.TypeExpected("string"))}
			}
			return &object.Integer{Value: big.NewInt(int64(strings.Count(str.Value, substr.Value)))}
		},
	},

	// ============================================================================
	// String Checks
	// ============================================================================

	// is_alphanumeric checks if a string contains only letters and digits.
	// Takes a string. Returns bool.
	// Empty string returns false.
	"strings.is_alphanumeric": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.is_alphanumeric()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.is_alphanumeric()"), errors.TypeExpected("string"))}
			}
			if len(str.Value) == 0 {
				return object.FALSE
			}
			for _, r := range str.Value {
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
					return object.FALSE
				}
			}
			return object.TRUE
		},
	},

	// is_whitespace checks if a string contains only whitespace characters.
	// Takes a string. Returns bool.
	// Empty string returns false.
	"strings.is_whitespace": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.is_whitespace()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.is_whitespace()"), errors.TypeExpected("string"))}
			}
			if len(str.Value) == 0 {
				return object.FALSE
			}
			for _, r := range str.Value {
				if !unicode.IsSpace(r) {
					return object.FALSE
				}
			}
			return object.TRUE
		},
	},

	// is_lowercase checks if all letters in the string are lowercase.
	// Takes a string. Returns bool.
	// Empty string or string with no letters returns false.
	"strings.is_lowercase": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.is_lowercase()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.is_lowercase()"), errors.TypeExpected("string"))}
			}
			if len(str.Value) == 0 {
				return object.FALSE
			}
			hasLetter := false
			for _, r := range str.Value {
				if unicode.IsLetter(r) {
					hasLetter = true
					if !unicode.IsLower(r) {
						return object.FALSE
					}
				}
			}
			if !hasLetter {
				return object.FALSE
			}
			return object.TRUE
		},
	},

	// is_uppercase checks if all letters in the string are uppercase.
	// Takes a string. Returns bool.
	// Empty string or string with no letters returns false.
	"strings.is_uppercase": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.is_uppercase()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.is_uppercase()"), errors.TypeExpected("string"))}
			}
			if len(str.Value) == 0 {
				return object.FALSE
			}
			hasLetter := false
			for _, r := range str.Value {
				if unicode.IsLetter(r) {
					hasLetter = true
					if !unicode.IsUpper(r) {
						return object.FALSE
					}
				}
			}
			if !hasLetter {
				return object.FALSE
			}
			return object.TRUE
		},
	},

	// is_ascii checks if all characters in the string are ASCII (0-127).
	// Takes a string. Returns bool.
	// Empty string returns true (no non-ASCII chars).
	"strings.is_ascii": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.is_ascii()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.is_ascii()"), errors.TypeExpected("string"))}
			}
			for _, r := range str.Value {
				if r > 127 {
					return object.FALSE
				}
			}
			return object.TRUE
		},
	},

	// is_empty checks if a string is empty or contains only whitespace.
	// Takes a string. Returns bool.
	"strings.is_empty": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.is_empty()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.is_empty()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.chars()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.chars()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.from_chars()"))}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: fmt.Sprintf("%s requires an %s argument", errors.Ident("strings.from_chars()"), errors.TypeExpected("array"))}
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
					return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires an array of %s", errors.Ident("strings.from_chars()"), errors.TypeExpected("chars"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.last_index()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.last_index()"), errors.TypeExpected("string"))}
			}
			substr, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.last_index()"), errors.TypeExpected("string"))}
			}
			return &object.Integer{Value: big.NewInt(int64(strings.LastIndex(str.Value, substr.Value)))}
		},
	},

	// capitalize uppercases the first character of a string.
	// Takes a string. Returns string.
	"strings.capitalize": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.capitalize()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.capitalize()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.title()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.title()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 4 arguments (string, old, new, n)", errors.Ident("strings.replace_n()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("strings.replace_n()"), errors.TypeExpected("string"))}
			}
			old, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("strings.replace_n()"), errors.TypeExpected("string"))}
			}
			newStr, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as third argument", errors.Ident("strings.replace_n()"), errors.TypeExpected("string"))}
			}
			n, ok := args[3].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires an %s as fourth argument", errors.Ident("strings.replace_n()"), errors.TypeExpected("integer"))}
			}
			return &object.String{Value: strings.Replace(str.Value, old.Value, newStr.Value, int(n.Value.Int64()))}
		},
	},

	// is_numeric checks if a string contains only digits.
	// Takes a string. Returns bool.
	"strings.is_numeric": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.is_numeric()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.is_numeric()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.is_alpha()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.is_alpha()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 3 arguments (string, length, suffix)", errors.Ident("strings.truncate()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("strings.truncate()"), errors.TypeExpected("string"))}
			}
			maxLen, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires an %s as second argument", errors.Ident("strings.truncate()"), errors.TypeExpected("integer"))}
			}
			suffix, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as third argument", errors.Ident("strings.truncate()"), errors.TypeExpected("string"))}
			}

			if maxLen.Value.Sign() < 0 {
				return &object.Error{Code: "E10001", Message: fmt.Sprintf("%s length cannot be negative", errors.Ident("strings.truncate()"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("strings.compare()"))}
			}
			a, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.compare()"), errors.TypeExpected("string"))}
			}
			b, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires %s arguments", errors.Ident("strings.compare()"), errors.TypeExpected("string"))}
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
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.to_int()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.to_int()"), errors.TypeExpected("string"))}
			}
			// Trim whitespace before parsing
			trimmed := strings.TrimSpace(str.Value)
			val, ok := new(big.Int).SetString(trimmed, 10)
			if !ok {
				return &object.Error{Code: "E7014", Message: fmt.Sprintf("%s cannot parse \"%s\" as %s", errors.Ident("strings.to_int()"), str.Value, errors.TypeExpected("integer"))}
			}
			return &object.Integer{Value: val}
		},
	},

	// to_float parses a string as a floating-point number.
	// Takes a string. Returns float or Error.
	"strings.to_float": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.to_float()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.to_float()"), errors.TypeExpected("string"))}
			}
			// Trim whitespace before parsing
			trimmed := strings.TrimSpace(str.Value)
			val, err := strconv.ParseFloat(trimmed, 64)
			if err != nil {
				return &object.Error{Code: "E7014", Message: fmt.Sprintf("%s cannot parse \"%s\" as %s", errors.Ident("strings.to_float()"), str.Value, errors.TypeExpected("float"))}
			}
			return &object.Float{Value: val}
		},
	},

	// to_bool parses a string as a boolean.
	// Takes a string ("true"/"false"/"1"/"0"/etc). Returns bool or Error.
	"strings.to_bool": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("strings.to_bool()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("strings.to_bool()"), errors.TypeExpected("string"))}
			}
			// Trim whitespace and lowercase before parsing
			trimmed := strings.ToLower(strings.TrimSpace(str.Value))
			switch trimmed {
			case "true", "1", "yes", "on":
				return object.TRUE
			case "false", "0", "no", "off":
				return object.FALSE
			default:
				return &object.Error{Code: "E7014", Message: fmt.Sprintf("%s cannot parse \"%s\" as %s", errors.Ident("strings.to_bool()"), str.Value, errors.TypeExpected("boolean"))}
			}
		},
	},
}
