package typechecker

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"testing"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/lexer"
	"github.com/marshallburns/ez/pkg/parser"
)

// ============================================================================
// Test Helpers
// ============================================================================

func typecheck(t *testing.T, input string) *TypeChecker {
	t.Helper()
	l := lexer.NewLexer(input)
	p := parser.New(l)
	program := p.ParseProgram()

	// Check for parser errors
	if len(p.Errors()) > 0 {
		for _, errMsg := range p.Errors() {
			t.Errorf("parser error: %s", errMsg)
		}
		t.FailNow()
	}

	tc := NewTypeChecker(input, "test.ez")
	tc.CheckProgram(program)
	return tc
}

func typecheckModule(t *testing.T, input string) *TypeChecker {
	t.Helper()
	l := lexer.NewLexer(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, errMsg := range p.Errors() {
			t.Errorf("parser error: %s", errMsg)
		}
		t.FailNow()
	}

	tc := NewTypeChecker(input, "module.ez")
	tc.SetSkipMainCheck(true)
	tc.CheckProgram(program)
	return tc
}

func assertNoErrors(t *testing.T, tc *TypeChecker) {
	t.Helper()
	errCount, _ := tc.Errors().Count()
	if errCount > 0 {
		for _, err := range tc.Errors().Errors {
			t.Errorf("unexpected error: [%s] %s at line %d", err.ErrorCode.Code, err.Message, err.Line)
		}
	}
}

func assertHasError(t *testing.T, tc *TypeChecker, expectedCode errors.ErrorCode) {
	t.Helper()
	for _, err := range tc.Errors().Errors {
		if err.ErrorCode == expectedCode {
			return
		}
	}
	t.Errorf("expected error %s but not found", expectedCode.Code)
	for _, err := range tc.Errors().Errors {
		t.Logf("  found: [%s] %s", err.ErrorCode.Code, err.Message)
	}
}

func assertHasWarning(t *testing.T, tc *TypeChecker, expectedCode errors.ErrorCode) {
	t.Helper()
	for _, warn := range tc.Errors().Warnings {
		if warn.ErrorCode == expectedCode {
			return
		}
	}
	t.Errorf("expected warning %s but not found", expectedCode.Code)
}

// ============================================================================
// Scope Tests
// ============================================================================

func TestNewScope(t *testing.T) {
	scope := NewScope(nil)
	if scope == nil {
		t.Fatal("NewScope returned nil")
	}
	if scope.parent != nil {
		t.Error("root scope should have nil parent")
	}
	if scope.variables == nil {
		t.Error("variables map should be initialized")
	}
	if scope.usingModules == nil {
		t.Error("usingModules map should be initialized")
	}
}

func TestScopeDefineAndLookup(t *testing.T) {
	scope := NewScope(nil)
	scope.Define("x", "int")

	typeName, found := scope.Lookup("x")
	if !found {
		t.Error("expected to find variable x")
	}
	if typeName != "int" {
		t.Errorf("expected type 'int', got '%s'", typeName)
	}

	// Lookup non-existent variable
	_, found = scope.Lookup("y")
	if found {
		t.Error("should not find undefined variable y")
	}
}

func TestScopeNesting(t *testing.T) {
	parent := NewScope(nil)
	parent.Define("x", "int")

	child := NewScope(parent)
	child.Define("y", "string")

	// Child can see parent's variables
	typeName, found := child.Lookup("x")
	if !found {
		t.Error("child should find parent's variable x")
	}
	if typeName != "int" {
		t.Errorf("expected type 'int', got '%s'", typeName)
	}

	// Child can see its own variables
	typeName, found = child.Lookup("y")
	if !found {
		t.Error("child should find its own variable y")
	}
	if typeName != "string" {
		t.Errorf("expected type 'string', got '%s'", typeName)
	}

	// Parent cannot see child's variables
	_, found = parent.Lookup("y")
	if found {
		t.Error("parent should not find child's variable y")
	}
}

func TestScopeShadowing(t *testing.T) {
	parent := NewScope(nil)
	parent.Define("x", "int")

	child := NewScope(parent)
	child.Define("x", "string") // Shadow parent's x

	// Child sees its own x
	typeName, found := child.Lookup("x")
	if !found {
		t.Error("child should find variable x")
	}
	if typeName != "string" {
		t.Errorf("expected shadowed type 'string', got '%s'", typeName)
	}

	// Parent still sees its own x
	typeName, found = parent.Lookup("x")
	if !found {
		t.Error("parent should find variable x")
	}
	if typeName != "int" {
		t.Errorf("expected original type 'int', got '%s'", typeName)
	}
}

func TestScopeUsingModules(t *testing.T) {
	parent := NewScope(nil)
	parent.AddUsingModule("arrays")

	child := NewScope(parent)
	child.AddUsingModule("strings")

	// Child can see parent's using modules
	if !child.HasUsingModule("arrays") {
		t.Error("child should have access to parent's using module 'arrays'")
	}

	// Child can see its own using modules
	if !child.HasUsingModule("strings") {
		t.Error("child should have access to its own using module 'strings'")
	}

	// Parent cannot see child's using modules
	if parent.HasUsingModule("strings") {
		t.Error("parent should not have access to child's using module 'strings'")
	}
}

func TestScopeGetAllUsingModules(t *testing.T) {
	parent := NewScope(nil)
	parent.AddUsingModule("arrays")
	parent.AddUsingModule("math")

	child := NewScope(parent)
	child.AddUsingModule("strings")

	modules := child.GetAllUsingModules()
	if len(modules) != 3 {
		t.Errorf("expected 3 modules, got %d", len(modules))
	}

	// Check all modules are present
	moduleSet := make(map[string]bool)
	for _, m := range modules {
		moduleSet[m] = true
	}
	for _, expected := range []string{"arrays", "math", "strings"} {
		if !moduleSet[expected] {
			t.Errorf("expected module '%s' not found", expected)
		}
	}
}

// ============================================================================
// Type Tests
// ============================================================================

func TestBuiltinTypes(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")

	primitives := []string{
		"i8", "i16", "i32", "i64", "i128", "i256", "int",
		"u8", "u16", "u32", "u64", "u128", "u256", "uint",
		"f32", "f64", "float",
		"bool", "char", "string",
		"void", "nil",
	}

	for _, typeName := range primitives {
		if !tc.TypeExists(typeName) {
			t.Errorf("primitive type '%s' should exist", typeName)
		}
	}

	// Check Error struct exists
	if !tc.TypeExists("Error") {
		t.Error("Error struct should exist")
	}
}

func TestTypeExistsForArrayTypes(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")

	// Array types should be recognized
	if !tc.TypeExists("[int]") {
		t.Error("array type [int] should exist")
	}
	if !tc.TypeExists("[string]") {
		t.Error("array type [string] should exist")
	}
	if !tc.TypeExists("[int, 5]") {
		t.Error("fixed-size array type [int, 5] should exist")
	}
}

func TestTypeExistsForMapTypes(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")

	// Valid map types
	if !tc.TypeExists("map[string:int]") {
		t.Error("map type map[string:int] should exist")
	}
	if !tc.TypeExists("map[int:string]") {
		t.Error("map type map[int:string] should exist")
	}

	// Invalid map type (missing value type)
	if tc.TypeExists("map[string]") {
		t.Error("invalid map type map[string] should not exist")
	}
}

func TestRegisterAndGetType(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")

	customType := &Type{
		Name: "Person",
		Kind: StructType,
		Fields: map[string]*Type{
			"name": {Name: "string", Kind: PrimitiveType},
			"age":  {Name: "int", Kind: PrimitiveType},
		},
	}

	tc.RegisterType("Person", customType)

	retrieved, ok := tc.GetType("Person")
	if !ok {
		t.Error("should find registered type Person")
	}
	if retrieved.Name != "Person" {
		t.Errorf("expected name 'Person', got '%s'", retrieved.Name)
	}
	if retrieved.Kind != StructType {
		t.Errorf("expected StructType, got %v", retrieved.Kind)
	}
}

func TestRegisterFunction(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")

	sig := &FunctionSignature{
		Name: "add",
		Parameters: []*Parameter{
			{Name: "a", Type: "int"},
			{Name: "b", Type: "int"},
		},
		ReturnTypes: []string{"int"},
	}

	tc.RegisterFunction("add", sig)

	// Function should be registered in tc.functions
	if tc.functions["add"] == nil {
		t.Error("function 'add' should be registered")
	}
	if tc.functions["add"].Name != "add" {
		t.Errorf("expected function name 'add', got '%s'", tc.functions["add"].Name)
	}
}

// ============================================================================
// Program Type Checking Tests
// ============================================================================

func TestValidProgram(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	temp y int = 10
	temp sum int = x + y
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMissingMainFunction(t *testing.T) {
	input := `
do helper() {
	temp x int = 5
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E4009)
}

func TestModuleWithoutMain(t *testing.T) {
	input := `
do helper() -> int {
	return 42
}
`
	tc := typecheckModule(t, input)
	assertNoErrors(t, tc) // Should not require main() for modules
}

// ============================================================================
// Variable Declaration Tests
// ============================================================================

func TestValidVariableDeclarations(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"integer",
			`do main() { temp x int = 5 }`,
		},
		{
			"float",
			`do main() { temp f float = 3.14 }`,
		},
		{
			"string",
			`do main() { temp s string = "hello" }`,
		},
		{
			"bool",
			`do main() { temp b bool = true }`,
		},
		{
			"char",
			`do main() { temp c char = 'a' }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := typecheck(t, tt.input)
			assertNoErrors(t, tc)
		})
	}
}

func TestUndefinedType(t *testing.T) {
	input := `
do main() {
	temp x UndefinedType = 5
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3008)
}

func TestTypeMismatchInDeclaration(t *testing.T) {
	input := `
do main() {
	temp x int = "hello"
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

// ============================================================================
// Struct Tests
// ============================================================================

func TestValidStructDeclaration(t *testing.T) {
	input := `
const Person struct {
	name string
	age int
}

do main() {
	temp p Person = Person{name: "Alice", age: 30}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStructWithUndefinedFieldType(t *testing.T) {
	input := `
const Person struct {
	name UndefinedType
}

do main() {}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3009)
}

func TestStructFieldAccess(t *testing.T) {
	input := `
const Person struct {
	name string
	age int
}

do main() {
	temp p Person = Person{name: "", age: 0}
	p.name = "Alice"
	p.age = 30
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStructWithArrayField(t *testing.T) {
	// Regression test for issue #336: typechecker crashed when accessing
	// struct fields with array types due to nil pointer dereference
	input := `
const TestResult struct {
	passed int
	bugs [string]
}

do main() {
	temp r TestResult = TestResult{passed: 1, bugs: {"bug1"}}
	temp bug_list [string] = r.bugs
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStructWithMapField(t *testing.T) {
	// Ensure struct fields with map types also work correctly
	input := `
const Config struct {
	name string
	settings map[string:int]
}

do main() {
	temp c Config = Config{name: "test", settings: {"a": 1}}
	temp s map[string:int] = c.settings
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStructWithNestedArrayField(t *testing.T) {
	// Test nested array types in struct fields
	input := `
const Matrix struct {
	rows [[int]]
}

do main() {
	temp m Matrix = Matrix{rows: {{1, 2}, {3, 4}}}
	temp r [[int]] = m.rows
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Enum Tests
// ============================================================================

func TestValidEnumDeclaration(t *testing.T) {
	input := `
const Color enum {
	Red
	Green
	Blue
}

do main() {}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Function Declaration Tests
// ============================================================================

func TestValidFunctionDeclaration(t *testing.T) {
	input := `
do add(a int, b int) -> int {
	return a + b
}

do main() {
	temp result int = add(1, 2)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFunctionWithUndefinedParamType(t *testing.T) {
	input := `
do process(x UndefinedType) {
}

do main() {}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3010)
}

func TestFunctionWithUndefinedReturnType(t *testing.T) {
	input := `
do getValue() -> UndefinedType {
	return nil
}

do main() {}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3011)
}

func TestFunctionMissingReturnError(t *testing.T) {
	input := `
do getValue() -> int {
	temp x int = 5
}

do main() {}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3024)
}

func TestFunctionWithReturn(t *testing.T) {
	input := `
do getValue() -> int {
	return 42
}

do main() {}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Return Statement Tests
// ============================================================================

func TestReturnTypeMismatch(t *testing.T) {
	input := `
do getValue() -> int {
	return "hello"
}

do main() {}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3012)
}

func TestReturnInVoidFunction(t *testing.T) {
	input := `
do doSomething() {
	return 42
}

do main() {}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3012)
}

func TestWrongReturnCount(t *testing.T) {
	input := `
do getTwoValues() -> (int, string) {
	return 42
}

do main() {}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3013)
}

func TestMultipleReturnValues(t *testing.T) {
	input := `
do getTwoValues() -> (int, string) {
	return 42, "hello"
}

do main() {}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Operator Type Checking Tests
// ============================================================================

func TestValidArithmeticOperations(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"int addition",
			`do main() { temp r int = 1 + 2 }`,
		},
		{
			"float addition",
			`do main() { temp r float = 1.5 + 2.5 }`,
		},
		{
			"string concatenation",
			`do main() { temp r string = "a" + "b" }`,
		},
		{
			"int subtraction",
			`do main() { temp r int = 5 - 3 }`,
		},
		{
			"int multiplication",
			`do main() { temp r int = 2 * 3 }`,
		},
		{
			"int division",
			`do main() { temp r int = 6 / 2 }`,
		},
		{
			"int modulo",
			`do main() { temp r int = 7 % 3 }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := typecheck(t, tt.input)
			assertNoErrors(t, tc)
		})
	}
}

func TestValidComparisonOperations(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"int equality",
			`do main() { temp r bool = 1 == 1 }`,
		},
		{
			"int inequality",
			`do main() { temp r bool = 1 != 2 }`,
		},
		{
			"int less than",
			`do main() { temp r bool = 1 < 2 }`,
		},
		{
			"int greater than",
			`do main() { temp r bool = 2 > 1 }`,
		},
		{
			"int less or equal",
			`do main() { temp r bool = 1 <= 2 }`,
		},
		{
			"int greater or equal",
			`do main() { temp r bool = 2 >= 1 }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := typecheck(t, tt.input)
			assertNoErrors(t, tc)
		})
	}
}

func TestValidLogicalOperations(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"logical and",
			`do main() { temp r bool = true && false }`,
		},
		{
			"logical or",
			`do main() { temp r bool = true || false }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := typecheck(t, tt.input)
			assertNoErrors(t, tc)
		})
	}
}

// ============================================================================
// Array Tests
// ============================================================================

func TestValidArrayDeclaration(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {1, 2, 3}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArrayTypeMismatch(t *testing.T) {
	input := `
do main() {
	temp arr [int] = "not an array"
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3018)
}

func TestFixedSizeArrayWarning(t *testing.T) {
	input := `
do main() {
	temp arr [int, 5] = {1, 2}
}
`
	tc := typecheck(t, input)
	assertHasWarning(t, tc, errors.W3003)
}

// ============================================================================
// Control Flow Tests
// ============================================================================

func TestIfStatement(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	if x > 0 {
		temp y int = 10
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestForLoop(t *testing.T) {
	input := `
do main() {
	for i in range(0, 10) {
		temp x int = i
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestWhileLoop(t *testing.T) {
	input := `
do main() {
	temp x int = 0
	as_long_as x < 10 {
		x = x + 1
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Assignment Tests
// ============================================================================

func TestValidAssignment(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	x = 10
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestAssignmentTypeMismatch(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	x = "hello"
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

// ============================================================================
// TypeChecker Method Tests
// ============================================================================

func TestSetSkipMainCheck(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")
	if tc.skipMainCheck {
		t.Error("skipMainCheck should be false by default")
	}

	tc.SetSkipMainCheck(true)
	if !tc.skipMainCheck {
		t.Error("skipMainCheck should be true after setting")
	}
}

func TestErrorsMethod(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")
	errs := tc.Errors()
	if errs == nil {
		t.Error("Errors() should return non-nil error list")
	}
}

// ============================================================================
// Type Compatibility Tests
// ============================================================================

func TestNumericTypeCompatibility(t *testing.T) {
	input := `
do main() {
	temp i8val i8 = 1
	temp i16val i16 = 2
	temp i32val i32 = 3
	temp i64val i64 = 4
	temp intval int = 5
	temp f32val f32 = 1.0
	temp f64val f64 = 2.0
	temp floatval float = 3.0
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestUnsignedTypes(t *testing.T) {
	input := `
do main() {
	temp u8val u8 = 1
	temp u16val u16 = 2
	temp u32val u32 = 3
	temp u64val u64 = 4
	temp uintval uint = 5
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Import and Using Tests
// ============================================================================

func TestImportStatement(t *testing.T) {
	input := `
import @arrays

do main() {}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestUsingStatement(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestEmptyMain(t *testing.T) {
	input := `do main() {}`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedScopes(t *testing.T) {
	input := `
do main() {
	temp x int = 1
	if true {
		temp y int = 2
		if true {
			temp z int = x + y
		}
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestConstDeclaration(t *testing.T) {
	input := `
do main() {
	const PI float = 3.14159
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestConstTypeInference(t *testing.T) {
	input := `
const Person struct {
	name string
	age int
}

do main() {
	const p1 = new(Person)
	const p2 = Person{name: "Alice", age: 30}
	const x = 42
	const s = "hello"
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapDeclaration(t *testing.T) {
	input := `
do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNilAssignment(t *testing.T) {
	input := `
const Person struct {
	name string
}

do main() {
	temp p Person = nil
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestComplexProgram(t *testing.T) {
	input := `
const Point struct {
	x int
	y int
}

const Direction enum {
	North
	South
	East
	West
}

do distance(p1 Point, p2 Point) -> int {
	temp dx int = p2.x - p1.x
	temp dy int = p2.y - p1.y
	return dx + dy
}

do main() {
	temp origin Point = Point{x: 0, y: 0}
	temp target Point = Point{x: 3, y: 4}
	temp dist int = distance(origin, target)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFunctionCallingFunction(t *testing.T) {
	input := `
do double(x int) -> int {
	return x * 2
}

do quadruple(x int) -> int {
	return double(double(x))
}

do main() {
	temp result int = quadruple(5)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestRecursiveFunction(t *testing.T) {
	input := `
do factorial(n int) -> int {
	if n <= 1 {
		return 1
	}
	return n * factorial(n - 1)
}

do main() {
	temp result int = factorial(5)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Mutable Parameter Tests (& prefix)
// ============================================================================

func TestMutableParameterValid(t *testing.T) {
	// Test: temp variable passed to & param - should be valid
	input := `
const Person struct {
	name string
	age int
}

do birthday(&p Person) {
	p.age = p.age + 1
}

do main() {
	temp bob Person = Person{name: "Bob", age: 30}
	birthday(bob)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestImmutableParameterValid(t *testing.T) {
	// Test: const variable passed to non-& param - should be valid
	input := `
const Person struct {
	name string
	age int
}

do getName(p Person) -> string {
	return p.name
}

do main() {
	const alice Person = Person{name: "Alice", age: 25}
	temp name string = getName(alice)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestE3023_ConstToMutableParam(t *testing.T) {
	// Test: const variable passed to & param - should error E3023
	input := `
const Person struct {
	name string
	age int
}

do modify(&p Person) {
	p.age = p.age + 1
}

do main() {
	const alice Person = Person{name: "Alice", age: 25}
	modify(alice)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3023)
}

func TestE5016_ModifyImmutableParam(t *testing.T) {
	// Test: modifying a non-& param inside function - should error E5017 (struct field)
	input := `
const Person struct {
	name string
	age int
}

do tryModify(p Person) {
	p.age = 100
}

do main() {
	temp bob Person = Person{name: "Bob", age: 30}
	tryModify(bob)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E5017)
}

func TestE5016_ModifyImmutableParamField(t *testing.T) {
	// Test: modifying a field on non-& param - should error E5017 (struct field)
	input := `
const Person struct {
	name string
	age int
}

do tryModifyField(p Person) {
	p.name = "Changed"
}

do main() {
	temp bob Person = Person{name: "Bob", age: 30}
	tryModifyField(bob)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E5017)
}

func TestE5016_ModifyImmutableArrayParam(t *testing.T) {
	// Test: modifying array element on non-& param - should error E5016
	input := `
do tryModifyArray(arr [int]) {
	arr[0] = 100
}

do main() {
	temp nums [int] = {1, 2, 3}
	tryModifyArray(nums)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E5016)
}

func TestMutableArrayParamValid(t *testing.T) {
	// Test: modifying array element on & param - should be valid
	input := `
do modifyArray(&arr [int]) {
	arr[0] = 100
}

do main() {
	temp nums [int] = {1, 2, 3}
	modifyArray(nums)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMixedMutableParamsValid(t *testing.T) {
	// Test: function with both mutable and immutable params
	input := `
do copyFirst(&dest [int], src [int]) {
	dest[0] = src[0]
}

do main() {
	temp target [int] = {0, 0, 0}
	temp source [int] = {1, 2, 3}
	copyFirst(target, source)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestE3023_ConstArrayToMutableParam(t *testing.T) {
	// Test: const array passed to & param - should error E3023
	input := `
do modify(&arr [int]) {
	arr[0] = 100
}

do main() {
	const nums [int] = {1, 2, 3}
	modify(nums)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3023)
}

func TestFixedSizeArrayIndexType(t *testing.T) {
	// Test: indexing fixed-size array returns element type, not "type,size"
	// This was bug #267 - [int,3][0] was returning type "int,3" instead of "int"
	input := `
do main() {
	temp arr [int, 3] = {1, 2, 3}
	temp x int = arr[0]
	temp y int = arr[1] + arr[2]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Bug Fix Tests - December 2025
// ============================================================================

func TestE4011_MemberAccessOnPrimitive(t *testing.T) {
	// Test: accessing member on primitive type should error E4011
	// Fixes #313
	input := `
do main() {
	temp s = "hello"
	temp x = s.name
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E4011)
}

func TestMemberAccessOnStructValid(t *testing.T) {
	// Test: accessing member on struct should work
	input := `
const Person struct {
	name string
	age int
}

do main() {
	temp p = Person{name: "Alice", age: 30}
	temp n = p.name
	temp a = p.age
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// ForEach Statement Tests
// ============================================================================

func TestForEachStatementArray(t *testing.T) {
	input := `
do main() {
	temp nums [int] = {1, 2, 3}
	temp sum int = 0
	for_each n in nums {
		sum = sum + n
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestForEachStatementString(t *testing.T) {
	input := `
do main() {
	temp str string = "hello"
	temp count int = 0
	for_each c in str {
		count = count + 1
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestForEachStatementMap(t *testing.T) {
	input := `
do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
	for_each k in m {
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Loop Statement Tests
// ============================================================================

func TestLoopStatement(t *testing.T) {
	input := `
do main() {
	temp count int = 0
	loop {
		count = count + 1
		if count == 5 {
			break
		}
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestLoopStatementWithContinue(t *testing.T) {
	input := `
do main() {
	temp count int = 0
	loop {
		count = count + 1
		if count == 3 {
			continue
		}
		if count == 5 {
			break
		}
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Stdlib Call Tests
// ============================================================================

func TestMathModuleCall(t *testing.T) {
	input := `
import @math
using math

do main() {
	temp x float = math.sqrt(16.0)
	temp y float = math.abs(-5.0)
	temp z float = math.pow(2.0, 3.0)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArraysModuleCall(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp nums [int] = {1, 2, 3}
	temp first int = arrays.first(nums)
	temp last int = arrays.last(nums)
	temp rev [int] = arrays.reverse(nums)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringsModuleCall(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp s string = "hello"
	temp upper string = strings.upper(s)
	temp lower string = strings.lower(s)
	temp trimmed string = strings.trim("  hello  ")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestTimeModuleCall(t *testing.T) {
	input := `
import @time
using time

do main() {
	temp year int = time.year()
	temp month int = time.month()
	temp day int = time.day()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapsModuleCall(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
	temp keys [string] = maps.keys(m)
	temp values [int] = maps.values(m)
	temp has bool = maps.has(m, "a")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Prefix Expression Tests
// ============================================================================

func TestPrefixExpressionNot(t *testing.T) {
	input := `
do main() {
	temp flag bool = true
	temp negated bool = !flag
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestPrefixExpressionNegation(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	temp neg int = -x
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Enum Declaration Tests
// ============================================================================

func TestEnumDeclaration(t *testing.T) {
	input := `
const Status enum {
	Pending
	Active
	Done
}

do main() {
	temp s Status = Status.Active
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEnumWithStringValues(t *testing.T) {
	input := `
const Status enum {
	TODO = "todo"
	IN_PROGRESS = "in_progress"
	DONE = "done"
}

do main() {
	temp s Status = Status.TODO
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMixedTypeEnumError(t *testing.T) {
	input := `
const MixedEnum enum {
	A = 1
	B = "hello"
	C = 3
}

do main() {
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3025) // enum members must all have the same type
}

// ============================================================================
// Global Variable Declaration Tests
// ============================================================================

func TestGlobalVariableDeclaration(t *testing.T) {
	input := `
const MAX_SIZE int = 100
temp counter int = 0

do main() {
	temp x int = MAX_SIZE
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Error Location Tests (Fixes #345)
// ============================================================================

func TestTypeErrorLocationNilInIfCondition(t *testing.T) {
	// Test that type errors report correct source location
	// Fixes #345
	input := `do main() {
	if nil {
		temp x int = 1
	}
}`
	tc := typecheck(t, input)

	if !tc.Errors().HasErrors() {
		t.Fatal("expected type error for nil in if condition")
	}

	// Check that error is NOT at line 1
	errs := tc.Errors()
	for _, err := range errs.Errors {
		if err.ErrorCode == errors.E3001 {
			if err.Line == 1 {
				t.Errorf("error should not be at line 1, got line %d", err.Line)
			}
			if err.Line != 2 {
				t.Errorf("error should be at line 2, got line %d", err.Line)
			}
			return
		}
	}
	t.Error("expected error E3001 not found")
}

func TestTypeErrorLocationMapValue(t *testing.T) {
	// Test that map value errors report correct location
	input := `do main() {
	if {"a": 1} {
		temp x int = 1
	}
}`
	tc := typecheck(t, input)

	if !tc.Errors().HasErrors() {
		t.Fatal("expected type error for map in if condition")
	}

	errs := tc.Errors()
	for _, err := range errs.Errors {
		if err.ErrorCode == errors.E3001 {
			if err.Line == 1 {
				t.Errorf("error should not be at line 1, got line %d", err.Line)
			}
			return
		}
	}
}
