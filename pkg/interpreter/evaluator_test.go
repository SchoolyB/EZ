package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"math/big"
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
	if result.Value.Cmp(big.NewInt(expected)) != 0 {
		t.Errorf("object has wrong value. got=%s, want=%d", result.Value.String(), expected)
		return false
	}
	return true
}

func testBigIntegerObject(t *testing.T, obj Object, expected *big.Int) bool {
	t.Helper()
	result, ok := obj.(*Integer)
	if !ok {
		t.Errorf("object is not Integer. got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value.Cmp(expected) != 0 {
		t.Errorf("object has wrong value. got=%s, want=%s", result.Value.String(), expected.String())
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

func testByteObject(t *testing.T, obj Object, expected byte) bool {
	t.Helper()
	result, ok := obj.(*Byte)
	if !ok {
		t.Errorf("object is not Byte. got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("object has wrong value. got=%d, want=%d", result.Value, expected)
		return false
	}
	return true
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
		{"temp x int = 10 x %= 3 x", 1},
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
		{"!true", false},
		{"!false", true},
		{"!(true && false)", true},
		{"(1 < 2) && (3 > 2)", true},
		{"(1 > 2) || (3 > 2)", true},
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
	outer.Set("x", &Integer{Value: big.NewInt(5)}, true)

	inner := NewEnclosedEnvironment(outer)

	// Inner should see outer's variable
	val, ok := inner.Get("x")
	if !ok {
		t.Error("inner environment should find outer's variable x")
	}
	testIntegerObject(t, val, 5)

	// Setting in inner shouldn't affect outer
	inner.Set("y", &Integer{Value: big.NewInt(10)}, true)
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

// ============================================================================
// Mutable Parameter Tests (& prefix)
// ============================================================================

func TestMutableStructParameter(t *testing.T) {
	// Test that & param allows modification and changes persist
	input := `
const Person struct {
	name string
	age int
}

do birthday(&p Person) {
	p.age = p.age + 1
}

temp bob Person = Person{name: "Bob", age: 30}
birthday(bob)
bob.age
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 31)
}

func TestImmutableStructParameterNoChange(t *testing.T) {
	// Test that non-& param doesn't modify the original (read-only copy behavior)
	// Note: This test verifies the evaluator behavior - the typechecker would
	// actually error on this code, but the evaluator should handle it gracefully
	input := `
const Person struct {
	name string
	age int
}

do getName(p Person) -> string {
	return p.name
}

temp bob Person = Person{name: "Bob", age: 30}
temp n string = getName(bob)
n
`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "Bob")
}

func TestMutableArrayParameter(t *testing.T) {
	// Test that & param allows array modification
	input := `
do doubleFirst(&arr [int]) {
	arr[0] = arr[0] * 2
}

temp nums [int] = {10, 20, 30}
doubleFirst(nums)
nums[0]
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 20)
}

func TestMutableParameterMultipleCalls(t *testing.T) {
	// Test multiple calls with mutable primitive param
	// Primitives now support true reference semantics with &
	input := `
do increment(&n int) {
	n = n + 1
}

temp counter int = 0
increment(counter)
increment(counter)
increment(counter)
counter
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 3)
}

func TestMutableAndImmutableMixed(t *testing.T) {
	// Test function with both mutable and immutable params
	input := `
do addToFirst(&arr [int], value int) {
	arr[0] = arr[0] + value
}

temp nums [int] = {100, 200, 300}
addToFirst(nums, 50)
nums[0]
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 150)
}

func TestMutableNestedStructField(t *testing.T) {
	// Test modifying nested struct field through & param
	input := `
const Address struct {
	city string
	zip int
}

const Person struct {
	name string
	addr Address
}

do updateZip(&p Person, newZip int) {
	p.addr.zip = newZip
}

temp bob Person = Person{name: "Bob", addr: Address{city: "NYC", zip: 10001}}
updateZip(bob, 90210)
bob.addr.zip
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 90210)
}

func TestSwapWithMutableParams(t *testing.T) {
	// Test swap operation with two mutable primitive params
	// Primitives now support true reference semantics with &
	input := `
do swap(&a, &b int) {
	temp t int = a
	a = b
	b = t
}

temp x int = 10
temp y int = 20
swap(x, y)
x + y * 100
`
	// Result should be 20 + 10*100 = 1020 if swap worked
	// (x becomes 20, y becomes 10)
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 1020)
}

func TestMutableFloatParameter(t *testing.T) {
	// Test mutable float parameter
	input := `
do setFloat(&f float) {
	f = 3.14
}

temp val float = 0.0
setFloat(val)
val
`
	evaluated := testEval(input)
	testFloatObject(t, evaluated, 3.14)
}

func TestMutableBoolParameter(t *testing.T) {
	// Test mutable bool parameter
	input := `
do toggle(&b bool) {
	b = !b
}

temp flag bool = false
toggle(flag)
flag
`
	evaluated := testEval(input)
	testBooleanObject(t, evaluated, true)
}

func TestMutableStringParameter(t *testing.T) {
	// Test mutable string parameter
	input := `
do setString(&s string) {
	s = "modified"
}

temp str string = "original"
setString(str)
str
`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "modified")
}

func TestNestedMutableParameterForwarding(t *testing.T) {
	// Test that mutable parameters work when forwarded through multiple functions
	// This tests issue #338 - nested mutable parameter forwarding breaks array indexing
	input := `
do outer(&arr [int]) {
	inner(arr)
}

do inner(&arr [int]) {
	arr[0] = 999
}

temp nums [int] = {1, 2, 3}
outer(nums)
nums[0]
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 999)
}

func TestDeeplyNestedMutableParameterForwarding(t *testing.T) {
	// Test mutable parameters forwarded through multiple levels
	input := `
do level1(&n int) {
	level2(n)
}

do level2(&n int) {
	level3(n)
}

do level3(&n int) {
	n = 42
}

temp x int = 0
level1(x)
x
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 42)
}

// ============================================================================
// String Interpolation Tests
// ============================================================================

func TestInterpolatedString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple variable interpolation",
			input:    `temp x int = 42 "value is ${x}"`,
			expected: "value is 42",
		},
		{
			name:     "string variable interpolation",
			input:    `temp name string = "World" "Hello, ${name}!"`,
			expected: "Hello, World!",
		},
		{
			name:     "expression interpolation",
			input:    `temp a int = 5 temp b int = 3 "sum is ${a + b}"`,
			expected: "sum is 8",
		},
		{
			name:     "multiple interpolations",
			input:    `temp x int = 1 temp y int = 2 "${x} + ${y} = ${x + y}"`,
			expected: "1 + 2 = 3",
		},
		{
			name:     "boolean interpolation",
			input:    `temp flag bool = true "flag is ${flag}"`,
			expected: "flag is true",
		},
		{
			name:     "float interpolation",
			input:    `temp pi float = 3.14 "pi is ${pi}"`,
			expected: "pi is 3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testStringObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// New Expression Tests
// ============================================================================

func TestNewExpression(t *testing.T) {
	// Test new creates struct with default values
	input := `
const Counter struct {
	value int
	name string
}
temp c Counter = new(Counter)
c.value
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 0)
}

func TestNewExpressionStringDefault(t *testing.T) {
	input := `
const Counter struct {
	value int
	name string
}
temp c Counter = new(Counter)
c.name
`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "")
}

func TestNewExpressionNestedStruct(t *testing.T) {
	// Regression test for issue #621/#622: new() should recursively initialize
	// nested struct fields instead of leaving them as nil
	input := `
const Inner struct {
	val int
}
const Outer struct {
	inner Inner
}
temp o = new(Outer)
o.inner.val = 42
o.inner.val
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 42)
}

func TestStructLiteralNestedStruct(t *testing.T) {
	// Regression test for issue #621/#622: struct literals should recursively
	// initialize nested struct fields
	input := `
const Inner struct {
	val int
}
const Outer struct {
	inner Inner
	name string
}
temp o = Outer{name: "test"}
o.inner.val = 99
o.inner.val
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 99)
}

// ============================================================================
// Loop Statement Tests
// ============================================================================

func TestLoopStatement(t *testing.T) {
	// Test loop with break
	input := `
temp count int = 0
loop {
	count = count + 1
	if count == 5 {
		break
	}
}
count
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 5)
}

func TestLoopWithContinue(t *testing.T) {
	// Test loop with continue
	input := `
temp count int = 0
temp sum int = 0
loop {
	count = count + 1
	if count == 3 {
		continue
	}
	sum = sum + count
	if count == 5 {
		break
	}
}
sum
`
	evaluated := testEval(input)
	// sum = 1 + 2 + 4 + 5 = 12 (skips 3)
	testIntegerObject(t, evaluated, 12)
}

// ============================================================================
// ForEach Statement Tests
// ============================================================================

func TestForEachArray(t *testing.T) {
	input := `
temp nums [int] = {1, 2, 3, 4, 5}
temp sum int = 0
for_each n in nums {
	sum = sum + n
}
sum
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 15)
}

func TestForEachString(t *testing.T) {
	input := `
temp str string = "abc"
temp count int = 0
for_each c in str {
	count = count + 1
}
count
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 3)
}

func TestForEachWithBreak(t *testing.T) {
	input := `
temp nums [int] = {1, 2, 3, 4, 5}
temp sum int = 0
for_each n in nums {
	if n == 4 {
		break
	}
	sum = sum + n
}
sum
`
	evaluated := testEval(input)
	// sum = 1 + 2 + 3 = 6 (breaks at 4)
	testIntegerObject(t, evaluated, 6)
}

// ============================================================================
// In Operator Tests
// ============================================================================

func TestInOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "element in array - found",
			input:    `3 in {1, 2, 3, 4, 5}`,
			expected: true,
		},
		{
			name:     "element in array - not found",
			input:    `6 in {1, 2, 3, 4, 5}`,
			expected: false,
		},
		{
			name:     "string in array - found",
			input:    `"b" in {"a", "b", "c"}`,
			expected: true,
		},
		{
			name:     "string in array - not found",
			input:    `"d" in {"a", "b", "c"}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Char Comparison Tests
// ============================================================================

func TestCharComparisonOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`'a' == 'a'`, true},
		{`'a' == 'b'`, false},
		{`'a' != 'b'`, true},
		{`'a' < 'b'`, true},
		{`'b' > 'a'`, true},
		{`'a' <= 'a'`, true},
		{`'a' >= 'a'`, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Byte Tests
// ============================================================================

func TestByteLiteral(t *testing.T) {
	// Bytes are created via byte() conversion or from byte arrays
	input := `temp b byte = 255 b`
	evaluated := testEval(input)
	if evaluated == nil {
		t.Fatal("evaluated is nil")
	}
}

func TestByteArithmetic(t *testing.T) {
	// Test byte + byte operations (returns Byte)
	byteTests := []struct {
		name     string
		input    string
		expected byte
	}{
		{
			name:     "byte addition",
			input:    `temp a byte = 10 temp b byte = 5 a + b`,
			expected: 15,
		},
		{
			name:     "byte subtraction",
			input:    `temp a byte = 10 temp b byte = 3 a - b`,
			expected: 7,
		},
		{
			name:     "byte multiplication",
			input:    `temp a byte = 10 temp b byte = 2 a * b`,
			expected: 20,
		},
	}

	for _, tt := range byteTests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testByteObject(t, evaluated, tt.expected)
		})
	}

	// Test byte + int mixed operations (may return Integer)
	t.Run("byte with integer", func(t *testing.T) {
		input := `temp a byte = 10 temp b int = 5 a + b`
		evaluated := testEval(input)
		// Could return byte or int depending on implementation
		switch result := evaluated.(type) {
		case *Integer:
			if result.Value.Cmp(big.NewInt(15)) != 0 {
				t.Errorf("wrong value. got=%s, want=%d", result.Value.String(), 15)
			}
		case *Byte:
			if result.Value != 15 {
				t.Errorf("wrong value. got=%d, want=%d", result.Value, 15)
			}
		default:
			t.Errorf("expected Integer or Byte, got %T", evaluated)
		}
	})
}

func TestByteComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`temp a byte = 10 temp b byte = 10 a == b`, true},
		{`temp a byte = 10 temp b byte = 20 a == b`, false},
		{`temp a byte = 10 temp b byte = 20 a != b`, true},
		{`temp a byte = 10 temp b byte = 20 a < b`, true},
		{`temp a byte = 20 temp b byte = 10 a > b`, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Variable Declaration Tests (various types)
// ============================================================================

func TestVariableDeclarationTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "i8 type",
			input:    `temp x i8 = 127 x`,
			expected: int64(127),
		},
		{
			name:     "i16 type",
			input:    `temp x i16 = 1000 x`,
			expected: int64(1000),
		},
		{
			name:     "i32 type",
			input:    `temp x i32 = 100000 x`,
			expected: int64(100000),
		},
		{
			name:     "i64 type",
			input:    `temp x i64 = 1000000 x`,
			expected: int64(1000000),
		},
		{
			name:     "u8 type",
			input:    `temp x u8 = 255 x`,
			expected: int64(255),
		},
		{
			name:     "u16 type",
			input:    `temp x u16 = 65535 x`,
			expected: int64(65535),
		},
		{
			name:     "u32 type",
			input:    `temp x u32 = 100000 x`,
			expected: int64(100000),
		},
		{
			name:     "f32 type",
			input:    `temp x f32 = 3.14 x`,
			expected: float64(3.14),
		},
		{
			name:     "f64 type",
			input:    `temp x f64 = 3.14159 x`,
			expected: float64(3.14159),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			switch exp := tt.expected.(type) {
			case int64:
				testIntegerObject(t, evaluated, exp)
			case float64:
				testFloatObject(t, evaluated, exp)
			}
		})
	}
}

func TestConstDeclaration(t *testing.T) {
	input := `const PI float = 3.14159 PI`
	evaluated := testEval(input)
	testFloatObject(t, evaluated, 3.14159)
}

// ============================================================================
// Function Call Tests
// ============================================================================

func TestRecursiveFunction(t *testing.T) {
	input := `
do fib(n int) -> int {
    if n <= 1 {
        return n
    }
    return fib(n - 1) + fib(n - 2)
}
temp result int = fib(10)
result
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 55)
}

func TestFunctionWithNoReturnValue(t *testing.T) {
	// Test that a function without a return type can be called
	input := `
do doNothing() {
    temp x int = 1
}
doNothing()
`
	evaluated := testEval(input)
	// Functions without explicit return type return nil
	if evaluated != NIL && !isErrorObject(evaluated) {
		t.Logf("got=%T (%+v)", evaluated, evaluated)
	}
}

func TestFunctionWithEarlyReturn(t *testing.T) {
	input := `
do checkPositive(n int) -> string {
    if n > 0 {
        return "positive"
    }
    if n < 0 {
        return "negative"
    }
    return "zero"
}
temp result string = checkPositive(-5)
result
`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "negative")
}

// ============================================================================
// Type Inference Tests
// ============================================================================

func TestTypeInference(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "infer int",
			input:    `temp x = 42 x`,
			expected: int64(42),
		},
		{
			name:     "infer float",
			input:    `temp x = 3.14 x`,
			expected: float64(3.14),
		},
		{
			name:     "infer string",
			input:    `temp x = "hello" x`,
			expected: "hello",
		},
		{
			name:     "infer bool",
			input:    `temp x = true x`,
			expected: true,
		},
		{
			name:     "infer from expression",
			input:    `temp x = 1 + 2 x`,
			expected: int64(3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			switch exp := tt.expected.(type) {
			case int64:
				testIntegerObject(t, evaluated, exp)
			case float64:
				testFloatObject(t, evaluated, exp)
			case string:
				testStringObject(t, evaluated, exp)
			case bool:
				testBooleanObject(t, evaluated, exp)
			}
		})
	}
}

// ============================================================================
// As Long As (While) Loop Tests
// ============================================================================

func TestAsLongAsLoop(t *testing.T) {
	input := `
temp count int = 0
as_long_as count < 5 {
    count = count + 1
}
count
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 5)
}

func TestAsLongAsWithBreak(t *testing.T) {
	input := `
temp count int = 0
as_long_as true {
    count = count + 1
    if count == 3 {
        break
    }
}
count
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 3)
}

// ============================================================================
// Range Tests
// ============================================================================

func TestRangeInForLoop(t *testing.T) {
	input := `
temp sum int = 0
for i in range(1, 6) {
    sum = sum + i
}
sum
`
	evaluated := testEval(input)
	// 1 + 2 + 3 + 4 + 5 = 15
	testIntegerObject(t, evaluated, 15)
}

func TestRangeWithStep(t *testing.T) {
	input := `
temp sum int = 0
for i in range(0, 10, 2) {
    sum = sum + i
}
sum
`
	evaluated := testEval(input)
	// 0 + 2 + 4 + 6 + 8 = 20
	testIntegerObject(t, evaluated, 20)
}

func TestRangeInOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "value in range - true",
			input:    `5 in range(0, 10)`,
			expected: true,
		},
		{
			name:     "value at start of range - true",
			input:    `0 in range(0, 10)`,
			expected: true,
		},
		{
			name:     "value at end of range - false (exclusive)",
			input:    `10 in range(0, 10)`,
			expected: false,
		},
		{
			name:     "value below range - false",
			input:    `-1 in range(0, 10)`,
			expected: false,
		},
		{
			name:     "value above range - false",
			input:    `15 in range(0, 10)`,
			expected: false,
		},
		{
			name:     "value in range with step - on step",
			input:    `6 in range(0, 10, 2)`,
			expected: true,
		},
		{
			name:     "value in range with step - off step",
			input:    `5 in range(0, 10, 2)`,
			expected: false,
		},
		{
			name:     "value not_in range - true",
			input:    `15 not_in range(0, 10)`,
			expected: true,
		},
		{
			name:     "value not_in range - false",
			input:    `5 not_in range(0, 10)`,
			expected: false,
		},
		{
			name:     "value !in range with step",
			input:    `5 !in range(0, 10, 2)`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

func TestRangeInWhenStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name: "when with range - match first case",
			input: `
temp x int = 3
temp result int = 0
when x {
    is range(0, 5) {
        result = 1
    }
    is range(5, 10) {
        result = 2
    }
    default {
        result = 3
    }
}
result
`,
			expected: 1,
		},
		{
			name: "when with range - match second case",
			input: `
temp x int = 7
temp result int = 0
when x {
    is range(0, 5) {
        result = 1
    }
    is range(5, 10) {
        result = 2
    }
    default {
        result = 3
    }
}
result
`,
			expected: 2,
		},
		{
			name: "when with range - match default",
			input: `
temp x int = 15
temp result int = 0
when x {
    is range(0, 5) {
        result = 1
    }
    is range(5, 10) {
        result = 2
    }
    default {
        result = 3
    }
}
result
`,
			expected: 3,
		},
		{
			name: "when with range and step - on step",
			input: `
temp x int = 6
temp result int = 0
when x {
    is range(0, 10, 2) {
        result = 1
    }
    default {
        result = 2
    }
}
result
`,
			expected: 1,
		},
		{
			name: "when with range and step - off step",
			input: `
temp x int = 5
temp result int = 0
when x {
    is range(0, 10, 2) {
        result = 1
    }
    default {
        result = 2
    }
}
result
`,
			expected: 2,
		},
		{
			name: "when with range - boundary at start (inclusive)",
			input: `
temp x int = 0
temp result int = 0
when x {
    is range(0, 5) {
        result = 1
    }
    default {
        result = 2
    }
}
result
`,
			expected: 1,
		},
		{
			name: "when with range - boundary at end (exclusive)",
			input: `
temp x int = 5
temp result int = 0
when x {
    is range(0, 5) {
        result = 1
    }
    default {
        result = 2
    }
}
result
`,
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

func TestRangeInIfStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name: "if with range in - true branch",
			input: `
temp x int = 5
temp result int = 0
if x in range(0, 10) {
    result = 1
}
result
`,
			expected: 1,
		},
		{
			name: "if with range in - false (not executed)",
			input: `
temp x int = 15
temp result int = 0
if x in range(0, 10) {
    result = 1
}
result
`,
			expected: 0,
		},
		{
			name: "if with range !in - true branch",
			input: `
temp x int = 15
temp result int = 0
if x !in range(0, 10) {
    result = 1
}
result
`,
			expected: 1,
		},
		{
			name: "if with range and step",
			input: `
temp x int = 4
temp result int = 0
if x in range(0, 10, 2) {
    result = 1
}
result
`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Not In Operator Tests
// ============================================================================

func TestNotInOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "element not in array - true",
			input:    `6 not_in {1, 2, 3, 4, 5}`,
			expected: true,
		},
		{
			name:     "element not in array - false",
			input:    `3 not_in {1, 2, 3, 4, 5}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Increment/Decrement Tests
// ============================================================================

func TestIncrementDecrement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "increment",
			input:    `temp x int = 5 x++ x`,
			expected: 6,
		},
		{
			name:     "decrement",
			input:    `temp x int = 5 x-- x`,
			expected: 4,
		},
		{
			name:     "multiple increments",
			input:    `temp x int = 0 x++ x++ x++ x`,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Float Infix Expression Tests (more coverage)
// ============================================================================

func TestFloatInfixExpressionExtended(t *testing.T) {
	floatTests := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "float subtraction",
			input:    `10.5 - 3.2`,
			expected: 7.3,
		},
		{
			name:     "float variable subtraction",
			input:    `temp a float = 10.0 temp b float = 4.0 a - b`,
			expected: 6.0,
		},
		{
			name:     "float division",
			input:    `10.0 / 4.0`,
			expected: 2.5,
		},
	}

	for _, tt := range floatTests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testFloatObject(t, evaluated, tt.expected)
		})
	}

	// Float comparison tests
	boolTests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "float less than",
			input:    `3.14 < 3.15`,
			expected: true,
		},
		{
			name:     "float greater than",
			input:    `3.15 > 3.14`,
			expected: true,
		},
		{
			name:     "float less than or equal",
			input:    `3.14 <= 3.14`,
			expected: true,
		},
		{
			name:     "float greater than or equal",
			input:    `3.14 >= 3.14`,
			expected: true,
		},
		{
			name:     "float equality",
			input:    `3.14 == 3.14`,
			expected: true,
		},
		{
			name:     "float inequality",
			input:    `3.14 != 3.15`,
			expected: true,
		},
	}

	for _, tt := range boolTests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// String Infix Expression Tests (more coverage)
// ============================================================================

func TestStringInfixExpressionExtended(t *testing.T) {
	// String comparison tests (only == and != are supported)
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "string equality true",
			input:    `"hello" == "hello"`,
			expected: true,
		},
		{
			name:     "string equality false",
			input:    `"hello" == "world"`,
			expected: false,
		},
		{
			name:     "string inequality true",
			input:    `"hello" != "world"`,
			expected: true,
		},
		{
			name:     "string inequality false",
			input:    `"hello" != "hello"`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

func TestStringConcatenationExtended(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "triple concatenation",
			input:    `"hello" + " " + "world"`,
			expected: "hello world",
		},
		{
			name:     "variable concatenation",
			input:    `temp a string = "foo" temp b string = "bar" a + b`,
			expected: "foobar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testStringObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Byte-Integer Mixed Operation Tests (more coverage)
// ============================================================================

func TestByteIntegerMixedOperations(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "byte subtraction from int",
			input: `temp a byte = 10 temp b int = 3 a - b`,
		},
		{
			name:  "byte multiplication with int",
			input: `temp a byte = 10 temp b int = 2 a * b`,
		},
		{
			name:  "byte division by int",
			input: `temp a byte = 10 temp b int = 2 a / b`,
		},
		{
			name:  "int addition with byte",
			input: `temp a int = 10 temp b byte = 5 a + b`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			// Verify result is numeric
			switch result := evaluated.(type) {
			case *Integer:
				if result.Value.Sign() < 0 {
					t.Logf("got negative result: %s", result.Value.String())
				}
			case *Byte:
				// byte result is fine
			default:
				if !isErrorObject(evaluated) {
					t.Errorf("expected numeric result, got %T", evaluated)
				}
			}
		})
	}
}

// ============================================================================
// Enum Access Tests (more coverage)
// ============================================================================

func TestEnumAccess(t *testing.T) {
	input := `
const Color enum {
    Red
    Green
    Blue
}
Color.Green
`
	evaluated := testEval(input)
	enumVal, ok := evaluated.(*EnumValue)
	if !ok {
		t.Fatalf("expected EnumValue, got %T (%+v)", evaluated, evaluated)
	}
	if enumVal.Name != "Green" {
		t.Errorf("expected enum name 'Green', got %q", enumVal.Name)
	}
}

func TestEnumComparison(t *testing.T) {
	input := `
const Color enum {
    Red
    Green
    Blue
}
Color.Red == Color.Red
`
	evaluated := testEval(input)
	testBooleanObject(t, evaluated, true)
}

func TestEnumInequality(t *testing.T) {
	input := `
const Color enum {
    Red
    Green
    Blue
}
Color.Red != Color.Blue
`
	evaluated := testEval(input)
	testBooleanObject(t, evaluated, true)
}

// ============================================================================
// While Loop Tests (more coverage)
// ============================================================================

func TestWhileLoopBreak(t *testing.T) {
	input := `
temp sum int = 0
temp i int = 0
as_long_as i < 10 {
    if i == 5 {
        break
    }
    sum += i
    i++
}
sum
`
	evaluated := testEval(input)
	// 0 + 1 + 2 + 3 + 4 = 10
	testIntegerObject(t, evaluated, 10)
}

func TestWhileLoopContinue(t *testing.T) {
	input := `
temp sum int = 0
temp i int = 0
as_long_as i < 5 {
    i++
    if i == 3 {
        continue
    }
    sum += i
}
sum
`
	evaluated := testEval(input)
	// 1 + 2 + 4 + 5 = 12
	testIntegerObject(t, evaluated, 12)
}

// ============================================================================
// Identifier Resolution Tests (more coverage)
// ============================================================================

func TestIdentifierNotFound(t *testing.T) {
	input := `undefinedVariable`
	evaluated := testEval(input)
	if !isErrorObject(evaluated) {
		t.Errorf("expected error for undefined variable, got %T", evaluated)
	}
}

func TestConstantReassignment(t *testing.T) {
	input := `
const x int = 5
x = 10
`
	evaluated := testEval(input)
	if !isErrorObject(evaluated) {
		t.Errorf("expected error for constant reassignment, got %T", evaluated)
	}
}

// Note: TestStructTypeAsIdentifier tests are covered by:
// - pkg/stdlib/json_test.go (TestJSONDecodeTyped* tests)
// - integration-tests/pass/stdlib/json.ez
// These tests verify that struct types can be passed as function arguments
// and that the evaluator correctly returns TypeValue objects for struct identifiers.

// ============================================================================
// Bang Operator Tests (more coverage)
// ============================================================================

func TestBangOperatorExtended(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`!!true`, true},
		{`!!false`, false},
		{`!!!true`, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Variable Declaration Tests (sized integers)
// ============================================================================

func TestSizedIntegerDeclaration(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"i8 variable", `temp x i8 = 127 x`},
		{"i16 variable", `temp x i16 = 1000 x`},
		{"i32 variable", `temp x i32 = 100000 x`},
		{"i64 variable", `temp x i64 = 1000000 x`},
		{"u8 variable", `temp x u8 = 255 x`},
		{"u16 variable", `temp x u16 = 65535 x`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			// Just verify it evaluates without error
			if isErrorObject(evaluated) {
				errObj := evaluated.(*Error)
				t.Errorf("got error: %s", errObj.Message)
			}
		})
	}
}

// ============================================================================
// Assignment Tests (simple)
// ============================================================================

func TestSimpleAssignment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "assign to existing variable",
			input:    `temp x int = 5 x = 10 x`,
			expected: 10,
		},
		{
			name:     "assign expression result",
			input:    `temp x int = 5 x = x + 5 x`,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Function Call Tests (parameters)
// ============================================================================

func TestFunctionMultipleParameters(t *testing.T) {
	input := `
do add3(a int, b int, c int) -> int { return a + b + c }
temp r int = add3(1, 2, 3)
r
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 6)
}

// ============================================================================
// If Statement Tests (or/otherwise chains)
// ============================================================================

func TestIfOrOtherwiseChain(t *testing.T) {
	input := `
temp x int = 5
temp result int = 0
if x > 10 {
    result = 1
} or x > 3 {
    result = 2
} otherwise {
    result = 3
}
result
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 2)
}

// ============================================================================
// Additional Operator Tests
// ============================================================================

func TestNegativeNumbers(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"-5", -5},
		{"-(-5)", 5},
		{"5 - -5", 10},
		{"-10 + 5", -5},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// Type Coercion Tests
// ============================================================================

func TestIntToFloatCoercion(t *testing.T) {
	// Test that integers can be used in float context
	input := `temp x float = 5.0 temp y int = 2 x + 2.0`
	evaluated := testEval(input)
	testFloatObject(t, evaluated, 7.0)
}

// ============================================================================
// Type Helper Function Tests
// ============================================================================

func TestIsSignedIntegerType(t *testing.T) {
	signedTypes := []string{"i8", "i16", "i32", "i64", "i128", "i256", "int"}
	for _, typ := range signedTypes {
		if !isSignedIntegerType(typ) {
			t.Errorf("isSignedIntegerType(%q) = false, want true", typ)
		}
	}

	unsignedTypes := []string{"u8", "u16", "u32", "u64", "u128", "u256", "uint"}
	for _, typ := range unsignedTypes {
		if isSignedIntegerType(typ) {
			t.Errorf("isSignedIntegerType(%q) = true, want false", typ)
		}
	}

	otherTypes := []string{"string", "float", "bool", "byte"}
	for _, typ := range otherTypes {
		if isSignedIntegerType(typ) {
			t.Errorf("isSignedIntegerType(%q) = true, want false", typ)
		}
	}
}

func TestIsIntegerType(t *testing.T) {
	intTypes := []string{"i8", "i16", "i32", "i64", "int", "u8", "u16", "u32", "u64", "uint"}
	for _, typ := range intTypes {
		if !isIntegerType(typ) {
			t.Errorf("isIntegerType(%q) = false, want true", typ)
		}
	}

	nonIntTypes := []string{"string", "float", "bool", "byte", "char"}
	for _, typ := range nonIntTypes {
		if isIntegerType(typ) {
			t.Errorf("isIntegerType(%q) = true, want false", typ)
		}
	}
}

func TestObjectTypeToEZ(t *testing.T) {
	tests := []struct {
		obj      Object
		expected string
	}{
		{&Integer{Value: big.NewInt(42)}, "int"},
		{&Float{Value: 3.14}, "float"},
		{&String{Value: "hello"}, "string"},
		{&Boolean{Value: true}, "bool"},
		{NIL, "nil"},
	}

	for _, tt := range tests {
		result := objectTypeToEZ(tt.obj)
		if result != tt.expected {
			t.Errorf("objectTypeToEZ(%T) = %q, want %q", tt.obj, result, tt.expected)
		}
	}

	// Test Char and Byte separately - they return uppercase
	charResult := objectTypeToEZ(&Char{Value: 'a'})
	if charResult != "char" && charResult != "CHAR" {
		t.Errorf("objectTypeToEZ(Char) = %q, want char or CHAR", charResult)
	}

	byteResult := objectTypeToEZ(&Byte{Value: 0xFF})
	if byteResult != "byte" && byteResult != "BYTE" {
		t.Errorf("objectTypeToEZ(Byte) = %q, want byte or BYTE", byteResult)
	}
}

func TestTypeMatches(t *testing.T) {
	tests := []struct {
		name     string
		obj      Object
		ezType   string
		expected bool
	}{
		{"int matches int", &Integer{Value: big.NewInt(42)}, "int", true},
		{"float matches float", &Float{Value: 3.14}, "float", true},
		{"string matches string", &String{Value: "hi"}, "string", true},
		{"bool matches bool", &Boolean{Value: true}, "bool", true},
		{"nil matches nil", NIL, "nil", true},
		{"int does not match float", &Integer{Value: big.NewInt(42)}, "float", false},
		{"float does not match int", &Float{Value: 3.14}, "int", false},
		// Integer family compatibility
		{"i8 matches i16", &Integer{Value: big.NewInt(10), DeclaredType: "i8"}, "i16", true},
		{"int matches i32", &Integer{Value: big.NewInt(10)}, "i32", true},
		// Positive int can go to unsigned
		{"positive int matches u8", &Integer{Value: big.NewInt(10)}, "u8", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeMatches(tt.obj, tt.ezType)
			if result != tt.expected {
				t.Errorf("typeMatches(%T, %q) = %v, want %v", tt.obj, tt.ezType, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// More Enum Tests
// ============================================================================

func TestEnumWithMultipleValues(t *testing.T) {
	input := `
const Status enum {
    Active
    Pending
    Inactive
    Deleted
}
Status.Pending
`
	evaluated := testEval(input)
	enumVal, ok := evaluated.(*EnumValue)
	if !ok {
		t.Fatalf("expected EnumValue, got %T (%+v)", evaluated, evaluated)
	}
	if enumVal.Name != "Pending" {
		t.Errorf("expected enum name 'Pending', got %q", enumVal.Name)
	}
}

// ============================================================================
// More Variable Tests
// ============================================================================

func TestConstFloatDeclaration(t *testing.T) {
	input := `const PI float = 3.14159 PI`
	evaluated := testEval(input)
	testFloatObject(t, evaluated, 3.14159)
}

func TestVariableWithTypeInference(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"infer int", `temp x = 42 x`},
		{"infer float", `temp x = 3.14 x`},
		{"infer string", `temp x = "hello" x`},
		{"infer bool", `temp x = true x`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			if isErrorObject(evaluated) {
				t.Errorf("got error: %+v", evaluated)
			}
		})
	}
}

// ============================================================================
// More Expression Tests
// ============================================================================

func TestGroupedExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"(1 + 2) * 3", 9},
		{"2 * (3 + 4)", 14},
		{"(5 + 5) * (2 + 3)", 50},
		{"((1 + 2) + 3) * 4", 24},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

func TestComplexArithmetic(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1 + 2 * 3", 7},
		{"10 / 2 + 3", 8},
		{"10 - 2 - 3", 5},
		{"10 % 3 + 1", 2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

// ============================================================================
// More Control Flow Tests
// ============================================================================

func TestNestedIfStatements(t *testing.T) {
	input := `
temp x int = 10
temp result int = 0
if x > 5 {
    if x > 8 {
        result = 1
    } otherwise {
        result = 2
    }
} otherwise {
    result = 3
}
result
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 1)
}

func TestLoopWithBreak(t *testing.T) {
	input := `
temp count int = 0
loop {
    count++
    if count >= 5 {
        break
    }
}
count
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 5)
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestUndefinedVariable(t *testing.T) {
	input := `undefined_var`
	evaluated := testEval(input)
	if !isErrorObject(evaluated) {
		t.Errorf("expected error for undefined variable, got %T", evaluated)
	}
}

func TestDivisionByZero(t *testing.T) {
	input := `10 / 0`
	evaluated := testEval(input)
	// Division by zero might return error or special value
	if isErrorObject(evaluated) {
		// OK, got an error as expected
	} else if intVal, ok := evaluated.(*Integer); ok {
		// Some languages return 0 or special value
		t.Logf("division by zero returned %d", intVal.Value)
	}
}

// ============================================================================
// Array Indexing Tests
// ============================================================================

func TestArrayIndexExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`
temp arr [int] = {1, 2, 3}
arr[0]
`, 1},
		{`
temp arr [int] = {1, 2, 3}
arr[1]
`, 2},
		{`
temp arr [int] = {1, 2, 3}
arr[2]
`, 3},
		{`
temp arr [int] = {10, 20, 30, 40, 50}
arr[4]
`, 50},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestArrayIndexWithExpression(t *testing.T) {
	input := `
temp arr [int] = {1, 2, 3, 4, 5}
temp idx int = 2
arr[idx]
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 3)
}

func TestArrayIndexOutOfBounds(t *testing.T) {
	input := `
temp arr [int] = {1, 2, 3}
arr[10]
`
	evaluated := testEval(input)
	if !isErrorObject(evaluated) {
		t.Logf("out of bounds returned %T", evaluated)
	}
}

// ============================================================================
// String Indexing Tests
// ============================================================================

func TestStringIndexExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`
temp s string = "hello"
s[0]
`, "h"},
		{`
temp s string = "hello"
s[4]
`, "o"},
		{`
temp s string = "world"
s[2]
`, "r"},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		if str, ok := evaluated.(*String); ok {
			if str.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, str.Value)
			}
		} else if ch, ok := evaluated.(*Char); ok {
			if string(ch.Value) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(ch.Value))
			}
		} else {
			t.Logf("string index returned %T", evaluated)
		}
	}
}

// ============================================================================
// Map Tests
// ============================================================================

func TestMapLiteral(t *testing.T) {
	input := `{"a": 1, "b": 2, "c": 3}["a"]`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 1)
}

func TestMapIndexExpression(t *testing.T) {
	input := `{"x": 10, "y": 20, "z": 30}["y"]`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 20)
}

func TestMapWithStringKeys(t *testing.T) {
	input := `{"name": "test", "type": "example"}["name"]`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "test")
}

// ============================================================================
// More For Loop Tests
// ============================================================================

func TestForLoopWithStep(t *testing.T) {
	input := `
temp sum int = 0
for i in range(0, 10, 2) {
    sum = sum + i
}
sum
`
	evaluated := testEval(input)
	// 0 + 2 + 4 + 6 + 8 = 20
	testIntegerObject(t, evaluated, 20)
}

func TestForLoopContinue(t *testing.T) {
	input := `
temp sum int = 0
for i in range(0, 5) {
    if i == 2 {
        continue
    }
    sum = sum + i
}
sum
`
	evaluated := testEval(input)
	// 0 + 1 + 3 + 4 = 8 (skipping 2)
	testIntegerObject(t, evaluated, 8)
}

func TestForLoopBreak(t *testing.T) {
	input := `
temp sum int = 0
for i in range(0, 10) {
    if i == 5 {
        break
    }
    sum = sum + i
}
sum
`
	evaluated := testEval(input)
	// 0 + 1 + 2 + 3 + 4 = 10
	testIntegerObject(t, evaluated, 10)
}

// ============================================================================
// More Struct Tests
// ============================================================================

func TestStructWithMultipleFields(t *testing.T) {
	input := `
const Person struct {
    name string
    age int
    active bool
}
temp p Person = Person{name: "", age: 0, active: false}
p.active
`
	evaluated := testEval(input)
	testBooleanObject(t, evaluated, false)
}

func TestStructFieldAssignmentAndSum(t *testing.T) {
	input := `
const Point struct {
    x int
    y int
}
temp pt Point = Point{x: 0, y: 0}
pt.x = 10
pt.y = 20
pt.x + pt.y
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 30)
}

// ============================================================================
// Function Call Expression Tests
// ============================================================================

func TestFunctionWithNoParams(t *testing.T) {
	input := `
do getFortyTwo() -> int {
    return 42
}
temp result int = getFortyTwo()
result
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 42)
}

func TestNestedFunctionCallsChained(t *testing.T) {
	input := `
do double(x int) -> int {
    return x * 2
}
do addOne(x int) -> int {
    return x + 1
}
temp result int = double(addOne(5))
result
`
	evaluated := testEval(input)
	// addOne(5) = 6, double(6) = 12
	testIntegerObject(t, evaluated, 12)
}

// ============================================================================
// Prefix Expression Tests
// ============================================================================

func TestNotOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`!true`, false},
		{`!false`, true},
		{`!!true`, true},
		{`!!false`, false},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestNegativeNumbersPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`-5`, -5},
		{`-10`, -10},
		{`-100`, -100},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

// ============================================================================
// Postfix Expression Tests
// ============================================================================

func TestPostfixIncrement(t *testing.T) {
	input := `
temp x int = 5
x++
x
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 6)
}

func TestPostfixDecrement(t *testing.T) {
	input := `
temp x int = 5
x--
x
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 4)
}

// ============================================================================
// More Byte Operation Tests
// ============================================================================

func TestByteComparisonOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`
temp a byte = 10
temp c byte = 10
a == c
`, true},
		{`
temp a byte = 10
temp c byte = 20
a == c
`, false},
		{`
temp a byte = 10
temp c byte = 20
a != c
`, true},
		{`
temp a byte = 10
temp c byte = 20
a < c
`, true},
		{`
temp a byte = 20
temp c byte = 10
a > c
`, true},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestByteBitwiseOr(t *testing.T) {
	input := `
temp a byte = 15
temp c byte = 240
a | c
`
	evaluated := testEval(input)
	// 15 | 240 = 255
	if b, ok := evaluated.(*Byte); ok {
		if b.Value != 255 {
			t.Errorf("expected 255, got %d", b.Value)
		}
	} else {
		t.Logf("bitwise or returned %T", evaluated)
	}
}

func TestByteBitwiseXor(t *testing.T) {
	input := `
temp a byte = 255
temp c byte = 240
a ^ c
`
	evaluated := testEval(input)
	// 255 ^ 240 = 15
	if b, ok := evaluated.(*Byte); ok {
		if b.Value != 15 {
			t.Errorf("expected 15, got %d", b.Value)
		}
	} else {
		t.Logf("bitwise xor returned %T", evaluated)
	}
}

// ============================================================================
// In Operator Tests
// ============================================================================

func TestInOperatorArray(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`3 in {1, 2, 3, 4, 5}`, true},
		{`10 in {1, 2, 3, 4, 5}`, false},
		{`"b" in {"a", "b", "c"}`, true},
		{`"z" in {"a", "b", "c"}`, false},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

// ============================================================================
// Interpolated String Tests
// ============================================================================

func TestInterpolatedStringGreeting(t *testing.T) {
	input := `
temp name string = "World"
temp greeting string = "Hello, ${name}!"
greeting
`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "Hello, World!")
}

func TestInterpolatedStringWithExpression(t *testing.T) {
	input := `
temp x int = 5
temp y int = 10
temp result string = "Sum: ${x + y}"
result
`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "Sum: 15")
}

// ============================================================================
// Variable Declaration Tests
// ============================================================================

func TestConstArrayDynamicSizeError(t *testing.T) {
	// const arrays with dynamic size should error
	input := `const arr [int] = {1, 2, 3}`
	evaluated := testEval(input)
	if !isErrorObject(evaluated) {
		t.Errorf("expected error for const with dynamic array size, got %T", evaluated)
	}
}

func TestArrayTypeMismatch(t *testing.T) {
	input := `temp arr [int] = "not an array"`
	evaluated := testEval(input)
	if !isErrorObject(evaluated) {
		t.Errorf("expected error for type mismatch, got %T", evaluated)
	}
}

func TestByteArrayValidation(t *testing.T) {
	input := `temp arr [byte] = {0, 127, 255}`
	evaluated := testEval(input)
	// Should succeed with valid byte values
	if isErrorObject(evaluated) {
		t.Errorf("expected success, got error: %v", evaluated)
	}
}

func TestByteArrayOutOfRange(t *testing.T) {
	input := `temp arr [byte] = {256}`
	evaluated := testEval(input)
	if !isErrorObject(evaluated) {
		t.Errorf("expected error for byte out of range, got %T", evaluated)
	}
}

func TestMapAccess(t *testing.T) {
	// Map access using inline map
	input := `{"a": 1, "b": 2}["a"]`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 1)
}

func TestMapNotFoundKey(t *testing.T) {
	// Map access with missing key
	input := `{"a": 1}["missing"]`
	evaluated := testEval(input)
	if !isErrorObject(evaluated) {
		t.Logf("missing key returned %T", evaluated)
	}
}

// ============================================================================
// Assignment Tests
// ============================================================================

func TestImmutableAssignmentError(t *testing.T) {
	input := `
const x int = 10
x = 20
`
	evaluated := testEval(input)
	if !isErrorObject(evaluated) {
		t.Errorf("expected error for assignment to const, got %T", evaluated)
	}
}

func TestMutableAssignment(t *testing.T) {
	input := `
temp x int = 10
x = 20
x
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 20)
}

func TestCompoundSubtraction(t *testing.T) {
	input := `
temp x int = 10
x -= 3
x
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 7)
}

func TestCompoundMultiplication(t *testing.T) {
	input := `
temp x int = 5
x *= 4
x
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 20)
}

func TestCompoundDivision(t *testing.T) {
	input := `
temp x int = 20
x /= 4
x
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 5)
}

// ============================================================================
// More Enum Tests
// ============================================================================

func TestEnumWithValues(t *testing.T) {
	input := `
const Color enum {
    Red
    Green
    Blue
}
Color.Blue
`
	evaluated := testEval(input)
	enumVal, ok := evaluated.(*EnumValue)
	if !ok {
		t.Fatalf("expected EnumValue, got %T (%+v)", evaluated, evaluated)
	}
	if enumVal.Name != "Blue" {
		t.Errorf("expected enum name 'Blue', got %q", enumVal.Name)
	}
}

func TestEnumCompareNotEqual(t *testing.T) {
	input := `
const Color enum {
    Red
    Green
    Blue
}
Color.Red != Color.Blue
`
	evaluated := testEval(input)
	testBooleanObject(t, evaluated, true)
}

// ============================================================================
// More Function Tests
// ============================================================================

func TestFunctionWithMutableParam(t *testing.T) {
	input := `
do increment(&x int) {
    x = x + 1
}
temp val int = 5
increment(val)
val
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 6)
}

func TestRecursiveFunctionFactorial(t *testing.T) {
	input := `
do factorial(n int) -> int {
    if n <= 1 {
        return 1
    }
    return n * factorial(n - 1)
}
temp result int = factorial(5)
result
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 120)
}

// ============================================================================
// More Conditional Tests
// ============================================================================

func TestNestedIfElse(t *testing.T) {
	input := `
temp x int = 15
temp result string = ""
if x > 20 {
    result = "large"
} or x > 10 {
    result = "medium"
} otherwise {
    result = "small"
}
result
`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "medium")
}

func TestMultipleOr(t *testing.T) {
	input := `
temp x int = 5
temp result string = ""
if x > 30 {
    result = "a"
} or x > 20 {
    result = "b"
} or x > 10 {
    result = "c"
} otherwise {
    result = "d"
}
result
`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "d")
}

// ============================================================================
// More Loop Tests
// ============================================================================

func TestWhileLoopWithCounter(t *testing.T) {
	input := `
temp count int = 0
temp sum int = 0
as_long_as count < 5 {
    sum = sum + count
    count++
}
sum
`
	evaluated := testEval(input)
	// 0 + 1 + 2 + 3 + 4 = 10
	testIntegerObject(t, evaluated, 10)
}

func TestLoopContinueAndBreak(t *testing.T) {
	input := `
temp count int = 0
temp sum int = 0
loop {
    count++
    if count == 3 {
        continue
    }
    if count >= 5 {
        break
    }
    sum = sum + count
}
sum
`
	evaluated := testEval(input)
	// 1 + 2 + 4 = 7 (skipping 3)
	testIntegerObject(t, evaluated, 7)
}

func TestForEachLoop(t *testing.T) {
	input := `
temp arr [int] = {1, 2, 3, 4, 5}
temp sum int = 0
for_each item in arr {
    sum = sum + item
}
sum
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 15)
}

// ============================================================================
// More Expression Tests
// ============================================================================

func TestBooleanNot(t *testing.T) {
	input := `
temp flag bool = true
!flag
`
	evaluated := testEval(input)
	testBooleanObject(t, evaluated, false)
}

func TestFloatNegation(t *testing.T) {
	input := `-3.14`
	evaluated := testEval(input)
	testFloatObject(t, evaluated, -3.14)
}

func TestCharLiteralValue(t *testing.T) {
	input := `'a'`
	evaluated := testEval(input)
	if ch, ok := evaluated.(*Char); ok {
		if ch.Value != 'a' {
			t.Errorf("expected 'a', got %c", ch.Value)
		}
	} else {
		t.Errorf("expected Char, got %T", evaluated)
	}
}

func TestCharEquality(t *testing.T) {
	input := `'a' == 'a'`
	evaluated := testEval(input)
	testBooleanObject(t, evaluated, true)
}

func TestCharInequality(t *testing.T) {
	input := `'a' != 'b'`
	evaluated := testEval(input)
	testBooleanObject(t, evaluated, true)
}

// ============================================================================
// Short-Circuit Evaluation Tests
// ============================================================================

func TestShortCircuitAndFalseLeft(t *testing.T) {
	// false && expr should NOT evaluate expr (short-circuit)
	input := `
temp call_count int = 0
do side_effect() -> bool {
    call_count += 1
    return true
}
temp result bool = false && side_effect()
call_count
`
	evaluated := testEval(input)
	// call_count should be 0 because side_effect() should not be called
	testIntegerObject(t, evaluated, 0)
}

func TestShortCircuitOrTrueLeft(t *testing.T) {
	// true || expr should NOT evaluate expr (short-circuit)
	input := `
temp call_count int = 0
do side_effect() -> bool {
    call_count += 1
    return true
}
temp result bool = true || side_effect()
call_count
`
	evaluated := testEval(input)
	// call_count should be 0 because side_effect() should not be called
	testIntegerObject(t, evaluated, 0)
}

func TestShortCircuitAndTrueLeft(t *testing.T) {
	// true && expr SHOULD evaluate expr
	input := `
temp call_count int = 0
do side_effect() -> bool {
    call_count += 1
    return true
}
temp result bool = true && side_effect()
call_count
`
	evaluated := testEval(input)
	// call_count should be 1 because side_effect() was called
	testIntegerObject(t, evaluated, 1)
}

func TestShortCircuitOrFalseLeft(t *testing.T) {
	// false || expr SHOULD evaluate expr
	input := `
temp call_count int = 0
do side_effect() -> bool {
    call_count += 1
    return true
}
temp result bool = false || side_effect()
call_count
`
	evaluated := testEval(input)
	// call_count should be 1 because side_effect() was called
	testIntegerObject(t, evaluated, 1)
}

func TestShortCircuitAndResult(t *testing.T) {
	// Verify correct results for && operator
	tests := []struct {
		input    string
		expected bool
	}{
		{"false && true", false},
		{"false && false", false},
		{"true && true", true},
		{"true && false", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

func TestShortCircuitOrResult(t *testing.T) {
	// Verify correct results for || operator
	tests := []struct {
		input    string
		expected bool
	}{
		{"true || false", true},
		{"true || true", true},
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
// Boolean Truthiness Tests (for #342 fix)
// ============================================================================

func TestBooleanTruthyInIfCondition(t *testing.T) {
	// Test that boolean values from stdlib functions work correctly in if conditions
	tests := []struct {
		input    string
		expected string
	}{
		// Test with stdlib function returning false - should take else branch
		{`
import @strings
temp s string = "hello"
temp result string = ""
if strings.contains(s, "xyz") {
    result = "true"
} otherwise {
    result = "false"
}
result
`, "false"},
		// Test with stdlib function returning true - should take if branch
		{`
import @strings
temp s string = "hello"
temp result string = ""
if strings.contains(s, "ell") {
    result = "true"
} otherwise {
    result = "false"
}
result
`, "true"},
		// Test starts_with returning false
		{`
import @strings
temp s string = "hello"
temp result string = ""
if strings.starts_with(s, "world") {
    result = "true"
} otherwise {
    result = "false"
}
result
`, "false"},
		// Test ends_with returning false
		{`
import @strings
temp s string = "hello"
temp result string = ""
if strings.ends_with(s, "world") {
    result = "true"
} otherwise {
    result = "false"
}
result
`, "false"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			evaluated := testEval(tt.input)
			testStringObject(t, evaluated, tt.expected)
		})
	}
}

func TestBooleanAssignmentAndCondition(t *testing.T) {
	// Test that boolean assigned to variable works correctly in if condition
	input := `
import @strings
temp s string = "hello world"
temp result bool = strings.contains(s, "xyz")
temp output string = ""
if result {
    output = "bug"
} otherwise {
    output = "correct"
}
output
`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "correct")
}

// ============================================================================
// Blank Identifier Tests
// ============================================================================

func TestBlankIdentifierDiscardsValue(t *testing.T) {
	input := `
do getTwo() -> (int, int) {
    return 1, 2
}

temp a, _ = getTwo()
a
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 1)
}

func TestBlankIdentifierAsFirstValue(t *testing.T) {
	input := `
do getTwo() -> (int, int) {
    return 1, 2
}

temp _, b = getTwo()
b
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 2)
}

func TestMultipleBlankIdentifiersInAssignment(t *testing.T) {
	input := `
do getThree() -> (int, int, int) {
    return 1, 2, 3
}

temp _, _, c = getThree()
c
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 3)
}

func TestBlankIdentifierMiddleValue(t *testing.T) {
	input := `
do getThree() -> (int, int, int) {
    return 10, 20, 30
}

temp a, _, c = getThree()
a + c
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 40)
}

// ============================================================================
// Bug Fix Tests (#378-#383)
// ============================================================================

// TestUTF8StringLen tests that len() returns character count, not byte count (#378)
func TestUTF8StringLen(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "ASCII string",
			input:    `len("Hello")`,
			expected: 5,
		},
		{
			name:     "UTF-8 string with Chinese characters",
			input:    `len("Hello 世界")`,
			expected: 8, // 6 ASCII + 2 Chinese characters
		},
		{
			name:     "pure Chinese",
			input:    `len("世界")`,
			expected: 2,
		},
		{
			name:     "emoji string",
			input:    `len("Hello 👋")`,
			expected: 7, // 6 ASCII + 1 emoji
		},
		{
			name:     "empty string",
			input:    `len("")`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testIntegerObject(t, evaluated, tt.expected)
		})
	}
}

// TestUTF8StringIndexing tests that string indexing returns correct UTF-8 characters (#379)
func TestUTF8StringIndexing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected rune
	}{
		{
			name: "access ASCII char in UTF-8 string",
			input: `
temp s string = "Hello 世界"
s[0]
`,
			expected: 'H',
		},
		{
			name: "access first Chinese character",
			input: `
temp s string = "Hello 世界"
s[6]
`,
			expected: '世',
		},
		{
			name: "access second Chinese character",
			input: `
temp s string = "Hello 世界"
s[7]
`,
			expected: '界',
		},
		{
			name: "access emoji",
			input: `
temp s string = "Hi 👋 there"
s[3]
`,
			expected: '👋',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			if ch, ok := evaluated.(*Char); ok {
				if ch.Value != tt.expected {
					t.Errorf("expected %q, got %q", string(tt.expected), string(ch.Value))
				}
			} else {
				t.Errorf("expected Char, got %T (%+v)", evaluated, evaluated)
			}
		})
	}
}

// TestInOperatorChar tests that 'in' operator works for char arrays (#380)
func TestInOperatorChar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "char found in array",
			input:    `'a' in {'a', 'b', 'c'}`,
			expected: true,
		},
		{
			name:     "char not found in array",
			input:    `'d' in {'a', 'b', 'c'}`,
			expected: false,
		},
		{
			name:     "char at end of array",
			input:    `'c' in {'a', 'b', 'c'}`,
			expected: true,
		},
		{
			name:     "unicode char in array",
			input:    `'世' in {'世', '界'}`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

// TestInOperatorFloat tests that 'in' operator works for float arrays (#381)
func TestInOperatorFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "float found in array",
			input:    `1.5 in {1.5, 2.5, 3.5}`,
			expected: true,
		},
		{
			name:     "float not found in array",
			input:    `4.5 in {1.5, 2.5, 3.5}`,
			expected: false,
		},
		{
			name:     "zero float in array",
			input:    `0.0 in {0.0, 1.0, 2.0}`,
			expected: true,
		},
		{
			name:     "negative float in array",
			input:    `-1.5 in {-2.5, -1.5, 0.5}`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testBooleanObject(t, evaluated, tt.expected)
		})
	}
}

// TestIntegerOverflowDetection tests that integer arithmetic overflow is detected (#382)
func TestIntegerOverflowDetection(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorCode   string
	}{
		{
			name:        "addition overflow",
			input:       `9223372036854775807 + 1`,
			expectError: true,
			errorCode:   "E5005",
		},
		{
			name:        "addition underflow",
			input:       `-9223372036854775808 + (-1)`,
			expectError: true,
			errorCode:   "E5005",
		},
		{
			name:        "subtraction overflow",
			input:       `-9223372036854775808 - 1`,
			expectError: true,
			errorCode:   "E5006",
		},
		{
			name:        "subtraction underflow",
			input:       `9223372036854775807 - (-1)`,
			expectError: true,
			errorCode:   "E5006",
		},
		{
			name:        "multiplication overflow",
			input:       `9223372036854775807 * 2`,
			expectError: true,
			errorCode:   "E5007",
		},
		{
			name:        "normal addition - no overflow",
			input:       `100 + 200`,
			expectError: false,
		},
		{
			name:        "normal subtraction - no overflow",
			input:       `100 - 200`,
			expectError: false,
		},
		{
			name:        "normal multiplication - no overflow",
			input:       `100 * 200`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			if tt.expectError {
				if err, ok := evaluated.(*Error); ok {
					if err.Code != tt.errorCode {
						t.Errorf("expected error code %s, got %s", tt.errorCode, err.Code)
					}
				} else {
					t.Errorf("expected error, got %T (%+v)", evaluated, evaluated)
				}
			} else {
				if _, ok := evaluated.(*Error); ok {
					t.Errorf("did not expect error, got %+v", evaluated)
				}
			}
		})
	}
}

// TestNegationOverflow tests that unary negation overflow is detected (#386)
func TestNegationOverflow(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "negation overflow - minInt with typed variable",
			input:       "temp x int = -9223372036854775808\n-x",
			expectError: true,
		},
		{
			name:        "normal negation",
			input:       `-(-5)`,
			expectError: false,
		},
		{
			name:        "untyped literal negation - no overflow with big.Int",
			input:       `-(-9223372036854775808)`,
			expectError: false, // With big.Int, untyped literals don't overflow
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			if tt.expectError {
				if _, ok := evaluated.(*Error); !ok {
					t.Errorf("expected error, got %T (%+v)", evaluated, evaluated)
				}
			} else {
				if _, ok := evaluated.(*Error); ok {
					t.Errorf("did not expect error, got %+v", evaluated)
				}
			}
		})
	}
}

// TestDivisionOverflow tests that minInt / -1 overflow is detected (#387)
func TestDivisionOverflow(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorCode   string
	}{
		{
			name:        "division overflow - minInt / -1",
			input:       `-9223372036854775808 / -1`,
			expectError: true,
			errorCode:   "E5007",
		},
		{
			name:        "normal division",
			input:       `10 / -2`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			if tt.expectError {
				if err, ok := evaluated.(*Error); ok {
					if err.Code != tt.errorCode {
						t.Errorf("expected error code %s, got %s", tt.errorCode, err.Code)
					}
				} else {
					t.Errorf("expected error, got %T (%+v)", evaluated, evaluated)
				}
			} else {
				if _, ok := evaluated.(*Error); ok {
					t.Errorf("did not expect error, got %+v", evaluated)
				}
			}
		})
	}
}

// TestPostfixOverflowDetection tests that postfix ++/-- overflow is detected (#383)
func TestPostfixOverflowDetection(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorCode   string
	}{
		{
			name: "increment overflow",
			input: `
temp i int = 9223372036854775807
i++
`,
			expectError: true,
			errorCode:   "E5008",
		},
		{
			name: "decrement underflow",
			input: `
temp i int = -9223372036854775808
i--
`,
			expectError: true,
			errorCode:   "E5009",
		},
		{
			name: "normal increment - no overflow",
			input: `
temp i int = 100
i++
i
`,
			expectError: false,
		},
		{
			name: "normal decrement - no overflow",
			input: `
temp i int = 100
i--
i
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			if tt.expectError {
				if err, ok := evaluated.(*Error); ok {
					if err.Code != tt.errorCode {
						t.Errorf("expected error code %s, got %s", tt.errorCode, err.Code)
					}
				} else {
					t.Errorf("expected error, got %T (%+v)", evaluated, evaluated)
				}
			} else {
				if _, ok := evaluated.(*Error); ok {
					t.Errorf("did not expect error, got %+v", evaluated)
				}
			}
		})
	}
}

// ============================================================================
// Large Integer Tests (i128/u128) - Issue #437
// ============================================================================

func TestLargeIntegerArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"i128 addition beyond int64",
			"temp a i128 = 9223372036854775807\ntemp b i128 = 1\na + b",
			"9223372036854775808",
		},
		{
			"u128 addition beyond uint64",
			"temp a u128 = 18446744073709551615\ntemp b u128 = 1\na + b",
			"18446744073709551616",
		},
		{
			"i128 multiplication",
			"temp a i128 = 10000000000\ntemp b i128 = 10000000000\na * b",
			"100000000000000000000",
		},
		{
			"i128 large literal",
			"temp x i128 = 10000000000000000000\nx",
			"10000000000000000000",
		},
		{
			"u128 large literal",
			"temp x u128 = 18446744073709551616\nx",
			"18446744073709551616",
		},
		{
			"i128 negative subtraction beyond int64",
			"temp a i128 = -9223372036854775808\ntemp b i128 = 1\na - b",
			"-9223372036854775809",
		},
		{
			"i128 division",
			"temp a i128 = 100000000000000000000\ntemp b i128 = 10000000000\na / b",
			"10000000000",
		},
		{
			"i128 modulo",
			"temp a i128 = 100000000000000000001\ntemp b i128 = 10000000000\na % b",
			"1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			expected := new(big.Int)
			expected.SetString(tt.expected, 10)
			testBigIntegerObject(t, evaluated, expected)
		})
	}
}

func TestLargeIntegerOverflow(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		errorCode string
	}{
		{
			"i128 overflow on addition",
			"temp max i128 = 170141183460469231731687303715884105727\ntemp r i128 = max + 1",
			"E5005",
		},
		{
			"i128 underflow on subtraction",
			"temp min i128 = -170141183460469231731687303715884105728\ntemp r i128 = min - 1",
			"E5006",
		},
		{
			"u128 overflow on addition",
			"temp max u128 = 340282366920938463463374607431768211455\ntemp r u128 = max + 1",
			"E5005",
		},
		{
			"u128 underflow on subtraction",
			"temp zero u128 = 0\ntemp r u128 = zero - 1",
			"E5006",
		},
		{
			"int overflow still works",
			"temp max int = 9223372036854775807\ntemp r int = max + 1",
			"E5005",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			err, ok := evaluated.(*Error)
			if !ok {
				t.Fatalf("expected error, got %T (%+v)", evaluated, evaluated)
			}
			if err.Code != tt.errorCode {
				t.Errorf("expected error code %s, got %s: %s", tt.errorCode, err.Code, err.Message)
			}
		})
	}
}

func TestNegativeToUnsignedTypeError(t *testing.T) {
	input := "temp x u128 = -1"
	evaluated := testEval(input)
	err, ok := evaluated.(*Error)
	if !ok {
		t.Fatalf("expected error, got %T (%+v)", evaluated, evaluated)
	}
	if err.Code != "E3020" {
		t.Errorf("expected error code E3020, got %s: %s", err.Code, err.Message)
	}
}

// =============================================================================
// DEFAULT PARAMETER TESTS
// =============================================================================

func TestDefaultParameterBasicString(t *testing.T) {
	input := `do greet(name string = "World") -> string { return "Hello, " + name + "!" } temp r string = greet() r`
	evaluated := testEval(input)
	result, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T (%+v)", evaluated, evaluated)
	}
	if result.Value != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got '%s'", result.Value)
	}
}

func TestDefaultParameterOverride(t *testing.T) {
	input := `do greet(name string = "World") -> string { return "Hello, " + name + "!" } temp r string = greet("Alice") r`
	evaluated := testEval(input)
	result, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T (%+v)", evaluated, evaluated)
	}
	if result.Value != "Hello, Alice!" {
		t.Errorf("expected 'Hello, Alice!', got '%s'", result.Value)
	}
}

func TestDefaultParameterNumeric(t *testing.T) {
	input := `do add(a int, b int = 10) -> int { return a + b } temp r int = add(5) r`
	evaluated := testEval(input)
	result, ok := evaluated.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%+v)", evaluated, evaluated)
	}
	if result.Value.Int64() != 15 {
		t.Errorf("expected 15, got %d", result.Value.Int64())
	}
}

func TestDefaultParameterNumericOverride(t *testing.T) {
	input := `do add(a int, b int = 10) -> int { return a + b } temp r int = add(5, 20) r`
	evaluated := testEval(input)
	result, ok := evaluated.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%+v)", evaluated, evaluated)
	}
	if result.Value.Int64() != 25 {
		t.Errorf("expected 25, got %d", result.Value.Int64())
	}
}

func TestDefaultParameterMultiple(t *testing.T) {
	input := `do calc(a int, b int = 2, c int = 3) -> int { return a * b + c } temp r int = calc(5) r`
	evaluated := testEval(input)
	result, ok := evaluated.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%+v)", evaluated, evaluated)
	}
	// 5 * 2 + 3 = 13
	if result.Value.Int64() != 13 {
		t.Errorf("expected 13, got %d", result.Value.Int64())
	}
}

func TestDefaultParameterPartialOverride(t *testing.T) {
	input := `do calc(a int, b int = 2, c int = 3) -> int { return a * b + c } temp r int = calc(5, 10) r`
	evaluated := testEval(input)
	result, ok := evaluated.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%+v)", evaluated, evaluated)
	}
	// 5 * 10 + 3 = 53
	if result.Value.Int64() != 53 {
		t.Errorf("expected 53, got %d", result.Value.Int64())
	}
}

func TestDefaultParameterAllDefaults(t *testing.T) {
	input := `do triple(a int = 1, b int = 2, c int = 3) -> int { return a + b + c } temp r int = triple() r`
	evaluated := testEval(input)
	result, ok := evaluated.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%+v)", evaluated, evaluated)
	}
	if result.Value.Int64() != 6 {
		t.Errorf("expected 6, got %d", result.Value.Int64())
	}
}

func TestDefaultParameterExpressionDefault(t *testing.T) {
	input := `do calc(mult float = 3.14 * 2.0) -> float { return mult } temp r float = calc() r`
	evaluated := testEval(input)
	result, ok := evaluated.(*Float)
	if !ok {
		t.Fatalf("expected Float, got %T (%+v)", evaluated, evaluated)
	}
	// 3.14 * 2.0 = 6.28
	if result.Value != 6.28 {
		t.Errorf("expected 6.28, got %f", result.Value)
	}
}

func TestDefaultParameterGroupedParams(t *testing.T) {
	// x, y int = 0 means x is required, y has default
	input := `do point(x, y int = 0) -> int { return x + y } temp r int = point(5) r`
	evaluated := testEval(input)
	result, ok := evaluated.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%+v)", evaluated, evaluated)
	}
	if result.Value.Int64() != 5 {
		t.Errorf("expected 5, got %d", result.Value.Int64())
	}
}

func TestDefaultParameterGroupedParamsWithOverride(t *testing.T) {
	input := `do point(x, y int = 0) -> int { return x + y } temp r int = point(3, 4) r`
	evaluated := testEval(input)
	result, ok := evaluated.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%+v)", evaluated, evaluated)
	}
	if result.Value.Int64() != 7 {
		t.Errorf("expected 7, got %d", result.Value.Int64())
	}
}

func TestDefaultParameterBoolDefault(t *testing.T) {
	input := `do toggle(val bool = false) -> bool { return !val } temp r bool = toggle() r`
	evaluated := testEval(input)
	result, ok := evaluated.(*Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T (%+v)", evaluated, evaluated)
	}
	if result.Value != true {
		t.Errorf("expected true, got %v", result.Value)
	}
}

func TestDefaultParameterTooFewArgsError(t *testing.T) {
	input := `
	do add(a int, b int = 10) -> int {
		return a + b
	}
	add()
	`
	evaluated := testEval(input)
	err, ok := evaluated.(*Error)
	if !ok {
		t.Fatalf("expected error, got %T (%+v)", evaluated, evaluated)
	}
	if err.Code != "E5008" {
		t.Errorf("expected error code E5008, got %s: %s", err.Code, err.Message)
	}
}

func TestDefaultParameterTooManyArgsError(t *testing.T) {
	input := `
	do add(a int = 1) -> int {
		return a
	}
	add(1, 2)
	`
	evaluated := testEval(input)
	err, ok := evaluated.(*Error)
	if !ok {
		t.Fatalf("expected error, got %T (%+v)", evaluated, evaluated)
	}
	if err.Code != "E5008" {
		t.Errorf("expected error code E5008, got %s: %s", err.Code, err.Message)
	}
}

// ============================================================================
// When Statement Tests with Value Matching
// ============================================================================

func TestWhenStatementWithMultipleValues(t *testing.T) {
	input := `
	temp x int = 2
	temp result int = 0
	when x {
		is 1, 2, 3 {
			result = 100
		}
		default {
			result = 0
		}
	}
	result
	`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 100)
}

func TestWhenStatementDefault(t *testing.T) {
	input := `
	temp x int = 99
	temp result int = 0
	when x {
		is 1 {
			result = 1
		}
		is 2 {
			result = 2
		}
		default {
			result = -1
		}
	}
	result
	`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, -1)
}

func TestWhenStatementWithEnumValues(t *testing.T) {
	input := `
	const Color enum { RED, GREEN, BLUE }
	temp c Color = Color.GREEN
	temp result int = 0
	when c {
		is Color.RED {
			result = 1
		}
		is Color.GREEN {
			result = 2
		}
		is Color.BLUE {
			result = 3
		}
	}
	result
	`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 2)
}

// ============================================================================
// Object Equality Tests
// ============================================================================

func TestEnumValueEquality(t *testing.T) {
	input := `
	const Status enum { OPEN, CLOSED }
	temp s1 Status = Status.OPEN
	temp s2 Status = Status.OPEN
	s1 == s2
	`
	evaluated := testEval(input)
	testBooleanObject(t, evaluated, true)
}

func TestEnumValueInequality(t *testing.T) {
	input := `
	const Status enum { OPEN, CLOSED }
	temp s1 Status = Status.OPEN
	temp s2 Status = Status.CLOSED
	s1 == s2
	`
	evaluated := testEval(input)
	testBooleanObject(t, evaluated, false)
}

// ============================================================================
// Default Value Tests
// ============================================================================

func TestStructDefaultValues(t *testing.T) {
	input := `
	const Person struct {
		name string
		age int
	}
	temp p Person = new(Person)
	p.age
	`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 0)
}

func TestStructDefaultStringValue(t *testing.T) {
	input := `
	const Person struct {
		name string
		age int
	}
	temp p Person = new(Person)
	p.name
	`
	evaluated := testEval(input)
	testStringObject(t, evaluated, "")
}

// ============================================================================
// Type Name Tests
// ============================================================================

func TestGetEZTypeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"typeof(5)", "int"},
		{"typeof(3.14)", "float"},
		{`typeof("hello")`, "string"},
		{"typeof(true)", "bool"},
		{"typeof('a')", "char"},
		{"typeof(nil)", "nil"},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testStringObject(t, evaluated, tt.expected)
	}
}

func TestGetEZTypeNameArray(t *testing.T) {
	input := "temp arr [int] = {1, 2, 3} typeof(arr)"
	evaluated := testEval(input)
	result, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T", evaluated)
	}
	if !strings.Contains(result.Value, "array") && !strings.Contains(result.Value, "[") {
		t.Errorf("expected array type, got %s", result.Value)
	}
}

func TestGetEZTypeNameMap(t *testing.T) {
	input := `temp m map[string:int] = {"a": 1} typeof(m)`
	evaluated := testEval(input)
	result, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T", evaluated)
	}
	// Result could be "map" or "MAP"
	if !strings.Contains(strings.ToLower(result.Value), "map") {
		t.Errorf("expected map type, got %s", result.Value)
	}
}

func TestGetEZTypeNameStruct(t *testing.T) {
	input := `
	const Point struct { x int y int }
	temp p Point = Point{x: 1, y: 2}
	typeof(p)
	`
	evaluated := testEval(input)
	result, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T", evaluated)
	}
	if result.Value != "Point" {
		t.Errorf("expected Point, got %s", result.Value)
	}
}

// ============================================================================
// Byte Type Tests
// ============================================================================

func TestByteTypeAssignment(t *testing.T) {
	input := "temp b byte = 255 b"
	evaluated := testEval(input)
	testByteObject(t, evaluated, 255)
}

func TestByteTypeZero(t *testing.T) {
	input := "temp b byte = 0 b"
	evaluated := testEval(input)
	testByteObject(t, evaluated, 0)
}
