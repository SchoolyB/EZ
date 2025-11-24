package lexer

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
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
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
			tok = newToken(tokenizer.ILLEGAL, l.ch, l.line, l.column)
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok = tokenizer.Token{Type: tokenizer.OR, Literal: "||", Line: l.line, Column: l.column}
		} else {
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
		if l.peekChar() == 'i' {
			// Check for @ignore
			start := l.position
			l.readChar() // consume '@'
			ident := l.readIdentifier()
			if ident == "ignore" {
				tok = tokenizer.Token{Type: tokenizer.IGNORE, Literal: "@ignore", Line: l.line, Column: l.column}
				return tok
			}
			// Not @ignore, backtrack (this is a simplification - may need better handling)
			tok = tokenizer.Token{Type: tokenizer.AT, Literal: "@", Line: l.line, Column: l.column}
			_ = start // silence unused variable
			return tok
		}
		tok = newToken(tokenizer.AT, l.ch, l.line, l.column)
	case '"':
		tok.Line = l.line
		tok.Column = l.column
		tok.Type = tokenizer.STRING
		tok.Literal = l.readString()
		l.readChar() // consume closing quote
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
			tok.Literal = l.readIdentifier()
			tok.Type = tokenizer.LookupIdentifier(tok.Literal)
			tok.Line = l.line
			tok.Column = l.column
			return tok
		} else if isDigit(l.ch) {
			tok.Line = l.line
			tok.Column = l.column
			tok.Literal, tok.Type = l.readNumber()
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
		l.readChar() // consume '/'
		l.readChar() // consume '*'
		for {
			if l.ch == 0 {
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
	tokenType := tokenizer.INT

	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = tokenizer.FLOAT
		l.readChar() // consume '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position], tokenType
}

func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
		if l.ch == '\\' {
			l.readChar() // skip escaped character
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) readCharValue() string {
	l.readChar() // skip opening quote
	ch := string(l.ch)
	if l.ch == '\\' {
		l.readChar()
		ch = "\\" + string(l.ch)
	}
	l.readChar() // skip to closing quote
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
