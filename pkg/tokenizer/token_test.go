package tokenizer

import "testing"

// TestTokenTypeConstants verifies all token type constants are defined correctly
func TestTokenTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		token    TokenType
		expected string
	}{
		// Special tokens
		{"ILLEGAL", ILLEGAL, "ILLEGAL"},
		{"EOF", EOF, "EOF"},

		// Identifiers and literals
		{"IDENT", IDENT, "IDENT"},
		{"INT", INT, "INT"},
		{"FLOAT", FLOAT, "FLOAT"},
		{"STRING", STRING, "STRING"},
		{"CHAR", CHAR, "CHAR"},

		// Operators
		{"ASSIGN", ASSIGN, "="},
		{"PLUS", PLUS, "+"},
		{"MINUS", MINUS, "-"},
		{"BANG", BANG, "!"},
		{"ASTERISK", ASTERISK, "*"},
		{"SLASH", SLASH, "/"},
		{"PERCENT", PERCENT, "%"},

		// Comparison
		{"LT", LT, "<"},
		{"GT", GT, ">"},
		{"EQ", EQ, "=="},
		{"NOT_EQ", NOT_EQ, "!="},
		{"LT_EQ", LT_EQ, "<="},
		{"GT_EQ", GT_EQ, ">="},

		// Compound assignment
		{"PLUS_ASSIGN", PLUS_ASSIGN, "+="},
		{"MINUS_ASSIGN", MINUS_ASSIGN, "-="},
		{"ASTERISK_ASSIGN", ASTERISK_ASSIGN, "*="},
		{"SLASH_ASSIGN", SLASH_ASSIGN, "/="},
		{"PERCENT_ASSIGN", PERCENT_ASSIGN, "%="},

		// Increment/Decrement
		{"INCREMENT", INCREMENT, "++"},
		{"DECREMENT", DECREMENT, "--"},

		// Logical
		{"AND", AND, "&&"},
		{"OR", OR, "||"},

		// Delimiters
		{"COMMA", COMMA, ","},
		{"COLON", COLON, ":"},
		{"SEMICOLON", SEMICOLON, ";"},
		{"NEWLINE", NEWLINE, "NEWLINE"},
		{"LPAREN", LPAREN, "("},
		{"RPAREN", RPAREN, ")"},
		{"LBRACE", LBRACE, "{"},
		{"RBRACE", RBRACE, "}"},
		{"LBRACKET", LBRACKET, "["},
		{"RBRACKET", RBRACKET, "]"},

		// Arrow and Dot
		{"ARROW", ARROW, "->"},
		{"DOT", DOT, "."},

		// At sign
		{"AT", AT, "@"},

		// Ampersand
		{"AMPERSAND", AMPERSAND, "&"},

		// Keywords
		{"TEMP", TEMP, "TEMP"},
		{"CONST", CONST, "CONST"},
		{"DO", DO, "DO"},
		{"RETURN", RETURN, "RETURN"},
		{"IF", IF, "IF"},
		{"OR_KW", OR_KW, "OR_KW"},
		{"OTHERWISE", OTHERWISE, "OTHERWISE"},
		{"FOR", FOR, "FOR"},
		{"FOR_EACH", FOR_EACH, "FOR_EACH"},
		{"AS_LONG_AS", AS_LONG_AS, "AS_LONG_AS"},
		{"LOOP", LOOP, "LOOP"},
		{"BREAK", BREAK, "BREAK"},
		{"CONTINUE", CONTINUE, "CONTINUE"},
		{"IN", IN, "IN"},
		{"NOT_IN", NOT_IN, "NOT_IN"},
		{"RANGE", RANGE, "RANGE"},
		{"IMPORT", IMPORT, "IMPORT"},
		{"USING", USING, "USING"},
		{"STRUCT", STRUCT, "STRUCT"},
		{"ENUM", ENUM, "ENUM"},
		{"NIL", NIL, "NIL"},
		{"NEW", NEW, "NEW"},
		{"TRUE", TRUE, "TRUE"},
		{"FALSE", FALSE, "FALSE"},
		{"IGNORE", IGNORE, "IGNORE"},
		{"SUPPRESS", SUPPRESS, "SUPPRESS"},
		{"MODULE", MODULE, "MODULE"},
		{"PRIVATE", PRIVATE, "PRIVATE"},
		{"FROM", FROM, "FROM"},
		{"USE", USE, "USE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.token) != tt.expected {
				t.Errorf("TokenType %s = %q, want %q", tt.name, tt.token, tt.expected)
			}
		})
	}
}

// TestLookupIdentifier tests keyword recognition
func TestLookupIdentifier(t *testing.T) {
	tests := []struct {
		ident    string
		expected TokenType
	}{
		// All keywords should return their specific token type
		{"temp", TEMP},
		{"const", CONST},
		{"do", DO},
		{"return", RETURN},
		{"if", IF},
		{"or", OR_KW},
		{"otherwise", OTHERWISE},
		{"for", FOR},
		{"for_each", FOR_EACH},
		{"as_long_as", AS_LONG_AS},
		{"loop", LOOP},
		{"break", BREAK},
		{"continue", CONTINUE},
		{"in", IN},
		{"not_in", NOT_IN},
		{"range", RANGE},
		{"import", IMPORT},
		{"using", USING},
		{"struct", STRUCT},
		{"enum", ENUM},
		{"nil", NIL},
		{"new", NEW},
		{"true", TRUE},
		{"false", FALSE},
		{"@ignore", IGNORE},
		{"module", MODULE},
		{"private", PRIVATE},
		{"from", FROM},
		{"use", USE},

		// Non-keywords should return IDENT
		{"foo", IDENT},
		{"bar", IDENT},
		{"myVariable", IDENT},
		{"x", IDENT},
		{"_underscore", IDENT},
		{"camelCase", IDENT},
		{"PascalCase", IDENT},
		{"snake_case", IDENT},
		{"with123numbers", IDENT},

		// Case-sensitive: uppercase keywords should be IDENT
		{"TEMP", IDENT},
		{"Const", IDENT},
		{"DO", IDENT},
		{"IF", IDENT},
		{"True", IDENT},
		{"FALSE", IDENT},
	}

	for _, tt := range tests {
		t.Run(tt.ident, func(t *testing.T) {
			got := LookupIdentifier(tt.ident)
			if got != tt.expected {
				t.Errorf("LookupIdentifier(%q) = %q, want %q", tt.ident, got, tt.expected)
			}
		})
	}
}

// TestTokenStruct tests Token struct initialization
func TestTokenStruct(t *testing.T) {
	tok := Token{
		Type:    INT,
		Literal: "42",
		Line:    10,
		Column:  5,
	}

	if tok.Type != INT {
		t.Errorf("Token.Type = %q, want %q", tok.Type, INT)
	}
	if tok.Literal != "42" {
		t.Errorf("Token.Literal = %q, want %q", tok.Literal, "42")
	}
	if tok.Line != 10 {
		t.Errorf("Token.Line = %d, want %d", tok.Line, 10)
	}
	if tok.Column != 5 {
		t.Errorf("Token.Column = %d, want %d", tok.Column, 5)
	}
}

// TestKeywordCount verifies the expected number of keywords
func TestKeywordCount(t *testing.T) {
	// Count keywords in the map (including @ignore)
	expectedCount := 29
	actualCount := 0

	// Test all known keywords
	knownKeywords := []string{
		"temp", "const", "do", "return", "if", "or", "otherwise",
		"for", "for_each", "as_long_as", "loop", "break", "continue",
		"in", "not_in", "range", "import", "using", "struct", "enum",
		"nil", "new", "true", "false", "@ignore", "module", "private",
		"from", "use",
	}

	for _, kw := range knownKeywords {
		if LookupIdentifier(kw) != IDENT {
			actualCount++
		}
	}

	if actualCount != expectedCount {
		t.Errorf("Keyword count = %d, want %d", actualCount, expectedCount)
	}
}

// TestIsKeyword tests the IsKeyword function
func TestIsKeyword(t *testing.T) {
	// All keyword token types should return true
	keywordTypes := []TokenType{
		TEMP, CONST, DO, RETURN, IF, OR_KW, OTHERWISE,
		FOR, FOR_EACH, AS_LONG_AS, LOOP, BREAK, CONTINUE,
		IN, NOT_IN, RANGE, IMPORT, USING, STRUCT, ENUM,
		NIL, NEW, TRUE, FALSE, IGNORE, MODULE, PRIVATE, FROM, USE,
	}

	for _, tt := range keywordTypes {
		t.Run("keyword_"+string(tt), func(t *testing.T) {
			if !IsKeyword(tt) {
				t.Errorf("IsKeyword(%q) = false, want true", tt)
			}
		})
	}

	// Non-keyword token types should return false
	nonKeywordTypes := []TokenType{
		ILLEGAL, EOF, IDENT, INT, FLOAT, STRING, CHAR,
		ASSIGN, PLUS, MINUS, BANG, ASTERISK, SLASH, PERCENT,
		LT, GT, EQ, NOT_EQ, LT_EQ, GT_EQ,
		PLUS_ASSIGN, MINUS_ASSIGN, ASTERISK_ASSIGN, SLASH_ASSIGN, PERCENT_ASSIGN,
		INCREMENT, DECREMENT, AND, OR,
		COMMA, COLON, SEMICOLON, NEWLINE,
		LPAREN, RPAREN, LBRACE, RBRACE, LBRACKET, RBRACKET,
		ARROW, DOT, AT, AMPERSAND, SUPPRESS,
	}

	for _, tt := range nonKeywordTypes {
		t.Run("non_keyword_"+string(tt), func(t *testing.T) {
			if IsKeyword(tt) {
				t.Errorf("IsKeyword(%q) = true, want false", tt)
			}
		})
	}
}

// TestKeywordLiteral tests the KeywordLiteral function
func TestKeywordLiteral(t *testing.T) {
	tests := []struct {
		tokenType TokenType
		expected  string
	}{
		{TEMP, "temp"},
		{CONST, "const"},
		{DO, "do"},
		{RETURN, "return"},
		{IF, "if"},
		{OR_KW, "or"},
		{OTHERWISE, "otherwise"},
		{FOR, "for"},
		{FOR_EACH, "for_each"},
		{AS_LONG_AS, "as_long_as"},
		{LOOP, "loop"},
		{BREAK, "break"},
		{CONTINUE, "continue"},
		{IN, "in"},
		{NOT_IN, "not_in"},
		{RANGE, "range"},
		{IMPORT, "import"},
		{USING, "using"},
		{STRUCT, "struct"},
		{ENUM, "enum"},
		{NIL, "nil"},
		{NEW, "new"},
		{TRUE, "true"},
		{FALSE, "false"},
		{IGNORE, "@ignore"},
		{MODULE, "module"},
		{PRIVATE, "private"},
		{FROM, "from"},
		{USE, "use"},
	}

	for _, tt := range tests {
		t.Run(string(tt.tokenType), func(t *testing.T) {
			got := KeywordLiteral(tt.tokenType)
			if got != tt.expected {
				t.Errorf("KeywordLiteral(%q) = %q, want %q", tt.tokenType, got, tt.expected)
			}
		})
	}
}

// TestKeywordLiteralNonKeyword tests KeywordLiteral with non-keyword tokens
func TestKeywordLiteralNonKeyword(t *testing.T) {
	// Non-keywords should return their string representation
	nonKeywords := []TokenType{
		IDENT, INT, FLOAT, STRING, PLUS, MINUS, ASSIGN,
	}

	for _, tt := range nonKeywords {
		t.Run("non_keyword_"+string(tt), func(t *testing.T) {
			got := KeywordLiteral(tt)
			expected := string(tt)
			if got != expected {
				t.Errorf("KeywordLiteral(%q) = %q, want %q", tt, got, expected)
			}
		})
	}
}
