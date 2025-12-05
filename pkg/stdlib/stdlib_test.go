package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"math"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// ============================================================================
// Test Helpers
// ============================================================================

func testIntegerObject(t *testing.T, obj object.Object, expected int64) bool {
	t.Helper()
	result, ok := obj.(*object.Integer)
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

func testFloatObject(t *testing.T, obj object.Object, expected float64) bool {
	t.Helper()
	result, ok := obj.(*object.Float)
	if !ok {
		t.Errorf("object is not Float. got=%T (%+v)", obj, obj)
		return false
	}
	// Use small epsilon for float comparison
	if math.Abs(result.Value-expected) > 0.0001 {
		t.Errorf("object has wrong value. got=%f, want=%f", result.Value, expected)
		return false
	}
	return true
}

func testBooleanObject(t *testing.T, obj object.Object, expected bool) bool {
	t.Helper()
	result, ok := obj.(*object.Boolean)
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

func testStringObject(t *testing.T, obj object.Object, expected string) bool {
	t.Helper()
	result, ok := obj.(*object.String)
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

func isErrorObject(obj object.Object) bool {
	_, ok := obj.(*object.Error)
	return ok
}

// ============================================================================
// GetAllBuiltins Tests
// ============================================================================

func TestGetAllBuiltins(t *testing.T) {
	builtins := GetAllBuiltins()
	if builtins == nil {
		t.Fatal("GetAllBuiltins returned nil")
	}
	if len(builtins) == 0 {
		t.Error("GetAllBuiltins returned empty map")
	}

	// Check some expected builtins exist
	expectedBuiltins := []string{
		"len", "typeof", "int", "float", "string",
		"math.add", "math.sub", "math.abs",
		"arrays.append", "arrays.pop", "arrays.is_empty",
		"strings.upper", "strings.lower",
	}

	for _, name := range expectedBuiltins {
		if builtins[name] == nil {
			t.Errorf("expected builtin %q not found", name)
		}
	}
}

// ============================================================================
// Builtins Tests (std module)
// ============================================================================

func TestLen(t *testing.T) {
	lenFn := StdBuiltins["len"].Fn

	tests := []struct {
		name     string
		input    object.Object
		expected int64
	}{
		{"empty string", &object.String{Value: ""}, 0},
		{"string", &object.String{Value: "hello"}, 5},
		{"empty array", &object.Array{Elements: []object.Object{}}, 0},
		{"array", &object.Array{Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
			&object.Integer{Value: 3},
		}}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lenFn(tt.input)
			testIntegerObject(t, result, tt.expected)
		})
	}
}

func TestLenErrors(t *testing.T) {
	lenFn := StdBuiltins["len"].Fn

	// Wrong number of arguments
	result := lenFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	result = lenFn(&object.Integer{Value: 5}, &object.Integer{Value: 10})
	if !isErrorObject(result) {
		t.Error("expected error for too many arguments")
	}

	// Unsupported type
	result = lenFn(&object.Integer{Value: 5})
	if !isErrorObject(result) {
		t.Error("expected error for unsupported type")
	}
}

func TestTypeof(t *testing.T) {
	typeofFn := StdBuiltins["typeof"].Fn

	tests := []struct {
		name     string
		input    object.Object
		expected string
	}{
		{"integer", &object.Integer{Value: 5}, "int"},
		{"float", &object.Float{Value: 3.14}, "float"},
		{"string", &object.String{Value: "hello"}, "string"},
		{"boolean", &object.Boolean{Value: true}, "bool"},
		{"nil", object.NIL, "nil"},
		{"array", &object.Array{Elements: []object.Object{}}, "array"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeofFn(tt.input)
			testStringObject(t, result, tt.expected)
		})
	}
}

func TestIntConversion(t *testing.T) {
	intFn := StdBuiltins["int"].Fn

	tests := []struct {
		name     string
		input    object.Object
		expected int64
	}{
		{"integer", &object.Integer{Value: 42}, 42},
		{"float", &object.Float{Value: 3.7}, 3},
		{"string", &object.String{Value: "123"}, 123},
		{"string with underscores", &object.String{Value: "1_000"}, 1000},
		{"char", &object.Char{Value: 'A'}, 65},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := intFn(tt.input)
			testIntegerObject(t, result, tt.expected)
		})
	}
}

func TestIntConversionErrors(t *testing.T) {
	intFn := StdBuiltins["int"].Fn

	// Invalid string
	result := intFn(&object.String{Value: "not a number"})
	if !isErrorObject(result) {
		t.Error("expected error for invalid string")
	}
}

func TestFloatConversion(t *testing.T) {
	floatFn := StdBuiltins["float"].Fn

	tests := []struct {
		name     string
		input    object.Object
		expected float64
	}{
		{"integer", &object.Integer{Value: 42}, 42.0},
		{"float", &object.Float{Value: 3.14}, 3.14},
		{"string", &object.String{Value: "3.14"}, 3.14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := floatFn(tt.input)
			testFloatObject(t, result, tt.expected)
		})
	}
}

func TestStringConversion(t *testing.T) {
	stringFn := StdBuiltins["string"].Fn

	tests := []struct {
		name     string
		input    object.Object
		expected string
	}{
		{"integer", &object.Integer{Value: 42}, "42"},
		{"float", &object.Float{Value: 3.14}, "3.14"},
		// Note: string() on a string returns the quoted version in EZ
		{"boolean true", &object.Boolean{Value: true}, "true"},
		{"boolean false", &object.Boolean{Value: false}, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringFn(tt.input)
			testStringObject(t, result, tt.expected)
		})
	}
}

func TestStringConversionForString(t *testing.T) {
	stringFn := StdBuiltins["string"].Fn
	// string() on a string returns the quoted version
	result := stringFn(&object.String{Value: "hello"})
	str, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	// The implementation returns the string with quotes
	if str.Value != "\"hello\"" && str.Value != "hello" {
		t.Errorf("unexpected result: %q", str.Value)
	}
}

// ============================================================================
// Char Conversion Tests
// ============================================================================

func testCharObject(t *testing.T, obj object.Object, expected rune) bool {
	t.Helper()
	result, ok := obj.(*object.Char)
	if !ok {
		t.Errorf("object is not Char. got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("object has wrong value. got=%c (%d), want=%c (%d)", result.Value, result.Value, expected, expected)
		return false
	}
	return true
}

func TestCharConversion(t *testing.T) {
	charFn := StdBuiltins["char"].Fn

	tests := []struct {
		name     string
		input    object.Object
		expected rune
	}{
		{"char identity", &object.Char{Value: 'A'}, 'A'},
		{"integer to char", &object.Integer{Value: 65}, 'A'},
		{"integer to char B", &object.Integer{Value: 66}, 'B'},
		{"float to char", &object.Float{Value: 67.9}, 'C'},
		{"byte to char", &object.Byte{Value: 68}, 'D'},
		{"string to char", &object.String{Value: "E"}, 'E'},
		{"unicode char", &object.Integer{Value: 0x1F600}, 0x1F600}, // 😀
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := charFn(tt.input)
			testCharObject(t, result, tt.expected)
		})
	}
}

func TestCharConversionErrors(t *testing.T) {
	charFn := StdBuiltins["char"].Fn

	tests := []struct {
		name  string
		input object.Object
	}{
		{"negative integer", &object.Integer{Value: -1}},
		{"too large integer", &object.Integer{Value: 0x110000}}, // Beyond Unicode range
		{"multi-char string", &object.String{Value: "AB"}},
		{"empty string", &object.String{Value: ""}},
		{"negative float", &object.Float{Value: -1.5}},
		{"array", &object.Array{Elements: []object.Object{}}},
		{"map", object.NewMap()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := charFn(tt.input)
			if !isErrorObject(result) {
				t.Errorf("expected error for %s, got %T", tt.name, result)
			}
		})
	}
}

func TestCharConversionArgCount(t *testing.T) {
	charFn := StdBuiltins["char"].Fn

	// No arguments
	result := charFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Too many arguments
	result = charFn(&object.Integer{Value: 65}, &object.Integer{Value: 66})
	if !isErrorObject(result) {
		t.Error("expected error for too many arguments")
	}
}

// ============================================================================
// Math Module Tests
// ============================================================================

func TestMathAdd(t *testing.T) {
	addFn := MathBuiltins["math.add"].Fn

	tests := []struct {
		name     string
		a, b     object.Object
		expected float64
		isInt    bool
	}{
		{"integers", &object.Integer{Value: 5}, &object.Integer{Value: 3}, 8, true},
		{"floats", &object.Float{Value: 2.5}, &object.Float{Value: 1.5}, 4.0, false},
		{"mixed", &object.Integer{Value: 5}, &object.Float{Value: 2.5}, 7.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addFn(tt.a, tt.b)
			if tt.isInt {
				testIntegerObject(t, result, int64(tt.expected))
			} else {
				testFloatObject(t, result, tt.expected)
			}
		})
	}
}

func TestMathSub(t *testing.T) {
	subFn := MathBuiltins["math.sub"].Fn

	result := subFn(&object.Integer{Value: 10}, &object.Integer{Value: 3})
	testIntegerObject(t, result, 7)

	result = subFn(&object.Float{Value: 5.5}, &object.Float{Value: 2.0})
	testFloatObject(t, result, 3.5)
}

func TestMathMul(t *testing.T) {
	mulFn := MathBuiltins["math.mul"].Fn

	result := mulFn(&object.Integer{Value: 6}, &object.Integer{Value: 7})
	testIntegerObject(t, result, 42)

	result = mulFn(&object.Float{Value: 2.5}, &object.Float{Value: 4.0})
	testFloatObject(t, result, 10.0)
}

func TestMathDiv(t *testing.T) {
	divFn := MathBuiltins["math.div"].Fn

	result := divFn(&object.Integer{Value: 10}, &object.Integer{Value: 4})
	testFloatObject(t, result, 2.5)

	// Division by zero
	result = divFn(&object.Integer{Value: 10}, &object.Integer{Value: 0})
	if !isErrorObject(result) {
		t.Error("expected error for division by zero")
	}
}

func TestMathMod(t *testing.T) {
	modFn := MathBuiltins["math.mod"].Fn

	result := modFn(&object.Integer{Value: 10}, &object.Integer{Value: 3})
	testFloatObject(t, result, 1.0)

	// Modulo by zero
	result = modFn(&object.Integer{Value: 10}, &object.Integer{Value: 0})
	if !isErrorObject(result) {
		t.Error("expected error for modulo by zero")
	}
}

func TestMathAbs(t *testing.T) {
	absFn := MathBuiltins["math.abs"].Fn

	tests := []struct {
		name     string
		input    object.Object
		expected float64
		isInt    bool
	}{
		{"positive int", &object.Integer{Value: 5}, 5, true},
		{"negative int", &object.Integer{Value: -5}, 5, true},
		{"positive float", &object.Float{Value: 3.14}, 3.14, false},
		{"negative float", &object.Float{Value: -3.14}, 3.14, false},
		{"zero", &object.Integer{Value: 0}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := absFn(tt.input)
			if tt.isInt {
				testIntegerObject(t, result, int64(tt.expected))
			} else {
				testFloatObject(t, result, tt.expected)
			}
		})
	}
}

func TestMathSign(t *testing.T) {
	signFn := MathBuiltins["math.sign"].Fn

	tests := []struct {
		name     string
		input    object.Object
		expected int64
	}{
		{"positive", &object.Integer{Value: 42}, 1},
		{"negative", &object.Integer{Value: -42}, -1},
		{"zero", &object.Integer{Value: 0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := signFn(tt.input)
			testIntegerObject(t, result, tt.expected)
		})
	}
}

func TestMathNeg(t *testing.T) {
	negFn := MathBuiltins["math.neg"].Fn

	result := negFn(&object.Integer{Value: 5})
	testIntegerObject(t, result, -5)

	result = negFn(&object.Integer{Value: -5})
	testIntegerObject(t, result, 5)

	result = negFn(&object.Float{Value: 3.14})
	testFloatObject(t, result, -3.14)
}

func TestMathMinMax(t *testing.T) {
	minFn := MathBuiltins["math.min"].Fn
	maxFn := MathBuiltins["math.max"].Fn

	result := minFn(&object.Integer{Value: 5}, &object.Integer{Value: 3})
	testIntegerObject(t, result, 3)

	result = maxFn(&object.Integer{Value: 5}, &object.Integer{Value: 3})
	testIntegerObject(t, result, 5)
}

func TestMathPow(t *testing.T) {
	powFn := MathBuiltins["math.pow"].Fn

	// Integer power returns integer when result is whole number
	result := powFn(&object.Integer{Value: 2}, &object.Integer{Value: 3})
	testIntegerObject(t, result, 8)

	// Float power returns float
	result = powFn(&object.Float{Value: 2.0}, &object.Float{Value: 0.5})
	testFloatObject(t, result, math.Sqrt(2.0))
}

func TestMathSqrt(t *testing.T) {
	sqrtFn := MathBuiltins["math.sqrt"].Fn

	result := sqrtFn(&object.Integer{Value: 16})
	testFloatObject(t, result, 4.0)

	result = sqrtFn(&object.Float{Value: 2.0})
	testFloatObject(t, result, math.Sqrt(2.0))
}

func TestMathFloorCeil(t *testing.T) {
	floorFn := MathBuiltins["math.floor"].Fn
	ceilFn := MathBuiltins["math.ceil"].Fn

	result := floorFn(&object.Float{Value: 3.7})
	testIntegerObject(t, result, 3)

	result = ceilFn(&object.Float{Value: 3.2})
	testIntegerObject(t, result, 4)
}

func TestMathRound(t *testing.T) {
	roundFn := MathBuiltins["math.round"].Fn

	result := roundFn(&object.Float{Value: 3.4})
	testIntegerObject(t, result, 3)

	result = roundFn(&object.Float{Value: 3.5})
	testIntegerObject(t, result, 4)
}

func TestMathTrigonometry(t *testing.T) {
	sinFn := MathBuiltins["math.sin"].Fn
	cosFn := MathBuiltins["math.cos"].Fn

	// sin(0) = 0
	result := sinFn(&object.Float{Value: 0})
	testFloatObject(t, result, 0.0)

	// cos(0) = 1
	result = cosFn(&object.Float{Value: 0})
	testFloatObject(t, result, 1.0)
}

func TestMathConstants(t *testing.T) {
	piFn := MathBuiltins["math.PI"].Fn
	eFn := MathBuiltins["math.E"].Fn

	result := piFn()
	testFloatObject(t, result, math.Pi)

	result = eFn()
	testFloatObject(t, result, math.E)
}

func TestMathLogBase(t *testing.T) {
	logBaseFn := MathBuiltins["math.log_base"].Fn

	// log_base(8, 2) = 3 (since 2^3 = 8)
	result := logBaseFn(&object.Integer{Value: 8}, &object.Integer{Value: 2})
	testFloatObject(t, result, 3.0)

	// log_base(1000, 10) = 3 (since 10^3 = 1000)
	result = logBaseFn(&object.Integer{Value: 1000}, &object.Integer{Value: 10})
	testFloatObject(t, result, 3.0)

	// log_base(27, 3) = 3 (since 3^3 = 27)
	result = logBaseFn(&object.Integer{Value: 27}, &object.Integer{Value: 3})
	testFloatObject(t, result, 3.0)

	// log_base(1, any) = 0
	result = logBaseFn(&object.Integer{Value: 1}, &object.Integer{Value: 5})
	testFloatObject(t, result, 0.0)
}

func TestMathLogBaseErrors(t *testing.T) {
	logBaseFn := MathBuiltins["math.log_base"].Fn

	// Value <= 0
	result := logBaseFn(&object.Integer{Value: 0}, &object.Integer{Value: 2})
	if !isErrorObject(result) {
		t.Error("expected error for value <= 0")
	}

	result = logBaseFn(&object.Integer{Value: -5}, &object.Integer{Value: 2})
	if !isErrorObject(result) {
		t.Error("expected error for negative value")
	}

	// Base <= 0
	result = logBaseFn(&object.Integer{Value: 8}, &object.Integer{Value: 0})
	if !isErrorObject(result) {
		t.Error("expected error for base <= 0")
	}

	// Base = 1 (undefined)
	result = logBaseFn(&object.Integer{Value: 8}, &object.Integer{Value: 1})
	if !isErrorObject(result) {
		t.Error("expected error for base = 1")
	}

	// Wrong number of args
	result = logBaseFn(&object.Integer{Value: 8})
	if !isErrorObject(result) {
		t.Error("expected error for 1 argument")
	}
}

// ============================================================================
// Arrays Module Tests
// ============================================================================

func TestArraysIsEmpty(t *testing.T) {
	isEmptyFn := ArraysBuiltins["arrays.is_empty"].Fn

	// Empty array
	result := isEmptyFn(&object.Array{Elements: []object.Object{}})
	testBooleanObject(t, result, true)

	// Non-empty array
	result = isEmptyFn(&object.Array{Elements: []object.Object{&object.Integer{Value: 1}}})
	testBooleanObject(t, result, false)
}

func TestArraysAppend(t *testing.T) {
	appendFn := ArraysBuiltins["arrays.append"].Fn

	arr := &object.Array{Elements: []object.Object{&object.Integer{Value: 1}}, Mutable: true}
	appendFn(arr, &object.Integer{Value: 2}, &object.Integer{Value: 3})

	if len(arr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestArraysAppendImmutable(t *testing.T) {
	appendFn := ArraysBuiltins["arrays.append"].Fn

	arr := &object.Array{Elements: []object.Object{&object.Integer{Value: 1}}, Mutable: false}
	result := appendFn(arr, &object.Integer{Value: 2})

	if !isErrorObject(result) {
		t.Error("expected error for immutable array")
	}
}

func TestArraysPop(t *testing.T) {
	popFn := ArraysBuiltins["arrays.pop"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
			&object.Integer{Value: 3},
		},
		Mutable: true,
	}

	result := popFn(arr)
	testIntegerObject(t, result, 3)

	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 elements after pop, got %d", len(arr.Elements))
	}
}

func TestArraysPopEmpty(t *testing.T) {
	popFn := ArraysBuiltins["arrays.pop"].Fn

	arr := &object.Array{Elements: []object.Object{}, Mutable: true}
	result := popFn(arr)

	if !isErrorObject(result) {
		t.Error("expected error for pop on empty array")
	}
}

func TestArraysLenUsingBuiltinLen(t *testing.T) {
	// arrays module doesn't have its own len, use the global len builtin
	lenFn := StdBuiltins["len"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: 1},
		&object.Integer{Value: 2},
		&object.Integer{Value: 3},
	}}

	result := lenFn(arr)
	testIntegerObject(t, result, 3)
}

func TestArraysFirst(t *testing.T) {
	firstFn := ArraysBuiltins["arrays.first"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: 10},
		&object.Integer{Value: 20},
	}}

	result := firstFn(arr)
	testIntegerObject(t, result, 10)
}

func TestArraysLast(t *testing.T) {
	lastFn := ArraysBuiltins["arrays.last"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: 10},
		&object.Integer{Value: 20},
	}}

	result := lastFn(arr)
	testIntegerObject(t, result, 20)
}

func TestArraysContains(t *testing.T) {
	containsFn := ArraysBuiltins["arrays.contains"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: 1},
		&object.Integer{Value: 2},
		&object.Integer{Value: 3},
	}}

	result := containsFn(arr, &object.Integer{Value: 2})
	testBooleanObject(t, result, true)

	result = containsFn(arr, &object.Integer{Value: 5})
	testBooleanObject(t, result, false)
}

func TestArraysIndexOf(t *testing.T) {
	indexOfFn := ArraysBuiltins["arrays.index_of"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: 10},
		&object.Integer{Value: 20},
		&object.Integer{Value: 30},
	}}

	result := indexOfFn(arr, &object.Integer{Value: 20})
	testIntegerObject(t, result, 1)

	result = indexOfFn(arr, &object.Integer{Value: 99})
	testIntegerObject(t, result, -1)
}

func TestArraysReverse(t *testing.T) {
	reverseFn := ArraysBuiltins["arrays.reverse"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
			&object.Integer{Value: 3},
		},
		Mutable: true,
	}

	// arrays.reverse returns a NEW reversed array
	result := reverseFn(arr)
	newArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	// Check reversed order in returned array
	testIntegerObject(t, newArr.Elements[0], 3)
	testIntegerObject(t, newArr.Elements[1], 2)
	testIntegerObject(t, newArr.Elements[2], 1)
}

func TestArraysSum(t *testing.T) {
	sumFn := ArraysBuiltins["arrays.sum"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: 1},
		&object.Integer{Value: 2},
		&object.Integer{Value: 3},
		&object.Integer{Value: 4},
	}}

	result := sumFn(arr)
	testIntegerObject(t, result, 10)
}

// ============================================================================
// Strings Module Tests
// ============================================================================

func TestStringsUpper(t *testing.T) {
	upperFn := StringsBuiltins["strings.upper"].Fn

	result := upperFn(&object.String{Value: "hello"})
	testStringObject(t, result, "HELLO")

	result = upperFn(&object.String{Value: "Hello World"})
	testStringObject(t, result, "HELLO WORLD")
}

func TestStringsLower(t *testing.T) {
	lowerFn := StringsBuiltins["strings.lower"].Fn

	result := lowerFn(&object.String{Value: "HELLO"})
	testStringObject(t, result, "hello")

	result = lowerFn(&object.String{Value: "Hello World"})
	testStringObject(t, result, "hello world")
}

func TestStringsCapitalize(t *testing.T) {
	capitalizeFn := StringsBuiltins["strings.capitalize"].Fn

	// capitalize only uppercases the first letter, leaves rest unchanged
	result := capitalizeFn(&object.String{Value: "hello"})
	testStringObject(t, result, "Hello")

	// For already uppercase, only first char is affected
	result = capitalizeFn(&object.String{Value: "HELLO"})
	testStringObject(t, result, "HELLO") // First char already upper, rest stays as-is
}

func TestStringsContains(t *testing.T) {
	containsFn := StringsBuiltins["strings.contains"].Fn

	result := containsFn(&object.String{Value: "hello world"}, &object.String{Value: "world"})
	testBooleanObject(t, result, true)

	result = containsFn(&object.String{Value: "hello world"}, &object.String{Value: "foo"})
	testBooleanObject(t, result, false)
}

func TestStringsStartsWith(t *testing.T) {
	startsWithFn := StringsBuiltins["strings.starts_with"].Fn

	result := startsWithFn(&object.String{Value: "hello world"}, &object.String{Value: "hello"})
	testBooleanObject(t, result, true)

	result = startsWithFn(&object.String{Value: "hello world"}, &object.String{Value: "world"})
	testBooleanObject(t, result, false)
}

func TestStringsEndsWith(t *testing.T) {
	endsWithFn := StringsBuiltins["strings.ends_with"].Fn

	result := endsWithFn(&object.String{Value: "hello world"}, &object.String{Value: "world"})
	testBooleanObject(t, result, true)

	result = endsWithFn(&object.String{Value: "hello world"}, &object.String{Value: "hello"})
	testBooleanObject(t, result, false)
}

func TestStringsTrim(t *testing.T) {
	trimFn := StringsBuiltins["strings.trim"].Fn

	result := trimFn(&object.String{Value: "  hello  "})
	testStringObject(t, result, "hello")

	result = trimFn(&object.String{Value: "\t\nhello\n\t"})
	testStringObject(t, result, "hello")
}

func TestStringsSplit(t *testing.T) {
	splitFn := StringsBuiltins["strings.split"].Fn

	result := splitFn(&object.String{Value: "a,b,c"}, &object.String{Value: ","})
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(arr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr.Elements))
	}

	testStringObject(t, arr.Elements[0], "a")
	testStringObject(t, arr.Elements[1], "b")
	testStringObject(t, arr.Elements[2], "c")
}

func TestStringsJoin(t *testing.T) {
	joinFn := StringsBuiltins["strings.join"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.String{Value: "a"},
		&object.String{Value: "b"},
		&object.String{Value: "c"},
	}}

	result := joinFn(arr, &object.String{Value: ","})
	testStringObject(t, result, "a,b,c")
}

func TestStringsReplace(t *testing.T) {
	replaceFn := StringsBuiltins["strings.replace"].Fn

	result := replaceFn(
		&object.String{Value: "hello world"},
		&object.String{Value: "world"},
		&object.String{Value: "Go"},
	)
	testStringObject(t, result, "hello Go")
}

func TestStringsLenUsingBuiltinLen(t *testing.T) {
	// strings module doesn't have its own len, use the global len builtin
	lenFn := StdBuiltins["len"].Fn

	result := lenFn(&object.String{Value: "hello"})
	testIntegerObject(t, result, 5)

	result = lenFn(&object.String{Value: ""})
	testIntegerObject(t, result, 0)
}

func TestStringsIsEmpty(t *testing.T) {
	isEmptyFn := StringsBuiltins["strings.is_empty"].Fn

	result := isEmptyFn(&object.String{Value: ""})
	testBooleanObject(t, result, true)

	result = isEmptyFn(&object.String{Value: "hello"})
	testBooleanObject(t, result, false)
}

func TestStringsRepeat(t *testing.T) {
	repeatFn := StringsBuiltins["strings.repeat"].Fn

	result := repeatFn(&object.String{Value: "ab"}, &object.Integer{Value: 3})
	testStringObject(t, result, "ababab")
}

func TestStringsReverse(t *testing.T) {
	reverseFn := StringsBuiltins["strings.reverse"].Fn

	result := reverseFn(&object.String{Value: "hello"})
	testStringObject(t, result, "olleh")
}

func TestStringsReplaceN(t *testing.T) {
	replaceNFn := StringsBuiltins["strings.replace_n"].Fn

	// Replace first 2 occurrences
	result := replaceNFn(
		&object.String{Value: "aaa"},
		&object.String{Value: "a"},
		&object.String{Value: "b"},
		&object.Integer{Value: 2},
	)
	testStringObject(t, result, "bba")

	// Replace all with n=-1
	result = replaceNFn(
		&object.String{Value: "aaa"},
		&object.String{Value: "a"},
		&object.String{Value: "b"},
		&object.Integer{Value: -1},
	)
	testStringObject(t, result, "bbb")

	// Replace 0
	result = replaceNFn(
		&object.String{Value: "aaa"},
		&object.String{Value: "a"},
		&object.String{Value: "b"},
		&object.Integer{Value: 0},
	)
	testStringObject(t, result, "aaa")
}

func TestStringsIsNumeric(t *testing.T) {
	isNumericFn := StringsBuiltins["strings.is_numeric"].Fn

	tests := []struct {
		input    string
		expected bool
	}{
		{"12345", true},
		{"0", true},
		{"", false},
		{"12a34", false},
		{"12.34", false},
		{"-123", false},
		{"abc", false},
	}

	for _, tt := range tests {
		result := isNumericFn(&object.String{Value: tt.input})
		testBooleanObject(t, result, tt.expected)
	}
}

func TestStringsIsAlpha(t *testing.T) {
	isAlphaFn := StringsBuiltins["strings.is_alpha"].Fn

	tests := []struct {
		input    string
		expected bool
	}{
		{"Hello", true},
		{"ABC", true},
		{"abc", true},
		{"", false},
		{"Hello123", false},
		{"Hello World", false},
		{"123", false},
	}

	for _, tt := range tests {
		result := isAlphaFn(&object.String{Value: tt.input})
		testBooleanObject(t, result, tt.expected)
	}
}

func TestStringsTruncate(t *testing.T) {
	truncateFn := StringsBuiltins["strings.truncate"].Fn

	// Normal truncation
	result := truncateFn(
		&object.String{Value: "Hello World"},
		&object.Integer{Value: 8},
		&object.String{Value: "..."},
	)
	testStringObject(t, result, "Hello...")

	// String shorter than max length
	result = truncateFn(
		&object.String{Value: "Hi"},
		&object.Integer{Value: 10},
		&object.String{Value: "..."},
	)
	testStringObject(t, result, "Hi")

	// Exact length
	result = truncateFn(
		&object.String{Value: "Hello"},
		&object.Integer{Value: 5},
		&object.String{Value: "..."},
	)
	testStringObject(t, result, "Hello")

	// Max length smaller than suffix
	result = truncateFn(
		&object.String{Value: "Hello World"},
		&object.Integer{Value: 2},
		&object.String{Value: "..."},
	)
	testStringObject(t, result, "..")
}

func TestStringsTruncateErrors(t *testing.T) {
	truncateFn := StringsBuiltins["strings.truncate"].Fn

	// Negative length
	result := truncateFn(
		&object.String{Value: "Hello"},
		&object.Integer{Value: -1},
		&object.String{Value: "..."},
	)
	if !isErrorObject(result) {
		t.Error("expected error for negative length")
	}
}

func TestStringsCompare(t *testing.T) {
	compareFn := StringsBuiltins["strings.compare"].Fn

	tests := []struct {
		a, b     string
		expected int64
	}{
		{"apple", "banana", -1},
		{"banana", "apple", 1},
		{"apple", "apple", 0},
		{"", "", 0},
		{"a", "", 1},
		{"", "a", -1},
	}

	for _, tt := range tests {
		result := compareFn(&object.String{Value: tt.a}, &object.String{Value: tt.b})
		testIntegerObject(t, result, tt.expected)
	}
}

// ============================================================================
// Maps Module Tests
// ============================================================================

func TestMapsIsEmpty(t *testing.T) {
	isEmptyFn := MapsBuiltins["maps.is_empty"].Fn

	emptyMap := object.NewMap()
	result := isEmptyFn(emptyMap)
	testBooleanObject(t, result, true)

	nonEmptyMap := object.NewMap()
	nonEmptyMap.Set(&object.String{Value: "key"}, &object.Integer{Value: 1})
	result = isEmptyFn(nonEmptyMap)
	testBooleanObject(t, result, false)
}

func TestMapsHasKey(t *testing.T) {
	hasKeyFn := MapsBuiltins["maps.has_key"].Fn

	m := object.NewMap()
	m.Set(&object.String{Value: "name"}, &object.String{Value: "Alice"})

	result := hasKeyFn(m, &object.String{Value: "name"})
	testBooleanObject(t, result, true)

	result = hasKeyFn(m, &object.String{Value: "age"})
	testBooleanObject(t, result, false)
}

func TestMapsGet(t *testing.T) {
	getFn := MapsBuiltins["maps.get"].Fn

	m := object.NewMap()
	m.Set(&object.String{Value: "name"}, &object.String{Value: "Alice"})

	result := getFn(m, &object.String{Value: "name"})
	testStringObject(t, result, "Alice")
}

func TestMapsSet(t *testing.T) {
	setFn := MapsBuiltins["maps.set"].Fn

	m := object.NewMap()
	m.Mutable = true

	setFn(m, &object.String{Value: "key"}, &object.Integer{Value: 42})

	val, exists := m.Get(&object.String{Value: "key"})
	if !exists {
		t.Error("key should exist after set")
	}
	testIntegerObject(t, val, 42)
}

func TestMapsDelete(t *testing.T) {
	deleteFn := MapsBuiltins["maps.delete"].Fn

	m := object.NewMap()
	m.Mutable = true
	m.Set(&object.String{Value: "key"}, &object.Integer{Value: 42})

	deleteFn(m, &object.String{Value: "key"})

	_, exists := m.Get(&object.String{Value: "key"})
	if exists {
		t.Error("key should not exist after delete")
	}
}

func TestMapsKeys(t *testing.T) {
	keysFn := MapsBuiltins["maps.keys"].Fn

	m := object.NewMap()
	m.Set(&object.String{Value: "a"}, &object.Integer{Value: 1})
	m.Set(&object.String{Value: "b"}, &object.Integer{Value: 2})

	result := keysFn(m)
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 keys, got %d", len(arr.Elements))
	}
}

func TestMapsValues(t *testing.T) {
	valuesFn := MapsBuiltins["maps.values"].Fn

	m := object.NewMap()
	m.Set(&object.String{Value: "a"}, &object.Integer{Value: 1})
	m.Set(&object.String{Value: "b"}, &object.Integer{Value: 2})

	result := valuesFn(m)
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 values, got %d", len(arr.Elements))
	}
}

func TestMapsLenUsingBuiltinLen(t *testing.T) {
	// maps module doesn't have its own len, use the global len builtin
	lenFn := StdBuiltins["len"].Fn

	m := object.NewMap()
	result := lenFn(m)
	testIntegerObject(t, result, 0)

	m.Set(&object.String{Value: "a"}, &object.Integer{Value: 1})
	m.Set(&object.String{Value: "b"}, &object.Integer{Value: 2})

	result = lenFn(m)
	testIntegerObject(t, result, 2)
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestArgumentCountErrors(t *testing.T) {
	// Test functions that require specific argument counts
	tests := []struct {
		name string
		fn   func(...object.Object) object.Object
		args []object.Object
	}{
		{"len no args", StdBuiltins["len"].Fn, []object.Object{}},
		{"typeof no args", StdBuiltins["typeof"].Fn, []object.Object{}},
		{"int no args", StdBuiltins["int"].Fn, []object.Object{}},
		{"math.add one arg", MathBuiltins["math.add"].Fn, []object.Object{&object.Integer{Value: 1}}},
		{"math.abs no args", MathBuiltins["math.abs"].Fn, []object.Object{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.args...)
			if !isErrorObject(result) {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestTypeErrors(t *testing.T) {
	// Test functions that require specific types
	tests := []struct {
		name string
		fn   func(...object.Object) object.Object
		args []object.Object
	}{
		{"arrays.is_empty with int", ArraysBuiltins["arrays.is_empty"].Fn, []object.Object{&object.Integer{Value: 1}}},
		{"arrays.pop with string", ArraysBuiltins["arrays.pop"].Fn, []object.Object{&object.String{Value: "hello"}}},
		{"strings.upper with int", StringsBuiltins["strings.upper"].Fn, []object.Object{&object.Integer{Value: 1}}},
		{"maps.has_key with int", MapsBuiltins["maps.has_key"].Fn, []object.Object{&object.Integer{Value: 1}, &object.String{Value: "key"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.args...)
			if !isErrorObject(result) {
				t.Errorf("expected error for %s, got %T", tt.name, result)
			}
		})
	}
}

// ============================================================================
// copy() Builtin Tests
// ============================================================================

func TestCopyPrimitives(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	tests := []struct {
		name  string
		input object.Object
	}{
		{"integer", &object.Integer{Value: 42, DeclaredType: "int"}},
		{"float", &object.Float{Value: 3.14}},
		{"string", &object.String{Value: "hello"}},
		{"boolean", &object.Boolean{Value: true}},
		{"char", &object.Char{Value: 'x'}},
		{"byte", &object.Byte{Value: 255}},
		{"nil", object.NIL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := copyFn(tt.input)
			if result.Inspect() != tt.input.Inspect() {
				t.Errorf("copy(%s) = %s, want %s", tt.input.Inspect(), result.Inspect(), tt.input.Inspect())
			}
		})
	}
}

func TestCopyIntegerPreservesDeclaredType(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	original := &object.Integer{Value: 100, DeclaredType: "u64"}
	result := copyFn(original)

	copied, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("copy(Integer) returned %T, want *object.Integer", result)
	}
	if copied.DeclaredType != original.DeclaredType {
		t.Errorf("copy() did not preserve DeclaredType: got %s, want %s", copied.DeclaredType, original.DeclaredType)
	}
	if copied.Value != original.Value {
		t.Errorf("copy() changed Value: got %d, want %d", copied.Value, original.Value)
	}
}

func TestCopyArray(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	original := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
			&object.Integer{Value: 3},
		},
	}

	result := copyFn(original)
	copied, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("copy(Array) returned %T, want *object.Array", result)
	}

	// Verify length
	if len(copied.Elements) != len(original.Elements) {
		t.Fatalf("copy() returned array with wrong length: got %d, want %d", len(copied.Elements), len(original.Elements))
	}

	// Verify values match
	for i, elem := range copied.Elements {
		origInt := original.Elements[i].(*object.Integer)
		copiedInt := elem.(*object.Integer)
		if copiedInt.Value != origInt.Value {
			t.Errorf("element %d: got %d, want %d", i, copiedInt.Value, origInt.Value)
		}
	}

	// Verify it's a deep copy - modifying copied doesn't affect original
	copied.Elements[0] = &object.Integer{Value: 100}
	origInt := original.Elements[0].(*object.Integer)
	if origInt.Value != 1 {
		t.Error("copy() did not create independent copy - modifying copy affected original")
	}
}

func TestCopyMap(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	original := object.NewMap()
	original.Set(&object.String{Value: "a"}, &object.Integer{Value: 1})
	original.Set(&object.String{Value: "b"}, &object.Integer{Value: 2})

	result := copyFn(original)
	copied, ok := result.(*object.Map)
	if !ok {
		t.Fatalf("copy(Map) returned %T, want *object.Map", result)
	}

	// Verify length
	if len(copied.Pairs) != len(original.Pairs) {
		t.Fatalf("copy() returned map with wrong length: got %d, want %d", len(copied.Pairs), len(original.Pairs))
	}

	// Verify it's a deep copy - modifying copied doesn't affect original
	copied.Set(&object.String{Value: "a"}, &object.Integer{Value: 100})
	origVal, _ := original.Get(&object.String{Value: "a"})
	if origVal.(*object.Integer).Value != 1 {
		t.Error("copy() did not create independent copy - modifying copy affected original")
	}
}

func TestCopyStruct(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	original := &object.Struct{
		TypeName: "Person",
		Fields: map[string]object.Object{
			"name": &object.String{Value: "Alice"},
			"age":  &object.Integer{Value: 30},
		},
		Mutable: true,
	}

	result := copyFn(original)
	copied, ok := result.(*object.Struct)
	if !ok {
		t.Fatalf("copy(Struct) returned %T, want *object.Struct", result)
	}

	// Verify TypeName
	if copied.TypeName != original.TypeName {
		t.Errorf("copy() did not preserve TypeName: got %s, want %s", copied.TypeName, original.TypeName)
	}

	// Verify Mutable - copy() always returns mutable values
	if !copied.Mutable {
		t.Error("copy() should return mutable struct")
	}

	// Verify field count
	if len(copied.Fields) != len(original.Fields) {
		t.Fatalf("copy() returned struct with wrong field count: got %d, want %d", len(copied.Fields), len(original.Fields))
	}

	// Verify it's a deep copy - modifying copied doesn't affect original
	copied.Fields["age"] = &object.Integer{Value: 31}
	origAge := original.Fields["age"].(*object.Integer)
	if origAge.Value != 30 {
		t.Error("copy() did not create independent copy - modifying copy affected original")
	}
}

func TestCopyNestedStruct(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	// Create nested struct: Outer{inner: Inner{val: 42}}
	inner := &object.Struct{
		TypeName: "Inner",
		Fields: map[string]object.Object{
			"val": &object.Integer{Value: 42},
		},
	}
	original := &object.Struct{
		TypeName: "Outer",
		Fields: map[string]object.Object{
			"inner": inner,
		},
	}

	result := copyFn(original)
	copied, ok := result.(*object.Struct)
	if !ok {
		t.Fatalf("copy(Struct) returned %T, want *object.Struct", result)
	}

	// Verify nested struct was copied
	copiedInner, ok := copied.Fields["inner"].(*object.Struct)
	if !ok {
		t.Fatalf("copied.inner is %T, want *object.Struct", copied.Fields["inner"])
	}

	// Modify the copied nested struct
	copiedInner.Fields["val"] = &object.Integer{Value: 99}

	// Verify original nested struct is unchanged
	origInnerVal := inner.Fields["val"].(*object.Integer)
	if origInnerVal.Value != 42 {
		t.Error("copy() did not create deep copy - modifying nested copy affected original")
	}
}

func TestCopyNestedArray(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	// Create nested array: [[1, 2], [3, 4]]
	inner1 := &object.Array{Elements: []object.Object{
		&object.Integer{Value: 1},
		&object.Integer{Value: 2},
	}}
	inner2 := &object.Array{Elements: []object.Object{
		&object.Integer{Value: 3},
		&object.Integer{Value: 4},
	}}
	original := &object.Array{Elements: []object.Object{inner1, inner2}}

	result := copyFn(original)
	copied, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("copy(Array) returned %T, want *object.Array", result)
	}

	// Modify the copied nested array
	copiedInner := copied.Elements[0].(*object.Array)
	copiedInner.Elements[0] = &object.Integer{Value: 100}

	// Verify original nested array is unchanged
	origInnerVal := inner1.Elements[0].(*object.Integer)
	if origInnerVal.Value != 1 {
		t.Error("copy() did not create deep copy - modifying nested array affected original")
	}
}

func TestCopyErrors(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	// Wrong number of arguments
	result := copyFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	result = copyFn(&object.Integer{Value: 1}, &object.Integer{Value: 2})
	if !isErrorObject(result) {
		t.Error("expected error for too many arguments")
	}
}

// ============================================================================
// error() Builtin Tests
// ============================================================================

func TestErrorConstructor(t *testing.T) {
	errorFn := StdBuiltins["error"].Fn

	result := errorFn(&object.String{Value: "something went wrong"})

	// Should return an Error struct, not an object.Error
	errStruct, ok := result.(*object.Struct)
	if !ok {
		t.Fatalf("error() returned %T, want *object.Struct", result)
	}

	if errStruct.TypeName != "Error" {
		t.Errorf("error() returned struct with TypeName %q, want \"Error\"", errStruct.TypeName)
	}

	// Check message field
	msg, ok := errStruct.Fields["message"].(*object.String)
	if !ok {
		t.Fatalf("error().message is %T, want *object.String", errStruct.Fields["message"])
	}
	if msg.Value != "something went wrong" {
		t.Errorf("error().message = %q, want \"something went wrong\"", msg.Value)
	}

	// Check code field (should be empty string)
	code, ok := errStruct.Fields["code"].(*object.String)
	if !ok {
		t.Fatalf("error().code is %T, want *object.String", errStruct.Fields["code"])
	}
	if code.Value != "" {
		t.Errorf("error().code = %q, want empty string", code.Value)
	}
}

func TestErrorConstructorErrors(t *testing.T) {
	errorFn := StdBuiltins["error"].Fn

	// Wrong number of arguments - no args
	result := errorFn()
	if !isErrorObject(result) {
		t.Error("expected runtime error for no arguments")
	}

	// Wrong number of arguments - too many
	result = errorFn(&object.String{Value: "a"}, &object.String{Value: "b"})
	if !isErrorObject(result) {
		t.Error("expected runtime error for too many arguments")
	}

	// Wrong type - integer instead of string
	result = errorFn(&object.Integer{Value: 42})
	if !isErrorObject(result) {
		t.Error("expected runtime error for non-string argument")
	}

	// Wrong type - boolean instead of string
	result = errorFn(&object.Boolean{Value: true})
	if !isErrorObject(result) {
		t.Error("expected runtime error for non-string argument")
	}
}

// ============================================================================
// Time Module Tests
// ============================================================================

func TestTimeNow(t *testing.T) {
	nowFn := TimeBuiltins["time.now"].Fn
	result := nowFn()

	// time.now() returns Unix timestamp as integer
	ts, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.now() returned %T, want Integer", result)
	}
	// Should be a reasonable Unix timestamp (after year 2020)
	if ts.Value < 1577836800 { // 2020-01-01
		t.Errorf("time.now() = %d, expected reasonable Unix timestamp", ts.Value)
	}
}

func TestTimeYear(t *testing.T) {
	yearFn := TimeBuiltins["time.year"].Fn
	result := yearFn()

	year, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.year() returned %T, want Integer", result)
	}
	// Year should be at least 2024
	if year.Value < 2024 {
		t.Errorf("time.year() = %d, expected >= 2024", year.Value)
	}
}

func TestTimeMonth(t *testing.T) {
	monthFn := TimeBuiltins["time.month"].Fn
	result := monthFn()

	month, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.month() returned %T, want Integer", result)
	}
	// Month should be 1-12
	if month.Value < 1 || month.Value > 12 {
		t.Errorf("time.month() = %d, expected 1-12", month.Value)
	}
}

func TestTimeDay(t *testing.T) {
	dayFn := TimeBuiltins["time.day"].Fn
	result := dayFn()

	day, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.day() returned %T, want Integer", result)
	}
	// Day should be 1-31
	if day.Value < 1 || day.Value > 31 {
		t.Errorf("time.day() = %d, expected 1-31", day.Value)
	}
}

func TestTimeHour(t *testing.T) {
	hourFn := TimeBuiltins["time.hour"].Fn
	result := hourFn()

	hour, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.hour() returned %T, want Integer", result)
	}
	// Hour should be 0-23
	if hour.Value < 0 || hour.Value > 23 {
		t.Errorf("time.hour() = %d, expected 0-23", hour.Value)
	}
}

func TestTimeMinute(t *testing.T) {
	minuteFn := TimeBuiltins["time.minute"].Fn
	result := minuteFn()

	minute, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.minute() returned %T, want Integer", result)
	}
	// Minute should be 0-59
	if minute.Value < 0 || minute.Value > 59 {
		t.Errorf("time.minute() = %d, expected 0-59", minute.Value)
	}
}

func TestTimeSecond(t *testing.T) {
	secondFn := TimeBuiltins["time.second"].Fn
	result := secondFn()

	second, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.second() returned %T, want Integer", result)
	}
	// Second should be 0-59
	if second.Value < 0 || second.Value > 59 {
		t.Errorf("time.second() = %d, expected 0-59", second.Value)
	}
}

func TestTimeSleep(t *testing.T) {
	sleepFn := TimeBuiltins["time.sleep"].Fn
	// Just test that it doesn't error - sleep for 1ms
	result := sleepFn(&object.Integer{Value: 1})
	if result != object.NIL {
		t.Errorf("time.sleep() = %v, want nil", result)
	}
}

// ============================================================================
// Arrays Shuffle Test
// ============================================================================

func TestArraysShuffle(t *testing.T) {
	shuffleFn := ArraysBuiltins["arrays.shuffle"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
			&object.Integer{Value: 3},
			&object.Integer{Value: 4},
			&object.Integer{Value: 5},
		},
		Mutable: true,
	}

	result := shuffleFn(arr)

	shuffled, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("arrays.shuffle() returned %T, want Array", result)
	}

	// Verify length is preserved
	if len(shuffled.Elements) != 5 {
		t.Errorf("arrays.shuffle() returned %d elements, want 5", len(shuffled.Elements))
	}
}

// ============================================================================
// Arrays Slice Test
// ============================================================================

func TestArraysSlice(t *testing.T) {
	sliceFn := ArraysBuiltins["arrays.slice"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
			&object.Integer{Value: 3},
			&object.Integer{Value: 4},
			&object.Integer{Value: 5},
		},
	}

	result := sliceFn(arr, &object.Integer{Value: 1}, &object.Integer{Value: 4})

	sliced, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("arrays.slice() returned %T, want Array", result)
	}

	if len(sliced.Elements) != 3 {
		t.Errorf("arrays.slice() returned %d elements, want 3", len(sliced.Elements))
	}
}

// ============================================================================
// Time Module Tests (more coverage)
// ============================================================================

func TestTimeWeekdayFunc(t *testing.T) {
	weekdayFn := TimeBuiltins["time.weekday"].Fn

	result := weekdayFn()

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.weekday() returned %T, want Integer", result)
	}

	if intVal.Value < 0 || intVal.Value > 6 {
		t.Errorf("time.weekday() returned %d, want between 0 and 6", intVal.Value)
	}
}

func TestTimeDayOfYear(t *testing.T) {
	yeardayFn := TimeBuiltins["time.day_of_year"].Fn

	result := yeardayFn()

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.day_of_year() returned %T, want Integer", result)
	}

	if intVal.Value < 1 || intVal.Value > 366 {
		t.Errorf("time.day_of_year() returned %d, want between 1 and 366", intVal.Value)
	}
}

func TestTimeNowMs(t *testing.T) {
	nowMsFn := TimeBuiltins["time.now_ms"].Fn

	result := nowMsFn()

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.now_ms() returned %T, want Integer", result)
	}

	// Millisecond timestamp should be a large positive number
	if intVal.Value < 1000000000000 {
		t.Errorf("time.now_ms() returned %d, want a reasonable ms timestamp", intVal.Value)
	}
}

// ============================================================================
// Math Module Tests (more coverage)
// ============================================================================

func TestMathClampValues(t *testing.T) {
	clampFn := MathBuiltins["math.clamp"].Fn

	result := clampFn(&object.Integer{Value: 5}, &object.Integer{Value: 0}, &object.Integer{Value: 10})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("math.clamp() returned %T, want Integer", result)
	}

	if intVal.Value != 5 {
		t.Errorf("math.clamp(5, 0, 10) returned %d, want 5", intVal.Value)
	}

	// Test clamping below minimum
	result = clampFn(&object.Integer{Value: -5}, &object.Integer{Value: 0}, &object.Integer{Value: 10})
	intVal, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("math.clamp() returned %T, want Integer", result)
	}
	if intVal.Value != 0 {
		t.Errorf("math.clamp(-5, 0, 10) returned %d, want 0", intVal.Value)
	}

	// Test clamping above maximum
	result = clampFn(&object.Integer{Value: 15}, &object.Integer{Value: 0}, &object.Integer{Value: 10})
	intVal, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("math.clamp() returned %T, want Integer", result)
	}
	if intVal.Value != 10 {
		t.Errorf("math.clamp(15, 0, 10) returned %d, want 10", intVal.Value)
	}
}

func TestMathIsInfFunc(t *testing.T) {
	isinfFn := MathBuiltins["math.is_inf"].Fn

	// Test with positive infinity
	result := isinfFn(&object.Float{Value: math.Inf(1)})

	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_inf() returned %T, want Boolean", result)
	}

	if !boolVal.Value {
		t.Errorf("math.is_inf(+Inf) returned false, want true")
	}

	// Test with normal number
	result = isinfFn(&object.Float{Value: 3.14})
	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_inf() returned %T, want Boolean", result)
	}

	if boolVal.Value {
		t.Errorf("math.is_inf(3.14) returned true, want false")
	}
}

func TestMathIsPrime(t *testing.T) {
	isprimeFn := MathBuiltins["math.is_prime"].Fn

	// Test with prime number
	result := isprimeFn(&object.Integer{Value: 7})

	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_prime() returned %T, want Boolean", result)
	}

	if !boolVal.Value {
		t.Errorf("math.is_prime(7) returned false, want true")
	}

	// Test with non-prime
	result = isprimeFn(&object.Integer{Value: 4})
	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_prime() returned %T, want Boolean", result)
	}

	if boolVal.Value {
		t.Errorf("math.is_prime(4) returned true, want false")
	}
}

func TestMathIsEven(t *testing.T) {
	isevenFn := MathBuiltins["math.is_even"].Fn

	// Test with even number
	result := isevenFn(&object.Integer{Value: 4})

	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_even() returned %T, want Boolean", result)
	}

	if !boolVal.Value {
		t.Errorf("math.is_even(4) returned false, want true")
	}

	// Test with odd number
	result = isevenFn(&object.Integer{Value: 5})
	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_even() returned %T, want Boolean", result)
	}

	if boolVal.Value {
		t.Errorf("math.is_even(5) returned true, want false")
	}
}

func TestMathIsOdd(t *testing.T) {
	isoddFn := MathBuiltins["math.is_odd"].Fn

	// Test with odd number
	result := isoddFn(&object.Integer{Value: 5})

	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_odd() returned %T, want Boolean", result)
	}

	if !boolVal.Value {
		t.Errorf("math.is_odd(5) returned false, want true")
	}

	// Test with even number
	result = isoddFn(&object.Integer{Value: 4})
	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_odd() returned %T, want Boolean", result)
	}

	if boolVal.Value {
		t.Errorf("math.is_odd(4) returned true, want false")
	}
}

// ============================================================================
// More Strings Module Tests
// ============================================================================

func TestStringsIndex(t *testing.T) {
	indexFn := StringsBuiltins["strings.index"].Fn

	result := indexFn(&object.String{Value: "hello world"}, &object.String{Value: "world"})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("strings.index() returned %T, want Integer", result)
	}

	if intVal.Value != 6 {
		t.Errorf("strings.index('hello world', 'world') = %d, want 6", intVal.Value)
	}
}

func TestStringsIndexNotFound(t *testing.T) {
	indexFn := StringsBuiltins["strings.index"].Fn

	result := indexFn(&object.String{Value: "hello"}, &object.String{Value: "xyz"})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("strings.index() returned %T, want Integer", result)
	}

	if intVal.Value != -1 {
		t.Errorf("strings.index('hello', 'xyz') = %d, want -1", intVal.Value)
	}
}

func TestStringsPadLeft(t *testing.T) {
	padFn := StringsBuiltins["strings.pad_left"].Fn

	result := padFn(&object.String{Value: "hello"}, &object.Integer{Value: 10}, &object.String{Value: " "})

	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("strings.pad_left() returned %T, want String", result)
	}

	if strVal.Value != "     hello" {
		t.Errorf("strings.pad_left('hello', 10, ' ') = %q, want '     hello'", strVal.Value)
	}
}

func TestStringsPadRight(t *testing.T) {
	padFn := StringsBuiltins["strings.pad_right"].Fn

	result := padFn(&object.String{Value: "hello"}, &object.Integer{Value: 10}, &object.String{Value: " "})

	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("strings.pad_right() returned %T, want String", result)
	}

	if strVal.Value != "hello     " {
		t.Errorf("strings.pad_right('hello', 10, ' ') = %q, want 'hello     '", strVal.Value)
	}
}

// ============================================================================
// More Arrays Module Tests
// ============================================================================

func TestArraysClear(t *testing.T) {
	clearFn := ArraysBuiltins["arrays.clear"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
			&object.Integer{Value: 3},
		},
		Mutable: true,
	}

	result := clearFn(arr)

	cleared, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("arrays.clear() returned %T, want Array", result)
	}

	if len(cleared.Elements) != 0 {
		t.Errorf("arrays.clear() returned %d elements, want 0", len(cleared.Elements))
	}
}

func TestArraysConcat(t *testing.T) {
	concatFn := ArraysBuiltins["arrays.concat"].Fn

	arr1 := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
		},
	}
	arr2 := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 3},
			&object.Integer{Value: 4},
		},
	}

	result := concatFn(arr1, arr2)

	concatenated, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("arrays.concat() returned %T, want Array", result)
	}

	if len(concatenated.Elements) != 4 {
		t.Errorf("arrays.concat() returned %d elements, want 4", len(concatenated.Elements))
	}
}

func TestArraysFill(t *testing.T) {
	fillFn := ArraysBuiltins["arrays.fill"].Fn

	// Create an array with 5 elements
	inputArr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
			&object.Integer{Value: 3},
			&object.Integer{Value: 4},
			&object.Integer{Value: 5},
		},
	}

	result := fillFn(inputArr, &object.Integer{Value: 0})

	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("arrays.fill() returned %T, want Array", result)
	}

	if len(arr.Elements) != 5 {
		t.Errorf("arrays.fill() returned %d elements, want 5", len(arr.Elements))
	}

	// All elements should now be 0
	for i, elem := range arr.Elements {
		intVal, ok := elem.(*object.Integer)
		if !ok || intVal.Value != 0 {
			t.Errorf("arr[%d] = %v, want 0", i, elem)
		}
	}
}

func TestArraysInsert(t *testing.T) {
	insertFn := ArraysBuiltins["arrays.insert"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
			&object.Integer{Value: 3},
		},
		Mutable: true,
	}

	// Insert 99 at index 1
	result := insertFn(arr, &object.Integer{Value: 1}, &object.Integer{Value: 99})

	// Should return nil (modifies in-place)
	if result != object.NIL {
		t.Errorf("arrays.insert() returned %T, want NIL", result)
	}

	// Array should now have 4 elements
	if len(arr.Elements) != 4 {
		t.Fatalf("arrays.insert() should modify array to have 4 elements, got %d", len(arr.Elements))
	}

	// Verify the elements: {1, 99, 2, 3}
	expected := []int64{1, 99, 2, 3}
	for i, exp := range expected {
		intVal, ok := arr.Elements[i].(*object.Integer)
		if !ok {
			t.Errorf("arr[%d] is %T, want Integer", i, arr.Elements[i])
			continue
		}
		if intVal.Value != exp {
			t.Errorf("arr[%d] = %d, want %d", i, intVal.Value, exp)
		}
	}
}

func TestArraysInsertAtStart(t *testing.T) {
	insertFn := ArraysBuiltins["arrays.insert"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 2},
			&object.Integer{Value: 3},
		},
		Mutable: true,
	}

	// Insert 1 at index 0
	insertFn(arr, &object.Integer{Value: 0}, &object.Integer{Value: 1})

	// Verify the elements: {1, 2, 3}
	expected := []int64{1, 2, 3}
	for i, exp := range expected {
		intVal := arr.Elements[i].(*object.Integer)
		if intVal.Value != exp {
			t.Errorf("arr[%d] = %d, want %d", i, intVal.Value, exp)
		}
	}
}

func TestArraysInsertAtEnd(t *testing.T) {
	insertFn := ArraysBuiltins["arrays.insert"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
		},
		Mutable: true,
	}

	// Insert 3 at end (index 2)
	insertFn(arr, &object.Integer{Value: 2}, &object.Integer{Value: 3})

	// Verify the elements: {1, 2, 3}
	expected := []int64{1, 2, 3}
	for i, exp := range expected {
		intVal := arr.Elements[i].(*object.Integer)
		if intVal.Value != exp {
			t.Errorf("arr[%d] = %d, want %d", i, intVal.Value, exp)
		}
	}
}

func TestArraysInsertImmutable(t *testing.T) {
	insertFn := ArraysBuiltins["arrays.insert"].Fn

	arr := &object.Array{
		Elements: []object.Object{&object.Integer{Value: 1}},
		Mutable:  false,
	}

	result := insertFn(arr, &object.Integer{Value: 0}, &object.Integer{Value: 99})

	if !isErrorObject(result) {
		t.Error("expected error for immutable array")
	}
}

func TestArraysRemove(t *testing.T) {
	removeFn := ArraysBuiltins["arrays.remove"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
			&object.Integer{Value: 3},
			&object.Integer{Value: 2},
			&object.Integer{Value: 4},
		},
		Mutable: true,
	}

	// Remove first occurrence of 2
	result := removeFn(arr, &object.Integer{Value: 2})

	// Should return nil (modifies in-place)
	if result != object.NIL {
		t.Errorf("arrays.remove() returned %T, want NIL", result)
	}

	// Array should now have 4 elements
	if len(arr.Elements) != 4 {
		t.Fatalf("arrays.remove() should modify array to have 4 elements, got %d", len(arr.Elements))
	}

	// Verify the elements: {1, 3, 2, 4} (first 2 removed)
	expected := []int64{1, 3, 2, 4}
	for i, exp := range expected {
		intVal, ok := arr.Elements[i].(*object.Integer)
		if !ok {
			t.Errorf("arr[%d] is %T, want Integer", i, arr.Elements[i])
			continue
		}
		if intVal.Value != exp {
			t.Errorf("arr[%d] = %d, want %d", i, intVal.Value, exp)
		}
	}
}

func TestArraysRemoveNotFound(t *testing.T) {
	removeFn := ArraysBuiltins["arrays.remove"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: 1},
			&object.Integer{Value: 2},
			&object.Integer{Value: 3},
		},
		Mutable: true,
	}

	// Remove non-existent element
	result := removeFn(arr, &object.Integer{Value: 99})

	// Should return nil (no error)
	if result != object.NIL {
		t.Errorf("arrays.remove() returned %T, want NIL", result)
	}

	// Array should remain unchanged
	if len(arr.Elements) != 3 {
		t.Errorf("arrays.remove() should not change array length, got %d", len(arr.Elements))
	}
}

func TestArraysRemoveImmutable(t *testing.T) {
	removeFn := ArraysBuiltins["arrays.remove"].Fn

	arr := &object.Array{
		Elements: []object.Object{&object.Integer{Value: 1}},
		Mutable:  false,
	}

	result := removeFn(arr, &object.Integer{Value: 1})

	if !isErrorObject(result) {
		t.Error("expected error for immutable array")
	}
}

// ============================================================================
// More Math Module Tests
// ============================================================================

func TestMathGCD(t *testing.T) {
	gcdFn := MathBuiltins["math.gcd"].Fn

	result := gcdFn(&object.Integer{Value: 48}, &object.Integer{Value: 18})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("math.gcd() returned %T, want Integer", result)
	}

	if intVal.Value != 6 {
		t.Errorf("math.gcd(48, 18) = %d, want 6", intVal.Value)
	}
}

func TestMathLCM(t *testing.T) {
	lcmFn := MathBuiltins["math.lcm"].Fn

	result := lcmFn(&object.Integer{Value: 4}, &object.Integer{Value: 6})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("math.lcm() returned %T, want Integer", result)
	}

	if intVal.Value != 12 {
		t.Errorf("math.lcm(4, 6) = %d, want 12", intVal.Value)
	}
}

func TestMathFactorial(t *testing.T) {
	factFn := MathBuiltins["math.factorial"].Fn

	result := factFn(&object.Integer{Value: 5})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("math.factorial() returned %T, want Integer", result)
	}

	if intVal.Value != 120 {
		t.Errorf("math.factorial(5) = %d, want 120", intVal.Value)
	}
}

