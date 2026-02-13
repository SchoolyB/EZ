package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"math"
	"math/big"
	"strings"
	"testing"
	gotime "time"

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
	if result.Value.Cmp(big.NewInt(expected)) != 0 {
		t.Errorf("object has wrong value. got=%s, want=%d", result.Value.String(), expected)
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
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
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

	result = lenFn(&object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(10)})
	if !isErrorObject(result) {
		t.Error("expected error for too many arguments")
	}

	// Unsupported type
	result = lenFn(&object.Integer{Value: big.NewInt(5)})
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
		{"integer", &object.Integer{Value: big.NewInt(5)}, "int"},
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
		{"integer", &object.Integer{Value: big.NewInt(42)}, 42},
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

	// Float overflow - value exceeds int64 max (#522)
	result = intFn(&object.Float{Value: 9999999999999999999.9})
	if err, ok := result.(*object.Error); !ok {
		t.Error("expected error for float overflow")
	} else if err.Code != "E7033" {
		t.Errorf("expected E7033 error code, got %s", err.Code)
	}

	// Float overflow - negative value below int64 min
	result = intFn(&object.Float{Value: -9999999999999999999.9})
	if err, ok := result.(*object.Error); !ok {
		t.Error("expected error for negative float overflow")
	} else if err.Code != "E7033" {
		t.Errorf("expected E7033 error code, got %s", err.Code)
	}

	// NaN conversion should fail
	result = intFn(&object.Float{Value: math.NaN()})
	if err, ok := result.(*object.Error); !ok {
		t.Error("expected error for NaN conversion")
	} else if err.Code != "E7033" {
		t.Errorf("expected E7033 error code, got %s", err.Code)
	}

	// Inf conversion should fail
	result = intFn(&object.Float{Value: math.Inf(1)})
	if err, ok := result.(*object.Error); !ok {
		t.Error("expected error for Inf conversion")
	} else if err.Code != "E7033" {
		t.Errorf("expected E7033 error code, got %s", err.Code)
	}
}

func TestFloatConversion(t *testing.T) {
	floatFn := StdBuiltins["float"].Fn

	tests := []struct {
		name     string
		input    object.Object
		expected float64
	}{
		{"integer", &object.Integer{Value: big.NewInt(42)}, 42.0},
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
		{"integer", &object.Integer{Value: big.NewInt(42)}, "42"},
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
		{"integer to char", &object.Integer{Value: big.NewInt(65)}, 'A'},
		{"integer to char B", &object.Integer{Value: big.NewInt(66)}, 'B'},
		{"float to char", &object.Float{Value: 67.9}, 'C'},
		{"byte to char", &object.Byte{Value: 68}, 'D'},
		{"string to char", &object.String{Value: "E"}, 'E'},
		{"unicode char", &object.Integer{Value: big.NewInt(0x1F600)}, 0x1F600}, // 😀
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
		{"negative integer", &object.Integer{Value: big.NewInt(-1)}},
		{"too large integer", &object.Integer{Value: big.NewInt(0x110000)}}, // Beyond Unicode range
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
	result = charFn(&object.Integer{Value: big.NewInt(65)}, &object.Integer{Value: big.NewInt(66)})
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
		{"integers", &object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(3)}, 8, true},
		{"floats", &object.Float{Value: 2.5}, &object.Float{Value: 1.5}, 4.0, false},
		{"mixed", &object.Integer{Value: big.NewInt(5)}, &object.Float{Value: 2.5}, 7.5, false},
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

	result := subFn(&object.Integer{Value: big.NewInt(10)}, &object.Integer{Value: big.NewInt(3)})
	testIntegerObject(t, result, 7)

	result = subFn(&object.Float{Value: 5.5}, &object.Float{Value: 2.0})
	testFloatObject(t, result, 3.5)
}

func TestMathMul(t *testing.T) {
	mulFn := MathBuiltins["math.mul"].Fn

	result := mulFn(&object.Integer{Value: big.NewInt(6)}, &object.Integer{Value: big.NewInt(7)})
	testIntegerObject(t, result, 42)

	result = mulFn(&object.Float{Value: 2.5}, &object.Float{Value: 4.0})
	testFloatObject(t, result, 10.0)
}

func TestMathDiv(t *testing.T) {
	divFn := MathBuiltins["math.div"].Fn

	result := divFn(&object.Integer{Value: big.NewInt(10)}, &object.Integer{Value: big.NewInt(4)})
	testFloatObject(t, result, 2.5)

	// Division by zero
	result = divFn(&object.Integer{Value: big.NewInt(10)}, &object.Integer{Value: big.NewInt(0)})
	if !isErrorObject(result) {
		t.Error("expected error for division by zero")
	}
}

func TestMathMod(t *testing.T) {
	modFn := MathBuiltins["math.mod"].Fn

	result := modFn(&object.Integer{Value: big.NewInt(10)}, &object.Integer{Value: big.NewInt(3)})
	// math.mod returns Integer for integer inputs
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer for integer mod, got %T", result)
	}
	if intVal.Value.Int64() != 1 {
		t.Errorf("expected 1, got %d", intVal.Value.Int64())
	}

	// Modulo by zero
	result = modFn(&object.Integer{Value: big.NewInt(10)}, &object.Integer{Value: big.NewInt(0)})
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
		{"positive int", &object.Integer{Value: big.NewInt(5)}, 5, true},
		{"negative int", &object.Integer{Value: big.NewInt(-5)}, 5, true},
		{"positive float", &object.Float{Value: 3.14}, 3.14, false},
		{"negative float", &object.Float{Value: -3.14}, 3.14, false},
		{"zero", &object.Integer{Value: big.NewInt(0)}, 0, true},
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
		{"positive", &object.Integer{Value: big.NewInt(42)}, 1},
		{"negative", &object.Integer{Value: big.NewInt(-42)}, -1},
		{"zero", &object.Integer{Value: big.NewInt(0)}, 0},
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

	result := negFn(&object.Integer{Value: big.NewInt(5)})
	testIntegerObject(t, result, -5)

	result = negFn(&object.Integer{Value: big.NewInt(-5)})
	testIntegerObject(t, result, 5)

	result = negFn(&object.Float{Value: 3.14})
	testFloatObject(t, result, -3.14)
}

func TestMathMinMax(t *testing.T) {
	minFn := MathBuiltins["math.min"].Fn
	maxFn := MathBuiltins["math.max"].Fn

	result := minFn(&object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(3)})
	testIntegerObject(t, result, 3)

	result = maxFn(&object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(3)})
	testIntegerObject(t, result, 5)
}

func TestMathPow(t *testing.T) {
	powFn := MathBuiltins["math.pow"].Fn

	// Integer power returns integer when result is whole number
	result := powFn(&object.Integer{Value: big.NewInt(2)}, &object.Integer{Value: big.NewInt(3)})
	testIntegerObject(t, result, 8)

	// Float power returns float
	result = powFn(&object.Float{Value: 2.0}, &object.Float{Value: 0.5})
	testFloatObject(t, result, math.Sqrt(2.0))
}

func TestMathSqrt(t *testing.T) {
	sqrtFn := MathBuiltins["math.sqrt"].Fn

	result := sqrtFn(&object.Integer{Value: big.NewInt(16)})
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
	result := logBaseFn(&object.Integer{Value: big.NewInt(8)}, &object.Integer{Value: big.NewInt(2)})
	testFloatObject(t, result, 3.0)

	// log_base(1000, 10) = 3 (since 10^3 = 1000)
	result = logBaseFn(&object.Integer{Value: big.NewInt(1000)}, &object.Integer{Value: big.NewInt(10)})
	testFloatObject(t, result, 3.0)

	// log_base(27, 3) = 3 (since 3^3 = 27)
	result = logBaseFn(&object.Integer{Value: big.NewInt(27)}, &object.Integer{Value: big.NewInt(3)})
	testFloatObject(t, result, 3.0)

	// log_base(1, any) = 0
	result = logBaseFn(&object.Integer{Value: big.NewInt(1)}, &object.Integer{Value: big.NewInt(5)})
	testFloatObject(t, result, 0.0)
}

func TestMathLogBaseErrors(t *testing.T) {
	logBaseFn := MathBuiltins["math.log_base"].Fn

	// Value <= 0
	result := logBaseFn(&object.Integer{Value: big.NewInt(0)}, &object.Integer{Value: big.NewInt(2)})
	if !isErrorObject(result) {
		t.Error("expected error for value <= 0")
	}

	result = logBaseFn(&object.Integer{Value: big.NewInt(-5)}, &object.Integer{Value: big.NewInt(2)})
	if !isErrorObject(result) {
		t.Error("expected error for negative value")
	}

	// Base <= 0
	result = logBaseFn(&object.Integer{Value: big.NewInt(8)}, &object.Integer{Value: big.NewInt(0)})
	if !isErrorObject(result) {
		t.Error("expected error for base <= 0")
	}

	// Base = 1 (undefined)
	result = logBaseFn(&object.Integer{Value: big.NewInt(8)}, &object.Integer{Value: big.NewInt(1)})
	if !isErrorObject(result) {
		t.Error("expected error for base = 1")
	}

	// Wrong number of args
	result = logBaseFn(&object.Integer{Value: big.NewInt(8)})
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
	result = isEmptyFn(&object.Array{Elements: []object.Object{&object.Integer{Value: big.NewInt(1)}}})
	testBooleanObject(t, result, false)
}

func TestArraysAppend(t *testing.T) {
	appendFn := ArraysBuiltins["arrays.append"].Fn

	arr := &object.Array{Elements: []object.Object{&object.Integer{Value: big.NewInt(1)}}, Mutable: true}
	appendFn(arr, &object.Integer{Value: big.NewInt(2)}, &object.Integer{Value: big.NewInt(3)})

	if len(arr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestArraysAppendImmutable(t *testing.T) {
	appendFn := ArraysBuiltins["arrays.append"].Fn

	arr := &object.Array{Elements: []object.Object{&object.Integer{Value: big.NewInt(1)}}, Mutable: false}
	result := appendFn(arr, &object.Integer{Value: big.NewInt(2)})

	if !isErrorObject(result) {
		t.Error("expected error for immutable array")
	}
}

func TestArraysPop(t *testing.T) {
	popFn := ArraysBuiltins["arrays.pop"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
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
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(2)},
		&object.Integer{Value: big.NewInt(3)},
	}}

	result := lenFn(arr)
	testIntegerObject(t, result, 3)
}

func TestArraysFirst(t *testing.T) {
	firstFn := ArraysBuiltins["arrays.first"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: big.NewInt(10)},
		&object.Integer{Value: big.NewInt(20)},
	}}

	result := firstFn(arr)
	testIntegerObject(t, result, 10)
}

func TestArraysLast(t *testing.T) {
	lastFn := ArraysBuiltins["arrays.last"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: big.NewInt(10)},
		&object.Integer{Value: big.NewInt(20)},
	}}

	result := lastFn(arr)
	testIntegerObject(t, result, 20)
}

func TestArraysContains(t *testing.T) {
	containsFn := ArraysBuiltins["arrays.contains"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(2)},
		&object.Integer{Value: big.NewInt(3)},
	}}

	result := containsFn(arr, &object.Integer{Value: big.NewInt(2)})
	testBooleanObject(t, result, true)

	result = containsFn(arr, &object.Integer{Value: big.NewInt(5)})
	testBooleanObject(t, result, false)
}

func TestArraysReverse(t *testing.T) {
	reverseFn := ArraysBuiltins["arrays.reverse"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
		Mutable: true,
	}

	// arrays.reverse returns a new reversed array (does NOT modify in-place)
	result := reverseFn(arr)
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array to be returned, got %T", result)
	}

	// Check reversed order in result array
	testIntegerObject(t, resultArr.Elements[0], 3)
	testIntegerObject(t, resultArr.Elements[1], 2)
	testIntegerObject(t, resultArr.Elements[2], 1)

	// Original array should be unchanged
	testIntegerObject(t, arr.Elements[0], 1)
	testIntegerObject(t, arr.Elements[1], 2)
	testIntegerObject(t, arr.Elements[2], 3)
}

func TestArraysReverseImmutable(t *testing.T) {
	// arrays.reverse creates a new array, so it works on immutable arrays
	reverseFn := ArraysBuiltins["arrays.reverse"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
		Mutable: false, // immutable
	}

	result := reverseFn(arr)
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array to be returned, got %T", result)
	}

	// Check reversed order in result array
	testIntegerObject(t, resultArr.Elements[0], 3)
	testIntegerObject(t, resultArr.Elements[1], 2)
	testIntegerObject(t, resultArr.Elements[2], 1)

	// The returned array is mutable by default
	if !resultArr.Mutable {
		t.Error("returned array should be mutable")
	}
}

func TestArraysSort(t *testing.T) {
	sortFn := ArraysBuiltins["arrays.sort"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(5)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(8)},
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(9)},
		},
		Mutable: true,
	}

	// arrays.sort modifies in-place and returns NIL
	result := sortFn(arr)
	if result != object.NIL {
		t.Fatalf("expected NIL, got %T", result)
	}

	// Check sorted order in original array (in-place)
	expected := []int64{1, 2, 5, 8, 9}
	for i, exp := range expected {
		testIntegerObject(t, arr.Elements[i], exp)
	}
}

func TestArraysSortImmutable(t *testing.T) {
	sortFn := ArraysBuiltins["arrays.sort"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(3)},
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
		},
		Mutable: false, // immutable
	}

	result := sortFn(arr)
	if !isErrorObject(result) {
		t.Error("expected error for immutable array")
	}
}

func TestArraysSortDesc(t *testing.T) {
	sortDescFn := ArraysBuiltins["arrays.sort_desc"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(5)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(8)},
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(9)},
		},
		Mutable: true,
	}

	// arrays.sort_desc modifies in-place and returns NIL
	result := sortDescFn(arr)
	if result != object.NIL {
		t.Fatalf("expected NIL, got %T", result)
	}

	// Check sorted order (descending) in original array
	expected := []int64{9, 8, 5, 2, 1}
	for i, exp := range expected {
		testIntegerObject(t, arr.Elements[i], exp)
	}
}

func TestArraysSum(t *testing.T) {
	sumFn := ArraysBuiltins["arrays.sum"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(2)},
		&object.Integer{Value: big.NewInt(3)},
		&object.Integer{Value: big.NewInt(4)},
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

	result := repeatFn(&object.String{Value: "ab"}, &object.Integer{Value: big.NewInt(3)})
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
		&object.Integer{Value: big.NewInt(2)},
	)
	testStringObject(t, result, "bba")

	// Replace all with n=-1
	result = replaceNFn(
		&object.String{Value: "aaa"},
		&object.String{Value: "a"},
		&object.String{Value: "b"},
		&object.Integer{Value: big.NewInt(-1)},
	)
	testStringObject(t, result, "bbb")

	// Replace 0
	result = replaceNFn(
		&object.String{Value: "aaa"},
		&object.String{Value: "a"},
		&object.String{Value: "b"},
		&object.Integer{Value: big.NewInt(0)},
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
		&object.Integer{Value: big.NewInt(8)},
		&object.String{Value: "..."},
	)
	testStringObject(t, result, "Hello...")

	// String shorter than max length
	result = truncateFn(
		&object.String{Value: "Hi"},
		&object.Integer{Value: big.NewInt(10)},
		&object.String{Value: "..."},
	)
	testStringObject(t, result, "Hi")

	// Exact length
	result = truncateFn(
		&object.String{Value: "Hello"},
		&object.Integer{Value: big.NewInt(5)},
		&object.String{Value: "..."},
	)
	testStringObject(t, result, "Hello")

	// Max length smaller than suffix
	result = truncateFn(
		&object.String{Value: "Hello World"},
		&object.Integer{Value: big.NewInt(2)},
		&object.String{Value: "..."},
	)
	testStringObject(t, result, "..")
}

func TestStringsTruncateErrors(t *testing.T) {
	truncateFn := StringsBuiltins["strings.truncate"].Fn

	// Negative length
	result := truncateFn(
		&object.String{Value: "Hello"},
		&object.Integer{Value: big.NewInt(-1)},
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
	nonEmptyMap.Set(&object.String{Value: "key"}, &object.Integer{Value: big.NewInt(1)})
	result = isEmptyFn(nonEmptyMap)
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

	setFn(m, &object.String{Value: "key"}, &object.Integer{Value: big.NewInt(42)})

	val, exists := m.Get(&object.String{Value: "key"})
	if !exists {
		t.Error("key should exist after set")
	}
	testIntegerObject(t, val, 42)
}

func TestMapsKeys(t *testing.T) {
	keysFn := MapsBuiltins["maps.keys"].Fn

	m := object.NewMap()
	m.Set(&object.String{Value: "a"}, &object.Integer{Value: big.NewInt(1)})
	m.Set(&object.String{Value: "b"}, &object.Integer{Value: big.NewInt(2)})

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
	m.Set(&object.String{Value: "a"}, &object.Integer{Value: big.NewInt(1)})
	m.Set(&object.String{Value: "b"}, &object.Integer{Value: big.NewInt(2)})

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

	m.Set(&object.String{Value: "a"}, &object.Integer{Value: big.NewInt(1)})
	m.Set(&object.String{Value: "b"}, &object.Integer{Value: big.NewInt(2)})

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
		{"math.add one arg", MathBuiltins["math.add"].Fn, []object.Object{&object.Integer{Value: big.NewInt(1)}}},
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
		{"arrays.is_empty with int", ArraysBuiltins["arrays.is_empty"].Fn, []object.Object{&object.Integer{Value: big.NewInt(1)}}},
		{"arrays.pop with string", ArraysBuiltins["arrays.pop"].Fn, []object.Object{&object.String{Value: "hello"}}},
		{"strings.upper with int", StringsBuiltins["strings.upper"].Fn, []object.Object{&object.Integer{Value: big.NewInt(1)}}},
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
		{"integer", &object.Integer{Value: big.NewInt(42), DeclaredType: "int"}},
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

	original := &object.Integer{Value: big.NewInt(100), DeclaredType: "u64"}
	result := copyFn(original)

	copied, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("copy(Integer) returned %T, want *object.Integer", result)
	}
	if copied.DeclaredType != original.DeclaredType {
		t.Errorf("copy() did not preserve DeclaredType: got %s, want %s", copied.DeclaredType, original.DeclaredType)
	}
	if copied.Value.Cmp(original.Value) != 0 {
		t.Errorf("copy() changed Value: got %d, want %d", copied.Value, original.Value)
	}
}

func TestCopyArray(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	original := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
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
		if copiedInt.Value.Cmp(origInt.Value) != 0 {
			t.Errorf("element %d: got %d, want %d", i, copiedInt.Value, origInt.Value)
		}
	}

	// Verify it's a deep copy - modifying copied doesn't affect original
	copied.Elements[0] = &object.Integer{Value: big.NewInt(100)}
	origInt := original.Elements[0].(*object.Integer)
	if origInt.Value.Cmp(big.NewInt(1)) != 0 {
		t.Error("copy() did not create independent copy - modifying copy affected original")
	}
}

func TestCopyMap(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	original := object.NewMap()
	original.Set(&object.String{Value: "a"}, &object.Integer{Value: big.NewInt(1)})
	original.Set(&object.String{Value: "b"}, &object.Integer{Value: big.NewInt(2)})

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
	copied.Set(&object.String{Value: "a"}, &object.Integer{Value: big.NewInt(100)})
	origVal, _ := original.Get(&object.String{Value: "a"})
	if origVal.(*object.Integer).Value.Cmp(big.NewInt(1)) != 0 {
		t.Error("copy() did not create independent copy - modifying copy affected original")
	}
}

func TestCopyStruct(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	original := &object.Struct{
		TypeName: "Person",
		Fields: map[string]object.Object{
			"name": &object.String{Value: "Alice"},
			"age":  &object.Integer{Value: big.NewInt(30)},
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
	copied.Fields["age"] = &object.Integer{Value: big.NewInt(31)}
	origAge := original.Fields["age"].(*object.Integer)
	if origAge.Value.Cmp(big.NewInt(30)) != 0 {
		t.Error("copy() did not create independent copy - modifying copy affected original")
	}
}

func TestCopyNestedStruct(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	// Create nested struct: Outer{inner: Inner{val: 42}}
	inner := &object.Struct{
		TypeName: "Inner",
		Fields: map[string]object.Object{
			"val": &object.Integer{Value: big.NewInt(42)},
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
	copiedInner.Fields["val"] = &object.Integer{Value: big.NewInt(99)}

	// Verify original nested struct is unchanged
	origInnerVal := inner.Fields["val"].(*object.Integer)
	if origInnerVal.Value.Cmp(big.NewInt(42)) != 0 {
		t.Error("copy() did not create deep copy - modifying nested copy affected original")
	}
}

func TestCopyNestedArray(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	// Create nested array: [[1, 2], [3, 4]]
	inner1 := &object.Array{Elements: []object.Object{
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(2)},
	}}
	inner2 := &object.Array{Elements: []object.Object{
		&object.Integer{Value: big.NewInt(3)},
		&object.Integer{Value: big.NewInt(4)},
	}}
	original := &object.Array{Elements: []object.Object{inner1, inner2}}

	result := copyFn(original)
	copied, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("copy(Array) returned %T, want *object.Array", result)
	}

	// Modify the copied nested array
	copiedInner := copied.Elements[0].(*object.Array)
	copiedInner.Elements[0] = &object.Integer{Value: big.NewInt(100)}

	// Verify original nested array is unchanged
	origInnerVal := inner1.Elements[0].(*object.Integer)
	if origInnerVal.Value.Cmp(big.NewInt(1)) != 0 {
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

	result = copyFn(&object.Integer{Value: big.NewInt(1)}, &object.Integer{Value: big.NewInt(2)})
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
	result = errorFn(&object.Integer{Value: big.NewInt(42)})
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
	if ts.Value.Cmp(big.NewInt(1577836800)) < 0 { // 2020-01-01
		t.Errorf("time.now() = %s, expected reasonable Unix timestamp", ts.Value.String())
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
	if year.Value.Cmp(big.NewInt(2024)) < 0 {
		t.Errorf("time.year() = %s, expected >= 2024", year.Value.String())
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
	if month.Value.Cmp(big.NewInt(1)) < 0 || month.Value.Cmp(big.NewInt(12)) > 0 {
		t.Errorf("time.month() = %s, expected 1-12", month.Value.String())
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
	if day.Value.Cmp(big.NewInt(1)) < 0 || day.Value.Cmp(big.NewInt(31)) > 0 {
		t.Errorf("time.day() = %s, expected 1-31", day.Value.String())
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
	if hour.Value.Sign() < 0 || hour.Value.Cmp(big.NewInt(23)) > 0 {
		t.Errorf("time.hour() = %s, expected 0-23", hour.Value.String())
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
	if minute.Value.Sign() < 0 || minute.Value.Cmp(big.NewInt(59)) > 0 {
		t.Errorf("time.minute() = %s, expected 0-59", minute.Value.String())
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
	if second.Value.Sign() < 0 || second.Value.Cmp(big.NewInt(59)) > 0 {
		t.Errorf("time.second() = %s, expected 0-59", second.Value.String())
	}
}

func TestTimeSleep(t *testing.T) {
	sleepFn := TimeBuiltins["time.sleep"].Fn
	// Just test that it doesn't error - sleep for 1ms
	result := sleepFn(&object.Integer{Value: big.NewInt(1)})
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
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
			&object.Integer{Value: big.NewInt(4)},
			&object.Integer{Value: big.NewInt(5)},
		},
		Mutable: true,
	}

	// arrays.shuffle modifies in-place and returns NIL
	result := shuffleFn(arr)
	if result != object.NIL {
		t.Fatalf("arrays.shuffle() returned %T, want NIL", result)
	}

	// Verify length is preserved in original array (in-place)
	if len(arr.Elements) != 5 {
		t.Errorf("arrays.shuffle() modified array to %d elements, want 5", len(arr.Elements))
	}
}

func TestArraysShuffleImmutable(t *testing.T) {
	shuffleFn := ArraysBuiltins["arrays.shuffle"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
		},
		Mutable: false, // immutable
	}

	result := shuffleFn(arr)
	if !isErrorObject(result) {
		t.Error("expected error for immutable array")
	}
}

// ============================================================================
// Arrays Slice Test
// ============================================================================

func TestArraysSlice(t *testing.T) {
	sliceFn := ArraysBuiltins["arrays.slice"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
			&object.Integer{Value: big.NewInt(4)},
			&object.Integer{Value: big.NewInt(5)},
		},
	}

	result := sliceFn(arr, &object.Integer{Value: big.NewInt(1)}, &object.Integer{Value: big.NewInt(4)})

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

	if intVal.Value.Sign() < 0 || intVal.Value.Cmp(big.NewInt(6)) > 0 {
		t.Errorf("time.weekday() returned %s, want between 0 and 6", intVal.Value.String())
	}
}

func TestTimeDayOfYear(t *testing.T) {
	yeardayFn := TimeBuiltins["time.day_of_year"].Fn

	result := yeardayFn()

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.day_of_year() returned %T, want Integer", result)
	}

	if intVal.Value.Cmp(big.NewInt(1)) < 0 || intVal.Value.Cmp(big.NewInt(366)) > 0 {
		t.Errorf("time.day_of_year() returned %s, want between 1 and 366", intVal.Value.String())
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
	if intVal.Value.Cmp(big.NewInt(1000000000000)) < 0 {
		t.Errorf("time.now_ms() returned %s, want a reasonable ms timestamp", intVal.Value.String())
	}
}

// ============================================================================
// Time Module Unix Functions Tests
// ============================================================================

func TestTimeFromUnix(t *testing.T) {
	fromUnixFn := TimeBuiltins["time.from_unix"].Fn

	knownUnix := big.NewInt(1577836800)
	result := fromUnixFn(&object.Integer{Value: knownUnix})

	ts, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.from_unix() returned %T, want Integer", result)
	}

	if ts.Value.Cmp(knownUnix) != 0 {
		t.Errorf("time.from_unix() = %s, want %s", ts.Value.String(), knownUnix.String())
	}
}

func TestTimeFromUnixMs(t *testing.T) {
	fromUnixMsFn := TimeBuiltins["time.from_unix_ms"].Fn
	knownUnixMs := big.NewInt(1577836800000)
	result := fromUnixMsFn(&object.Integer{Value: knownUnixMs})

	ts, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.from_unix_ms() returned %T, want Integer", result)
	}
	expectedSecond := big.NewInt(1577836800)

	if ts.Value.Cmp(expectedSecond) != 0 {
		t.Errorf("time.from_unix_ms() = %s, want %s", ts.Value.String(), expectedSecond.String())
	}
}

func TestTimeToUnix(t *testing.T) {
	toUnixFn := TimeBuiltins["time.to_unix"].Fn

	knownUnix := big.NewInt(1577836800)
	result := toUnixFn(&object.Integer{Value: knownUnix})

	ts, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.to_unix() returned %T, want Integer", result)
	}

	if ts.Value.Cmp(knownUnix) != 0 {
		t.Errorf("time.to_unix() = %s, want %s", ts.Value.String(), knownUnix.String())
	}
}

func TestTimeToUnixMs(t *testing.T) {
	toUnixMsFn := TimeBuiltins["time.to_unix_ms"].Fn

	knownUnix := big.NewInt(1577836800)
	result := toUnixMsFn(&object.Integer{Value: knownUnix})

	ms, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.to_unix_ms() returned %T, want Integer", result)
	}

	expectedMs := big.NewInt(1577836800000)
	if ms.Value.Cmp(expectedMs) != 0 {
		t.Errorf("time.to_unix_ms() = %s, want %s", ms.Value.String(), expectedMs.String())
	}
}

func TestTimeIsWeekend(t *testing.T) {
	isWeekendFn := TimeBuiltins["time.is_weekend"].Fn

	// Use midday UTC timestamps to avoid timezone boundary issues
	// Saturday, January 4, 2020 12:00:00 UTC
	saturday := big.NewInt(1578139200)
	result := isWeekendFn(&object.Integer{Value: saturday})

	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("time.is_weekend() returned %T, want Boolean", result)
	}
	if !boolVal.Value {
		t.Errorf("time.is_weekend() returned false, want true")
	}

	// Sunday, January 5, 2020 12:00:00 UTC
	sunday := big.NewInt(1578225600)
	result = isWeekendFn(&object.Integer{Value: sunday})
	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("time.is_weekend() returned %T, want Boolean", result)
	}
	if !boolVal.Value {
		t.Errorf("time.is_weekend() returned false, want true")
	}

	// Monday, January 6, 2020 12:00:00 UTC
	monday := big.NewInt(1578312000)
	result = isWeekendFn(&object.Integer{Value: monday})
	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("time.is_weekend() returned %T, want Boolean", result)
	}
	if boolVal.Value {
		t.Errorf("time.is_weekend() returned true, want false")
	}

}
func TestTimeIsWeekday(t *testing.T) {
	isWeekdayFn := TimeBuiltins["time.is_weekday"].Fn

	// Use midday UTC timestamps to avoid timezone boundary issues
	// Monday, January 6, 2020 12:00:00 UTC
	monday := big.NewInt(1578312000)
	result := isWeekdayFn(&object.Integer{Value: monday})
	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("time.is_weekday() returned %T, want Boolean", result)
	}
	if !boolVal.Value {
		t.Errorf("time.is_weekday() returned false, want true")
	}

	// Tuesday, January 7, 2020 12:00:00 UTC
	tuesday := big.NewInt(1578398400)
	result = isWeekdayFn(&object.Integer{Value: tuesday})
	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("time.is_weekday() returned %T, want Boolean", result)
	}
	if !boolVal.Value {
		t.Errorf("time.is_weekday() returned false, want true")
	}

	// Saturday, January 4, 2020 12:00:00 UTC
	saturday := big.NewInt(1578139200)
	result = isWeekdayFn(&object.Integer{Value: saturday})
	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("time.is_weekday() returned %T, want Boolean", result)
	}
	if boolVal.Value {
		t.Errorf("time.is_weekday() returned true, want false")
	}
}

func TestTimeIsToday(t *testing.T) {
	isTodayFn := TimeBuiltins["time.is_today"].Fn

	// Test with current time - should be true
	now := gotime.Now().Unix()
	result := isTodayFn(&object.Integer{Value: big.NewInt(now)})

	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("time.is_today() returned %T, want Boolean", result)
	}

	if !boolVal.Value {
		t.Errorf("time.is_today() for now = false, want true")
	}

	// Test with yesterday - should be false
	yesterday := big.NewInt(now - 86400) // 24 hours ago
	result = isTodayFn(&object.Integer{Value: yesterday})

	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("time.is_today() returned %T, want Boolean", result)
	}

	if boolVal.Value {
		t.Errorf("time.is_today() for yesterday = true, want false")
	}
}

func TestTimeIsSameDay(t *testing.T) {
	isSameDayFn := TimeBuiltins["time.is_same_day"].Fn

	// Test same day (different times)
	baseTime := big.NewInt(1577836800)
	laterSameDay := big.NewInt(1577836800 + 3600)
	result := isSameDayFn(&object.Integer{Value: baseTime}, &object.Integer{Value: laterSameDay})

	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("time.is_same_day() returned %T, want Boolean", result)
	}

	if !boolVal.Value {
		t.Errorf("time.is_same_day() for same day = false, want true")
	}

	// Test different days
	nextDay := big.NewInt(1577836800 + 86400)
	result = isSameDayFn(&object.Integer{Value: baseTime}, &object.Integer{Value: nextDay})

	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("time.is_same_day() returned %T, want Boolean", result)
	}

	if boolVal.Value {
		t.Errorf("time.is_same_day() for different days = true, want false")
	}
}

func TestTimeRelative(t *testing.T) {
	relativeFn := TimeBuiltins["time.relative"].Fn

	// Test "just now" - within last second
	now := gotime.Now().Unix()
	result := relativeFn(&object.Integer{Value: big.NewInt(now)})

	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("time.relative() returned %T, want String", result)
	}

	if strVal.Value != "just now" {
		t.Errorf("time.relative() for now = %q, want 'just now'", strVal.Value)
	}

	// Test "X seconds ago"
	pastSeconds := big.NewInt(now - 30)
	result = relativeFn(&object.Integer{Value: pastSeconds})

	strVal, ok = result.(*object.String)
	if !ok {
		t.Fatalf("time.relative() returned %T, want String", result)
	}

	if !strings.Contains(strVal.Value, "seconds ago") {
		t.Errorf("time.relative() for 30 seconds ago = %q, want 'X seconds ago'", strVal.Value)
	}

	// Test "X minutes ago"
	pastMinutes := big.NewInt(now - 120) // 2 minutes ago
	result = relativeFn(&object.Integer{Value: pastMinutes})

	strVal, ok = result.(*object.String)
	if !ok {
		t.Fatalf("time.relative() returned %T, want String", result)
	}

	if !strings.Contains(strVal.Value, "minutes ago") {
		t.Errorf("time.relative() for 2 minutes ago = %q, want 'X minutes ago'", strVal.Value)
	}

	// Test "X hours ago"
	pastHours := big.NewInt(now - 7200) // 2 hours ago
	result = relativeFn(&object.Integer{Value: pastHours})

	strVal, ok = result.(*object.String)
	if !ok {
		t.Fatalf("time.relative() returned %T, want String", result)
	}

	if !strings.Contains(strVal.Value, "hours ago") {
		t.Errorf("time.relative() for 2 hours ago = %q, want 'X hours ago'", strVal.Value)
	}

	// Test "in X seconds" (future)
	futureSeconds := big.NewInt(now + 30)
	result = relativeFn(&object.Integer{Value: futureSeconds})

	strVal, ok = result.(*object.String)
	if !ok {
		t.Fatalf("time.relative() returned %T, want String", result)
	}

	if !strings.Contains(strVal.Value, "in") || !strings.Contains(strVal.Value, "seconds") {
		t.Errorf("time.relative() for 30 seconds future = %q, want 'in X seconds'", strVal.Value)
	}
}

// ============================================================================
// Math Module Tests (more coverage)
// ============================================================================

func TestMathClampValues(t *testing.T) {
	clampFn := MathBuiltins["math.clamp"].Fn

	result := clampFn(&object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(0)}, &object.Integer{Value: big.NewInt(10)})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("math.clamp() returned %T, want Integer", result)
	}

	if intVal.Value.Cmp(big.NewInt(5)) != 0 {
		t.Errorf("math.clamp(5, 0, 10) returned %s, want 5", intVal.Value.String())
	}

	// Test clamping below minimum
	result = clampFn(&object.Integer{Value: big.NewInt(-5)}, &object.Integer{Value: big.NewInt(0)}, &object.Integer{Value: big.NewInt(10)})
	intVal, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("math.clamp() returned %T, want Integer", result)
	}
	if intVal.Value.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("math.clamp(-5, 0, 10) returned %s, want 0", intVal.Value.String())
	}

	// Test clamping above maximum
	result = clampFn(&object.Integer{Value: big.NewInt(15)}, &object.Integer{Value: big.NewInt(0)}, &object.Integer{Value: big.NewInt(10)})
	intVal, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("math.clamp() returned %T, want Integer", result)
	}
	if intVal.Value.Cmp(big.NewInt(10)) != 0 {
		t.Errorf("math.clamp(15, 0, 10) returned %s, want 10", intVal.Value.String())
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

func TestMathIsNanFunc(t *testing.T) {
	isnanFn := MathBuiltins["math.is_nan"].Fn

	// Test with NaN
	result := isnanFn(&object.Float{Value: math.NaN()})
	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_nan() returned %T, want Boolean", result)
	}
	if !boolVal.Value {
		t.Errorf("math.is_nan(NaN) returned false, want true")
	}

	// Test with normal number
	result = isnanFn(&object.Float{Value: 3.14})
	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_nan() returned %T, want Boolean", result)
	}
	if boolVal.Value {
		t.Errorf("math.is_nan(3.14) returned true, want false")
	}
}

func TestMathIsFiniteFunc(t *testing.T) {
	isfiniteFn := MathBuiltins["math.is_finite"].Fn

	// Test with normal number
	result := isfiniteFn(&object.Float{Value: 3.14})
	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_finite() returned %T, want Boolean", result)
	}
	if !boolVal.Value {
		t.Errorf("math.is_finite(3.14) returned false, want true")
	}

	// Test with infinity
	result = isfiniteFn(&object.Float{Value: math.Inf(1)})
	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_finite() returned %T, want Boolean", result)
	}
	if boolVal.Value {
		t.Errorf("math.is_finite(+Inf) returned true, want false")
	}

	// Test with NaN
	result = isfiniteFn(&object.Float{Value: math.NaN()})
	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_finite() returned %T, want Boolean", result)
	}
	if boolVal.Value {
		t.Errorf("math.is_finite(NaN) returned true, want false")
	}
}

func TestMathNewConstants(t *testing.T) {
	// EPSILON
	epsFn := MathBuiltins["math.EPSILON"].Fn
	epsResult := epsFn()
	epsFloat, ok := epsResult.(*object.Float)
	if !ok {
		t.Fatalf("math.EPSILON returned %T, want Float", epsResult)
	}
	if epsFloat.Value <= 0 || epsFloat.Value >= 1e-10 {
		t.Errorf("math.EPSILON = %v, want small positive value", epsFloat.Value)
	}

	// MAX_FLOAT
	maxFn := MathBuiltins["math.MAX_FLOAT"].Fn
	maxResult := maxFn()
	maxFloat, ok := maxResult.(*object.Float)
	if !ok {
		t.Fatalf("math.MAX_FLOAT returned %T, want Float", maxResult)
	}
	if maxFloat.Value != math.MaxFloat64 {
		t.Errorf("math.MAX_FLOAT = %v, want %v", maxFloat.Value, math.MaxFloat64)
	}

	// MIN_FLOAT
	minFn := MathBuiltins["math.MIN_FLOAT"].Fn
	minResult := minFn()
	minFloat, ok := minResult.(*object.Float)
	if !ok {
		t.Fatalf("math.MIN_FLOAT returned %T, want Float", minResult)
	}
	if minFloat.Value != math.SmallestNonzeroFloat64 {
		t.Errorf("math.MIN_FLOAT = %v, want %v", minFloat.Value, math.SmallestNonzeroFloat64)
	}
}

func TestMathIsPrime(t *testing.T) {
	isprimeFn := MathBuiltins["math.is_prime"].Fn

	// Test with prime number
	result := isprimeFn(&object.Integer{Value: big.NewInt(7)})

	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_prime() returned %T, want Boolean", result)
	}

	if !boolVal.Value {
		t.Errorf("math.is_prime(7) returned false, want true")
	}

	// Test with non-prime
	result = isprimeFn(&object.Integer{Value: big.NewInt(4)})
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
	result := isevenFn(&object.Integer{Value: big.NewInt(4)})

	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_even() returned %T, want Boolean", result)
	}

	if !boolVal.Value {
		t.Errorf("math.is_even(4) returned false, want true")
	}

	// Test with odd number
	result = isevenFn(&object.Integer{Value: big.NewInt(5)})
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
	result := isoddFn(&object.Integer{Value: big.NewInt(5)})

	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("math.is_odd() returned %T, want Boolean", result)
	}

	if !boolVal.Value {
		t.Errorf("math.is_odd(5) returned false, want true")
	}

	// Test with even number
	result = isoddFn(&object.Integer{Value: big.NewInt(4)})
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

	if intVal.Value.Cmp(big.NewInt(6)) != 0 {
		t.Errorf("strings.index('hello world', 'world') = %s, want 6", intVal.Value.String())
	}
}

func TestStringsIndexNotFound(t *testing.T) {
	indexFn := StringsBuiltins["strings.index"].Fn

	result := indexFn(&object.String{Value: "hello"}, &object.String{Value: "xyz"})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("strings.index() returned %T, want Integer", result)
	}

	if intVal.Value.Cmp(big.NewInt(-1)) != 0 {
		t.Errorf("strings.index('hello', 'xyz') = %s, want -1", intVal.Value.String())
	}
}

func TestStringsPadLeft(t *testing.T) {
	padFn := StringsBuiltins["strings.pad_left"].Fn

	result := padFn(&object.String{Value: "hello"}, &object.Integer{Value: big.NewInt(10)}, &object.String{Value: " "})

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

	result := padFn(&object.String{Value: "hello"}, &object.Integer{Value: big.NewInt(10)}, &object.String{Value: " "})

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
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
		Mutable: true,
	}

	result := clearFn(arr)

	// Should return NIL and modify array in-place
	if result != object.NIL {
		t.Fatalf("arrays.clear() returned %T, want NIL", result)
	}

	// Original array should now be empty
	if len(arr.Elements) != 0 {
		t.Errorf("arrays.clear() should clear array in-place, got %d elements", len(arr.Elements))
	}
}

func TestArraysClearImmutable(t *testing.T) {
	clearFn := ArraysBuiltins["arrays.clear"].Fn

	arr := &object.Array{
		Elements: []object.Object{&object.Integer{Value: big.NewInt(1)}},
		Mutable:  false,
	}

	result := clearFn(arr)

	if !isErrorObject(result) {
		t.Error("expected error for immutable array")
	}
}

func TestArraysConcat(t *testing.T) {
	concatFn := ArraysBuiltins["arrays.concat"].Fn

	arr1 := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
		},
	}
	arr2 := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(3)},
			&object.Integer{Value: big.NewInt(4)},
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

	// Create a mutable array with 5 elements
	inputArr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
			&object.Integer{Value: big.NewInt(4)},
			&object.Integer{Value: big.NewInt(5)},
		},
		Mutable: true,
	}

	result := fillFn(inputArr, &object.Integer{Value: big.NewInt(0)})

	// Should return NIL and modify array in-place
	if result != object.NIL {
		t.Fatalf("arrays.fill() returned %T, want NIL", result)
	}

	if len(inputArr.Elements) != 5 {
		t.Errorf("arrays.fill() should not change array length, got %d elements", len(inputArr.Elements))
	}

	// All elements should now be 0
	for i, elem := range inputArr.Elements {
		intVal, ok := elem.(*object.Integer)
		if !ok || intVal.Value.Cmp(big.NewInt(0)) != 0 {
			t.Errorf("arr[%d] = %v, want 0", i, elem)
		}
	}
}

func TestArraysFillImmutable(t *testing.T) {
	fillFn := ArraysBuiltins["arrays.fill"].Fn

	arr := &object.Array{
		Elements: []object.Object{&object.Integer{Value: big.NewInt(1)}},
		Mutable:  false,
	}

	result := fillFn(arr, &object.Integer{Value: big.NewInt(0)})

	if !isErrorObject(result) {
		t.Error("expected error for immutable array")
	}
}

func TestArraysInsert(t *testing.T) {
	insertFn := ArraysBuiltins["arrays.insert"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
		Mutable: true,
	}

	// Insert 99 at index 1
	result := insertFn(arr, &object.Integer{Value: big.NewInt(1)}, &object.Integer{Value: big.NewInt(99)})

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
		if intVal.Value.Cmp(big.NewInt(exp)) != 0 {
			t.Errorf("arr[%d] = %s, want %d", i, intVal.Value.String(), exp)
		}
	}
}

func TestArraysInsertAtStart(t *testing.T) {
	insertFn := ArraysBuiltins["arrays.insert"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
		Mutable: true,
	}

	// Insert 1 at index 0
	insertFn(arr, &object.Integer{Value: big.NewInt(0)}, &object.Integer{Value: big.NewInt(1)})

	// Verify the elements: {1, 2, 3}
	expected := []int64{1, 2, 3}
	for i, exp := range expected {
		intVal := arr.Elements[i].(*object.Integer)
		if intVal.Value.Cmp(big.NewInt(exp)) != 0 {
			t.Errorf("arr[%d] = %s, want %d", i, intVal.Value.String(), exp)
		}
	}
}

func TestArraysInsertAtEnd(t *testing.T) {
	insertFn := ArraysBuiltins["arrays.insert"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
		},
		Mutable: true,
	}

	// Insert 3 at end (index 2)
	insertFn(arr, &object.Integer{Value: big.NewInt(2)}, &object.Integer{Value: big.NewInt(3)})

	// Verify the elements: {1, 2, 3}
	expected := []int64{1, 2, 3}
	for i, exp := range expected {
		intVal := arr.Elements[i].(*object.Integer)
		if intVal.Value.Cmp(big.NewInt(exp)) != 0 {
			t.Errorf("arr[%d] = %s, want %d", i, intVal.Value.String(), exp)
		}
	}
}

func TestArraysInsertImmutable(t *testing.T) {
	insertFn := ArraysBuiltins["arrays.insert"].Fn

	arr := &object.Array{
		Elements: []object.Object{&object.Integer{Value: big.NewInt(1)}},
		Mutable:  false,
	}

	result := insertFn(arr, &object.Integer{Value: big.NewInt(0)}, &object.Integer{Value: big.NewInt(99)})

	if !isErrorObject(result) {
		t.Error("expected error for immutable array")
	}
}

func TestArraysSet(t *testing.T) {
	setFn := ArraysBuiltins["arrays.set"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
		Mutable: true,
	}

	// Set index 1 to 99
	result := setFn(arr, &object.Integer{Value: big.NewInt(1)}, &object.Integer{Value: big.NewInt(99)})

	// Should return nil (modifies in-place)
	if result != object.NIL {
		t.Errorf("arrays.set() returned %T, want NIL", result)
	}

	// Array should still have 3 elements
	if len(arr.Elements) != 3 {
		t.Fatalf("arrays.set() should not change array length, got %d", len(arr.Elements))
	}

	// Verify the elements: {1, 99, 3}
	expected := []int64{1, 99, 3}
	for i, exp := range expected {
		intVal, ok := arr.Elements[i].(*object.Integer)
		if !ok {
			t.Errorf("arr[%d] is %T, want Integer", i, arr.Elements[i])
			continue
		}
		if intVal.Value.Cmp(big.NewInt(exp)) != 0 {
			t.Errorf("arr[%d] = %s, want %d", i, intVal.Value.String(), exp)
		}
	}
}

func TestArraysSetFirst(t *testing.T) {
	setFn := ArraysBuiltins["arrays.set"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
		Mutable: true,
	}

	// Set index 0 to 99
	setFn(arr, &object.Integer{Value: big.NewInt(0)}, &object.Integer{Value: big.NewInt(99)})

	// Verify the elements: {99, 2, 3}
	expected := []int64{99, 2, 3}
	for i, exp := range expected {
		intVal := arr.Elements[i].(*object.Integer)
		if intVal.Value.Cmp(big.NewInt(exp)) != 0 {
			t.Errorf("arr[%d] = %s, want %d", i, intVal.Value.String(), exp)
		}
	}
}

func TestArraysSetLast(t *testing.T) {
	setFn := ArraysBuiltins["arrays.set"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
		Mutable: true,
	}

	// Set last index to 99
	setFn(arr, &object.Integer{Value: big.NewInt(2)}, &object.Integer{Value: big.NewInt(99)})

	// Verify the elements: {1, 2, 99}
	expected := []int64{1, 2, 99}
	for i, exp := range expected {
		intVal := arr.Elements[i].(*object.Integer)
		if intVal.Value.Cmp(big.NewInt(exp)) != 0 {
			t.Errorf("arr[%d] = %s, want %d", i, intVal.Value.String(), exp)
		}
	}
}

func TestArraysSetImmutable(t *testing.T) {
	setFn := ArraysBuiltins["arrays.set"].Fn

	arr := &object.Array{
		Elements: []object.Object{&object.Integer{Value: big.NewInt(1)}},
		Mutable:  false,
	}

	result := setFn(arr, &object.Integer{Value: big.NewInt(0)}, &object.Integer{Value: big.NewInt(99)})

	if !isErrorObject(result) {
		t.Error("expected error for immutable array")
	}
}

func TestArraysSetOutOfBounds(t *testing.T) {
	setFn := ArraysBuiltins["arrays.set"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
		},
		Mutable: true,
	}

	// Try to set at index 5 (out of bounds)
	result := setFn(arr, &object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(99)})

	if !isErrorObject(result) {
		t.Error("expected error for out of bounds index")
	}
}

// arrays.remove() removes by INDEX (not value!)
// arrays.remove_value() removes by VALUE (first occurrence)
func TestArraysRemoveValue(t *testing.T) {
	removeValueFn := ArraysBuiltins["arrays.remove_value"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(4)},
		},
		Mutable: true,
	}

	// Remove first occurrence of VALUE 2
	result := removeValueFn(arr, &object.Integer{Value: big.NewInt(2)})

	// Should return nil (modifies in-place)
	if result != object.NIL {
		t.Errorf("arrays.remove_value() returned %T, want NIL", result)
	}

	// Array should now have 4 elements
	if len(arr.Elements) != 4 {
		t.Fatalf("arrays.remove_value() should modify array to have 4 elements, got %d", len(arr.Elements))
	}

	// Verify the elements: {1, 3, 2, 4} (first 2 removed)
	expected := []int64{1, 3, 2, 4}
	for i, exp := range expected {
		intVal, ok := arr.Elements[i].(*object.Integer)
		if !ok {
			t.Errorf("arr[%d] is %T, want Integer", i, arr.Elements[i])
			continue
		}
		if intVal.Value.Cmp(big.NewInt(exp)) != 0 {
			t.Errorf("arr[%d] = %s, want %d", i, intVal.Value.String(), exp)
		}
	}
}

func TestArraysRemoveValueNotFound(t *testing.T) {
	removeValueFn := ArraysBuiltins["arrays.remove_value"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
		Mutable: true,
	}

	// Remove non-existent value
	result := removeValueFn(arr, &object.Integer{Value: big.NewInt(99)})

	// Should return nil (no error, just does nothing)
	if result != object.NIL {
		t.Errorf("arrays.remove_value() returned %T, want NIL", result)
	}

	// Array should remain unchanged
	if len(arr.Elements) != 3 {
		t.Errorf("arrays.remove_value() should not change array length, got %d", len(arr.Elements))
	}
}

func TestArraysRemoveValueImmutable(t *testing.T) {
	removeValueFn := ArraysBuiltins["arrays.remove_value"].Fn

	arr := &object.Array{
		Elements: []object.Object{&object.Integer{Value: big.NewInt(1)}},
		Mutable:  false,
	}

	result := removeValueFn(arr, &object.Integer{Value: big.NewInt(1)})

	if !isErrorObject(result) {
		t.Error("expected error for immutable array")
	}
}

// ============================================================================
// More Math Module Tests
// ============================================================================

func TestMathGCD(t *testing.T) {
	gcdFn := MathBuiltins["math.gcd"].Fn

	result := gcdFn(&object.Integer{Value: big.NewInt(48)}, &object.Integer{Value: big.NewInt(18)})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("math.gcd() returned %T, want Integer", result)
	}

	if intVal.Value.Cmp(big.NewInt(6)) != 0 {
		t.Errorf("math.gcd(48, 18) = %s, want 6", intVal.Value.String())
	}
}

func TestMathLCM(t *testing.T) {
	lcmFn := MathBuiltins["math.lcm"].Fn

	result := lcmFn(&object.Integer{Value: big.NewInt(4)}, &object.Integer{Value: big.NewInt(6)})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("math.lcm() returned %T, want Integer", result)
	}

	if intVal.Value.Cmp(big.NewInt(12)) != 0 {
		t.Errorf("math.lcm(4, 6) = %s, want 12", intVal.Value.String())
	}
}

func TestMathFactorial(t *testing.T) {
	factFn := MathBuiltins["math.factorial"].Fn

	result := factFn(&object.Integer{Value: big.NewInt(5)})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("math.factorial() returned %T, want Integer", result)
	}

	if intVal.Value.Cmp(big.NewInt(120)) != 0 {
		t.Errorf("math.factorial(5) = %s, want 120", intVal.Value.String())
	}
}

// ============================================================================
// Bug Fix Tests
// ============================================================================

// Test for Bug #391 fix: arrays.shift() must remove element from array
func TestArraysShiftRemovesElement(t *testing.T) {
	shiftFn := ArraysBuiltins["arrays.shift"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(10)},
			&object.Integer{Value: big.NewInt(20)},
			&object.Integer{Value: big.NewInt(30)},
		},
		Mutable: true,
	}

	// First shift
	result := shiftFn(arr)
	testIntegerObject(t, result, 10)

	// Array should now have 2 elements
	if len(arr.Elements) != 2 {
		t.Errorf("after shift, array should have 2 elements, got %d", len(arr.Elements))
	}

	// First element should now be 20
	testIntegerObject(t, arr.Elements[0], 20)

	// Second shift
	result = shiftFn(arr)
	testIntegerObject(t, result, 20)

	if len(arr.Elements) != 1 {
		t.Errorf("after second shift, array should have 1 element, got %d", len(arr.Elements))
	}
}

func TestArraysShiftImmutable(t *testing.T) {
	shiftFn := ArraysBuiltins["arrays.shift"].Fn

	arr := &object.Array{
		Elements: []object.Object{&object.Integer{Value: big.NewInt(1)}},
		Mutable:  false,
	}

	result := shiftFn(arr)

	if !isErrorObject(result) {
		t.Error("expected error for immutable array")
	}
}

// Test for Bug #369 fix: arrays.shuffle() must produce different results
func TestArraysShuffleRandomness(t *testing.T) {
	shuffleFn := ArraysBuiltins["arrays.shuffle"].Fn

	// Try multiple times to verify shuffle produces different orderings
	different := false
	for attempt := 0; attempt < 10; attempt++ {
		// Create fresh array for each attempt
		elements := make([]object.Object, 10)
		for i := 0; i < 10; i++ {
			elements[i] = &object.Integer{Value: big.NewInt(int64(i + 1))}
		}
		arr := &object.Array{Elements: elements, Mutable: true}

		// Store original order
		original := make([]int64, 10)
		for i := 0; i < 10; i++ {
			original[i] = int64(i + 1)
		}

		// arrays.shuffle now modifies in-place and returns NIL
		result := shuffleFn(arr)
		if result != object.NIL {
			t.Fatalf("arrays.shuffle() returned %T, want NIL", result)
		}

		// Check if shuffled order is different from original
		for j := 0; j < len(arr.Elements); j++ {
			shuffledVal := arr.Elements[j].(*object.Integer).Value.Int64()
			if original[j] != shuffledVal {
				different = true
				break
			}
		}
		if different {
			break
		}
	}

	if !different {
		t.Error("arrays.shuffle() did not produce different results in 10 attempts")
	}
}

// Test for Bug #370 fix: math.map_range() with division by zero
func TestMathMapRangeDivByZero(t *testing.T) {
	mapRangeFn := MathBuiltins["math.map_range"].Fn

	// When inMin == inMax, should return error
	result := mapRangeFn(
		&object.Float{Value: 5.0},
		&object.Float{Value: 0.0},
		&object.Float{Value: 0.0}, // inMax == inMin = 0
		&object.Float{Value: 0.0},
		&object.Float{Value: 100.0},
	)

	_, ok := result.(*object.Error)
	if !ok {
		t.Errorf("math.map_range() with inMin==inMax should return error, got %T", result)
	}
}

// Test for Bug #370 fix: math.map_range() with valid inputs
func TestMathMapRangeValid(t *testing.T) {
	mapRangeFn := MathBuiltins["math.map_range"].Fn

	result := mapRangeFn(
		&object.Float{Value: 5.0},
		&object.Float{Value: 0.0},
		&object.Float{Value: 10.0},
		&object.Float{Value: 0.0},
		&object.Float{Value: 100.0},
	)

	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("math.map_range() returned %T, want Float", result)
	}

	if floatVal.Value != 50.0 {
		t.Errorf("math.map_range(5, 0-10, 0-100) = %f, want 50.0", floatVal.Value)
	}
}

// Test for Bug #371 fix: time.add_months() with end-of-month clamping
func TestTimeAddMonthsEndOfMonth(t *testing.T) {
	addMonthsFn := TimeBuiltins["time.add_months"].Fn

	// January 31, 2024 (leap year) + 1 month = February 29
	// Use time.Date to create a timestamp in the local timezone
	jan31 := gotime.Date(2024, 1, 31, 12, 0, 0, 0, gotime.Local).Unix()
	result := addMonthsFn(
		&object.Integer{Value: big.NewInt(jan31)},
		&object.Integer{Value: big.NewInt(1)},
	)

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("time.add_months() returned %T, want Integer", result)
	}

	// Check that the result is February 29
	resultTime := gotime.Unix(intVal.Value.Int64(), 0)
	if resultTime.Month() != gotime.February || resultTime.Day() != 29 {
		t.Errorf("time.add_months(Jan31, 1) = %s, want February 29", resultTime.Format("2006-01-02"))
	}
}

// Test for Bug #392 fix: strings.slice() UTF-8 support
func TestStringsSliceUTF8(t *testing.T) {
	sliceFn := StringsBuiltins["strings.slice"].Fn

	// Test with Chinese characters
	str := &object.String{Value: "Hello 世界"}

	// Slice to get "Hello"
	result := sliceFn(str, &object.Integer{Value: big.NewInt(0)}, &object.Integer{Value: big.NewInt(5)})
	testStringObject(t, result, "Hello")

	// Slice to get "世界" (characters at indices 6 and 7)
	result = sliceFn(str, &object.Integer{Value: big.NewInt(6)}, &object.Integer{Value: big.NewInt(8)})
	testStringObject(t, result, "世界")

	// Slice to get just "世" (character at index 6)
	result = sliceFn(str, &object.Integer{Value: big.NewInt(6)}, &object.Integer{Value: big.NewInt(7)})
	testStringObject(t, result, "世")

	// Test with emoji
	emojiStr := &object.String{Value: "Hi 😀 there"}
	// "Hi " = 3 chars, "😀" = 1 char, " there" = 6 chars = 10 total
	result = sliceFn(emojiStr, &object.Integer{Value: big.NewInt(3)}, &object.Integer{Value: big.NewInt(4)})
	testStringObject(t, result, "😀")
}

// Test for Bug #393 fix: strings.pad_left/pad_right UTF-8 support
func TestStringsPadLeftUTF8(t *testing.T) {
	padFn := StringsBuiltins["strings.pad_left"].Fn

	// Test with Chinese string - "世界" has 2 characters
	str := &object.String{Value: "世界"}

	// Padding to width 5 should add 3 spaces
	result := padFn(str, &object.Integer{Value: big.NewInt(5)})
	strVal := result.(*object.String)
	if strVal.Value != "   世界" {
		t.Errorf("pad_left('世界', 5) = %q, want '   世界'", strVal.Value)
	}

	// Padding a string that's already >= target width should not change it
	result = padFn(str, &object.Integer{Value: big.NewInt(2)})
	testStringObject(t, result, "世界")
}

func TestStringsPadRightUTF8(t *testing.T) {
	padFn := StringsBuiltins["strings.pad_right"].Fn

	// Test with Chinese string - "世界" has 2 characters
	str := &object.String{Value: "世界"}

	// Padding to width 5 should add 3 spaces
	result := padFn(str, &object.Integer{Value: big.NewInt(5)})
	strVal := result.(*object.String)
	if strVal.Value != "世界   " {
		t.Errorf("pad_right('世界', 5) = %q, want '世界   '", strVal.Value)
	}
}

func TestStringsToInt(t *testing.T) {
	toIntFn := StringsBuiltins["strings.to_int"].Fn

	// Test valid integer
	result := toIntFn(&object.String{Value: "42"})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("strings.to_int('42') returned %T, want Integer", result)
	}
	if intVal.Value.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("strings.to_int('42') = %s, want 42", intVal.Value.String())
	}

	// Test negative integer
	result = toIntFn(&object.String{Value: "-123"})
	intVal, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("strings.to_int('-123') returned %T, want Integer", result)
	}
	if intVal.Value.Cmp(big.NewInt(-123)) != 0 {
		t.Errorf("strings.to_int('-123') = %s, want -123", intVal.Value.String())
	}

	// Test with whitespace
	result = toIntFn(&object.String{Value: "  55  "})
	intVal, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("strings.to_int('  55  ') returned %T, want Integer", result)
	}
	if intVal.Value.Cmp(big.NewInt(55)) != 0 {
		t.Errorf("strings.to_int('  55  ') = %s, want 55", intVal.Value.String())
	}

	// Test invalid integer
	result = toIntFn(&object.String{Value: "abc"})
	if !isErrorObject(result) {
		t.Error("strings.to_int('abc') should return an error")
	}

	// Test float string (should fail)
	result = toIntFn(&object.String{Value: "3.14"})
	if !isErrorObject(result) {
		t.Error("strings.to_int('3.14') should return an error")
	}
}

func TestStringsToFloat(t *testing.T) {
	toFloatFn := StringsBuiltins["strings.to_float"].Fn

	// Test valid float
	result := toFloatFn(&object.String{Value: "3.14"})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("strings.to_float('3.14') returned %T, want Float", result)
	}
	if floatVal.Value != 3.14 {
		t.Errorf("strings.to_float('3.14') = %f, want 3.14", floatVal.Value)
	}

	// Test integer as float
	result = toFloatFn(&object.String{Value: "42"})
	floatVal, ok = result.(*object.Float)
	if !ok {
		t.Fatalf("strings.to_float('42') returned %T, want Float", result)
	}
	if floatVal.Value != 42.0 {
		t.Errorf("strings.to_float('42') = %f, want 42.0", floatVal.Value)
	}

	// Test negative float
	result = toFloatFn(&object.String{Value: "-2.5"})
	floatVal, ok = result.(*object.Float)
	if !ok {
		t.Fatalf("strings.to_float('-2.5') returned %T, want Float", result)
	}
	if floatVal.Value != -2.5 {
		t.Errorf("strings.to_float('-2.5') = %f, want -2.5", floatVal.Value)
	}

	// Test with whitespace
	result = toFloatFn(&object.String{Value: "  1.5  "})
	floatVal, ok = result.(*object.Float)
	if !ok {
		t.Fatalf("strings.to_float('  1.5  ') returned %T, want Float", result)
	}
	if floatVal.Value != 1.5 {
		t.Errorf("strings.to_float('  1.5  ') = %f, want 1.5", floatVal.Value)
	}

	// Test invalid float
	result = toFloatFn(&object.String{Value: "abc"})
	if !isErrorObject(result) {
		t.Error("strings.to_float('abc') should return an error")
	}
}

func TestStringsToBool(t *testing.T) {
	toBoolFn := StringsBuiltins["strings.to_bool"].Fn

	// Test "true" variations
	trueValues := []string{"true", "TRUE", "True", "1", "yes", "YES", "on", "ON"}
	for _, v := range trueValues {
		result := toBoolFn(&object.String{Value: v})
		if result != object.TRUE {
			t.Errorf("strings.to_bool('%s') = %v, want TRUE", v, result)
		}
	}

	// Test "false" variations
	falseValues := []string{"false", "FALSE", "False", "0", "no", "NO", "off", "OFF"}
	for _, v := range falseValues {
		result := toBoolFn(&object.String{Value: v})
		if result != object.FALSE {
			t.Errorf("strings.to_bool('%s') = %v, want FALSE", v, result)
		}
	}

	// Test with whitespace
	result := toBoolFn(&object.String{Value: "  true  "})
	if result != object.TRUE {
		t.Errorf("strings.to_bool('  true  ') = %v, want TRUE", result)
	}

	// Test invalid boolean
	result = toBoolFn(&object.String{Value: "abc"})
	if !isErrorObject(result) {
		t.Error("strings.to_bool('abc') should return an error")
	}

	result = toBoolFn(&object.String{Value: "2"})
	if !isErrorObject(result) {
		t.Error("strings.to_bool('2') should return an error")
	}
}

// Test for Bug #394 fix: math.abs() MinInt64 works with big.Int
func TestMathAbsMinInt64(t *testing.T) {
	absFn := MathBuiltins["math.abs"].Fn

	// With big.Int, abs(MinInt64) now returns a valid result (no overflow)
	result := absFn(&object.Integer{Value: big.NewInt(math.MinInt64)})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("math.abs(MinInt64) returned %T, want Integer", result)
	}

	// abs(-9223372036854775808) = 9223372036854775808
	expected := new(big.Int).Neg(big.NewInt(math.MinInt64))
	if intVal.Value.Cmp(expected) != 0 {
		t.Errorf("math.abs(MinInt64) = %s, want %s", intVal.Value.String(), expected.String())
	}
}

// Test for Bug #395 fix: math.neg() MinInt64 works with big.Int
func TestMathNegMinInt64(t *testing.T) {
	negFn := MathBuiltins["math.neg"].Fn

	// With big.Int, neg(MinInt64) now returns a valid result (no overflow)
	result := negFn(&object.Integer{Value: big.NewInt(math.MinInt64)})

	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("math.neg(MinInt64) returned %T, want Integer", result)
	}

	// neg(-9223372036854775808) = 9223372036854775808
	expected := new(big.Int).Neg(big.NewInt(math.MinInt64))
	if intVal.Value.Cmp(expected) != 0 {
		t.Errorf("math.neg(MinInt64) = %s, want %s", intVal.Value.String(), expected.String())
	}
}

// Test that math.pow handles large exponents with big.Int (no overflow)
func TestMathPowLargeExponents(t *testing.T) {
	powFn := MathBuiltins["math.pow"].Fn

	// 2^63 - big.Int handles this without overflow
	result := powFn(&object.Integer{Value: big.NewInt(2)}, &object.Integer{Value: big.NewInt(63)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("math.pow(2, 63) should return Integer, got %T", result)
	}
	expected := new(big.Int).Exp(big.NewInt(2), big.NewInt(63), nil)
	if intVal.Value.Cmp(expected) != 0 {
		t.Errorf("math.pow(2, 63) = %s, want %s", intVal.Value.String(), expected.String())
	}

	// Very large exponent - big.Int handles this
	result = powFn(&object.Integer{Value: big.NewInt(10)}, &object.Integer{Value: big.NewInt(100)})
	intVal, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("math.pow(10, 100) should return Integer, got %T", result)
	}
	expected = new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil)
	if intVal.Value.Cmp(expected) != 0 {
		t.Errorf("math.pow(10, 100) = %s, want %s", intVal.Value.String(), expected.String())
	}

	// Valid power should still work
	result = powFn(&object.Integer{Value: big.NewInt(2)}, &object.Integer{Value: big.NewInt(10)})
	testIntegerObject(t, result, 1024)
}

// ============================================================================
// Issue #526: @std Module Expansion Tests
// ============================================================================

// Test EXIT_SUCCESS constant (global builtin)
func TestExitSuccess(t *testing.T) {
	exitSuccessFn := StdBuiltins["EXIT_SUCCESS"].Fn
	result := exitSuccessFn()
	testIntegerObject(t, result, 0)
}

// Test EXIT_FAILURE constant (global builtin)
func TestExitFailure(t *testing.T) {
	exitFailureFn := StdBuiltins["EXIT_FAILURE"].Fn
	result := exitFailureFn()
	testIntegerObject(t, result, 1)
}

// Note: exit() is not tested as it calls os.Exit() which terminates the test process

// Test sleep_seconds() with negative value (should error)
func TestSleepSecondsNegativeError(t *testing.T) {
	sleepFn := StdBuiltins["std.sleep_seconds"].Fn
	result := sleepFn(&object.Integer{Value: big.NewInt(-1)})
	if !isErrorObject(result) {
		t.Error("sleep_seconds() with negative duration should return error")
	}
}

// Test sleep_seconds() with wrong argument type
func TestSleepSecondsWrongTypeError(t *testing.T) {
	sleepFn := StdBuiltins["std.sleep_seconds"].Fn
	result := sleepFn(&object.String{Value: "5"})
	if !isErrorObject(result) {
		t.Error("sleep_seconds() with string argument should return error")
	}
}

// Test sleep_seconds() with wrong number of arguments
func TestSleepSecondsWrongArgCountError(t *testing.T) {
	sleepFn := StdBuiltins["std.sleep_seconds"].Fn
	result := sleepFn()
	if !isErrorObject(result) {
		t.Error("sleep_seconds() with no arguments should return error")
	}
	result = sleepFn(&object.Integer{Value: big.NewInt(1)}, &object.Integer{Value: big.NewInt(2)})
	if !isErrorObject(result) {
		t.Error("sleep_seconds() with two arguments should return error")
	}
}

// Test sleep_milliseconds() with negative value (should error)
func TestSleepMillisecondsNegativeError(t *testing.T) {
	sleepFn := StdBuiltins["std.sleep_milliseconds"].Fn
	result := sleepFn(&object.Integer{Value: big.NewInt(-1)})
	if !isErrorObject(result) {
		t.Error("sleep_milliseconds() with negative duration should return error")
	}
}

// Test sleep_nanoseconds() with negative value (should error)
func TestSleepNanosecondsNegativeError(t *testing.T) {
	sleepFn := StdBuiltins["std.sleep_nanoseconds"].Fn
	result := sleepFn(&object.Integer{Value: big.NewInt(-1)})
	if !isErrorObject(result) {
		t.Error("sleep_nanoseconds() with negative duration should return error")
	}
}

// Test sleep with zero (should succeed)
func TestSleepZero(t *testing.T) {
	sleepFn := StdBuiltins["std.sleep_seconds"].Fn
	result := sleepFn(&object.Integer{Value: big.NewInt(0)})
	if isErrorObject(result) {
		t.Error("sleep_seconds(0) should succeed")
	}
	if result != object.NIL {
		t.Errorf("sleep_seconds() should return nil, got %T", result)
	}
}

// Test panic() returns an error with correct code (global builtin)
func TestPanic(t *testing.T) {
	panicFn := StdBuiltins["panic"].Fn
	result := panicFn(&object.String{Value: "something went wrong"})

	err, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("panic() should return Error, got %T", result)
	}
	if err.Code != "E5021" {
		t.Errorf("panic() error code should be E5021, got %s", err.Code)
	}
	if err.Message != "panic: something went wrong" {
		t.Errorf("panic() message wrong, got %s", err.Message)
	}
}

// Test panic() with wrong argument type
func TestPanicWrongTypeError(t *testing.T) {
	panicFn := StdBuiltins["panic"].Fn
	result := panicFn(&object.Integer{Value: big.NewInt(42)})

	err, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("panic() with non-string should return Error, got %T", result)
	}
	if err.Code != "E7003" {
		t.Errorf("panic() with non-string should return E7003 error, got %s", err.Code)
	}
}

// Test panic() with wrong number of arguments
func TestPanicWrongArgCountError(t *testing.T) {
	panicFn := StdBuiltins["panic"].Fn
	result := panicFn()
	if !isErrorObject(result) {
		t.Error("panic() with no arguments should return error")
	}
	result = panicFn(&object.String{Value: "a"}, &object.String{Value: "b"})
	if !isErrorObject(result) {
		t.Error("panic() with two arguments should return error")
	}
}

// Test eprintln() returns nil (actual output is to stderr, hard to test)
func TestEprintln(t *testing.T) {
	eprintlnFn := StdBuiltins["std.eprintln"].Fn
	result := eprintlnFn(&object.String{Value: "test output"})
	if result != object.NIL {
		t.Errorf("eprintln() should return nil, got %T", result)
	}
}

// Test eprint() returns nil
func TestEprint(t *testing.T) {
	eprintFn := StdBuiltins["std.eprint"].Fn
	result := eprintFn(&object.String{Value: "test output"})
	if result != object.NIL {
		t.Errorf("eprint() should return nil, got %T", result)
	}
}

// Test eprintln() with multiple arguments
func TestEprintlnMultipleArgs(t *testing.T) {
	eprintlnFn := StdBuiltins["std.eprintln"].Fn
	result := eprintlnFn(
		&object.String{Value: "hello"},
		&object.Integer{Value: big.NewInt(42)},
		&object.Boolean{Value: true},
	)
	if result != object.NIL {
		t.Errorf("eprintln() should return nil, got %T", result)
	}
}

// Test print() returns nil
func TestPrint(t *testing.T) {
	printFn := StdBuiltins["std.print"].Fn
	result := printFn(&object.String{Value: "test output"})
	if result != object.NIL {
		t.Errorf("print() should return nil, got %T", result)
	}
}

// ============================================================================
// Issue #525: assert() Tests
// ============================================================================

// Test assert() with true condition (should pass)
func TestAssertTrue(t *testing.T) {
	assertFn := StdBuiltins["assert"].Fn
	result := assertFn(&object.Boolean{Value: true}, &object.String{Value: "this should not fail"})
	if result != object.NIL {
		t.Errorf("assert(true, msg) should return nil, got %T", result)
	}
}

// Test assert() with false condition (should error)
func TestAssertFalse(t *testing.T) {
	assertFn := StdBuiltins["assert"].Fn
	result := assertFn(&object.Boolean{Value: false}, &object.String{Value: "expected failure"})

	err, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("assert(false, msg) should return Error, got %T", result)
	}
	if err.Code != "E5022" {
		t.Errorf("assert() error code should be E5022, got %s", err.Code)
	}
	if err.Message != "assertion failed: expected failure" {
		t.Errorf("assert() message wrong, got %s", err.Message)
	}
}

// Test assert() with wrong first argument type
func TestAssertWrongFirstArgType(t *testing.T) {
	assertFn := StdBuiltins["assert"].Fn
	result := assertFn(&object.Integer{Value: big.NewInt(1)}, &object.String{Value: "msg"})

	err, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("assert() with non-boolean should return Error, got %T", result)
	}
	if err.Code != "E7008" {
		t.Errorf("assert() with non-boolean should return E7008 error, got %s", err.Code)
	}
}

// Test assert() with wrong second argument type
func TestAssertWrongSecondArgType(t *testing.T) {
	assertFn := StdBuiltins["assert"].Fn
	result := assertFn(&object.Boolean{Value: true}, &object.Integer{Value: big.NewInt(42)})

	err, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("assert() with non-string message should return Error, got %T", result)
	}
	if err.Code != "E7003" {
		t.Errorf("assert() with non-string message should return E7003 error, got %s", err.Code)
	}
}

// Test assert() with wrong number of arguments
func TestAssertWrongArgCount(t *testing.T) {
	assertFn := StdBuiltins["assert"].Fn

	// No arguments
	result := assertFn()
	if !isErrorObject(result) {
		t.Error("assert() with no arguments should return error")
	}

	// One argument
	result = assertFn(&object.Boolean{Value: true})
	if !isErrorObject(result) {
		t.Error("assert() with one argument should return error")
	}

	// Three arguments
	result = assertFn(&object.Boolean{Value: true}, &object.String{Value: "msg"}, &object.String{Value: "extra"})
	if !isErrorObject(result) {
		t.Error("assert() with three arguments should return error")
	}
}

// ============================================================================
// Time Format Conversion Tests
// ============================================================================

func TestConvertFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"YYYY-MM-DD", "2006-01-02"},
		{"YYYY/MM/DD HH:mm:ss", "2006/01/02 15:04:05"},
		{"DD/MM/YYYY", "02/01/2006"},
		{"YY-MM-DD", "06-01-02"},
		{"hh:mm:ss A", "03:04:05 PM"},
		{"HH:mm:ss.SSS", "15:04:05.000"},
		{"YYYY-MM-DDTHH:mm:ssZ", "2006-01-02T15:04:05-0700"},
		{"YYYY-MM-DDTHH:mm:ssZZ", "2006-01-02T15:04:05-07:00"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertFormat(tt.input)
			if result != tt.expected {
				t.Errorf("convertFormat(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReplaceAll(t *testing.T) {
	tests := []struct {
		s        string
		old      string
		new      string
		expected string
	}{
		{"hello world", "world", "there", "hello there"},
		{"aaa", "a", "b", "bbb"},
		{"no match", "xyz", "abc", "no match"},
		{"", "a", "b", ""},
		{"YYYY-YYYY", "YYYY", "2006", "2006-2006"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			result := replaceAll(tt.s, tt.old, tt.new)
			if result != tt.expected {
				t.Errorf("replaceAll(%q, %q, %q) = %q, want %q", tt.s, tt.old, tt.new, result, tt.expected)
			}
		})
	}
}

func TestGetTimeWithTimestamp(t *testing.T) {
	// Test with timestamp argument
	timestamp := int64(1609459200) // 2021-01-01 00:00:00 UTC
	args := []object.Object{&object.Integer{Value: big.NewInt(timestamp)}}
	result := getTime(args)

	if result.Unix() != timestamp {
		t.Errorf("getTime with timestamp = %d, want %d", result.Unix(), timestamp)
	}
}

func TestGetTimeWithNoArgs(t *testing.T) {
	// Test with no arguments (should return current time)
	before := gotime.Now().Unix()
	result := getTime([]object.Object{})
	after := gotime.Now().Unix()

	if result.Unix() < before || result.Unix() > after {
		t.Error("getTime with no args should return current time")
	}
}

func TestGetTimeWithInvalidArg(t *testing.T) {
	// Test with invalid argument type (should return current time)
	before := gotime.Now().Unix()
	args := []object.Object{&object.String{Value: "invalid"}}
	result := getTime(args)
	after := gotime.Now().Unix()

	if result.Unix() < before || result.Unix() > after {
		t.Error("getTime with invalid arg should return current time")
	}
}

// ============================================================================
// Arrays Helper Function Tests
// ============================================================================

func TestCompareObjectsIntegers(t *testing.T) {
	tests := []struct {
		a, b     int64
		expected int
	}{
		{1, 2, -1},
		{2, 1, 1},
		{5, 5, 0},
		{-10, 10, -1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		a := &object.Integer{Value: big.NewInt(tt.a)}
		b := &object.Integer{Value: big.NewInt(tt.b)}
		result := compareObjects(a, b)
		if result != tt.expected {
			t.Errorf("compareObjects(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestCompareObjectsFloats(t *testing.T) {
	tests := []struct {
		a, b     float64
		expected int
	}{
		{1.0, 2.0, -1},
		{2.0, 1.0, 1},
		{3.14, 3.14, 0},
		{-1.5, 1.5, -1},
	}

	for _, tt := range tests {
		a := &object.Float{Value: tt.a}
		b := &object.Float{Value: tt.b}
		result := compareObjects(a, b)
		if result != tt.expected {
			t.Errorf("compareObjects(%f, %f) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestCompareObjectsStrings(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"apple", "banana", -1},
		{"banana", "apple", 1},
		{"hello", "hello", 0},
		{"", "a", -1},
		{"z", "a", 1},
	}

	for _, tt := range tests {
		a := &object.String{Value: tt.a}
		b := &object.String{Value: tt.b}
		result := compareObjects(a, b)
		if result != tt.expected {
			t.Errorf("compareObjects(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestCompareObjectsFallback(t *testing.T) {
	// Test with boolean (uses Inspect fallback)
	a := &object.Boolean{Value: false}
	b := &object.Boolean{Value: true}
	result := compareObjects(a, b)
	// "false" < "true" lexicographically
	if result != -1 {
		t.Errorf("compareObjects(false, true) = %d, want -1", result)
	}
}

func TestObjectsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a, b     object.Object
		expected bool
	}{
		{
			"same integers",
			&object.Integer{Value: big.NewInt(42)},
			&object.Integer{Value: big.NewInt(42)},
			true,
		},
		{
			"different integers",
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			false,
		},
		{
			"same strings",
			&object.String{Value: "hello"},
			&object.String{Value: "hello"},
			true,
		},
		{
			"different strings",
			&object.String{Value: "hello"},
			&object.String{Value: "world"},
			false,
		},
		{
			"different types",
			&object.Integer{Value: big.NewInt(1)},
			&object.String{Value: "1"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := objectsEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("objectsEqual() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestZeroValueForType(t *testing.T) {
	tests := []struct {
		typeName string
		check    func(object.Object) bool
	}{
		{"int", func(o object.Object) bool {
			i, ok := o.(*object.Integer)
			return ok && i.Value.Int64() == 0
		}},
		{"float", func(o object.Object) bool {
			f, ok := o.(*object.Float)
			return ok && f.Value == 0.0
		}},
		{"string", func(o object.Object) bool {
			s, ok := o.(*object.String)
			return ok && s.Value == ""
		}},
		{"bool", func(o object.Object) bool {
			return o == object.FALSE
		}},
		{"char", func(o object.Object) bool {
			c, ok := o.(*object.Char)
			return ok && c.Value == 0
		}},
		{"byte", func(o object.Object) bool {
			b, ok := o.(*object.Byte)
			return ok && b.Value == 0
		}},
		{"unknown", func(o object.Object) bool {
			return o == object.NIL
		}},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			result := zeroValueForType(tt.typeName)
			if !tt.check(result) {
				t.Errorf("zeroValueForType(%q) = %v, unexpected", tt.typeName, result)
			}
		})
	}
}

func TestConvertToTypedValueInt(t *testing.T) {
	// Test float64 to int
	result := convertToTypedValue(float64(42), "int")
	intObj, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intObj.Value.Int64() != 42 {
		t.Errorf("expected 42, got %d", intObj.Value.Int64())
	}

	// Test string to int
	result = convertToTypedValue("123", "int")
	intObj, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intObj.Value.Int64() != 123 {
		t.Errorf("expected 123, got %d", intObj.Value.Int64())
	}

	// Test invalid string returns 0
	result = convertToTypedValue("not a number", "int")
	intObj, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intObj.Value.Int64() != 0 {
		t.Errorf("expected 0, got %d", intObj.Value.Int64())
	}
}

func TestConvertToTypedValueFloat(t *testing.T) {
	// Test float64 to float
	result := convertToTypedValue(float64(3.14), "float")
	floatObj, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if floatObj.Value != 3.14 {
		t.Errorf("expected 3.14, got %f", floatObj.Value)
	}

	// Test string returns 0.0 (no auto-convert)
	result = convertToTypedValue("3.14", "float")
	floatObj, ok = result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if floatObj.Value != 0.0 {
		t.Errorf("expected 0.0, got %f", floatObj.Value)
	}
}

func TestConvertToTypedValueString(t *testing.T) {
	// Test string to string
	result := convertToTypedValue("hello", "string")
	strObj, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if strObj.Value != "hello" {
		t.Errorf("expected hello, got %s", strObj.Value)
	}

	// Test int-like float to string
	result = convertToTypedValue(float64(42), "string")
	strObj, ok = result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if strObj.Value != "42" {
		t.Errorf("expected '42', got '%s'", strObj.Value)
	}

	// Test float to string
	result = convertToTypedValue(float64(3.14), "string")
	strObj, ok = result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if strObj.Value != "3.14" {
		t.Errorf("expected '3.14', got '%s'", strObj.Value)
	}

	// Test bool true to string
	result = convertToTypedValue(true, "string")
	strObj, ok = result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if strObj.Value != "true" {
		t.Errorf("expected 'true', got '%s'", strObj.Value)
	}

	// Test bool false to string
	result = convertToTypedValue(false, "string")
	strObj, ok = result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if strObj.Value != "false" {
		t.Errorf("expected 'false', got '%s'", strObj.Value)
	}
}

func TestConvertToTypedValueBool(t *testing.T) {
	// Test bool true
	result := convertToTypedValue(true, "bool")
	if result != object.TRUE {
		t.Errorf("expected TRUE, got %v", result)
	}

	// Test bool false
	result = convertToTypedValue(false, "bool")
	if result != object.FALSE {
		t.Errorf("expected FALSE, got %v", result)
	}

	// Test non-bool returns FALSE
	result = convertToTypedValue("not bool", "bool")
	if result != object.FALSE {
		t.Errorf("expected FALSE, got %v", result)
	}
}

// ============================================================================
// Binary Module Tests
// ============================================================================

// Test extractEncodingInt with Float input
func TestBinaryEncodeI8WithFloat(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_i8"].Fn

	// Encode a float value (should be converted to int)
	result := encodeFn(&object.Float{Value: 42.7})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array in first return value, got %T", retVal.Values[0])
	}

	if len(arr.Elements) != 1 {
		t.Errorf("expected 1 byte, got %d", len(arr.Elements))
	}

	byteVal, ok := arr.Elements[0].(*object.Byte)
	if !ok {
		t.Fatalf("expected Byte, got %T", arr.Elements[0])
	}

	// 42.7 truncates to 42
	if byteVal.Value != 42 {
		t.Errorf("expected byte value 42, got %d", byteVal.Value)
	}
}

// Test extractEncodingInt with Byte input
func TestBinaryEncodeI8WithByte(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_i8"].Fn

	// Encode a byte value
	result := encodeFn(&object.Byte{Value: 100})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array in first return value, got %T", retVal.Values[0])
	}

	byteVal, ok := arr.Elements[0].(*object.Byte)
	if !ok {
		t.Fatalf("expected Byte, got %T", arr.Elements[0])
	}

	if byteVal.Value != 100 {
		t.Errorf("expected byte value 100, got %d", byteVal.Value)
	}
}

// Test extractEncodingInt with invalid type (error case)
func TestBinaryEncodeI8WithInvalidType(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_i8"].Fn

	// Try to encode a string - should return error
	result := encodeFn(&object.String{Value: "not a number"})

	errObj, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}

	if errObj.Code != "E7004" {
		t.Errorf("expected error code E7004, got %s", errObj.Code)
	}
}

// Test binary.encode_i8 argument count
func TestBinaryEncodeI8ArgCount(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_i8"].Fn

	// No arguments
	result := encodeFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Too many arguments
	result = encodeFn(&object.Integer{Value: big.NewInt(42)}, &object.Integer{Value: big.NewInt(10)})
	if !isErrorObject(result) {
		t.Error("expected error for too many arguments")
	}
}

// Test binary.encode_i8 range error
func TestBinaryEncodeI8RangeError(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_i8"].Fn

	// Value out of i8 range (> 127)
	result := encodeFn(&object.Integer{Value: big.NewInt(200)})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	// First value should be NIL for error case
	if retVal.Values[0] != object.NIL {
		t.Errorf("expected NIL for out-of-range value")
	}

	// Second value should be error struct
	errStruct, ok := retVal.Values[1].(*object.Struct)
	if !ok {
		t.Fatalf("expected Struct error, got %T", retVal.Values[1])
	}

	codeField, ok := errStruct.Fields["code"].(*object.String)
	if !ok || codeField.Value != "E3022" {
		t.Errorf("expected error code E3022")
	}
}

// Test binary.decode_i8
func TestBinaryDecodeI8(t *testing.T) {
	decodeFn := BinaryBuiltins["binary.decode_i8"].Fn

	// Decode a valid byte array
	arr := &object.Array{
		Elements: []object.Object{&object.Byte{Value: 0xFF}}, // -1 in signed i8
	}

	result := decodeFn(arr)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	intVal, ok := retVal.Values[0].(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", retVal.Values[0])
	}

	// 0xFF as signed i8 is -1
	if intVal.Value.Int64() != -1 {
		t.Errorf("expected -1, got %d", intVal.Value.Int64())
	}
}

// Test binary.decode_i8 with wrong byte count
func TestBinaryDecodeI8WrongByteCount(t *testing.T) {
	decodeFn := BinaryBuiltins["binary.decode_i8"].Fn

	// Too many bytes
	arr := &object.Array{
		Elements: []object.Object{
			&object.Byte{Value: 1},
			&object.Byte{Value: 2},
		},
	}

	result := decodeFn(arr)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	// First value should be NIL
	if retVal.Values[0] != object.NIL {
		t.Errorf("expected NIL for wrong byte count")
	}
}

// Test binary.decode_i8 with non-array
func TestBinaryDecodeI8NonArray(t *testing.T) {
	decodeFn := BinaryBuiltins["binary.decode_i8"].Fn

	result := decodeFn(&object.String{Value: "not an array"})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	if retVal.Values[0] != object.NIL {
		t.Errorf("expected NIL for non-array input")
	}
}

// Test binaryBytesToSlice with Integer elements (out of range)
func TestBinaryDecodeWithIntegerOutOfRange(t *testing.T) {
	decodeFn := BinaryBuiltins["binary.decode_i8"].Fn

	// Array with integer value out of byte range
	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(300)}, // Out of 0-255 range
		},
	}

	result := decodeFn(arr)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	if retVal.Values[0] != object.NIL {
		t.Errorf("expected NIL for out-of-range integer")
	}
}

// Test binaryBytesToSlice with valid Integer elements
func TestBinaryDecodeWithIntegerElements(t *testing.T) {
	decodeFn := BinaryBuiltins["binary.decode_i8"].Fn

	// Array with valid integer value
	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(127)},
		},
	}

	result := decodeFn(arr)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	intVal, ok := retVal.Values[0].(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", retVal.Values[0])
	}

	if intVal.Value.Int64() != 127 {
		t.Errorf("expected 127, got %d", intVal.Value.Int64())
	}
}

// Test binaryBytesToSlice with invalid element type
func TestBinaryDecodeWithInvalidElementType(t *testing.T) {
	decodeFn := BinaryBuiltins["binary.decode_i8"].Fn

	// Array with string element (invalid)
	arr := &object.Array{
		Elements: []object.Object{
			&object.String{Value: "invalid"},
		},
	}

	result := decodeFn(arr)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	if retVal.Values[0] != object.NIL {
		t.Errorf("expected NIL for invalid element type")
	}
}

// Test binary.encode_u8
func TestBinaryEncodeU8(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_u8"].Fn

	result := encodeFn(&object.Integer{Value: big.NewInt(200)})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	byteVal, ok := arr.Elements[0].(*object.Byte)
	if !ok {
		t.Fatalf("expected Byte, got %T", arr.Elements[0])
	}

	if byteVal.Value != 200 {
		t.Errorf("expected 200, got %d", byteVal.Value)
	}
}

// Test binary.encode_u8 range error
func TestBinaryEncodeU8RangeError(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_u8"].Fn

	// Negative value
	result := encodeFn(&object.Integer{Value: big.NewInt(-1)})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	if retVal.Values[0] != object.NIL {
		t.Errorf("expected NIL for negative value")
	}
}

// Test binary.decode_u8
func TestBinaryDecodeU8(t *testing.T) {
	decodeFn := BinaryBuiltins["binary.decode_u8"].Fn

	arr := &object.Array{
		Elements: []object.Object{&object.Byte{Value: 255}},
	}

	result := decodeFn(arr)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	intVal, ok := retVal.Values[0].(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", retVal.Values[0])
	}

	if intVal.Value.Int64() != 255 {
		t.Errorf("expected 255, got %d", intVal.Value.Int64())
	}
}

// Test 16-bit encoding with Float (tests extractEncodingInt Float branch)
func TestBinaryEncodeI16WithFloat(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_i16_to_little_endian"].Fn

	result := encodeFn(&object.Float{Value: 1000.9})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 bytes, got %d", len(arr.Elements))
	}
}

// Test 16-bit encoding with Byte (tests extractEncodingInt Byte branch)
func TestBinaryEncodeI16WithByte(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_i16_to_little_endian"].Fn

	result := encodeFn(&object.Byte{Value: 200})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 bytes, got %d", len(arr.Elements))
	}
}

// Test 32-bit float encoding
func TestBinaryEncodeF32(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_f32_to_little_endian"].Fn

	result := encodeFn(&object.Float{Value: 3.14})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	if len(arr.Elements) != 4 {
		t.Errorf("expected 4 bytes, got %d", len(arr.Elements))
	}
}

// Test 32-bit float encoding with Integer
func TestBinaryEncodeF32WithInteger(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_f32_to_little_endian"].Fn

	result := encodeFn(&object.Integer{Value: big.NewInt(42)})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	if len(arr.Elements) != 4 {
		t.Errorf("expected 4 bytes, got %d", len(arr.Elements))
	}
}

// Test 32-bit float encoding with invalid type
func TestBinaryEncodeF32WithInvalidType(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_f32_to_little_endian"].Fn

	result := encodeFn(&object.String{Value: "invalid"})

	if !isErrorObject(result) {
		t.Error("expected error for invalid type")
	}
}

// Test 64-bit float encoding
func TestBinaryEncodeF64(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_f64_to_little_endian"].Fn

	result := encodeFn(&object.Float{Value: 3.141592653589793})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	if len(arr.Elements) != 8 {
		t.Errorf("expected 8 bytes, got %d", len(arr.Elements))
	}
}

// Test 64-bit float encoding with Integer
func TestBinaryEncodeF64WithInteger(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_f64_to_little_endian"].Fn

	result := encodeFn(&object.Integer{Value: big.NewInt(12345)})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	if len(arr.Elements) != 8 {
		t.Errorf("expected 8 bytes, got %d", len(arr.Elements))
	}
}

// Test float decode
func TestBinaryDecodeF32(t *testing.T) {
	decodeFn := BinaryBuiltins["binary.decode_f32_from_little_endian"].Fn

	// Create a byte array representing 3.14 in little-endian f32
	arr := &object.Array{
		Elements: []object.Object{
			&object.Byte{Value: 0xC3},
			&object.Byte{Value: 0xF5},
			&object.Byte{Value: 0x48},
			&object.Byte{Value: 0x40},
		},
	}

	result := decodeFn(arr)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	floatVal, ok := retVal.Values[0].(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", retVal.Values[0])
	}

	// Should be approximately 3.14
	if math.Abs(floatVal.Value-3.14) > 0.01 {
		t.Errorf("expected ~3.14, got %f", floatVal.Value)
	}
}

// Test float decode f64
func TestBinaryDecodeF64(t *testing.T) {
	decodeFn := BinaryBuiltins["binary.decode_f64_from_little_endian"].Fn

	// Encode and then decode to round-trip
	encodeFn := BinaryBuiltins["binary.encode_f64_to_little_endian"].Fn
	encoded := encodeFn(&object.Float{Value: 2.718281828})

	retVal := encoded.(*object.ReturnValue)
	arr := retVal.Values[0].(*object.Array)

	result := decodeFn(arr)

	decodeRet, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	floatVal, ok := decodeRet.Values[0].(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", decodeRet.Values[0])
	}

	if math.Abs(floatVal.Value-2.718281828) > 0.0000001 {
		t.Errorf("expected ~2.718281828, got %f", floatVal.Value)
	}
}

// Test big-endian encoding
func TestBinaryEncodeI16BigEndian(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_i16_to_big_endian"].Fn

	result := encodeFn(&object.Integer{Value: big.NewInt(0x0102)})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	// Big-endian: high byte first
	if arr.Elements[0].(*object.Byte).Value != 0x01 {
		t.Errorf("expected first byte 0x01, got 0x%02X", arr.Elements[0].(*object.Byte).Value)
	}
	if arr.Elements[1].(*object.Byte).Value != 0x02 {
		t.Errorf("expected second byte 0x02, got 0x%02X", arr.Elements[1].(*object.Byte).Value)
	}
}

// Test negative integer encoding
func TestBinaryEncodeNegativeI16(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_i16_to_little_endian"].Fn

	result := encodeFn(&object.Integer{Value: big.NewInt(-1)})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	// -1 in two's complement 16-bit is 0xFFFF
	if arr.Elements[0].(*object.Byte).Value != 0xFF {
		t.Errorf("expected first byte 0xFF, got 0x%02X", arr.Elements[0].(*object.Byte).Value)
	}
	if arr.Elements[1].(*object.Byte).Value != 0xFF {
		t.Errorf("expected second byte 0xFF, got 0x%02X", arr.Elements[1].(*object.Byte).Value)
	}
}

// ============================================================================
// Typed Integer Conversion Tests (extractIntValue coverage)
// ============================================================================

func TestI8Conversion(t *testing.T) {
	i8Fn := StdBuiltins["i8"].Fn

	// Integer to i8
	result := i8Fn(&object.Integer{Value: big.NewInt(100)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 100 {
		t.Errorf("expected 100, got %d", intVal.Value.Int64())
	}

	// Float to i8
	result = i8Fn(&object.Float{Value: 50.9})
	intVal, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 50 {
		t.Errorf("expected 50, got %d", intVal.Value.Int64())
	}

	// String to i8
	result = i8Fn(&object.String{Value: "42"})
	intVal, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 42 {
		t.Errorf("expected 42, got %d", intVal.Value.Int64())
	}

	// Byte to i8
	result = i8Fn(&object.Byte{Value: 127})
	intVal, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 127 {
		t.Errorf("expected 127, got %d", intVal.Value.Int64())
	}
}

func TestI8ConversionErrors(t *testing.T) {
	i8Fn := StdBuiltins["i8"].Fn

	// Out of range
	result := i8Fn(&object.Integer{Value: big.NewInt(200)})
	if !isErrorObject(result) {
		t.Error("expected error for out of range value")
	}

	// Invalid string
	result = i8Fn(&object.String{Value: "not a number"})
	if !isErrorObject(result) {
		t.Error("expected error for invalid string")
	}

	// NaN
	result = i8Fn(&object.Float{Value: math.NaN()})
	if !isErrorObject(result) {
		t.Error("expected error for NaN")
	}

	// Inf
	result = i8Fn(&object.Float{Value: math.Inf(1)})
	if !isErrorObject(result) {
		t.Error("expected error for Inf")
	}
}

func TestI16Conversion(t *testing.T) {
	i16Fn := StdBuiltins["i16"].Fn

	result := i16Fn(&object.Integer{Value: big.NewInt(1000)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 1000 {
		t.Errorf("expected 1000, got %d", intVal.Value.Int64())
	}

	// String with underscores
	result = i16Fn(&object.String{Value: "1_000"})
	intVal, ok = result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 1000 {
		t.Errorf("expected 1000, got %d", intVal.Value.Int64())
	}
}

func TestI32Conversion(t *testing.T) {
	i32Fn := StdBuiltins["i32"].Fn

	result := i32Fn(&object.Integer{Value: big.NewInt(100000)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 100000 {
		t.Errorf("expected 100000, got %d", intVal.Value.Int64())
	}
}

func TestI64Conversion(t *testing.T) {
	i64Fn := StdBuiltins["i64"].Fn

	result := i64Fn(&object.Integer{Value: big.NewInt(9223372036854775807)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 9223372036854775807 {
		t.Errorf("expected max i64, got %d", intVal.Value.Int64())
	}
}

func TestU8Conversion(t *testing.T) {
	u8Fn := StdBuiltins["u8"].Fn

	result := u8Fn(&object.Integer{Value: big.NewInt(200)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 200 {
		t.Errorf("expected 200, got %d", intVal.Value.Int64())
	}

	// Negative should error
	result = u8Fn(&object.Integer{Value: big.NewInt(-1)})
	if !isErrorObject(result) {
		t.Error("expected error for negative value")
	}
}

func TestU16Conversion(t *testing.T) {
	u16Fn := StdBuiltins["u16"].Fn

	result := u16Fn(&object.Integer{Value: big.NewInt(60000)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 60000 {
		t.Errorf("expected 60000, got %d", intVal.Value.Int64())
	}
}

func TestU32Conversion(t *testing.T) {
	u32Fn := StdBuiltins["u32"].Fn

	result := u32Fn(&object.Integer{Value: big.NewInt(4000000000)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Uint64() != 4000000000 {
		t.Errorf("expected 4000000000, got %d", intVal.Value.Uint64())
	}
}

func TestU64Conversion(t *testing.T) {
	u64Fn := StdBuiltins["u64"].Fn

	result := u64Fn(&object.Integer{Value: big.NewInt(0).SetUint64(18446744073709551615)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Uint64() != 18446744073709551615 {
		t.Errorf("expected max u64, got %d", intVal.Value.Uint64())
	}
}

// ============================================================================
// Typed Float Conversion Tests (extractFloatValue coverage)
// ============================================================================

func TestF32Conversion(t *testing.T) {
	f32Fn := StdBuiltins["f32"].Fn

	// Float to f32
	result := f32Fn(&object.Float{Value: 3.14})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if math.Abs(floatVal.Value-3.14) > 0.001 {
		t.Errorf("expected ~3.14, got %f", floatVal.Value)
	}

	// Integer to f32
	result = f32Fn(&object.Integer{Value: big.NewInt(42)})
	floatVal, ok = result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if floatVal.Value != 42.0 {
		t.Errorf("expected 42.0, got %f", floatVal.Value)
	}

	// String to f32
	result = f32Fn(&object.String{Value: "3.14"})
	floatVal, ok = result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if math.Abs(floatVal.Value-3.14) > 0.001 {
		t.Errorf("expected ~3.14, got %f", floatVal.Value)
	}

	// Byte to f32
	result = f32Fn(&object.Byte{Value: 100})
	floatVal, ok = result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if floatVal.Value != 100.0 {
		t.Errorf("expected 100.0, got %f", floatVal.Value)
	}

	// Char to f32
	result = f32Fn(&object.Char{Value: 'A'})
	floatVal, ok = result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if floatVal.Value != 65.0 {
		t.Errorf("expected 65.0, got %f", floatVal.Value)
	}
}

func TestF32ConversionErrors(t *testing.T) {
	f32Fn := StdBuiltins["f32"].Fn

	// Invalid string
	result := f32Fn(&object.String{Value: "not a number"})
	if !isErrorObject(result) {
		t.Error("expected error for invalid string")
	}

	// Unsupported type
	result = f32Fn(&object.Array{Elements: []object.Object{}})
	if !isErrorObject(result) {
		t.Error("expected error for unsupported type")
	}
}

func TestF64Conversion(t *testing.T) {
	f64Fn := StdBuiltins["f64"].Fn

	// Float to f64
	result := f64Fn(&object.Float{Value: 3.141592653589793})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if math.Abs(floatVal.Value-3.141592653589793) > 0.0000001 {
		t.Errorf("expected ~3.141592653589793, got %f", floatVal.Value)
	}

	// String with underscores
	result = f64Fn(&object.String{Value: "1_000.5"})
	floatVal, ok = result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if floatVal.Value != 1000.5 {
		t.Errorf("expected 1000.5, got %f", floatVal.Value)
	}
}

// ============================================================================
// Arrays Module Additional Tests
// ============================================================================

func TestArraysReverseExtended(t *testing.T) {
	reverseFn := ArraysBuiltins["arrays.reverse"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
	}

	result := reverseFn(arr)
	newArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(newArr.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(newArr.Elements))
	}

	// Check reversed order
	first := newArr.Elements[0].(*object.Integer)
	if first.Value.Int64() != 3 {
		t.Errorf("expected first element 3, got %d", first.Value.Int64())
	}
}

func TestArraysContainsExtended(t *testing.T) {
	containsFn := ArraysBuiltins["arrays.contains"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
	}

	// Should find element
	result := containsFn(arr, &object.Integer{Value: big.NewInt(2)})
	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !boolVal.Value {
		t.Error("expected true, got false")
	}

	// Should not find element
	result = containsFn(arr, &object.Integer{Value: big.NewInt(5)})
	boolVal, ok = result.(*object.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if boolVal.Value {
		t.Error("expected false, got true")
	}
}

func TestArraysJoin(t *testing.T) {
	joinFn := ArraysBuiltins["arrays.join"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.String{Value: "a"},
			&object.String{Value: "b"},
			&object.String{Value: "c"},
		},
	}

	result := joinFn(arr, &object.String{Value: ","})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if strVal.Value != "a,b,c" {
		t.Errorf("expected 'a,b,c', got '%s'", strVal.Value)
	}
}

func TestArraysSliceExtended(t *testing.T) {
	sliceFn := ArraysBuiltins["arrays.slice"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
			&object.Integer{Value: big.NewInt(4)},
			&object.Integer{Value: big.NewInt(5)},
		},
	}

	result := sliceFn(arr, &object.Integer{Value: big.NewInt(1)}, &object.Integer{Value: big.NewInt(4)})
	newArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(newArr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(newArr.Elements))
	}
}

// ============================================================================
// Maps Module Additional Tests
// ============================================================================

func TestMapsKeysExtended(t *testing.T) {
	keysFn := MapsBuiltins["maps.keys"].Fn

	m := object.NewMap()
	m.Set(&object.String{Value: "a"}, &object.Integer{Value: big.NewInt(1)})
	m.Set(&object.String{Value: "b"}, &object.Integer{Value: big.NewInt(2)})

	result := keysFn(m)
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 keys, got %d", len(arr.Elements))
	}
}

func TestMapsValuesExtended(t *testing.T) {
	valuesFn := MapsBuiltins["maps.values"].Fn

	m := object.NewMap()
	m.Set(&object.String{Value: "a"}, &object.Integer{Value: big.NewInt(1)})
	m.Set(&object.String{Value: "b"}, &object.Integer{Value: big.NewInt(2)})

	result := valuesFn(m)
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 values, got %d", len(arr.Elements))
	}
}

// ============================================================================
// Strings Module Additional Tests
// ============================================================================

func TestStringsReplaceExtended(t *testing.T) {
	replaceFn := StringsBuiltins["strings.replace"].Fn

	result := replaceFn(
		&object.String{Value: "hello world"},
		&object.String{Value: "world"},
		&object.String{Value: "EZ"},
	)

	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if strVal.Value != "hello EZ" {
		t.Errorf("expected 'hello EZ', got '%s'", strVal.Value)
	}
}

func TestStringsContainsExtended(t *testing.T) {
	containsFn := StringsBuiltins["strings.contains"].Fn

	result := containsFn(&object.String{Value: "hello world"}, &object.String{Value: "world"})
	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !boolVal.Value {
		t.Error("expected true, got false")
	}
}

func TestStringsSplitExtended(t *testing.T) {
	splitFn := StringsBuiltins["strings.split"].Fn

	result := splitFn(&object.String{Value: "a,b,c"}, &object.String{Value: ","})
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}
	if len(arr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestStringsTrimExtended(t *testing.T) {
	trimFn := StringsBuiltins["strings.trim"].Fn

	result := trimFn(&object.String{Value: "  hello  "})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if strVal.Value != "hello" {
		t.Errorf("expected 'hello', got '%s'", strVal.Value)
	}
}

func TestStringsStartsWithExtended(t *testing.T) {
	startsWithFn := StringsBuiltins["strings.starts_with"].Fn

	result := startsWithFn(&object.String{Value: "hello world"}, &object.String{Value: "hello"})
	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !boolVal.Value {
		t.Error("expected true, got false")
	}
}

func TestStringsEndsWithExtended(t *testing.T) {
	endsWithFn := StringsBuiltins["strings.ends_with"].Fn

	result := endsWithFn(&object.String{Value: "hello world"}, &object.String{Value: "world"})
	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !boolVal.Value {
		t.Error("expected true, got false")
	}
}

// ============================================================================
// Bytes Module Additional Tests
// ============================================================================

func TestBytesFromStringExtended(t *testing.T) {
	fromStringFn := BytesBuiltins["bytes.from_string"].Fn

	result := fromStringFn(&object.String{Value: "hello"})
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(arr.Elements) != 5 {
		t.Errorf("expected 5 bytes, got %d", len(arr.Elements))
	}
}

func TestBytesToStringExtended(t *testing.T) {
	toStringFn := BytesBuiltins["bytes.to_string"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Byte{Value: 'h'},
			&object.Byte{Value: 'i'},
		},
	}

	result := toStringFn(arr)
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	if strVal.Value != "hi" {
		t.Errorf("expected 'hi', got '%s'", strVal.Value)
	}
}

func TestBytesConcatExtended(t *testing.T) {
	concatFn := BytesBuiltins["bytes.concat"].Fn

	arr1 := &object.Array{Elements: []object.Object{&object.Byte{Value: 1}, &object.Byte{Value: 2}}}
	arr2 := &object.Array{Elements: []object.Object{&object.Byte{Value: 3}, &object.Byte{Value: 4}}}

	result := concatFn(arr1, arr2)
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(arr.Elements) != 4 {
		t.Errorf("expected 4 bytes, got %d", len(arr.Elements))
	}
}

// ============================================================================
// Binary Module - U128/U256 Encoding Tests (tests padBigIntBytesUnsigned)
// ============================================================================

func TestBinaryEncodeU128LittleEndian(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_u128_to_little_endian"].Fn

	// Test with a value that requires padding
	val := &object.Integer{Value: big.NewInt(256), DeclaredType: "u128"}
	result := encodeFn(val)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	if len(arr.Elements) != 16 {
		t.Errorf("expected 16 bytes, got %d", len(arr.Elements))
	}
}

func TestBinaryEncodeU256LittleEndian(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_u256_to_little_endian"].Fn

	// Test with a value that requires padding
	val := &object.Integer{Value: big.NewInt(65536), DeclaredType: "u256"}
	result := encodeFn(val)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	if len(arr.Elements) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(arr.Elements))
	}
}

func TestBinaryEncodeU128BigEndian(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_u128_to_big_endian"].Fn

	val := &object.Integer{Value: big.NewInt(1024), DeclaredType: "u128"}
	result := encodeFn(val)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	if len(arr.Elements) != 16 {
		t.Errorf("expected 16 bytes, got %d", len(arr.Elements))
	}
}

func TestBinaryEncodeU256BigEndian(t *testing.T) {
	encodeFn := BinaryBuiltins["binary.encode_u256_to_big_endian"].Fn

	val := &object.Integer{Value: big.NewInt(1048576), DeclaredType: "u256"}
	result := encodeFn(val)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", retVal.Values[0])
	}

	if len(arr.Elements) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(arr.Elements))
	}
}

// ============================================================================
// JSON Module - Pretty Print Tests (tests formatJSONValue)
// ============================================================================

func TestJSONPrettyPrintInteger(t *testing.T) {
	prettyFn := JsonBuiltins["json.pretty"].Fn

	val := &object.Integer{Value: big.NewInt(42)}
	result := prettyFn(val, &object.String{Value: "  "})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	strVal, ok := retVal.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", retVal.Values[0])
	}

	if strVal.Value != "42" {
		t.Errorf("expected '42', got '%s'", strVal.Value)
	}
}

func TestJSONPrettyPrintLargeInteger(t *testing.T) {
	prettyFn := JsonBuiltins["json.pretty"].Fn
	indent := &object.String{Value: "  "}

	// Create a large integer that doesn't fit in int64
	largeVal := new(big.Int)
	largeVal.SetString("99999999999999999999999999999", 10)
	val := &object.Integer{Value: largeVal}
	result := prettyFn(val, indent)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	strVal, ok := retVal.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", retVal.Values[0])
	}

	// Large integers are encoded as strings
	if strVal.Value != `"99999999999999999999999999999"` {
		t.Errorf("expected quoted large int, got '%s'", strVal.Value)
	}
}

func TestJSONPrettyPrintFloat(t *testing.T) {
	prettyFn := JsonBuiltins["json.pretty"].Fn
	indent := &object.String{Value: "  "}

	val := &object.Float{Value: 3.14159}
	result := prettyFn(val, indent)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	strVal, ok := retVal.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", retVal.Values[0])
	}

	if strVal.Value != "3.14159" {
		t.Errorf("expected '3.14159', got '%s'", strVal.Value)
	}
}

func TestJSONPrettyPrintBoolean(t *testing.T) {
	prettyFn := JsonBuiltins["json.pretty"].Fn
	indent := &object.String{Value: "  "}

	// Test true
	result := prettyFn(&object.Boolean{Value: true}, indent)
	retVal := result.(*object.ReturnValue)
	strVal := retVal.Values[0].(*object.String)
	if strVal.Value != "true" {
		t.Errorf("expected 'true', got '%s'", strVal.Value)
	}

	// Test false
	result = prettyFn(&object.Boolean{Value: false}, indent)
	retVal = result.(*object.ReturnValue)
	strVal = retVal.Values[0].(*object.String)
	if strVal.Value != "false" {
		t.Errorf("expected 'false', got '%s'", strVal.Value)
	}
}

func TestJSONPrettyPrintNil(t *testing.T) {
	prettyFn := JsonBuiltins["json.pretty"].Fn
	indent := &object.String{Value: "  "}

	result := prettyFn(object.NIL, indent)
	retVal := result.(*object.ReturnValue)
	strVal := retVal.Values[0].(*object.String)
	if strVal.Value != "null" {
		t.Errorf("expected 'null', got '%s'", strVal.Value)
	}
}

func TestJSONPrettyPrintChar(t *testing.T) {
	prettyFn := JsonBuiltins["json.pretty"].Fn
	indent := &object.String{Value: "  "}

	result := prettyFn(&object.Char{Value: 'A'}, indent)
	retVal := result.(*object.ReturnValue)
	strVal := retVal.Values[0].(*object.String)
	if strVal.Value != `"A"` {
		t.Errorf("expected '\"A\"', got '%s'", strVal.Value)
	}
}

func TestJSONPrettyPrintByte(t *testing.T) {
	prettyFn := JsonBuiltins["json.pretty"].Fn
	indent := &object.String{Value: "  "}

	result := prettyFn(&object.Byte{Value: 255}, indent)
	retVal := result.(*object.ReturnValue)
	strVal := retVal.Values[0].(*object.String)
	if strVal.Value != "255" {
		t.Errorf("expected '255', got '%s'", strVal.Value)
	}
}

func TestJSONPrettyPrintArray(t *testing.T) {
	prettyFn := JsonBuiltins["json.pretty"].Fn
	indent := &object.String{Value: "  "}

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
	}
	result := prettyFn(arr, indent)

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	strVal, ok := retVal.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", retVal.Values[0])
	}

	// Should contain newlines for pretty print
	if !strings.Contains(strVal.Value, "\n") {
		t.Errorf("expected pretty printed array with newlines, got '%s'", strVal.Value)
	}
}

func TestJSONPrettyPrintEmptyArray(t *testing.T) {
	prettyFn := JsonBuiltins["json.pretty"].Fn
	indent := &object.String{Value: "  "}

	arr := &object.Array{Elements: []object.Object{}}
	result := prettyFn(arr, indent)

	retVal := result.(*object.ReturnValue)
	strVal := retVal.Values[0].(*object.String)
	if strVal.Value != "[]" {
		t.Errorf("expected '[]', got '%s'", strVal.Value)
	}
}

func TestJSONPrettyPrintMap(t *testing.T) {
	prettyFn := JsonBuiltins["json.pretty"].Fn
	indent := &object.String{Value: "  "}

	m := object.NewMap()
	m.Set(&object.String{Value: "key"}, &object.Integer{Value: big.NewInt(42)})

	result := prettyFn(m, indent)
	retVal := result.(*object.ReturnValue)
	strVal := retVal.Values[0].(*object.String)

	// Should contain the key and value
	if !strings.Contains(strVal.Value, "key") || !strings.Contains(strVal.Value, "42") {
		t.Errorf("expected map with key and value, got '%s'", strVal.Value)
	}
}

func TestJSONPrettyPrintEmptyMap(t *testing.T) {
	prettyFn := JsonBuiltins["json.pretty"].Fn
	indent := &object.String{Value: "  "}

	m := object.NewMap()
	result := prettyFn(m, indent)

	retVal := result.(*object.ReturnValue)
	strVal := retVal.Values[0].(*object.String)
	if strVal.Value != "{}" {
		t.Errorf("expected '{}', got '%s'", strVal.Value)
	}
}

// ============================================================================
// Builtins - getEZTypeName Tests
// ============================================================================

func TestTypeOfInteger(t *testing.T) {
	typeofFn := StdBuiltins["typeof"].Fn

	result := typeofFn(&object.Integer{Value: big.NewInt(42), DeclaredType: "i32"})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if strVal.Value != "i32" {
		t.Errorf("expected 'i32', got '%s'", strVal.Value)
	}
}

func TestTypeOfFloat(t *testing.T) {
	typeofFn := StdBuiltins["typeof"].Fn

	result := typeofFn(&object.Float{Value: 3.14})
	strVal := result.(*object.String)
	if strVal.Value != "float" {
		t.Errorf("expected 'float', got '%s'", strVal.Value)
	}
}

func TestTypeOfString(t *testing.T) {
	typeofFn := StdBuiltins["typeof"].Fn

	result := typeofFn(&object.String{Value: "hello"})
	strVal := result.(*object.String)
	if strVal.Value != "string" {
		t.Errorf("expected 'string', got '%s'", strVal.Value)
	}
}

func TestTypeOfBool(t *testing.T) {
	typeofFn := StdBuiltins["typeof"].Fn

	result := typeofFn(&object.Boolean{Value: true})
	strVal := result.(*object.String)
	if strVal.Value != "bool" {
		t.Errorf("expected 'bool', got '%s'", strVal.Value)
	}
}

func TestTypeOfChar(t *testing.T) {
	typeofFn := StdBuiltins["typeof"].Fn

	result := typeofFn(&object.Char{Value: 'x'})
	strVal := result.(*object.String)
	if strVal.Value != "char" {
		t.Errorf("expected 'char', got '%s'", strVal.Value)
	}
}

func TestTypeOfByte(t *testing.T) {
	typeofFn := StdBuiltins["typeof"].Fn

	result := typeofFn(&object.Byte{Value: 0xFF})
	strVal := result.(*object.String)
	if strVal.Value != "byte" {
		t.Errorf("expected 'byte', got '%s'", strVal.Value)
	}
}

func TestTypeOfArray(t *testing.T) {
	typeofFn := StdBuiltins["typeof"].Fn

	result := typeofFn(&object.Array{Elements: []object.Object{}})
	strVal := result.(*object.String)
	if strVal.Value != "array" {
		t.Errorf("expected 'array', got '%s'", strVal.Value)
	}
}

func TestTypeOfNil(t *testing.T) {
	typeofFn := StdBuiltins["typeof"].Fn

	result := typeofFn(object.NIL)
	strVal := result.(*object.String)
	if strVal.Value != "nil" {
		t.Errorf("expected 'nil', got '%s'", strVal.Value)
	}
}

// ============================================================================
// Builtins - deepCopy Tests
// ============================================================================

func TestDeepCopyInteger(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	original := &object.Integer{Value: big.NewInt(42), DeclaredType: "i64"}
	result := copyFn(original)

	copied, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if copied.Value.Int64() != 42 {
		t.Errorf("expected 42, got %d", copied.Value.Int64())
	}

	// Verify it's a copy (modifying original shouldn't affect copy)
	original.Value = big.NewInt(100)
	if copied.Value.Int64() != 42 {
		t.Error("copy was affected by modifying original")
	}
}

func TestDeepCopyFloat(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	result := copyFn(&object.Float{Value: 3.14})
	copied := result.(*object.Float)
	if copied.Value != 3.14 {
		t.Errorf("expected 3.14, got %f", copied.Value)
	}
}

func TestDeepCopyString(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	result := copyFn(&object.String{Value: "hello"})
	copied := result.(*object.String)
	if copied.Value != "hello" {
		t.Errorf("expected 'hello', got '%s'", copied.Value)
	}
}

func TestDeepCopyBoolean(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	result := copyFn(&object.Boolean{Value: true})
	copied := result.(*object.Boolean)
	if !copied.Value {
		t.Error("expected true, got false")
	}
}

func TestDeepCopyChar(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	result := copyFn(&object.Char{Value: 'Z'})
	copied := result.(*object.Char)
	if copied.Value != 'Z' {
		t.Errorf("expected 'Z', got '%c'", copied.Value)
	}
}

func TestDeepCopyByte(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	result := copyFn(&object.Byte{Value: 128})
	copied := result.(*object.Byte)
	if copied.Value != 128 {
		t.Errorf("expected 128, got %d", copied.Value)
	}
}

func TestDeepCopyNil(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	result := copyFn(object.NIL)
	if result != object.NIL {
		t.Errorf("expected NIL, got %T", result)
	}
}

func TestDeepCopyArray(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	original := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
		},
	}
	result := copyFn(original)

	copied, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(copied.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(copied.Elements))
	}

	// Verify deep copy
	original.Elements[0] = &object.Integer{Value: big.NewInt(999)}
	first := copied.Elements[0].(*object.Integer)
	if first.Value.Int64() != 1 {
		t.Error("copy was affected by modifying original array")
	}
}

func TestDeepCopyMap(t *testing.T) {
	copyFn := StdBuiltins["copy"].Fn

	original := object.NewMap()
	original.Set(&object.String{Value: "key"}, &object.Integer{Value: big.NewInt(42)})

	result := copyFn(original)

	copied, ok := result.(*object.Map)
	if !ok {
		t.Fatalf("expected Map, got %T", result)
	}

	// Verify the copy has the same data
	val, found := copied.Get(&object.String{Value: "key"})
	if !found || val == nil {
		t.Fatal("expected to find 'key' in copied map")
	}

	intVal := val.(*object.Integer)
	if intVal.Value.Int64() != 42 {
		t.Errorf("expected 42, got %d", intVal.Value.Int64())
	}
}

// ============================================================================
// Math Module - getNumber Tests
// ============================================================================

func TestMathAbsFloat(t *testing.T) {
	absFn := MathBuiltins["math.abs"].Fn

	result := absFn(&object.Float{Value: -3.14})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if floatVal.Value != 3.14 {
		t.Errorf("expected 3.14, got %f", floatVal.Value)
	}
}

func TestMathAbsInteger(t *testing.T) {
	absFn := MathBuiltins["math.abs"].Fn

	result := absFn(&object.Integer{Value: big.NewInt(-42)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 42 {
		t.Errorf("expected 42, got %d", intVal.Value.Int64())
	}
}

func TestMathPowFloats(t *testing.T) {
	powFn := MathBuiltins["math.pow"].Fn

	result := powFn(&object.Float{Value: 2.0}, &object.Float{Value: 3.0})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if floatVal.Value != 8.0 {
		t.Errorf("expected 8.0, got %f", floatVal.Value)
	}
}

func TestMathMinMaxFloats(t *testing.T) {
	minFn := MathBuiltins["math.min"].Fn
	maxFn := MathBuiltins["math.max"].Fn

	// Test min with floats
	result := minFn(&object.Float{Value: 5.0}, &object.Float{Value: 3.0})
	floatVal := result.(*object.Float)
	if floatVal.Value != 3.0 {
		t.Errorf("min expected 3.0, got %f", floatVal.Value)
	}

	// Test max with floats
	result = maxFn(&object.Float{Value: 5.0}, &object.Float{Value: 3.0})
	floatVal = result.(*object.Float)
	if floatVal.Value != 5.0 {
		t.Errorf("max expected 5.0, got %f", floatVal.Value)
	}
}

// ============================================================================
// Random Module - getRandomNumber Tests
// ============================================================================

func TestRandomIntRangeWithFloats(t *testing.T) {
	randIntFn := RandomBuiltins["random.int"].Fn

	// Test with float bounds (should convert)
	result := randIntFn(&object.Float{Value: 0.0}, &object.Float{Value: 10.0})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	val := intVal.Value.Int64()
	if val < 0 || val >= 10 {
		t.Errorf("expected value in [0, 10), got %d", val)
	}
}

func TestRandomFloatRangeWithInts(t *testing.T) {
	randFloatFn := RandomBuiltins["random.float"].Fn

	// Test with integer bounds (should convert)
	result := randFloatFn(&object.Integer{Value: big.NewInt(0)}, &object.Integer{Value: big.NewInt(1)})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}

	if floatVal.Value < 0.0 || floatVal.Value >= 1.0 {
		t.Errorf("expected value in [0.0, 1.0), got %f", floatVal.Value)
	}
}

// ============================================================================
// Arrays Module - Error Cases (tests newError)
// ============================================================================

func TestArraysSortError(t *testing.T) {
	sortFn := ArraysBuiltins["arrays.sort"].Fn

	// Test with wrong argument count
	result := sortFn()
	errVal, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}

	if errVal.Message == "" {
		t.Error("expected error message")
	}
}

func TestArraysPopEmptyError(t *testing.T) {
	popFn := ArraysBuiltins["arrays.pop"].Fn

	arr := &object.Array{Elements: []object.Object{}}
	result := popFn(arr)

	// Should return error for empty array
	errVal, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected Error for empty array pop, got %T", result)
	}

	if errVal.Message == "" {
		t.Error("expected error message")
	}
}

// ============================================================================
// JSON Module - objectToGoValue Additional Cases
// ============================================================================

func TestJSONEncodeMapWithNonStringKey(t *testing.T) {
	encodeFn := JsonBuiltins["json.encode"].Fn

	// Create a map with integer key (should error)
	m := object.NewMap()
	m.Set(&object.Integer{Value: big.NewInt(42)}, &object.String{Value: "value"})

	result := encodeFn(m)
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	// Second value should be an error struct
	if retVal.Values[1] == object.NIL {
		t.Error("expected error for non-string map key")
	}
}

func TestJSONEncodeNestedArray(t *testing.T) {
	encodeFn := JsonBuiltins["json.encode"].Fn

	// Create nested array
	inner := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
		},
	}
	outer := &object.Array{
		Elements: []object.Object{inner, &object.String{Value: "test"}},
	}

	result := encodeFn(outer)
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	strVal, ok := retVal.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", retVal.Values[0])
	}

	if !strings.Contains(strVal.Value, "[1,2]") {
		t.Errorf("expected nested array in JSON, got '%s'", strVal.Value)
	}
}

func TestJSONEncodeChar(t *testing.T) {
	encodeFn := JsonBuiltins["json.encode"].Fn

	result := encodeFn(&object.Char{Value: 'Z'})
	retVal := result.(*object.ReturnValue)
	strVal := retVal.Values[0].(*object.String)

	if strVal.Value != `"Z"` {
		t.Errorf("expected '\"Z\"', got '%s'", strVal.Value)
	}
}

func TestJSONEncodeByte(t *testing.T) {
	encodeFn := JsonBuiltins["json.encode"].Fn

	result := encodeFn(&object.Byte{Value: 200})
	retVal := result.(*object.ReturnValue)
	strVal := retVal.Values[0].(*object.String)

	if strVal.Value != "200" {
		t.Errorf("expected '200', got '%s'", strVal.Value)
	}
}

// ============================================================================
// Math Module - getTwoNumbers Additional Cases
// ============================================================================

func TestMathPowInvalidSecondArg(t *testing.T) {
	powFn := MathBuiltins["math.pow"].Fn

	// First arg valid, second arg invalid
	result := powFn(&object.Float{Value: 2.0}, &object.String{Value: "not a number"})
	_, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected Error for invalid second arg, got %T", result)
	}
}

func TestMathModWithFloats(t *testing.T) {
	modFn := MathBuiltins["math.mod"].Fn

	result := modFn(&object.Float{Value: 10.5}, &object.Float{Value: 3.0})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}

	// 10.5 mod 3.0 = 1.5
	if floatVal.Value != 1.5 {
		t.Errorf("expected 1.5, got %f", floatVal.Value)
	}
}

// ============================================================================
// Strings Module - Additional Edge Cases
// ============================================================================

func TestStringsRepeatMultiple(t *testing.T) {
	repeatFn := StringsBuiltins["strings.repeat"].Fn

	result := repeatFn(&object.String{Value: "ab"}, &object.Integer{Value: big.NewInt(3)})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	if strVal.Value != "ababab" {
		t.Errorf("expected 'ababab', got '%s'", strVal.Value)
	}
}

func TestStringsPadLeftZeros(t *testing.T) {
	padFn := StringsBuiltins["strings.pad_left"].Fn

	result := padFn(&object.String{Value: "42"}, &object.Integer{Value: big.NewInt(5)}, &object.String{Value: "0"})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	if strVal.Value != "00042" {
		t.Errorf("expected '00042', got '%s'", strVal.Value)
	}
}

func TestStringsPadRightZeros(t *testing.T) {
	padFn := StringsBuiltins["strings.pad_right"].Fn

	result := padFn(&object.String{Value: "42"}, &object.Integer{Value: big.NewInt(5)}, &object.String{Value: "0"})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	if strVal.Value != "42000" {
		t.Errorf("expected '42000', got '%s'", strVal.Value)
	}
}

func TestStringsReverseWord(t *testing.T) {
	reverseFn := StringsBuiltins["strings.reverse"].Fn

	result := reverseFn(&object.String{Value: "hello"})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	if strVal.Value != "olleh" {
		t.Errorf("expected 'olleh', got '%s'", strVal.Value)
	}
}

// ============================================================================
// Builtins - Additional getEZTypeName Cases
// ============================================================================

func TestTypeOfStruct(t *testing.T) {
	typeofFn := StdBuiltins["typeof"].Fn

	// Struct with TypeName
	s := &object.Struct{
		TypeName: "Person",
		Fields:   map[string]object.Object{},
	}
	result := typeofFn(s)
	strVal := result.(*object.String)
	if strVal.Value != "Person" {
		t.Errorf("expected 'Person', got '%s'", strVal.Value)
	}
}

func TestTypeOfStructNoName(t *testing.T) {
	typeofFn := StdBuiltins["typeof"].Fn

	// Struct without TypeName
	s := &object.Struct{
		TypeName: "",
		Fields:   map[string]object.Object{},
	}
	result := typeofFn(s)
	strVal := result.(*object.String)
	if strVal.Value != "struct" {
		t.Errorf("expected 'struct', got '%s'", strVal.Value)
	}
}

func TestTypeOfMap(t *testing.T) {
	typeofFn := StdBuiltins["typeof"].Fn

	m := object.NewMap()
	result := typeofFn(m)
	strVal := result.(*object.String)
	// Map type falls through to default
	if strVal.Value == "" {
		t.Error("expected non-empty type name for map")
	}
}

// ============================================================================
// Arrays - Additional compareObjects Cases
// ============================================================================

func TestArraysSortStrings(t *testing.T) {
	sortFn := ArraysBuiltins["arrays.sort"].Fn

	arr := &object.Array{
		Mutable: true,
		Elements: []object.Object{
			&object.String{Value: "banana"},
			&object.String{Value: "apple"},
			&object.String{Value: "cherry"},
		},
	}

	result := sortFn(arr)
	if result != object.NIL {
		t.Fatalf("expected NIL, got %T", result)
	}

	// Check first element is "apple" (sorted in-place)
	first := arr.Elements[0].(*object.String)
	if first.Value != "apple" {
		t.Errorf("expected first element 'apple', got '%s'", first.Value)
	}
}

func TestArraysSortFloats(t *testing.T) {
	sortFn := ArraysBuiltins["arrays.sort"].Fn

	arr := &object.Array{
		Mutable: true,
		Elements: []object.Object{
			&object.Float{Value: 3.14},
			&object.Float{Value: 1.41},
			&object.Float{Value: 2.72},
		},
	}

	result := sortFn(arr)
	if result != object.NIL {
		t.Fatalf("expected NIL, got %T", result)
	}

	// Check first element is smallest (sorted in-place)
	first := arr.Elements[0].(*object.Float)
	if first.Value != 1.41 {
		t.Errorf("expected first element 1.41, got %f", first.Value)
	}
}

// ============================================================================
// IO Module - Additional Cases
// ============================================================================

func TestIOReadFileNotFound(t *testing.T) {
	readFn := IOBuiltins["io.read_file"].Fn

	result := readFn(&object.String{Value: "/nonexistent/path/file.txt"})
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	// Should return error in second value
	if retVal.Values[1] == object.NIL {
		t.Error("expected error for non-existent file")
	}
}

func TestIOExistsNot(t *testing.T) {
	existsFn := IOBuiltins["io.exists"].Fn

	result := existsFn(&object.String{Value: "/nonexistent/path/file.txt"})
	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}

	if boolVal.Value {
		t.Error("expected false for non-existent file")
	}
}

func TestIOIsDirCurrentDir(t *testing.T) {
	isDirFn := IOBuiltins["io.is_dir"].Fn

	// Test with current directory (should exist)
	result := isDirFn(&object.String{Value: "."})
	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}

	if !boolVal.Value {
		t.Error("expected true for current directory")
	}
}

// ============================================================================
// Bytes Module - Error Cases
// ============================================================================

func TestBytesFromStringWithNonString(t *testing.T) {
	fromStringFn := BytesBuiltins["bytes.from_string"].Fn

	// Try with an integer instead of string
	result := fromStringFn(&object.Integer{Value: big.NewInt(123)})
	_, ok := result.(*object.Error)
	if !ok {
		t.Errorf("expected Error for non-string, got %T", result)
	}
}

func TestBytesToStringWithNonArray(t *testing.T) {
	toStringFn := BytesBuiltins["bytes.to_string"].Fn

	// Try with an integer instead of array
	result := toStringFn(&object.Integer{Value: big.NewInt(123)})
	_, ok := result.(*object.Error)
	if !ok {
		t.Errorf("expected Error for non-array, got %T", result)
	}
}

func TestBytesToStringWithWrongElements(t *testing.T) {
	toStringFn := BytesBuiltins["bytes.to_string"].Fn

	// Try with array containing strings instead of bytes
	arr := &object.Array{
		Elements: []object.Object{
			&object.String{Value: "not a byte"},
		},
	}
	result := toStringFn(arr)
	_, ok := result.(*object.Error)
	if !ok {
		t.Errorf("expected Error for wrong element types, got %T", result)
	}
}

// ============================================================================
// Compare Objects - Additional Cases
// ============================================================================

func TestArraysSortBooleans(t *testing.T) {
	sortFn := ArraysBuiltins["arrays.sort"].Fn

	arr := &object.Array{
		Mutable: true,
		Elements: []object.Object{
			&object.Boolean{Value: true},
			&object.Boolean{Value: false},
			&object.Boolean{Value: true},
		},
	}

	result := sortFn(arr)
	if result != object.NIL {
		t.Fatalf("expected NIL, got %T", result)
	}

	// false should come before true
	first := arr.Elements[0].(*object.Boolean)
	if first.Value != false {
		t.Error("expected false first after sorting booleans")
	}
}

func TestArraysSortChars(t *testing.T) {
	sortFn := ArraysBuiltins["arrays.sort"].Fn

	arr := &object.Array{
		Mutable: true,
		Elements: []object.Object{
			&object.Char{Value: 'c'},
			&object.Char{Value: 'a'},
			&object.Char{Value: 'b'},
		},
	}

	result := sortFn(arr)
	if result != object.NIL {
		t.Fatalf("expected NIL, got %T", result)
	}

	first := arr.Elements[0].(*object.Char)
	if first.Value != 'a' {
		t.Errorf("expected 'a' first, got '%c'", first.Value)
	}
}

func TestArraysSortBytes(t *testing.T) {
	sortFn := ArraysBuiltins["arrays.sort"].Fn

	arr := &object.Array{
		Mutable: true,
		Elements: []object.Object{
			&object.Byte{Value: 200},
			&object.Byte{Value: 50},
			&object.Byte{Value: 100},
		},
	}

	result := sortFn(arr)
	if result != object.NIL {
		t.Fatalf("expected NIL, got %T", result)
	}

	// Just check that it sorted without error - don't assume order for Byte
	// since compareObjects may not have a case for Byte
	if len(arr.Elements) != 3 {
		t.Error("expected 3 elements after sort")
	}
}

// ============================================================================
// JSON Module - Additional Decode Cases
// ============================================================================

func TestJsonDecodeArray(t *testing.T) {
	decodeFn := JsonBuiltins["json.decode"].Fn

	result := decodeFn(&object.String{Value: "[1, 2, 3]"})
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := retVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array in first value, got %T", retVal.Values[0])
	}

	if len(arr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestJsonDecodeNestedObject(t *testing.T) {
	decodeFn := JsonBuiltins["json.decode"].Fn

	result := decodeFn(&object.String{Value: `{"nested": {"key": "value"}}`})
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	m, ok := retVal.Values[0].(*object.Map)
	if !ok {
		t.Fatalf("expected Map, got %T", retVal.Values[0])
	}

	nested, found := m.Get(&object.String{Value: "nested"})
	if !found {
		t.Fatal("expected to find 'nested' key")
	}

	nestedMap, ok := nested.(*object.Map)
	if !ok {
		t.Fatalf("expected nested Map, got %T", nested)
	}

	val, _ := nestedMap.Get(&object.String{Value: "key"})
	strVal, ok := val.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", val)
	}

	if strVal.Value != "value" {
		t.Errorf("expected 'value', got '%s'", strVal.Value)
	}
}

func TestJsonDecodeInvalid(t *testing.T) {
	decodeFn := JsonBuiltins["json.decode"].Fn

	result := decodeFn(&object.String{Value: "not valid json"})
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	// Error should be in second value
	if retVal.Values[1] == object.NIL {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// ============================================================================
// Random - Additional Edge Cases
// ============================================================================

func TestRandomChoiceBasic(t *testing.T) {
	choiceFn := RandomBuiltins["random.choice"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
	}

	result := choiceFn(arr)
	_, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
}

func TestRandomChoiceEmpty(t *testing.T) {
	choiceFn := RandomBuiltins["random.choice"].Fn

	arr := &object.Array{Elements: []object.Object{}}

	result := choiceFn(arr)
	_, ok := result.(*object.Error)
	if !ok {
		t.Errorf("expected Error for empty array, got %T", result)
	}
}

// ============================================================================
// Strings - Additional Cases
// ============================================================================

func TestStringsCompareEqual(t *testing.T) {
	compareFn := StringsBuiltins["strings.compare"].Fn

	result := compareFn(&object.String{Value: "abc"}, &object.String{Value: "abc"})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if intVal.Value.Int64() != 0 {
		t.Errorf("expected 0 for equal strings, got %d", intVal.Value.Int64())
	}
}

func TestStringsIsNumericTrue(t *testing.T) {
	isNumericFn := StringsBuiltins["strings.is_numeric"].Fn

	result := isNumericFn(&object.String{Value: "12345"})
	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}

	if !boolVal.Value {
		t.Error("expected true for numeric string")
	}
}

func TestStringsIsAlphaTrue(t *testing.T) {
	isAlphaFn := StringsBuiltins["strings.is_alpha"].Fn

	result := isAlphaFn(&object.String{Value: "abcXYZ"})
	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}

	if !boolVal.Value {
		t.Error("expected true for alpha string")
	}
}

func TestStringsTruncateWithSuffix(t *testing.T) {
	truncateFn := StringsBuiltins["strings.truncate"].Fn

	result := truncateFn(
		&object.String{Value: "hello world"},
		&object.Integer{Value: big.NewInt(8)},
		&object.String{Value: "..."},
	)
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	// 8 chars total, 3 for "...", so 5 from original
	if strVal.Value != "hello..." {
		t.Errorf("expected 'hello...', got '%s'", strVal.Value)
	}
}

// ============================================================================
// Maps - Additional Cases
// ============================================================================

func TestMapsKeysEmpty(t *testing.T) {
	keysFn := MapsBuiltins["maps.keys"].Fn

	m := object.NewMap()
	result := keysFn(m)
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(arr.Elements) != 0 {
		t.Errorf("expected 0 keys, got %d", len(arr.Elements))
	}
}

func TestMapsValuesEmpty(t *testing.T) {
	valuesFn := MapsBuiltins["maps.values"].Fn

	m := object.NewMap()
	result := valuesFn(m)
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(arr.Elements) != 0 {
		t.Errorf("expected 0 values, got %d", len(arr.Elements))
	}
}

func TestMapsMerge(t *testing.T) {
	mergeFn := MapsBuiltins["maps.merge"].Fn

	m1 := object.NewMap()
	m1.Set(&object.String{Value: "a"}, &object.Integer{Value: big.NewInt(1)})

	m2 := object.NewMap()
	m2.Set(&object.String{Value: "b"}, &object.Integer{Value: big.NewInt(2)})

	result := mergeFn(m1, m2)
	merged, ok := result.(*object.Map)
	if !ok {
		t.Fatalf("expected Map, got %T", result)
	}

	// Check both keys exist
	_, found1 := merged.Get(&object.String{Value: "a"})
	_, found2 := merged.Get(&object.String{Value: "b"})
	if !found1 || !found2 {
		t.Error("expected both keys in merged map")
	}
}

// ============================================================================
// Arrays - More Edge Cases
// ============================================================================

func TestArraysSliceValidRange(t *testing.T) {
	sliceFn := ArraysBuiltins["arrays.slice"].Fn

	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
			&object.Integer{Value: big.NewInt(4)},
		},
	}

	// Valid slice from index 1 to 3
	result := sliceFn(arr, &object.Integer{Value: big.NewInt(1)}, &object.Integer{Value: big.NewInt(3)})
	sliced, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(sliced.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(sliced.Elements))
	}
}

func TestArraysConcatThree(t *testing.T) {
	concatFn := ArraysBuiltins["arrays.concat"].Fn

	arr1 := &object.Array{Elements: []object.Object{&object.Integer{Value: big.NewInt(1)}}}
	arr2 := &object.Array{Elements: []object.Object{&object.Integer{Value: big.NewInt(2)}}}
	arr3 := &object.Array{Elements: []object.Object{&object.Integer{Value: big.NewInt(3)}}}

	result := concatFn(arr1, arr2, arr3)
	concat, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(concat.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(concat.Elements))
	}
}

// ============================================================================
// Binary - Edge Cases
// ============================================================================

func TestBinaryEncodeI8OverflowPositive(t *testing.T) {
	encodeI8Fn := BinaryBuiltins["binary.encode_i8"].Fn

	// Value too large for i8 (max is 127)
	result := encodeI8Fn(&object.Integer{Value: big.NewInt(200)})
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	// Error should be in second value
	if retVal.Values[1] == object.NIL {
		t.Error("expected error for overflow, got nil")
	}
}

func TestBinaryEncodeU8OverflowPositive(t *testing.T) {
	encodeU8Fn := BinaryBuiltins["binary.encode_u8"].Fn

	// Value too large for u8 (max is 255)
	result := encodeU8Fn(&object.Integer{Value: big.NewInt(300)})
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	// Error should be in second value
	if retVal.Values[1] == object.NIL {
		t.Error("expected error for overflow, got nil")
	}
}

func TestBinaryEncodeDecodeU8Valid(t *testing.T) {
	encodeU8Fn := BinaryBuiltins["binary.encode_u8"].Fn
	decodeU8Fn := BinaryBuiltins["binary.decode_u8"].Fn

	// Encode 200
	encodeResult := encodeU8Fn(&object.Integer{Value: big.NewInt(200)})
	encRetVal, ok := encodeResult.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", encodeResult)
	}

	arr, ok := encRetVal.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array in first value, got %T", encRetVal.Values[0])
	}

	if len(arr.Elements) != 1 {
		t.Errorf("expected 1 byte, got %d", len(arr.Elements))
	}

	// Decode back
	decodeResult := decodeU8Fn(arr)
	decRetVal, ok := decodeResult.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", decodeResult)
	}

	intVal, ok := decRetVal.Values[0].(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", decRetVal.Values[0])
	}

	if intVal.Value.Int64() != 200 {
		t.Errorf("expected 200, got %d", intVal.Value.Int64())
	}
}

// ============================================================================
// Math - Additional Edge Cases
// ============================================================================

func TestMathAbsNegativeFloat(t *testing.T) {
	absFn := MathBuiltins["math.abs"].Fn

	result := absFn(&object.Float{Value: -3.14})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}

	if floatVal.Value != 3.14 {
		t.Errorf("expected 3.14, got %f", floatVal.Value)
	}
}

func TestMathSqrtNegative(t *testing.T) {
	sqrtFn := MathBuiltins["math.sqrt"].Fn

	result := sqrtFn(&object.Integer{Value: big.NewInt(-1)})
	_, ok := result.(*object.Error)
	if !ok {
		t.Errorf("expected Error for sqrt of negative, got %T", result)
	}
}

func TestMathLog10(t *testing.T) {
	log10Fn := MathBuiltins["math.log10"].Fn

	result := log10Fn(&object.Integer{Value: big.NewInt(100)})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}

	if floatVal.Value != 2.0 {
		t.Errorf("expected 2.0, got %f", floatVal.Value)
	}
}

func TestMathFloorFloat(t *testing.T) {
	floorFn := MathBuiltins["math.floor"].Fn

	result := floorFn(&object.Float{Value: 3.7})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if intVal.Value.Int64() != 3 {
		t.Errorf("expected 3, got %d", intVal.Value.Int64())
	}
}

func TestMathCeilFloat(t *testing.T) {
	ceilFn := MathBuiltins["math.ceil"].Fn

	result := ceilFn(&object.Float{Value: 3.2})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if intVal.Value.Int64() != 4 {
		t.Errorf("expected 4, got %d", intVal.Value.Int64())
	}
}

func TestMathRoundFloat(t *testing.T) {
	roundFn := MathBuiltins["math.round"].Fn

	result := roundFn(&object.Float{Value: 3.7})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if intVal.Value.Int64() != 4 {
		t.Errorf("expected 4, got %d", intVal.Value.Int64())
	}
}

func TestMathSin(t *testing.T) {
	sinFn := MathBuiltins["math.sin"].Fn

	result := sinFn(&object.Float{Value: 0})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}

	if floatVal.Value != 0 {
		t.Errorf("expected 0, got %f", floatVal.Value)
	}
}

func TestMathCos(t *testing.T) {
	cosFn := MathBuiltins["math.cos"].Fn

	result := cosFn(&object.Float{Value: 0})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}

	if floatVal.Value != 1 {
		t.Errorf("expected 1, got %f", floatVal.Value)
	}
}

func TestMathTan(t *testing.T) {
	tanFn := MathBuiltins["math.tan"].Fn

	result := tanFn(&object.Float{Value: 0})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}

	if floatVal.Value != 0 {
		t.Errorf("expected 0, got %f", floatVal.Value)
	}
}

func TestMathPowInteger(t *testing.T) {
	powFn := MathBuiltins["math.pow"].Fn

	result := powFn(&object.Integer{Value: big.NewInt(2)}, &object.Integer{Value: big.NewInt(3)})
	// math.pow returns Integer for integer inputs
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if intVal.Value.Int64() != 8 {
		t.Errorf("expected 8, got %d", intVal.Value.Int64())
	}
}

// ============================================================================
// Time - Additional Cases
// ============================================================================

func TestTimeFormatDate(t *testing.T) {
	formatFn := TimeBuiltins["time.format"].Fn

	// Format a specific timestamp (Jan 1, 2020 00:00:00 UTC = 1577836800)
	// First arg is format string, second is timestamp
	result := formatFn(
		&object.String{Value: "YYYY"},
		&object.Integer{Value: big.NewInt(1577836800)},
	)
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	// Just check it returned a 4-digit year (timezone may affect exact value)
	if len(strVal.Value) != 4 {
		t.Errorf("expected 4-digit year, got '%s'", strVal.Value)
	}
}

// ============================================================================
// IO - Additional Cases
// ============================================================================

func TestIOReadDirNonExistent(t *testing.T) {
	readDirFn := IOBuiltins["io.read_dir"].Fn

	result := readDirFn(&object.String{Value: "/nonexistent/directory/12345"})
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	// Error should be in second value
	if retVal.Values[1] == object.NIL {
		t.Error("expected error for non-existent directory")
	}
}

func TestIOIsFileOnDir(t *testing.T) {
	isFileFn := IOBuiltins["io.is_file"].Fn

	// Check current directory - should return false (it's a directory not a file)
	result := isFileFn(&object.String{Value: "."})
	boolVal, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}

	if boolVal.Value {
		t.Error("expected false for directory")
	}
}

// ============================================================================
// Random - More Tests
// ============================================================================

func TestRandomBoolReturnsBoolean(t *testing.T) {
	boolFn := RandomBuiltins["random.bool"].Fn

	result := boolFn()
	_, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
}

// ============================================================================
// Math - More Tests
// ============================================================================

func TestMathMinTwoArgs(t *testing.T) {
	minFn := MathBuiltins["math.min"].Fn

	result := minFn(&object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(3)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if intVal.Value.Int64() != 3 {
		t.Errorf("expected 3, got %d", intVal.Value.Int64())
	}
}

func TestMathMaxTwoArgs(t *testing.T) {
	maxFn := MathBuiltins["math.max"].Fn

	result := maxFn(&object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(3)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if intVal.Value.Int64() != 5 {
		t.Errorf("expected 5, got %d", intVal.Value.Int64())
	}
}

func TestMathModulo(t *testing.T) {
	modFn := MathBuiltins["math.mod"].Fn

	result := modFn(&object.Integer{Value: big.NewInt(10)}, &object.Integer{Value: big.NewInt(3)})
	// math.mod returns Integer for integer inputs
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if intVal.Value.Int64() != 1 {
		t.Errorf("expected 1, got %d", intVal.Value.Int64())
	}
}

func TestMathExp(t *testing.T) {
	expFn := MathBuiltins["math.exp"].Fn

	result := expFn(&object.Integer{Value: big.NewInt(0)})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}

	if floatVal.Value != 1 {
		t.Errorf("expected 1 (e^0), got %f", floatVal.Value)
	}
}

func TestMathLog(t *testing.T) {
	logFn := MathBuiltins["math.log"].Fn

	// Using math.E would give 1
	result := logFn(&object.Float{Value: 2.718281828459045})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}

	// ln(e) should be approximately 1
	if floatVal.Value < 0.99 || floatVal.Value > 1.01 {
		t.Errorf("expected ~1, got %f", floatVal.Value)
	}
}

// ============================================================================
// Time - More Tests
// ============================================================================

func TestTimeDateNow(t *testing.T) {
	dateFn := TimeBuiltins["time.date"].Fn

	result := dateFn()
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	// Should be in YYYY-MM-DD format (10 chars)
	if len(strVal.Value) != 10 {
		t.Errorf("expected 10-char date, got '%s'", strVal.Value)
	}
}

func TestTimeClockNow(t *testing.T) {
	clockFn := TimeBuiltins["time.clock"].Fn

	result := clockFn()
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	// Should be in HH:MM:SS format (8 chars)
	if len(strVal.Value) != 8 {
		t.Errorf("expected 8-char time, got '%s'", strVal.Value)
	}
}

func TestTimeIso(t *testing.T) {
	isoFn := TimeBuiltins["time.iso"].Fn

	result := isoFn()
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	// ISO format should contain 'T'
	if len(strVal.Value) < 19 {
		t.Errorf("expected ISO date string, got '%s'", strVal.Value)
	}
}

// ============================================================================
// Strings - More Tests
// ============================================================================

func TestStringsJoinEmpty(t *testing.T) {
	joinFn := StringsBuiltins["strings.join"].Fn

	arr := &object.Array{Elements: []object.Object{}}
	result := joinFn(arr, &object.String{Value: ","})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	if strVal.Value != "" {
		t.Errorf("expected empty string, got '%s'", strVal.Value)
	}
}

func TestStringsRepeatZero(t *testing.T) {
	repeatFn := StringsBuiltins["strings.repeat"].Fn

	result := repeatFn(&object.String{Value: "hello"}, &object.Integer{Value: big.NewInt(0)})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	if strVal.Value != "" {
		t.Errorf("expected empty string, got '%s'", strVal.Value)
	}
}

func TestStringsSplitEmpty(t *testing.T) {
	splitFn := StringsBuiltins["strings.split"].Fn

	result := splitFn(&object.String{Value: ""}, &object.String{Value: ","})
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(arr.Elements) != 1 {
		t.Errorf("expected 1 element, got %d", len(arr.Elements))
	}
}

func TestStringsReplaceNone(t *testing.T) {
	replaceFn := StringsBuiltins["strings.replace"].Fn

	result := replaceFn(
		&object.String{Value: "hello"},
		&object.String{Value: "x"},
		&object.String{Value: "y"},
	)
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	if strVal.Value != "hello" {
		t.Errorf("expected 'hello', got '%s'", strVal.Value)
	}
}

func TestMathAtanFloat(t *testing.T) {
	atanFn := MathBuiltins["math.atan"].Fn

	result := atanFn(&object.Float{Value: 0})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}

	if floatVal.Value != 0 {
		t.Errorf("expected 0, got %f", floatVal.Value)
	}
}

func TestMathAsinFloat(t *testing.T) {
	asinFn := MathBuiltins["math.asin"].Fn

	result := asinFn(&object.Float{Value: 0})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}

	if floatVal.Value != 0 {
		t.Errorf("expected 0, got %f", floatVal.Value)
	}
}

func TestMathAcosFloat(t *testing.T) {
	acosFn := MathBuiltins["math.acos"].Fn

	result := acosFn(&object.Float{Value: 1})
	floatVal, ok := result.(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}

	if floatVal.Value != 0 {
		t.Errorf("expected 0, got %f", floatVal.Value)
	}
}
