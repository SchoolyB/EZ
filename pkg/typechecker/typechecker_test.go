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
	// Auto-register std module for test convenience (tests don't need explicit imports)
	tc.fileUsingModules["std"] = true
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

func TestStructFieldAssignmentTypeMismatch(t *testing.T) {
	// Regression test for issue #622: struct field assignment type mismatch
	// was not being caught after initialization
	input := `
const Person struct {
	name string
	age int
}

do main() {
	temp p Person = Person{name: "", age: 0}
	p.name = 123
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
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
// All Paths Return Tests (E3035)
// ============================================================================

func TestNotAllPathsReturn_IfWithoutOtherwise(t *testing.T) {
	input := `
do maybe_return(x int) -> int {
	if x > 0 {
		return x
	}
}

do main() {}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3035)
}

func TestNotAllPathsReturn_IfOrWithoutOtherwise(t *testing.T) {
	input := `
do classify(x int) -> string {
	if x > 0 {
		return "positive"
	} or x < 0 {
		return "negative"
	}
}

do main() {}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3035)
}

func TestNotAllPathsReturn_ReturnOnlyInLoop(t *testing.T) {
	input := `
do find_first(arr [int]) -> int {
	for_each i in arr {
		return i
	}
}

do main() {}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3035)
}

func TestAllPathsReturn_IfOtherwise(t *testing.T) {
	input := `
do get_sign(x int) -> string {
	if x > 0 {
		return "positive"
	} otherwise {
		return "non-positive"
	}
}

do main() {}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestAllPathsReturn_IfOrOtherwise(t *testing.T) {
	input := `
do classify(x int) -> string {
	if x > 0 {
		return "positive"
	} or x < 0 {
		return "negative"
	} otherwise {
		return "zero"
	}
}

do main() {}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestAllPathsReturn_DirectReturn(t *testing.T) {
	input := `
do double(x int) -> int {
	return x * 2
}

do main() {}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestAllPathsReturn_ReturnAfterIf(t *testing.T) {
	input := `
do abs_value(x int) -> int {
	if x < 0 {
		return -x
	}
	return x
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
	assertHasError(t, tc, errors.E3027)
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
	assertHasError(t, tc, errors.E3027)
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

func TestForEachStatementMapError(t *testing.T) {
	input := `
do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
	for_each k in m {
	}
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3017) // for_each on map should be an error (#595)
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
	temp first_item int = arrays.first(nums)
	temp last_item int = arrays.last(nums)
	temp rev [int] = arrays.reverse(nums)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArraysSumPreservesIntType(t *testing.T) {
	input := `
import @arrays

do main() {
	temp nums [int] = {1, 2, 3, 4, 5}
	temp sum int = arrays.sum(nums)
	temp min_val int = arrays.min(nums)
	temp max_val int = arrays.max(nums)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathAbsPreservesIntType(t *testing.T) {
	input := `
import @math

do main() {
	temp abs_val int = math.abs(-5)
	temp min_val int = math.min(3, 7)
	temp max_val int = math.max(3, 7)
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
	temp upper_str string = strings.upper(s)
	temp lower_str string = strings.lower(s)
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
	temp cur_year int = time.year()
	temp cur_month int = time.month()
	temp cur_day int = time.day()
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
	temp map_keys [string] = maps.keys(m)
	temp map_values [int] = maps.values(m)
	temp key_exists bool = maps.has(m, "a")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestVariableShadowsUsedModuleFunction(t *testing.T) {
	// Test that declaring a variable with the same name as a function from a 'used' module
	// triggers an error - issue #616
	input := `
import & use @maps

do main() {
	temp m map[string:int] = {"a": 1}
	temp values [int] = {1, 2, 3}
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E4015)
}

func TestVariableShadowsUsedModuleFunctionWithSeparateUsing(t *testing.T) {
	// Test with separate 'using' statement instead of 'import & use'
	input := `
import @arrays
using arrays

do main() {
	temp nums [int] = {1, 2, 3}
	temp reverse [int] = {3, 2, 1}
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E4015)
}

func TestNoErrorWhenNotUsingModule(t *testing.T) {
	// Test that variable names matching module functions are OK when not using the module
	input := `
import @maps

do main() {
	temp m map[string:int] = {"a": 1}
	temp values [int] = {1, 2, 3}
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
	assertHasError(t, tc, errors.E3028) // enum members must all have the same type
}

func TestNilAssignmentToPrimitiveError(t *testing.T) {
	input := `
do main() {
	temp x int = nil
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001) // cannot assign nil to int
}

func TestNilAssignmentToStructAllowed(t *testing.T) {
	input := `
const Point struct {
	x int
	y int
}

do main() {
	temp p Point = nil
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
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

// ============================================================================
// Large Integer Type Tests (i128/u128)
// ============================================================================

func TestLargeIntegerTypes(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"i128 with large positive value",
			`do main() {
				temp x i128 = 10000000000000000000
			}`,
		},
		{
			"i128 with value larger than int64 max",
			`do main() {
				temp x i128 = 9223372036854775808
			}`,
		},
		{
			"u128 with large value",
			`do main() {
				temp x u128 = 18446744073709551615
			}`,
		},
		{
			"u128 with value larger than uint64 max",
			`do main() {
				temp x u128 = 18446744073709551616
			}`,
		},
		{
			"i128 with i128 max value",
			`do main() {
				temp x i128 = 170141183460469231731687303715884105727
			}`,
		},
		{
			"u128 with u128 max value",
			`do main() {
				temp x u128 = 340282366920938463463374607431768211455
			}`,
		},
		{
			"i128 arithmetic",
			`do main() {
				temp a i128 = 9223372036854775807
				temp b i128 = 1
				temp c i128 = a + b
			}`,
		},
		{
			"u128 arithmetic",
			`do main() {
				temp a u128 = 18446744073709551615
				temp b u128 = 1
				temp c u128 = a + b
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := typecheck(t, tt.input)
			assertNoErrors(t, tc)
		})
	}
}

func TestLargeIntegerNegativeValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"i128 with large negative value",
			`do main() {
				temp x i128 = -10000000000000000000
			}`,
		},
		{
			"i128 with value smaller than int64 min",
			`do main() {
				temp x i128 = -9223372036854775809
			}`,
		},
		{
			"i128 with i128 min value",
			`do main() {
				temp x i128 = -170141183460469231731687303715884105728
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := typecheck(t, tt.input)
			assertNoErrors(t, tc)
		})
	}
}

// =============================================================================
// DEFAULT PARAMETER TESTS
// =============================================================================

func TestDefaultParameterBasic(t *testing.T) {
	input := `
	do greet(name string = "World") {
		println(name)
	}
	do main() {
		greet()
		greet("Alice")
	}`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestDefaultParameterMixedRequiredAndOptional(t *testing.T) {
	input := `
	do create(name string, health int = 100, mana int = 50) {
		println(name)
	}
	do main() {
		create("Hero")
		create("Boss", 200)
		create("Wizard", 80, 150)
	}`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestDefaultParameterAllOptional(t *testing.T) {
	input := `
	do config(debug bool = false, level int = 1) {
		println(debug)
	}
	do main() {
		config()
		config(true)
		config(true, 5)
	}`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestDefaultParameterExpressionDefault(t *testing.T) {
	input := `
	do calc(mult float = 3.14 * 2.0) -> float {
		return mult
	}
	do main() {
		temp a = calc()
		temp b = calc(10.0)
	}`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestDefaultParameterGroupedParams(t *testing.T) {
	// x, y int = 0 means x is required, y has default
	input := `
	do point(x, y int = 0) {
		println(x + y)
	}
	do main() {
		point(5)
		point(3, 4)
	}`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestDefaultParameterTooFewArgs(t *testing.T) {
	input := `
	do create(name string, health int = 100) {
		println(name)
	}
	do main() {
		create()
	}`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E5008)
}

func TestDefaultParameterTooManyArgs(t *testing.T) {
	input := `
	do greet(name string = "World") {
		println(name)
	}
	do main() {
		greet("Alice", "extra")
	}`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E5008)
}

func TestDefaultParameterWithReturnType(t *testing.T) {
	input := `
	do greet(name string = "World") -> string {
		return "Hello, ${name}!"
	}
	do main() {
		temp msg1 = greet()
		temp msg2 = greet("Alice")
	}`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Type as Function Argument Tests
// ============================================================================

func TestStructTypeAsFunctionArgument(t *testing.T) {
	// Struct types should be allowed as function arguments
	// This enables features like json.decode(text, Type)
	input := `
	import @std, @json
	using std
	const Person struct {
		name string
		age int
	}
	do main() {
		temp p Person, err error = json.decode("{}", Person)
	}`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStructTypeAsArgumentDoesNotErrorE3030(t *testing.T) {
	// Previously, struct types would cause E3030 error when used as values
	// Now they should be allowed when passed as function arguments (like json.decode)
	input := `
	import @json
	const Task struct {
		title string
		completed bool
	}
	do main() {
		temp t Task, err error = json.decode("{}", Task)
	}`
	tc := typecheck(t, input)
	// Should not have E3030 error (struct type cannot be used as value)
	for _, err := range tc.Errors().Errors {
		if err.ErrorCode == errors.E3030 {
			t.Errorf("got unexpected E3030 error: %s", err.Message)
		}
	}
}

func TestFunctionAsArgumentStillErrors(t *testing.T) {
	// Functions should error when used as values (not called)
	// This test verifies E3031 fires even in assignment context
	input := `
	do helper() {
		// Some helper function
	}
	do main() {
		temp x string = helper
	}`
	tc := typecheck(t, input)
	// Should have E3031 error (function cannot be used as value)
	assertHasError(t, tc, errors.E3031)
}

// ============================================================================
// Test E3034: 'any' type not allowed for user code
// ============================================================================

func TestAnyTypeNotAllowedInVariableDeclaration(t *testing.T) {
	input := `
	do main() {
		temp x any = "hello"
	}`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3034)
}

func TestAnyTypeNotAllowedInFunctionReturnType(t *testing.T) {
	input := `
	do getData() -> any {
		return "hello"
	}
	do main() {
	}`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3034)
}

func TestAnyTypeNotAllowedInFunctionParameter(t *testing.T) {
	input := `
	do process(x any) {
	}
	do main() {
	}`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3034)
}

func TestAnyTypeNotAllowedInArray(t *testing.T) {
	input := `
	do main() {
		temp x [any] = {}
	}`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3034)
}

func TestAnyTypeNotAllowedInMap(t *testing.T) {
	input := `
	do main() {
		temp x map[string:any] = {}
	}`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3034)
}

// ============================================================================
// #strict When Statement Tests (#628)
// ============================================================================

func TestStrictWhenRejectsRangeExpression(t *testing.T) {
	input := `
const Color enum { RED, GREEN, BLUE }
do main() {
	temp c Color = Color.RED
	#strict
	when c {
		is range(0, 2) {
		}
	}
}`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E2054)
}

func TestStrictWhenRejectsIntegerLiteral(t *testing.T) {
	input := `
const Color enum { RED, GREEN, BLUE }
do main() {
	temp c Color = Color.RED
	#strict
	when c {
		is 0 {
		}
	}
}`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E2054)
}

func TestStrictWhenAcceptsEnumMembers(t *testing.T) {
	input := `
const Color enum { RED, GREEN, BLUE }
do main() {
	temp c Color = Color.RED
	#strict
	when c {
		is Color.RED {
		}
		is Color.GREEN {
		}
		is Color.BLUE {
		}
	}
}`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStrictWhenMissingEnumCases(t *testing.T) {
	input := `
const Color enum { RED, GREEN, BLUE }
do main() {
	temp c Color = Color.RED
	#strict
	when c {
		is Color.RED {
		}
	}
}`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E2046)
}

func TestStrictWhenPartialEnumCases(t *testing.T) {
	input := `
const Color enum { RED, GREEN, BLUE }
do main() {
	temp c Color = Color.RED
	#strict
	when c {
		is Color.RED {
		}
		is Color.GREEN {
		}
	}
}`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E2046)
}

// ============================================================================
// Module Registration Tests
// ============================================================================

func TestRegisterModuleFunction(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")

	sig := &FunctionSignature{
		Name:        "helper",
		Parameters:  []*Parameter{{Name: "x", Type: "int"}},
		ReturnTypes: []string{"int"},
	}

	tc.RegisterModuleFunction("mymodule", "helper", sig)

	// Verify function was registered
	retrieved, ok := tc.GetModuleFunction("mymodule", "helper")
	if !ok {
		t.Error("should find registered module function")
	}
	if retrieved.Name != "helper" {
		t.Errorf("expected function name 'helper', got '%s'", retrieved.Name)
	}

	// Test non-existent module function
	_, ok = tc.GetModuleFunction("nonexistent", "helper")
	if ok {
		t.Error("should not find function in non-existent module")
	}

	// Test non-existent function in existing module
	_, ok = tc.GetModuleFunction("mymodule", "nonexistent")
	if ok {
		t.Error("should not find non-existent function in module")
	}
}

func TestRegisterModuleType(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")

	customType := &Type{
		Name: "ModuleStruct",
		Kind: StructType,
		Fields: map[string]*Type{
			"value": {Name: "int", Kind: PrimitiveType},
		},
	}

	tc.RegisterModuleType("mymodule", "ModuleStruct", customType)

	// Verify type was registered by checking types map
	types := tc.GetTypes()
	// Module types are stored in moduleTypes, not types
	// But GetTypes returns the types map which is used for registration
	if types == nil {
		t.Error("GetTypes should not return nil")
	}
}

func TestGetFunctions(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")

	sig := &FunctionSignature{
		Name:        "testFunc",
		Parameters:  []*Parameter{},
		ReturnTypes: []string{"void"},
	}

	tc.RegisterFunction("testFunc", sig)

	functions := tc.GetFunctions()
	if functions == nil {
		t.Error("GetFunctions should not return nil")
	}
	if functions["testFunc"] == nil {
		t.Error("should find registered function via GetFunctions")
	}
}

func TestGetTypes(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")

	customType := &Type{
		Name: "CustomType",
		Kind: StructType,
	}

	tc.RegisterType("CustomType", customType)

	types := tc.GetTypes()
	if types == nil {
		t.Error("GetTypes should not return nil")
	}
	if types["CustomType"] == nil {
		t.Error("should find registered type via GetTypes")
	}
}

// ============================================================================
// Postfix Expression Tests
// ============================================================================

func TestPostfixIncrementValid(t *testing.T) {
	input := `
do main() {
	temp x int = 0
	x++
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestPostfixDecrementValid(t *testing.T) {
	input := `
do main() {
	temp x int = 10
	x--
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestPostfixOnNonIntegerError(t *testing.T) {
	input := `
do main() {
	temp s string = "hello"
	s++
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestPostfixOnConstError(t *testing.T) {
	input := `
do main() {
	const x int = 5
	x++
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E5016)
}

// ============================================================================
// Builtin Call Type Inference Tests
// ============================================================================

func TestLenBuiltinReturnsInt(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {1, 2, 3}
	temp length int = len(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestTypeofBuiltinReturnsString(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	temp t string = typeof(x)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIntConversionBuiltin(t *testing.T) {
	input := `
do main() {
	temp f float = 3.14
	temp i int = int(f)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFloatConversionBuiltin(t *testing.T) {
	input := `
do main() {
	temp i int = 42
	temp f float = float(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringConversionBuiltin(t *testing.T) {
	input := `
do main() {
	temp i int = 123
	temp s string = string(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBoolConversionBuiltin(t *testing.T) {
	input := `
do main() {
	temp i int = 1
	temp b bool = bool(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCharConversionBuiltin(t *testing.T) {
	input := `
do main() {
	temp i int = 65
	temp c char = char(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestByteConversionBuiltin(t *testing.T) {
	input := `
do main() {
	temp i int = 255
	temp b byte = byte(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Standard Call Type Inference Tests
// ============================================================================

func TestInputReturnsString(t *testing.T) {
	input := `
import @std
using std

do main() {
	temp name string = std.input("Enter name: ")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestPrintlnIsVoid(t *testing.T) {
	input := `
do main() {
	println("hello")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Printf Validation Tests
// ============================================================================

func TestPrintfWithFormatString(t *testing.T) {
	input := `
do main() {
	printf("Value: %d", 42)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestPrintfNoArgsError(t *testing.T) {
	input := `
import @std
using std

do main() {
	std.printf()
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E5008)
}

func TestPrintfNonStringFormatError(t *testing.T) {
	input := `
import @std
using std

do main() {
	std.printf(123, "value")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

// ============================================================================
// Comparable Enum Type Tests
// ============================================================================

func TestEnumWithIntValuesIsComparable(t *testing.T) {
	input := `
const Priority enum {
	LOW = 1
	MEDIUM = 2
	HIGH = 3
}

do main() {
	temp p1 Priority = Priority.LOW
	temp p2 Priority = Priority.HIGH
	temp result bool = p1 < p2
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEnumWithStringValuesIsComparable(t *testing.T) {
	input := `
const Status enum {
	OPEN = "open"
	CLOSED = "closed"
}

do main() {
	temp s1 Status = Status.OPEN
	temp s2 Status = Status.CLOSED
	temp result bool = s1 == s2
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Map Key Type Validation Tests
// ============================================================================

func TestMapWithStringKeyValid(t *testing.T) {
	input := `
do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapWithIntKeyValid(t *testing.T) {
	input := `
do main() {
	temp m map[int:string] = {1: "one", 2: "two"}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapWithBoolKeyValid(t *testing.T) {
	input := `
do main() {
	temp m map[bool:string] = {true: "yes", false: "no"}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapWithCharKeyValid(t *testing.T) {
	input := `
do main() {
	temp m map[char:int] = {'a': 1, 'b': 2}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Type Promotion Tests
// ============================================================================

func TestByteWithIntPromotion(t *testing.T) {
	input := `
do main() {
	temp b byte = 255
	temp i int = 100
	temp result int = i + int(b)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Additional Edge Case Tests
// ============================================================================

func TestLenOnString(t *testing.T) {
	input := `
do main() {
	temp s string = "hello"
	temp length int = len(s)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestLenOnMap(t *testing.T) {
	input := `
do main() {
	temp m map[string:int] = {"a": 1}
	temp count int = len(m)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMultipleModuleFunctions(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")

	// Register multiple functions for same module
	tc.RegisterModuleFunction("mymod", "func1", &FunctionSignature{Name: "func1", ReturnTypes: []string{"int"}})
	tc.RegisterModuleFunction("mymod", "func2", &FunctionSignature{Name: "func2", ReturnTypes: []string{"string"}})

	// Both should be retrievable
	sig1, ok := tc.GetModuleFunction("mymod", "func1")
	if !ok || sig1.Name != "func1" {
		t.Error("should find func1")
	}

	sig2, ok := tc.GetModuleFunction("mymod", "func2")
	if !ok || sig2.Name != "func2" {
		t.Error("should find func2")
	}
}

func TestPostfixOnUnsignedIntegers(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"u8 increment",
			`do main() { temp x u8 = 0
			x++ }`,
		},
		{
			"u16 decrement",
			`do main() { temp x u16 = 10
			x-- }`,
		},
		{
			"u32 increment",
			`do main() { temp x u32 = 100
			x++ }`,
		},
		{
			"u64 decrement",
			`do main() { temp x u64 = 1000
			x-- }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := typecheck(t, tt.input)
			assertNoErrors(t, tc)
		})
	}
}

func TestPostfixOnSignedIntegers(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"i8 increment",
			`do main() { temp x i8 = 0
			x++ }`,
		},
		{
			"i16 decrement",
			`do main() { temp x i16 = 10
			x-- }`,
		},
		{
			"i32 increment",
			`do main() { temp x i32 = 100
			x++ }`,
		},
		{
			"i64 decrement",
			`do main() { temp x i64 = 1000
			x-- }`,
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
// IO Module Type Checking Tests
// ============================================================================

func TestIoModuleReadFile(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp content string, err error = io.read_file("test.txt")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIoModuleWriteFile(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp success bool, err error = io.write_file("test.txt", "content")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIoModuleAppendFile(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp success bool, err error = io.append_file("test.txt", "more content")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIoModuleFileExists(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp file_exists_result bool = io.file_exists("test.txt")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIoModuleReadLines(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp lines [string], err error = io.read_lines("test.txt")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIoModuleReadBytes(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp data [byte], err error = io.read_bytes("test.bin")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIoModuleWriteBytes(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp data [byte] = {0x48, 0x65, 0x6c, 0x6c, 0x6f}
	temp success bool, err error = io.write_bytes("test.bin", data)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIoModuleDeleteFile(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp success bool, err error = io.delete_file("test.txt")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIoModuleCopyFile(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp success bool, err error = io.copy_file("src.txt", "dst.txt")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIoModuleMoveFile(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp success bool, err error = io.move_file("old.txt", "new.txt")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIoModuleWrongArgCount(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp content string, err error = io.read_file()
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E5008)
}

func TestIoModuleWrongArgType(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp content string, err error = io.read_file(123)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

// ============================================================================
// OS Module Type Checking Tests
// ============================================================================

func TestOsModuleGetEnv(t *testing.T) {
	input := `
import @os
using os

do main() {
	temp value string = os.get_env("HOME")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestOsModuleSetEnv(t *testing.T) {
	input := `
import @os
using os

do main() {
	temp success bool, err error = os.set_env("MY_VAR", "value")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestOsModuleUnsetEnv(t *testing.T) {
	input := `
import @os
using os

do main() {
	temp success bool, err error = os.unset_env("MY_VAR")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestOsModuleGetCwd(t *testing.T) {
	input := `
import @os
using os

do main() {
	temp current_dir string = os.get_cwd()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestOsModuleArgs(t *testing.T) {
	input := `
import @os
using os

do main() {
	temp cli_args [string] = os.args()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestOsModuleExit(t *testing.T) {
	input := `
import @os
using os

do main() {
	os.exit(0)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestOsModuleWrongArgCount(t *testing.T) {
	input := `
import @os
using os

do main() {
	temp value string = os.get_env()
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E5008)
}

func TestOsModuleWrongArgType(t *testing.T) {
	input := `
import @os
using os

do main() {
	temp success bool, err error = os.set_env(123, "value")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

// ============================================================================
// Random Module Type Checking Tests
// ============================================================================

func TestRandomModuleInt(t *testing.T) {
	input := `
import @random
using random

do main() {
	temp r int = random.int(100)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestRandomModuleIntRange(t *testing.T) {
	input := `
import @random
using random

do main() {
	temp r int = random.int(10, 100)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestRandomModuleFloat(t *testing.T) {
	input := `
import @random
using random

do main() {
	temp r float = random.float()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestRandomModuleFloatRange(t *testing.T) {
	input := `
import @random
using random

do main() {
	temp r float = random.float(0.0, 1.0)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestRandomModuleBool(t *testing.T) {
	input := `
import @random
using random

do main() {
	temp r bool = random.bool()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestRandomModuleChoice(t *testing.T) {
	input := `
import @random
using random

do main() {
	temp arr [int] = {1, 2, 3, 4, 5}
	temp r int = random.choice(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestRandomModuleShuffle(t *testing.T) {
	input := `
import @random
using random

do main() {
	temp arr [int] = {1, 2, 3, 4, 5}
	temp shuffled [int] = random.shuffle(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestRandomModuleWrongArgCount(t *testing.T) {
	input := `
import @random
using random

do main() {
	temp r int = random.int()
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E5008)
}

// ============================================================================
// Bytes Module Type Checking Tests
// ============================================================================

func TestBytesModuleFromString(t *testing.T) {
	input := `
import @bytes
using bytes

do main() {
	temp b [byte] = bytes.from_string("hello")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBytesModuleFromArray(t *testing.T) {
	input := `
import @bytes
using bytes

do main() {
	temp arr [int] = {72, 101, 108, 108, 111}
	temp b [byte] = bytes.from_array(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBytesModuleToString(t *testing.T) {
	input := `
import @bytes
using bytes

do main() {
	temp b [byte] = {72, 101, 108, 108, 111}
	temp s string = bytes.to_string(b)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBytesModuleFromHex(t *testing.T) {
	input := `
import @bytes
using bytes

do main() {
	temp b [byte], err error = bytes.from_hex("48656c6c6f")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBytesModuleToHex(t *testing.T) {
	input := `
import @bytes
using bytes

do main() {
	temp b [byte] = {72, 101, 108, 108, 111}
	temp hex string = bytes.to_hex(b)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBytesModuleSlice(t *testing.T) {
	input := `
import @bytes
using bytes

do main() {
	temp b [byte] = {1, 2, 3, 4, 5}
	temp sliced [byte] = bytes.slice(b, 1, 3)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBytesModuleConcat(t *testing.T) {
	input := `
import @bytes
using bytes

do main() {
	temp a [byte] = {1, 2, 3}
	temp b [byte] = {4, 5, 6}
	temp c [byte] = bytes.concat(a, b)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBytesModuleWrongArgType(t *testing.T) {
	input := `
import @bytes
using bytes

do main() {
	temp b [byte] = bytes.from_string(123)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

// ============================================================================
// Binary Module Type Checking Tests
// ============================================================================

func TestBinaryModuleEncodeI8(t *testing.T) {
	input := `
import @binary
using binary

do main() {
	temp b [byte], err error = binary.encode_i8(127)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBinaryModuleDecodeI8(t *testing.T) {
	input := `
import @binary
using binary

do main() {
	temp data [byte] = {127}
	temp value int, err error = binary.decode_i8(data)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBinaryModuleEncodeU16LE(t *testing.T) {
	input := `
import @binary
using binary

do main() {
	temp b [byte], err error = binary.encode_u16_le(65535)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBinaryModuleDecodeU16LE(t *testing.T) {
	input := `
import @binary
using binary

do main() {
	temp data [byte] = {0xFF, 0xFF}
	temp value int, err error = binary.decode_u16_le(data)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBinaryModuleEncodeI32BE(t *testing.T) {
	input := `
import @binary
using binary

do main() {
	temp b [byte], err error = binary.encode_i32_be(2147483647)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBinaryModuleDecodeI32BE(t *testing.T) {
	input := `
import @binary
using binary

do main() {
	temp data [byte] = {0x7F, 0xFF, 0xFF, 0xFF}
	temp value int, err error = binary.decode_i32_be(data)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBinaryModuleEncodeF32LE(t *testing.T) {
	input := `
import @binary
using binary

do main() {
	temp b [byte], err error = binary.encode_f32_le(3.14)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBinaryModuleDecodeF32LE(t *testing.T) {
	input := `
import @binary
using binary

do main() {
	temp data [byte] = {0xC3, 0xF5, 0x48, 0x40}
	temp value float, err error = binary.decode_f32_le(data)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBinaryModuleEncodeF64BE(t *testing.T) {
	input := `
import @binary
using binary

do main() {
	temp b [byte], err error = binary.encode_f64_be(3.14159265359)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBinaryModuleDecodeF64BE(t *testing.T) {
	input := `
import @binary
using binary

do main() {
	temp data [byte] = {0x40, 0x09, 0x21, 0xFB, 0x54, 0x44, 0x2D, 0x18}
	temp value float, err error = binary.decode_f64_be(data)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBinaryModuleWrongArgType(t *testing.T) {
	input := `
import @binary
using binary

do main() {
	temp b [byte], err error = binary.encode_i8("not a number")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestBinaryModuleWrongArgCount(t *testing.T) {
	input := `
import @binary
using binary

do main() {
	temp b [byte], err error = binary.encode_i8()
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E5008)
}

// ============================================================================
// Array Element Type Compatibility Tests
// ============================================================================

func TestArrayElementTypeMismatch(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {1, "two", 3}
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArrayElementTypeCompatibleIntegers(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {1, 2, 3, 4, 5}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArrayElementTypeCompatibleFloats(t *testing.T) {
	input := `
do main() {
	temp arr [float] = {1.1, 2.2, 3.3}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArrayElementTypeCompatibleStrings(t *testing.T) {
	input := `
do main() {
	temp arr [string] = {"a", "b", "c"}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedArrayElementTypeMismatch(t *testing.T) {
	input := `
do main() {
	temp arr [[int]] = {{1, 2}, {"a", "b"}}
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysModuleWithTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp nums [string] = {"a", "b"}
	temp sum int = arrays.sum(nums)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

// ============================================================================
// Map Key/Value Type Compatibility Tests
// ============================================================================

func TestMapLiteralAcceptsMatchingTypes(t *testing.T) {
	input := `
do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapKeyValueTypesCompatible(t *testing.T) {
	input := `
do main() {
	temp m map[string:int] = {"a": 1, "b": 2, "c": 3}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapsModuleKeysType(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[int:string] = {1: "one", 2: "two"}
	temp map_keys [int] = maps.keys(m)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapsModuleValuesType(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[int:string] = {1: "one", 2: "two"}
	temp map_values [string] = maps.values(m)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Additional TypeChecker Coverage Tests
// ============================================================================

func TestSetCurrentModule(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")
	// Just verify SetCurrentModule doesn't panic
	tc.SetCurrentModule("mymodule")
}

func TestRegisterModuleVariable(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")
	tc.RegisterModuleVariable("mymodule", "myvar", "int")

	// Verify variable was registered
	typeName, ok := tc.GetModuleVariable("mymodule", "myvar")
	if !ok {
		t.Error("should find registered module variable")
	}
	if typeName != "int" {
		t.Errorf("expected type 'int', got '%s'", typeName)
	}
}

func TestRegisterVariable(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")
	tc.RegisterVariable("globalVar", "string")

	// Verify variable was registered
	vars := tc.GetVariables()
	if vars == nil {
		t.Error("GetVariables should not return nil")
	}
	if vars["globalVar"] != "string" {
		t.Errorf("expected type 'string', got '%s'", vars["globalVar"])
	}
}

func TestGetVariables(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")
	tc.RegisterVariable("var1", "int")
	tc.RegisterVariable("var2", "string")

	vars := tc.GetVariables()
	if len(vars) != 2 {
		t.Errorf("expected 2 variables, got %d", len(vars))
	}
}

func TestJsonModuleEncode(t *testing.T) {
	input := `
import @json
using json

const Person struct {
	name string
	age int
}

do main() {
	temp p Person = Person{name: "Alice", age: 30}
	temp jsonStr string, err error = json.encode(p)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestJsonModuleDecode(t *testing.T) {
	input := `
import @json
using json

const Person struct {
	name string
	age int
}

do main() {
	temp p Person, err error = json.decode("{}", Person)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestJsonModulePretty(t *testing.T) {
	input := `
import @json
using json

const Person struct {
	name string
	age int
}

do main() {
	temp p Person = Person{name: "Alice", age: 30}
	temp jsonStr string, err error = json.pretty(p, "  ")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathModuleFloorCall(t *testing.T) {
	// Tests module function call validation
	input := `
import @math
using math

do main() {
	temp x float = math.floor(3.7)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Index Expression Tests
// ============================================================================

func TestArrayIndexExpression(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {1, 2, 3}
	temp x int = arr[0]
	temp y int = arr[1]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapIndexExpression(t *testing.T) {
	input := `
do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
	temp x int = m["a"]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringIndexExpression(t *testing.T) {
	input := `
do main() {
	temp s string = "hello"
	temp c char = s[0]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedIndexExpression(t *testing.T) {
	input := `
do main() {
	temp arr [[int]] = {{1, 2}, {3, 4}}
	temp x int = arr[0][1]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Builtin Type Conversion Tests
// ============================================================================

func TestCastToI8(t *testing.T) {
	input := `
do main() {
	temp x i8 = cast(127, i8)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToI16(t *testing.T) {
	input := `
do main() {
	temp x i16 = cast(1000, i16)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToI32(t *testing.T) {
	input := `
do main() {
	temp x i32 = cast(100000, i32)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToU8(t *testing.T) {
	input := `
do main() {
	temp x u8 = cast(255, u8)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToU16(t *testing.T) {
	input := `
do main() {
	temp x u16 = cast(65535, u16)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToF32(t *testing.T) {
	input := `
do main() {
	temp x f32 = cast(3.14, f32)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToF64(t *testing.T) {
	input := `
do main() {
	temp x f64 = cast(3.14159, f64)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Integer Bounds Tests
// ============================================================================

func TestI8Bounds(t *testing.T) {
	input := `
do main() {
	temp min i8 = -128
	temp max i8 = 127
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestU8Bounds(t *testing.T) {
	input := `
do main() {
	temp min u8 = 0
	temp max u8 = 255
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestI16Bounds(t *testing.T) {
	input := `
do main() {
	temp min i16 = -32768
	temp max i16 = 32767
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestU16Bounds(t *testing.T) {
	input := `
do main() {
	temp min u16 = 0
	temp max u16 = 65535
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Strings Module Call Tests
// ============================================================================

func TestStringsContains(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp result bool = strings.contains("hello world", "world")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringsSplit(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp parts [string] = strings.split("a,b,c", ",")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringsJoin(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp parts [string] = {"a", "b", "c"}
	temp joined string = strings.join(parts, ",")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringsReplace(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp result string = strings.replace("hello world", "world", "there")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringsSubstring(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp sub string = strings.substring("hello", 0, 3)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Time Module Call Tests
// ============================================================================

func TestTimeNow(t *testing.T) {
	input := `
import @time
using time

do main() {
	temp timestamp int = time.now()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestTimeFormat(t *testing.T) {
	input := `
import @time
using time

do main() {
	temp formatted string = time.format("YYYY-MM-DD", 1234567890)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestTimeHour(t *testing.T) {
	input := `
import @time
using time

do main() {
	temp h int = time.hour()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestTimeMinute(t *testing.T) {
	input := `
import @time
using time

do main() {
	temp m int = time.minute()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestTimeSecond(t *testing.T) {
	input := `
import @time
using time

do main() {
	temp s int = time.second()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// File Scope Statement Tests
// ============================================================================

func TestGlobalConstDeclaration(t *testing.T) {
	input := `
const MAX_VALUE int = 100
const MIN_VALUE int = 0

do main() {
	temp x int = MAX_VALUE
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestGlobalTempDeclaration(t *testing.T) {
	input := `
temp counter int = 0

do main() {
	counter = counter + 1
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Multi-Return Declaration Tests
// ============================================================================

func TestMultiReturnWithError(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp content string, err error = io.read_file("test.txt")
	if err != nil {
		println("Error reading file")
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMultiReturnBothValues(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp data string, err error = io.read_file("test.txt")
	println(data)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Enum Value Type Tests
// ============================================================================

func TestEnumWithIntValues(t *testing.T) {
	input := `
const Priority enum {
	LOW = 0
	MEDIUM = 1
	HIGH = 2
}

do main() {
	temp p Priority = Priority.HIGH
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEnumWithFloatValues(t *testing.T) {
	input := `
const Scale enum {
	SMALL = 0.5
	MEDIUM = 1.0
	LARGE = 2.0
}

do main() {
	temp s Scale = Scale.MEDIUM
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEnumDefaultValues(t *testing.T) {
	input := `
const Direction enum {
	NORTH
	SOUTH
	EAST
	WEST
}

do main() {
	temp d Direction = Direction.NORTH
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Range Expression Tests
// ============================================================================

func TestRangeInForLoop(t *testing.T) {
	input := `
do main() {
	for i in range(0, 10) {
		println(i)
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestRangeWithStep(t *testing.T) {
	input := `
do main() {
	for i in range(0, 10, 2) {
		println(i)
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Member Access Tests
// ============================================================================

func TestStructMemberAccess(t *testing.T) {
	input := `
const Point struct {
	x int
	y int
}

do main() {
	temp p Point = Point{x: 10, y: 20}
	temp sum int = p.x + p.y
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedStructMemberAccess(t *testing.T) {
	input := `
const Inner struct {
	value int
}

const Outer struct {
	inner Inner
}

do main() {
	temp o Outer = Outer{inner: Inner{value: 42}}
	temp v int = o.inner.value
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEnumMemberAccess(t *testing.T) {
	input := `
const Color enum {
	RED = 1
	GREEN = 2
	BLUE = 3
}

do main() {
	temp c Color = Color.RED
	temp isRed bool = c == Color.RED
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Additional Coverage Tests
// ============================================================================

func TestWhenStatementWithEnumValue(t *testing.T) {
	input := `
const Status enum {
	PENDING
	ACTIVE
	DONE
}

do main() {
	temp s Status = Status.PENDING
	when s {
		is Status.PENDING {
			println("pending")
		}
		is Status.ACTIVE {
			println("active")
		}
		is Status.DONE {
			println("done")
		}
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestWhenStatementWithInteger(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	when x {
		is 1 {
			println("one")
		}
		is 5 {
			println("five")
		}
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNewBuiltinFunction(t *testing.T) {
	input := `
const Person struct {
	name string
	age int
}

do main() {
	temp p Person = new(Person)
	p.name = "Alice"
	p.age = 30
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArrayLength(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {1, 2, 3, 4, 5}
	temp length int = len(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringLength(t *testing.T) {
	input := `
do main() {
	temp s string = "hello world"
	temp length int = len(s)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCompoundAssignment(t *testing.T) {
	input := `
do main() {
	temp x int = 10
	x += 5
	x -= 3
	x *= 2
	x /= 4
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBreakAndContinue(t *testing.T) {
	input := `
do main() {
	for i in range(0, 10) {
		if i == 5 {
			break
		}
		if i == 3 {
			continue
		}
		println(i)
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedLoops(t *testing.T) {
	input := `
do main() {
	for i in range(0, 5) {
		for j in range(0, 5) {
			temp product int = i * j
		}
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStructWithOptionalField(t *testing.T) {
	input := `
const Config struct {
	name string
	timeout int
}

do main() {
	temp c Config = Config{name: "test", timeout: 30}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArrayLengthCheck(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3}
	temp isEmpty bool = arrays.is_empty(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArrayFirst(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3}
	temp firstElement int = arrays.first(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapMerge(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m1 map[string:int] = {"a": 1}
	temp m2 map[string:int] = {"b": 2}
	temp merged map[string:int] = maps.merge(m1, m2)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapDelete(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
	temp deleted bool = maps.delete(m, "a")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringInterpolation(t *testing.T) {
	input := `
do main() {
	temp name string = "world"
	temp greeting string = "Hello, ${name}!"
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIfCondition(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	if x > 0 {
		println("positive")
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestLogicalOperators(t *testing.T) {
	input := `
do main() {
	temp a bool = true
	temp b bool = false
	temp andResult bool = a && b
	temp orResult bool = a || b
	temp notResult bool = !a
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestComparisonOperators(t *testing.T) {
	input := `
do main() {
	temp a int = 5
	temp b int = 3
	temp gt bool = a > b
	temp lt bool = a < b
	temp eq bool = a == b
	temp neq bool = a != b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Additional Tests for 60% Coverage
// ============================================================================

func TestNestedIfStatements(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	if x > 10 {
		println("big")
	}
	if x > 0 {
		println("positive")
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestForEachWithArray(t *testing.T) {
	input := `
do main() {
	temp items [string] = {"a", "b", "c"}
	for_each item in items {
		println(item)
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestForLoopIteration(t *testing.T) {
	input := `
do main() {
	temp sum int = 0
	for i in range(0, 10) {
		sum = sum + i
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFunctionCall(t *testing.T) {
	input := `
do add(a int, b int) -> int {
	return a + b
}

do main() {
	temp sum int = add(5, 3)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFunctionWithMultipleParams(t *testing.T) {
	input := `
do greet(name string, times int) {
	for i in range(0, times) {
		println(name)
	}
}

do main() {
	greet("Hello", 3)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArraysReverse(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3}
	temp reversed [int] = arrays.reverse(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArraysContains(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3}
	temp hasTwo bool = arrays.contains(arr, 2)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathPow(t *testing.T) {
	input := `
import @math
using math

do main() {
	temp result float = math.pow(2.0, 10.0)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathSqrt(t *testing.T) {
	input := `
import @math
using math

do main() {
	temp result float = math.sqrt(16.0)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathCeil(t *testing.T) {
	input := `
import @math
using math

do main() {
	temp result float = math.ceil(3.2)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathRound(t *testing.T) {
	input := `
import @math
using math

do main() {
	temp result float = math.round(3.7)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringsUpper(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp result string = strings.upper("hello")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringsLower(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp result string = strings.lower("HELLO")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringsTrim(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp result string = strings.trim("  hello  ")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringsStartsWith(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp result bool = strings.starts_with("hello world", "hello")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringsEndsWith(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp result bool = strings.ends_with("hello world", "world")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringsIndexOf(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp idx int = strings.index_of("hello world", "world")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathMin(t *testing.T) {
	input := `
import @math
using math

do main() {
	temp result int = math.min(5, 3)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathMax(t *testing.T) {
	input := `
import @math
using math

do main() {
	temp result int = math.max(5, 3)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArraysSort(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {3, 1, 4, 1, 5}
	temp sorted [int] = arrays.sort(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArraysSum(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3, 4, 5}
	temp total int = arrays.sum(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapsContains(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
	temp hasKey bool = maps.has(m, "a")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapsSize(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
	temp mapSize int = maps.len(m)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Cast Expression Tests
// ============================================================================

func TestCastIntToFloat(t *testing.T) {
	input := `
do main() {
	temp x int = 42
	temp f float = cast(x, float)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastFloatToInt(t *testing.T) {
	input := `
do main() {
	temp f float = 3.14
	temp x int = cast(f, int)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastIntToString(t *testing.T) {
	input := `
do main() {
	temp x int = 42
	temp s string = cast(x, string)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastStringToInt(t *testing.T) {
	input := `
do main() {
	temp s string = "42"
	temp x int = cast(s, int)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastIntToByte(t *testing.T) {
	input := `
do main() {
	temp x int = 65
	temp b byte = cast(x, byte)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastByteToInt(t *testing.T) {
	input := `
do main() {
	temp b byte = 65
	temp x int = cast(b, int)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastIntToChar(t *testing.T) {
	input := `
do main() {
	temp x int = 65
	temp c char = cast(x, char)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastCharToInt(t *testing.T) {
	input := `
do main() {
	temp c char = 'A'
	temp x int = cast(c, int)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// More Type Conversion Tests
// ============================================================================

func TestIntToFloatConversion(t *testing.T) {
	input := `
do main() {
	temp x int = 42
	temp f float = float(x)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFloatToIntConversion(t *testing.T) {
	input := `
do main() {
	temp f float = 3.14
	temp x int = int(f)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIntToStringConversion(t *testing.T) {
	input := `
do main() {
	temp x int = 42
	temp s string = string(x)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBoolToStringConversion(t *testing.T) {
	input := `
do main() {
	temp b bool = true
	temp s string = string(b)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Return Type Tests
// ============================================================================

func TestFunctionReturnsStruct(t *testing.T) {
	input := `
const Point struct {
	x int
	y int
}

do makePoint(x int, y int) -> Point {
	return Point{x: x, y: y}
}

do main() {
	temp p Point = makePoint(10, 20)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFunctionReturnsArray(t *testing.T) {
	input := `
do makeNumbers() -> [int] {
	return {1, 2, 3, 4, 5}
}

do main() {
	temp nums [int] = makeNumbers()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFunctionReturnsBool(t *testing.T) {
	input := `
do isPositive(x int) -> bool {
	return x > 0
}

do main() {
	temp result bool = isPositive(5)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFunctionReturnsString(t *testing.T) {
	input := `
do greet(name string) -> string {
	return "Hello, " + name
}

do main() {
	temp msg string = greet("World")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Additional Tests for 60% Threshold
// ============================================================================

func TestRecursiveFunctionFactorial(t *testing.T) {
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

func TestFunctionWithNoParamsGetZero(t *testing.T) {
	input := `
do getZero() -> int {
	return 0
}

do main() {
	temp x int = getZero()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStructFieldAccessPerson(t *testing.T) {
	input := `
const Person struct {
	name string
	age int
}

do main() {
	temp p Person = Person{name: "Alice", age: 30}
	temp n string = p.name
	temp a int = p.age
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStructFieldAssignment(t *testing.T) {
	input := `
const Person struct {
	name string
	age int
}

do main() {
	temp p Person = Person{name: "Alice", age: 30}
	p.name = "Bob"
	p.age = 25
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNilComparison(t *testing.T) {
	input := `
import @io
using io

do main() {
	temp content string, err error = io.read_file("test.txt")
	if err == nil {
		println("success")
	}
	if err != nil {
		println("error")
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEnumComparison(t *testing.T) {
	input := `
const Status enum {
	PENDING
	ACTIVE
	DONE
}

do main() {
	temp s Status = Status.PENDING
	if s == Status.PENDING {
		println("pending")
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArrayOfStructs(t *testing.T) {
	input := `
const Point struct {
	x int
	y int
}

do main() {
	temp points [Point] = {Point{x: 0, y: 0}, Point{x: 1, y: 1}}
	temp first Point = points[0]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapOfStructs(t *testing.T) {
	input := `
const Person struct {
	name string
	age int
}

do main() {
	temp people map[string:Person] = {
		"alice": Person{name: "Alice", age: 30}
	}
	temp p Person = people["alice"]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToI64(t *testing.T) {
	input := `
do main() {
	temp x int = 1000000
	temp y i64 = cast(x, i64)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToU32(t *testing.T) {
	input := `
do main() {
	temp x int = 1000
	temp y u32 = cast(x, u32)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToBool(t *testing.T) {
	input := `
do main() {
	temp x int = 1
	temp b bool = cast(x, bool)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapWithIntKeys(t *testing.T) {
	input := `
do main() {
	temp m map[int:string] = {1: "one", 2: "two", 3: "three"}
	temp s string = m[1]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedForLoops(t *testing.T) {
	input := `
do main() {
	for i in range(0, 3) {
		for j in range(0, 3) {
			temp product int = i * j
		}
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMultipleFunctionParams(t *testing.T) {
	input := `
do calculate(a int, b int, c int, d int) -> int {
	return a + b + c + d
}

do main() {
	temp result int = calculate(1, 2, 3, 4)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStructWithManyFields(t *testing.T) {
	input := `
const Record struct {
	id int
	name string
	active bool
	score float
}

do main() {
	temp r Record = Record{id: 1, name: "test", active: true, score: 95.5}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestGlobalConstant(t *testing.T) {
	input := `
const PI float = 3.14159
const MAX_SIZE int = 100

do main() {
	temp area float = PI * 10.0 * 10.0
	temp size int = MAX_SIZE
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMultipleStructTypes(t *testing.T) {
	input := `
const Point struct {
	x int
	y int
}

const Size struct {
	width int
	height int
}

const Rectangle struct {
	origin Point
	size Size
}

do main() {
	temp r Rectangle = Rectangle{origin: Point{x: 0, y: 0}, size: Size{width: 100, height: 50}}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEnumInSwitch(t *testing.T) {
	input := `
const Color enum {
	RED = 1
	GREEN = 2
	BLUE = 3
}

do main() {
	temp c Color = Color.RED
	when c {
		is Color.RED {
			println("red")
		}
		is Color.GREEN {
			println("green")
		}
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestComplexExpression(t *testing.T) {
	input := `
do main() {
	temp a int = 5
	temp b int = 10
	temp c int = 3
	temp result int = (a + b) * c - (b / a)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringConcatenation(t *testing.T) {
	input := `
do main() {
	temp first string = "Hello"
	temp second string = " "
	temp third string = "World"
	temp result string = first + second + third
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArrayWithMixedOps(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {5, 3, 8, 1, 9}
	temp length int = len(arr)
	temp isEmpty bool = arrays.is_empty(arr)
	temp firstEl int = arrays.first(arr)
	temp lastEl int = arrays.last(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapWithMixedOps(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[string:int] = {"a": 1, "b": 2, "c": 3}
	temp hasA bool = maps.has(m, "a")
	temp mapLen int = maps.len(m)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFunctionParameterTypes(t *testing.T) {
	input := `
do processData(text string, count int, flag bool, value float) {
	println(text)
}

do main() {
	processData("test", 10, true, 3.14)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedStructAccess(t *testing.T) {
	input := `
const Inner struct {
	value int
}

const Middle struct {
	inner Inner
}

const Outer struct {
	middle Middle
}

do main() {
	temp o Outer = Outer{middle: Middle{inner: Inner{value: 42}}}
	temp v int = o.middle.inner.value
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEmptyArray(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {}
	temp arrLen int = len(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEmptyMap(t *testing.T) {
	input := `
do main() {
	temp m map[string:int] = {}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBooleanExpression(t *testing.T) {
	input := `
do main() {
	temp a bool = true
	temp b bool = false
	temp result bool = (a && !b) || (!a && b)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestComparisonChain(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	temp isInRange bool = x > 0 && x < 10
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathOperations(t *testing.T) {
	input := `
do main() {
	temp a int = 10
	temp b int = 3
	temp sum int = a + b
	temp diff int = a - b
	temp prod int = a * b
	temp quot int = a / b
	temp rem int = a % b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFloatOperations(t *testing.T) {
	input := `
do main() {
	temp a float = 10.5
	temp b float = 3.2
	temp sum float = a + b
	temp diff float = a - b
	temp prod float = a * b
	temp quot float = a / b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestPrintlnVariousTypes(t *testing.T) {
	input := `
do main() {
	println("string")
	println(42)
	println(3.14)
	println(true)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestUnaryOperators(t *testing.T) {
	input := `
do main() {
	temp a int = 5
	temp neg int = -a
	temp b bool = true
	temp notB bool = !b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestLargeArrayLiteral(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestLargeMapLiteral(t *testing.T) {
	input := `
do main() {
	temp m map[string:int] = {
		"one": 1,
		"two": 2,
		"three": 3,
		"four": 4,
		"five": 5
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestVariableReassignment(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	x = 10
	x = x + 1
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestConstVariableUsage(t *testing.T) {
	input := `
const MAX int = 100

do main() {
	temp x int = MAX
	temp y int = MAX + 1
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// Additional tests to push coverage to 60%

func TestHashableMapKeyTypesExtended(t *testing.T) {
	input := `
do main() {
	temp m1 map[int:string] = {1: "one", 2: "two"}
	temp m2 map[bool:string] = {true: "yes", false: "no"}
	temp m3 map[char:int] = {'a': 1, 'b': 2}
	temp m4 map[byte:int] = {}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestComparisonOperatorsExtended(t *testing.T) {
	input := `
do main() {
	temp a int = 5
	temp b int = 10
	temp lt bool = a < b
	temp gt bool = a > b
	temp lte bool = a <= b
	temp gte bool = a >= b
	temp eq bool = a == b
	temp neq bool = a != b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestLogicalOperatorsExtended(t *testing.T) {
	input := `
do main() {
	temp a bool = true
	temp b bool = false
	temp andResult bool = a && b
	temp orResult bool = a || b
	temp complex bool = (a && b) || (!a && !b)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestRangeInForLoopExtended(t *testing.T) {
	input := `
do main() {
	for i in range(0, 10) {
		println(i)
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestRangeWithStepInForLoopExtended(t *testing.T) {
	input := `
do main() {
	for i in range(0, 10, 2) {
		println(i)
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedStructAccessExtended(t *testing.T) {
	input := `
const Inner struct {
	value int
}

const Outer struct {
	inner Inner
}

do main() {
	temp o Outer = new(Outer)
	o.inner.value = 42
	temp v int = o.inner.value
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFunctionWithSingleReturnValue(t *testing.T) {
	input := `
do square(n int) -> int {
	return n * n
}

do main() {
	temp result int = square(5)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringIndexing(t *testing.T) {
	input := `
do main() {
	temp s string = "hello"
	temp ch char = s[0]
	temp last char = s[4]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIndexExpressionOnMapExtended(t *testing.T) {
	input := `
do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
	temp val int = m["a"]
	temp val2 int = m["b"]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestGlobalVariableWithInitialization(t *testing.T) {
	input := `
temp GLOBAL_VAL int = 100

do main() {
	temp x int = GLOBAL_VAL
	GLOBAL_VAL = 200
	temp y int = GLOBAL_VAL
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestWhileLoopExtended(t *testing.T) {
	input := `
do main() {
	temp i int = 0
	as_long_as i < 10 {
		i = i + 1
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBreakStatementInLoop(t *testing.T) {
	input := `
do main() {
	for i in range(0, 10) {
		break
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestContinueStatementInLoop(t *testing.T) {
	input := `
do main() {
	for i in range(0, 10) {
		continue
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestConditionalExtended(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	if x > 3 {
		println("greater")
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEnumComparisonExtended(t *testing.T) {
	input := `
const Status enum {
	PENDING = 1
	ACTIVE = 2
	DONE = 3
}

do main() {
	temp s Status = Status.PENDING
	temp isActive bool = s == Status.ACTIVE
	temp notDone bool = s != Status.DONE
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEmptyArrayLiteral(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {}
	temp strArr [string] = {}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEmptyMapLiteral(t *testing.T) {
	input := `
do main() {
	temp m map[string:int] = {}
	temp m2 map[int:bool] = {}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestTypeExistsForBuiltins(t *testing.T) {
	tc := NewTypeChecker("", "test.ez")
	tc.SetSkipMainCheck(true)

	builtinTypes := []string{
		"int", "i8", "i16", "i32", "i64",
		"u8", "u16", "u32", "u64",
		"float", "f32", "f64",
		"string", "bool", "char", "byte",
		"nil", "error",
	}

	for _, typeName := range builtinTypes {
		if !tc.TypeExists(typeName) {
			t.Errorf("expected type %s to exist", typeName)
		}
	}
}

func TestFunctionCallWithNoArgs(t *testing.T) {
	input := `
do greet() {
	println("hello")
}

do main() {
	greet()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringConcatenationExtended(t *testing.T) {
	input := `
do main() {
	temp a string = "hello"
	temp b string = "world"
	temp c string = a + " " + b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFloatArithmetic(t *testing.T) {
	input := `
do main() {
	temp a float = 3.14
	temp b float = 2.71
	temp sum float = a + b
	temp diff float = a - b
	temp prod float = a * b
	temp quot float = a / b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIntegerArithmeticExtended(t *testing.T) {
	input := `
do main() {
	temp a int = 10
	temp b int = 3
	temp sum int = a + b
	temp diff int = a - b
	temp prod int = a * b
	temp quot int = a / b
	temp mod int = a % b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedIfStatement(t *testing.T) {
	input := `
do main() {
	temp x int = 5
	temp y int = 10
	if x > 0 {
		if y > 5 {
			println("both positive and y > 5")
		}
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestForEachWithArrayExtended(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {1, 2, 3, 4, 5}
	for_each item in arr {
		println(item)
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestForEachWithStringExtended(t *testing.T) {
	input := `
do main() {
	temp s string = "hello"
	for_each ch in s {
		println(ch)
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedForLoop(t *testing.T) {
	input := `
do main() {
	for i in range(0, 3) {
		for j in range(0, 3) {
			println(i + j)
		}
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFunctionWithParameters(t *testing.T) {
	input := `
do add(a int, b int) -> int {
	return a + b
}

do main() {
	temp result int = add(5, 3)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStructWithMultipleFieldsExtended(t *testing.T) {
	input := `
const Rectangle struct {
	w int
	h int
	n string
}

do main() {
	temp r Rectangle = new(Rectangle)
	r.w = 10
	r.h = 20
	r.n = "test"
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMethodCallsExtended(t *testing.T) {
	input := `
import @strings
using strings

do main() {
	temp s string = "hello world"
	temp upperStr string = strings.upper(s)
	temp lowerStr string = strings.lower(s)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathModuleCalls(t *testing.T) {
	input := `
import @math
using math

do main() {
	temp absVal int = math.abs(-5)
	temp maxVal int = math.max(10, 20)
	temp minVal int = math.min(10, 20)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArraysModuleCalls(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {3, 1, 4, 1, 5}
	temp isEmpty bool = arrays.is_empty(arr)
	temp length int = len(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Array Element Type Compatibility Tests (checkArrayElementTypeCompatibility)
// ============================================================================

func TestArraysAppendTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3}
	arrays.append(arr, "wrong")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysUnshiftTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [string] = {"a", "b"}
	arrays.unshift(arr, 123)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysContainsTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3}
	temp found bool = arrays.contains(arr, "string")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysIndexOfTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [float] = {1.0, 2.0, 3.0}
	temp idx int = arrays.index_of(arr, "not a float")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysInsertTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3}
	arrays.insert(arr, 0, "wrong type")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysSetTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [bool] = {true, false}
	arrays.set(arr, 0, "not bool")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysConcatTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr1 [int] = {1, 2}
	temp arr2 [string] = {"a", "b"}
	temp result [int] = arrays.concat(arr1, arr2)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysFillTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3}
	arrays.fill(arr, "not an int")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysRemoveTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3}
	arrays.remove(arr, "string")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysRemoveAllTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3, 2}
	arrays.remove_all(arr, true)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysLastIndexOfTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3, 2}
	temp idx int = arrays.last_index_of(arr, 3.14)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysCountTypeMismatch(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [string] = {"a", "b", "a"}
	temp count int = arrays.count(arr, 42)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArraysAppendCorrectType(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [int] = {1, 2, 3}
	arrays.append(arr, 4)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArraysInsertCorrectType(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr [string] = {"a", "b"}
	arrays.insert(arr, 1, "c")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArraysConcatCorrectType(t *testing.T) {
	input := `
import @arrays
using arrays

do main() {
	temp arr1 [int] = {1, 2}
	temp arr2 [int] = {3, 4}
	temp result [int] = arrays.concat(arr1, arr2)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Map Key/Value Type Compatibility Tests (checkMapKeyValueTypeCompatibility)
// ============================================================================

func TestMapsHasKeyTypeMismatch(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
	temp found bool = maps.has_key(m, 123)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestMapsDeleteKeyTypeMismatch(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[int:string] = {1: "one", 2: "two"}
	maps.delete(m, "not an int")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestMapsSetKeyTypeMismatch(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[string:int] = {"a": 1}
	maps.set(m, 123, 5)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestMapsSetValueTypeMismatch(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[string:int] = {"a": 1}
	maps.set(m, "b", "wrong")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestMapsGetKeyTypeMismatch(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[string:int] = {"a": 1}
	maps.get(m, 999)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestMapsGetOrSetKeyTypeMismatch(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[int:string] = {1: "one"}
	maps.get_or_set(m, "wrong", "default")
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestMapsGetOrSetValueTypeMismatch(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[int:string] = {1: "one"}
	maps.get_or_set(m, 2, 42)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestMapsHasCorrectKeyType(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[string:int] = {"a": 1, "b": 2}
	temp found bool = maps.has(m, "a")
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapsSetCorrectTypes(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[string:int] = {"a": 1}
	maps.set(m, "b", 2)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapsRemoveKeyTypeMismatch(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	temp m map[string:int] = {"a": 1}
	maps.remove(m, 123)
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

// ============================================================================
// Type Lookup and Module Tests (lookupType)
// ============================================================================

func TestLookupTypeFromLocalScope(t *testing.T) {
	input := `
const Point struct {
	x int
	y int
}

do main() {
	temp p Point = Point{x: 1, y: 2}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)

	// Verify the type was registered
	pointType, found := tc.GetType("Point")
	if !found || pointType == nil {
		t.Error("expected Point type to be registered")
	}
}

func TestTypeLookupForNestedStructs(t *testing.T) {
	input := `
const Inner struct {
	value int
}

const Outer struct {
	inner Inner
}

do main() {
	temp o Outer = Outer{inner: Inner{value: 42}}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Builtin Type Conversion Coverage Tests (checkBuiltinTypeConversion)
// ============================================================================

func TestStringToFloatConversionCoverage(t *testing.T) {
	input := `
do main() {
	temp s string = "3.14"
	temp f float = float(s)
}
`
	_ = typecheck(t, input)
	// This is expected to give a warning about runtime conversion
	// Just check that it compiles
}

func TestCharToIntConversionCoverage(t *testing.T) {
	input := `
do main() {
	temp c char = 'A'
	temp i int = int(c)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIntToCharConversionCoverage(t *testing.T) {
	input := `
do main() {
	temp i int = 65
	temp c char = char(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestByteConversionsCoverage(t *testing.T) {
	input := `
do main() {
	temp b byte = byte(65)
	temp i int = int(b)
	temp c char = char(b)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Cast Expression Tests (checkCastExpression)
// ============================================================================

func TestCastIntToI32(t *testing.T) {
	input := `
do main() {
	temp i int = 42
	temp i32val i32 = i32(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastI64ToInt(t *testing.T) {
	input := `
do main() {
	temp i64val i64 = 42
	temp i int = int(i64val)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastU8ToU16(t *testing.T) {
	input := `
do main() {
	temp u8val u8 = 255
	temp u16val u16 = u16(u8val)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastF32ToFloat(t *testing.T) {
	input := `
do main() {
	temp f32val f32 = 3.14
	temp fval float = float(f32val)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Multi-Return Declaration Tests (checkMultiReturnDeclaration)
// ============================================================================

func TestMultiReturnDeclarationWithBlankIdentifier(t *testing.T) {
	input := `
do getTwo() -> (int, string) {
	return 42, "hello"
}

do main() {
	temp x, _ = getTwo()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMultiReturnDeclarationAllBlank(t *testing.T) {
	input := `
do getThree() -> (int, string, bool) {
	return 1, "test", true
}

do main() {
	temp _, _, _ = getThree()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMultiReturnWithTwoVariables(t *testing.T) {
	input := `
do getTwo() -> (int, string) {
	return 42, "hello"
}

do main() {
	temp x, y = getTwo()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Integer Bounds Coverage Tests (getIntegerBounds)
// ============================================================================

func TestI32BoundsCoverage(t *testing.T) {
	input := `
do main() {
	temp min i32 = -2147483648
	temp max i32 = 2147483647
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestU32BoundsCoverage(t *testing.T) {
	input := `
do main() {
	temp min u32 = 0
	temp max u32 = 4294967295
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestI64BoundsCoverage(t *testing.T) {
	input := `
do main() {
	temp val i64 = 9223372036854775807
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestU64BoundsCoverage(t *testing.T) {
	input := `
do main() {
	temp val u64 = 18446744073709551615
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Direct Stdlib Call Tests (checkDirectStdlibCall)
// ============================================================================

func TestDirectStdlibCallWithoutImport(t *testing.T) {
	input := `
do main() {
	arrays.reverse({1, 2, 3})
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E4007)
}

func TestDirectStdlibCallWithImport(t *testing.T) {
	input := `
import @arrays

do main() {
	arrays.reverse({1, 2, 3})
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Infer Builtin Call Type Tests (inferBuiltinCallType)
// ============================================================================

func TestInferLenReturnType(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {1, 2, 3}
	temp length int = len(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestInferTypeofReturnType(t *testing.T) {
	input := `
do main() {
	temp x int = 42
	temp typeName string = typeof(x)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestInferCopyReturnType(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {1, 2, 3}
	temp arrCopy [int] = copy(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestInferAppendReturnType(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {1, 2, 3}
	temp newArr [int] = append(arr, 4)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestInferRangeReturnType(t *testing.T) {
	input := `
do main() {
	temp nums [int] = range(1, 10)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Suppression Tests (isSuppressed)
// ============================================================================

func TestSuppressionComment(t *testing.T) {
	input := `
do main() {
	// @suppress E3002
	temp x = 42
}
`
	tc := typecheck(t, input)
	// Should not have E3002 error because it's suppressed
	for _, err := range tc.Errors().Errors {
		if err.ErrorCode == errors.E3002 {
			t.Error("E3002 should be suppressed")
		}
	}
}

// ============================================================================
// Has Return in If Statement Tests (hasReturnInIfStatement)
// ============================================================================

func TestNestedIfWithReturns(t *testing.T) {
	input := `
do nested_return(x int, y int) -> int {
	if x > 0 {
		if y > 0 {
			return x + y
		} otherwise {
			return x - y
		}
	} otherwise {
		return -x
	}
}

do main() {}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedIfMissingReturn(t *testing.T) {
	input := `
do nested_missing(x int, y int) -> int {
	if x > 0 {
		if y > 0 {
			return x + y
		}
	} otherwise {
		return -x
	}
}

do main() {}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3035)
}

// Tests for loop variable shadowing (#114)

func TestLoopVariableShadowsOuterLoopVariable_For(t *testing.T) {
	input := `
do main() {
	for i in range(0, 2) {
		for i in range(0, 3) {
			println(i)
		}
	}
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E4016)
}

func TestLoopVariableShadowsOuterLoopVariable_ForEach(t *testing.T) {
	input := `
do main() {
	temp outer [int] = {1, 2}
	temp inner [int] = {3, 4}
	for_each item in outer {
		for_each item in inner {
			println(item)
		}
	}
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E4016)
}

func TestLoopVariableShadowsOuterLoopVariable_Mixed(t *testing.T) {
	input := `
do main() {
	temp arr [int] = {10, 20}
	for i in range(0, 2) {
		for_each i in arr {
			println(i)
		}
	}
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E4016)
}

func TestLoopVariableNoShadowing_DifferentNames(t *testing.T) {
	input := `
do main() {
	for i in range(0, 2) {
		for j in range(0, 3) {
			println(i + j)
		}
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestLoopVariableCanShadowRegularVariable(t *testing.T) {
	input := `
do main() {
	temp i int = 100
	for i in range(0, 3) {
		println(i)
	}
	println(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestRegularVariableShadowingStillAllowed(t *testing.T) {
	input := `
do main() {
	temp x int = 10
	if true {
		temp x int = 20
		println(x)
	}
	println(x)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}
