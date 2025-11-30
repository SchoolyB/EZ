package tokenizer

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

const (
	// Special tokens
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"

	// Identifiers and literals
	IDENT  TokenType = "IDENT"
	INT    TokenType = "INT"
	FLOAT  TokenType = "FLOAT"
	STRING TokenType = "STRING"
	CHAR   TokenType = "CHAR"

	// Operators
	ASSIGN   TokenType = "="
	PLUS     TokenType = "+"
	MINUS    TokenType = "-"
	BANG     TokenType = "!"
	ASTERISK TokenType = "*"
	SLASH    TokenType = "/"
	PERCENT  TokenType = "%"

	// Comparison
	LT     TokenType = "<"
	GT     TokenType = ">"
	EQ     TokenType = "=="
	NOT_EQ TokenType = "!="
	LT_EQ  TokenType = "<="
	GT_EQ  TokenType = ">="

	// Compound assignment
	PLUS_ASSIGN     TokenType = "+="
	MINUS_ASSIGN    TokenType = "-="
	ASTERISK_ASSIGN TokenType = "*="
	SLASH_ASSIGN    TokenType = "/="
	PERCENT_ASSIGN  TokenType = "%="

	// Increment/Decrement
	INCREMENT TokenType = "++"
	DECREMENT TokenType = "--"

	// Logical
	AND TokenType = "&&"
	OR  TokenType = "||"

	// Delimiters
	COMMA     TokenType = ","
	COLON     TokenType = ":"
	SEMICOLON TokenType = ";"
	NEWLINE   TokenType = "NEWLINE"

	LPAREN   TokenType = "("
	RPAREN   TokenType = ")"
	LBRACE   TokenType = "{"
	RBRACE   TokenType = "}"
	LBRACKET TokenType = "["
	RBRACKET TokenType = "]"

	// Arrow
	ARROW TokenType = "->"

	// Dot
	DOT TokenType = "."

	// At sign (for attributes and @ignore)
	AT TokenType = "@"

	// Keywords
	TEMP       TokenType = "TEMP"
	CONST      TokenType = "CONST"
	DO         TokenType = "DO"
	RETURN     TokenType = "RETURN"
	IF         TokenType = "IF"
	OR_KW      TokenType = "OR_KW"
	OTHERWISE  TokenType = "OTHERWISE"
	FOR        TokenType = "FOR"
	FOR_EACH   TokenType = "FOR_EACH"
	AS_LONG_AS TokenType = "AS_LONG_AS"
	LOOP       TokenType = "LOOP"
	BREAK      TokenType = "BREAK"
	CONTINUE   TokenType = "CONTINUE"
	IN         TokenType = "IN"
	NOT_IN     TokenType = "NOT_IN"
	RANGE      TokenType = "RANGE"
	IMPORT     TokenType = "IMPORT"
	USING      TokenType = "USING"
	STRUCT     TokenType = "STRUCT"
	ENUM       TokenType = "ENUM"
	NIL        TokenType = "NIL"
	NEW        TokenType = "NEW"
	TRUE       TokenType = "TRUE"
	FALSE      TokenType = "FALSE"
	IGNORE     TokenType = "IGNORE"
	SUPPRESS   TokenType = "SUPPRESS"

	// Module system keywords
	MODULE  TokenType = "MODULE"
	PRIVATE TokenType = "PRIVATE"
	FROM    TokenType = "FROM"

	// Ampersand (for import & use syntax)
	AMPERSAND TokenType = "&"
)

var keywords = map[string]TokenType{
	"temp":       TEMP,
	"const":      CONST,
	"do":         DO,
	"return":     RETURN,
	"if":         IF,
	"or":         OR_KW,
	"otherwise":  OTHERWISE,
	"for":        FOR,
	"for_each":   FOR_EACH,
	"as_long_as": AS_LONG_AS,
	"loop":       LOOP,
	"break":      BREAK,
	"continue":   CONTINUE,
	"in":         IN,
	"not_in":     NOT_IN,
	"range":      RANGE,
	"import":     IMPORT,
	"using":      USING,
	"struct":     STRUCT,
	"enum":       ENUM,
	"nil":        NIL,
	"new":        NEW,
	"true":       TRUE,
	"false":      FALSE,
	"@ignore":    IGNORE,
	"module":     MODULE,
	"private":    PRIVATE,
	"from":       FROM,
}

// Looks up the passed in identifier(i)
// if found, returns the TokenType
// representation of said identifer.
func LookupIdentifier(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
