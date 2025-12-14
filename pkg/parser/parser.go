package parser

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"math/big"
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
	"module": true, "private": true, "from": true, "use": true,
}

// Builtin function and type names that cannot be redefined
var builtinNames = map[string]bool{
	// Builtin functions
	"len": true, "typeof": true, "input": true,
	"println": true, "print": true, "read_int": true,
	"copy": true, "append": true, "error": true,
	// Primitive type names
	"int": true, "float": true, "string": true, "bool": true, "char": true, "byte": true,
	// Sized integers
	"i8": true, "i16": true, "i32": true, "i64": true, "i128": true, "i256": true,
	"u8": true, "u16": true, "u32": true, "u64": true, "u128": true, "u256": true, "uint": true,
	// Sized floats
	"f32": true, "f64": true,
	// Collection types
	"map": true,
}

// isReservedName checks if a name is a reserved keyword or builtin
func isReservedName(name string) bool {
	return reservedKeywords[name] || builtinNames[name]
}

// allWarningCodes contains all valid warning codes in the language
var allWarningCodes = map[string]bool{
	// Code Style Warnings (W1xxx)
	"W1001": true, // unused-variable
	"W1002": true, // unused-import
	"W1003": true, // unused-function
	"W1004": true, // unused-parameter
	// Potential Bug Warnings (W2xxx)
	"W2001": true, // unreachable-code
	"W2002": true, // shadowed-variable
	"W2003": true, // missing-return
	"W2004": true, // implicit-type-conversion
	"W2005": true, // deprecated-feature
	"W2006": true, // byte-overflow-potential
	// Code Quality Warnings (W3xxx)
	"W3001": true, // empty-block
	"W3002": true, // redundant-condition
	"W3003": true, // array-size-mismatch
	// Module Warnings (W4xxx)
	"W4001": true, // module-name-mismatch
}

// suppressibleWarnings contains warning codes that can be suppressed via @suppress
// These are function-body warnings that make sense to suppress at the function level
var suppressibleWarnings = map[string]bool{
	"W1001": true, // unused-variable
	"W1004": true, // unused-parameter
	"W2001": true, // unreachable-code
	"W2002": true, // shadowed-variable
	"W2003": true, // missing-return
	"W2004": true, // implicit-type-conversion
	"W2005": true, // deprecated-feature
	"W2006": true, // byte-overflow-potential
	"W3001": true, // empty-block
	"W3002": true, // redundant-condition
	"W3003": true, // array-size-mismatch
}

// isValidWarningCode checks if a warning code exists
func isValidWarningCode(code string) bool {
	return allWarningCodes[code]
}

// isWarningSuppressible checks if a warning code can be suppressed
func isWarningSuppressible(code string) bool {
	return suppressibleWarnings[code]
}

// hasSuppressAttribute checks if attributes contain a @suppress attribute
func hasSuppressAttribute(attrs []*Attribute) *Attribute {
	for _, attr := range attrs {
		if attr.Name == "suppress" {
			return attr
		}
	}
	return nil
}

// Operator precedence levels
const (
	_ int = iota
	LOWEST
	OR_PREC     // ||
	AND_PREC    // &&
	EQUALS      // ==, !=
	LESSGREATER // <, >, <=, >=
	MEMBERSHIP  // in, not_in, !in
	SUM         // +, -
	PRODUCT     // *, /, %
	PREFIX      // -X, !X
	POSTFIX     // x++, x--
	CALL        // function(x)
	INDEX       // array[index]
	MEMBER      // object.member
)

// Setting the order in which tokens should be evaluated
var priorities = map[TokenType]int{
	OR:        OR_PREC,
	AND:       AND_PREC,
	EQ:        EQUALS,
	NOT_EQ:    EQUALS,
	LT:        LESSGREATER,
	GT:        LESSGREATER,
	LT_EQ:     LESSGREATER,
	GT_EQ:     LESSGREATER,
	IN:        MEMBERSHIP,
	NOT_IN:    MEMBERSHIP,
	PLUS:      SUM,
	MINUS:     SUM,
	ASTERISK:  PRODUCT,
	SLASH:     PRODUCT,
	PERCENT:   PRODUCT,
	INCREMENT: POSTFIX,
	DECREMENT: POSTFIX,
	LPAREN:    CALL,
	LBRACKET:  INDEX,
	DOT:       MEMBER,
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

	currentToken Token
	peekToken    Token

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

// addWarning adds a deprecation or other warning to the warning list
func (p *Parser) addWarning(code string, message string, tok Token) {
	sourceLine := ""
	if p.source != "" {
		sourceLine = errors.GetSourceLine(p.source, tok.Line)
	}

	warn := errors.NewErrorWithSource(
		errors.W2005, // deprecated-feature
		message,
		p.filename,
		tok.Line,
		tok.Column,
		sourceLine,
	)
	warn.EndColumn = tok.Column + len(tok.Literal)
	p.ezErrors.AddWarning(warn)
}

func (p *Parser) nextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.l.NextToken()
	// Set file info on tokens for multi-file module support
	if p.filename != "" && p.filename != "<unknown>" {
		p.currentToken.File = p.filename
		p.peekToken.File = p.filename
	}
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
			code = errors.E2005
		case RBRACE:
			msg = "unclosed brace - expected '}'"
			code = errors.E2004
		case RBRACKET:
			msg = "unclosed bracket - expected ']'"
			code = errors.E2006
		default:
			msg = fmt.Sprintf("unexpected end of file, expected %s", tType)
			code = errors.E2002
		}
	} else {
		msg = fmt.Sprintf("expected %s, got %s instead", tType, p.peekToken.Type)
		code = errors.E2002
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
	seenModuleDecl := false
	importedModules := make(map[string]bool) // Track imported modules

	// Check for module declaration first (must be first non-comment token)
	if p.currentTokenMatches(MODULE) {
		program.Module = p.parseModuleDeclaration()
		seenModuleDecl = true
		p.nextToken()
	}

	for !p.currentTokenMatches(EOF) {
		// Module declaration must come first if present
		if p.currentTokenMatches(MODULE) {
			if seenModuleDecl {
				p.addEZError(errors.E2002, "duplicate module declaration", p.currentToken)
			} else if seenOtherDeclaration {
				p.addEZError(errors.E2002, "module declaration must be the first statement in the file", p.currentToken)
			}
			p.nextToken()
			continue
		}

		// Track what we've seen for placement validation
		if p.currentTokenMatches(USING) {
			// File-scoped using: must come before other declarations
			if seenOtherDeclaration {
				p.addEZError(errors.E2009, "file-scoped 'using' must come before all declarations", p.currentToken)
				p.nextToken()
				continue
			}

			// Parse and add to FileUsing
			usingStmt := p.parseUsingStatement()
			if usingStmt != nil {
				// Validate that all modules in the using statement have been imported
				for _, module := range usingStmt.Modules {
					if !importedModules[module.Value] {
						msg := fmt.Sprintf("cannot use module '%s' before importing it", module.Value)
						p.addEZError(errors.E2010, msg, usingStmt.Token)
					}
				}
				program.FileUsing = append(program.FileUsing, usingStmt)
			}
			p.nextToken()
			continue
		} else if !p.currentTokenMatches(EOF) && !p.currentTokenMatches(IMPORT) {
			seenOtherDeclaration = true
		}

		// Check for imports after other declarations (functions, types, etc.)
		if p.currentTokenMatches(IMPORT) && seenOtherDeclaration {
			p.addEZError(errors.E2036, "import statements must appear at the top of the file, before any declarations", p.currentToken)
		}

		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)

			// Track imported modules by both alias and module name
			if importStmt, ok := stmt.(*ImportStatement); ok {
				// Handle multiple imports (new comma-separated syntax)
				if len(importStmt.Imports) > 0 {
					for _, item := range importStmt.Imports {
						// Track the alias so "using arr" works with "import arr@arrays"
						importedModules[item.Alias] = true
						// Also track the module name so "using arrays" still works
						importedModules[item.Module] = true
					}
				} else {
					// Backward compatibility: track single import
					importedModules[importStmt.Alias] = true
					importedModules[importStmt.Module] = true
				}
			}
		}
		p.nextToken()
	}

	return program
}

// ParseLine parses a single line for REPL mode
// Returns a statement node or nil if the line is empty/comment-only
func (p *Parser) ParseLine() Statement {
	// Skip if we're at EOF
	if p.currentTokenMatches(EOF) {
		return nil
	}

	stmt := p.parseStatement()
	return stmt
}

func (p *Parser) parseStatement() Statement {
	// Check for @suppress, @strict, @enum(...), @flags, or unknown @ attributes
	var attrs []*Attribute
	if p.currentTokenMatches(SUPPRESS) || p.currentTokenMatches(STRICT) || p.currentTokenMatches(ENUM_ATTR) || p.currentTokenMatches(FLAGS) || p.currentTokenMatches(AT) {
		attrs = p.parseAttributes()
		// parseAttributes advances to the declaration token
	}

	// Check for private modifier
	visibility := VisibilityPublic
	if p.currentTokenMatches(PRIVATE) {
		visibility = VisibilityPrivate
		// Check for private:module syntax
		if p.peekTokenMatches(COLON) {
			p.nextToken() // consume PRIVATE
			p.nextToken() // consume COLON
			if p.currentTokenMatches(IDENT) && p.currentToken.Literal == "module" {
				visibility = VisibilityPrivateModule
				p.nextToken() // move past "module"
			} else {
				p.addEZError(errors.E2002, "expected 'module' after 'private:'", p.currentToken)
			}
		} else {
			p.nextToken() // move past PRIVATE to the declaration
		}
	}

	// Validate @suppress is only used on function declarations
	if suppressAttr := hasSuppressAttribute(attrs); suppressAttr != nil {
		if !p.currentTokenMatches(DO) {
			p.addEZError(errors.E2051, "@suppress can only be applied to function declarations", suppressAttr.Token)
			// Clear @suppress attributes to prevent them from being applied
			newAttrs := []*Attribute{}
			for _, attr := range attrs {
				if attr.Name != "suppress" {
					newAttrs = append(newAttrs, attr)
				}
			}
			attrs = newAttrs
		}
	}

	switch p.currentToken.Type {
	case CONST:
		// For struct declarations like "const Name struct { ... }",
		// we handle them in parseVarableDeclaration by detecting the STRUCT token
		stmt := p.parseVarableDeclarationOrStruct()
		if stmt != nil {
			// Apply visibility and attributes
			switch s := stmt.(type) {
			case *VariableDeclaration:
				s.Visibility = visibility
				if len(attrs) > 0 {
					s.Attributes = attrs
				}
			case *StructDeclaration:
				s.Visibility = visibility
			case *EnumDeclaration:
				s.Visibility = visibility
				if len(attrs) > 0 {
					s.Attributes = p.parseEnumAttributes(attrs)
				}
			}
		}
		return stmt
	case TEMP:
		stmt := p.parseVarableDeclaration()
		if stmt != nil {
			stmt.Visibility = visibility
			if len(attrs) > 0 {
				stmt.Attributes = attrs
			}
		}
		return stmt
	case DO:
		stmt := p.parseFunctionDeclarationWithAttrs(attrs)
		if stmt != nil {
			stmt.Visibility = visibility
		}
		return stmt
	case RETURN:
		return p.parseReturnStatement()
	case IF:
		return p.parseIfStatement()
	case WHEN:
		return p.parseWhenStatement(attrs)
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
				p.addEZError(errors.E2008, msg, p.currentToken)
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
			result := p.parseStructDeclaration()
			if result == nil {
				return nil // Return untyped nil to avoid typed nil interface issue
			}
			return result
		}

		if p.peekTokenMatches(ENUM) {
			// This is an enum declaration
			// currentToken is now the enum name, which is what parseEnumDeclaration expects
			result := p.parseEnumDeclaration()
			if result == nil {
				return nil // Return untyped nil to avoid typed nil interface issue
			}
			return result
		}

		// Not a struct, restore and parse as variable
		// But we've already consumed one token, so we need to account for that
		// The best approach is to just continue with variable parsing from here
		// We're at the name token, so we can build the VariableDeclaration
		stmt := &VariableDeclaration{Token: savedCurrent}
		stmt.Mutable = false // it's a const

		// Check for reserved names
		if isReservedName(nameToken.Literal) {
			msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a variable name", nameToken.Literal)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2020, msg, nameToken)
			return nil
		}

		// Check for duplicate declaration
		if !p.declareInScope(nameToken.Literal, nameToken) {
			msg := fmt.Sprintf("'%s' is already declared in this scope", nameToken.Literal)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2023, msg, nameToken)
			return nil
		}

		stmt.Names = append(stmt.Names, &Label{Token: nameToken, Value: nameToken.Literal})
		stmt.Name = stmt.Names[0]

		// Check if this is a type-inferred assignment (const x = value)
		// This allows: const p = new(Person) or const p = Person{...}
		if p.peekTokenMatches(ASSIGN) {
			p.nextToken() // consume =
			p.nextToken() // move to value
			stmt.Value = p.parseExpression(LOWEST)
			return stmt
		}

		// Parse type using parseTypeName which handles qualified names (module.Type),
		// arrays ([type], [type,size]), and maps (map[key:value])
		p.nextToken()
		stmt.TypeName = p.parseTypeName()
		if stmt.TypeName == "" {
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
				p.addEZError(errors.E2003, msg, assignToken)
				return nil
			}

			stmt.Value = p.parseExpression(LOWEST)
		} else if !stmt.Mutable {
			// const must be initialized
			msg := fmt.Sprintf("const '%s' must be initialized with a value", stmt.Name.Value)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2011, msg, p.currentToken)
			return nil
		}

		return stmt
	}

	// Not the special "const Name struct" case, use regular variable declaration
	result := p.parseVarableDeclaration()
	if result == nil {
		return nil // Return untyped nil to avoid typed nil interface issue
	}
	return result
}

func (p *Parser) parseVarableDeclaration() *VariableDeclaration {
	stmt := &VariableDeclaration{Token: p.currentToken}
	stmt.Mutable = p.currentToken.Type == TEMP

	// Check for blank identifier (_) or IDENT
	if p.peekTokenMatches(BLANK) {
		p.nextToken()
		stmt.Names = append(stmt.Names, &Label{Token: p.currentToken, Value: "_"})
	} else if IsKeyword(p.peekToken.Type) {
		// If user tries to use a keyword as variable name, give helpful error
		keyword := KeywordLiteral(p.peekToken.Type)
		msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a variable name", keyword)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2020, msg, p.peekToken)
		return nil
	} else if !p.expectPeek(IDENT) {
		return nil
	} else {
		name := p.currentToken.Literal
		// Check for reserved names
		if isReservedName(name) {
			msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a variable name", name)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2020, msg, p.currentToken)
			return nil
		}
		// Check for duplicate declaration
		if !p.declareInScope(name, p.currentToken) {
			msg := fmt.Sprintf("'%s' is already declared in this scope", name)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2023, msg, p.currentToken)
			return nil
		}
		stmt.Names = append(stmt.Names, &Label{Token: p.currentToken, Value: name})
	}

	// Check for multiple assignment: temp result, err = ...
	for p.peekTokenMatches(COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next identifier or _

		if p.currentTokenMatches(BLANK) {
			stmt.Names = append(stmt.Names, &Label{Token: p.currentToken, Value: "_"})
		} else if IsKeyword(p.currentToken.Type) {
			// If user tries to use a keyword as variable name, give helpful error
			keyword := KeywordLiteral(p.currentToken.Type)
			msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a variable name", keyword)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2020, msg, p.currentToken)
			return nil
		} else if p.currentTokenMatches(IDENT) {
			name := p.currentToken.Literal
			// Check for reserved names
			if isReservedName(name) {
				msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a variable name", name)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E2020, msg, p.currentToken)
				return nil
			}
			// Check for duplicate declaration
			if !p.declareInScope(name, p.currentToken) {
				msg := fmt.Sprintf("'%s' is already declared in this scope", name)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E2023, msg, p.currentToken)
				return nil
			}
			stmt.Names = append(stmt.Names, &Label{Token: p.currentToken, Value: name})
		} else {
			msg := fmt.Sprintf("expected identifier or _ (blank identifier), got %s", p.currentToken.Type)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2029, msg, p.currentToken)
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

	// Check for typed tuple unpacking: temp a int, b string = getValues()
	if p.peekTokenMatches(COMMA) {
		// Initialize TypeNames with the first type
		stmt.TypeNames = append(stmt.TypeNames, stmt.TypeName)

		// Parse additional (name type) pairs
		for p.peekTokenMatches(COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next identifier

			if !p.currentTokenMatches(IDENT) {
				msg := fmt.Sprintf("expected identifier, got %s", p.currentToken.Type)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E2029, msg, p.currentToken)
				return nil
			}

			name := p.currentToken.Literal
			// Check for reserved names
			if isReservedName(name) {
				msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a variable name", name)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E2020, msg, p.currentToken)
				return nil
			}
			// Check for duplicate declaration
			if !p.declareInScope(name, p.currentToken) {
				msg := fmt.Sprintf("'%s' is already declared in this scope", name)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E2023, msg, p.currentToken)
				return nil
			}
			stmt.Names = append(stmt.Names, &Label{Token: p.currentToken, Value: name})

			// Parse type for this variable
			p.nextToken()
			typeName := p.parseTypeName()
			if typeName == "" {
				return nil
			}
			stmt.TypeNames = append(stmt.TypeNames, typeName)
		}

		// Typed tuple unpacking requires initialization
		if !p.expectPeek(ASSIGN) {
			return nil
		}
		p.nextToken() // move to value
		stmt.Value = p.parseExpression(LOWEST)
		return stmt
	}

	// Optional initialization
	if p.peekTokenMatches(ASSIGN) {
		assignToken := p.peekToken // save the = token for error reporting
		p.nextToken()              // consume =
		p.nextToken()              // move to value

		// Check if we immediately hit end of block or EOF (incomplete statement)
		if p.currentTokenMatches(RBRACE) || p.currentTokenMatches(EOF) {
			msg := "expected expression after '='"
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2003, msg, assignToken)
			return nil
		}

		stmt.Value = p.parseExpression(LOWEST)
	} else if !stmt.Mutable {
		// const must be initialized
		msg := fmt.Sprintf("const '%s' must be initialized with a value", stmt.Name.Value)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2011, msg, p.currentToken)
		return nil
	}

	return stmt
}

func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{Token: p.currentToken}
	stmt.Values = []Expression{}

	// Check for empty return BEFORE advancing
	// This prevents consuming the closing brace which would leave the block unclosed
	if p.peekTokenMatches(RBRACE) || p.peekTokenMatches(EOF) || p.peekTokenMatches(NEWLINE) {
		return stmt
	}

	p.nextToken() // Only advance if there's a value to parse

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
		// Check for import statements inside blocks (not allowed)
		if p.currentTokenMatches(IMPORT) {
			p.addEZError(errors.E2036, "import statements must appear at the top of the file, not inside blocks", p.currentToken)
			// Skip the import statement to continue parsing
			for !p.currentTokenMatches(NEWLINE) && !p.currentTokenMatches(RBRACE) && !p.currentTokenMatches(EOF) {
				p.nextToken()
			}
			continue
		}

		stmt := p.parseStatement()
		if stmt != nil {
			// Check for unreachable code
			// Don't warn if the unreachable code is due to an if-statement alternative
			// (otherwise/or) which will be handled by the if statement parser
			if unreachable && !p.peekTokenMatches(OTHERWISE) && !p.peekTokenMatches(OR_KW) {
				// Check if W2001 is suppressed
				if !p.isSuppressed("W2001", suppressions) && !p.isSuppressed("unreachable_code", suppressions) {
					// Warn about unreachable code
					msg := "unreachable code after return/break/continue/exit/panic"
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
			switch s := stmt.(type) {
			case *ReturnStatement, *BreakStatement, *ContinueStatement:
				unreachable = true
			case *ExpressionStatement:
				// Check for noreturn function calls like exit() and panic()
				if call, ok := s.Expression.(*CallExpression); ok {
					if isNoReturnCall(call) {
						unreachable = true
					}
				}
			}
		}
		// Always advance to the next token after parsing a statement.
		// The loop condition will exit when we reach the block's closing brace.
		p.nextToken()
	}

	// Check if we exited due to EOF (unclosed brace)
	if p.currentTokenMatches(EOF) {
		msg := "unclosed brace - expected '}'"
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2004, msg, openBrace)
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

// parseWhenStatement parses when/is/default statements
// Syntax: when value { is 1 { ... } is 2, 3 { ... } default { ... } }
func (p *Parser) parseWhenStatement(attrs []*Attribute) *WhenStatement {
	stmt := &WhenStatement{Token: p.currentToken, Attributes: attrs}

	// Check for @strict attribute
	for _, attr := range attrs {
		// @strict is parsed as an attribute with Name="strict"
		if attr.Name == "strict" {
			stmt.IsStrict = true
			break
		}
	}

	p.nextToken() // move past 'when'
	stmt.Value = p.parseExpression(LOWEST)

	if !p.expectPeek(LBRACE) {
		return nil
	}

	p.nextToken() // move past '{'

	// Parse cases and default
	for !p.currentTokenMatches(RBRACE) && !p.currentTokenMatches(EOF) {
		// Skip newlines
		for p.currentTokenMatches(NEWLINE) {
			p.nextToken()
		}

		if p.currentTokenMatches(RBRACE) {
			break
		}

		if p.currentTokenMatches(IS) {
			whenCase := p.parseWhenCase()
			if whenCase != nil {
				stmt.Cases = append(stmt.Cases, whenCase)
			}
		} else if p.currentTokenMatches(DEFAULT) {
			p.nextToken() // move past 'default'
			if !p.currentTokenMatches(LBRACE) {
				p.addEZError(errors.E2002, "expected '{' after default", p.currentToken)
				return nil
			}
			stmt.Default = p.parseBlockStatement()
			p.nextToken() // move past '}'
		} else {
			p.addEZError(errors.E2001, fmt.Sprintf("unexpected token '%s' in when statement, expected 'is' or 'default'", p.currentToken.Literal), p.currentToken)
			p.nextToken()
		}
	}

	// Validate: default is required unless @strict
	if stmt.Default == nil && !stmt.IsStrict {
		p.addEZError(errors.E2041, "when statement requires a 'default' case", stmt.Token)
	}

	// Validate: @strict cannot have default
	if stmt.Default != nil && stmt.IsStrict {
		p.addEZError(errors.E2042, "@strict when statement cannot have a 'default' case", stmt.Token)
	}

	return stmt
}

// parseWhenCase parses a single is case: is 1, 2, 3 { ... }
func (p *Parser) parseWhenCase() *WhenCase {
	whenCase := &WhenCase{Token: p.currentToken}

	p.nextToken() // move past 'is'

	// Parse the first value
	firstVal := p.parseExpression(LOWEST)
	if firstVal == nil {
		return nil
	}

	// Check if this is a range expression
	if _, ok := firstVal.(*RangeExpression); ok {
		whenCase.IsRange = true
	}
	if call, ok := firstVal.(*CallExpression); ok {
		if label, ok := call.Function.(*Label); ok && label.Value == "range" {
			whenCase.IsRange = true
		}
	}

	whenCase.Values = append(whenCase.Values, firstVal)

	// Parse additional comma-separated values
	for p.peekTokenMatches(COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next value
		val := p.parseExpression(LOWEST)
		if val != nil {
			whenCase.Values = append(whenCase.Values, val)
		}
	}

	// Expect the block
	if !p.expectPeek(LBRACE) {
		return nil
	}

	whenCase.Body = p.parseBlockStatement()
	p.nextToken() // move past '}'

	return whenCase
}

func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{Token: p.currentToken}

	// Check for optional opening parenthesis
	hasParens := p.peekTokenMatches(LPAREN)
	if hasParens {
		p.nextToken() // consume '('
	}

	if IsKeyword(p.peekToken.Type) {
		// If user tries to use a keyword as loop variable, give helpful error
		keyword := KeywordLiteral(p.peekToken.Type)
		msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a variable name", keyword)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2020, msg, p.peekToken)
		return nil
	}
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

	// Check for optional closing parenthesis
	if hasParens {
		if !p.expectPeek(RPAREN) {
			return nil
		}
	}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseForEachStatement() *ForEachStatement {
	stmt := &ForEachStatement{Token: p.currentToken}

	// Check for optional opening parenthesis
	hasParens := p.peekTokenMatches(LPAREN)
	if hasParens {
		p.nextToken() // consume '('
	}

	if IsKeyword(p.peekToken.Type) {
		// If user tries to use a keyword as loop variable, give helpful error
		keyword := KeywordLiteral(p.peekToken.Type)
		msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a variable name", keyword)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2020, msg, p.peekToken)
		return nil
	}
	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Variable = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

	if !p.expectPeek(IN) {
		return nil
	}

	p.nextToken() // move past 'in'
	stmt.Collection = p.parseExpression(LOWEST)

	// Check for optional closing parenthesis
	if hasParens {
		if !p.expectPeek(RPAREN) {
			return nil
		}
	}

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

func (p *Parser) parseFunctionDeclarationWithAttrs(attrs []*Attribute) *FunctionDeclaration {
	stmt := &FunctionDeclaration{Token: p.currentToken, Attributes: attrs}

	// Check if we're inside a function - nested functions are not allowed
	if p.functionDepth > 0 {
		msg := "function declarations are not allowed inside functions"
		p.addEZError(errors.E2019, msg, p.currentToken)
		return nil
	}

	if IsKeyword(p.peekToken.Type) {
		// If user tries to use a keyword as function name, give helpful error
		keyword := KeywordLiteral(p.peekToken.Type)
		msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a function name", keyword)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2021, msg, p.peekToken)
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
		p.addEZError(errors.E2021, msg, p.currentToken)
		return nil
	}
	// Check for duplicate declaration
	if !p.declareInScope(name, p.currentToken) {
		msg := fmt.Sprintf("'%s' is already declared in this scope", name)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2023, msg, p.currentToken)
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
		p.nextToken()             // consume ->
		p.nextToken()             // move to return type

		// Check if return type is missing (e.g., "do func() -> {")
		if p.currentTokenMatches(LBRACE) {
			msg := "expected return type after '->'"
			p.addEZError(errors.E2015, msg, arrowToken)
			return nil
		}

		if p.currentTokenMatches(LPAREN) {
			// Multiple return types
			stmt.ReturnTypes = p.parseReturnTypes()
		} else if p.currentTokenMatches(IDENT) || p.currentTokenMatches(LBRACKET) {
			// Single return type (identifier or array)
			typeName := p.parseTypeName()
			if typeName == "" {
				return nil
			}
			stmt.ReturnTypes = []string{typeName}
		} else {
			// Invalid token after arrow
			msg := fmt.Sprintf("expected return type after '->', got %s instead", p.currentToken.Type)
			p.addEZError(errors.E2015, msg, arrowToken)
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
	seenDefault := false                 // track if we've seen a parameter with default value

	if p.peekTokenMatches(RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()

	// Helper struct to track name and mutability together
	type paramInfo struct {
		name    *Label
		mutable bool
	}

	for {
		// Collect parameter names that will share a type
		// e.g., in "x, y int", collect ["x", "y"]
		// e.g., in "&x, &y int", collect ["x", "y"] with mutable=true
		namesForType := []paramInfo{}

		// Check for & prefix (mutable parameter)
		isMutable := false
		if p.currentTokenMatches(AMPERSAND) {
			isMutable = true
			p.nextToken() // consume &
		}

		// Read first parameter name
		if IsKeyword(p.currentToken.Type) {
			// If user tries to use a keyword as parameter name, give helpful error
			keyword := KeywordLiteral(p.currentToken.Type)
			msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a parameter name", keyword)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2033, msg, p.currentToken)
			return nil
		}
		if !p.currentTokenMatches(IDENT) {
			msg := fmt.Sprintf("expected parameter name, got %s", p.currentToken.Type)
			p.addEZError(errors.E2002, msg, p.currentToken)
			return nil
		}

		// Collect all names before the type
		// Strategy: collect IDENT tokens while they're followed by COMMA
		// The last IDENT (not followed by COMMA) is the type
		for {
			if IsKeyword(p.currentToken.Type) {
				// If user tries to use a keyword as parameter name, give helpful error
				keyword := KeywordLiteral(p.currentToken.Type)
				msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a parameter name", keyword)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E2033, msg, p.currentToken)
				return nil
			}
			if !p.currentTokenMatches(IDENT) {
				msg := fmt.Sprintf("expected parameter name, got %s", p.currentToken.Type)
				p.addEZError(errors.E2002, msg, p.currentToken)
				return nil
			}

			currentIdent := &Label{Token: p.currentToken, Value: p.currentToken.Literal}

			// Look ahead to see what follows this IDENT
			if p.peekTokenMatches(COMMA) {
				// This IDENT is a parameter name (more names or params follow)
				// Check for reserved names
				if isReservedName(currentIdent.Value) {
					msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a parameter name", currentIdent.Value)
					p.errors = append(p.errors, msg)
					p.addEZError(errors.E2033, msg, currentIdent.Token)
					return nil
				}
				// Check for duplicate
				if prevToken, exists := paramNames[currentIdent.Value]; exists {
					msg := fmt.Sprintf("duplicate parameter name '%s'", currentIdent.Value)
					p.addEZError(errors.E2012, msg, currentIdent.Token)
					helpMsg := fmt.Sprintf("parameter '%s' first declared at line %d", currentIdent.Value, prevToken.Line)
					p.errors = append(p.errors, helpMsg)
				} else {
					paramNames[currentIdent.Value] = currentIdent.Token
				}
				namesForType = append(namesForType, paramInfo{name: currentIdent, mutable: isMutable})
				p.nextToken() // consume IDENT
				p.nextToken() // consume COMMA, move to next token

				// Check for & prefix on next parameter in the shared-type group
				isMutable = false
				if p.currentTokenMatches(AMPERSAND) {
					isMutable = true
					p.nextToken() // consume &
				}
				continue
			} else if p.peekTokenMatches(IDENT) || p.peekTokenMatches(LBRACKET) {
				// This IDENT is a parameter name, and the next token is the type
				// Check for reserved names
				if isReservedName(currentIdent.Value) {
					msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a parameter name", currentIdent.Value)
					p.errors = append(p.errors, msg)
					p.addEZError(errors.E2033, msg, currentIdent.Token)
					return nil
				}
				// Check for duplicate
				if prevToken, exists := paramNames[currentIdent.Value]; exists {
					msg := fmt.Sprintf("duplicate parameter name '%s'", currentIdent.Value)
					p.addEZError(errors.E2012, msg, currentIdent.Token)
					helpMsg := fmt.Sprintf("parameter '%s' first declared at line %d", currentIdent.Value, prevToken.Line)
					p.errors = append(p.errors, helpMsg)
				} else {
					paramNames[currentIdent.Value] = currentIdent.Token
				}
				namesForType = append(namesForType, paramInfo{name: currentIdent, mutable: isMutable})
				p.nextToken() // move to the type
				break
			} else if p.peekTokenMatches(RPAREN) {
				// Incomplete parameter - name without type before closing paren
				msg := fmt.Sprintf("parameter '%s' is missing a type", currentIdent.Value)
				p.addEZError(errors.E2014, msg, currentIdent.Token)
				return nil
			} else {
				msg := fmt.Sprintf("expected ',', type, or ')' after parameter name, got %s", p.peekToken.Type)
				p.addEZError(errors.E2002, msg, p.peekToken)
				return nil
			}
		}

		// Now current token should be the type for all collected names
		if !p.currentTokenMatches(IDENT) && !p.currentTokenMatches(LBRACKET) {
			msg := fmt.Sprintf("expected parameter type, got %s", p.currentToken.Type)
			p.addEZError(errors.E2014, msg, p.currentToken)
			return nil
		}

		// Use parseTypeName to handle qualified types (e.g., module.Type) and arrays
		typeName := p.parseTypeName()
		if typeName == "" {
			return nil
		}

		// Check for default value (= expression)
		var defaultValue Expression
		if p.peekTokenMatches(ASSIGN) {
			p.nextToken() // consume =
			p.nextToken() // move to expression

			// Check if any parameter in this group is mutable - can't have default on mutable params
			for _, info := range namesForType {
				if info.mutable {
					msg := fmt.Sprintf("mutable parameter '%s' cannot have a default value", info.name.Value)
					p.addEZError(errors.E2040, msg, info.name.Token)
					return nil
				}
			}

			defaultValue = p.parseExpression(LOWEST)
			if defaultValue == nil {
				return nil
			}
		} else if seenDefault {
			// Required parameter after one with default - error
			// Get the first param name from this group for the error message
			paramName := namesForType[0].name.Value
			msg := fmt.Sprintf("required parameter '%s' cannot follow a parameter with a default value", paramName)
			p.addEZError(errors.E2039, msg, namesForType[0].name.Token)
			return nil
		}

		// Apply the type to all collected names
		// For grouped params like "x, y int = 0", only the LAST param gets the default
		for i, info := range namesForType {
			param := &Parameter{Name: info.name, TypeName: typeName, Mutable: info.mutable}
			// Only the last parameter in the group gets the default value
			if i == len(namesForType)-1 && defaultValue != nil {
				param.DefaultValue = defaultValue
				seenDefault = true // Mark that we've seen a default (for next iteration)
			}
			params = append(params, param)
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
			p.addEZError(errors.E2002, msg, p.peekToken)
			return nil
		}
	}

	return params
}

func (p *Parser) parseReturnTypes() []string {
	types := []string{}

	p.nextToken() // move past (

	for !p.currentTokenMatches(RPAREN) {
		// Parse type name (can be IDENT or array type starting with LBRACKET)
		if p.currentTokenMatches(IDENT) || p.currentTokenMatches(LBRACKET) {
			typeName := p.parseTypeName()
			if typeName == "" {
				return nil
			}
			types = append(types, typeName)
		} else {
			msg := fmt.Sprintf("expected type name in return types, got %s", p.currentToken.Type)
			p.addEZError(errors.E2002, msg, p.currentToken)
			return nil
		}

		// Check for comma or end of list
		if p.peekTokenMatches(COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next type
		} else if p.peekTokenMatches(RPAREN) {
			p.nextToken() // move to )
		} else {
			msg := fmt.Sprintf("expected ',' or ')' in return types, got %s", p.peekToken.Type)
			p.addEZError(errors.E2002, msg, p.peekToken)
			return nil
		}
	}

	return types
}

func (p *Parser) parseImportStatement() *ImportStatement {
	stmt := &ImportStatement{Token: p.currentToken}
	stmt.Imports = []ImportItem{}

	p.nextToken()

	// Check for "import & use" syntax
	if p.currentTokenMatches(AMPERSAND) {
		p.nextToken() // move past &
		if p.currentTokenMatches(USE) {
			stmt.AutoUse = true
			p.nextToken() // move past 'use' to the module
		} else {
			p.addEZError(errors.E2002, "expected 'use' after '&' in import statement", p.currentToken)
		}
	}

	// Parse first import
	item := p.parseSingleImport()
	if item.Module != "" || item.Path != "" {
		stmt.Imports = append(stmt.Imports, item)
		// For backward compatibility, populate the old fields for single imports
		stmt.Module = item.Module
		stmt.Alias = item.Alias
	}

	// Check for comma-separated imports
	for p.peekTokenMatches(COMMA) {
		p.nextToken() // move to comma
		p.nextToken() // move past comma

		item := p.parseSingleImport()
		if item.Module != "" || item.Path != "" {
			stmt.Imports = append(stmt.Imports, item)
		}
	}

	return stmt
}

// parseSingleImport parses a single import
// Supports:
//   - @module (stdlib)
//   - alias@module (stdlib with alias)
//   - "./path" (user module)
//   - alias"./path" (user module with alias)
func (p *Parser) parseSingleImport() ImportItem {
	item := ImportItem{}

	// Check for path import: "./path" or alias"./path"
	if p.currentTokenMatches(STRING) {
		// Direct path import: "./utils"
		item.Path = p.currentToken.Literal
		item.IsStdlib = false
		// Extract module name from path for default alias
		item.Alias = extractModuleNameFromPath(item.Path)
		return item
	}

	// Stdlib or aliased import
	if p.currentTokenMatches(AT) {
		// @module - no alias, use module name
		p.nextToken()
		if p.currentTokenMatches(IDENT) {
			item.Module = p.currentToken.Literal
			item.Alias = item.Module // use module name as alias
			item.IsStdlib = true
		}
	} else if p.currentTokenMatches(IDENT) {
		// Could be alias@module or alias"./path"
		item.Alias = p.currentToken.Literal
		p.nextToken()

		if p.currentTokenMatches(AT) {
			// alias@module (stdlib)
			p.nextToken()
			if p.currentTokenMatches(IDENT) {
				item.Module = p.currentToken.Literal
				item.IsStdlib = true
			}
		} else if p.currentTokenMatches(STRING) {
			// alias"./path" (user module)
			item.Path = p.currentToken.Literal
			item.IsStdlib = false
		}
	}

	return item
}

// extractModuleNameFromPath extracts the module name from a path
// e.g., "./server" -> "server", "../utils" -> "utils", "./src/networking" -> "networking"
func extractModuleNameFromPath(path string) string {
	// Remove leading ./ or ../
	for strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		if strings.HasPrefix(path, "./") {
			path = path[2:]
		} else if strings.HasPrefix(path, "../") {
			path = path[3:]
		}
	}
	// Get the last component
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

// parseModuleDeclaration parses "module modulename"
func (p *Parser) parseModuleDeclaration() *ModuleDeclaration {
	stmt := &ModuleDeclaration{Token: p.currentToken}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Name = &Label{Token: p.currentToken, Value: p.currentToken.Literal}
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

	// Check for reserved names
	if isReservedName(p.currentToken.Literal) {
		msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as an enum name", p.currentToken.Literal)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2038, msg, p.currentToken)
		return nil
	}

	stmt.Name = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

	p.nextToken() // move past name to 'enum'
	p.nextToken() // move past 'enum' to ':' or '{'

	// Check for type annotation: enum : type { ... }
	if p.currentTokenMatches(COLON) {
		p.nextToken() // move past ':' to type name

		if !p.currentTokenMatches(IDENT) {
			msg := fmt.Sprintf("expected type name after ':', got %s", p.currentToken.Type)
			p.addEZError(errors.E2024, msg, p.currentToken)
			return nil
		}

		typeName := p.currentToken.Literal

		// Validate that type is a primitive (int, float, or string)
		if typeName != "int" && typeName != "float" && typeName != "string" {
			msg := fmt.Sprintf("enum type must be a primitive type (int, float, or string), got '%s'", typeName)
			p.addEZError(errors.E2026, msg, p.currentToken)
			return nil
		}

		// Initialize attributes with the specified type
		stmt.Attributes = &EnumAttributes{
			TypeName: typeName,
			IsFlags:  false,
		}

		p.nextToken() // move past type name to '{'
	}

	if !p.currentTokenMatches(LBRACE) {
		msg := "expected '{' after enum declaration"
		p.addEZError(errors.E2030, msg, p.currentToken)
		return nil
	}

	stmt.Values = []*EnumValue{}
	valueNames := make(map[string]Token) // track value names for duplicate detection

	p.nextToken() // move into enum body

	for !p.currentTokenMatches(RBRACE) && !p.currentTokenMatches(EOF) {
		if !p.currentTokenMatches(IDENT) {
			msg := fmt.Sprintf("expected enum value name, got %s", p.currentToken.Type)
			p.addEZError(errors.E2002, msg, p.currentToken)
			return nil
		}

		enumValue := &EnumValue{}
		enumValue.Name = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

		// Check for reserved names
		if isReservedName(enumValue.Name.Value) {
			msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as an enum value name", enumValue.Name.Value)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2035, msg, enumValue.Name.Token)
			return nil
		}

		// Check for duplicate value names
		if prevToken, exists := valueNames[enumValue.Name.Value]; exists {
			msg := fmt.Sprintf("duplicate value name '%s' in enum '%s'", enumValue.Name.Value, stmt.Name.Value)
			p.addEZError(errors.E2023, msg, enumValue.Name.Token)
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

		// Skip optional comma separator between enum values
		if p.currentTokenMatches(COMMA) {
			p.nextToken()
		}
	}

	if len(stmt.Values) == 0 {
		msg := fmt.Sprintf("enum '%s' must have at least one value", stmt.Name.Value)
		p.addEZError(errors.E2016, msg, stmt.Token)
	}

	return stmt
}

// parseTypeName parses a type name which can be:
// - Simple type: int, string, StructName
// - Array type: [int], [string], [StructName]
// - Fixed-size array: [int,5], [string,10]
// - Map type: map[keyType:valueType]
func (p *Parser) parseTypeName() string {
	if p.currentTokenMatches(LBRACKET) {
		// Array type [type] or [type, size]
		typeName := "["
		p.nextToken() // move past [

		// The element type can be an identifier, another array type (for matrices),
		// or a map type (for arrays of maps)
		var elementType string
		if p.currentTokenMatches(LBRACKET) {
			// Nested array type [[type]] - recursive call
			elementType = p.parseTypeName()
			if elementType == "" {
				return ""
			}
		} else if p.currentTokenMatches(IDENT) {
			// Check for map type inside array: [map[K:V]]
			if p.currentToken.Literal == "map" && p.peekTokenMatches(LBRACKET) {
				elementType = p.parseMapTypeName()
				if elementType == "" {
					return ""
				}
			} else {
				elementType = p.currentToken.Literal
				// Check for qualified type name inside array: [module.TypeName]
				if p.peekTokenMatches(DOT) {
					p.nextToken() // consume the DOT
					p.nextToken() // get the type name
					if !p.currentTokenMatches(IDENT) {
						msg := fmt.Sprintf("expected type name after '.', got %s", p.currentToken.Type)
						p.errors = append(p.errors, msg)
						p.addEZError(errors.E2024, msg, p.currentToken)
						return ""
					}
					elementType = elementType + "." + p.currentToken.Literal
				}
			}
		} else {
			msg := fmt.Sprintf("expected type name, got %s", p.currentToken.Type)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2024, msg, p.currentToken)
			return ""
		}
		typeName += elementType

		// Check for fixed-size array syntax: [type, size]
		if p.peekTokenMatches(COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // get size

			// Size should be an integer
			if !p.currentTokenMatches(INT) {
				msg := fmt.Sprintf("expected integer for array size, got %s", p.currentToken.Type)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E2025, msg, p.currentToken)
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
		// Check for map type: map[keyType:valueType]
		if p.currentToken.Literal == "map" && p.peekTokenMatches(LBRACKET) {
			return p.parseMapTypeName()
		}

		// Check for qualified type name: module.TypeName
		typeName := p.currentToken.Literal
		if p.peekTokenMatches(DOT) {
			p.nextToken() // consume the DOT
			p.nextToken() // get the type name
			if !p.currentTokenMatches(IDENT) {
				msg := fmt.Sprintf("expected type name after '.', got %s", p.currentToken.Type)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E2024, msg, p.currentToken)
				return ""
			}
			typeName = typeName + "." + p.currentToken.Literal
		}
		return typeName
	} else {
		msg := fmt.Sprintf("expected type, got %s", p.currentToken.Type)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2024, msg, p.currentToken)
		return ""
	}
}

// parseMapTypeName parses map[keyType:valueType]
func (p *Parser) parseMapTypeName() string {
	typeName := "map["

	if !p.expectPeek(LBRACKET) {
		return ""
	}

	// Parse key type
	if !p.expectPeek(IDENT) {
		return ""
	}
	typeName += p.currentToken.Literal

	// Expect colon separator
	if !p.expectPeek(COLON) {
		msg := "expected ':' in map type declaration (e.g., map[string:int])"
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2024, msg, p.currentToken)
		return ""
	}
	typeName += ":"

	// Parse value type - can be an identifier, array type, or nested map type
	p.nextToken()
	if p.currentTokenMatches(LBRACKET) {
		// Array value type: map[K:[V]]
		valueType := p.parseTypeName()
		if valueType == "" {
			return ""
		}
		typeName += valueType
	} else if p.currentTokenMatches(IDENT) {
		// Check for nested map type: map[K:map[K2:V2]]
		if p.currentToken.Literal == "map" && p.peekTokenMatches(LBRACKET) {
			valueType := p.parseMapTypeName()
			if valueType == "" {
				return ""
			}
			typeName += valueType
		} else {
			typeName += p.currentToken.Literal
		}
	} else {
		msg := fmt.Sprintf("expected value type in map declaration, got %s", p.currentToken.Type)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2024, msg, p.currentToken)
		return ""
	}

	// Expect closing bracket
	if !p.expectPeek(RBRACKET) {
		return ""
	}
	typeName += "]"

	return typeName
}

func (p *Parser) parseStructDeclaration() *StructDeclaration {
	stmt := &StructDeclaration{Token: p.currentToken}

	// Check for reserved names
	if isReservedName(p.currentToken.Literal) {
		msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a struct name", p.currentToken.Literal)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2037, msg, p.currentToken)
		return nil
	}

	stmt.Name = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

	p.nextToken() // move past name to 'struct'
	p.nextToken() // move past 'struct' to '{'

	if !p.currentTokenMatches(LBRACE) {
		msg := "expected '{' after struct declaration"
		p.addEZError(errors.E2030, msg, p.currentToken)
		return nil
	}

	stmt.Fields = []*StructField{}
	fieldNames := make(map[string]Token) // track field names for duplicate detection

	p.nextToken() // move into struct body

	for !p.currentTokenMatches(RBRACE) && !p.currentTokenMatches(EOF) {
		// Collect all field names (supports "name, email string" syntax)
		var names []*Label

		// Validate first field name
		fieldName := p.currentToken.Literal
		if isReservedName(fieldName) {
			msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a struct field name", fieldName)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2034, msg, p.currentToken)
			return nil
		}
		names = append(names, &Label{Token: p.currentToken, Value: fieldName})

		// Check for comma-separated names
		for p.peekTokenMatches(COMMA) {
			p.nextToken() // move to comma
			p.nextToken() // move to next name
			if !p.currentTokenMatches(IDENT) {
				msg := "expected field name after comma"
				p.addEZError(errors.E2003, msg, p.currentToken)
				return nil
			}
			// Validate subsequent field names
			fieldName := p.currentToken.Literal
			if isReservedName(fieldName) {
				msg := fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a struct field name", fieldName)
				p.errors = append(p.errors, msg)
				p.addEZError(errors.E2034, msg, p.currentToken)
				return nil
			}
			names = append(names, &Label{Token: p.currentToken, Value: fieldName})
		}

		// Move to type token (can be IDENT or LBRACKET for arrays)
		p.nextToken()

		// Parse the type name (handles both simple types and array types)
		typeName := p.parseTypeName()
		if typeName == "" {
			return nil
		}

		// Create a field for each name with the same type
		for _, name := range names {
			// Check for duplicate field names
			if prevToken, exists := fieldNames[name.Value]; exists {
				msg := fmt.Sprintf("duplicate field name '%s' in struct '%s'", name.Value, stmt.Name.Value)
				p.addEZError(errors.E2013, msg, name.Token)
				// Add help message pointing to first declaration
				helpMsg := fmt.Sprintf("field '%s' first declared at line %d", name.Value, prevToken.Line)
				p.errors = append(p.errors, helpMsg)
			} else {
				fieldNames[name.Value] = name.Token
			}

			field := &StructField{
				Name:     name,
				TypeName: typeName,
			}
			stmt.Fields = append(stmt.Fields, field)
		}

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
			code = errors.E2003
		case RPAREN:
			msg = "expected expression, found ')'"
			code = errors.E2003
		case RBRACKET:
			msg = "expected expression, found ']'"
			code = errors.E2003
		case AMPERSAND:
			msg = "'&' is only valid for mutable parameters (e.g., 'do foo(&x int)') or import & use"
			code = errors.E2001
		default:
			msg = fmt.Sprintf("unexpected token '%s'", p.currentToken.Literal)
			code = errors.E2001
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
	// Only treat it as struct literal if:
	// 1. identifier starts with uppercase (type naming convention)
	// 2. next token is {
	// 3. it actually looks like a struct literal (empty {}, or field: value pattern)
	//    This check prevents treating "if val > MAX_VALUE {" as a struct literal
	if p.peekTokenMatches(LBRACE) && len(ident.Value) > 0 && ident.Value[0] >= 'A' && ident.Value[0] <= 'Z' {
		// Use speculative lookahead to check what's inside the brace
		if p.looksLikeStructLiteral() {
			p.nextToken() // move to {
			return p.parseStructLiteralFromIdent(ident)
		}
	}

	return ident
}

// looksLikeStructLiteral checks if the upcoming { starts a struct literal
// by peeking ahead: struct literals have {} or {field: ...} pattern
func (p *Parser) looksLikeStructLiteral() bool {
	// Save lexer state for speculative lookahead
	savedState := p.l.SaveState()

	// Get the token after { (which is currently peekToken)
	// The lexer is positioned to read what comes after {
	afterBrace := p.l.NextToken()

	// Empty struct: {}
	if afterBrace.Type == RBRACE {
		p.l.RestoreState(savedState)
		return true
	}

	// Non-empty struct must have field: pattern
	if afterBrace.Type == IDENT {
		// Get the token after the identifier (don't restore yet)
		afterIdent := p.l.NextToken()

		// Restore lexer state now
		p.l.RestoreState(savedState)

		// If IDENT is followed by COLON, it's a struct literal
		if afterIdent.Type == COLON {
			return true
		}
		return false
	}

	// Not a struct literal pattern
	p.l.RestoreState(savedState)
	return false
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

func (p *Parser) parseIntegerValue() Expression {
	lit := &IntegerValue{Token: p.currentToken}

	// Strip underscores for numeric conversion (they're only for readability)
	cleanedLiteral := stripUnderscores(p.currentToken.Literal)

	// Use big.Int to parse integers of arbitrary size
	value := new(big.Int)
	_, ok := value.SetString(cleanedLiteral, 0)
	if !ok {
		msg := fmt.Sprintf("could not parse %q as integer", p.currentToken.Literal)
		p.errors = append(p.errors, msg)
		p.addEZError(errors.E2027, msg, p.currentToken)
		return nil
	}

	lit.Value = value
	return lit
}

// isAllUpperCase returns true if the string contains only uppercase letters
// and underscores (enum member naming convention like ACTIVE, PENDING, FOO_BAR).
// Returns false for PascalCase type names like Person, MyType.
func isAllUpperCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if ch >= 'a' && ch <= 'z' {
			return false // has lowercase letter, so it's not all uppercase
		}
	}
	return true
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
		p.addEZError(errors.E2028, msg, p.currentToken)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseStringValue() Expression {
	token := p.currentToken
	literal := p.currentToken.Literal

	// Process escape sequences like \n, \t, etc.
	literal = processEscapeSequences(literal)

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
				p.addEZError(errors.E2007, "unclosed interpolation in string", token)
				return &StringValue{Token: token, Value: literal}
			}

			// Parse the expression inside ${}
			exprStr := literal[exprStart:i]
			if exprStr == "" {
				p.addEZError(errors.E2002, "empty interpolation in string", token)
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

// processEscapeSequences converts escape sequences like \n, \t to actual characters
func processEscapeSequences(s string) string {
	var result strings.Builder
	result.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result.WriteByte('\n')
				i++
			case 't':
				result.WriteByte('\t')
				i++
			case 'r':
				result.WriteByte('\r')
				i++
			case '\\':
				result.WriteByte('\\')
				i++
			case '"':
				result.WriteByte('"')
				i++
			case '\'':
				result.WriteByte('\'')
				i++
			case '0':
				result.WriteByte(0)
				i++
			default:
				// Not a valid escape, keep as-is (lexer already reported error)
				result.WriteByte(s[i])
			}
		} else {
			result.WriteByte(s[i])
		}
	}
	return result.String()
}

// parseInterpolatedExpression parses an expression from within ${}
func (p *Parser) parseInterpolatedExpression(exprStr string, origToken Token) Expression {
	// Create a new lexer for the expression with the original token's line/column
	// This ensures error messages point to the correct location in the source
	lexer := NewLexerWithOffset(exprStr, origToken.Line, origToken.Column)

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
	literal := p.currentToken.Literal
	if len(literal) == 0 {
		return &CharValue{Token: p.currentToken, Value: 0}
	}

	// Handle escape sequences
	if literal[0] == '\\' && len(literal) >= 2 {
		var ch rune
		switch literal[1] {
		case 'n':
			ch = '\n'
		case 't':
			ch = '\t'
		case 'r':
			ch = '\r'
		case '\\':
			ch = '\\'
		case '\'':
			ch = '\''
		case '0':
			ch = 0
		default:
			ch = rune(literal[1])
		}
		return &CharValue{Token: p.currentToken, Value: ch}
	}

	value := []rune(literal)
	return &CharValue{Token: p.currentToken, Value: value[0]}
}

func (p *Parser) parseBooleanValue() Expression {
	return &BooleanValue{Token: p.currentToken, Value: p.currentTokenMatches(TRUE)}
}

func (p *Parser) parseNilValue() Expression {
	return &NilValue{Token: p.currentToken}
}

func (p *Parser) parseArrayValue() Expression {
	startToken := p.currentToken

	// Check if this is an empty literal {} - treat as empty array
	if p.peekTokenMatches(RBRACE) {
		p.nextToken()
		return &ArrayValue{Token: startToken, Elements: []Expression{}}
	}

	// Look ahead to determine if this is a map literal or array literal
	// Map literals have the form {key: value, ...}
	// Array literals have the form {elem1, elem2, ...}
	// We need to parse the first expression and check if a colon follows

	p.nextToken() // move to first element

	firstExpr := p.parseExpression(LOWEST)
	if firstExpr == nil {
		return nil
	}

	// Check if this is a map literal (colon after first expression)
	if p.peekTokenMatches(COLON) {
		return p.parseMapValueFromFirst(startToken, firstExpr)
	}

	// This is an array literal - continue parsing remaining elements
	elements := []Expression{firstExpr}

	for p.peekTokenMatches(COMMA) {
		p.nextToken() // consume comma
		// Check for trailing comma
		if p.peekTokenMatches(RBRACE) {
			p.addEZError(errors.E2017,
				"unexpected trailing comma in array literal\n\n"+
					"Trailing commas are not allowed. Remove the comma before the closing bracket.\n\n"+
					"Help: Remove the trailing comma or add another element:\n"+
					"  Valid: {1, 2, 3}\n"+
					"  Invalid: {1, 2, 3,}",
				p.peekToken)
			return nil
		}
		p.nextToken() // move to next expression
		elements = append(elements, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(RBRACE) {
		return nil
	}

	return &ArrayValue{Token: startToken, Elements: elements}
}

// parseMapValueFromFirst parses a map literal when we've already parsed the first key
func (p *Parser) parseMapValueFromFirst(startToken Token, firstKey Expression) Expression {
	mapLit := &MapValue{Token: startToken, Pairs: []*MapPair{}}

	// We already have the first key, now parse the first value
	if !p.expectPeek(COLON) {
		return nil
	}
	p.nextToken() // move to value
	firstValue := p.parseExpression(LOWEST)
	if firstValue == nil {
		return nil
	}
	mapLit.Pairs = append(mapLit.Pairs, &MapPair{Key: firstKey, Value: firstValue})

	// Parse remaining key-value pairs
	for p.peekTokenMatches(COMMA) {
		p.nextToken() // consume comma
		// Check for trailing comma
		if p.peekTokenMatches(RBRACE) {
			p.addEZError(errors.E2017,
				"unexpected trailing comma in map literal\n\n"+
					"Trailing commas are not allowed. Remove the comma before the closing bracket.\n\n"+
					"Help: Remove the trailing comma or add another entry:\n"+
					"  Valid: {\"a\": 1, \"b\": 2}\n"+
					"  Invalid: {\"a\": 1, \"b\": 2,}",
				p.peekToken)
			return nil
		}
		p.nextToken() // move to next key
		key := p.parseExpression(LOWEST)
		if key == nil {
			return nil
		}
		if !p.expectPeek(COLON) {
			return nil
		}
		p.nextToken() // move to value
		value := p.parseExpression(LOWEST)
		if value == nil {
			return nil
		}
		mapLit.Pairs = append(mapLit.Pairs, &MapPair{Key: key, Value: value})
	}

	if !p.expectPeek(RBRACE) {
		return nil
	}

	return mapLit
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
		// Check for trailing comma
		if p.peekTokenMatches(end) {
			var endName string
			var errorCode errors.ErrorCode
			switch end {
			case RBRACE:
				endName = "array literal"
				errorCode = errors.E2017
			case RPAREN:
				endName = "function call"
				errorCode = errors.E2018
			default:
				endName = "list"
				errorCode = errors.E2017
			}
			p.addEZError(errorCode,
				fmt.Sprintf("unexpected trailing comma in %s\n\n"+
					"Trailing commas are not allowed. Remove the comma before the closing bracket.\n\n"+
					"Help: Remove the trailing comma or add another element:\n"+
					"  Valid: {1, 2, 3}\n"+
					"  Invalid: {1, 2, 3,}", endName),
				p.peekToken)
			return nil
		}
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

	// Special case for minimum int64: -9223372036854775808
	// This value can't be parsed as a positive int64 first then negated,
	// because 9223372036854775808 overflows int64.
	if expression.Operator == "-" && p.peekTokenMatches(INT) {
		cleanedLiteral := stripUnderscores(p.peekToken.Literal)
		if cleanedLiteral == "9223372036854775808" {
			p.nextToken() // consume the integer token
			minInt64 := new(big.Int)
			minInt64.SetString("-9223372036854775808", 10) // math.MinInt64
			return &IntegerValue{
				Token: p.currentToken,
				Value: minInt64,
			}
		}
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

	// Check if this is a qualified struct literal: module.TypeName{...}
	// Only if member starts with uppercase (type naming convention) AND is not all uppercase.
	// All-uppercase names like STATUS.ACTIVE are enum members, not struct types.
	// Struct types use PascalCase (e.g., Person, MyType).
	// Additionally, if the object (left side) starts with uppercase, it's likely an enum
	// access (e.g., Color.Red) not a module.TypeName struct literal.
	memberName := exp.Member.Value
	if p.peekTokenMatches(LBRACE) && len(memberName) > 0 && memberName[0] >= 'A' && memberName[0] <= 'Z' && !isAllUpperCase(memberName) {
		// Build the qualified name from the member expression
		if ident, ok := left.(*Label); ok {
			// If the object (module name) also starts with uppercase, this is likely
			// an enum access like Color.Red, not a struct literal like module.Person{}
			if len(ident.Value) > 0 && ident.Value[0] >= 'A' && ident.Value[0] <= 'Z' {
				return exp // It's an enum access, not a struct literal
			}
			// Use speculative lookahead to verify this is actually a struct literal
			// and not a block statement (e.g., in for_each item in lib.Items { ... })
			if p.looksLikeStructLiteral() {
				qualifiedName := &Label{
					Token: exp.Token,
					Value: ident.Value + "." + memberName,
				}
				p.nextToken() // move to {
				return p.parseStructLiteralFromIdent(qualifiedName)
			}
		}
	}

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

	typeName := p.currentToken.Literal

	// Check for qualified type name: new(module.TypeName)
	if p.peekTokenMatches(DOT) {
		p.nextToken() // consume the DOT
		p.nextToken() // get the type name
		if !p.currentTokenMatches(IDENT) {
			msg := fmt.Sprintf("expected type name after '.', got %s", p.currentToken.Type)
			p.errors = append(p.errors, msg)
			p.addEZError(errors.E2024, msg, p.currentToken)
			return nil
		}
		typeName = typeName + "." + p.currentToken.Literal
	}

	exp.TypeName = &Label{Token: p.currentToken, Value: typeName}

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
	firstArg := p.parseExpression(LOWEST)

	// Check if this is the single-argument form: range(end)
	if p.peekTokenMatches(RPAREN) {
		p.nextToken() // consume RPAREN
		// range(end) -> start=nil (defaults to 0), end=firstArg, step=nil (defaults to 1)
		exp.Start = nil
		exp.End = firstArg
		exp.Step = nil
		return exp
	}

	// Two or three argument form
	if !p.expectPeek(COMMA) {
		return nil
	}

	p.nextToken()
	secondArg := p.parseExpression(LOWEST)

	// Check if this is the two-argument form: range(start, end)
	if p.peekTokenMatches(RPAREN) {
		p.nextToken() // consume RPAREN
		// range(start, end) -> start=firstArg, end=secondArg, step=nil (defaults to 1)
		exp.Start = firstArg
		exp.End = secondArg
		exp.Step = nil
		return exp
	}

	// Three argument form: range(start, end, step)
	if !p.expectPeek(COMMA) {
		return nil
	}

	p.nextToken()
	thirdArg := p.parseExpression(LOWEST)

	if !p.expectPeek(RPAREN) {
		return nil
	}

	exp.Start = firstArg
	exp.End = secondArg
	exp.Step = thirdArg
	return exp
}

// ============================================================================
// Attribute Parsing
// ============================================================================

// Known attribute names (without the @ prefix)
var knownAttributeNames = []string{"suppress", "strict", "enum", "flags", "ignore"}

// parseAttributes parses @suppress(...), @strict, @enum(...), @flags, and detects unknown attributes
func (p *Parser) parseAttributes() []*Attribute {
	attributes := []*Attribute{}

	// Handle @suppress(...), @strict, @enum(...), @flags, and detect unknown attributes
	for p.currentTokenMatches(SUPPRESS) || p.currentTokenMatches(STRICT) || p.currentTokenMatches(ENUM_ATTR) || p.currentTokenMatches(FLAGS) || p.currentTokenMatches(AT) {
		if p.currentTokenMatches(SUPPRESS) {
			// Handle @suppress(...) - existing code
			attributes = append(attributes, p.parseSuppressAttribute())
			continue
		}

		if p.currentTokenMatches(STRICT) {
			// Handle @strict - simple flag attribute for when statements
			attributes = append(attributes, p.parseStrictAttribute())
			continue
		}

		if p.currentTokenMatches(ENUM_ATTR) {
			// Handle @enum(type) - enum type attribute
			attributes = append(attributes, p.parseEnumAttrAttribute())
			continue
		}

		if p.currentTokenMatches(FLAGS) {
			// Handle @flags - power-of-2 enum attribute
			attributes = append(attributes, p.parseFlagsAttribute())
			continue
		}

		// Handle AT token - could be @identifier (unknown/misspelled attribute)
		if p.currentTokenMatches(AT) {
			if p.peekTokenMatches(IDENT) {
				// @identifier - unknown or misspelled attribute name
				atToken := p.currentToken
				p.nextToken() // move to the identifier
				attrName := p.currentToken.Literal

				// Try to suggest a known attribute name
				suggestion := errors.SuggestFromList(attrName, knownAttributeNames)
				var msg string
				if suggestion != "" {
					msg = fmt.Sprintf("unknown attribute '@%s', did you mean '@%s'?", attrName, suggestion)
				} else {
					msg = fmt.Sprintf("unknown attribute '@%s'", attrName)
				}
				p.addEZError(errors.E2001, msg, atToken)

				// Skip past the attribute arguments if present: @name(...)
				if p.peekTokenMatches(LPAREN) {
					p.nextToken() // consume LPAREN
					// Skip until we find matching RPAREN or hit declaration tokens
					depth := 1
					for depth > 0 && !p.currentTokenMatches(EOF) {
						p.nextToken()
						if p.currentTokenMatches(LPAREN) {
							depth++
						} else if p.currentTokenMatches(RPAREN) {
							depth--
						}
					}
				}

				// Move to next token
				p.nextToken()
				continue
			}

			// @ followed by something unexpected - skip it and emit error
			msg := fmt.Sprintf("unexpected token after '@': %s", p.peekToken.Literal)
			p.addEZError(errors.E2001, msg, p.currentToken)
			p.nextToken() // skip the @
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
			code := p.currentToken.Literal
			// Validate the warning code exists and is suppressible
			if !isValidWarningCode(code) {
				msg := fmt.Sprintf("%s is not a valid warning code", code)
				p.addEZError(errors.E2052, msg, p.currentToken)
			} else if !isWarningSuppressible(code) {
				msg := fmt.Sprintf("warning code %s cannot be suppressed", code)
				p.addEZError(errors.E2052, msg, p.currentToken)
			} else {
				attr.Args = append(attr.Args, code)
			}
		}

		// Additional arguments
		for p.peekTokenMatches(COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next argument

			if p.currentTokenMatches(IDENT) || p.currentTokenMatches(STRING) {
				code := p.currentToken.Literal
				// Validate the warning code exists and is suppressible
				if !isValidWarningCode(code) {
					msg := fmt.Sprintf("%s is not a valid warning code", code)
					p.addEZError(errors.E2052, msg, p.currentToken)
				} else if !isWarningSuppressible(code) {
					msg := fmt.Sprintf("warning code %s cannot be suppressed", code)
					p.addEZError(errors.E2052, msg, p.currentToken)
				} else {
					attr.Args = append(attr.Args, code)
				}
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

func (p *Parser) parseStrictAttribute() *Attribute {
	attr := &Attribute{
		Token: p.currentToken,
		Name:  "strict",
		Args:  []string{},
	}

	// Move to next token (the declaration that follows)
	p.nextToken()

	return attr
}

func (p *Parser) parseEnumAttrAttribute() *Attribute {
	attr := &Attribute{
		Token: p.currentToken,
		Name:  "enum",
		Args:  []string{},
	}

	// Expect opening paren
	if !p.expectPeek(LPAREN) {
		msg := "expected '(' after @enum"
		p.addEZError(errors.E2001, msg, p.currentToken)
		return attr
	}

	// Parse the type argument
	if !p.peekTokenMatches(RPAREN) {
		p.nextToken() // move to type argument

		if p.currentTokenMatches(IDENT) {
			typeName := p.currentToken.Literal
			// Validate that type is a primitive (int, float, or string)
			if typeName != "int" && typeName != "float" && typeName != "string" {
				msg := fmt.Sprintf("@enum type must be int, float, or string, got '%s'", typeName)
				p.addEZError(errors.E2026, msg, p.currentToken)
			}
			attr.Args = append(attr.Args, typeName)
		} else {
			msg := fmt.Sprintf("expected type name in @enum(...), got %s", p.currentToken.Type)
			p.addEZError(errors.E2001, msg, p.currentToken)
		}
	} else {
		msg := "@enum requires a type argument, e.g., @enum(int), @enum(float), or @enum(string)"
		p.addEZError(errors.E2001, msg, p.currentToken)
	}

	// Expect closing paren
	if !p.expectPeek(RPAREN) {
		return attr
	}

	// Move to next token
	p.nextToken()

	return attr
}

func (p *Parser) parseFlagsAttribute() *Attribute {
	attr := &Attribute{
		Token: p.currentToken,
		Name:  "flags",
		Args:  []string{},
	}

	// Move to next token (the declaration that follows)
	p.nextToken()

	return attr
}

// parseEnumAttributes converts @enum(type) and @flags attributes to EnumAttributes
func (p *Parser) parseEnumAttributes(attrs []*Attribute) *EnumAttributes {
	enumAttrs := &EnumAttributes{
		TypeName: "int", // default
		IsFlags:  false,
	}

	for _, attr := range attrs {
		if attr.Name == "enum" && len(attr.Args) > 0 {
			// @enum(type) - set the enum type
			enumAttrs.TypeName = attr.Args[0]
		} else if attr.Name == "flags" {
			// @flags - power-of-2 enum (always int type)
			enumAttrs.IsFlags = true
			enumAttrs.TypeName = "int" // flags are always int
		}
	}

	return enumAttrs
}

// isNoReturnCall checks if a call expression is a "noreturn" function
// (i.e., a function that never returns, like exit() or panic())
func isNoReturnCall(call *CallExpression) bool {
	// Check for direct function calls like exit() or panic()
	if label, ok := call.Function.(*Label); ok {
		switch label.Value {
		case "exit", "panic":
			return true
		}
	}

	// Check for member calls like std.exit() or std.panic()
	if member, ok := call.Function.(*MemberExpression); ok {
		// MemberExpression.Member is already a *Label
		switch member.Member.Value {
		case "exit", "panic":
			return true
		}
	}

	return false
}
