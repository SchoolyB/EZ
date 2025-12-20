package lexer

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"testing"

	"github.com/marshallburns/ez/pkg/tokenizer"
)

// Helper function to collect all tokens from input
func tokenize(input string) []tokenizer.Token {
	l := NewLexer(input)
	var tokens []tokenizer.Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == tokenizer.EOF {
			break
		}
	}
	return tokens
}

// Helper function to get lexer errors
func lexErrors(input string) []LexerError {
	l := NewLexer(input)
	for {
		tok := l.NextToken()
		if tok.Type == tokenizer.EOF {
			break
		}
	}
	return l.Errors()
}

// TestNewLexer verifies lexer initialization
func TestNewLexer(t *testing.T) {
	l := NewLexer("test")
	if l == nil {
		t.Fatal("NewLexer returned nil")
	}
	if l.line != 1 {
		t.Errorf("line = %d, want 1", l.line)
	}
	if len(l.errors) != 0 {
		t.Errorf("errors = %d, want 0", len(l.errors))
	}
}

// TestSingleCharTokens tests all single-character tokens
func TestSingleCharTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected tokenizer.TokenType
	}{
		{"=", tokenizer.ASSIGN},
		{"+", tokenizer.PLUS},
		{"-", tokenizer.MINUS},
		{"!", tokenizer.BANG},
		{"*", tokenizer.ASTERISK},
		{"/", tokenizer.SLASH},
		{"%", tokenizer.PERCENT},
		{"<", tokenizer.LT},
		{">", tokenizer.GT},
		{",", tokenizer.COMMA},
		{":", tokenizer.COLON},
		{";", tokenizer.SEMICOLON},
		{"(", tokenizer.LPAREN},
		{")", tokenizer.RPAREN},
		{"{", tokenizer.LBRACE},
		{"}", tokenizer.RBRACE},
		{"[", tokenizer.LBRACKET},
		{"]", tokenizer.RBRACKET},
		{".", tokenizer.DOT},
		{"@", tokenizer.AT},
		{"&", tokenizer.AMPERSAND},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if len(tokens) < 1 {
				t.Fatal("no tokens produced")
			}
			if tokens[0].Type != tt.expected {
				t.Errorf("got %s, want %s", tokens[0].Type, tt.expected)
			}
		})
	}
}

// TestMultiCharOperators tests two-character operators
func TestMultiCharOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected tokenizer.TokenType
		literal  string
	}{
		{"==", tokenizer.EQ, "=="},
		{"!=", tokenizer.NOT_EQ, "!="},
		{"<=", tokenizer.LT_EQ, "<="},
		{">=", tokenizer.GT_EQ, ">="},
		{"+=", tokenizer.PLUS_ASSIGN, "+="},
		{"-=", tokenizer.MINUS_ASSIGN, "-="},
		{"*=", tokenizer.ASTERISK_ASSIGN, "*="},
		{"/=", tokenizer.SLASH_ASSIGN, "/="},
		{"%=", tokenizer.PERCENT_ASSIGN, "%="},
		{"++", tokenizer.INCREMENT, "++"},
		{"--", tokenizer.DECREMENT, "--"},
		{"&&", tokenizer.AND, "&&"},
		{"||", tokenizer.OR, "||"},
		{"->", tokenizer.ARROW, "->"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if len(tokens) < 1 {
				t.Fatal("no tokens produced")
			}
			if tokens[0].Type != tt.expected {
				t.Errorf("type = %s, want %s", tokens[0].Type, tt.expected)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("literal = %q, want %q", tokens[0].Literal, tt.literal)
			}
		})
	}
}

// TestIntegerLiterals tests integer token parsing
func TestIntegerLiterals(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"0", "0"},
		{"42", "42"},
		{"123456789", "123456789"},
		{"1_000", "1_000"},
		{"1_000_000", "1_000_000"},
		{"100_200_300", "100_200_300"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if tokens[0].Type != tokenizer.INT {
				t.Errorf("type = %s, want INT", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("literal = %q, want %q", tokens[0].Literal, tt.literal)
			}
		})
	}
}

// TestFloatLiterals tests float token parsing
func TestFloatLiterals(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"0.0", "0.0"},
		{"3.14", "3.14"},
		{"3.14159", "3.14159"},
		{"0.5", "0.5"},
		{"100.001", "100.001"},
		{"1_000.5", "1_000.5"},
		{"1_000.123_456", "1_000.123_456"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if tokens[0].Type != tokenizer.FLOAT {
				t.Errorf("type = %s, want FLOAT", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("literal = %q, want %q", tokens[0].Literal, tt.literal)
			}
		})
	}
}

// TestScientificNotation tests scientific notation parsing (Bug #366 fix)
func TestScientificNotation(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"1e10", "1e10"},
		{"1E10", "1E10"},
		{"1e+10", "1e+10"},
		{"1e-10", "1e-10"},
		{"1.5e10", "1.5e10"},
		{"1.5e+10", "1.5e+10"},
		{"1.5e-10", "1.5e-10"},
		{"1.5E10", "1.5E10"},
		{"2.5e308", "2.5e308"},
		{"5e-3", "5e-3"},
		{"3.14e2", "3.14e2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if tokens[0].Type != tokenizer.FLOAT {
				t.Errorf("type = %s, want FLOAT", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("literal = %q, want %q", tokens[0].Literal, tt.literal)
			}
		})
	}
}

// TestStringLiterals tests string token parsing
func TestStringLiterals(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{`""`, ""},
		{`"hello"`, "hello"},
		{`"hello world"`, "hello world"},
		{`"hello\nworld"`, `hello\nworld`},
		{`"hello\tworld"`, `hello\tworld`},
		{`"hello\\world"`, `hello\\world`},
		{`"hello\"world"`, `hello\"world`},
		{`"line1\nline2"`, `line1\nline2`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if tokens[0].Type != tokenizer.STRING {
				t.Errorf("type = %s, want STRING", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("literal = %q, want %q", tokens[0].Literal, tt.literal)
			}
		})
	}
}

// TestStringInterpolation tests string interpolation parsing
func TestStringInterpolation(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{`"${x}"`, "${x}"},
		{`"hello ${name}"`, "hello ${name}"},
		{`"${a} and ${b}"`, "${a} and ${b}"},
		{`"nested ${obj.field}"`, "nested ${obj.field}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if tokens[0].Type != tokenizer.STRING {
				t.Errorf("type = %s, want STRING", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("literal = %q, want %q", tokens[0].Literal, tt.literal)
			}
		})
	}
}

// TestCharLiterals tests character token parsing
func TestCharLiterals(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"'a'", "a"},
		{"'Z'", "Z"},
		{"'0'", "0"},
		{"' '", " "},
		{`'\n'`, `\n`},
		{`'\t'`, `\t`},
		{`'\\'`, `\\`},
		{`'\''`, `\'`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if tokens[0].Type != tokenizer.CHAR {
				t.Errorf("type = %s, want CHAR", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("literal = %q, want %q", tokens[0].Literal, tt.literal)
			}
		})
	}
}

// TestKeywords tests keyword recognition
func TestKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected tokenizer.TokenType
	}{
		{"temp", tokenizer.TEMP},
		{"const", tokenizer.CONST},
		{"do", tokenizer.DO},
		{"return", tokenizer.RETURN},
		{"if", tokenizer.IF},
		{"or", tokenizer.OR_KW},
		{"otherwise", tokenizer.OTHERWISE},
		{"for", tokenizer.FOR},
		{"for_each", tokenizer.FOR_EACH},
		{"as_long_as", tokenizer.AS_LONG_AS},
		{"loop", tokenizer.LOOP},
		{"break", tokenizer.BREAK},
		{"continue", tokenizer.CONTINUE},
		{"in", tokenizer.IN},
		{"not_in", tokenizer.NOT_IN},
		{"range", tokenizer.RANGE},
		{"import", tokenizer.IMPORT},
		{"using", tokenizer.USING},
		{"struct", tokenizer.STRUCT},
		{"enum", tokenizer.ENUM},
		{"nil", tokenizer.NIL},
		{"new", tokenizer.NEW},
		{"true", tokenizer.TRUE},
		{"false", tokenizer.FALSE},
		{"module", tokenizer.MODULE},
		{"private", tokenizer.PRIVATE},
		{"use", tokenizer.USE},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if tokens[0].Type != tt.expected {
				t.Errorf("type = %s, want %s", tokens[0].Type, tt.expected)
			}
		})
	}
}

// TestIdentifiers tests identifier recognition
func TestIdentifiers(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"x", "x"},
		{"foo", "foo"},
		{"bar123", "bar123"},
		{"_private", "_private"},
		{"camelCase", "camelCase"},
		{"PascalCase", "PascalCase"},
		{"snake_case", "snake_case"},
		{"SCREAMING_CASE", "SCREAMING_CASE"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if tokens[0].Type != tokenizer.IDENT {
				t.Errorf("type = %s, want IDENT", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("literal = %q, want %q", tokens[0].Literal, tt.literal)
			}
		})
	}
}

// TestSpecialTokens tests #suppress, !in, and blank identifier
func TestSpecialTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected tokenizer.TokenType
		literal  string
	}{
		{"#suppress", tokenizer.SUPPRESS, "#suppress"},
		{"!in", tokenizer.NOT_IN, "!in"},
		{"_", tokenizer.BLANK, "_"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if tokens[0].Type != tt.expected {
				t.Errorf("type = %s, want %s", tokens[0].Type, tt.expected)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("literal = %q, want %q", tokens[0].Literal, tt.literal)
			}
		})
	}
}

// TestComments tests comment handling
func TestComments(t *testing.T) {
	t.Run("single line comment", func(t *testing.T) {
		tokens := tokenize("x // this is a comment\ny")
		// Should get: x, y, EOF
		if len(tokens) != 3 {
			t.Errorf("got %d tokens, want 3", len(tokens))
		}
		if tokens[0].Literal != "x" {
			t.Errorf("first token = %q, want x", tokens[0].Literal)
		}
		if tokens[1].Literal != "y" {
			t.Errorf("second token = %q, want y", tokens[1].Literal)
		}
	})

	t.Run("multi-line comment", func(t *testing.T) {
		tokens := tokenize("x /* comment */ y")
		if len(tokens) != 3 {
			t.Errorf("got %d tokens, want 3", len(tokens))
		}
		if tokens[0].Literal != "x" {
			t.Errorf("first token = %q, want x", tokens[0].Literal)
		}
		if tokens[1].Literal != "y" {
			t.Errorf("second token = %q, want y", tokens[1].Literal)
		}
	})

	t.Run("multi-line comment spanning lines", func(t *testing.T) {
		tokens := tokenize("x /* line1\nline2\nline3 */ y")
		if len(tokens) != 3 {
			t.Errorf("got %d tokens, want 3", len(tokens))
		}
	})
}

// TestWhitespace tests whitespace handling
func TestWhitespace(t *testing.T) {
	input := "  x  \t  y  \n  z  "
	tokens := tokenize(input)
	// Should get: x, y, z, EOF
	if len(tokens) != 4 {
		t.Errorf("got %d tokens, want 4", len(tokens))
	}
	if tokens[0].Literal != "x" {
		t.Errorf("token 0 = %q, want x", tokens[0].Literal)
	}
	if tokens[1].Literal != "y" {
		t.Errorf("token 1 = %q, want y", tokens[1].Literal)
	}
	if tokens[2].Literal != "z" {
		t.Errorf("token 2 = %q, want z", tokens[2].Literal)
	}
}

// TestLineColumnTracking tests line/column numbers
func TestLineColumnTracking(t *testing.T) {
	input := "x\ny\nz"
	tokens := tokenize(input)

	expected := []struct {
		literal string
		line    int
		column  int
	}{
		{"x", 1, 1},
		{"y", 2, 1},
		{"z", 3, 1},
	}

	for i, exp := range expected {
		if tokens[i].Literal != exp.literal {
			t.Errorf("token %d: literal = %q, want %q", i, tokens[i].Literal, exp.literal)
		}
		if tokens[i].Line != exp.line {
			t.Errorf("token %d (%s): line = %d, want %d", i, exp.literal, tokens[i].Line, exp.line)
		}
		if tokens[i].Column != exp.column {
			t.Errorf("token %d (%s): column = %d, want %d", i, exp.literal, tokens[i].Column, exp.column)
		}
	}
}

// TestCompleteProgram tests tokenizing a complete program
func TestCompleteProgram(t *testing.T) {
	input := `do main() {
    temp x int = 42
    if x > 0 {
        return true
    }
}`
	tokens := tokenize(input)

	expected := []tokenizer.TokenType{
		tokenizer.DO, tokenizer.IDENT, tokenizer.LPAREN, tokenizer.RPAREN, tokenizer.LBRACE,
		tokenizer.TEMP, tokenizer.IDENT, tokenizer.IDENT, tokenizer.ASSIGN, tokenizer.INT,
		tokenizer.IF, tokenizer.IDENT, tokenizer.GT, tokenizer.INT, tokenizer.LBRACE,
		tokenizer.RETURN, tokenizer.TRUE,
		tokenizer.RBRACE,
		tokenizer.RBRACE,
		tokenizer.EOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}

	for i, exp := range expected {
		if tokens[i].Type != exp {
			t.Errorf("token %d: type = %s, want %s", i, tokens[i].Type, exp)
		}
	}
}

// ============================================================================
// ERROR CASES
// ============================================================================

// TestE1001IllegalCharacter tests illegal character detection
func TestE1001IllegalCharacter(t *testing.T) {
	tests := []string{
		"$", // standalone $ is illegal
		"#",
		"~",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			tokens := tokenize(input)
			if tokens[0].Type != tokenizer.ILLEGAL {
				t.Errorf("expected ILLEGAL token for %q", input)
			}
		})
	}
}

// TestE1002IllegalOrCharacter tests single | detection
func TestE1002IllegalOrCharacter(t *testing.T) {
	errs := lexErrors("|")
	if len(errs) == 0 {
		t.Error("expected error for single |")
		return
	}
	if errs[0].Code != "E1002" {
		t.Errorf("code = %s, want E1002", errs[0].Code)
	}
}

// TestE1003UnclosedComment tests unclosed multi-line comment
func TestE1003UnclosedComment(t *testing.T) {
	errs := lexErrors("/* unclosed comment")
	if len(errs) == 0 {
		t.Error("expected error for unclosed comment")
		return
	}
	if errs[0].Code != "E1003" {
		t.Errorf("code = %s, want E1003", errs[0].Code)
	}
}

// TestE1004UnclosedString tests unclosed string detection
func TestE1004UnclosedString(t *testing.T) {
	tests := []string{
		`"unclosed`,
		`"unclosed string`,
		"\"string\nwith newline\"",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			errs := lexErrors(input)
			found := false
			for _, e := range errs {
				if e.Code == "E1004" {
					found = true
					break
				}
			}
			if !found {
				t.Error("expected E1004 for unclosed string")
			}
		})
	}
}

// TestE1005UnclosedChar tests unclosed character literal
func TestE1005UnclosedChar(t *testing.T) {
	tests := []string{
		"'a",
		"'",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			errs := lexErrors(input)
			found := false
			for _, e := range errs {
				if e.Code == "E1005" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected E1005 for %q", input)
			}
		})
	}
}

// TestE1006InvalidEscapeString tests invalid escape in string
func TestE1006InvalidEscapeString(t *testing.T) {
	errs := lexErrors(`"hello\x"`)
	found := false
	for _, e := range errs {
		if e.Code == "E1006" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1006 for invalid escape in string")
	}
}

// TestE1007InvalidEscapeChar tests invalid escape in char
func TestE1007InvalidEscapeChar(t *testing.T) {
	errs := lexErrors(`'\x'`)
	found := false
	for _, e := range errs {
		if e.Code == "E1007" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1007 for invalid escape in char")
	}
}

// TestE1008EmptyCharLiteral tests empty character literal
func TestE1008EmptyCharLiteral(t *testing.T) {
	errs := lexErrors("''")
	if len(errs) == 0 {
		t.Error("expected error for empty char")
		return
	}
	if errs[0].Code != "E1008" {
		t.Errorf("code = %s, want E1008", errs[0].Code)
	}
}

// TestE1009MultiCharLiteral tests multi-character literal
func TestE1009MultiCharLiteral(t *testing.T) {
	errs := lexErrors("'ab'")
	found := false
	for _, e := range errs {
		if e.Code == "E1009" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1009 for multi-char literal")
	}
}

// TestE1011ConsecutiveUnderscores tests consecutive underscores in number
func TestE1011ConsecutiveUnderscores(t *testing.T) {
	errs := lexErrors("1__000")
	found := false
	for _, e := range errs {
		if e.Code == "E1011" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1011 for consecutive underscores")
	}
}

// TestE1013TrailingUnderscore tests trailing underscore in number
func TestE1013TrailingUnderscore(t *testing.T) {
	errs := lexErrors("100_")
	found := false
	for _, e := range errs {
		if e.Code == "E1013" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1013 for trailing underscore")
	}
}

// TestE1014UnderscoreBeforeDecimal tests underscore before decimal point
func TestE1014UnderscoreBeforeDecimal(t *testing.T) {
	errs := lexErrors("100_.5")
	found := false
	for _, e := range errs {
		if e.Code == "E1014" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1014 for underscore before decimal")
	}
}

// TestE1015UnderscoreAfterDecimal tests underscore after decimal point
func TestE1015UnderscoreAfterDecimal(t *testing.T) {
	errs := lexErrors("100._5")
	found := false
	for _, e := range errs {
		if e.Code == "E1015" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1015 for underscore after decimal")
	}
}

// TestE1016TrailingDecimal tests trailing decimal point
func TestE1016TrailingDecimal(t *testing.T) {
	// "1." followed by non-digit should trigger E1016
	l := NewLexer("1. ")
	l.NextToken() // get the number
	errs := l.Errors()
	found := false
	for _, e := range errs {
		if e.Code == "E1016" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1016 for trailing decimal")
	}
}

// TestLexerErrors tests the Errors() method
func TestLexerErrors(t *testing.T) {
	l := NewLexer("''") // empty char - will produce error
	l.NextToken()
	errs := l.Errors()

	if len(errs) == 0 {
		t.Error("expected errors, got none")
	}

	// Verify error structure
	if errs[0].Line == 0 {
		t.Error("error should have line number")
	}
	if errs[0].Column == 0 {
		t.Error("error should have column number")
	}
	if errs[0].Code == "" {
		t.Error("error should have code")
	}
	if errs[0].Message == "" {
		t.Error("error should have message")
	}
}

// TestEOF tests end of file handling
func TestEOF(t *testing.T) {
	tokens := tokenize("")
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1", len(tokens))
	}
	if tokens[0].Type != tokenizer.EOF {
		t.Errorf("type = %s, want EOF", tokens[0].Type)
	}
}

// ============================================================================
// Raw String Tests (backtick strings)
// ============================================================================

// TestRawStringBasic tests basic raw string lexing
func TestRawStringBasic(t *testing.T) {
	tokens := tokenize("`hello world`")
	if len(tokens) != 2 { // RAW_STRING + EOF
		t.Fatalf("got %d tokens, want 2", len(tokens))
	}
	if tokens[0].Type != tokenizer.RAW_STRING {
		t.Errorf("type = %s, want RAW_STRING", tokens[0].Type)
	}
	if tokens[0].Literal != "hello world" {
		t.Errorf("literal = %q, want %q", tokens[0].Literal, "hello world")
	}
}

// TestRawStringNoEscapeProcessing tests that escapes are not processed
func TestRawStringNoEscapeProcessing(t *testing.T) {
	tokens := tokenize("`hello\\nworld`")
	if tokens[0].Literal != "hello\\nworld" {
		t.Errorf("literal = %q, want %q", tokens[0].Literal, "hello\\nworld")
	}
}

// TestRawStringWithQuotes tests that quotes are literal inside raw strings
func TestRawStringWithQuotes(t *testing.T) {
	input := "`{\"name\": \"Alice\"}`"
	tokens := tokenize(input)
	expected := "{\"name\": \"Alice\"}"
	if tokens[0].Literal != expected {
		t.Errorf("literal = %q, want %q", tokens[0].Literal, expected)
	}
}

// TestRawStringMultiline tests multiline raw strings
func TestRawStringMultiline(t *testing.T) {
	input := "`line1\nline2\nline3`"
	tokens := tokenize(input)
	if tokens[0].Type != tokenizer.RAW_STRING {
		t.Errorf("type = %s, want RAW_STRING", tokens[0].Type)
	}
	expected := "line1\nline2\nline3"
	if tokens[0].Literal != expected {
		t.Errorf("literal = %q, want %q", tokens[0].Literal, expected)
	}
}

// TestRawStringWithDollarBrace tests that ${} is not processed as interpolation
func TestRawStringWithDollarBrace(t *testing.T) {
	tokens := tokenize("`Hello ${name}!`")
	if tokens[0].Literal != "Hello ${name}!" {
		t.Errorf("literal = %q, want %q", tokens[0].Literal, "Hello ${name}!")
	}
}

// TestE1017UnclosedRawString tests unclosed raw string detection
func TestE1017UnclosedRawString(t *testing.T) {
	errs := lexErrors("`unclosed raw string")
	found := false
	for _, e := range errs {
		if e.Code == "E1017" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1017 for unclosed raw string")
	}
}

// ============================================================================
// Number Literal Coverage Tests
// ============================================================================

// TestHexLiterals tests hex number parsing
func TestHexLiterals(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"0x10", "0x10"},
		{"0X10", "0X10"},
		{"0xFF", "0xFF"},
		{"0xABCDEF", "0xABCDEF"},
		{"0x1234567890", "0x1234567890"},
		{"0xabcdef", "0xabcdef"},
		{"0x1_2_3", "0x1_2_3"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if len(tokens) < 1 {
				t.Fatal("expected at least 1 token")
			}
			if tokens[0].Type != tokenizer.INT {
				t.Errorf("type = %s, want INT", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("literal = %s, want %s", tokens[0].Literal, tt.literal)
			}
		})
	}
}

// TestBinaryLiterals tests binary number parsing
func TestBinaryLiterals(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"0b101", "0b101"},
		{"0B101", "0B101"},
		{"0b11111111", "0b11111111"},
		{"0b0", "0b0"},
		{"0b1", "0b1"},
		{"0b1010_1010", "0b1010_1010"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if len(tokens) < 1 {
				t.Fatal("expected at least 1 token")
			}
			if tokens[0].Type != tokenizer.INT {
				t.Errorf("type = %s, want INT", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("literal = %s, want %s", tokens[0].Literal, tt.literal)
			}
		})
	}
}

// TestE1010InvalidHexLiteral tests invalid hex literal detection
func TestE1010InvalidHexLiteral(t *testing.T) {
	errs := lexErrors("0x")
	found := false
	for _, e := range errs {
		if e.Code == "E1010" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1010 for invalid hex literal")
	}
}

// TestE1010InvalidBinaryLiteral tests invalid binary literal detection
func TestE1010InvalidBinaryLiteral(t *testing.T) {
	errs := lexErrors("0b")
	found := false
	for _, e := range errs {
		if e.Code == "E1010" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1010 for invalid binary literal")
	}
}

// TestE1010InvalidBinaryDigit tests invalid digit in binary literal
func TestE1010InvalidBinaryDigit(t *testing.T) {
	errs := lexErrors("0b2")
	found := false
	for _, e := range errs {
		if e.Code == "E1010" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1010 for invalid binary digit")
	}
}

// TestE1010MultipleDecimalPoints tests multiple decimal points detection
func TestE1010MultipleDecimalPoints(t *testing.T) {
	errs := lexErrors("1.2.3")
	found := false
	for _, e := range errs {
		if e.Code == "E1010" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1010 for multiple decimal points")
	}
}

// TestE1010InvalidExponent tests invalid exponent detection
func TestE1010InvalidExponent(t *testing.T) {
	errs := lexErrors("1e")
	found := false
	for _, e := range errs {
		if e.Code == "E1010" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1010 for invalid exponent")
	}
}

// TestE1016DecimalNoDigits tests decimal point without following digits
func TestE1016DecimalNoDigits(t *testing.T) {
	errs := lexErrors("1. ")
	found := false
	for _, e := range errs {
		if e.Code == "E1016" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1016 for decimal point without digits")
	}
}

// TestHexUnderscores tests underscores in hex literals
func TestHexUnderscores(t *testing.T) {
	tokens := tokenize("0xFF_FF")
	if len(tokens) < 1 {
		t.Fatal("expected at least 1 token")
	}
	if tokens[0].Literal != "0xFF_FF" {
		t.Errorf("literal = %s, want 0xFF_FF", tokens[0].Literal)
	}
}

// TestHexConsecutiveUnderscores tests consecutive underscores in hex
func TestHexConsecutiveUnderscores(t *testing.T) {
	errs := lexErrors("0xFF__FF")
	found := false
	for _, e := range errs {
		if e.Code == "E1011" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1011 for consecutive underscores in hex")
	}
}

// TestBinaryConsecutiveUnderscores tests consecutive underscores in binary
func TestBinaryConsecutiveUnderscores(t *testing.T) {
	errs := lexErrors("0b10__10")
	found := false
	for _, e := range errs {
		if e.Code == "E1011" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1011 for consecutive underscores in binary")
	}
}

// TestHexTrailingUnderscore tests trailing underscore in hex
func TestHexTrailingUnderscore(t *testing.T) {
	errs := lexErrors("0xFF_")
	found := false
	for _, e := range errs {
		if e.Code == "E1013" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1013 for trailing underscore in hex")
	}
}

// TestBinaryTrailingUnderscore tests trailing underscore in binary
func TestBinaryTrailingUnderscore(t *testing.T) {
	errs := lexErrors("0b101_")
	found := false
	for _, e := range errs {
		if e.Code == "E1013" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1013 for trailing underscore in binary")
	}
}

// TestScientificNotationWithSign tests scientific notation with explicit signs
func TestScientificNotationWithSign(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"1e+5", "1e+5"},
		{"1e-5", "1e-5"},
		{"2.5E+10", "2.5E+10"},
		{"2.5E-10", "2.5E-10"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if len(tokens) < 1 {
				t.Fatal("expected at least 1 token")
			}
			if tokens[0].Type != tokenizer.FLOAT {
				t.Errorf("type = %s, want FLOAT", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("literal = %s, want %s", tokens[0].Literal, tt.literal)
			}
		})
	}
}

// TestExponentUnderscores tests underscores in exponent
func TestExponentUnderscores(t *testing.T) {
	tokens := tokenize("1e1_0")
	if len(tokens) < 1 {
		t.Fatal("expected at least 1 token")
	}
	if tokens[0].Literal != "1e1_0" {
		t.Errorf("literal = %s, want 1e1_0", tokens[0].Literal)
	}
}

// TestExponentConsecutiveUnderscores tests consecutive underscores in exponent
func TestExponentConsecutiveUnderscores(t *testing.T) {
	errs := lexErrors("1e1__0")
	found := false
	for _, e := range errs {
		if e.Code == "E1011" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1011 for consecutive underscores in exponent")
	}
}

// TestExponentTrailingUnderscore tests trailing underscore in exponent
func TestExponentTrailingUnderscore(t *testing.T) {
	errs := lexErrors("1e10_")
	found := false
	for _, e := range errs {
		if e.Code == "E1013" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected E1013 for trailing underscore in exponent")
	}
}

// ============================================================================
// Lexer State Tests
// ============================================================================

// TestLexerSaveRestoreState tests save and restore state functionality
func TestLexerSaveRestoreState(t *testing.T) {
	l := NewLexer("hello world")

	// Read first token
	tok1 := l.NextToken()
	if tok1.Literal != "hello" {
		t.Errorf("first token = %s, want hello", tok1.Literal)
	}

	// Save state
	state := l.SaveState()

	// Read second token
	tok2 := l.NextToken()
	if tok2.Literal != "world" {
		t.Errorf("second token = %s, want world", tok2.Literal)
	}

	// Restore state
	l.RestoreState(state)

	// Should read "world" again
	tok3 := l.NextToken()
	if tok3.Literal != "world" {
		t.Errorf("after restore = %s, want world", tok3.Literal)
	}
}

// TestNewLexerWithOffset tests creating lexer with position offset
func TestNewLexerWithOffset(t *testing.T) {
	l := NewLexerWithOffset("test code", 10, 5)
	tok := l.NextToken()

	// Token should be at line 10, column adjusted by offset
	if tok.Line != 10 {
		t.Errorf("line = %d, want 10", tok.Line)
	}
}
