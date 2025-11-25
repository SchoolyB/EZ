package parser

import (
	"fmt"
	"strconv"

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
	peekToken Token

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
	program.Statements = []Statement{}

	for !p.currentTokenMatches(EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() Statement {
	// Check for @suppress attribute
	var attrs []*Attribute
	if p.currentTokenMatches(SUPPRESS) {
		attrs = p.parseAttributes()
		// parseAttributes advances to the declaration token
	}

	switch p.currentToken.Type {
	case CONST:
		// Check if this is a struct or enum declaration
		if p.peekTokenMatches(IDENT) {
			// Save current position
			savedCurrent := p.currentToken
			savedPeek := p.peekToken

			p.nextToken() // move to IDENT

			if p.peekTokenMatches(STRUCT) {
				// This is a struct declaration: const Name struct { ... }
				return p.parseStructDeclaration()
			} else if p.peekTokenMatches(ENUM) {
				// This is an enum declaration: const Name enum { ... }
				// TODO: Implement parseEnumDeclaration()
				// return p.parseEnumDeclaration()
			}

			// Not a struct or enum, restore and parse as variable
			p.currentToken = savedCurrent
			p.peekToken = savedPeek
		}
		stmt := p.parseVarableDeclaration()
		if stmt != nil && len(attrs) > 0 {
			stmt.Attributes = attrs
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
	if p.currentTokenMatches(LBRACKET) {
		// Array type [type] or [type, size]
		typeName := "["
		p.nextToken() // move past [
		typeName += p.currentToken.Literal
		if p.peekTokenMatches(COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // get size
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

	p.nextToken() // move past {

	unreachable := false // track if we've hit a terminating statement

	for !p.currentTokenMatches(RBRACE) && !p.currentTokenMatches(EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			// Check for unreachable code
			if unreachable {
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

	return block
}

// isSuppressed checks if a warning code is suppressed by the given attributes
func (p *Parser) isSuppressed(warningCode string, attrs []*Attribute) bool {
	if attrs == nil {
		return false
	}
	for _, attr := range attrs {
		if attr.Name == "suppress" {
			for _, arg := range attr.Args {
				if arg == warningCode {
					return true
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
		p.nextToken() // consume ->
		p.nextToken() // move to return type

		if p.currentTokenMatches(LPAREN) {
			// Multiple return types
			stmt.ReturnTypes = p.parseReturnTypes()
		} else {
			// Single return type
			stmt.ReturnTypes = []string{p.currentToken.Literal}
		}
	}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	// Enter function scope
	p.functionDepth++

	// Parse body with attributes for suppression
	stmt.Body = p.parseBlockStatementWithSuppress(stmt.Attributes)

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
		// Read parameter name
		if !p.currentTokenMatches(IDENT) {
			msg := fmt.Sprintf("expected parameter name, got %s", p.currentToken.Type)
			p.addEZError(errors.E1002, msg, p.currentToken)
			return nil
		}

		paramName := &Label{Token: p.currentToken, Value: p.currentToken.Literal}

		// Check for duplicate parameter names
		if prevToken, exists := paramNames[paramName.Value]; exists {
			msg := fmt.Sprintf("duplicate parameter name '%s'", paramName.Value)
			p.addEZError(errors.E1003, msg, paramName.Token)
			helpMsg := fmt.Sprintf("parameter '%s' first declared at line %d", paramName.Value, prevToken.Line)
			p.errors = append(p.errors, helpMsg)
		} else {
			paramNames[paramName.Value] = paramName.Token
		}

		// Read parameter type
		p.nextToken()
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

		params = append(params, &Parameter{Name: paramName, TypeName: typeName})

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

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Module = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

	return stmt
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

		if !p.expectPeek(IDENT) {
			return nil
		}
		field.TypeName = p.currentToken.Literal

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

	value, err := strconv.ParseInt(p.currentToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.currentToken.Literal)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E1006, msg, p.currentToken)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseFloatValue() Expression {
	lit := &FloatValue{Token: p.currentToken}

	value, err := strconv.ParseFloat(p.currentToken.Literal, 64)
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
	return &StringValue{Token: p.currentToken, Value: p.currentToken.Literal}
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

// parseAttributes parses @suppress(...) attributes before declarations
func (p *Parser) parseAttributes() []*Attribute {
	attributes := []*Attribute{}

	for p.currentTokenMatches(SUPPRESS) {
		attr := &Attribute{
			Token: p.currentToken,
			Name:  "suppress",
			Args:  []string{},
		}

		// Expect opening paren
		if !p.expectPeek(LPAREN) {
			return attributes
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
			return attributes
		}

		attributes = append(attributes, attr)

		// Move to next token (might be another @suppress or the declaration)
		p.nextToken()
	}

	return attributes
}
