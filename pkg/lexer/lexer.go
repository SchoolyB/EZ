package lexer

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"github.com/marshallburns/ez/pkg/tokenizer"
)

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int
	column       int
	errors       []LexerError
}

// LexerError holds error information from lexing
type LexerError struct {
	Message string
	Line    int
	Column  int
	Code    string // Error code like "E1005"
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0, errors: []LexerError{}}
	l.readChar()
	return l
}

// NewLexerWithOffset creates a new Lexer with specified starting line and column
// This is useful for parsing substrings like string interpolation expressions
// where the offset should match the original source position
func NewLexerWithOffset(input string, startLine, startColumn int) *Lexer {
	l := &Lexer{input: input, line: startLine, column: startColumn, errors: []LexerError{}}
	l.readChar()
	return l
}

// Errors returns any lexer errors
func (l *Lexer) Errors() []LexerError {
	return l.errors
}

// addError adds an error to the lexer
func (l *Lexer) addError(code, message string, line, column int) {
	l.errors = append(l.errors, LexerError{
		Code:    code,
		Message: message,
		Line:    line,
		Column:  column,
	})
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() tokenizer.Token {
	var tok tokenizer.Token

	// Skip whitespace and comments in a loop
	for {
		l.skipWhitespace()
		if l.ch == '/' && (l.peekChar() == '/' || l.peekChar() == '*') {
			l.skipComment()
		} else {
			break
		}
	}

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.EQ, Literal: "==", Line: l.line, Column: l.column}
		} else {
			tok = newToken(tokenizer.ASSIGN, l.ch, l.line, l.column)
		}
	case '+':
		if l.peekChar() == '=' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.PLUS_ASSIGN, Literal: "+=", Line: l.line, Column: l.column}
		} else if l.peekChar() == '+' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.INCREMENT, Literal: "++", Line: l.line, Column: l.column}
		} else {
			tok = newToken(tokenizer.PLUS, l.ch, l.line, l.column)
		}
	case '-':
		if l.peekChar() == '>' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.ARROW, Literal: "->", Line: l.line, Column: l.column}
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.MINUS_ASSIGN, Literal: "-=", Line: l.line, Column: l.column}
		} else if l.peekChar() == '-' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.DECREMENT, Literal: "--", Line: l.line, Column: l.column}
		} else {
			tok = newToken(tokenizer.MINUS, l.ch, l.line, l.column)
		}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.NOT_EQ, Literal: "!=", Line: l.line, Column: l.column}
		} else if l.peekChar() == 'i' && l.peekAhead(2) == 'n' {
			// Check for !in
			l.readChar() // consume 'i'
			l.readChar() // consume 'n'
			tok = tokenizer.Token{Type: tokenizer.NOT_IN, Literal: "!in", Line: l.line, Column: l.column}
		} else {
			tok = newToken(tokenizer.BANG, l.ch, l.line, l.column)
		}
	case '*':
		if l.peekChar() == '=' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.ASTERISK_ASSIGN, Literal: "*=", Line: l.line, Column: l.column}
		} else {
			tok = newToken(tokenizer.ASTERISK, l.ch, l.line, l.column)
		}
	case '/':
		if l.peekChar() == '=' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.SLASH_ASSIGN, Literal: "/=", Line: l.line, Column: l.column}
		} else {
			tok = newToken(tokenizer.SLASH, l.ch, l.line, l.column)
		}
	case '%':
		if l.peekChar() == '=' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.PERCENT_ASSIGN, Literal: "%=", Line: l.line, Column: l.column}
		} else {
			tok = newToken(tokenizer.PERCENT, l.ch, l.line, l.column)
		}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.LT_EQ, Literal: "<=", Line: l.line, Column: l.column}
		} else {
			tok = newToken(tokenizer.LT, l.ch, l.line, l.column)
		}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.GT_EQ, Literal: ">=", Line: l.line, Column: l.column}
		} else {
			tok = newToken(tokenizer.GT, l.ch, l.line, l.column)
		}
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.AND, Literal: "&&", Line: l.line, Column: l.column}
		} else {
			// Single & is valid for "import & use" syntax
			tok = newToken(tokenizer.AMPERSAND, l.ch, l.line, l.column)
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.OR, Literal: "||", Line: l.line, Column: l.column}
		} else {
			l.addError("E1002", "unexpected character '|', did you mean '||'?", l.line, l.column)
			tok = newToken(tokenizer.ILLEGAL, l.ch, l.line, l.column)
		}
	case ',':
		tok = newToken(tokenizer.COMMA, l.ch, l.line, l.column)
	case ':':
		tok = newToken(tokenizer.COLON, l.ch, l.line, l.column)
	case ';':
		tok = newToken(tokenizer.SEMICOLON, l.ch, l.line, l.column)
	case '(':
		tok = newToken(tokenizer.LPAREN, l.ch, l.line, l.column)
	case ')':
		tok = newToken(tokenizer.RPAREN, l.ch, l.line, l.column)
	case '{':
		tok = newToken(tokenizer.LBRACE, l.ch, l.line, l.column)
	case '}':
		tok = newToken(tokenizer.RBRACE, l.ch, l.line, l.column)
	case '[':
		tok = newToken(tokenizer.LBRACKET, l.ch, l.line, l.column)
	case ']':
		tok = newToken(tokenizer.RBRACKET, l.ch, l.line, l.column)
	case '.':
		tok = newToken(tokenizer.DOT, l.ch, l.line, l.column)
	case '@':
		// Peek ahead to check for @ignore or @suppress
		if l.peekAheadString(7) == "@ignore" {
			// Found @ignore
			tok = tokenizer.Token{Type: tokenizer.IGNORE, Literal: "@ignore", Line: l.line, Column: l.column}
			// Consume the entire @ignore token
			for i := 0; i < 7; i++ {
				l.readChar()
			}
			return tok
		} else if l.peekAheadString(9) == "@suppress" {
			// Found @suppress
			tok = tokenizer.Token{Type: tokenizer.SUPPRESS, Literal: "@suppress", Line: l.line, Column: l.column}
			// Consume the entire @suppress token
			for i := 0; i < 9; i++ {
				l.readChar()
			}
			return tok
		}
		// Just a regular @ symbol (e.g., for imports like @std)
		tok = newToken(tokenizer.AT, l.ch, l.line, l.column)
	case '"':
		tok.Line = l.line
		tok.Column = l.column
		tok.Type = tokenizer.STRING
		var ok bool
		tok.Literal, ok = l.readString()
		if ok {
			l.readChar() // consume closing quote
		}
		return tok
	case '\'':
		tok.Type = tokenizer.CHAR
		tok.Literal = l.readCharValue()
		tok.Line = l.line
		tok.Column = l.column
		return tok
	case 0:
		tok.Literal = ""
		tok.Type = tokenizer.EOF
	default:
		if isLetter(l.ch) {
			startLine := l.line
			startColumn := l.column
			tok.Literal = l.readIdentifier()
			tok.Type = tokenizer.LookupIdentifier(tok.Literal)
			tok.Line = startLine
			tok.Column = startColumn
			return tok
		} else if isDigit(l.ch) {
			startLine := l.line
			startColumn := l.column
			tok.Literal, tok.Type = l.readNumber()
			tok.Line = startLine
			tok.Column = startColumn
			return tok
		} else {
			tok = newToken(tokenizer.ILLEGAL, l.ch, l.line, l.column)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) peekAhead(n int) byte {
	pos := l.readPosition + n - 1
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

// peekAheadString looks ahead n characters and returns the string (including current char)
func (l *Lexer) peekAheadString(n int) string {
	end := l.position + n
	if end > len(l.input) {
		end = len(l.input)
	}
	return l.input[l.position:end]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	if l.ch == '/' && l.peekChar() == '/' {
		// Single-line comment
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
	} else if l.ch == '/' && l.peekChar() == '*' {
		// Multi-line comment
		startLine := l.line
		startColumn := l.column
		l.readChar() // consume '/'
		l.readChar() // consume '*'
		for {
			if l.ch == 0 {
				// Reached EOF without closing comment
				l.addError("E1003", "unclosed multi-line comment", startLine, startColumn)
				break
			}
			if l.ch == '*' && l.peekChar() == '/' {
				l.readChar() // consume '*'
				l.readChar() // consume '/'
				break
			}
			l.readChar()
		}
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() (string, tokenizer.TokenType) {
	position := l.position
	startLine := l.line
	startColumn := l.column
	tokenType := tokenizer.INT

	// Check for hex (0x/0X) or binary (0b/0B) prefix
	if l.ch == '0' {
		next := l.peekChar()
		if next == 'x' || next == 'X' {
			// Hex literal
			l.readChar() // consume '0'
			l.readChar() // consume 'x'
			if !isHexDigit(l.ch) {
				l.addError("E1010", "invalid hex literal: expected hex digit after 0x", startLine, startColumn)
				return l.input[position:l.position], tokenizer.INT
			}
			for isHexDigit(l.ch) || l.ch == '_' {
				if l.ch == '_' {
					next := l.peekChar()
					if next == '_' {
						l.addError("E1011", "consecutive underscores not allowed in numeric literals", startLine, startColumn)
					}
					if !isHexDigit(next) {
						l.addError("E1013", "numeric literal cannot end with underscore", startLine, startColumn)
					}
				}
				l.readChar()
			}
			return l.input[position:l.position], tokenizer.INT
		} else if next == 'b' || next == 'B' {
			// Binary literal
			l.readChar() // consume '0'
			l.readChar() // consume 'b'
			if l.ch != '0' && l.ch != '1' {
				l.addError("E1010", "invalid binary literal: expected 0 or 1 after 0b", startLine, startColumn)
				return l.input[position:l.position], tokenizer.INT
			}
			for l.ch == '0' || l.ch == '1' || l.ch == '_' {
				if l.ch == '_' {
					next := l.peekChar()
					if next == '_' {
						l.addError("E1011", "consecutive underscores not allowed in numeric literals", startLine, startColumn)
					}
					if next != '0' && next != '1' {
						l.addError("E1013", "numeric literal cannot end with underscore", startLine, startColumn)
					}
				}
				l.readChar()
			}
			return l.input[position:l.position], tokenizer.INT
		}
	}

	// Read integer part (digits and underscores)
	for isDigit(l.ch) || l.ch == '_' {
		if l.ch == '_' {
			// Check for invalid underscore placement
			next := l.peekChar()

			// Cannot have consecutive underscores
			if next == '_' {
				l.addError("E1011", "consecutive underscores not allowed in numeric literals", startLine, startColumn)
				l.readChar()
				continue
			}

			// Cannot have underscore at end (followed by non-digit/non-underscore)
			if !isDigit(next) && next != '.' {
				// This might be the trailing underscore
				l.addError("E1013", "numeric literal cannot end with underscore", startLine, startColumn)
			}
		}
		l.readChar()
	}

	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = tokenizer.FLOAT

		// Check if there's an underscore before the decimal point
		if position < l.position && l.input[l.position-1] == '_' {
			l.addError("E1014", "underscore cannot appear immediately before decimal point", startLine, startColumn)
		}

		l.readChar() // consume '.'

		// Check if there's an underscore after the decimal point
		if l.ch == '_' {
			l.addError("E1015", "underscore cannot appear immediately after decimal point", startLine, startColumn)
			l.readChar()
		}

		// Read fractional part (digits and underscores)
		for isDigit(l.ch) || l.ch == '_' {
			if l.ch == '_' {
				next := l.peekChar()

				// Cannot have consecutive underscores
				if next == '_' {
					l.addError("E1011", "consecutive underscores not allowed in numeric literals", startLine, startColumn)
					l.readChar()
					continue
				}

				// Cannot end with underscore
				if !isDigit(next) {
					l.addError("E1013", "numeric literal cannot end with underscore", startLine, startColumn)
				}
			}
			l.readChar()
		}

		// Check for invalid multiple decimal points
		if l.ch == '.' {
			l.addError("E1010", "invalid number format: multiple decimal points", startLine, startColumn)
		}
	} else if l.ch == '.' && !isDigit(l.peekChar()) {
		// Number ends with a dot but no digits after (e.g., "1." or "1._")
		// But check if underscore before dot
		if position < l.position && l.input[l.position-1] == '_' {
			l.addError("E1014", "underscore cannot appear immediately before decimal point", startLine, startColumn)
		}
		// Check if underscore after dot
		if l.peekChar() == '_' {
			l.addError("E1015", "underscore cannot appear immediately after decimal point", startLine, startColumn)
		} else {
			l.addError("E1016", "invalid number format: decimal point must be followed by digits", startLine, startColumn)
		}
	}

	// Check for leading underscore (number starts with underscore)
	literal := l.input[position:l.position]
	if len(literal) > 0 && literal[0] == '_' {
		l.addError("E1012", "numeric literal cannot start with underscore", startLine, startColumn)
	}

	return literal, tokenType
}

func (l *Lexer) readString() (string, bool) {
	startLine := l.line
	startColumn := l.column
	position := l.position + 1
	braceDepth := 0 // Track nested ${...} interpolation blocks

	for {
		l.readChar()

		// Handle interpolation start ${
		if l.ch == '$' && l.peekChar() == '{' {
			l.readChar() // consume '{'
			braceDepth++
			continue
		}

		// Handle nested braces inside interpolation
		if braceDepth > 0 {
			if l.ch == '{' {
				braceDepth++
			} else if l.ch == '}' {
				braceDepth--
			}
			// Inside interpolation, ignore quotes
			if l.ch == 0 || l.ch == '\n' {
				l.addError("E1004", "unclosed string literal", startLine, startColumn)
				return l.input[position:l.position], false
			}
			continue
		}

		// Outside interpolation - normal string handling
		if l.ch == '"' {
			return l.input[position:l.position], true
		}
		if l.ch == 0 || l.ch == '\n' {
			// Unclosed string
			l.addError("E1004", "unclosed string literal", startLine, startColumn)
			return l.input[position:l.position], false
		}
		if l.ch == '\\' {
			// Check for valid escape sequences
			next := l.peekChar()
			validEscapes := map[byte]bool{
				'n': true, 't': true, 'r': true, '\\': true, '"': true, '\'': true, '0': true,
			}
			if !validEscapes[next] && next != 0 {
				l.addError("E1006", "invalid escape sequence '\\"+string(next)+"' in string", l.line, l.column)
			}
			l.readChar() // skip escaped character
		}
	}
}

func (l *Lexer) readCharValue() string {
	startLine := l.line
	startColumn := l.column
	l.readChar() // skip opening quote

	// Check for empty char literal
	if l.ch == '\'' {
		l.addError("E1008", "empty character literal", startLine, startColumn)
		l.readChar() // consume closing quote
		return ""
	}

	// Check for EOF or newline
	if l.ch == 0 || l.ch == '\n' {
		l.addError("E1005", "unclosed character literal", startLine, startColumn)
		return ""
	}

	ch := string(l.ch)

	// Handle escape sequences
	if l.ch == '\\' {
		l.readChar()
		// Validate escape sequence
		validEscapes := map[byte]bool{
			'n': true, 't': true, 'r': true, '\\': true, '"': true, '\'': true, '0': true,
		}
		if !validEscapes[l.ch] && l.ch != 0 {
			l.addError("E1007", "invalid escape sequence '\\"+string(l.ch)+"' in character literal", startLine, startColumn)
		}
		ch = "\\" + string(l.ch)
	}

	l.readChar() // move to what should be closing quote

	// Check for multi-character literal
	if l.ch != '\'' {
		// This is a multi-character literal
		l.addError("E1009", "character literal must contain exactly one character", startLine, startColumn)
		// Consume until we find closing quote or newline/EOF
		for l.ch != '\'' && l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
	}

	// Check for closing quote
	if l.ch != '\'' {
		l.addError("E1005", "unclosed character literal", startLine, startColumn)
	} else {
		l.readChar() // consume closing quote
	}

	return ch
}

func newToken(tokenType tokenizer.TokenType, ch byte, line, column int) tokenizer.Token {
	return tokenizer.Token{Type: tokenType, Literal: string(ch), Line: line, Column: column}
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isHexDigit(ch byte) bool {
	return ('0' <= ch && ch <= '9') || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}
