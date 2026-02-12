package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"math/big"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// Helper to create a string object
func makeString(s string) *object.String {
	return &object.String{Value: s}
}

// Helper to create an integer object
func makeInt(i int64) *object.Integer {
	return &object.Integer{Value: big.NewInt(i)}
}

// ============================================================================
// Character Type Checks
// ============================================================================

func TestStringsIsAlphanumeric(t *testing.T) {
	fn := StringsBuiltins["strings.is_alphanumeric"].Fn

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"letters only", "Hello", true},
		{"digits only", "12345", true},
		{"alphanumeric", "Hello123", true},
		{"with space", "Hello 123", false},
		{"with punctuation", "Hello!", false},
		{"empty string", "", false},
		{"unicode letters", "Привет", true},
		{"unicode with digit", "世界123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.input))
			boolResult, ok := result.(*object.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolResult.Value != tt.expected {
				t.Errorf("is_alphanumeric(%q) = %v, want %v", tt.input, boolResult.Value, tt.expected)
			}
		})
	}

	// Test wrong argument count
	t.Run("wrong argument count", func(t *testing.T) {
		result := fn()
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong argument count")
		}
	})

	// Test wrong type
	t.Run("wrong type", func(t *testing.T) {
		result := fn(makeInt(42))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong type")
		}
	})
}

func TestStringsIsWhitespace(t *testing.T) {
	fn := StringsBuiltins["strings.is_whitespace"].Fn

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"spaces only", "   ", true},
		{"tabs only", "\t\t", true},
		{"mixed whitespace", " \t\n\r ", true},
		{"empty string", "", false},
		{"with letters", " a ", false},
		{"single space", " ", true},
		{"newlines", "\n\n\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.input))
			boolResult, ok := result.(*object.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolResult.Value != tt.expected {
				t.Errorf("is_whitespace(%q) = %v, want %v", tt.input, boolResult.Value, tt.expected)
			}
		})
	}
}

func TestStringsIsLowercase(t *testing.T) {
	fn := StringsBuiltins["strings.is_lowercase"].Fn

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"all lowercase", "hello", true},
		{"with uppercase", "Hello", false},
		{"all uppercase", "HELLO", false},
		{"empty string", "", false},
		{"digits only", "12345", false},
		{"lowercase with digits", "hello123", true},
		{"lowercase with space", "hello world", true},
		{"unicode lowercase", "привет", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.input))
			boolResult, ok := result.(*object.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolResult.Value != tt.expected {
				t.Errorf("is_lowercase(%q) = %v, want %v", tt.input, boolResult.Value, tt.expected)
			}
		})
	}
}

func TestStringsIsUppercase(t *testing.T) {
	fn := StringsBuiltins["strings.is_uppercase"].Fn

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"all uppercase", "HELLO", true},
		{"with lowercase", "Hello", false},
		{"all lowercase", "hello", false},
		{"empty string", "", false},
		{"digits only", "12345", false},
		{"uppercase with digits", "HELLO123", true},
		{"uppercase with space", "HELLO WORLD", true},
		{"unicode uppercase", "ПРИВЕТ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.input))
			boolResult, ok := result.(*object.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolResult.Value != tt.expected {
				t.Errorf("is_uppercase(%q) = %v, want %v", tt.input, boolResult.Value, tt.expected)
			}
		})
	}
}

func TestStringsIsASCII(t *testing.T) {
	fn := StringsBuiltins["strings.is_ascii"].Fn

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"ascii letters", "Hello", true},
		{"ascii with digits", "Hello123", true},
		{"ascii with punctuation", "Hello, World!", true},
		{"empty string", "", true},
		{"unicode letters", "Привет", false},
		{"chinese characters", "世界", false},
		{"mixed ascii and unicode", "Hello世界", false},
		{"control characters", "\x00\x1f", true},
		{"max ascii", "\x7f", true},
		{"above ascii", "\x80", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.input))
			boolResult, ok := result.(*object.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolResult.Value != tt.expected {
				t.Errorf("is_ascii(%q) = %v, want %v", tt.input, boolResult.Value, tt.expected)
			}
		})
	}
}

// ============================================================================
// Splitting Utilities
// ============================================================================

func TestStringsLines(t *testing.T) {
	fn := StringsBuiltins["strings.lines"].Fn

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"unix newlines", "a\nb\nc", []string{"a", "b", "c"}},
		{"windows newlines", "a\r\nb\r\nc", []string{"a", "b", "c"}},
		{"mixed newlines", "a\nb\r\nc", []string{"a", "b", "c"}},
		{"single line", "hello", []string{"hello"}},
		{"empty string", "", []string{""}},
		{"trailing newline", "a\nb\n", []string{"a", "b", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.input))
			arr, ok := result.(*object.Array)
			if !ok {
				t.Fatalf("expected Array, got %T", result)
			}
			if len(arr.Elements) != len(tt.expected) {
				t.Fatalf("lines(%q) returned %d elements, want %d", tt.input, len(arr.Elements), len(tt.expected))
			}
			for i, elem := range arr.Elements {
				str, ok := elem.(*object.String)
				if !ok {
					t.Fatalf("element %d: expected String, got %T", i, elem)
				}
				if str.Value != tt.expected[i] {
					t.Errorf("lines(%q)[%d] = %q, want %q", tt.input, i, str.Value, tt.expected[i])
				}
			}
		})
	}
}

func TestStringsWords(t *testing.T) {
	fn := StringsBuiltins["strings.words"].Fn

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"simple", "hello world", []string{"hello", "world"}},
		{"multiple spaces", "hello   world", []string{"hello", "world"}},
		{"leading/trailing spaces", "  hello world  ", []string{"hello", "world"}},
		{"tabs and newlines", "hello\t\nworld", []string{"hello", "world"}},
		{"single word", "hello", []string{"hello"}},
		{"empty string", "", []string{}},
		{"whitespace only", "   ", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.input))
			arr, ok := result.(*object.Array)
			if !ok {
				t.Fatalf("expected Array, got %T", result)
			}
			if len(arr.Elements) != len(tt.expected) {
				t.Fatalf("words(%q) returned %d elements, want %d", tt.input, len(arr.Elements), len(tt.expected))
			}
			for i, elem := range arr.Elements {
				str, ok := elem.(*object.String)
				if !ok {
					t.Fatalf("element %d: expected String, got %T", i, elem)
				}
				if str.Value != tt.expected[i] {
					t.Errorf("words(%q)[%d] = %q, want %q", tt.input, i, str.Value, tt.expected[i])
				}
			}
		})
	}
}

// ============================================================================
// Manipulation Functions
// ============================================================================

func TestStringsCharAt(t *testing.T) {
	fn := StringsBuiltins["strings.char_at"].Fn

	tests := []struct {
		name     string
		str      string
		index    int64
		expected string
		isError  bool
	}{
		{"first char", "hello", 0, "h", false},
		{"last char positive", "hello", 4, "o", false},
		{"middle char", "hello", 2, "l", false},
		{"negative index -1", "hello", -1, "o", false},
		{"negative index -2", "hello", -2, "l", false},
		{"unicode char", "世界", 0, "世", false},
		{"unicode negative", "世界", -1, "界", false},
		{"out of bounds positive", "hello", 5, "", true},
		{"out of bounds negative", "hello", -6, "", true},
		{"empty string", "", 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.str), makeInt(tt.index))
			if tt.isError {
				if _, ok := result.(*object.Error); !ok {
					t.Errorf("char_at(%q, %d) expected error, got %v", tt.str, tt.index, result)
				}
			} else {
				str, ok := result.(*object.String)
				if !ok {
					t.Fatalf("expected String, got %T", result)
				}
				if str.Value != tt.expected {
					t.Errorf("char_at(%q, %d) = %q, want %q", tt.str, tt.index, str.Value, tt.expected)
				}
			}
		})
	}

	// Test wrong argument count
	t.Run("wrong argument count", func(t *testing.T) {
		result := fn(makeString("hello"))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong argument count")
		}
	})
}

func TestStringsInsert(t *testing.T) {
	fn := StringsBuiltins["strings.insert"].Fn

	tests := []struct {
		name     string
		str      string
		pos      int64
		substr   string
		expected string
	}{
		{"insert at beginning", "world", 0, "hello ", "hello world"},
		{"insert at end", "hello", 5, " world", "hello world"},
		{"insert in middle", "hllo", 1, "e", "hello"},
		{"insert empty string", "hello", 2, "", "hello"},
		{"negative position clamps to 0", "world", -5, "hello ", "hello world"},
		{"position beyond length clamps to end", "hello", 100, " world", "hello world"},
		{"unicode string", "世", 1, "界", "世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.str), makeInt(tt.pos), makeString(tt.substr))
			str, ok := result.(*object.String)
			if !ok {
				t.Fatalf("expected String, got %T", result)
			}
			if str.Value != tt.expected {
				t.Errorf("insert(%q, %d, %q) = %q, want %q", tt.str, tt.pos, tt.substr, str.Value, tt.expected)
			}
		})
	}
}

func TestStringsCenter(t *testing.T) {
	fn := StringsBuiltins["strings.center"].Fn

	tests := []struct {
		name     string
		str      string
		width    int64
		pad      string
		expected string
	}{
		{"center with spaces", "hi", 6, "", "  hi  "},
		{"center with asterisks", "hi", 6, "*", "**hi**"},
		{"odd padding", "hi", 7, "*", "**hi***"},
		{"already wider", "hello", 3, "*", "hello"},
		{"exact width", "hi", 2, "*", "hi"},
		{"unicode pad char", "hi", 6, "★", "★★hi★★"},
		{"empty string", "", 4, "-", "----"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result object.Object
			if tt.pad == "" {
				result = fn(makeString(tt.str), makeInt(tt.width))
			} else {
				result = fn(makeString(tt.str), makeInt(tt.width), makeString(tt.pad))
			}
			str, ok := result.(*object.String)
			if !ok {
				t.Fatalf("expected String, got %T", result)
			}
			if str.Value != tt.expected {
				t.Errorf("center(%q, %d, %q) = %q, want %q", tt.str, tt.width, tt.pad, str.Value, tt.expected)
			}
		})
	}
}

func TestStringsRemove(t *testing.T) {
	fn := StringsBuiltins["strings.remove"].Fn

	tests := []struct {
		name     string
		str      string
		substr   string
		expected string
	}{
		{"remove first occurrence", "hello hello", "hello", " hello"},
		{"remove in middle", "abcabc", "bc", "aabc"},
		{"substring not found", "hello", "xyz", "hello"},
		{"remove entire string", "hello", "hello", ""},
		{"empty substring", "hello", "", "hello"},
		{"unicode removal", "世界世界", "世", "界世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.str), makeString(tt.substr))
			str, ok := result.(*object.String)
			if !ok {
				t.Fatalf("expected String, got %T", result)
			}
			if str.Value != tt.expected {
				t.Errorf("remove(%q, %q) = %q, want %q", tt.str, tt.substr, str.Value, tt.expected)
			}
		})
	}
}

func TestStringsRemoveAll(t *testing.T) {
	fn := StringsBuiltins["strings.remove_all"].Fn

	tests := []struct {
		name     string
		str      string
		substr   string
		expected string
	}{
		{"remove all occurrences", "hello hello", "hello", " "},
		{"remove all in middle", "abcabc", "bc", "aa"},
		{"substring not found", "hello", "xyz", "hello"},
		{"remove entire string", "aaa", "a", ""},
		{"empty substring", "hello", "", "hello"},
		{"unicode removal", "世界世界", "世", "界界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.str), makeString(tt.substr))
			str, ok := result.(*object.String)
			if !ok {
				t.Fatalf("expected String, got %T", result)
			}
			if str.Value != tt.expected {
				t.Errorf("remove_all(%q, %q) = %q, want %q", tt.str, tt.substr, str.Value, tt.expected)
			}
		})
	}
}

// ============================================================================
// Case Manipulation
// ============================================================================

func TestStringsTitle(t *testing.T) {
	fn := StringsBuiltins["strings.title"].Fn

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "hello world", "Hello World"},
		{"already title case", "Hello World", "Hello World"},
		{"all lowercase", "foo bar baz", "Foo Bar Baz"},
		{"all uppercase", "FOO BAR", "FOO BAR"},
		{"single word", "hello", "Hello"},
		{"empty string", "", ""},
		{"multiple spaces", "hello  world", "Hello  World"},
		{"with tabs", "hello\tworld", "Hello\tWorld"},
		{"with newlines", "hello\nworld", "Hello\nWorld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.input))
			str, ok := result.(*object.String)
			if !ok {
				t.Fatalf("expected String, got %T", result)
			}
			if str.Value != tt.expected {
				t.Errorf("title(%q) = %q, want %q", tt.input, str.Value, tt.expected)
			}
		})
	}

	// Test wrong argument count
	t.Run("wrong argument count", func(t *testing.T) {
		result := fn()
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong argument count")
		}
	})

	// Test wrong type
	t.Run("wrong type", func(t *testing.T) {
		result := fn(makeInt(42))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong type")
		}
	})
}

// ============================================================================
// Index / Search Functions
// ============================================================================

func TestStringsLastIndex(t *testing.T) {
	fn := StringsBuiltins["strings.last_index"].Fn

	tests := []struct {
		name     string
		str      string
		substr   string
		expected int64
	}{
		{"last occurrence", "hello hello", "hello", 6},
		{"single occurrence", "hello world", "world", 6},
		{"not found", "hello world", "xyz", -1},
		{"empty substr", "hello", "", 5},
		{"empty string", "", "hello", -1},
		{"repeated char", "aabaa", "a", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.str), makeString(tt.substr))
			intResult, ok := result.(*object.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %T", result)
			}
			if intResult.Value.Int64() != tt.expected {
				t.Errorf("last_index(%q, %q) = %d, want %d", tt.str, tt.substr, intResult.Value.Int64(), tt.expected)
			}
		})
	}

	// Test wrong argument count
	t.Run("wrong argument count", func(t *testing.T) {
		result := fn(makeString("hello"))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong argument count")
		}
	})

	// Test wrong types
	t.Run("wrong type first arg", func(t *testing.T) {
		result := fn(makeInt(42), makeString("hello"))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong type")
		}
	})

	t.Run("wrong type second arg", func(t *testing.T) {
		result := fn(makeString("hello"), makeInt(42))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong type")
		}
	})
}

// ============================================================================
// Count
// ============================================================================

func TestStringsCount(t *testing.T) {
	fn := StringsBuiltins["strings.count"].Fn

	tests := []struct {
		name     string
		str      string
		substr   string
		expected int64
	}{
		{"multiple occurrences", "hello hello hello", "hello", 3},
		{"single occurrence", "hello world", "world", 1},
		{"no occurrences", "hello world", "xyz", 0},
		{"overlapping not counted", "aaa", "aa", 1},
		{"empty substr", "hello", "", 6}, // strings.Count returns len+1 for empty sep
		{"single char", "banana", "a", 3},
		{"unicode", "世界世界世", "世", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeString(tt.str), makeString(tt.substr))
			intResult, ok := result.(*object.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %T", result)
			}
			if intResult.Value.Int64() != tt.expected {
				t.Errorf("count(%q, %q) = %d, want %d", tt.str, tt.substr, intResult.Value.Int64(), tt.expected)
			}
		})
	}

	// Test wrong argument count
	t.Run("wrong argument count", func(t *testing.T) {
		result := fn(makeString("hello"))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong argument count")
		}
	})

	// Test wrong types
	t.Run("wrong type first arg", func(t *testing.T) {
		result := fn(makeInt(42), makeString("hello"))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong type")
		}
	})

	t.Run("wrong type second arg", func(t *testing.T) {
		result := fn(makeString("hello"), makeInt(42))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong type")
		}
	})
}
