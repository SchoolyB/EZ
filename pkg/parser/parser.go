package parser

import (
	"fmt"
	"strconv"
)

import . "github.com/marshallburns/ez/pkg/ast"
import . "github.com/marshallburns/ez/pkg/lexer"
import . "github.com/marshallburns/ez/pkg/tokenizer"

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
}

// Returns a pointer to Parser(p)
// with initialized members
func New(l *Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
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
	msg := fmt.Sprintf("line %d: expected next token to be %s, got %s instead",
		p.peekToken.Line, tType, p.peekToken.Type)
	p.errors = append(p.errors, msg)
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
	switch p.currentToken.Type {
	case TEMP, CONST:
		return p.parseVarableDeclaration()
	case DO:
		return p.parseFunctionDeclaration()
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
		// Check if this is a struct declaration (Name struct { ... })
		if p.peekTokenMatches(STRUCT) {
			return p.parseStructDeclaration()
		}
		// Check if this is an assignment
		if p.peekTokenMatches(ASSIGN) || p.peekTokenMatches(PLUS_ASSIGN) ||
			p.peekTokenMatches(MINUS_ASSIGN) || p.peekTokenMatches(ASTERISK_ASSIGN) ||
			p.peekTokenMatches(SLASH_ASSIGN) || p.peekTokenMatches(PERCENT_ASSIGN) {
			return p.parseAssignmentStatement()
		}
		return p.parseExpressionStatement()
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
		stmt.Names = append(stmt.Names, &Label{Token: p.currentToken, Value: p.currentToken.Literal})
	}

	// Check for multiple assignment: temp result, err = ...
	for p.peekTokenMatches(COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next identifier or @ignore

		if p.currentTokenMatches(IGNORE) {
			stmt.Names = append(stmt.Names, &Label{Token: p.currentToken, Value: "@ignore"})
		} else if p.currentTokenMatches(IDENT) {
			stmt.Names = append(stmt.Names, &Label{Token: p.currentToken, Value: p.currentToken.Literal})
		} else {
			msg := fmt.Sprintf("line %d: expected identifier or @ignore, got %s",
				p.currentToken.Line, p.currentToken.Type)
			p.errors = append(p.errors, msg)
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
		msg := fmt.Sprintf("line %d: expected type, got %s", p.currentToken.Line, p.currentToken.Type)
		p.errors = append(p.errors, msg)
		return nil
	}

	// Optional initialization
	if p.peekTokenMatches(ASSIGN) {
		p.nextToken() // consume =
		p.nextToken() // move to value
		stmt.Value = p.parseExpression(LOWEST)
	} else if !stmt.Mutable {
		// const must be initialized
		msg := fmt.Sprintf("line %d: const '%s' must be initialized with a value",
			p.currentToken.Line, stmt.Name.Value)
		p.errors = append(p.errors, msg)
		return nil
	}

	return stmt
}

func (p *Parser) parseAssignmentStatement() *AssignmentStatement {
	stmt := &AssignmentStatement{Token: p.currentToken}

	// Parse the left-hand side (could be identifier, index, or member)
	stmt.Name = p.parseExpression(LOWEST)

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
	block := &BlockStatement{Token: p.currentToken}
	block.Statements = []Statement{}

	p.nextToken() // move past {

	for !p.currentTokenMatches(RBRACE) && !p.currentTokenMatches(EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
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
	stmt := &FunctionDeclaration{Token: p.currentToken}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Name = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

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

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseFunctionParameters() []*Parameter {
	params := []*Parameter{}

	if p.peekTokenMatches(RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()

	// Collect parameter names until we hit a type
	names := []*Label{{Token: p.currentToken, Value: p.currentToken.Literal}}

	for p.peekTokenMatches(COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next token

		if p.peekTokenMatches(COMMA) || p.peekTokenMatches(RPAREN) {
			// This is another name
			names = append(names, &Label{Token: p.currentToken, Value: p.currentToken.Literal})
		} else {
			// This is a name, next is the type
			names = append(names, &Label{Token: p.currentToken, Value: p.currentToken.Literal})
			p.nextToken() // move to type
			typeName := p.currentToken.Literal

			// Create parameters for all collected names
			for _, name := range names {
				params = append(params, &Parameter{Name: name, TypeName: typeName})
			}
			names = []*Label{}
		}
	}

	// Handle remaining names with type
	if len(names) > 0 {
		p.nextToken() // move to type
		typeName := p.currentToken.Literal
		for _, name := range names {
			params = append(params, &Parameter{Name: name, TypeName: typeName})
		}
	}

	if !p.expectPeek(RPAREN) {
		return nil
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

	p.nextToken() // move into struct body

	for !p.currentTokenMatches(RBRACE) && !p.currentTokenMatches(EOF) {
		field := &StructField{}
		field.Name = &Label{Token: p.currentToken, Value: p.currentToken.Literal}

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
		msg := fmt.Sprintf("line %d: no prefix parse function for %s found",
			p.currentToken.Line, p.currentToken.Type)
		p.errors = append(p.errors, msg)
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
	return &Label{Token: p.currentToken, Value: p.currentToken.Literal}
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
		msg := fmt.Sprintf("line %d: could not parse %q as integer",
			p.currentToken.Line, p.currentToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseFloatValue() Expression {
	lit := &FloatValue{Token: p.currentToken}

	value, err := strconv.ParseFloat(p.currentToken.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("line %d: could not parse %q as float",
			p.currentToken.Line, p.currentToken.Literal)
		p.errors = append(p.errors, msg)
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
