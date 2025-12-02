package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"strings"
	"testing"

	"github.com/marshallburns/ez/pkg/lexer"
	"github.com/marshallburns/ez/pkg/parser"
)

// ============================================================================
// Test Helpers
// ============================================================================

func testEval(input string) Object {
	l := lexer.NewLexer(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := NewEnvironment()
	return Eval(program, env)
}

func testEvalWithEnv(input string, env *Environment) Object {
	l := lexer.NewLexer(input)
	p := parser.New(l)
	program := p.ParseProgram()
	return Eval(program, env)
}

func testIntegerObject(t *testing.T, obj Object, expected int64) bool {
	t.Helper()
	result, ok := obj.(*Integer)
	if !ok {
		t.Errorf("object is not Integer. got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("object has wrong value. got=%d, want=%d", result.Value, expected)
		return false
	}
	return true
}

func testFloatObject(t *testing.T, obj Object, expected float64) bool {
	t.Helper()
	result, ok := obj.(*Float)
	if !ok {
		t.Errorf("object is not Float. got=%T (%+v)", obj, obj)
		return false
	}
	// Use a small epsilon for float comparison
	if result.Value != expected {
		t.Errorf("object has wrong value. got=%f, want=%f", result.Value, expected)
		return false
	}
	return true
}

func testBooleanObject(t *testing.T, obj Object, expected bool) bool {
	t.Helper()
	result, ok := obj.(*Boolean)
	if !ok {
		t.Errorf("object is not Boolean. got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("object has wrong value. got=%t, want=%t", result.Value, expected)
		return false
	}
	return true
}

func testStringObject(t *testing.T, obj Object, expected string) bool {
	t.Helper()
	result, ok := obj.(*String)
	if !ok {
		t.Errorf("object is not String. got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("object has wrong value. got=%s, want=%s", result.Value, expected)
		return false
	}
	return true
}

func testNilObject(t *testing.T, obj Object) bool {
	t.Helper()
	if obj != NIL {
		t.Errorf("object is not NIL. got=%T (%+v)", obj, obj)
		return false
	}
	return true
}

func isErrorObject(obj Object) bool {
	_, ok := obj.(*Error)
	return ok
}

// ============================================================================
// Integer Literal Tests
// ============================================================================

func TestEvalIntegerExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5", 5},
		{"10", 10},
		{"-5", -5},
		{"-10", -10},
		{"5 + 5 + 5 + 5 - 10", 10},
		{"2 * 2 * 2 * 2 * 2", 32},
		{"-50 + 100 + -50", 0},
		{"5 * 2 + 10", 20},
		{"5 + 2 * 10", 25},
		{"20 + 2 * -10", 0},
		{"50 / 2 * 2 + 10", 60},
		{"2 * (5 + 10)", 30},
		{"3 * 3 * 3 + 10", 37},
		{"3 * (3 * 3) + 10", 37},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10", 50},
		{"10 % 3", 1},
		{"15 % 4", 3},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Float Literal Tests
// ============================================================================

func TestEvalFloatExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"3.14", 3.14},
		{"-3.14", -3.14},
		{"3.14 + 2.86", 6.0},
		{"3.14 * 2.0", 6.28},
		{"10.0 / 4.0", 2.5},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testFloatObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Boolean Expression Tests
// ============================================================================

func TestEvalBooleanExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1 < 2", true},
		{"1 > 2", false},
		{"1 < 1", false},
		{"1 > 1", false},
		{"1 == 1", true},
		{"1 != 1", false},
		{"1 == 2", false},
		{"1 != 2", true},
		{"true == true", true},
		{"false == false", true},
		{"true == false", false},
		{"true != false", true},
		{"(1 < 2) == true", true},
		{"(1 < 2) == false", false},
		{"(1 > 2) == true", false},
		{"(1 > 2) == false", true},
		{"1 <= 2", true},
		{"2 <= 2", true},
		{"3 <= 2", false},
		{"1 >= 2", false},
		{"2 >= 2", true},
		{"3 >= 2", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// String Tests
// ============================================================================

func TestEvalStringExpression(t *testing.T) {
	input := `"Hello World!"`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "Hello World!")
}

func TestStringConcatenation(t *testing.T) {
	input := `"Hello" + " " + "World!"`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "Hello World!")
}

// ============================================================================
// Prefix Expression Tests
// ============================================================================

func TestBangOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"!true", false},
		{"!false", true},
		{"!!true", true},
		{"!!false", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

func TestNegationOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"-5", -5},
		{"-10", -10},
		{"-(5)", -5},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

func TestNegationWithVariable(t *testing.T) {
	input := `
temp x int = 5
temp y int = -x
y
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, -5)
}

// ============================================================================
// If/Else Tests
// ============================================================================

func TestIfElseExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"if true { 10 }", 10},
		{"if false { 10 }", nil},
		{"if 1 < 2 { 10 }", 10},
		{"if 1 > 2 { 10 }", nil},
		{"if 1 > 2 { 10 } otherwise { 20 }", 20},
		{"if 1 < 2 { 10 } otherwise { 20 }", 10},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			switch expected := tt.expected.(type) {
			case int:
				testIntegerObject(t, evaluated, int64(expected))
			case nil:
				testNilObject(t, evaluated)
			}
		})
	}
}

// ============================================================================
// Return Statement Tests
// ============================================================================

func TestReturnStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"do test() -> int { return 10 } temp r int = test() r", 10},
		{"do test() -> int { return 10 return 9 } temp r int = test() r", 10},
		{"do test() -> int { return 2 * 5 return 9 } temp r int = test() r", 10},
		{"do test() -> int { 9 return 2 * 5 return 9 } temp r int = test() r", 10},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Variable Declaration Tests
// ============================================================================

func TestVariableDeclarations(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"temp a int = 5 a", 5},
		{"temp a int = 5 * 5 a", 25},
		{"temp a int = 5 temp b int = a b", 5},
		{"temp a int = 5 temp b int = a temp c int = a + b + 5 c", 15},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

func TestConstDeclarations(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"const PI int = 314 PI", 314},
		{"const X int = 10 const Y int = X + 5 Y", 15},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Function Tests
// ============================================================================

func TestFunctionObject(t *testing.T) {
	// In EZ, function declarations don't return the function object
	// Functions are registered in the environment and called by name
	// So we test function existence by calling it
	input := `do add(x int, y int) -> int { return x + y } temp r int = add(1, 2) r`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 3)
}

func TestFunctionApplication(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"do identity(x int) -> int { return x } temp r int = identity(5) r", 5},
		{"do double(x int) -> int { return x * 2 } temp r int = double(5) r", 10},
		{"do add(x int, y int) -> int { return x + y } temp r int = add(5, 5) r", 10},
		{"do add(x int, y int) -> int { return x + y } temp r int = add(5 + 5, add(5, 5)) r", 20},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

func TestRecursiveFunctions(t *testing.T) {
	input := `
do factorial(n int) -> int {
	if n <= 1 {
		return 1
	}
	return n * factorial(n - 1)
}
temp r int = factorial(5)
r
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 120)
}

// ============================================================================
// Array Tests
// ============================================================================

func TestArrayLiterals(t *testing.T) {
	input := "{1, 2, 3}"
	evaluated := testEval(input)

	arr, ok := evaluated.(*Array)
	if !ok {
		t.Fatalf("object is not Array. got=%T (%+v)", evaluated, evaluated)
	}

	if len(arr.Elements) != 3 {
		t.Fatalf("array has wrong num of elements. got=%d", len(arr.Elements))
	}

	testIntegerObject(t, arr.Elements[0], 1)
	testIntegerObject(t, arr.Elements[1], 2)
	testIntegerObject(t, arr.Elements[2], 3)
}

func TestArrayIndexExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"{1, 2, 3}[0]", 1},
		{"{1, 2, 3}[1]", 2},
		{"{1, 2, 3}[2]", 3},
		{"temp i int = 0 {1}[i]", 1},
		{"{1, 2, 3}[1 + 1]", 3},
		{"temp myArray [int] = {1, 2, 3} myArray[2]", 3},
		{"temp myArray [int] = {1, 2, 3} myArray[0] + myArray[1] + myArray[2]", 6},
		{"temp myArray [int] = {1, 2, 3} temp i int = myArray[0] myArray[i]", 2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			if expected, ok := tt.expected.(int); ok {
				testIntegerObject(t, evaluated, int64(expected))
			}
		})
	}
}

// ============================================================================
// Map Tests
// ============================================================================

func TestMapLiterals(t *testing.T) {
	input := `{"one": 1, "two": 2}`
	evaluated := testEval(input)

	mapObj, ok := evaluated.(*Map)
	if !ok {
		t.Fatalf("object is not Map. got=%T (%+v)", evaluated, evaluated)
	}

	if len(mapObj.Pairs) != 2 {
		t.Fatalf("map has wrong num of pairs. got=%d", len(mapObj.Pairs))
	}
}

func TestMapIndexExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`{"one": 1}["one"]`, 1},
		{`{"two": 2, "one": 1}["two"]`, 2},
		{`temp key string = "one" {"one": 1}[key]`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Loop Tests
// ============================================================================

func TestWhileLoop(t *testing.T) {
	input := `
temp counter int = 0
as_long_as counter < 5 {
	counter = counter + 1
}
counter
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 5)
}

func TestForLoop(t *testing.T) {
	input := `
temp sum int = 0
for i in range(0, 5) {
	sum = sum + i
}
sum
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 10) // 0 + 1 + 2 + 3 + 4 = 10
}

func TestBreakStatement(t *testing.T) {
	input := `
temp counter int = 0
as_long_as true {
	counter = counter + 1
	if counter == 5 {
		break
	}
}
counter
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 5)
}

func TestContinueStatement(t *testing.T) {
	input := `
temp sum int = 0
for i in range(0, 10) {
	if i % 2 == 0 {
		continue
	}
	sum = sum + i
}
sum
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 25) // 1 + 3 + 5 + 7 + 9 = 25
}

// ============================================================================
// Assignment Tests
// ============================================================================

func TestAssignment(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"temp x int = 5 x = 10 x", 10},
		{"temp x int = 5 x = x + 5 x", 10},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

func TestCompoundAssignment(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"temp x int = 10 x += 5 x", 15},
		{"temp x int = 10 x -= 3 x", 7},
		{"temp x int = 10 x *= 2 x", 20},
		{"temp x int = 10 x /= 2 x", 5},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

func TestPostfixOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"temp x int = 5 x++ x", 6},
		{"temp x int = 5 x-- x", 4},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Error Tests
// ============================================================================

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		input           string
		expectedMessage string
	}{
		{"-true", "unknown operator"},
		{"true + false", "unknown operator"},
		{"foobar", "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)

			errObj, ok := evaluated.(*Error)
			if !ok {
				t.Fatalf("no error object returned. got=%T (%+v)", evaluated, evaluated)
			}

			if !strings.Contains(errObj.Message, tt.expectedMessage) {
				t.Errorf("wrong error message. expected to contain=%q, got=%q",
					tt.expectedMessage, errObj.Message)
			}
		})
	}
}

func TestTypeMismatchError(t *testing.T) {
	input := `5 + true`
	evaluated := testEval(input)

	errObj, ok := evaluated.(*Error)
	if !ok {
		t.Fatalf("no error object returned. got=%T (%+v)", evaluated, evaluated)
	}

	// The exact error message may vary, but it should indicate a type issue
	if !strings.Contains(errObj.Message, "type") && !strings.Contains(errObj.Message, "mismatch") &&
		!strings.Contains(errObj.Message, "unknown operator") {
		t.Errorf("expected type-related error, got=%q", errObj.Message)
	}
}

func TestBreakOutsideLoop(t *testing.T) {
	input := `break`
	evaluated := testEval(input)

	if !isErrorObject(evaluated) {
		t.Fatalf("expected error, got %T (%+v)", evaluated, evaluated)
	}
}

func TestContinueOutsideLoop(t *testing.T) {
	input := `continue`
	evaluated := testEval(input)

	if !isErrorObject(evaluated) {
		t.Fatalf("expected error, got %T (%+v)", evaluated, evaluated)
	}
}

// ============================================================================
// Struct Tests
// ============================================================================

func TestStructLiteral(t *testing.T) {
	input := `
const Point struct {
	x int
	y int
}
temp p Point = Point{x: 5, y: 10}
p.x + p.y
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 15)
}

func TestStructFieldAccess(t *testing.T) {
	input := `
const Person struct {
	name string
	age int
}
temp person Person = Person{name: "Alice", age: 30}
person.name
`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "Alice")
}

func TestStructFieldAssignment(t *testing.T) {
	input := `
const Point struct {
	x int
	y int
}
temp p Point = Point{x: 0, y: 0}
p.x = 5
p.y = 10
p.x + p.y
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 15)
}

// ============================================================================
// Enum Tests
// ============================================================================

func TestEnumDeclaration(t *testing.T) {
	input := `
const Color enum {
	Red
	Green
	Blue
}
Color.Red
`
	evaluated := testEval(input)

	enumVal, ok := evaluated.(*EnumValue)
	if !ok {
		t.Fatalf("expected EnumValue, got %T (%+v)", evaluated, evaluated)
	}

	if enumVal.Name != "Red" {
		t.Errorf("expected enum name 'Red', got %q", enumVal.Name)
	}
}

// ============================================================================
// Logical Operator Tests
// ============================================================================

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true && true", true},
		{"true && false", false},
		{"false && true", false},
		{"false && false", false},
		{"true || true", true},
		{"true || false", true},
		{"false || true", true},
		{"false || false", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Nil Tests
// ============================================================================

func TestNilLiteral(t *testing.T) {
	input := "nil"
	evaluated := testEval(input)
	testNilObject(t, evaluated)
}

// ============================================================================
// Char Tests
// ============================================================================

func TestCharLiteral(t *testing.T) {
	input := "'a'"
	evaluated := testEval(input)

	charObj, ok := evaluated.(*Char)
	if !ok {
		t.Fatalf("object is not Char. got=%T (%+v)", evaluated, evaluated)
	}

	if charObj.Value != 'a' {
		t.Errorf("wrong value. expected='a', got=%q", charObj.Value)
	}
}

// ============================================================================
// Environment Tests
// ============================================================================

func TestNewEnvironment(t *testing.T) {
	env := NewEnvironment()
	if env == nil {
		t.Fatal("NewEnvironment returned nil")
	}
}

func TestEnclosedEnvironment(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("x", &Integer{Value: 5}, true)

	inner := NewEnclosedEnvironment(outer)

	// Inner should see outer's variable
	val, ok := inner.Get("x")
	if !ok {
		t.Error("inner environment should find outer's variable x")
	}
	testIntegerObject(t, val, 5)

	// Setting in inner shouldn't affect outer
	inner.Set("y", &Integer{Value: 10}, true)
	_, ok = outer.Get("y")
	if ok {
		t.Error("outer environment should not see inner's variable y")
	}
}

// ============================================================================
// Singleton Tests
// ============================================================================

func TestSingletonValues(t *testing.T) {
	// Test that TRUE, FALSE, NIL are singletons
	if testEval("true") != TRUE {
		t.Error("true should return TRUE singleton")
	}
	if testEval("false") != FALSE {
		t.Error("false should return FALSE singleton")
	}
	if testEval("nil") != NIL {
		t.Error("nil should return NIL singleton")
	}
}

// ============================================================================
// Complex Expression Tests
// ============================================================================

func TestNestedFunctionCalls(t *testing.T) {
	input := `
do add(a int, b int) -> int { return a + b }
do multiply(a int, b int) -> int { return a * b }
temp r int = add(multiply(2, 3), multiply(4, 5))
r
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 26) // (2*3) + (4*5) = 6 + 20 = 26
}

func TestComplexControlFlow(t *testing.T) {
	input := `
do max(a int, b int) -> int {
	if a > b {
		return a
	}
	return b
}
temp r int = max(max(1, 2), max(3, 4))
r
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 4)
}

func TestFibonacci(t *testing.T) {
	input := `
do fib(n int) -> int {
	if n <= 1 {
		return n
	}
	return fib(n - 1) + fib(n - 2)
}
temp r int = fib(10)
r
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 55)
}
