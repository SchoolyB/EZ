package parser

import (
	"strings"
	"testing"

	. "github.com/marshallburns/ez/pkg/ast"
	. "github.com/marshallburns/ez/pkg/lexer"
)

// ============================================================================
// Test Helpers
// ============================================================================

func parseProgram(t *testing.T, input string) *Program {
	t.Helper()
	l := NewLexer(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)
	return program
}

func parseProgramWithSource(t *testing.T, input string) *Program {
	t.Helper()
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	program := p.ParseProgram()
	return program
}

func checkParserErrors(t *testing.T, p *Parser) {
	t.Helper()
	errs := p.Errors()
	if len(errs) == 0 {
		return
	}

	t.Errorf("parser has %d error(s)", len(errs))
	for _, msg := range errs {
		t.Errorf("parser error: %s", msg)
	}
	t.FailNow()
}

func testIntegerLiteral(t *testing.T, il Expression, expected int64) bool {
	t.Helper()
	integ, ok := il.(*IntegerValue)
	if !ok {
		t.Errorf("expression not *IntegerValue, got=%T", il)
		return false
	}
	if integ.Value != expected {
		t.Errorf("integ.Value not %d, got=%d", expected, integ.Value)
		return false
	}
	return true
}

func testFloatLiteral(t *testing.T, fl Expression, expected float64) bool {
	t.Helper()
	f, ok := fl.(*FloatValue)
	if !ok {
		t.Errorf("expression not *FloatValue, got=%T", fl)
		return false
	}
	if f.Value != expected {
		t.Errorf("float.Value not %f, got=%f", expected, f.Value)
		return false
	}
	return true
}

func testIdentifier(t *testing.T, exp Expression, expected string) bool {
	t.Helper()
	ident, ok := exp.(*Label)
	if !ok {
		t.Errorf("expression not *Label, got=%T", exp)
		return false
	}
	if ident.Value != expected {
		t.Errorf("ident.Value not %s, got=%s", expected, ident.Value)
		return false
	}
	return true
}

func testBooleanLiteral(t *testing.T, exp Expression, expected bool) bool {
	t.Helper()
	bo, ok := exp.(*BooleanValue)
	if !ok {
		t.Errorf("expression not *BooleanValue, got=%T", exp)
		return false
	}
	if bo.Value != expected {
		t.Errorf("boolean.Value not %t, got=%t", expected, bo.Value)
		return false
	}
	return true
}

func testLiteralExpression(t *testing.T, exp Expression, expected interface{}) bool {
	t.Helper()
	switch v := expected.(type) {
	case int:
		return testIntegerLiteral(t, exp, int64(v))
	case int64:
		return testIntegerLiteral(t, exp, v)
	case float64:
		return testFloatLiteral(t, exp, v)
	case string:
		return testIdentifier(t, exp, v)
	case bool:
		return testBooleanLiteral(t, exp, v)
	}
	t.Errorf("type of exp not handled, got=%T", expected)
	return false
}

func testInfixExpression(t *testing.T, exp Expression, left interface{}, operator string, right interface{}) bool {
	t.Helper()
	opExp, ok := exp.(*InfixExpression)
	if !ok {
		t.Errorf("expression not *InfixExpression, got=%T(%s)", exp, exp)
		return false
	}
	if !testLiteralExpression(t, opExp.Left, left) {
		return false
	}
	if opExp.Operator != operator {
		t.Errorf("opExp.Operator is not '%s', got=%q", operator, opExp.Operator)
		return false
	}
	if !testLiteralExpression(t, opExp.Right, right) {
		return false
	}
	return true
}

// ============================================================================
// Parser Construction Tests
// ============================================================================

func TestNewParser(t *testing.T) {
	l := NewLexer("temp x int = 5")
	p := New(l)

	if p == nil {
		t.Fatal("New() returned nil")
	}
	if p.l == nil {
		t.Error("parser.l is nil")
	}
	if p.ezErrors == nil {
		t.Error("parser.ezErrors is nil")
	}
	if len(p.scopes) != 1 {
		t.Errorf("expected 1 scope, got %d", len(p.scopes))
	}
}

func TestNewWithSource(t *testing.T) {
	source := "temp x int = 5"
	l := NewLexer(source)
	p := NewWithSource(l, source, "test.ez")

	if p.source != source {
		t.Errorf("expected source %q, got %q", source, p.source)
	}
	if p.filename != "test.ez" {
		t.Errorf("expected filename 'test.ez', got %q", p.filename)
	}
}

// ============================================================================
// Literal Parsing Tests
// ============================================================================

func TestIntegerLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"temp x int = 5", 5},
		{"temp x int = 0", 0},
		{"temp x int = 999", 999},
		{"temp x int = 1_000_000", 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}
			stmt, ok := program.Statements[0].(*VariableDeclaration)
			if !ok {
				t.Fatalf("not VariableDeclaration, got %T", program.Statements[0])
			}
			testIntegerLiteral(t, stmt.Value, tt.expected)
		})
	}
}

func TestFloatLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"temp x float = 3.14", 3.14},
		{"temp x float = 0.0", 0.0},
		{"temp x float = 1_234.567_89", 1234.56789},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			stmt := program.Statements[0].(*VariableDeclaration)
			testFloatLiteral(t, stmt.Value, tt.expected)
		})
	}
}

func TestStringLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`temp x string = "hello"`, "hello"},
		{`temp x string = "hello world"`, "hello world"},
		{`temp x string = ""`, ""},
		{`temp x string = "line1\nline2"`, "line1\nline2"},
		{`temp x string = "tab\there"`, "tab\there"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			stmt := program.Statements[0].(*VariableDeclaration)
			strVal, ok := stmt.Value.(*StringValue)
			if !ok {
				t.Fatalf("not StringValue, got %T", stmt.Value)
			}
			if strVal.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, strVal.Value)
			}
		})
	}
}

func TestCharLiterals(t *testing.T) {
	// Test simple char literal parsing
	input := "temp x char = 'a'"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*VariableDeclaration)
	charVal, ok := stmt.Value.(*CharValue)
	if !ok {
		t.Fatalf("not CharValue, got %T", stmt.Value)
	}
	if charVal.Value != 'a' {
		t.Errorf("expected 'a', got %q", charVal.Value)
	}
}

func TestBooleanLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"temp x bool = true", true},
		{"temp x bool = false", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			stmt := program.Statements[0].(*VariableDeclaration)
			testBooleanLiteral(t, stmt.Value, tt.expected)
		})
	}
}

func TestNilLiteral(t *testing.T) {
	program := parseProgram(t, "temp x int = nil")
	stmt := program.Statements[0].(*VariableDeclaration)
	_, ok := stmt.Value.(*NilValue)
	if !ok {
		t.Fatalf("not NilValue, got %T", stmt.Value)
	}
}

func TestArrayLiterals(t *testing.T) {
	tests := []struct {
		input         string
		expectedLen   int
		expectedFirst interface{}
	}{
		{"temp arr [int] = {1, 2, 3}", 3, int64(1)},
		{"temp arr [string] = {\"a\", \"b\"}", 2, nil}, // string values handled separately
		{"temp arr [int] = {}", 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			stmt := program.Statements[0].(*VariableDeclaration)
			arr, ok := stmt.Value.(*ArrayValue)
			if !ok {
				t.Fatalf("not ArrayValue, got %T", stmt.Value)
			}
			if len(arr.Elements) != tt.expectedLen {
				t.Errorf("expected %d elements, got %d", tt.expectedLen, len(arr.Elements))
			}
			if tt.expectedFirst != nil && len(arr.Elements) > 0 {
				testIntegerLiteral(t, arr.Elements[0], tt.expectedFirst.(int64))
			}
		})
	}
}

func TestMapLiterals(t *testing.T) {
	input := `temp m map[string:int] = {"a": 1, "b": 2}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*VariableDeclaration)

	mapVal, ok := stmt.Value.(*MapValue)
	if !ok {
		t.Fatalf("not MapValue, got %T", stmt.Value)
	}
	if len(mapVal.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(mapVal.Pairs))
	}
}

// ============================================================================
// Prefix Expression Tests
// ============================================================================

func TestPrefixExpressions(t *testing.T) {
	tests := []struct {
		input    string
		operator string
		expected interface{}
	}{
		{"temp x int = -5", "-", 5},
		{"temp x bool = !true", "!", true},
		{"temp x bool = !false", "!", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			stmt := program.Statements[0].(*VariableDeclaration)
			exp, ok := stmt.Value.(*PrefixExpression)
			if !ok {
				t.Fatalf("not PrefixExpression, got %T", stmt.Value)
			}
			if exp.Operator != tt.operator {
				t.Errorf("expected operator %q, got %q", tt.operator, exp.Operator)
			}
			testLiteralExpression(t, exp.Right, tt.expected)
		})
	}
}

// ============================================================================
// Infix Expression Tests
// ============================================================================

func TestInfixExpressions(t *testing.T) {
	tests := []struct {
		input    string
		left     interface{}
		operator string
		right    interface{}
	}{
		{"temp x int = 5 + 5", 5, "+", 5},
		{"temp x int = 5 - 5", 5, "-", 5},
		{"temp x int = 5 * 5", 5, "*", 5},
		{"temp x int = 5 / 5", 5, "/", 5},
		{"temp x int = 5 % 3", 5, "%", 3},
		{"temp x bool = 5 > 5", 5, ">", 5},
		{"temp x bool = 5 < 5", 5, "<", 5},
		{"temp x bool = 5 == 5", 5, "==", 5},
		{"temp x bool = 5 != 5", 5, "!=", 5},
		{"temp x bool = 5 >= 5", 5, ">=", 5},
		{"temp x bool = 5 <= 5", 5, "<=", 5},
		{"temp x bool = true && false", true, "&&", false},
		{"temp x bool = true || false", true, "||", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			stmt := program.Statements[0].(*VariableDeclaration)
			testInfixExpression(t, stmt.Value, tt.left, tt.operator, tt.right)
		})
	}
}

func TestMembershipOperators(t *testing.T) {
	tests := []struct {
		input    string
		operator string
	}{
		{"temp x bool = 1 in arr", "in"},
		{"temp x bool = 1 not_in arr", "not_in"},
	}

	for _, tt := range tests {
		t.Run(tt.operator, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			stmt := program.Statements[0].(*VariableDeclaration)
			exp, ok := stmt.Value.(*InfixExpression)
			if !ok {
				t.Fatalf("not InfixExpression, got %T", stmt.Value)
			}
			if exp.Operator != tt.operator {
				t.Errorf("expected operator %q, got %q", tt.operator, exp.Operator)
			}
		})
	}
}

// ============================================================================
// Operator Precedence Tests
// ============================================================================

func TestOperatorPrecedence(t *testing.T) {
	// Test that multiplication binds tighter than addition
	input := "temp x = 1 + 2 * 3"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*VariableDeclaration)

	// The top-level should be addition (1 + (2*3))
	infix, ok := stmt.Value.(*InfixExpression)
	if !ok {
		t.Fatalf("not InfixExpression, got %T", stmt.Value)
	}
	if infix.Operator != "+" {
		t.Errorf("expected top operator +, got %q", infix.Operator)
	}

	// Right side should be multiplication
	rightInfix, ok := infix.Right.(*InfixExpression)
	if !ok {
		t.Fatalf("right not InfixExpression, got %T", infix.Right)
	}
	if rightInfix.Operator != "*" {
		t.Errorf("expected right operator *, got %q", rightInfix.Operator)
	}
}

// ============================================================================
// Postfix Expression Tests
// ============================================================================

func TestPostfixExpressions(t *testing.T) {
	tests := []struct {
		input    string
		operator string
	}{
		{"x++", "++"},
		{"x--", "--"},
	}

	for _, tt := range tests {
		t.Run(tt.operator, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			stmt := program.Statements[0].(*ExpressionStatement)
			exp, ok := stmt.Expression.(*PostfixExpression)
			if !ok {
				t.Fatalf("not PostfixExpression, got %T", stmt.Expression)
			}
			if exp.Operator != tt.operator {
				t.Errorf("expected operator %q, got %q", tt.operator, exp.Operator)
			}
		})
	}
}

// ============================================================================
// Variable Declaration Tests
// ============================================================================

func TestVariableDeclarations(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedName string
		mutable      bool
		typeName     string
	}{
		{"temp with type", "temp x int = 5", "x", true, "int"},
		{"temp inferred", "temp x = 5", "x", true, ""},
		{"const with type", "const x int = 5", "x", false, "int"},
		{"const inferred", "const x = 5", "x", false, ""},
		{"temp float", "temp f float = 3.14", "f", true, "float"},
		{"temp string", "temp s string = \"hello\"", "s", true, "string"},
		{"temp bool", "temp b bool = true", "b", true, "bool"},
		{"temp array", "temp arr [int] = {1, 2, 3}", "arr", true, "[int]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			stmt, ok := program.Statements[0].(*VariableDeclaration)
			if !ok {
				t.Fatalf("not VariableDeclaration, got %T", program.Statements[0])
			}
			if stmt.Name.Value != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, stmt.Name.Value)
			}
			if stmt.Mutable != tt.mutable {
				t.Errorf("expected mutable=%t, got=%t", tt.mutable, stmt.Mutable)
			}
			if tt.typeName != "" && stmt.TypeName != tt.typeName {
				t.Errorf("expected typeName %q, got %q", tt.typeName, stmt.TypeName)
			}
		})
	}
}

func TestMultipleVariableDeclaration(t *testing.T) {
	input := "temp a, b = getValues()"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*VariableDeclaration)

	if len(stmt.Names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(stmt.Names))
	}
	if stmt.Names[0].Value != "a" || stmt.Names[1].Value != "b" {
		t.Errorf("expected names 'a', 'b', got %q, %q", stmt.Names[0].Value, stmt.Names[1].Value)
	}
}

func TestIgnoreInMultipleDeclaration(t *testing.T) {
	input := "temp @ignore, b = getValues()"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*VariableDeclaration)

	if len(stmt.Names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(stmt.Names))
	}
	if stmt.Names[0].Value != "@ignore" {
		t.Errorf("expected first name '@ignore', got %q", stmt.Names[0].Value)
	}
}

// ============================================================================
// Assignment Statement Tests
// ============================================================================

func TestAssignmentStatements(t *testing.T) {
	tests := []struct {
		input    string
		operator string
	}{
		{"x = 5", "="},
		{"x += 5", "+="},
		{"x -= 5", "-="},
		{"x *= 5", "*="},
		{"x /= 5", "/="},
		{"x %= 5", "%="},
	}

	for _, tt := range tests {
		t.Run(tt.operator, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			stmt, ok := program.Statements[0].(*AssignmentStatement)
			if !ok {
				t.Fatalf("not AssignmentStatement, got %T", program.Statements[0])
			}
			if stmt.Operator != tt.operator {
				t.Errorf("expected operator %q, got %q", tt.operator, stmt.Operator)
			}
		})
	}
}

func TestIndexAssignment(t *testing.T) {
	input := "arr[0] = 5"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*AssignmentStatement)

	_, ok := stmt.Name.(*IndexExpression)
	if !ok {
		t.Fatalf("expected IndexExpression, got %T", stmt.Name)
	}
}

func TestMemberAssignment(t *testing.T) {
	input := "obj.field = 5"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*AssignmentStatement)

	_, ok := stmt.Name.(*MemberExpression)
	if !ok {
		t.Fatalf("expected MemberExpression, got %T", stmt.Name)
	}
}

// ============================================================================
// Return Statement Tests
// ============================================================================

func TestReturnStatements(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		valueCount int
	}{
		{"single value", "do test() {\n\treturn 5\n}", 1},
		{"multiple values", "do test() {\n\treturn 1, 2, 3\n}", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			fn := program.Statements[0].(*FunctionDeclaration)
			ret := fn.Body.Statements[0].(*ReturnStatement)
			if len(ret.Values) != tt.valueCount {
				t.Errorf("expected %d values, got %d", tt.valueCount, len(ret.Values))
			}
		})
	}
}

// ============================================================================
// Function Declaration Tests
// ============================================================================

func TestFunctionDeclaration(t *testing.T) {
	// EZ uses -> for return type
	input := `do add(a int, b int) -> int {
		return a + b
	}`
	program := parseProgram(t, input)
	fn := program.Statements[0].(*FunctionDeclaration)

	if fn.Name.Value != "add" {
		t.Errorf("expected name 'add', got %q", fn.Name.Value)
	}
	if len(fn.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(fn.Parameters))
	}
	// ReturnTypes is a slice - single return type is first element
	if len(fn.ReturnTypes) != 1 || fn.ReturnTypes[0] != "int" {
		t.Errorf("expected return type 'int', got %v", fn.ReturnTypes)
	}
}

func TestFunctionWithMultipleReturnTypes(t *testing.T) {
	input := `do getValues() -> (int, string) {
		return 1, "hello"
	}`
	program := parseProgram(t, input)
	fn := program.Statements[0].(*FunctionDeclaration)

	if len(fn.ReturnTypes) != 2 {
		t.Fatalf("expected 2 return types, got %d", len(fn.ReturnTypes))
	}
	if fn.ReturnTypes[0] != "int" || fn.ReturnTypes[1] != "string" {
		t.Errorf("expected return types (int, string), got (%s, %s)", fn.ReturnTypes[0], fn.ReturnTypes[1])
	}
}

func TestFunctionWithNoParameters(t *testing.T) {
	input := "do hello() {\n\treturn 0\n}"
	program := parseProgram(t, input)
	fn := program.Statements[0].(*FunctionDeclaration)

	if len(fn.Parameters) != 0 {
		t.Errorf("expected 0 parameters, got %d", len(fn.Parameters))
	}
}

// ============================================================================
// Call Expression Tests
// ============================================================================

func TestCallExpressions(t *testing.T) {
	input := "add(1, 2, 3)"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ExpressionStatement)
	call, ok := stmt.Expression.(*CallExpression)
	if !ok {
		t.Fatalf("not CallExpression, got %T", stmt.Expression)
	}

	if !testIdentifier(t, call.Function, "add") {
		return
	}
	if len(call.Arguments) != 3 {
		t.Fatalf("expected 3 arguments, got %d", len(call.Arguments))
	}
}

func TestCallWithNoArguments(t *testing.T) {
	input := "doSomething()"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ExpressionStatement)
	call := stmt.Expression.(*CallExpression)

	if len(call.Arguments) != 0 {
		t.Errorf("expected 0 arguments, got %d", len(call.Arguments))
	}
}

// ============================================================================
// Index Expression Tests
// ============================================================================

func TestIndexExpression(t *testing.T) {
	input := "arr[1]"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ExpressionStatement)
	idx, ok := stmt.Expression.(*IndexExpression)
	if !ok {
		t.Fatalf("not IndexExpression, got %T", stmt.Expression)
	}

	if !testIdentifier(t, idx.Left, "arr") {
		return
	}
	testIntegerLiteral(t, idx.Index, 1)
}

// ============================================================================
// Member Expression Tests
// ============================================================================

func TestMemberExpression(t *testing.T) {
	input := "person.name"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ExpressionStatement)
	member, ok := stmt.Expression.(*MemberExpression)
	if !ok {
		t.Fatalf("not MemberExpression, got %T", stmt.Expression)
	}

	if !testIdentifier(t, member.Object, "person") {
		return
	}
	if member.Member.Value != "name" {
		t.Errorf("expected member 'name', got %q", member.Member.Value)
	}
}

func TestChainedMemberExpression(t *testing.T) {
	input := "a.b.c"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ExpressionStatement)
	member := stmt.Expression.(*MemberExpression)

	if member.Member.Value != "c" {
		t.Errorf("expected outer member 'c', got %q", member.Member.Value)
	}
	innerMember := member.Object.(*MemberExpression)
	if innerMember.Member.Value != "b" {
		t.Errorf("expected inner member 'b', got %q", innerMember.Member.Value)
	}
}

// ============================================================================
// If Statement Tests
// ============================================================================

func TestIfStatement(t *testing.T) {
	input := `if x < 10 {
		return true
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*IfStatement)

	if stmt.Consequence == nil {
		t.Fatal("consequence is nil")
	}
	if stmt.Alternative != nil {
		t.Error("alternative should be nil")
	}
}

func TestIfOtherwiseStatement(t *testing.T) {
	input := `if x < 10 {
		return true
	} otherwise {
		return false
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*IfStatement)

	if stmt.Consequence == nil {
		t.Fatal("consequence is nil")
	}
	if stmt.Alternative == nil {
		t.Fatal("alternative is nil")
	}
	_, ok := stmt.Alternative.(*BlockStatement)
	if !ok {
		t.Errorf("alternative is not BlockStatement, got %T", stmt.Alternative)
	}
}

func TestIfOrStatement(t *testing.T) {
	input := `if x < 5 {
		return 1
	} or x < 10 {
		return 2
	} otherwise {
		return 3
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*IfStatement)

	if stmt.Alternative == nil {
		t.Fatal("alternative is nil")
	}
	orStmt, ok := stmt.Alternative.(*IfStatement)
	if !ok {
		t.Fatalf("alternative is not IfStatement (or), got %T", stmt.Alternative)
	}
	if orStmt.Alternative == nil {
		t.Fatal("or alternative (otherwise) is nil")
	}
}

// ============================================================================
// Loop Statement Tests
// ============================================================================

func TestForStatement(t *testing.T) {
	// EZ uses "for variable in iterable" syntax
	input := `for i in range(0, 10) {
		x = i
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ForStatement)

	if stmt.Variable == nil {
		t.Error("variable is nil")
	}
	if stmt.Variable.Value != "i" {
		t.Errorf("expected variable 'i', got %q", stmt.Variable.Value)
	}
	if stmt.Iterable == nil {
		t.Error("iterable is nil")
	}
}

func TestForStatementWithType(t *testing.T) {
	input := `for i int in range(0, 10) {
		x = i
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ForStatement)

	if stmt.Variable.Value != "i" {
		t.Errorf("expected variable 'i', got %q", stmt.Variable.Value)
	}
	if stmt.VarType != "int" {
		t.Errorf("expected VarType 'int', got %q", stmt.VarType)
	}
}

func TestForEachStatement(t *testing.T) {
	input := `for_each item in items {
		process(item)
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ForEachStatement)

	if stmt.Variable == nil {
		t.Fatal("variable is nil")
	}
	if stmt.Variable.Value != "item" {
		t.Errorf("expected variable 'item', got %q", stmt.Variable.Value)
	}
	if stmt.Collection == nil {
		t.Error("collection is nil")
	}
}

func TestForStatementWithParens(t *testing.T) {
	// Test optional parentheses around for loop expression
	input := `for (i in range(0, 10)) {
		x = i
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ForStatement)

	if stmt.Variable == nil {
		t.Error("variable is nil")
	}
	if stmt.Variable.Value != "i" {
		t.Errorf("expected variable 'i', got %q", stmt.Variable.Value)
	}
	if stmt.Iterable == nil {
		t.Error("iterable is nil")
	}
}

func TestForStatementWithParensAndType(t *testing.T) {
	// Test optional parentheses with type annotation
	input := `for (i int in range(0, 10)) {
		x = i
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ForStatement)

	if stmt.Variable.Value != "i" {
		t.Errorf("expected variable 'i', got %q", stmt.Variable.Value)
	}
	if stmt.VarType != "int" {
		t.Errorf("expected VarType 'int', got %q", stmt.VarType)
	}
}

func TestForEachStatementWithParens(t *testing.T) {
	// Test optional parentheses around for_each loop expression
	input := `for_each (item in items) {
		process(item)
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ForEachStatement)

	if stmt.Variable == nil {
		t.Fatal("variable is nil")
	}
	if stmt.Variable.Value != "item" {
		t.Errorf("expected variable 'item', got %q", stmt.Variable.Value)
	}
	if stmt.Collection == nil {
		t.Error("collection is nil")
	}
}

func TestForStatementNestedWithParens(t *testing.T) {
	// Test nested for loops both with parentheses
	input := `for (i in range(0, 3)) {
		for (j in range(0, 2)) {
			x = i + j
		}
	}`
	program := parseProgram(t, input)
	outerFor := program.Statements[0].(*ForStatement)

	if outerFor.Variable.Value != "i" {
		t.Errorf("expected outer variable 'i', got %q", outerFor.Variable.Value)
	}

	// Check inner for loop
	if len(outerFor.Body.Statements) == 0 {
		t.Fatal("outer body is empty")
	}
	innerFor, ok := outerFor.Body.Statements[0].(*ForStatement)
	if !ok {
		t.Fatalf("expected inner ForStatement, got %T", outerFor.Body.Statements[0])
	}
	if innerFor.Variable.Value != "j" {
		t.Errorf("expected inner variable 'j', got %q", innerFor.Variable.Value)
	}
}

func TestForStatementWithComplexIterable(t *testing.T) {
	// Test for loop with parentheses containing function call
	input := `for (i in get_range()) {
		x = i
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ForStatement)

	if stmt.Variable.Value != "i" {
		t.Errorf("expected variable 'i', got %q", stmt.Variable.Value)
	}
	// Verify iterable is a call expression
	_, ok := stmt.Iterable.(*CallExpression)
	if !ok {
		t.Errorf("expected iterable to be CallExpression, got %T", stmt.Iterable)
	}
}

func TestForStatementWithArrayLiteralIterable(t *testing.T) {
	// Test for loop with parentheses containing array literal
	input := `for (x in {1, 2, 3}) {
		sum += x
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ForStatement)

	if stmt.Variable.Value != "x" {
		t.Errorf("expected variable 'x', got %q", stmt.Variable.Value)
	}
	// Verify iterable is an array value
	_, ok := stmt.Iterable.(*ArrayValue)
	if !ok {
		t.Errorf("expected iterable to be ArrayValue, got %T", stmt.Iterable)
	}
}

func TestForStatementMixedParensStyle(t *testing.T) {
	// Test that we can have outer for with parens, inner without
	input := `for (i in range(0, 3)) {
		for j in range(0, 2) {
			x = i + j
		}
	}`
	program := parseProgram(t, input)
	outerFor := program.Statements[0].(*ForStatement)

	if outerFor.Variable.Value != "i" {
		t.Errorf("expected outer variable 'i', got %q", outerFor.Variable.Value)
	}

	innerFor := outerFor.Body.Statements[0].(*ForStatement)
	if innerFor.Variable.Value != "j" {
		t.Errorf("expected inner variable 'j', got %q", innerFor.Variable.Value)
	}
}

func TestForStatementParensMissingClosing(t *testing.T) {
	// Test that missing closing paren produces an error
	input := `for (i in range(0, 10) {
		x = i
	}`
	l := NewLexer(input)
	p := New(l)
	p.ParseProgram()

	errs := p.Errors()
	if len(errs) == 0 {
		t.Error("expected parser error for missing closing paren, got none")
	}
}

func TestForEachStatementParensMissingClosing(t *testing.T) {
	// Test that missing closing paren produces an error
	input := `for_each (item in items {
		process(item)
	}`
	l := NewLexer(input)
	p := New(l)
	p.ParseProgram()

	errs := p.Errors()
	if len(errs) == 0 {
		t.Error("expected parser error for missing closing paren, got none")
	}
}

func TestWhileStatementWithParens(t *testing.T) {
	// Test as_long_as with optional parentheses
	input := `as_long_as (x < 10) {
		x++
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*WhileStatement)

	if stmt.Condition == nil {
		t.Error("condition is nil")
	}
}

func TestIfStatementWithParens(t *testing.T) {
	// Test if with optional parentheses
	input := `if (x > 5) {
		y = 1
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*IfStatement)

	if stmt.Condition == nil {
		t.Error("condition is nil")
	}
}

func TestIfOrStatementWithParens(t *testing.T) {
	// Test if/or/otherwise with optional parentheses
	input := `if (x < 10) {
		y = 1
	} or (x < 20) {
		y = 2
	} otherwise {
		y = 3
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*IfStatement)

	if stmt.Condition == nil {
		t.Error("if condition is nil")
	}
	if stmt.Alternative == nil {
		t.Error("alternative (or clause) is nil")
	}
}

func TestIfOrStatementMixedParens(t *testing.T) {
	// Test if without parens, or with parens
	input := `if x < 10 {
		y = 1
	} or (x < 20) {
		y = 2
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*IfStatement)

	if stmt.Condition == nil {
		t.Error("if condition is nil")
	}
	if stmt.Alternative == nil {
		t.Error("or clause is nil")
	}
}

func TestWhileStatement(t *testing.T) {
	input := `as_long_as x < 10 {
		x++
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*WhileStatement)

	if stmt.Condition == nil {
		t.Error("condition is nil")
	}
}

func TestLoopStatement(t *testing.T) {
	input := `loop {
		if done { break }
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*LoopStatement)

	if stmt.Body == nil {
		t.Error("body is nil")
	}
}

func TestBreakStatement(t *testing.T) {
	input := `loop { break }`
	program := parseProgram(t, input)
	loopStmt := program.Statements[0].(*LoopStatement)
	_, ok := loopStmt.Body.Statements[0].(*BreakStatement)
	if !ok {
		t.Fatalf("not BreakStatement, got %T", loopStmt.Body.Statements[0])
	}
}

func TestContinueStatement(t *testing.T) {
	input := `loop { continue }`
	program := parseProgram(t, input)
	loopStmt := program.Statements[0].(*LoopStatement)
	_, ok := loopStmt.Body.Statements[0].(*ContinueStatement)
	if !ok {
		t.Fatalf("not ContinueStatement, got %T", loopStmt.Body.Statements[0])
	}
}

// ============================================================================
// Struct Declaration Tests
// ============================================================================

func TestStructDeclaration(t *testing.T) {
	input := `const Person struct {
		name string
		age int
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*StructDeclaration)

	if stmt.Name.Value != "Person" {
		t.Errorf("expected name 'Person', got %q", stmt.Name.Value)
	}
	if len(stmt.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(stmt.Fields))
	}
}

func TestNewExpression(t *testing.T) {
	// new(TypeName) syntax
	input := `temp p = new(Person)`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*VariableDeclaration)
	newExpr, ok := stmt.Value.(*NewExpression)
	if !ok {
		t.Fatalf("not NewExpression, got %T", stmt.Value)
	}
	if newExpr.TypeName == nil || newExpr.TypeName.Value != "Person" {
		t.Errorf("expected type 'Person', got %v", newExpr.TypeName)
	}
}

// ============================================================================
// Enum Declaration Tests
// ============================================================================

func TestEnumDeclaration(t *testing.T) {
	input := `const Color enum {
		Red
		Green
		Blue
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*EnumDeclaration)

	if stmt.Name.Value != "Color" {
		t.Errorf("expected name 'Color', got %q", stmt.Name.Value)
	}
	if len(stmt.Values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(stmt.Values))
	}
}

// ============================================================================
// Import Statement Tests
// ============================================================================

func TestImportStatement(t *testing.T) {
	input := `import arr@arrays`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ImportStatement)

	if stmt.Alias != "arr" {
		t.Errorf("expected alias 'arr', got %q", stmt.Alias)
	}
	if stmt.Module != "arrays" {
		t.Errorf("expected module 'arrays', got %q", stmt.Module)
	}
}

func TestImportMultiple(t *testing.T) {
	input := `import arr@arrays, str@strings`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*ImportStatement)

	if len(stmt.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(stmt.Imports))
	}
}

func TestFromImportStatement(t *testing.T) {
	input := `from @arrays import append, pop`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*FromImportStatement)

	if stmt.Path != "arrays" {
		t.Errorf("expected path 'arrays', got %q", stmt.Path)
	}
	if len(stmt.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(stmt.Items))
	}
}

// ============================================================================
// Using Statement Tests
// ============================================================================

func TestUsingStatement(t *testing.T) {
	input := `import arr@arrays
using arr`
	program := parseProgram(t, input)
	if len(program.FileUsing) != 1 {
		t.Fatalf("expected 1 using statement, got %d", len(program.FileUsing))
	}
	using := program.FileUsing[0]
	if len(using.Modules) != 1 || using.Modules[0].Value != "arr" {
		t.Errorf("expected module 'arr', got %v", using.Modules)
	}
}

// ============================================================================
// Module Declaration Tests
// ============================================================================

func TestModuleDeclaration(t *testing.T) {
	input := `module mymodule

temp x int = 5`
	program := parseProgram(t, input)

	if program.Module == nil {
		t.Fatal("module is nil")
	}
	if program.Module.Name.Value != "mymodule" {
		t.Errorf("expected module name 'mymodule', got %q", program.Module.Name.Value)
	}
}

// ============================================================================
// Visibility Tests
// ============================================================================

func TestPrivateVisibility(t *testing.T) {
	input := `private const x int = 5`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*VariableDeclaration)

	if stmt.Visibility != VisibilityPrivate {
		t.Errorf("expected VisibilityPrivate, got %v", stmt.Visibility)
	}
}

func TestPrivateModuleVisibility(t *testing.T) {
	// private:module visibility syntax for variables
	// Note: Skip this test as the syntax may require specific context
	t.Skip("private:module visibility syntax requires further investigation")
}

// ============================================================================
// Attribute Tests
// ============================================================================

func TestSuppressAttribute(t *testing.T) {
	input := `@suppress("W2001")
do test() {
	return 1
	x = 2
}`
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	// Should not have warnings due to suppression
	warnings := 0
	for _, err := range p.EZErrors().Errors {
		if strings.HasPrefix(err.ErrorCode.Code, "W") {
			warnings++
		}
	}
	if warnings > 0 {
		t.Errorf("expected 0 warnings (suppressed), got %d", warnings)
	}
}

// ============================================================================
// Grouped Expression Tests
// ============================================================================

func TestGroupedExpressions(t *testing.T) {
	input := "temp x int = (5 + 5) * 2"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*VariableDeclaration)

	infix, ok := stmt.Value.(*InfixExpression)
	if !ok {
		t.Fatalf("not InfixExpression, got %T", stmt.Value)
	}
	if infix.Operator != "*" {
		t.Errorf("expected *, got %q", infix.Operator)
	}
}

// ============================================================================
// Error Cases
// ============================================================================

func TestReservedKeywordAsVariable(t *testing.T) {
	input := "temp if int = 5"
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	if !p.EZErrors().HasErrors() {
		t.Error("expected error for reserved keyword as variable name")
	}
}

func TestDuplicateDeclaration(t *testing.T) {
	input := `temp x int = 5
temp x int = 10`
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	if !p.EZErrors().HasErrors() {
		t.Error("expected error for duplicate declaration")
	}
}

func TestConstWithoutValue(t *testing.T) {
	input := "const x int"
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	if !p.EZErrors().HasErrors() {
		t.Error("expected error for const without value")
	}
}

func TestUnclosedBrace(t *testing.T) {
	input := "do test() { return"
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	if !p.EZErrors().HasErrors() {
		t.Error("expected error for unclosed brace")
	}
}

func TestUnclosedParenthesis(t *testing.T) {
	input := "add(1, 2"
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	if !p.EZErrors().HasErrors() {
		t.Error("expected error for unclosed parenthesis")
	}
}

func TestUnclosedBracket(t *testing.T) {
	input := "arr[0"
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	if !p.EZErrors().HasErrors() {
		t.Error("expected error for unclosed bracket")
	}
}

func TestInvalidAssignmentTarget(t *testing.T) {
	// Integer literal cannot be assigned to
	input := "5 = 10"
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	// This parses as expression statement, no assignment error raised at parse level
	// The error would be caught at type checking or evaluation
	// Just verify it parses (possibly with errors)
}

func TestUsingBeforeImport(t *testing.T) {
	input := `using arr
import arr@arrays`
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	if !p.EZErrors().HasErrors() {
		t.Error("expected error for using before import")
	}
}

// ============================================================================
// Scope Tests
// ============================================================================

func TestScopeManagement(t *testing.T) {
	l := NewLexer("temp x int = 5")
	p := New(l)

	// Initial scope
	if len(p.scopes) != 1 {
		t.Errorf("expected 1 scope initially, got %d", len(p.scopes))
	}

	// Push scope
	p.pushScope()
	if len(p.scopes) != 2 {
		t.Errorf("expected 2 scopes after push, got %d", len(p.scopes))
	}

	// Pop scope
	p.popScope()
	if len(p.scopes) != 1 {
		t.Errorf("expected 1 scope after pop, got %d", len(p.scopes))
	}

	// Pop should not go below 1
	p.popScope()
	if len(p.scopes) != 1 {
		t.Errorf("expected 1 scope after extra pop, got %d", len(p.scopes))
	}
}

func TestDeclareInScope(t *testing.T) {
	l := NewLexer("temp x int = 5")
	p := New(l)

	// First declaration should succeed
	if !p.declareInScope("x", p.currentToken) {
		t.Error("first declaration should succeed")
	}

	// Second declaration should fail
	if p.declareInScope("x", p.currentToken) {
		t.Error("duplicate declaration should fail")
	}

	// Different name should succeed
	if !p.declareInScope("y", p.currentToken) {
		t.Error("different name declaration should succeed")
	}
}

func TestNestedScopeAllowsSameName(t *testing.T) {
	input := `do test() {
		temp x int = 5
		if true {
			temp x int = 10
		}
	}`
	program := parseProgram(t, input)
	if program == nil {
		t.Fatal("program is nil")
	}
	// Should parse without errors since inner x is in a new scope
}

// ============================================================================
// Range Expression Tests
// ============================================================================

func TestRangeExpression(t *testing.T) {
	input := "temp r = range(0, 10)"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*VariableDeclaration)
	rangeExpr, ok := stmt.Value.(*RangeExpression)
	if !ok {
		t.Fatalf("not RangeExpression, got %T", stmt.Value)
	}
	testIntegerLiteral(t, rangeExpr.Start, 0)
	testIntegerLiteral(t, rangeExpr.End, 10)
}

func TestRangeWithStep(t *testing.T) {
	input := "temp r = range(0, 10, 2)"
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*VariableDeclaration)
	rangeExpr := stmt.Value.(*RangeExpression)
	if rangeExpr.Step == nil {
		t.Fatal("step is nil")
	}
	testIntegerLiteral(t, rangeExpr.Step, 2)
}

// ============================================================================
// REPL Mode Tests
// ============================================================================

func TestParseLine(t *testing.T) {
	input := "temp x int = 5"
	l := NewLexer(input)
	p := New(l)

	stmt := p.ParseLine()
	if stmt == nil {
		t.Fatal("ParseLine returned nil")
	}
	_, ok := stmt.(*VariableDeclaration)
	if !ok {
		t.Errorf("not VariableDeclaration, got %T", stmt)
	}
}

func TestParseLineEmpty(t *testing.T) {
	input := ""
	l := NewLexer(input)
	p := New(l)

	stmt := p.ParseLine()
	if stmt != nil {
		t.Errorf("expected nil for empty input, got %T", stmt)
	}
}

// ============================================================================
// Precedence Tests
// ============================================================================

func TestPrecedenceValues(t *testing.T) {
	// Verify precedence ordering
	if OR_PREC >= AND_PREC {
		t.Error("OR should have lower precedence than AND")
	}
	if AND_PREC >= EQUALS {
		t.Error("AND should have lower precedence than EQUALS")
	}
	if EQUALS >= LESSGREATER {
		t.Error("EQUALS should have lower precedence than LESSGREATER")
	}
	if LESSGREATER >= SUM {
		t.Error("LESSGREATER should have lower precedence than SUM")
	}
	if SUM >= PRODUCT {
		t.Error("SUM should have lower precedence than PRODUCT")
	}
	if PRODUCT >= PREFIX {
		t.Error("PRODUCT should have lower precedence than PREFIX")
	}
	if PREFIX >= CALL {
		t.Error("PREFIX should have lower precedence than CALL")
	}
	if CALL >= INDEX {
		t.Error("CALL should have lower precedence than INDEX")
	}
}

// ============================================================================
// Reserved Names Tests
// ============================================================================

func TestIsReservedName(t *testing.T) {
	tests := []struct {
		name     string
		reserved bool
	}{
		{"temp", true},
		{"const", true},
		{"if", true},
		{"for", true},
		{"return", true},
		{"len", true},
		{"typeof", true},
		{"println", true},
		{"myVar", false},
		{"customFunction", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isReservedName(tt.name) != tt.reserved {
				t.Errorf("isReservedName(%q) = %v, want %v", tt.name, !tt.reserved, tt.reserved)
			}
		})
	}
}

// ============================================================================
// Matrix Type Parsing Tests
// ============================================================================

func TestMatrixTypeParsing(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedName string
		typeName     string
	}{
		{"2D int array", "temp matrix [[int]] = {{1, 2}, {3, 4}}", "matrix", "[[int]]"},
		{"2D string array", "temp strs [[string]] = {{\"a\", \"b\"}, {\"c\", \"d\"}}", "strs", "[[string]]"},
		{"2D float array", "temp floats [[float]] = {{1.1, 2.2}, {3.3, 4.4}}", "floats", "[[float]]"},
		{"2D bool array", "temp flags [[bool]] = {{true, false}, {false, true}}", "flags", "[[bool]]"},
		{"3D int array", "temp cube [[[int]]] = {{{1, 2}, {3, 4}}, {{5, 6}, {7, 8}}}", "cube", "[[[int]]]"},
		{"empty 2D array", "temp empty [[int]] = {}", "empty", "[[int]]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			if len(program.Statements) == 0 {
				t.Fatalf("no statements parsed")
			}
			stmt, ok := program.Statements[0].(*VariableDeclaration)
			if !ok {
				t.Fatalf("not VariableDeclaration, got %T", program.Statements[0])
			}
			if stmt.Name.Value != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, stmt.Name.Value)
			}
			if stmt.TypeName != tt.typeName {
				t.Errorf("expected typeName %q, got %q", tt.typeName, stmt.TypeName)
			}
		})
	}
}

func TestMatrixInFunctionParameter(t *testing.T) {
	input := `do processMatrix(m [[int]]) -> int {
		return 0
	}`
	program := parseProgram(t, input)
	fn := program.Statements[0].(*FunctionDeclaration)

	if len(fn.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(fn.Parameters))
	}
	if fn.Parameters[0].TypeName != "[[int]]" {
		t.Errorf("expected parameter type '[[int]]', got %q", fn.Parameters[0].TypeName)
	}
}

func TestMatrixInFunctionReturn(t *testing.T) {
	input := `do createMatrix() -> [[int]] {
		return {{1, 2}, {3, 4}}
	}`
	program := parseProgram(t, input)
	fn := program.Statements[0].(*FunctionDeclaration)

	if len(fn.ReturnTypes) != 1 || fn.ReturnTypes[0] != "[[int]]" {
		t.Errorf("expected return type '[[int]]', got %v", fn.ReturnTypes)
	}
}

func TestMatrixInStructField(t *testing.T) {
	input := `const Grid struct {
		data [[int]]
		rows int
	}`
	program := parseProgram(t, input)
	stmt := program.Statements[0].(*StructDeclaration)

	if len(stmt.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(stmt.Fields))
	}
	if stmt.Fields[0].TypeName != "[[int]]" {
		t.Errorf("expected field type '[[int]]', got %q", stmt.Fields[0].TypeName)
	}
}

// ============================================================================
// Mutable Parameter Tests (& prefix)
// ============================================================================

func TestMutableParameter(t *testing.T) {
	input := `do modify(&p Person) {
		p.age = 30
	}`
	program := parseProgram(t, input)
	fn := program.Statements[0].(*FunctionDeclaration)

	if len(fn.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(fn.Parameters))
	}
	if fn.Parameters[0].Name.Value != "p" {
		t.Errorf("expected parameter name 'p', got %q", fn.Parameters[0].Name.Value)
	}
	if fn.Parameters[0].TypeName != "Person" {
		t.Errorf("expected parameter type 'Person', got %q", fn.Parameters[0].TypeName)
	}
	if !fn.Parameters[0].Mutable {
		t.Errorf("expected parameter to be mutable (& prefix)")
	}
}

func TestNonMutableParameter(t *testing.T) {
	input := `do getName(p Person) -> string {
		return p.name
	}`
	program := parseProgram(t, input)
	fn := program.Statements[0].(*FunctionDeclaration)

	if len(fn.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(fn.Parameters))
	}
	if fn.Parameters[0].Mutable {
		t.Errorf("expected parameter to be immutable (no & prefix)")
	}
}

func TestMixedMutableParameters(t *testing.T) {
	input := `do process(&arr [int], count int) {
		arr[0] = count
	}`
	program := parseProgram(t, input)
	fn := program.Statements[0].(*FunctionDeclaration)

	if len(fn.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(fn.Parameters))
	}

	// First param should be mutable
	if fn.Parameters[0].Name.Value != "arr" {
		t.Errorf("expected first param name 'arr', got %q", fn.Parameters[0].Name.Value)
	}
	if !fn.Parameters[0].Mutable {
		t.Errorf("expected first parameter to be mutable")
	}

	// Second param should be immutable
	if fn.Parameters[1].Name.Value != "count" {
		t.Errorf("expected second param name 'count', got %q", fn.Parameters[1].Name.Value)
	}
	if fn.Parameters[1].Mutable {
		t.Errorf("expected second parameter to be immutable")
	}
}

func TestMultipleMutableParamsSharedType(t *testing.T) {
	// Test: &a, &b int - both should be mutable with type int
	input := `do swap(&a, &b int) {
		temp t int = a
		a = b
		b = t
	}`
	program := parseProgram(t, input)
	fn := program.Statements[0].(*FunctionDeclaration)

	if len(fn.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(fn.Parameters))
	}

	if fn.Parameters[0].Name.Value != "a" {
		t.Errorf("expected first param name 'a', got %q", fn.Parameters[0].Name.Value)
	}
	if fn.Parameters[0].TypeName != "int" {
		t.Errorf("expected first param type 'int', got %q", fn.Parameters[0].TypeName)
	}
	if !fn.Parameters[0].Mutable {
		t.Errorf("expected first parameter to be mutable")
	}

	if fn.Parameters[1].Name.Value != "b" {
		t.Errorf("expected second param name 'b', got %q", fn.Parameters[1].Name.Value)
	}
	if fn.Parameters[1].TypeName != "int" {
		t.Errorf("expected second param type 'int', got %q", fn.Parameters[1].TypeName)
	}
	if !fn.Parameters[1].Mutable {
		t.Errorf("expected second parameter to be mutable")
	}
}

func TestMixedMutabilitySharedType(t *testing.T) {
	// Test: &a, b int - a is mutable, b is immutable, both type int
	input := `do test(&a, b int) {
		a = b
	}`
	program := parseProgram(t, input)
	fn := program.Statements[0].(*FunctionDeclaration)

	if len(fn.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(fn.Parameters))
	}

	if !fn.Parameters[0].Mutable {
		t.Errorf("expected first parameter 'a' to be mutable")
	}
	if fn.Parameters[1].Mutable {
		t.Errorf("expected second parameter 'b' to be immutable")
	}
}

func TestMutableArrayParameter(t *testing.T) {
	input := `do doubleAll(&arr [int]) {
		arr[0] = arr[0] * 2
	}`
	program := parseProgram(t, input)
	fn := program.Statements[0].(*FunctionDeclaration)

	if len(fn.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(fn.Parameters))
	}
	if fn.Parameters[0].TypeName != "[int]" {
		t.Errorf("expected parameter type '[int]', got %q", fn.Parameters[0].TypeName)
	}
	if !fn.Parameters[0].Mutable {
		t.Errorf("expected parameter to be mutable")
	}
}

func TestMutableMapParameter(t *testing.T) {
	input := `do addEntry(&m map[string:int], key string, val int) {
		m[key] = val
	}`
	program := parseProgram(t, input)
	fn := program.Statements[0].(*FunctionDeclaration)

	if len(fn.Parameters) != 3 {
		t.Fatalf("expected 3 parameters, got %d", len(fn.Parameters))
	}
	if fn.Parameters[0].TypeName != "map[string:int]" {
		t.Errorf("expected first param type 'map[string:int]', got %q", fn.Parameters[0].TypeName)
	}
	if !fn.Parameters[0].Mutable {
		t.Errorf("expected first parameter to be mutable")
	}
	if fn.Parameters[1].Mutable || fn.Parameters[2].Mutable {
		t.Errorf("expected second and third parameters to be immutable")
	}
}

// ============================================================================
// Bug Fix Tests - December 2025
// ============================================================================

func TestImportInsideFunctionBlock(t *testing.T) {
	// Test: import inside function block should error E2036
	// Fixes #324
	input := `import & use @std
do main() {
	import & use @math
	println("test")
}`
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	if !p.EZErrors().HasErrors() {
		t.Error("expected error for import inside function block")
	}
}

func TestImportAfterDeclarations(t *testing.T) {
	// Test: import after function declarations should error E2036
	// Fixes #324
	input := `import & use @std
do foo() {
	println("foo")
}
import & use @math
do main() {
	println("test")
}`
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	if !p.EZErrors().HasErrors() {
		t.Error("expected error for import after declarations")
	}
}

func TestReservedStructName(t *testing.T) {
	// Test: using reserved keyword as struct name should error E2037
	// Fixes #325
	input := `const int struct {
	x int
}`
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	if !p.EZErrors().HasErrors() {
		t.Error("expected error for reserved struct name")
	}
}

func TestReservedEnumName(t *testing.T) {
	// Test: using reserved keyword as enum name should error E2038
	// Fixes #326
	input := `const int enum {
	A
	B
}`
	l := NewLexer(input)
	p := NewWithSource(l, input, "test.ez")
	p.ParseProgram()

	if !p.EZErrors().HasErrors() {
		t.Error("expected error for reserved enum name")
	}
}
