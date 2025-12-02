package errors

import "testing"

// TestLevenshteinDistance verifies edit distance calculation
func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		// Identical strings
		{"", "", 0},
		{"a", "a", 0},
		{"hello", "hello", 0},

		// Empty string cases
		{"", "abc", 3},
		{"abc", "", 3},

		// Single character differences
		{"a", "b", 1},
		{"cat", "bat", 1},
		{"cat", "car", 1},
		{"cat", "cats", 1},

		// Multiple changes
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"hello", "world", 4},

		// Typos
		{"tempp", "temp", 1},
		{"cosnt", "const", 2},
		{"fucntion", "function", 2},
		{"pritn", "print", 2},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_"+tt.s2, func(t *testing.T) {
			got := LevenshteinDistance(tt.s1, tt.s2)
			if got != tt.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tt.s1, tt.s2, got, tt.expected)
			}

			// Verify symmetry
			reverse := LevenshteinDistance(tt.s2, tt.s1)
			if reverse != tt.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d (symmetry)", tt.s2, tt.s1, reverse, tt.expected)
			}
		})
	}
}

// TestSuggestKeyword tests keyword suggestions
func TestSuggestKeyword(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Close typos should suggest
		{"tempp", "temp"},
		{"cosnt", "const"},
		{"retrun", "return"},
		{"iff", "if"},
		{"forrr", "for"},
		{"breake", "break"},
		{"contine", "continue"},
		{"imoprt", "import"},
		{"strcut", "struct"},
		{"enuem", "enum"},
		{"treu", "true"},
		{"flase", "false"},

		// Too far should return empty
		{"xyz", ""},
		{"abcdefg", ""},
		{"function", ""}, // Not a keyword

		// Exact matches shouldn't need suggestion (but will match)
		{"temp", "temp"},
		{"const", "const"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SuggestKeyword(tt.input)
			if got != tt.expected {
				t.Errorf("SuggestKeyword(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestSuggestBuiltin tests builtin function suggestions
func TestSuggestBuiltin(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Close typos
		{"lne", "len"},
		{"typof", "typeof"},
		{"inpt", "input"},
		{"pritnln", "println"},
		{"prnt", "print"},

		// Too far
		{"xyz", ""},
		{"abcdef", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SuggestBuiltin(tt.input)
			if got != tt.expected {
				t.Errorf("SuggestBuiltin(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestSuggestFromList tests custom list suggestions
func TestSuggestFromList(t *testing.T) {
	candidates := []string{"apple", "banana", "cherry", "date"}

	tests := []struct {
		input    string
		expected string
	}{
		{"aple", "apple"},
		{"banan", "banana"},
		{"chery", "cherry"},
		{"dat", "date"},
		{"xyz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SuggestFromList(tt.input, candidates)
			if got != tt.expected {
				t.Errorf("SuggestFromList(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestMinFunction tests the min helper
func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b, c  int
		expected int
	}{
		{1, 2, 3, 1},
		{3, 2, 1, 1},
		{2, 1, 3, 1},
		{5, 5, 5, 5},
		{0, 1, 2, 0},
		{-1, 0, 1, -1},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b, tt.c)
		if got != tt.expected {
			t.Errorf("min(%d, %d, %d) = %d, want %d", tt.a, tt.b, tt.c, got, tt.expected)
		}
	}
}

// TestKeywordsList verifies Keywords slice
func TestKeywordsList(t *testing.T) {
	if len(Keywords) == 0 {
		t.Error("Keywords should not be empty")
	}

	// Check some expected keywords exist
	expected := []string{"temp", "const", "do", "return", "if", "for", "struct", "enum"}
	for _, kw := range expected {
		found := false
		for _, k := range Keywords {
			if k == kw {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Keywords should contain %q", kw)
		}
	}
}

// TestBuiltinsList verifies Builtins slice
func TestBuiltinsList(t *testing.T) {
	if len(Builtins) == 0 {
		t.Error("Builtins should not be empty")
	}

	// Check some expected builtins exist
	expected := []string{"len", "typeof", "println", "print"}
	for _, b := range expected {
		found := false
		for _, builtin := range Builtins {
			if builtin == b {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Builtins should contain %q", b)
		}
	}
}
