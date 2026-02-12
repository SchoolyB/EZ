package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"regexp"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/object"
)

// RegexBuiltins contains the regex module functions for regular expression operations
var RegexBuiltins = map[string]*object.Builtin{
	// ============================================================================
	// Validation
	// ============================================================================

	// is_valid checks if a string is a valid regex pattern.
	// Takes pattern string. Returns bool.
	"regex.is_valid": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument (pattern)", errors.Ident("regex.is_valid()"))}
			}
			pattern, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s pattern", errors.Ident("regex.is_valid()"), errors.TypeExpected("string"))}
			}

			_, err := regexp.Compile(pattern.Value)
			if err != nil {
				return object.FALSE
			}
			return object.TRUE
		},
	},

	// ============================================================================
	// Matching
	// ============================================================================

	// match checks if a pattern matches anywhere in the string.
	// Takes pattern and string. Returns (bool, Error) tuple.
	"regex.match": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments (pattern, string)", errors.Ident("regex.match()"))}
			}
			pattern, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s pattern as first argument", errors.Ident("regex.match()"), errors.TypeExpected("string"))}
			}
			str, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("regex.match()"), errors.TypeExpected("string"))}
			}

			re, regexErr := compileRegex(pattern.Value)
			if regexErr != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					regexErr,
				}}
			}

			if re.MatchString(str.Value) {
				return &object.ReturnValue{Values: []object.Object{
					object.TRUE,
					object.NIL,
				}}
			}
			return &object.ReturnValue{Values: []object.Object{
				object.FALSE,
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// Finding
	// ============================================================================

	// find returns the first match of the pattern in the string.
	// Takes pattern and string. Returns (string|nil, Error) tuple.
	"regex.find": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments (pattern, string)", errors.Ident("regex.find()"))}
			}
			pattern, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s pattern as first argument", errors.Ident("regex.find()"), errors.TypeExpected("string"))}
			}
			str, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("regex.find()"), errors.TypeExpected("string"))}
			}

			re, regexErr := compileRegex(pattern.Value)
			if regexErr != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					regexErr,
				}}
			}

			match := re.FindString(str.Value)
			if match == "" {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					object.NIL,
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: match},
				object.NIL,
			}}
		},
	},

	// find_all returns all matches of the pattern in the string.
	// Takes pattern and string. Returns ([string], Error) tuple.
	"regex.find_all": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments (pattern, string)", errors.Ident("regex.find_all()"))}
			}
			pattern, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s pattern as first argument", errors.Ident("regex.find_all()"), errors.TypeExpected("string"))}
			}
			str, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("regex.find_all()"), errors.TypeExpected("string"))}
			}

			re, regexErr := compileRegex(pattern.Value)
			if regexErr != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					regexErr,
				}}
			}

			matches := re.FindAllString(str.Value, -1)
			return &object.ReturnValue{Values: []object.Object{
				stringsToArray(matches),
				object.NIL,
			}}
		},
	},

	// find_all_n returns the first n matches of the pattern in the string.
	// Takes pattern, string, and n. Returns ([string], Error) tuple.
	"regex.find_all_n": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 3 arguments (pattern, string, n)", errors.Ident("regex.find_all_n()"))}
			}
			pattern, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s pattern as first argument", errors.Ident("regex.find_all_n()"), errors.TypeExpected("string"))}
			}
			str, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("regex.find_all_n()"), errors.TypeExpected("string"))}
			}
			n, ok := args[2].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires an %s as third argument", errors.Ident("regex.find_all_n()"), errors.TypeExpected("integer"))}
			}

			re, regexErr := compileRegex(pattern.Value)
			if regexErr != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					regexErr,
				}}
			}

			matches := re.FindAllString(str.Value, int(n.Value.Int64()))
			return &object.ReturnValue{Values: []object.Object{
				stringsToArray(matches),
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// Replacing
	// ============================================================================

	// replace replaces the first match of the pattern with the replacement string.
	// Takes pattern, string, and replacement. Returns (string, Error) tuple.
	"regex.replace": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 3 arguments (pattern, string, replacement)", errors.Ident("regex.replace()"))}
			}
			pattern, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s pattern as first argument", errors.Ident("regex.replace()"), errors.TypeExpected("string"))}
			}
			str, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("regex.replace()"), errors.TypeExpected("string"))}
			}
			repl, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s replacement as third argument", errors.Ident("regex.replace()"), errors.TypeExpected("string"))}
			}

			re, regexErr := compileRegex(pattern.Value)
			if regexErr != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					regexErr,
				}}
			}

			// Replace only the first match
			result := re.ReplaceAllStringFunc(str.Value, func(match string) string {
				return repl.Value
			})
			// ReplaceAllStringFunc replaces all, so we need a different approach
			// Find the first match location and replace manually
			loc := re.FindStringIndex(str.Value)
			if loc == nil {
				// No match, return original string
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: str.Value},
					object.NIL,
				}}
			}

			result = str.Value[:loc[0]] + repl.Value + str.Value[loc[1]:]
			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: result},
				object.NIL,
			}}
		},
	},

	// replace_all replaces all matches of the pattern with the replacement string.
	// Takes pattern, string, and replacement. Returns (string, Error) tuple.
	"regex.replace_all": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 3 arguments (pattern, string, replacement)", errors.Ident("regex.replace_all()"))}
			}
			pattern, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s pattern as first argument", errors.Ident("regex.replace_all()"), errors.TypeExpected("string"))}
			}
			str, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("regex.replace_all()"), errors.TypeExpected("string"))}
			}
			repl, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s replacement as third argument", errors.Ident("regex.replace_all()"), errors.TypeExpected("string"))}
			}

			re, regexErr := compileRegex(pattern.Value)
			if regexErr != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					regexErr,
				}}
			}

			result := re.ReplaceAllString(str.Value, repl.Value)
			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: result},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// Splitting
	// ============================================================================

	// split splits the string by the pattern.
	// Takes pattern and string. Returns ([string], Error) tuple.
	"regex.split": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments (pattern, string)", errors.Ident("regex.split()"))}
			}
			pattern, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s pattern as first argument", errors.Ident("regex.split()"), errors.TypeExpected("string"))}
			}
			str, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("regex.split()"), errors.TypeExpected("string"))}
			}

			re, regexErr := compileRegex(pattern.Value)
			if regexErr != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					regexErr,
				}}
			}

			parts := re.Split(str.Value, -1)
			return &object.ReturnValue{Values: []object.Object{
				stringsToArray(parts),
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// Capture Groups
	// ============================================================================

	// groups returns the capture groups from the first match.
	// Takes pattern and string. Returns ([string], Error) tuple.
	// The first element is the full match, followed by each capture group.
	"regex.groups": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments (pattern, string)", errors.Ident("regex.groups()"))}
			}
			pattern, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s pattern as first argument", errors.Ident("regex.groups()"), errors.TypeExpected("string"))}
			}
			str, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("regex.groups()"), errors.TypeExpected("string"))}
			}

			re, regexErr := compileRegex(pattern.Value)
			if regexErr != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					regexErr,
				}}
			}

			matches := re.FindStringSubmatch(str.Value)
			if matches == nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.Array{Elements: []object.Object{}, ElementType: "string"},
					object.NIL,
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				stringsToArray(matches),
				object.NIL,
			}}
		},
	},

	// groups_all returns the capture groups from all matches.
	// Takes pattern and string. Returns ([[string]], Error) tuple.
	"regex.groups_all": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments (pattern, string)", errors.Ident("regex.groups_all()"))}
			}
			pattern, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s pattern as first argument", errors.Ident("regex.groups_all()"), errors.TypeExpected("string"))}
			}
			str, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("regex.groups_all()"), errors.TypeExpected("string"))}
			}

			re, regexErr := compileRegex(pattern.Value)
			if regexErr != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					regexErr,
				}}
			}

			allMatches := re.FindAllStringSubmatch(str.Value, -1)
			if allMatches == nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.Array{Elements: []object.Object{}, ElementType: "[string]"},
					object.NIL,
				}}
			}

			elements := make([]object.Object, len(allMatches))
			for i, matches := range allMatches {
				elements[i] = stringsToArray(matches)
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Array{Elements: elements, ElementType: "[string]"},
				object.NIL,
			}}
		},
	},
}

// ============================================================================
// Helper Functions
// ============================================================================

// compileRegex compiles a regex pattern and returns an error struct if invalid
func compileRegex(pattern string) (*regexp.Regexp, *object.Struct) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, CreateStdlibError("E15001", "invalid regex pattern: "+err.Error())
	}
	return re, nil
}

// stringsToArray converts a Go []string to an EZ [string] array
func stringsToArray(strs []string) *object.Array {
	if strs == nil {
		return &object.Array{Elements: []object.Object{}, ElementType: "string"}
	}
	elements := make([]object.Object, len(strs))
	for i, s := range strs {
		elements[i] = &object.String{Value: s}
	}
	return &object.Array{Elements: elements, ElementType: "string"}
}
