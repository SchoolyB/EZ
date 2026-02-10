package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"math/big"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// Helper to create a string object
func makeStr(s string) *object.String {
	return &object.String{Value: s}
}

// Helper to create an integer object
func makeInteger(i int64) *object.Integer {
	return &object.Integer{Value: big.NewInt(i)}
}

// regexGetReturnValues extracts return values from a ReturnValue (regex-specific to avoid conflicts)
func regexGetReturnValues(t *testing.T, obj object.Object) []object.Object {
	rv, ok := obj.(*object.ReturnValue)
	if !ok {
		t.Helper()
		t.Fatalf("expected ReturnValue, got %T", obj)
	}
	return rv.Values
}

// ============================================================================
// Validation Tests
// ============================================================================

func TestRegexIsValid(t *testing.T) {
	fn := RegexBuiltins["regex.is_valid"].Fn

	tests := []struct {
		name     string
		pattern  string
		expected bool
	}{
		{"simple pattern", "hello", true},
		{"character class", "[a-z]+", true},
		{"anchors", "^start.*end$", true},
		{"groups", "(foo|bar)", true},
		{"quantifiers", "a{2,5}", true},
		{"invalid bracket", "[", false},
		{"invalid group", "(", false},
		{"invalid quantifier", "*abc", false},
		{"empty pattern", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeStr(tt.pattern))
			boolResult, ok := result.(*object.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolResult.Value != tt.expected {
				t.Errorf("is_valid(%q) = %v, want %v", tt.pattern, boolResult.Value, tt.expected)
			}
		})
	}

	t.Run("wrong argument count", func(t *testing.T) {
		result := fn()
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong argument count")
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		result := fn(makeInteger(42))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong type")
		}
	})
}

// ============================================================================
// Matching Tests
// ============================================================================

func TestRegexMatch(t *testing.T) {
	fn := RegexBuiltins["regex.match"].Fn

	tests := []struct {
		name     string
		pattern  string
		input    string
		expected bool
	}{
		{"simple match", "hello", "hello world", true},
		{"no match", "foo", "hello world", false},
		{"regex match", "[0-9]+", "abc123def", true},
		{"anchor match", "^hello", "hello world", true},
		{"anchor no match", "^world", "hello world", false},
		{"empty pattern", "", "anything", true},
		{"empty input", ".", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeStr(tt.pattern), makeStr(tt.input))
			values := regexGetReturnValues(t, result)
			boolResult, ok := values[0].(*object.Boolean)
			if !ok {
				t.Fatalf("expected Boolean as first return value, got %T", values[0])
			}
			if boolResult.Value != tt.expected {
				t.Errorf("match(%q, %q) = %v, want %v", tt.pattern, tt.input, boolResult.Value, tt.expected)
			}
		})
	}

	t.Run("invalid pattern returns error", func(t *testing.T) {
		result := fn(makeStr("["), makeStr("test"))
		values := regexGetReturnValues(t, result)
		// First value should be false, second should be error struct
		if values[0] != object.FALSE {
			t.Error("expected false for invalid pattern")
		}
		if values[1] == object.NIL {
			t.Error("expected error struct for invalid pattern")
		}
	})
}

// ============================================================================
// Finding Tests
// ============================================================================

func TestRegexFind(t *testing.T) {
	fn := RegexBuiltins["regex.find"].Fn

	tests := []struct {
		name     string
		pattern  string
		input    string
		expected string
		isNil    bool
	}{
		{"simple find", "world", "hello world", "world", false},
		{"regex find", "[0-9]+", "abc123def456", "123", false},
		{"no match", "xyz", "hello world", "", true},
		{"first match only", "o", "hello world", "o", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeStr(tt.pattern), makeStr(tt.input))
			values := regexGetReturnValues(t, result)
			if tt.isNil {
				if values[0] != object.NIL {
					t.Errorf("find(%q, %q) expected nil, got %v", tt.pattern, tt.input, values[0])
				}
			} else {
				strResult, ok := values[0].(*object.String)
				if !ok {
					t.Fatalf("expected String, got %T", values[0])
				}
				if strResult.Value != tt.expected {
					t.Errorf("find(%q, %q) = %q, want %q", tt.pattern, tt.input, strResult.Value, tt.expected)
				}
			}
		})
	}
}

func TestRegexFindAll(t *testing.T) {
	fn := RegexBuiltins["regex.find_all"].Fn

	tests := []struct {
		name     string
		pattern  string
		input    string
		expected []string
	}{
		{"find all digits", "[0-9]+", "a1b22c333", []string{"1", "22", "333"}},
		{"find all words", "[a-z]+", "hello world foo", []string{"hello", "world", "foo"}},
		{"no matches", "xyz", "hello world", []string{}},
		{"overlapping", "o", "hello world", []string{"o", "o"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeStr(tt.pattern), makeStr(tt.input))
			values := regexGetReturnValues(t, result)
			arr, ok := values[0].(*object.Array)
			if !ok {
				t.Fatalf("expected Array, got %T", values[0])
			}
			if len(arr.Elements) != len(tt.expected) {
				t.Fatalf("find_all(%q, %q) returned %d elements, want %d", tt.pattern, tt.input, len(arr.Elements), len(tt.expected))
			}
			for i, elem := range arr.Elements {
				str, ok := elem.(*object.String)
				if !ok {
					t.Fatalf("element %d: expected String, got %T", i, elem)
				}
				if str.Value != tt.expected[i] {
					t.Errorf("find_all(%q, %q)[%d] = %q, want %q", tt.pattern, tt.input, i, str.Value, tt.expected[i])
				}
			}
		})
	}
}

func TestRegexFindAllN(t *testing.T) {
	fn := RegexBuiltins["regex.find_all_n"].Fn

	t.Run("limit results", func(t *testing.T) {
		result := fn(makeStr("[0-9]+"), makeStr("a1b2c3d4e5"), makeInteger(3))
		values := regexGetReturnValues(t, result)
		arr, ok := values[0].(*object.Array)
		if !ok {
			t.Fatalf("expected Array, got %T", values[0])
		}
		if len(arr.Elements) != 3 {
			t.Errorf("find_all_n with n=3 returned %d elements, want 3", len(arr.Elements))
		}
	})

	t.Run("n larger than matches", func(t *testing.T) {
		result := fn(makeStr("[0-9]+"), makeStr("a1b2"), makeInteger(10))
		values := regexGetReturnValues(t, result)
		arr := values[0].(*object.Array)
		if len(arr.Elements) != 2 {
			t.Errorf("expected 2 elements, got %d", len(arr.Elements))
		}
	})
}

// ============================================================================
// Replacing Tests
// ============================================================================

func TestRegexReplace(t *testing.T) {
	fn := RegexBuiltins["regex.replace"].Fn

	tests := []struct {
		name        string
		pattern     string
		input       string
		replacement string
		expected    string
	}{
		{"simple replace", "world", "hello world", "there", "hello there"},
		{"replace first only", "o", "hello world", "0", "hell0 world"},
		{"regex replace", "[0-9]+", "abc123def", "NUM", "abcNUMdef"},
		{"no match", "xyz", "hello", "abc", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeStr(tt.pattern), makeStr(tt.input), makeStr(tt.replacement))
			values := regexGetReturnValues(t, result)
			strResult, ok := values[0].(*object.String)
			if !ok {
				t.Fatalf("expected String, got %T", values[0])
			}
			if strResult.Value != tt.expected {
				t.Errorf("replace(%q, %q, %q) = %q, want %q", tt.pattern, tt.input, tt.replacement, strResult.Value, tt.expected)
			}
		})
	}
}

func TestRegexReplaceAll(t *testing.T) {
	fn := RegexBuiltins["regex.replace_all"].Fn

	tests := []struct {
		name        string
		pattern     string
		input       string
		replacement string
		expected    string
	}{
		{"replace all", "o", "hello world", "0", "hell0 w0rld"},
		{"replace all digits", "[0-9]", "a1b2c3", "X", "aXbXcX"},
		{"no match", "xyz", "hello", "abc", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeStr(tt.pattern), makeStr(tt.input), makeStr(tt.replacement))
			values := regexGetReturnValues(t, result)
			strResult, ok := values[0].(*object.String)
			if !ok {
				t.Fatalf("expected String, got %T", values[0])
			}
			if strResult.Value != tt.expected {
				t.Errorf("replace_all(%q, %q, %q) = %q, want %q", tt.pattern, tt.input, tt.replacement, strResult.Value, tt.expected)
			}
		})
	}
}

// ============================================================================
// Splitting Tests
// ============================================================================

func TestRegexSplit(t *testing.T) {
	fn := RegexBuiltins["regex.split"].Fn

	tests := []struct {
		name     string
		pattern  string
		input    string
		expected []string
	}{
		{"split by comma", ",", "a,b,c", []string{"a", "b", "c"}},
		{"split by whitespace", "\\s+", "hello   world  foo", []string{"hello", "world", "foo"}},
		{"split by digits", "[0-9]+", "a1b22c333d", []string{"a", "b", "c", "d"}},
		{"no match", "xyz", "hello", []string{"hello"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeStr(tt.pattern), makeStr(tt.input))
			values := regexGetReturnValues(t, result)
			arr, ok := values[0].(*object.Array)
			if !ok {
				t.Fatalf("expected Array, got %T", values[0])
			}
			if len(arr.Elements) != len(tt.expected) {
				t.Fatalf("split(%q, %q) returned %d elements, want %d", tt.pattern, tt.input, len(arr.Elements), len(tt.expected))
			}
			for i, elem := range arr.Elements {
				str, ok := elem.(*object.String)
				if !ok {
					t.Fatalf("element %d: expected String, got %T", i, elem)
				}
				if str.Value != tt.expected[i] {
					t.Errorf("split(%q, %q)[%d] = %q, want %q", tt.pattern, tt.input, i, str.Value, tt.expected[i])
				}
			}
		})
	}
}

// ============================================================================
// Capture Groups Tests
// ============================================================================

func TestRegexGroups(t *testing.T) {
	fn := RegexBuiltins["regex.groups"].Fn

	t.Run("single group", func(t *testing.T) {
		result := fn(makeStr("(\\d+)"), makeStr("abc123def"))
		values := regexGetReturnValues(t, result)
		arr := values[0].(*object.Array)
		if len(arr.Elements) != 2 {
			t.Fatalf("expected 2 elements (full match + 1 group), got %d", len(arr.Elements))
		}
		// First element is full match, second is group
		if arr.Elements[0].(*object.String).Value != "123" {
			t.Error("full match should be '123'")
		}
		if arr.Elements[1].(*object.String).Value != "123" {
			t.Error("group 1 should be '123'")
		}
	})

	t.Run("multiple groups", func(t *testing.T) {
		result := fn(makeStr("(\\w+)@(\\w+)\\.(\\w+)"), makeStr("test@example.com"))
		values := regexGetReturnValues(t, result)
		arr := values[0].(*object.Array)
		if len(arr.Elements) != 4 {
			t.Fatalf("expected 4 elements, got %d", len(arr.Elements))
		}
		expected := []string{"test@example.com", "test", "example", "com"}
		for i, exp := range expected {
			if arr.Elements[i].(*object.String).Value != exp {
				t.Errorf("groups()[%d] = %q, want %q", i, arr.Elements[i].(*object.String).Value, exp)
			}
		}
	})

	t.Run("no match", func(t *testing.T) {
		result := fn(makeStr("(xyz)"), makeStr("hello"))
		values := regexGetReturnValues(t, result)
		arr := values[0].(*object.Array)
		if len(arr.Elements) != 0 {
			t.Errorf("expected empty array for no match, got %d elements", len(arr.Elements))
		}
	})
}

func TestRegexGroupsAll(t *testing.T) {
	fn := RegexBuiltins["regex.groups_all"].Fn

	t.Run("multiple matches with groups", func(t *testing.T) {
		result := fn(makeStr("(\\d+)-(\\d+)"), makeStr("1-2 and 3-4"))
		values := regexGetReturnValues(t, result)
		arr := values[0].(*object.Array)
		if len(arr.Elements) != 2 {
			t.Fatalf("expected 2 match groups, got %d", len(arr.Elements))
		}

		// First match: "1-2"
		first := arr.Elements[0].(*object.Array)
		if first.Elements[0].(*object.String).Value != "1-2" {
			t.Error("first full match should be '1-2'")
		}
		if first.Elements[1].(*object.String).Value != "1" {
			t.Error("first group 1 should be '1'")
		}
		if first.Elements[2].(*object.String).Value != "2" {
			t.Error("first group 2 should be '2'")
		}

		// Second match: "3-4"
		second := arr.Elements[1].(*object.Array)
		if second.Elements[0].(*object.String).Value != "3-4" {
			t.Error("second full match should be '3-4'")
		}
	})

	t.Run("no matches", func(t *testing.T) {
		result := fn(makeStr("(xyz)"), makeStr("hello"))
		values := regexGetReturnValues(t, result)
		arr := values[0].(*object.Array)
		if len(arr.Elements) != 0 {
			t.Errorf("expected empty array, got %d elements", len(arr.Elements))
		}
	})
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestRegexInvalidPatternErrors(t *testing.T) {
	fns := map[string]func(...object.Object) object.Object{
		"match":       RegexBuiltins["regex.match"].Fn,
		"find":        RegexBuiltins["regex.find"].Fn,
		"find_all":    RegexBuiltins["regex.find_all"].Fn,
		"replace":     RegexBuiltins["regex.replace"].Fn,
		"replace_all": RegexBuiltins["regex.replace_all"].Fn,
		"split":       RegexBuiltins["regex.split"].Fn,
		"groups":      RegexBuiltins["regex.groups"].Fn,
		"groups_all":  RegexBuiltins["regex.groups_all"].Fn,
	}

	invalidPattern := makeStr("[")
	testStr := makeStr("test")
	replStr := makeStr("repl")

	for name, fn := range fns {
		t.Run(name+" with invalid pattern", func(t *testing.T) {
			var result object.Object
			switch name {
			case "replace", "replace_all":
				result = fn(invalidPattern, testStr, replStr)
			case "find_all_n":
				result = fn(invalidPattern, testStr, makeInteger(5))
			default:
				result = fn(invalidPattern, testStr)
			}

			values := regexGetReturnValues(t, result)
			// Second value should be error struct (not nil)
			if values[1] == object.NIL {
				t.Error("expected error struct for invalid pattern")
			}
		})
	}
}
