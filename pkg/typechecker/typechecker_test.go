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

func assertNoWarning(t *testing.T, tc *TypeChecker, unexpectedCode errors.ErrorCode) {
	t.Helper()
	for _, warn := range tc.Errors().Warnings {
		if warn.ErrorCode == unexpectedCode {
			t.Errorf("unexpected warning %s: %s at line %d", warn.ErrorCode.Code, warn.Message, warn.Line)
		}
	}
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
	mut x int = 5
	mut y int = 10
	mut sum int = x + y
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMissingMainFunction(t *testing.T) {
	input := `
do helper() {
	mut x int = 5
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
			`do main() { mut x int = 5 }`,
		},
		{
			"float",
			`do main() { mut f float = 3.14 }`,
		},
		{
			"string",
			`do main() { mut s string = "hello" }`,
		},
		{
			"bool",
			`do main() { mut b bool = true }`,
		},
		{
			"char",
			`do main() { mut c char = 'a' }`,
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
	mut x UndefinedType = 5
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3008)
}

func TestTypeMismatchInDeclaration(t *testing.T) {
	input := `
do main() {
	mut x int = "hello"
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
	mut p Person = Person{name: "Alice", age: 30}
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
	mut p Person = Person{name: "", age: 0}
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
	mut p Person = Person{name: "", age: 0}
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
	mut r TestResult = TestResult{passed: 1, bugs: {"bug1"}}
	mut bug_list [string] = r.bugs
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
	mut c Config = Config{name: "test", settings: {"a": 1}}
	mut s map[string:int] = c.settings
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
	mut m Matrix = Matrix{rows: {{1, 2}, {3, 4}}}
	mut r [[int]] = m.rows
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
	mut result int = add(1, 2)
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
	mut x int = 5
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
			`do main() { mut r int = 1 + 2 }`,
		},
		{
			"float addition",
			`do main() { mut r float = 1.5 + 2.5 }`,
		},
		{
			"string concatenation",
			`do main() { mut r string = "a" + "b" }`,
		},
		{
			"int subtraction",
			`do main() { mut r int = 5 - 3 }`,
		},
		{
			"int multiplication",
			`do main() { mut r int = 2 * 3 }`,
		},
		{
			"int division",
			`do main() { mut r int = 6 / 2 }`,
		},
		{
			"int modulo",
			`do main() { mut r int = 7 % 3 }`,
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
			`do main() { mut r bool = 1 == 1 }`,
		},
		{
			"int inequality",
			`do main() { mut r bool = 1 != 2 }`,
		},
		{
			"int less than",
			`do main() { mut r bool = 1 < 2 }`,
		},
		{
			"int greater than",
			`do main() { mut r bool = 2 > 1 }`,
		},
		{
			"int less or equal",
			`do main() { mut r bool = 1 <= 2 }`,
		},
		{
			"int greater or equal",
			`do main() { mut r bool = 2 >= 1 }`,
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
			`do main() { mut r bool = true && false }`,
		},
		{
			"logical or",
			`do main() { mut r bool = true || false }`,
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
	mut arr [int] = {1, 2, 3}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArrayTypeMismatch(t *testing.T) {
	input := `
do main() {
	mut arr [int] = "not an array"
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3018)
}

func TestFixedSizeArrayWarning(t *testing.T) {
	input := `
do main() {
	mut arr [int, 5] = {1, 2}
}
`
	tc := typecheck(t, input)
	assertHasWarning(t, tc, errors.W3003)
}

func TestFixedSizeArrayOverflow(t *testing.T) {
	input := `
do main() {
	const arr [int, 3] = {0, 1, 2, 3}
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3041)
}

func TestFixedSizeArrayOverflowGlobal(t *testing.T) {
	input := `
const arr [int, 3] = {0, 1, 2, 3}
do main() {}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3041)
}

// ============================================================================
// Control Flow Tests
// ============================================================================

func TestIfStatement(t *testing.T) {
	input := `
do main() {
	mut x int = 5
	if x > 0 {
		mut y int = 10
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
		mut x int = i
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestWhileLoop(t *testing.T) {
	input := `
do main() {
	mut x int = 0
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
	mut x int = 5
	x = 10
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestAssignmentTypeMismatch(t *testing.T) {
	input := `
do main() {
	mut x int = 5
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
	mut i8val i8 = 1
	mut i16val i16 = 2
	mut i32val i32 = 3
	mut i64val i64 = 4
	mut intval int = 5
	mut f32val f32 = 1.0
	mut f64val f64 = 2.0
	mut floatval float = 3.0
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestUnsignedTypes(t *testing.T) {
	input := `
do main() {
	mut u8val u8 = 1
	mut u16val u16 = 2
	mut u32val u32 = 3
	mut u64val u64 = 4
	mut uintval uint = 5
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
	mut x int = 1
	if true {
		mut y int = 2
		if true {
			mut z int = x + y
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
	mut m map[string:int] = {"a": 1, "b": 2}
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
	mut p Person = nil
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
	mut dx int = p2.x - p1.x
	mut dy int = p2.y - p1.y
	return dx + dy
}

do main() {
	mut origin Point = Point{x: 0, y: 0}
	mut target Point = Point{x: 3, y: 4}
	mut dist int = distance(origin, target)
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
	mut result int = quadruple(5)
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
	mut result int = factorial(5)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Mutable Parameter Tests (& prefix)
// ============================================================================

func TestMutableParameterValid(t *testing.T) {
	// Test: mut variable passed to & param - should be valid
	input := `
const Person struct {
	name string
	age int
}

do birthday(&p Person) {
	p.age = p.age + 1
}

do main() {
	mut bob Person = Person{name: "Bob", age: 30}
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
	mut name string = getName(alice)
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
	mut bob Person = Person{name: "Bob", age: 30}
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
	mut bob Person = Person{name: "Bob", age: 30}
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
	mut nums [int] = {1, 2, 3}
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
	mut nums [int] = {1, 2, 3}
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
	mut target [int] = {0, 0, 0}
	mut source [int] = {1, 2, 3}
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
	mut arr [int, 3] = {1, 2, 3}
	mut x int = arr[0]
	mut y int = arr[1] + arr[2]
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
	mut s = "hello"
	mut x = s.name
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
	mut p = Person{name: "Alice", age: 30}
	mut n = p.name
	mut a = p.age
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
	mut nums [int] = {1, 2, 3}
	mut sum int = 0
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
	mut str string = "hello"
	mut count int = 0
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
	mut m map[string:int] = {"a": 1, "b": 2}
	for_each k in m {
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc) // for_each on map is valid — iterates over keys
}

func TestForEachStatementMapKeyValue(t *testing.T) {
	input := `
do main() {
	mut m map[string:int] = {"a": 1, "b": 2}
	for_each k, v in m {
	}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc) // for_each k, v on map is valid — iterates key-value pairs
}

// ============================================================================
// Loop Statement Tests
// ============================================================================

func TestLoopStatement(t *testing.T) {
	input := `
do main() {
	mut count int = 0
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
	mut count int = 0
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
	mut x float = math.sqrt(16.0)
	mut y float = math.abs(-5.0)
	mut z float = math.pow(2.0, 3.0)
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
	mut nums [int] = {1, 2, 3}
	mut first_item int = arrays.first(nums)
	mut last_item int = arrays.last(nums)
	mut rev [int] = arrays.reverse(nums)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArraysSumPreservesIntType(t *testing.T) {
	input := `
import @arrays

do main() {
	mut nums [int] = {1, 2, 3, 4, 5}
	mut sum int = arrays.sum(nums)
	mut min_val int = arrays.min(nums)
	mut max_val int = arrays.max(nums)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathAbsPreservesIntType(t *testing.T) {
	input := `
import @math

do main() {
	mut abs_val int = math.abs(-5)
	mut min_val int = math.min(3, 7)
	mut max_val int = math.max(3, 7)
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
	mut s string = "hello"
	mut upper_str string = strings.upper(s)
	mut lower_str string = strings.lower(s)
	mut trimmed string = strings.trim("  hello  ")
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
	mut cur_year int = time.year()
	mut cur_month int = time.month()
	mut cur_day int = time.day()
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
	mut m map[string:int] = {"a": 1, "b": 2}
	mut map_keys [string] = maps.keys(m)
	mut map_values [int] = maps.values(m)
	mut key_exists bool = maps.contains(m, "a")
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
	mut m map[string:int] = {"a": 1}
	mut values [int] = {1, 2, 3}
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
	mut nums [int] = {1, 2, 3}
	mut reverse [int] = {3, 2, 1}
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
	mut m map[string:int] = {"a": 1}
	mut values [int] = {1, 2, 3}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStdlibConstantsWithUsing(t *testing.T) {
	input := `
import @std
using std

do main() {
	const exit_success_val int = EXIT_SUCCESS
	const exit_failure_val int = EXIT_FAILURE
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestDbConstantsWithUsing(t *testing.T) {
	input := `
import @db
using db

do main() {
	const alpha_val int = ALPHA
	const alpha_desc_val int = ALPHA_DESC
	const value_alpha_val int = VALUE_ALPHA
	const value_alpha_desc_val int = VALUE_ALPHA_DESC
	const key_len_val int = KEY_LEN
	const key_len_desc_val int = KEY_LEN_DESC
	const value_len_val int = VALUE_LEN
	const value_len_desc_val int = VALUE_LEN_DESC
	const numeric_val int = NUMERIC
	const numberic_desc_val int = NUMERIC_DESC
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestHttpConstantsWithUsing(t *testing.T) {
	input := `
import @http
using http

do main() {
	const status_200 int = OK
	const status_201 int = CREATED
	const status_202 int = ACCEPTED
	const status_204 int = NO_CONTENT
	const status_301 int = MOVED_PERMANENTLY
	const status_302 int = FOUND
	const status_304 int = NOT_MODIFIED
	const status_307 int = TEMPORARY_REDIRECT
	const status_308 int = PERMANENT_REDIRECT
	const status_400 int = BAD_REQUEST
	const status_401 int = UNAUTHORIZED
	const status_402 int = PAYMENT_REQUIRED
	const status_403 int = FORBIDDEN
	const status_404 int = NOT_FOUND
	const status_405 int = METHOD_NOT_ALLOWED
	const status_409 int = CONFLICT
	const status_500 int = INTERNAL_SERVER_ERROR
	const status_502 int = BAD_GATEWAY
	const status_503 int = SERVICE_UNAVAILABLE
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIOConstantsWithUsing(t *testing.T) {
	input := `
import @io
using io

do main() {
	const read_only_val int = READ_ONLY
	const write_only_val int = WRITE_ONLY
	const read_write_val int = READ_WRITE
	const append_val int = APPEND
	const create_val int = CREATE
	const truncate_val int = TRUNCATE
	const exclusive_val int = EXCLUSIVE
	const seek_start_val int = SEEK_START
	const seek_current_val int = SEEK_CURRENT
	const seek_end_val int = SEEK_END
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathConstantsWithUsing(t *testing.T) {
	input := `
import @math
using math

do main() {
	const pi_val float = PI
	const e_val float = E
	const phi_val float = PHI
	const sqrt2_val float = SQRT2
	const ln2_val float = LN2
	const ln10_val float = LN10
	const tau_val float = TAU
	const inf_val float = INF
	const neg_inf_val float = NEG_INF
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestOSConstantsWithUsing(t *testing.T) {
	input := `
import @os
using os

do main() {
	const mac_os_val int = MAC_OS
	const linux_val int = LINUX
	const windows_val int = WINDOWS
	const current_os_val int = CURRENT_OS
	const line_sep_val string = line_separator
	const dev_null_val string = dev_null
}
`

	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestTimeConstantsWithUsing(t *testing.T) {
	input := `
import @time
using time

do main() {
	const sunday_val int = SUNDAY
	const monday_val int = MONDAY
	const tuesday_val int = TUESDAY
	const wednesday_val int = WEDNESDAY
	const thursday_val int = THURSDAY
	const friday_val int = FRIDAY
	const saturday_val int = SATURDAY
	const january_val int = JANUARY
	const february_val int = FEBRUARY
	const march_val int = MARCH
	const april_val int = APRIL
	const may_val int = MAY
	const june_val int = JUNE
	const july_val int = JULY
	const august_val int = AUGUST
	const september_val int = SEPTEMBER
	const october_val int = OCTOBER
	const november_val int = NOVEMBER
	const december_val int = DECEMBER
	const second_val int = SECOND
	const minute_val int = MINUTE
	const hour_val int = HOUR
	const day_val int = DAY
	const week_val int = WEEK
}
`

	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestUUIDConstantsWithUsing(t *testing.T) {
	input := `
import @uuid
using uuid

do main() {
	const nil_val string = NIL
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
	mut flag bool = true
	mut negated bool = !flag
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestPrefixExpressionNegation(t *testing.T) {
	input := `
do main() {
	mut x int = 5
	mut neg int = -x
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
	mut s Status = Status.Active
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
	mut s Status = Status.TODO
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
	mut x int = nil
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
	mut p Point = nil
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
mut counter int = 0

do main() {
	mut x int = MAX_SIZE
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
		mut x int = 1
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
		mut x int = 1
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
				mut x i128 = 10000000000000000000
			}`,
		},
		{
			"i128 with value larger than int64 max",
			`do main() {
				mut x i128 = 9223372036854775808
			}`,
		},
		{
			"u128 with large value",
			`do main() {
				mut x u128 = 18446744073709551615
			}`,
		},
		{
			"u128 with value larger than uint64 max",
			`do main() {
				mut x u128 = 18446744073709551616
			}`,
		},
		{
			"i128 with i128 max value",
			`do main() {
				mut x i128 = 170141183460469231731687303715884105727
			}`,
		},
		{
			"u128 with u128 max value",
			`do main() {
				mut x u128 = 340282366920938463463374607431768211455
			}`,
		},
		{
			"i128 arithmetic",
			`do main() {
				mut a i128 = 9223372036854775807
				mut b i128 = 1
				mut c i128 = a + b
			}`,
		},
		{
			"u128 arithmetic",
			`do main() {
				mut a u128 = 18446744073709551615
				mut b u128 = 1
				mut c u128 = a + b
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
				mut x i128 = -10000000000000000000
			}`,
		},
		{
			"i128 with value smaller than int64 min",
			`do main() {
				mut x i128 = -9223372036854775809
			}`,
		},
		{
			"i128 with i128 min value",
			`do main() {
				mut x i128 = -170141183460469231731687303715884105728
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
		mut a = calc()
		mut b = calc(10.0)
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
		mut msg1 = greet()
		mut msg2 = greet("Alice")
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
		mut p Person, err error = json.decode("{}", Person)
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
		mut t Task, err error = json.decode("{}", Task)
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
		mut x string = helper
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
		mut x any = "hello"
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
		mut x [any] = {}
	}`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3034)
}

func TestAnyTypeNotAllowedInMap(t *testing.T) {
	input := `
	do main() {
		mut x map[string:any] = {}
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
	mut c Color = Color.RED
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
	mut c Color = Color.RED
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
	mut c Color = Color.RED
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
	mut c Color = Color.RED
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
	mut c Color = Color.RED
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
	mut x int = 0
	x++
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestPostfixDecrementValid(t *testing.T) {
	input := `
do main() {
	mut x int = 10
	x--
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestPostfixOnNonIntegerError(t *testing.T) {
	input := `
do main() {
	mut s string = "hello"
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
	mut arr [int] = {1, 2, 3}
	mut length int = len(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestTypeofBuiltinReturnsString(t *testing.T) {
	input := `
do main() {
	mut x int = 5
	mut t string = typeof(x)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIntConversionBuiltin(t *testing.T) {
	input := `
do main() {
	mut f float = 3.14
	mut i int = int(f)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFloatConversionBuiltin(t *testing.T) {
	input := `
do main() {
	mut i int = 42
	mut f float = float(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringConversionBuiltin(t *testing.T) {
	input := `
do main() {
	mut i int = 123
	mut s string = string(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBoolConversionBuiltin(t *testing.T) {
	input := `
do main() {
	mut i int = 1
	mut b bool = bool(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCharConversionBuiltin(t *testing.T) {
	input := `
do main() {
	mut i int = 65
	mut c char = char(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestByteConversionBuiltin(t *testing.T) {
	input := `
do main() {
	mut i int = 255
	mut b byte = byte(i)
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
	mut name string = std.input("Enter name: ")
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
// Print Validation Tests
// ============================================================================

func TestPrintWithFormatString(t *testing.T) {
	input := `
do main() {
	print("Value: %d", 42)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestPrintNoArgsError(t *testing.T) {
	input := `
import @std
using std

do main() {
	std.print()
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
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
	mut p1 Priority = Priority.LOW
	mut p2 Priority = Priority.HIGH
	mut result bool = p1 < p2
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
	mut s1 Status = Status.OPEN
	mut s2 Status = Status.CLOSED
	mut result bool = s1 == s2
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
	mut m map[string:int] = {"a": 1, "b": 2}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapWithIntKeyValid(t *testing.T) {
	input := `
do main() {
	mut m map[int:string] = {1: "one", 2: "two"}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapWithBoolKeyValid(t *testing.T) {
	input := `
do main() {
	mut m map[bool:string] = {true: "yes", false: "no"}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapWithCharKeyValid(t *testing.T) {
	input := `
do main() {
	mut m map[char:int] = {'a': 1, 'b': 2}
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
	mut b byte = 255
	mut i int = 100
	mut result int = i + int(b)
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
	mut s string = "hello"
	mut length int = len(s)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestLenOnMap(t *testing.T) {
	input := `
do main() {
	mut m map[string:int] = {"a": 1}
	mut count int = len(m)
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
			`do main() { mut x u8 = 0
			x++ }`,
		},
		{
			"u16 decrement",
			`do main() { mut x u16 = 10
			x-- }`,
		},
		{
			"u32 increment",
			`do main() { mut x u32 = 100
			x++ }`,
		},
		{
			"u64 decrement",
			`do main() { mut x u64 = 1000
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
			`do main() { mut x i8 = 0
			x++ }`,
		},
		{
			"i16 decrement",
			`do main() { mut x i16 = 10
			x-- }`,
		},
		{
			"i32 increment",
			`do main() { mut x i32 = 100
			x++ }`,
		},
		{
			"i64 decrement",
			`do main() { mut x i64 = 1000
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
	mut content string, err error = io.read_file("test.txt")
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
	mut success bool, err error = io.write_file("test.txt", "content")
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
	mut success bool, err error = io.append_file("test.txt", "more content")
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
	mut file_exists_result bool = io.file_exists("test.txt")
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
	mut lines [string], err error = io.read_lines("test.txt")
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
	mut data [byte], err error = io.read_bytes("test.bin")
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
	mut data [byte] = {0x48, 0x65, 0x6c, 0x6c, 0x6f}
	mut success bool, err error = io.write_bytes("test.bin", data)
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
	mut success bool, err error = io.delete_file("test.txt")
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
	mut success bool, err error = io.copy_file("src.txt", "dst.txt")
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
	mut success bool, err error = io.move_file("old.txt", "new.txt")
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
	mut content string, err error = io.read_file()
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
	mut content string, err error = io.read_file(123)
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
	mut value string, err error = os.get_env("HOME")
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
	mut success bool, err error = os.set_env("MY_VAR", "value")
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
	mut success bool, err error = os.unset_env("MY_VAR")
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
	mut current_dir string = os.get_cwd()
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
	mut cli_args [string] = os.args()
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
	mut value string = os.get_env()
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
	mut success bool, err error = os.set_env(123, "value")
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
	mut r int = random.int(100)
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
	mut r int = random.int(10, 100)
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
	mut r float = random.float()
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
	mut r float = random.float(0.0, 1.0)
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
	mut r bool = random.bool()
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
	mut arr [int] = {1, 2, 3, 4, 5}
	mut r int = random.choice(arr)
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
	mut arr [int] = {1, 2, 3, 4, 5}
	mut shuffled [int] = random.shuffle(arr)
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
	mut r int = random.int()
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
	mut b [byte] = bytes.from_string("hello")
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
	mut arr [int] = {72, 101, 108, 108, 111}
	mut b [byte] = bytes.from_array(arr)
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
	mut b [byte] = {72, 101, 108, 108, 111}
	mut s string = bytes.to_string(b)
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
	mut b [byte], err error = bytes.from_hex("48656c6c6f")
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
	mut b [byte] = {72, 101, 108, 108, 111}
	mut hex string = bytes.to_hex(b)
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
	mut b [byte] = {1, 2, 3, 4, 5}
	mut sliced [byte] = bytes.slice(b, 1, 3)
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
	mut a [byte] = {1, 2, 3}
	mut b [byte] = {4, 5, 6}
	mut c [byte] = bytes.concat(a, b)
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
	mut b [byte] = bytes.from_string(123)
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
	mut b [byte], err error = binary.encode_i8(127)
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
	mut data [byte] = {127}
	mut value int, err error = binary.decode_i8(data)
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
	mut b [byte], err error = binary.encode_u16_le(65535)
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
	mut data [byte] = {0xFF, 0xFF}
	mut value int, err error = binary.decode_u16_le(data)
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
	mut b [byte], err error = binary.encode_i32_be(2147483647)
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
	mut data [byte] = {0x7F, 0xFF, 0xFF, 0xFF}
	mut value int, err error = binary.decode_i32_be(data)
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
	mut b [byte], err error = binary.encode_f32_le(3.14)
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
	mut data [byte] = {0xC3, 0xF5, 0x48, 0x40}
	mut value float, err error = binary.decode_f32_le(data)
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
	mut b [byte], err error = binary.encode_f64_be(3.14159265359)
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
	mut data [byte] = {0x40, 0x09, 0x21, 0xFB, 0x54, 0x44, 0x2D, 0x18}
	mut value float, err error = binary.decode_f64_be(data)
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
	mut b [byte], err error = binary.encode_i8("not a number")
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
	mut b [byte], err error = binary.encode_i8()
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
	mut arr [int] = {1, "two", 3}
}
`
	tc := typecheck(t, input)
	assertHasError(t, tc, errors.E3001)
}

func TestArrayElementTypeCompatibleIntegers(t *testing.T) {
	input := `
do main() {
	mut arr [int] = {1, 2, 3, 4, 5}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArrayElementTypeCompatibleFloats(t *testing.T) {
	input := `
do main() {
	mut arr [float] = {1.1, 2.2, 3.3}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestArrayElementTypeCompatibleStrings(t *testing.T) {
	input := `
do main() {
	mut arr [string] = {"a", "b", "c"}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedArrayElementTypeMismatch(t *testing.T) {
	input := `
do main() {
	mut arr [[int]] = {{1, 2}, {"a", "b"}}
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
	mut nums [string] = {"a", "b"}
	mut sum int = arrays.sum(nums)
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
	mut m map[string:int] = {"a": 1, "b": 2}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapKeyValueTypesCompatible(t *testing.T) {
	input := `
do main() {
	mut m map[string:int] = {"a": 1, "b": 2, "c": 3}
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
	mut m map[int:string] = {1: "one", 2: "two"}
	mut map_keys [int] = maps.keys(m)
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
	mut m map[int:string] = {1: "one", 2: "two"}
	mut map_values [string] = maps.values(m)
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
	mut p Person = Person{name: "Alice", age: 30}
	mut jsonStr string, err error = json.encode(p)
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
	mut p Person, err error = json.decode("{}", Person)
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
	mut p Person = Person{name: "Alice", age: 30}
	mut jsonStr string, err error = json.pretty(p, "  ")
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
	mut x float = math.floor(3.7)
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
	mut arr [int] = {1, 2, 3}
	mut x int = arr[0]
	mut y int = arr[1]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapIndexExpression(t *testing.T) {
	input := `
do main() {
	mut m map[string:int] = {"a": 1, "b": 2}
	mut x int = m["a"]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringIndexExpression(t *testing.T) {
	input := `
do main() {
	mut s string = "hello"
	mut c char = s[0]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedIndexExpression(t *testing.T) {
	input := `
do main() {
	mut arr [[int]] = {{1, 2}, {3, 4}}
	mut x int = arr[0][1]
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
	mut x i8 = cast(127, i8)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToI16(t *testing.T) {
	input := `
do main() {
	mut x i16 = cast(1000, i16)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToI32(t *testing.T) {
	input := `
do main() {
	mut x i32 = cast(100000, i32)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToU8(t *testing.T) {
	input := `
do main() {
	mut x u8 = cast(255, u8)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToU16(t *testing.T) {
	input := `
do main() {
	mut x u16 = cast(65535, u16)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToF32(t *testing.T) {
	input := `
do main() {
	mut x f32 = cast(3.14, f32)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToF64(t *testing.T) {
	input := `
do main() {
	mut x f64 = cast(3.14159, f64)
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
	mut min i8 = -128
	mut max i8 = 127
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestU8Bounds(t *testing.T) {
	input := `
do main() {
	mut min u8 = 0
	mut max u8 = 255
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestI16Bounds(t *testing.T) {
	input := `
do main() {
	mut min i16 = -32768
	mut max i16 = 32767
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestU16Bounds(t *testing.T) {
	input := `
do main() {
	mut min u16 = 0
	mut max u16 = 65535
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
	mut result bool = strings.contains("hello world", "world")
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
	mut parts [string] = strings.split("a,b,c", ",")
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
	mut parts [string] = {"a", "b", "c"}
	mut joined string = strings.join(parts, ",")
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
	mut result string = strings.replace("hello world", "world", "there")
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
	mut sub string = strings.substring("hello", 0, 3)
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
	mut ts int = time.now()
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
	mut formatted string = time.format("YYYY-MM-DD", 1234567890)
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
	mut h int = time.hour()
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
	mut m int = time.minute()
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
	mut s int = time.second()
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
	mut x int = MAX_VALUE
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestGlobalTempDeclaration(t *testing.T) {
	input := `
mut counter int = 0

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
	mut content string, err error = io.read_file("test.txt")
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
	mut data string, err error = io.read_file("test.txt")
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
	mut p Priority = Priority.HIGH
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
	mut s Scale = Scale.MEDIUM
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
	mut d Direction = Direction.NORTH
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
	mut p Point = Point{x: 10, y: 20}
	mut sum int = p.x + p.y
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
	mut o Outer = Outer{inner: Inner{value: 42}}
	mut v int = o.inner.value
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
	mut c Color = Color.RED
	mut isRed bool = c == Color.RED
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
	mut s Status = Status.PENDING
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
	mut x int = 5
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
	mut p Person = new(Person)
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
	mut arr [int] = {1, 2, 3, 4, 5}
	mut length int = len(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringLength(t *testing.T) {
	input := `
do main() {
	mut s string = "hello world"
	mut length int = len(s)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCompoundAssignment(t *testing.T) {
	input := `
do main() {
	mut x int = 10
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
			mut product int = i * j
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
	mut c Config = Config{name: "test", timeout: 30}
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
	mut arr [int] = {1, 2, 3}
	mut isEmpty bool = arrays.is_empty(arr)
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
	mut arr [int] = {1, 2, 3}
	mut firstElement int = arrays.first(arr)
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
	mut m1 map[string:int] = {"a": 1}
	mut m2 map[string:int] = {"b": 2}
	mut merged map[string:int] = maps.merge(m1, m2)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapRemove(t *testing.T) {
	input := `
import @maps
using maps

do takeBool(value bool) {
}

do main() {
	mut m map[string:int] = {"a": 1, "b": 2}
	mut removed = maps.remove(m, "a")
	takeBool(removed)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStdlibInference(t *testing.T) {
	input := `
import @uuid
import @random
import @maps
import @json

do takeString(value string) {
}

do takeInt(value int) {
}

do takeBool(value bool) {
}

do takeStrings(value [string]) {
}

do main() {
	mut id = uuid.generate()
	takeString(id)

	mut number = random.int(1, 10)
	takeInt(number)

	mut m map[string:int] = {"a": 1, "b": 2}
	mut keys = maps.keys(m)
	takeStrings(keys)

	mut valid = json.is_valid("{\"a\": 1}")
	takeBool(valid)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestUsingStdlibInference(t *testing.T) {
	input := `
import @maps
using maps

do takeBool(value bool) {
}

do main() {
	mut m map[string:int] = {"a": 1}
	mut empty = is_empty(m)
	takeBool(empty)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStdlibAliasInference(t *testing.T) {
	input := `
import m@maps

do takeBool(value bool) {
}

do main() {
	mut m0 map[string:int] = {"a": 1}
	mut empty = m.is_empty(m0)
	takeBool(empty)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringInterpolation(t *testing.T) {
	input := `
do main() {
	mut name string = "world"
	mut greeting string = "Hello, ${name}!"
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIfCondition(t *testing.T) {
	input := `
do main() {
	mut x int = 5
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
	mut a bool = true
	mut b bool = false
	mut andResult bool = a && b
	mut orResult bool = a || b
	mut notResult bool = !a
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestComparisonOperators(t *testing.T) {
	input := `
do main() {
	mut a int = 5
	mut b int = 3
	mut gt bool = a > b
	mut lt bool = a < b
	mut eq bool = a == b
	mut neq bool = a != b
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
	mut x int = 5
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
	mut items [string] = {"a", "b", "c"}
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
	mut sum int = 0
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
	mut sum int = add(5, 3)
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
	mut arr [int] = {1, 2, 3}
	mut reversed [int] = arrays.reverse(arr)
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
	mut arr [int] = {1, 2, 3}
	mut hasTwo bool = arrays.contains(arr, 2)
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
	mut result float = math.pow(2.0, 10.0)
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
	mut result float = math.sqrt(16.0)
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
	mut result float = math.ceil(3.2)
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
	mut result float = math.round(3.7)
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
	mut result string = strings.upper("hello")
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
	mut result string = strings.lower("HELLO")
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
	mut result string = strings.trim("  hello  ")
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
	mut result bool = strings.starts_with("hello world", "hello")
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
	mut result bool = strings.ends_with("hello world", "world")
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
	mut idx int = strings.index_of("hello world", "world")
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
	mut result int = math.min(5, 3)
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
	mut result int = math.max(5, 3)
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
	mut arr [int] = {3, 1, 4, 1, 5}
	arrays.sort(arr)
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
	mut arr [int] = {1, 2, 3, 4, 5}
	mut total int = arrays.sum(arr)
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
	mut m map[string:int] = {"a": 1, "b": 2}
	mut hasKey bool = maps.contains(m, "a")
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
	mut m map[string:int] = {"a": 1, "b": 2}
	mut mapSize int = maps.len(m)
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
	mut x int = 42
	mut f float = cast(x, float)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastFloatToInt(t *testing.T) {
	input := `
do main() {
	mut f float = 3.14
	mut x int = cast(f, int)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastIntToString(t *testing.T) {
	input := `
do main() {
	mut x int = 42
	mut s string = cast(x, string)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastStringToInt(t *testing.T) {
	input := `
do main() {
	mut s string = "42"
	mut x int = cast(s, int)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastIntToByte(t *testing.T) {
	input := `
do main() {
	mut x int = 65
	mut b byte = cast(x, byte)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastByteToInt(t *testing.T) {
	input := `
do main() {
	mut b byte = 65
	mut x int = cast(b, int)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastIntToChar(t *testing.T) {
	input := `
do main() {
	mut x int = 65
	mut c char = cast(x, char)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastCharToInt(t *testing.T) {
	input := `
do main() {
	mut c char = 'A'
	mut x int = cast(c, int)
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
	mut x int = 42
	mut f float = float(x)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFloatToIntConversion(t *testing.T) {
	input := `
do main() {
	mut f float = 3.14
	mut x int = int(f)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIntToStringConversion(t *testing.T) {
	input := `
do main() {
	mut x int = 42
	mut s string = string(x)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBoolToStringConversion(t *testing.T) {
	input := `
do main() {
	mut b bool = true
	mut s string = string(b)
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
	mut p Point = makePoint(10, 20)
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
	mut nums [int] = makeNumbers()
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
	mut result bool = isPositive(5)
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
	mut msg string = greet("World")
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
	mut result int = factorial(5)
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
	mut x int = getZero()
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
	mut p Person = Person{name: "Alice", age: 30}
	mut n string = p.name
	mut a int = p.age
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
	mut p Person = Person{name: "Alice", age: 30}
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
	mut content string, err error = io.read_file("test.txt")
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
	mut s Status = Status.PENDING
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
	mut points [Point] = {Point{x: 0, y: 0}, Point{x: 1, y: 1}}
	mut first Point = points[0]
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
	mut people map[string:Person] = {
		"alice": Person{name: "Alice", age: 30}
	}
	mut p Person = people["alice"]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToI64(t *testing.T) {
	input := `
do main() {
	mut x int = 1000000
	mut y i64 = cast(x, i64)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToU32(t *testing.T) {
	input := `
do main() {
	mut x int = 1000
	mut y u32 = cast(x, u32)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastToBool(t *testing.T) {
	input := `
do main() {
	mut x int = 1
	mut b bool = cast(x, bool)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMapWithIntKeys(t *testing.T) {
	input := `
do main() {
	mut m map[int:string] = {1: "one", 2: "two", 3: "three"}
	mut s string = m[1]
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
			mut product int = i * j
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
	mut result int = calculate(1, 2, 3, 4)
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
	mut r Record = Record{id: 1, name: "test", active: true, score: 95.5}
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
	mut area float = PI * 10.0 * 10.0
	mut size int = MAX_SIZE
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
	mut r Rectangle = Rectangle{origin: Point{x: 0, y: 0}, size: Size{width: 100, height: 50}}
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
	mut c Color = Color.RED
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
	mut a int = 5
	mut b int = 10
	mut c int = 3
	mut result int = (a + b) * c - (b / a)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringConcatenation(t *testing.T) {
	input := `
do main() {
	mut first string = "Hello"
	mut second string = " "
	mut third string = "World"
	mut result string = first + second + third
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
	mut arr [int] = {5, 3, 8, 1, 9}
	mut length int = len(arr)
	mut isEmpty bool = arrays.is_empty(arr)
	mut firstEl int = arrays.first(arr)
	mut lastEl int = arrays.last(arr)
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
	mut m map[string:int] = {"a": 1, "b": 2, "c": 3}
	mut hasA bool = maps.contains(m, "a")
	mut mapLen int = maps.len(m)
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
	mut o Outer = Outer{middle: Middle{inner: Inner{value: 42}}}
	mut v int = o.middle.inner.value
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEmptyArray(t *testing.T) {
	input := `
do main() {
	mut arr [int] = {}
	mut arrLen int = len(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEmptyMap(t *testing.T) {
	input := `
do main() {
	mut m map[string:int] = {}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestBooleanExpression(t *testing.T) {
	input := `
do main() {
	mut a bool = true
	mut b bool = false
	mut result bool = (a && !b) || (!a && b)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestComparisonChain(t *testing.T) {
	input := `
do main() {
	mut x int = 5
	mut isInRange bool = x > 0 && x < 10
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestMathOperations(t *testing.T) {
	input := `
do main() {
	mut a int = 10
	mut b int = 3
	mut sum int = a + b
	mut diff int = a - b
	mut prod int = a * b
	mut quot int = a / b
	mut rem int = a % b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFloatOperations(t *testing.T) {
	input := `
do main() {
	mut a float = 10.5
	mut b float = 3.2
	mut sum float = a + b
	mut diff float = a - b
	mut prod float = a * b
	mut quot float = a / b
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
	mut a int = 5
	mut neg int = -a
	mut b bool = true
	mut notB bool = !b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestLargeArrayLiteral(t *testing.T) {
	input := `
do main() {
	mut arr [int] = {1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestLargeMapLiteral(t *testing.T) {
	input := `
do main() {
	mut m map[string:int] = {
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
	mut x int = 5
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
	mut x int = MAX
	mut y int = MAX + 1
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// Additional tests to push coverage to 60%

func TestHashableMapKeyTypesExtended(t *testing.T) {
	input := `
do main() {
	mut m1 map[int:string] = {1: "one", 2: "two"}
	mut m2 map[bool:string] = {true: "yes", false: "no"}
	mut m3 map[char:int] = {'a': 1, 'b': 2}
	mut m4 map[byte:int] = {}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestComparisonOperatorsExtended(t *testing.T) {
	input := `
do main() {
	mut a int = 5
	mut b int = 10
	mut lt bool = a < b
	mut gt bool = a > b
	mut lte bool = a <= b
	mut gte bool = a >= b
	mut eq bool = a == b
	mut neq bool = a != b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestLogicalOperatorsExtended(t *testing.T) {
	input := `
do main() {
	mut a bool = true
	mut b bool = false
	mut andResult bool = a && b
	mut orResult bool = a || b
	mut complex bool = (a && b) || (!a && !b)
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
	mut o Outer = new(Outer)
	o.inner.value = 42
	mut v int = o.inner.value
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
	mut result int = square(5)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestStringIndexing(t *testing.T) {
	input := `
do main() {
	mut s string = "hello"
	mut ch char = s[0]
	mut last char = s[4]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIndexExpressionOnMapExtended(t *testing.T) {
	input := `
do main() {
	mut m map[string:int] = {"a": 1, "b": 2}
	mut val int = m["a"]
	mut val2 int = m["b"]
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestGlobalVariableWithInitialization(t *testing.T) {
	input := `
mut GLOBAL_VAL int = 100

do main() {
	mut x int = GLOBAL_VAL
	GLOBAL_VAL = 200
	mut y int = GLOBAL_VAL
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestWhileLoopExtended(t *testing.T) {
	input := `
do main() {
	mut i int = 0
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
	mut x int = 5
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
	mut s Status = Status.PENDING
	mut isActive bool = s == Status.ACTIVE
	mut notDone bool = s != Status.DONE
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEmptyArrayLiteral(t *testing.T) {
	input := `
do main() {
	mut arr [int] = {}
	mut strArr [string] = {}
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestEmptyMapLiteral(t *testing.T) {
	input := `
do main() {
	mut m map[string:int] = {}
	mut m2 map[int:bool] = {}
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
		"nil", "Error",
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
	mut a string = "hello"
	mut b string = "world"
	mut c string = a + " " + b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestFloatArithmetic(t *testing.T) {
	input := `
do main() {
	mut a float = 3.14
	mut b float = 2.71
	mut sum float = a + b
	mut diff float = a - b
	mut prod float = a * b
	mut quot float = a / b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIntegerArithmeticExtended(t *testing.T) {
	input := `
do main() {
	mut a int = 10
	mut b int = 3
	mut sum int = a + b
	mut diff int = a - b
	mut prod int = a * b
	mut quot int = a / b
	mut mod int = a % b
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestNestedIfStatement(t *testing.T) {
	input := `
do main() {
	mut x int = 5
	mut y int = 10
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
	mut arr [int] = {1, 2, 3, 4, 5}
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
	mut s string = "hello"
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
	mut result int = add(5, 3)
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
	mut r Rectangle = new(Rectangle)
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
	mut s string = "hello world"
	mut upperStr string = strings.upper(s)
	mut lowerStr string = strings.lower(s)
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
	mut absVal int = math.abs(-5)
	mut maxVal int = math.max(10, 20)
	mut minVal int = math.min(10, 20)
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
	mut arr [int] = {3, 1, 4, 1, 5}
	mut isEmpty bool = arrays.is_empty(arr)
	mut length int = len(arr)
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
	mut arr [int] = {1, 2, 3}
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
	mut arr [string] = {"a", "b"}
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
	mut arr [int] = {1, 2, 3}
	mut found bool = arrays.contains(arr, "string")
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
	mut arr [int] = {1, 2, 3}
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
	mut arr [bool] = {true, false}
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
	mut arr1 [int] = {1, 2}
	mut arr2 [string] = {"a", "b"}
	mut result [int] = arrays.concat(arr1, arr2)
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
	mut arr [int] = {1, 2, 3}
	arrays.fill(arr, "not an int")
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
	mut arr [int] = {1, 2, 3, 2}
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
	mut arr [int] = {1, 2, 3, 2}
	mut idx int = arrays.last_index(arr, 3.14)
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
	mut arr [string] = {"a", "b", "a"}
	mut count int = arrays.count(arr, 42)
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
	mut arr [int] = {1, 2, 3}
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
	mut arr [string] = {"a", "b"}
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
	mut arr1 [int] = {1, 2}
	mut arr2 [int] = {3, 4}
	mut result [int] = arrays.concat(arr1, arr2)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// Map Key/Value Type Compatibility Tests (checkMapKeyValueTypeCompatibility)
// ============================================================================

func TestMapsSetKeyTypeMismatch(t *testing.T) {
	input := `
import @maps
using maps

do main() {
	mut m map[string:int] = {"a": 1}
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
	mut m map[string:int] = {"a": 1}
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
	mut m map[string:int] = {"a": 1}
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
	mut m map[int:string] = {1: "one"}
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
	mut m map[int:string] = {1: "one"}
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
	mut m map[string:int] = {"a": 1, "b": 2}
	mut found bool = maps.contains(m, "a")
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
	mut m map[string:int] = {"a": 1}
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
	mut m map[string:int] = {"a": 1}
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
	mut p Point = Point{x: 1, y: 2}
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
	mut o Outer = Outer{inner: Inner{value: 42}}
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
	mut s string = "3.14"
	mut f float = float(s)
}
`
	_ = typecheck(t, input)
	// This is expected to give a warning about runtime conversion
	// Just check that it compiles
}

func TestCharToIntConversionCoverage(t *testing.T) {
	input := `
do main() {
	mut c char = 'A'
	mut i int = int(c)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestIntToCharConversionCoverage(t *testing.T) {
	input := `
do main() {
	mut i int = 65
	mut c char = char(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestByteConversionsCoverage(t *testing.T) {
	input := `
do main() {
	mut b byte = byte(65)
	mut i int = int(b)
	mut c char = char(b)
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
	mut i int = 42
	mut i32val i32 = i32(i)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastI64ToInt(t *testing.T) {
	input := `
do main() {
	mut i64val i64 = 42
	mut i int = int(i64val)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastU8ToU16(t *testing.T) {
	input := `
do main() {
	mut u8val u8 = 255
	mut u16val u16 = u16(u8val)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestCastF32ToFloat(t *testing.T) {
	input := `
do main() {
	mut f32val f32 = 3.14
	mut fval float = float(f32val)
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
	mut x, _ = getTwo()
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
	mut _, _, _ = getThree()
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
	mut x, y = getTwo()
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
	mut min i32 = -2147483648
	mut max i32 = 2147483647
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestU32BoundsCoverage(t *testing.T) {
	input := `
do main() {
	mut min u32 = 0
	mut max u32 = 4294967295
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestI64BoundsCoverage(t *testing.T) {
	input := `
do main() {
	mut val i64 = 9223372036854775807
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestU64BoundsCoverage(t *testing.T) {
	input := `
do main() {
	mut val u64 = 18446744073709551615
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
	mut arr [int] = {1, 2, 3}
	mut length int = len(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestInferTypeofReturnType(t *testing.T) {
	input := `
do main() {
	mut x int = 42
	mut typeName string = typeof(x)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestInferCopyReturnType(t *testing.T) {
	input := `
do main() {
	mut arr [int] = {1, 2, 3}
	mut arrCopy [int] = copy(arr)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestInferAppendReturnType(t *testing.T) {
	input := `
do main() {
	mut arr [int] = {1, 2, 3}
	mut newArr [int] = append(arr, 4)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

func TestInferRangeReturnType(t *testing.T) {
	input := `
do main() {
	mut nums [int] = range(1, 10)
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
	mut x = 42
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
	mut outer [int] = {1, 2}
	mut inner [int] = {3, 4}
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
	mut arr [int] = {10, 20}
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
	mut i int = 100
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
	mut x int = 10
	if true {
		mut x int = 20
		println(x)
	}
	println(x)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
}

// ============================================================================
// W2010 Nested Struct Initialization Tests (#1107)
// ============================================================================

func TestW2010_NestedStructLiteralNoFalsePositive(t *testing.T) {
	// When a nested struct field is explicitly initialized with a struct literal,
	// W2010 should NOT fire on chained member access
	input := `
const Container struct {
	label string
}

const Wrapper struct {
	inner Container
}

do main() {
	mut w Wrapper = Wrapper{ inner: Container{ label: "test" } }
	w.inner.label = "modified"
	println(w.inner.label)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
	assertNoWarning(t, tc, errors.W2010)
}

func TestW2010_DeeplyNestedStructLiteralNoFalsePositive(t *testing.T) {
	// Test deeply nested struct initialization
	input := `
const Inner struct {
	value string
}

const Middle struct {
	nested Inner
}

const Outer struct {
	mid Middle
}

do main() {
	mut o Outer = Outer{
		mid: Middle{
			nested: Inner{ value: "deep" }
		}
	}
	o.mid.nested.value = "modified"
	println(o.mid.nested.value)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
	assertNoWarning(t, tc, errors.W2010)
}

func TestW2010_ReassignmentWithStructLiteral(t *testing.T) {
	// Test that reassigning a variable with a struct literal also marks nested fields
	input := `
const Container struct {
	label string
}

const Wrapper struct {
	inner Container
}

do main() {
	mut w Wrapper = Wrapper{}
	w = Wrapper{ inner: Container{ label: "test" } }
	w.inner.label = "modified"
	println(w.inner.label)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
	assertNoWarning(t, tc, errors.W2010)
}

// ============================================================================
// Named Return Variables Tests (#1131)
// ============================================================================

func TestNamedReturnBasicNoWarning(t *testing.T) {
	input := `
do getName() -> (name string) {
	name = "Alice"
	return name
}

do main() {
	mut n string = getName()
	println(n)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
	assertNoWarning(t, tc, errors.W2011)
}

func TestNamedReturnDifferentVariableWarning(t *testing.T) {
	input := `
do getName() -> (name string) {
	mut other string = "Hello"
	return other
}

do main() {
	mut n string = getName()
	println(n)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
	assertHasWarning(t, tc, errors.W2011)
}

func TestNamedReturnExpressionWarning(t *testing.T) {
	input := `
do getNumber() -> (result int) {
	return 42
}

do main() {
	mut n int = getNumber()
	println(n)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
	assertHasWarning(t, tc, errors.W2011)
}

func TestNamedReturnMultipleCorrect(t *testing.T) {
	input := `
do getValues() -> (age int, name string) {
	age = 25
	name = "Bob"
	return age, name
}

do main() {
	mut a int, n string = getValues()
	println(a)
	println(n)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
	assertNoWarning(t, tc, errors.W2011)
}

func TestNamedReturnMultipleOneWrong(t *testing.T) {
	input := `
do getValues() -> (age int, name string) {
	mut other string = "Alice"
	age = 25
	return age, other
}

do main() {
	mut a int, n string = getValues()
	println(a)
	println(n)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
	assertHasWarning(t, tc, errors.W2011)
}

func TestNamedReturnSharedType(t *testing.T) {
	input := `
do getNames() -> (first, last string) {
	first = "John"
	last = "Doe"
	return first, last
}

do main() {
	mut f string, l string = getNames()
	println(f)
	println(l)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
	assertNoWarning(t, tc, errors.W2011)
}

func TestNamedReturnZeroValueUsage(t *testing.T) {
	// Using the auto-initialized zero value is valid
	input := `
do getZero() -> (count int) {
	return count
}

do main() {
	mut c int = getZero()
	println(c)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
	assertNoWarning(t, tc, errors.W2011)
}

func TestUnnamedReturnNoWarning(t *testing.T) {
	// Unnamed returns should not trigger W2011
	input := `
do getValue() -> int {
	return 42
}

do main() {
	mut v int = getValue()
	println(v)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
	assertNoWarning(t, tc, errors.W2011)
}

func TestUnnamedMultiReturnNoWarning(t *testing.T) {
	// Unnamed multi-returns should not trigger W2011
	input := `
do getValues() -> (int, string) {
	return 42, "hello"
}

do main() {
	mut a int, b string = getValues()
	println(a)
	println(b)
}
`
	tc := typecheck(t, input)
	assertNoErrors(t, tc)
	assertNoWarning(t, tc, errors.W2011)
}
