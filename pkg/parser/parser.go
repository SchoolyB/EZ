package parser
// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/errors"
	. "github.com/marshallburns/ez/pkg/lexer"
	. "github.com/marshallburns/ez/pkg/tokenizer"
)

// Reserved keywords that cannot be used as identifiers
var reservedKeywords = map[string]bool{
	"temp": true, "const": true, "do": true, "return": true,
	"if": true, "or": true, "otherwise": true,
	"for": true, "for_each": true, "as_long_as": true, "loop": true,
	"break": true, "continue": true, "in": true, "not_in": true, "range": true,
	"import": true, "using": true, "struct": true, "enum": true,
	"nil": true, "new": true, "true": true, "false": true,
}

// Builtin function names that cannot be redefined
var builtinNames = map[string]bool{
	"len": true, "typeof": true, "input": true,
	"int": true, "float": true, "string": true, "bool": true, "char": true,
	"println": true, "print": true, "read_int": true,
}

// isReservedName checks if a name is a reserved keyword or builtin
func isReservedName(name string) bool {
	return reservedKeywords[name] || builtinNames[name]
}

// Operator precedence levels
const (
	_ int = iota
	LOWEST
	OR_PREC      // ||
	AND_PREC     // &&
	EQUALS       // ==, !=
	LESSGREATER  // <, >, <=, >=
	MEMBERSHIP   // in, not_in, !in
	SUM          // +, -
	PRODUCT      // *, /, %
	PREFIX       // -X, !X
	POSTFIX      // x++, x--
	CALL         // function(x)
	INDEX        // array[index]
	MEMBER       // object.member
)

// Setting the order in which tokens should be evaluated
var priorities = map[TokenType]int{
	OR:           OR_PREC,
	AND:          AND_PREC,
	EQ:           EQUALS,
	NOT_EQ:       EQUALS,
	LT:           LESSGREATER,
	GT:           LESSGREATER,
	LT_EQ:        LESSGREATER,
	GT_EQ:        LESSGREATER,
	IN:           MEMBERSHIP,
	NOT_IN:       MEMBERSHIP,
	PLUS:         SUM,
	MINUS:        SUM,
	ASTERISK:     PRODUCT,
	SLASH:        PRODUCT,
	PERCENT:      PRODUCT,
	INCREMENT:    POSTFIX,
	DECREMENT:    POSTFIX,
	LPAREN:       CALL,
	LBRACKET:     INDEX,
	DOT:          MEMBER,
}

// Function types for parsing expressions based on token position.
// prefixParseFn: parses tokens at the start of an expression (literals, identifiers, unary operators)
// infixParseFn: parses tokens between expressions, receives the left-hand side (binary operators, calls, indexing)
type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

type Parser struct {
	l      *Lexer
	errors []string

	currentToken  Token
	peekToken     Token

	prefixParseFns map[TokenType]prefixParseFn
	infixParseFns  map[TokenType]infixParseFn

	// New error system
	ezErrors *errors.EZErrorList
	source   string
	filename string

	// Scope tracking for duplicate detection
	scopes []map[string]Token // stack of scopes, each mapping name to declaration token

	// Function scope tracking
	functionDepth int // depth of nested function scopes (0 = global, >0 = inside function)

	// Active suppressions from function/block attributes
	activeSuppressions []*Attribute
}

// Scope management methods
func (p *Parser) pushScope() {
	p.scopes = append(p.scopes, make(map[string]Token))
}

func (p *Parser) popScope() {
	if len(p.scopes) > 1 {
		p.scopes = p.scopes[:len(p.scopes)-1]
	}
}

func (p *Parser) currentScope() map[string]Token {
	return p.scopes[len(p.scopes)-1]
}

// declareInScope declares a name in current scope, returns error if duplicate
func (p *Parser) declareInScope(name string, token Token) bool {
	scope := p.currentScope()
	if _, exists := scope[name]; exists {
		return false // duplicate
	}
	scope[name] = token
	return true
}

// Returns a pointer to Parser(p)
// with initialized members
func New(l *Lexer) *Parser {
	p := &Parser{
		l:        l,
		errors:   []string{},
		ezErrors: errors.NewErrorList(),
		source:   "",
		filename: "<unknown>",
		scopes:   []map[string]Token{make(map[string]Token)}, // start with global scope
	}

	p.prefixParseFns = make(map[TokenType]prefixParseFn)
	p.setPrefix(IDENT, p.parseIdentifier)
	p.setPrefix(INT, p.parseIntegerValue)
	p.setPrefix(FLOAT, p.parseFloatValue)
	p.setPrefix(STRING, p.parseStringValue)
	p.setPrefix(CHAR, p.parseCharValue)
	p.setPrefix(TRUE, p.parseBooleanValue)
	p.setPrefix(FALSE, p.parseBooleanValue)
	p.setPrefix(NIL, p.parseNilValue)
	p.setPrefix(BANG, p.parsePrefixExpression)
	p.setPrefix(MINUS, p.parsePrefixExpression)
	p.setPrefix(LPAREN, p.parseGroupedExpression)
	p.setPrefix(LBRACE, p.parseArrayValue)
	p.setPrefix(NEW, p.parseNewExpression)
	p.setPrefix(RANGE, p.parseRangeExpression)

	p.infixParseFns = make(map[TokenType]infixParseFn)
	p.setInfix(PLUS, p.parseInfixExpression)
	p.setInfix(MINUS, p.parseInfixExpression)
	p.setInfix(ASTERISK, p.parseInfixExpression)
	p.setInfix(SLASH, p.parseInfixExpression)
	p.setInfix(PERCENT, p.parseInfixExpression)
	p.setInfix(EQ, p.parseInfixExpression)
	p.setInfix(NOT_EQ, p.parseInfixExpression)
	p.setInfix(LT, p.parseInfixExpression)
	p.setInfix(GT, p.parseInfixExpression)
	p.setInfix(LT_EQ, p.parseInfixExpression)
	p.setInfix(GT_EQ, p.parseInfixExpression)
	p.setInfix(AND, p.parseInfixExpression)
	p.setInfix(OR, p.parseInfixExpression)
	p.setInfix(IN, p.parseInfixExpression)
	p.setInfix(NOT_IN, p.parseInfixExpression)
	p.setInfix(LPAREN, p.parseCallExpression)
	p.setInfix(LBRACKET, p.parseIndexExpression)
	p.setInfix(DOT, p.parseMemberExpression)
	p.setInfix(INCREMENT, p.parsePostfixExpression)
	p.setInfix(DECREMENT, p.parsePostfixExpression)

	// Read two tokens to initialize currentToken and peekToken
	p.nextToken()
	p.nextToken()

	return p
}

// NewWithSource creates a parser with source context for better errors
func NewWithSource(l *Lexer, source, filename string) *Parser {
	p := &Parser{
		l:        l,
		errors:   []string{},
		ezErrors: errors.NewErrorList(),
		source:   source,
		filename: filename,
		scopes:   []map[string]Token{make(map[string]Token)}, // start with global scope
	}

	p.prefixParseFns = make(map[TokenType]prefixParseFn)
	p.setPrefix(IDENT, p.parseIdentifier)
	p.setPrefix(INT, p.parseIntegerValue)
	p.setPrefix(FLOAT, p.parseFloatValue)
	p.setPrefix(STRING, p.parseStringValue)
	p.setPrefix(CHAR, p.parseCharValue)
	p.setPrefix(TRUE, p.parseBooleanValue)
	p.setPrefix(FALSE, p.parseBooleanValue)
	p.setPrefix(NIL, p.parseNilValue)
	p.setPrefix(BANG, p.parsePrefixExpression)
	p.setPrefix(MINUS, p.parsePrefixExpression)
	p.setPrefix(LPAREN, p.parseGroupedExpression)
	p.setPrefix(LBRACE, p.parseArrayValue)
	p.setPrefix(NEW, p.parseNewExpression)
	p.setPrefix(RANGE, p.parseRangeExpression)

	p.infixParseFns = make(map[TokenType]infixParseFn)
	p.setInfix(PLUS, p.parseInfixExpression)
	p.setInfix(MINUS, p.parseInfixExpression)
	p.setInfix(ASTERISK, p.parseInfixExpression)
	p.setInfix(SLASH, p.parseInfixExpression)
	p.setInfix(PERCENT, p.parseInfixExpression)
	p.setInfix(EQ, p.parseInfixExpression)
	p.setInfix(NOT_EQ, p.parseInfixExpression)
	p.setInfix(LT, p.parseInfixExpression)
	p.setInfix(GT, p.parseInfixExpression)
	p.setInfix(LT_EQ, p.parseInfixExpression)
	p.setInfix(GT_EQ, p.parseInfixExpression)
	p.setInfix(AND, p.parseInfixExpression)
	p.setInfix(OR, p.parseInfixExpression)
	p.setInfix(IN, p.parseInfixExpression)
	p.setInfix(NOT_IN, p.parseInfixExpression)
	p.setInfix(LPAREN, p.parseCallExpression)
	p.setInfix(LBRACKET, p.parseIndexExpression)
	p.setInfix(DOT, p.parseMemberExpression)
	p.setInfix(INCREMENT, p.parsePostfixExpression)
	p.setInfix(DECREMENT, p.parsePostfixExpression)

	// Read two tokens to initialize currentToken and peekToken
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) setPrefix(tType TokenType, fn prefixParseFn) {
	p.prefixParseFns[tType] = fn
}

func (p *Parser) setInfix(tType TokenType, fn infixParseFn) {
	p.infixParseFns[tType] = fn
}

func (p *Parser) Errors() []string {
	return p.errors
}

// EZErrors returns the new error list
func (p *Parser) EZErrors() *errors.EZErrorList {
	return p.ezErrors
}

// addEZError adds a formatted error to the error list
func (p *Parser) addEZError(code errors.ErrorCode, message string, tok Token) {
	sourceLine := ""
	if p.source != "" {
		sourceLine = errors.GetSourceLine(p.source, tok.Line)
	}

	err := errors.NewErrorWithSource(
		code,
		message,
		p.filename,
		tok.Line,
		tok.Column,
		sourceLine,
	)
	err.EndColumn = tok.Column + len(tok.Literal)
	p.ezErrors.AddError(err)
}

func (p *Parser) nextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) currentTokenMatches(tType TokenType) bool {
	return p.currentToken.Type == tType
}

func (p *Parser) peekTokenMatches(tType TokenType) bool {
	return p.peekToken.Type == tType
}

func (p *Parser) expectPeek(tType TokenType) bool {
	if p.peekTokenMatches(tType) {
		p.nextToken()
		return true
	}
	p.peekError(tType)
	return false
}

func (p *Parser) peekError(tType TokenType) {
	var msg string
	var code errors.ErrorCode

	// Provide better error messages for unclosed delimiters
	if p.peekToken.Type == EOF {
		switch tType {
		case RPAREN:
			msg = "unclosed parenthesis - expected ')'"
			code = errors.E1004
		case RBRACE:
			msg = "unclosed brace - expected '}'"
			code = errors.E1004
		case RBRACKET:
			msg = "unclosed bracket - expected ']'"
			code = errors.E1004
		default:
			msg = fmt.Sprintf("unexpected end of file, expected %s", tType)
			code = errors.E1002
		}
	} else {
		msg = fmt.Sprintf("expected %s, got %s instead", tType, p.peekToken.Type)
		code = errors.E1002
	}

	p.errors = append(p.errors, msg)
	p.addEZError(code, msg, p.peekToken)
}

func (p *Parser) peekPrecedence() int {
	if p, ok := priorities[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := priorities[p.currentToken.Type]; ok {
		return p
	}
	return LOWEST
}

// ParseProgram parses the entire program
func (p *Parser) ParseProgram() *Program {
	program := &Program{}
	program.FileUsing = []*UsingStatement{}
	program.Statements = []Statement{}

	seenOtherDeclaration := false

	for !p.currentTokenMatches(EOF) {
		// Track what we've seen for placement validation
		if p.currentTokenMatches(USING) {
			// File-scoped using: must come before other declarations
			if seenOtherDeclaration {
				p.addEZError(errors.E1003, "file-scoped 'using' must come before all declarations", p.currentToken)
				p.nextToken()
				continue
			}

			// Parse and add to FileUsing
			usingStmt := p.parseUsingStatement()
			if usingStmt != nil {
				program.FileUsing = append(program.FileUsing, usingStmt)
			}
			p.nextToken()
			continue
		} else if !p.currentTokenMatches(EOF) && !p.currentTokenMatches(IMPORT) {
			seenOtherDeclaration = true
		}

		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() Statement {
	// Check for @suppress or @(...) attributes
	var attrs []*Attribute
	if p.currentTokenMatches(SUPPRESS) || p.currentTokenMatches(AT) {
		attrs = p.parseAttributes()
		// parseAttributes advances to the declaration token
	}

	switch p.currentToken.Type {
	case CONST:
		// For struct declarations like "const Name struct { ... }",
		// we handle them in parseVarableDeclaration by detecting the STRUCT token
		stmt := p.parseVarableDeclarationOrStruct()
		if stmt != nil && len(attrs) > 0 {
			// Handle attributes for VariableDeclaration, StructDeclaration, and EnumDeclaration
			switch s := stmt.(type) {
			case *VariableDeclaration:
				s.Attributes = attrs
			case *StructDeclaration:
				// Structs don't currently support attributes, but we could add it
			case *EnumDeclaration:
				// Parse enum-specific attributes from generic attribute list
				s.Attributes = p.parseEnumAttributes(attrs)
			}
		}
		return stmt
	case TEMP:
		stmt := p.parseVarableDeclaration()
		if stmt != nil && len(attrs) > 0 {
			stmt.Attributes = attrs
		}
		return stmt
	case DO:
		stmt := p.parseFunctionDeclarationWithAttrs(attrs)
		return stmt
	case RETURN:
		return p.parseReturnStatement()
	case IF:
		return p.parseIfStatement()
	case FOR:
		return p.parseForStatement()
	case FOR_EACH:
		return p.parseForEachStatement()
	case AS_LONG_AS:
		return p.parseWhileStatement()
	case LOOP:
		return p.parseLoopStatement()
	case BREAK:
		return p.parseBreakStatement()
	case CONTINUE:
		return p.parseContinueStatement()
	case IMPORT:
		return p.parseImportStatement()
	case USING:
		return p.parseUsingStatement()
	case IDENT:
		// Parse the expression first to handle identifiers, index expressions, and member access
		expr := p.parseExpression(LOWEST)

		// Check if an assignment operator follows
		if p.peekTokenMatches(ASSIGN) || p.peekTokenMatches(PLUS_ASSIGN) ||
			p.peekTokenMatches(MINUS_ASSIGN) || p.peekTokenMatches(ASTERISK_ASSIGN) ||
			p.peekTokenMatches(SLASH_ASSIGN) || p.peekTokenMatches(PERCENT_ASSIGN) {
			// Convert to assignment statement
			stmt := &AssignmentStatement{Token: p.currentToken}
			stmt.Name = expr

			// Validate assignment target
			switch expr.(type) {
			case *Label, *IndexExpression, *MemberExpression:
				// Valid assignment targets
			default:
				msg := fmt.Sprintf("invalid assignment target: cannot assign to %T", expr)
				p.addEZError(errors.E1008, msg, p.currentToken)
			}

			// Get the assignment operator
			p.nextToken()
			stmt.Operator = p.currentToken.Literal

			// Parse the value expression
			p.nextToken()
			stmt.Value = p.parseExpression(LOWEST)

			return stmt
		}

		// Not an assignment, treat as expression statement
		return &ExpressionStatement{Token: p.currentToken, Expression: expr}
	default:
		return p.parseExpressionStatement()
	}
}

// ============================================================================
// Statement Parsers
// ============================================================================

// parseVarableDeclarationOrStruct handles variable declarations, struct declarations,
// and enum declarations that start with "const Name struct/enum { ... }"
func (p *Parser) parseVarableDeclarationOrStruct() Statement {
	// Check if this is "const Name struct {...}" or "const Name enum {...}"
	if p.currentTokenMatches(CONST) && p.peekTokenMatches(IDENT) {
		// Peek ahead one more token to see if it's STRUCT or ENUM
		savedCurrent := p.currentToken

		p.nextToken() // move to Name
		nameToken := p.currentToken

		if p.peekTokenMatches(STRUCT) {
			// This is a struct declaration
			// currentToken is now the struct name, which is what parseStructDeclaration expects
			return p.parseStructDeclaration()
		}

		if p.peekTokenMatches(ENUM) {
			// This is an enum declaration
			// currentToken is now the enum name, which is what parseEnumDeclaration expects
			return p.parseEnumDeclaration()
		}

		// Not a struct, restore and parse as variable
		// But we've already consumed one token, so we need to account for that
		// The best approach is to just continue with variable parsing from here
		// We're at the name token, so we can build the VariableDeclaration
		stmt := &VariableDeclaration{Token: savedCurrent}
		stmt.Mutable = false // it's a const
		stmt.Names = append(stmt.Names, &Label{Token: nameToken, Value: nameToken.Literal})
		stmt.Name = stmt.Names[0]

		// Now continue with type parsing (the current position is at the name,
		// peekToken should be the type or [)
		// Parse type - can be IDENT or [type] for arrays
		p.nextToken()
		if p.currentTokenMatches(LBRACKET) {
			// Array type - use the array type parsing logic
			typeName := "["
			p.nextToken() // move past [

			// The element type should be an identifier
			if !p.currentTokenMatches(IDENT) {
				msg := fmt.Sprintf("expected type name, got %s", p.currentToken.Type)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E1007, msg, p.currentToken)
				return nil
			}
			typeName += p.currentToken.Literal

			// Check for fixed-size array syntax: [type, size]
			if p.peekTokenMatches(COMMA) {
				p.nextToken() // consume comma
				p.nextToken() // get size

				// Size should be an integer
				if !p.currentTokenMatches(INT) {
					msg := fmt.Sprintf("expected integer for array size, got %s", p.currentToken.Type)
					p.errors = append(p.errors, msg)
					p.addEZError(errors.E1007, msg, p.currentToken)
					return nil
				}
				typeName += "," + p.currentToken.Literal
			}

			if !p.expectPeek(RBRACKET) {
				return nil
			}
			typeName += "]"
			stmt.TypeName = typeName
		} else if p.currentTokenMatches(IDENT) {
			stmt.TypeName = p.currentToken.Literal
		} else {
			msg := fmt.Sprintf("expected type, got %s", p.currentToken.Type)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E1007, msg, p.currentToken)
			return nil
		}

		// Optional initialization
		if p.peekTokenMatches(ASSIGN) {
			assignToken := p.peekToken
			p.nextToken() // consume =
			p.nextToken() // move to value

			if p.currentTokenMatches(RBRACE) || p.currentTokenMatches(EOF) {
				msg := "expected expression after '='"
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E1003, msg, assignToken)
				return nil
			}

			stmt.Value = p.parseExpression(LOWEST)
		} else if !stmt.Mutable {
			// const must be initialized
			msg := fmt.Sprintf("const '%s' must be initialized with a value", stmt.Name.Value)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E1003, msg, p.currentToken)
			return nil
		}

		return stmt
	}

	// Not the special "const Name struct" case, use regular variable declaration
	return p.parseVarableDeclaration()
}

func (p *Parser) parseVarableDeclaration() *VariableDeclaration {
	stmt := &VariableDeclaration{Token: p.currentToken}
	stmt.Mutable = p.currentToken.Type == TEMP

	// Check for @ignore or IDENT
	if p.peekTokenMatches(IGNORE) {
		p.nextToken()
		stmt.Names = append(stmt.Names, &Label{Token: p.currentToken, Value: "@ignore"})
	} else if !p.expectPeek(IDENT) {
		return nil
	} else {
		name := p.currentToken.Literal
		// Check for reserved names
		if isReservedName(name) {
			msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a variable name", name)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E1009, msg, p.currentToken)
			return nil
		}
		// Check for duplicate declaration
		if !p.declareInScope(name, p.currentToken) {
			msg := fmt.Sprintf("'%s' is already declared in this scope", name)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E1010, msg, p.currentToken)
			return nil
		}
		stmt.Names = append(stmt.Names, &Label{Token: p.currentToken, Value: name})
	}

	// Check for multiple assignment: temp result, err = ...
	for p.peekTokenMatches(COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next identifier or @ignore

		if p.currentTokenMatches(IGNORE) {
			stmt.Names = append(stmt.Names, &Label{Token: p.currentToken, Value: "@ignore"})
		} else if p.currentTokenMatches(IDENT) {
			name := p.currentToken.Literal
			// Check for reserved names
			if isReservedName(name) {
				msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a variable name", name)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E1009, msg, p.currentToken)
				return nil
			}
			// Check for duplicate declaration
			if !p.declareInScope(name, p.currentToken) {
				msg := fmt.Sprintf("'%s' is already declared in this scope", name)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E1010, msg, p.currentToken)
				return nil
			}
			stmt.Names = append(stmt.Names, &Label{Token: p.currentToken, Value: name})
		} else {
			msg := fmt.Sprintf("expected identifier or @ignore, got %s", p.currentToken.Type)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E1001, msg, p.currentToken)
			return nil
		}
	}

	// If multiple names, expect = directly (type inferred from function return)
	if len(stmt.Names) > 1 {
		stmt.Name = stmt.Names[0] // keep first name for backward compat
		if !p.expectPeek(ASSIGN) {
			return nil
		}
		p.nextToken() // move to value
		stmt.Value = p.parseExpression(LOWEST)
		return stmt
	}

	// Single variable declaration - set Name for backward compatibility
	stmt.Name = stmt.Names[0]

	// Check if this is a type-inferred assignment (temp x = value)
	if p.peekTokenMatches(ASSIGN) {
		p.nextToken() // consume =
		p.nextToken() // move to value
		stmt.Value = p.parseExpression(LOWEST)
		return stmt
	}

	// Parse type - can be IDENT or [type] for arrays
	p.nextToken()
	stmt.TypeName = p.parseTypeName()
	if stmt.TypeName == "" {
		return nil
	}

	// Optional initialization
	if p.peekTokenMatches(ASSIGN) {
		assignToken := p.peekToken // save the = token for error reporting
		p.nextToken() // consume =
		p.nextToken() // move to value

		// Check if we immediately hit end of block or EOF (incomplete statement)
		if p.currentTokenMatches(RBRACE) || p.currentTokenMatches(EOF) {
			msg := "expected expression after '='"
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E1003, msg, assignToken)
			return nil
		}

		stmt.Value = p.parseExpression(LOWEST)
	} else if !stmt.Mutable {
		// const must be initialized
		msg := fmt.Sprintf("const '%s' must be initialized with a value", stmt.Name.Value)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E1003, msg, p.currentToken)
		return nil
	}

	return stmt
}

func (p *Parser) parseAssignmentStatement() *AssignmentStatement {
	stmt := &AssignmentStatement{Token: p.currentToken}

	// Parse the left-hand side (could be identifier, index, or member)
	stmt.Name = p.parseExpression(LOWEST)

	// Validate that the left-hand side is a valid assignment target
	// Valid targets: Label (identifier), IndexExpression, MemberExpression
	switch stmt.Name.(type) {
	case *Label, *IndexExpression, *MemberExpression:
		// Valid assignment targets
	default:
		// Invalid assignment target
		msg := fmt.Sprintf("invalid assignment target: cannot assign to %T", stmt.Name)
		p.addEZError(errors.E1008, msg, p.currentToken)
	}

	// Get the assignment operator
	p.nextToken()
	stmt.Operator = p.currentToken.Literal

	// Parse the right-hand side
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{Token: p.currentToken}
	stmt.Values = []Expression{}

	p.nextToken()

	// Check for empty return
	if p.currentTokenMatches(RBRACE) || p.currentTokenMatches(EOF) {
		return stmt
	}

	// Parse return values
	stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))

	for p.peekTokenMatches(COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next value
		stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() *ExpressionStatement {
	stmt := &ExpressionStatement{Token: p.currentToken}
	stmt.Expression = p.parseExpression(LOWEST)
	return stmt
}

func (p *Parser) parseBlockStatement() *BlockStatement {
	return p.parseBlockStatementWithSuppress(nil)
}

func (p *Parser) parseBlockStatementWithSuppress(suppressions []*Attribute) *BlockStatement {
	block := &BlockStatement{Token: p.currentToken}
	openBrace := p.currentToken // save the { token for error reporting
	block.Statements = []Statement{}

	// Push a new scope for this block
	p.pushScope()

	p.nextToken() // move past {

	unreachable := false // track if we've hit a terminating statement

	for !p.currentTokenMatches(RBRACE) && !p.currentTokenMatches(EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			// Check for unreachable code
			// Don't warn if the unreachable code is due to an if-statement alternative
			// (otherwise/or) which will be handled by the if statement parser
			if unreachable && !p.peekTokenMatches(OTHERWISE) && !p.peekTokenMatches(OR_KW) {
				// Check if W2001 is suppressed
				if !p.isSuppressed("W2001", suppressions) && !p.isSuppressed("unreachable_code", suppressions) {
					// Warn about unreachable code
					msg := "unreachable code after return/break/continue"
					warn := errors.NewErrorWithSource(
						errors.W2001,
						msg,
						p.filename,
						p.currentToken.Line,
						p.currentToken.Column,
						errors.GetSourceLine(p.source, p.currentToken.Line),
					)
					warn.Help = "remove this code or restructure your control flow"
					p.ezErrors.AddWarning(warn)
				}
			}

			block.Statements = append(block.Statements, stmt)

			// Check if this statement terminates the block
			switch stmt.(type) {
			case *ReturnStatement, *BreakStatement, *ContinueStatement:
				unreachable = true
				// If we're at the closing brace after a terminating statement,
				// don't advance - let the loop exit naturally
				if p.currentTokenMatches(RBRACE) {
					continue // Skip p.nextToken() and re-check loop condition
				}
			}
		}
		p.nextToken()
	}

	// Check if we exited due to EOF (unclosed brace)
	if p.currentTokenMatches(EOF) {
		msg := "unclosed brace - expected '}'"
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E1004, msg, openBrace)
	}

	// Pop the scope when exiting the block
	p.popScope()

	return block
}

// isSuppressed checks if a warning code is suppressed by active suppressions
func (p *Parser) isSuppressed(warningCode string, attrs []*Attribute) bool {
	// First check attrs parameter (for backwards compatibility)
	if attrs != nil {
		for _, attr := range attrs {
			if attr.Name == "suppress" {
				for _, arg := range attr.Args {
					if arg == warningCode {
						return true
					}
				}
			}
		}
	}

	// Then check active suppressions from parser state
	if p.activeSuppressions != nil {
		for _, attr := range p.activeSuppressions {
			if attr.Name == "suppress" {
				for _, arg := range attr.Args {
					if arg == warningCode {
						return true
					}
				}
			}
		}
	}

	return false
}

func (p *Parser) parseIfStatement() *IfStatement {
	stmt := &IfStatement{Token: p.currentToken}

	p.nextToken() // move past 'if'
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Consequence = p.parseBlockStatement()

	// Check for 'or' (else-if) or 'otherwise' (else)
	stmt.Alternative = p.parseAlternative()

	return stmt
}

func (p *Parser) parseAlternative() Statement {
	if p.peekTokenMatches(OR_KW) {
		p.nextToken() // move to 'or'
		altStmt := &IfStatement{Token: p.currentToken}
		p.nextToken() // move past 'or'
		altStmt.Condition = p.parseExpression(LOWEST)
		if !p.expectPeek(LBRACE) {
			return nil
		}
		altStmt.Consequence = p.parseBlockStatement()
		altStmt.Alternative = p.parseAlternative()
		return altStmt
	} else if p.peekTokenMatches(OTHERWISE) {
		p.nextToken() // move to 'otherwise'
		if !p.expectPeek(LBRACE) {
			return nil
		}
		return p.parseBlockStatement()
	}
	return nil
}

func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{Token: p.currentToken}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Variable = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

	// Check for optional type
	if p.peekTokenMatches(IDENT) {
		p.nextToken()
		stmt.VarType = p.currentToken.Literal
	}

	if !p.expectPeek(IN) {
		return nil
	}

	p.nextToken() // move past 'in'
	stmt.Iterable = p.parseExpression(LOWEST)

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseForEachStatement() *ForEachStatement {
	stmt := &ForEachStatement{Token: p.currentToken}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Variable = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

	if !p.expectPeek(IN) {
		return nil
	}

	p.nextToken() // move past 'in'
	stmt.Collection = p.parseExpression(LOWEST)

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseWhileStatement() *WhileStatement {
	stmt := &WhileStatement{Token: p.currentToken}

	p.nextToken() // move past 'as_long_as'
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseLoopStatement() *LoopStatement {
	stmt := &LoopStatement{Token: p.currentToken}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseBreakStatement() *BreakStatement {
	return &BreakStatement{Token: p.currentToken}
}

func (p *Parser) parseContinueStatement() *ContinueStatement {
	return &ContinueStatement{Token: p.currentToken}
}

func (p *Parser) parseFunctionDeclaration() *FunctionDeclaration {
	return p.parseFunctionDeclarationWithAttrs(nil)
}

func (p *Parser) parseFunctionDeclarationWithAttrs(attrs []*Attribute) *FunctionDeclaration {
	stmt := &FunctionDeclaration{Token: p.currentToken, Attributes: attrs}

	// Check if we're inside a function - nested functions are not allowed
	if p.functionDepth > 0 {
		msg := "function declarations are not allowed inside functions"
		p.addEZError(errors.E1011, msg, p.currentToken)
		return nil
	}

	if !p.expectPeek(IDENT) {
		return nil
	}

	name := p.currentToken.Literal
	// Check for reserved names (except 'main' which is special)
	if isReservedName(name) {
		msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a function name", name)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E1009, msg, p.currentToken)
		return nil
	}
	// Check for duplicate declaration
	if !p.declareInScope(name, p.currentToken) {
		msg := fmt.Sprintf("'%s' is already declared in this scope", name)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E1010, msg, p.currentToken)
		return nil
	}

	stmt.Name = &Label{Token: p.currentToken, Value: name}

	if !p.expectPeek(LPAREN) {
		return nil
	}

	stmt.Parameters = p.parseFunctionParameters()

	// Parse return type(s)
	if p.peekTokenMatches(ARROW) {
		arrowToken := p.peekToken // save for error reporting
		p.nextToken() // consume ->
		p.nextToken() // move to return type

		// Check if return type is missing (e.g., "do func() -> {")
		if p.currentTokenMatches(LBRACE) {
			msg := "expected return type after '->'"
			p.addEZError(errors.E1002, msg, arrowToken)
			return nil
		}

		if p.currentTokenMatches(LPAREN) {
			// Multiple return types
			stmt.ReturnTypes = p.parseReturnTypes()
		} else if p.currentTokenMatches(IDENT) {
			// Single return type
			stmt.ReturnTypes = []string{p.currentToken.Literal}
		} else {
			// Invalid token after arrow
			msg := fmt.Sprintf("expected return type after '->', got %s instead", p.currentToken.Type)
			p.addEZError(errors.E1002, msg, arrowToken)
			return nil
		}
	}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	// Enter function scope
	p.functionDepth++

	// Save parent suppressions and add this function's suppressions
	parentSuppressions := p.activeSuppressions
	if stmt.Attributes != nil {
		p.activeSuppressions = append(p.activeSuppressions, stmt.Attributes...)
	}

	// Parse body - suppressions will be inherited from activeSuppressions
	stmt.Body = p.parseBlockStatement()

	// Restore parent suppressions
	p.activeSuppressions = parentSuppressions

	// Exit function scope
	p.functionDepth--

	return stmt
}

func (p *Parser) parseFunctionParameters() []*Parameter {
	params := []*Parameter{}
	paramNames := make(map[string]Token) // track parameter names for duplicate detection

	if p.peekTokenMatches(RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()

	for {
		// Collect parameter names that will share a type
		// e.g., in "x, y int", collect ["x", "y"]
		namesForType := []*Label{}

		// Read first parameter name
		if !p.currentTokenMatches(IDENT) {
			msg := fmt.Sprintf("expected parameter name, got %s", p.currentToken.Type)
			p.addEZError(errors.E1002, msg, p.currentToken)
			return nil
		}

		// Collect all names before the type
		// Strategy: collect IDENT tokens while they're followed by COMMA
		// The last IDENT (not followed by COMMA) is the type
		for {
			if !p.currentTokenMatches(IDENT) {
				msg := fmt.Sprintf("expected parameter name, got %s", p.currentToken.Type)
				p.addEZError(errors.E1002, msg, p.currentToken)
				return nil
			}

			currentIdent := &Label{Token: p.currentToken, Value: p.currentToken.Literal}

			// Look ahead to see what follows this IDENT
			if p.peekTokenMatches(COMMA) {
				// This IDENT is a parameter name (more names or params follow)
				// Check for duplicate
				if prevToken, exists := paramNames[currentIdent.Value]; exists {
					msg := fmt.Sprintf("duplicate parameter name '%s'", currentIdent.Value)
					p.addEZError(errors.E1003, msg, currentIdent.Token)
					helpMsg := fmt.Sprintf("parameter '%s' first declared at line %d", currentIdent.Value, prevToken.Line)
					p.errors = append(p.errors, helpMsg)
				} else {
					paramNames[currentIdent.Value] = currentIdent.Token
				}
				namesForType = append(namesForType, currentIdent)
				p.nextToken() // consume IDENT
				p.nextToken() // consume COMMA, move to next IDENT
				continue
			} else if p.peekTokenMatches(IDENT) || p.peekTokenMatches(LBRACKET) {
				// This IDENT is a parameter name, and the next token is the type
				// Check for duplicate
				if prevToken, exists := paramNames[currentIdent.Value]; exists {
					msg := fmt.Sprintf("duplicate parameter name '%s'", currentIdent.Value)
					p.addEZError(errors.E1003, msg, currentIdent.Token)
					helpMsg := fmt.Sprintf("parameter '%s' first declared at line %d", currentIdent.Value, prevToken.Line)
					p.errors = append(p.errors, helpMsg)
				} else {
					paramNames[currentIdent.Value] = currentIdent.Token
				}
				namesForType = append(namesForType, currentIdent)
				p.nextToken() // move to the type
				break
			} else if p.peekTokenMatches(RPAREN) {
				// Incomplete parameter - name without type before closing paren
				msg := fmt.Sprintf("parameter '%s' is missing a type", currentIdent.Value)
				p.addEZError(errors.E1002, msg, currentIdent.Token)
				return nil
			} else {
				msg := fmt.Sprintf("expected ',', type, or ')' after parameter name, got %s", p.peekToken.Type)
				p.addEZError(errors.E1002, msg, p.peekToken)
				return nil
			}
		}

		// Now current token should be the type for all collected names
		if !p.currentTokenMatches(IDENT) && !p.currentTokenMatches(LBRACKET) {
			msg := fmt.Sprintf("expected parameter type, got %s", p.currentToken.Type)
			p.addEZError(errors.E1002, msg, p.currentToken)
			return nil
		}

		var typeName string
		if p.currentTokenMatches(LBRACKET) {
			// Array type [type]
			typeName = "["
			p.nextToken() // move past [
			typeName += p.currentToken.Literal
			if !p.expectPeek(RBRACKET) {
				return nil
			}
			typeName += "]"
		} else {
			typeName = p.currentToken.Literal
		}

		// Apply the type to all collected names
		for _, name := range namesForType {
			params = append(params, &Parameter{Name: name, TypeName: typeName})
		}

		// Check for comma (more parameters) or closing paren
		if p.peekTokenMatches(COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next parameter name
			continue
		} else if p.peekTokenMatches(RPAREN) {
			p.nextToken() // consume )
			break
		} else {
			msg := fmt.Sprintf("expected ',' or ')', got %s", p.peekToken.Type)
			p.addEZError(errors.E1002, msg, p.peekToken)
			return nil
		}
	}

	return params
}

func (p *Parser) parseReturnTypes() []string {
	types := []string{}

	p.nextToken() // move past (

	for !p.currentTokenMatches(RPAREN) {
		types = append(types, p.currentToken.Literal)
		if p.peekTokenMatches(COMMA) {
			p.nextToken() // consume comma
		}
		p.nextToken()
	}

	return types
}

func (p *Parser) parseImportStatement() *ImportStatement {
	stmt := &ImportStatement{Token: p.currentToken}

	p.nextToken()

	// New syntax: import @module or import alias@module
	if p.currentTokenMatches(AT) {
		// import @module - no alias, use module name
		p.nextToken()
		if p.currentTokenMatches(IDENT) {
			stmt.Module = p.currentToken.Literal
			stmt.Alias = stmt.Module // use module name as alias
		}
	} else if p.currentTokenMatches(IDENT) {
		// import alias@module
		stmt.Alias = p.currentToken.Literal
		p.nextToken()
		if p.currentTokenMatches(AT) {
			p.nextToken()
			if p.currentTokenMatches(IDENT) {
				stmt.Module = p.currentToken.Literal
			}
		}
	}

	return stmt
}

func (p *Parser) parseUsingStatement() *UsingStatement {
	stmt := &UsingStatement{Token: p.currentToken}
	stmt.Modules = []*Label{}

	// Parse first module
	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Modules = append(stmt.Modules, &Label{Token: p.currentToken, Value: p.currentToken.Literal})

	// Check for comma-separated modules
	for p.peekTokenMatches(COMMA) {
		p.nextToken() // move to comma
		if !p.expectPeek(IDENT) {
			return nil
		}
		stmt.Modules = append(stmt.Modules, &Label{Token: p.currentToken, Value: p.currentToken.Literal})
	}

	return stmt
}

func (p *Parser) parseEnumDeclaration() *EnumDeclaration {
	stmt := &EnumDeclaration{Token: p.currentToken}
	stmt.Name = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

	p.nextToken() // move past name to 'enum'
	p.nextToken() // move past 'enum' to '{'

	if !p.currentTokenMatches(LBRACE) {
		msg := "expected '{' after enum name"
		p.addEZError(errors.E1002, msg, p.currentToken)
		return nil
	}

	stmt.Values = []*EnumValue{}
	valueNames := make(map[string]Token) // track value names for duplicate detection

	p.nextToken() // move into enum body

	for !p.currentTokenMatches(RBRACE) && !p.currentTokenMatches(EOF) {
		if !p.currentTokenMatches(IDENT) {
			msg := fmt.Sprintf("expected enum value name, got %s", p.currentToken.Type)
			p.addEZError(errors.E1002, msg, p.currentToken)
			return nil
		}

		enumValue := &EnumValue{}
		enumValue.Name = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

		// Check for duplicate value names
		if prevToken, exists := valueNames[enumValue.Name.Value]; exists {
			msg := fmt.Sprintf("duplicate value name '%s' in enum '%s'", enumValue.Name.Value, stmt.Name.Value)
			p.addEZError(errors.E1003, msg, enumValue.Name.Token)
			helpMsg := fmt.Sprintf("value '%s' first declared at line %d", enumValue.Name.Value, prevToken.Line)
			p.errors = append(p.errors, helpMsg)
		} else {
			valueNames[enumValue.Name.Value] = enumValue.Name.Token
		}

		// Check for explicit value assignment: VALUE = expression
		if p.peekTokenMatches(ASSIGN) {
			p.nextToken() // move to =
			p.nextToken() // move to value expression
			enumValue.Value = p.parseExpression(LOWEST)
		}

		stmt.Values = append(stmt.Values, enumValue)
		p.nextToken() // move to next value or }
	}

	if len(stmt.Values) == 0 {
		msg := fmt.Sprintf("enum '%s' must have at least one value", stmt.Name.Value)
		p.addEZError(errors.E1003, msg, stmt.Token)
	}

	return stmt
}

// parseTypeName parses a type name which can be:
// - Simple type: int, string, StructName
// - Array type: [int], [string], [StructName]
// - Fixed-size array: [int,5], [string,10]
func (p *Parser) parseTypeName() string {
	if p.currentTokenMatches(LBRACKET) {
		// Array type [type] or [type, size]
		typeName := "["
		p.nextToken() // move past [

		// The element type should be an identifier
		if !p.currentTokenMatches(IDENT) {
			msg := fmt.Sprintf("expected type name, got %s", p.currentToken.Type)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E1007, msg, p.currentToken)
			return ""
		}
		typeName += p.currentToken.Literal

		// Check for fixed-size array syntax: [type, size]
		if p.peekTokenMatches(COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // get size

			// Size should be an integer
			if !p.currentTokenMatches(INT) {
				msg := fmt.Sprintf("expected integer for array size, got %s", p.currentToken.Type)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E1007, msg, p.currentToken)
				return ""
			}
			typeName += "," + p.currentToken.Literal
		}

		if !p.expectPeek(RBRACKET) {
			return ""
		}
		typeName += "]"
		return typeName
	} else if p.currentTokenMatches(IDENT) {
		return p.currentToken.Literal
	} else {
		msg := fmt.Sprintf("expected type, got %s", p.currentToken.Type)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E1007, msg, p.currentToken)
		return ""
	}
}

func (p *Parser) parseStructDeclaration() *StructDeclaration {
	stmt := &StructDeclaration{Token: p.currentToken}

	stmt.Name = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

	p.nextToken() // move past name to 'struct'
	p.nextToken() // move past 'struct' to '{'

	if !p.currentTokenMatches(LBRACE) {
		return nil
	}

	stmt.Fields = []*StructField{}
	fieldNames := make(map[string]Token) // track field names for duplicate detection

	p.nextToken() // move into struct body

	for !p.currentTokenMatches(RBRACE) && !p.currentTokenMatches(EOF) {
		field := &StructField{}
		field.Name = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

		// Check for duplicate field names
		if prevToken, exists := fieldNames[field.Name.Value]; exists {
			msg := fmt.Sprintf("duplicate field name '%s' in struct '%s'", field.Name.Value, stmt.Name.Value)
			p.addEZError(errors.E1003, msg, field.Name.Token)
			// Add help message pointing to first declaration
			helpMsg := fmt.Sprintf("field '%s' first declared at line %d", field.Name.Value, prevToken.Line)
			p.errors = append(p.errors, helpMsg)
		} else {
			fieldNames[field.Name.Value] = field.Name.Token
		}

		// Move to type token (can be IDENT or LBRACKET for arrays)
		p.nextToken()

		// Parse the type name (handles both simple types and array types)
		field.TypeName = p.parseTypeName()
		if field.TypeName == "" {
			return nil
		}

		stmt.Fields = append(stmt.Fields, field)
		p.nextToken()
	}

	return stmt
}

// ============================================================================
// Expression Parsers
// ============================================================================

func (p *Parser) parseExpression(precedence int) Expression {
	prefix := p.prefixParseFns[p.currentToken.Type]
	if prefix == nil {
		var msg string
		var code errors.ErrorCode

		// Provide more specific error messages
		switch p.currentToken.Type {
		case RBRACE, EOF:
			msg = "expected expression, found end of block"
			code = errors.E1003
		case RPAREN:
			msg = "expected expression, found ')'"
			code = errors.E1003
		case RBRACKET:
			msg = "expected expression, found ']'"
			code = errors.E1003
		default:
			msg = fmt.Sprintf("unexpected token '%s'", p.currentToken.Literal)
			code = errors.E1001
		}

		p.errors = append(p.errors, msg)
		p.addEZError(code, msg, p.currentToken)
		return nil
	}
	leftExp := prefix()

	for precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() Expression {
	ident := &Label{Token: p.currentToken, Value: p.currentToken.Literal}

	// Check if this is a struct literal: TypeName{...}
	// Only treat it as struct literal if identifier starts with uppercase (type naming convention)
	if p.peekTokenMatches(LBRACE) && len(ident.Value) > 0 && ident.Value[0] >= 'A' && ident.Value[0] <= 'Z' {
		p.nextToken() // move to {
		return p.parseStructLiteralFromIdent(ident)
	}

	return ident
}

// parseStructLiteralFromIdent parses a struct literal when we've already identified the type name
func (p *Parser) parseStructLiteralFromIdent(name *Label) Expression {
	lit := &StructValue{Token: p.currentToken, Name: name}
	lit.Fields = make(map[string]Expression)

	// Already at {, move into struct
	p.nextToken()

	for !p.currentTokenMatches(RBRACE) && !p.currentTokenMatches(EOF) {
		// Handle empty struct literal
		if p.currentTokenMatches(RBRACE) {
			break
		}

		fieldName := p.currentToken.Literal

		if !p.expectPeek(COLON) {
			return nil
		}

		p.nextToken() // move past :
		lit.Fields[fieldName] = p.parseExpression(LOWEST)

		if p.peekTokenMatches(COMMA) {
			p.nextToken() // consume comma
		}
		p.nextToken()
	}

	return lit
}


func (p *Parser) parseStructValue(name *Label) Expression {
	lit := &StructValue{Token: p.currentToken, Name: name}
	lit.Fields = make(map[string]Expression)

	p.nextToken() // move to {
	p.nextToken() // move into struct

	for !p.currentTokenMatches(RBRACE) && !p.currentTokenMatches(EOF) {
		fieldName := p.currentToken.Literal

		if !p.expectPeek(COLON) {
			return nil
		}

		p.nextToken() // move past :
		lit.Fields[fieldName] = p.parseExpression(LOWEST)

		if p.peekTokenMatches(COMMA) {
			p.nextToken() // consume comma
		}
		p.nextToken()
	}

	return lit
}

func (p *Parser) parseIntegerValue() Expression {
	lit := &IntegerValue{Token: p.currentToken}

	// Strip underscores for numeric conversion (they're only for readability)
	cleanedLiteral := stripUnderscores(p.currentToken.Literal)

	value, err := strconv.ParseInt(cleanedLiteral, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.currentToken.Literal)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E1006, msg, p.currentToken)
		return nil
	}

	lit.Value = value
	return lit
}

// stripUnderscores removes all underscores from a numeric literal
func stripUnderscores(s string) string {
	if !strings.Contains(s, "_") {
		return s
	}
	var result strings.Builder
	for _, ch := range s {
		if ch != '_' {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

func (p *Parser) parseFloatValue() Expression {
	lit := &FloatValue{Token: p.currentToken}

	// Strip underscores for numeric conversion (they're only for readability)
	cleanedLiteral := stripUnderscores(p.currentToken.Literal)

	value, err := strconv.ParseFloat(cleanedLiteral, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as float", p.currentToken.Literal)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E1006, msg, p.currentToken)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseStringValue() Expression {
	token := p.currentToken
	literal := p.currentToken.Literal

	// Check if string contains interpolation patterns ${...}
	if !containsInterpolation(literal) {
		return &StringValue{Token: token, Value: literal}
	}

	// Parse interpolated string
	parts := make([]Expression, 0)
	i := 0
	currentLiteral := ""

	for i < len(literal) {
		// Look for ${
		if i < len(literal)-1 && literal[i] == '$' && literal[i+1] == '{' {
			// Add any accumulated literal before this interpolation
			if currentLiteral != "" {
				parts = append(parts, &StringValue{
					Token: token,
					Value: currentLiteral,
				})
				currentLiteral = ""
			}

			// Find matching closing brace
			i += 2 // skip ${
			braceCount := 1
			exprStart := i

			for i < len(literal) && braceCount > 0 {
				if literal[i] == '{' {
					braceCount++
				} else if literal[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount != 0 {
				// Unclosed interpolation
				p.addEZError(errors.E1005, "unclosed interpolation in string", token)
				return &StringValue{Token: token, Value: literal}
			}

			// Parse the expression inside ${}
			exprStr := literal[exprStart:i]
			if exprStr == "" {
				p.addEZError(errors.E1002, "empty interpolation in string", token)
			} else {
				// Create a mini-lexer and parser for the expression
				expr := p.parseInterpolatedExpression(exprStr, token)
				if expr != nil {
					parts = append(parts, expr)
				}
			}

			i++ // skip the closing }
		} else {
			currentLiteral += string(literal[i])
			i++
		}
	}

	// Add any remaining literal
	if currentLiteral != "" {
		parts = append(parts, &StringValue{
			Token: token,
			Value: currentLiteral,
		})
	}

	return &InterpolatedString{
		Token: token,
		Parts: parts,
	}
}

// containsInterpolation checks if a string contains ${} patterns
func containsInterpolation(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '$' && s[i+1] == '{' {
			return true
		}
	}
	return false
}

// parseInterpolatedExpression parses an expression from within ${}
func (p *Parser) parseInterpolatedExpression(exprStr string, origToken Token) Expression {
	// Create a new lexer for the expression
	lexer := NewLexer(exprStr)

	// Create a temporary parser
	tempParser := &Parser{
		l:              lexer,
		prefixParseFns: make(map[TokenType]prefixParseFn),
		infixParseFns:  make(map[TokenType]infixParseFn),
		ezErrors:       p.ezErrors,
		source:         exprStr,
		filename:       p.filename,
	}

	// Register the same parse functions
	tempParser.registerParseFunctions()

	// Initialize tokens
	tempParser.nextToken()
	tempParser.nextToken()

	// Parse the expression
	expr := tempParser.parseExpression(LOWEST)

	// Check for errors
	if len(tempParser.errors) > 0 {
		for _, err := range tempParser.errors {
			p.errors = append(p.errors, err)
		}
		return nil
	}

	return expr
}

// registerParseFunctions sets up all prefix and infix parse functions
func (p *Parser) registerParseFunctions() {
	// Prefix parse functions
	p.setPrefix(IDENT, p.parseIdentifier)
	p.setPrefix(INT, p.parseIntegerValue)
	p.setPrefix(FLOAT, p.parseFloatValue)
	p.setPrefix(STRING, p.parseStringValue)
	p.setPrefix(CHAR, p.parseCharValue)
	p.setPrefix(TRUE, p.parseBooleanValue)
	p.setPrefix(FALSE, p.parseBooleanValue)
	p.setPrefix(NIL, p.parseNilValue)
	p.setPrefix(BANG, p.parsePrefixExpression)
	p.setPrefix(MINUS, p.parsePrefixExpression)
	p.setPrefix(LPAREN, p.parseGroupedExpression)
	p.setPrefix(LBRACE, p.parseArrayValue)
	p.setPrefix(NEW, p.parseNewExpression)
	p.setPrefix(RANGE, p.parseRangeExpression)

	// Infix parse functions
	p.setInfix(PLUS, p.parseInfixExpression)
	p.setInfix(MINUS, p.parseInfixExpression)
	p.setInfix(ASTERISK, p.parseInfixExpression)
	p.setInfix(SLASH, p.parseInfixExpression)
	p.setInfix(PERCENT, p.parseInfixExpression)
	p.setInfix(EQ, p.parseInfixExpression)
	p.setInfix(NOT_EQ, p.parseInfixExpression)
	p.setInfix(LT, p.parseInfixExpression)
	p.setInfix(GT, p.parseInfixExpression)
	p.setInfix(LT_EQ, p.parseInfixExpression)
	p.setInfix(GT_EQ, p.parseInfixExpression)
	p.setInfix(AND, p.parseInfixExpression)
	p.setInfix(OR, p.parseInfixExpression)
	p.setInfix(IN, p.parseInfixExpression)
	p.setInfix(NOT_IN, p.parseInfixExpression)
	p.setInfix(LPAREN, p.parseCallExpression)
	p.setInfix(LBRACKET, p.parseIndexExpression)
	p.setInfix(DOT, p.parseMemberExpression)
	p.setInfix(INCREMENT, p.parsePostfixExpression)
	p.setInfix(DECREMENT, p.parsePostfixExpression)
}

func (p *Parser) parseCharValue() Expression {
	value := []rune(p.currentToken.Literal)
	if len(value) == 0 {
		return &CharValue{Token: p.currentToken, Value: 0}
	}
	return &CharValue{Token: p.currentToken, Value: value[0]}
}

func (p *Parser) parseBooleanValue() Expression {
	return &BooleanValue{Token: p.currentToken, Value: p.currentTokenMatches(TRUE)}
}

func (p *Parser) parseNilValue() Expression {
	return &NilValue{Token: p.currentToken}
}

func (p *Parser) parseArrayValue() Expression {
	lit := &ArrayValue{Token: p.currentToken}
	lit.Elements = p.parseExpressionList(RBRACE)
	return lit
}

func (p *Parser) parseExpressionList(end TokenType) []Expression {
	list := []Expression{}

	if p.peekTokenMatches(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenMatches(COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next expression
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parsePrefixExpression() Expression {
	expression := &PrefixExpression{
		Token:    p.currentToken,
		Operator: p.currentToken.Literal,
	}

	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseInfixExpression(left Expression) Expression {
	expression := &InfixExpression{
		Token:    p.currentToken,
		Operator: p.currentToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parsePostfixExpression(left Expression) Expression {
	return &PostfixExpression{
		Token:    p.currentToken,
		Operator: p.currentToken.Literal,
		Left:     left,
	}
}

func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseCallExpression(function Expression) Expression {
	exp := &CallExpression{Token: p.currentToken, Function: function}
	exp.Arguments = p.parseExpressionList(RPAREN)
	return exp
}

func (p *Parser) parseIndexExpression(left Expression) Expression {
	exp := &IndexExpression{Token: p.currentToken, Left: left}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(RBRACKET) {
		return nil
	}

	return exp
}

func (p *Parser) parseMemberExpression(left Expression) Expression {
	exp := &MemberExpression{Token: p.currentToken, Object: left}

	p.nextToken()
	exp.Member = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

	return exp
}

func (p *Parser) parseNewExpression() Expression {
	exp := &NewExpression{Token: p.currentToken}

	if !p.expectPeek(LPAREN) {
		return nil
	}

	if !p.expectPeek(IDENT) {
		return nil
	}

	exp.TypeName = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

	if !p.expectPeek(RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseRangeExpression() Expression {
	exp := &RangeExpression{Token: p.currentToken}

	if !p.expectPeek(LPAREN) {
		return nil
	}

	p.nextToken()
	exp.Start = p.parseExpression(LOWEST)

	if !p.expectPeek(COMMA) {
		return nil
	}

	p.nextToken()
	exp.End = p.parseExpression(LOWEST)

	if !p.expectPeek(RPAREN) {
		return nil
	}

	return exp
}

// ============================================================================
// Attribute Parsing
// ============================================================================

// parseAttributes parses @suppress(...) and @(...) attributes before declarations
func (p *Parser) parseAttributes() []*Attribute {
	attributes := []*Attribute{}

	// Handle both @suppress(...) and @(...)
	for p.currentTokenMatches(SUPPRESS) || p.currentTokenMatches(AT) {
		if p.currentTokenMatches(SUPPRESS) {
			// Handle @suppress(...) - existing code
			attributes = append(attributes, p.parseSuppressAttribute())
			continue
		}

		// Handle @(...) for enum attributes
		if p.currentTokenMatches(AT) {
			attributes = append(attributes, p.parseGenericAttribute())
			continue
		}
	}

	return attributes
}

func (p *Parser) parseSuppressAttribute() *Attribute {
	attr := &Attribute{
		Token: p.currentToken,
		Name:  "suppress",
		Args:  []string{},
	}

	// Expect opening paren
	if !p.expectPeek(LPAREN) {
		return attr
	}

	// Parse comma-separated warning codes
	if !p.peekTokenMatches(RPAREN) {
		p.nextToken() // move to first identifier

		// First argument
		if p.currentTokenMatches(IDENT) || p.currentTokenMatches(STRING) {
			attr.Args = append(attr.Args, p.currentToken.Literal)
		}

		// Additional arguments
		for p.peekTokenMatches(COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next argument

			if p.currentTokenMatches(IDENT) || p.currentTokenMatches(STRING) {
				attr.Args = append(attr.Args, p.currentToken.Literal)
			}
		}
	}

	// Expect closing paren
	if !p.expectPeek(RPAREN) {
		return attr
	}

	// Move to next token (might be another attribute or the declaration)
	p.nextToken()

	return attr
}

func (p *Parser) parseGenericAttribute() *Attribute {
	attr := &Attribute{
		Token: p.currentToken,
		Name:  "enum_config",
		Args:  []string{},
	}

	// Expect opening paren
	if !p.expectPeek(LPAREN) {
		return attr
	}

	// Parse comma-separated arguments (type, "skip" keyword, increment value)
	if !p.peekTokenMatches(RPAREN) {
		p.nextToken() // move to first argument

		// Collect all arguments
		for {
			if p.currentTokenMatches(IDENT) {
				attr.Args = append(attr.Args, p.currentToken.Literal)
			} else if p.currentTokenMatches(INT) {
				attr.Args = append(attr.Args, p.currentToken.Literal)
			} else if p.currentTokenMatches(FLOAT) {
				attr.Args = append(attr.Args, p.currentToken.Literal)
			} else if p.currentTokenMatches(STRING) {
				attr.Args = append(attr.Args, p.currentToken.Literal)
			}

			if !p.peekTokenMatches(COMMA) {
				break
			}

			p.nextToken() // consume comma
			p.nextToken() // move to next argument
		}
	}

	// Expect closing paren
	if !p.expectPeek(RPAREN) {
		return attr
	}

	// Move to next token
	p.nextToken()

	return attr
}

// parseEnumAttributes converts generic attributes to EnumAttributes
func (p *Parser) parseEnumAttributes(attrs []*Attribute) *EnumAttributes {
	enumAttrs := &EnumAttributes{
		TypeName: "int", // default
		Skip:     false,
		Increment: nil,
	}

	for _, attr := range attrs {
		if attr.Name == "enum_config" && len(attr.Args) > 0 {
			// First arg is the type
			enumAttrs.TypeName = attr.Args[0]

			// Check for "skip" keyword
			for i, arg := range attr.Args {
				if arg == "skip" {
					enumAttrs.Skip = true
					// Next arg (if exists) is the increment value
					if i+1 < len(attr.Args) {
						// Parse the increment value
						incrementStr := attr.Args[i+1]
						// Try to parse as int64
						if val, err := strconv.ParseInt(incrementStr, 10, 64); err == nil {
							enumAttrs.Increment = &IntegerValue{
								Token: Token{Type: INT, Literal: incrementStr},
								Value: val,
							}
						} else {
							// Try to parse as float64
							if fval, err := strconv.ParseFloat(incrementStr, 64); err == nil {
								enumAttrs.Increment = &FloatValue{
									Token: Token{Type: FLOAT, Literal: incrementStr},
									Value: fval,
								}
							}
						}
					}
					break
				}
			}
		}
	}

	return enumAttrs
}

