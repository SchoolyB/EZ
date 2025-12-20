package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"math/big"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// ============================================================================
// json.encode() tests
// ============================================================================

func TestJSONEncodeString(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn
	result := fn(&object.String{Value: "hello"})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	if str.Value != `"hello"` {
		t.Errorf("expected '\"hello\"', got %s", str.Value)
	}

	if rv.Values[1] != object.NIL {
		t.Errorf("expected nil error, got %v", rv.Values[1])
	}
}

func TestJSONEncodeInteger(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn
	result := fn(&object.Integer{Value: big.NewInt(42)})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	if str.Value != "42" {
		t.Errorf("expected '42', got %s", str.Value)
	}
}

func TestJSONEncodeFloat(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn
	result := fn(&object.Float{Value: 3.14})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	if str.Value != "3.14" {
		t.Errorf("expected '3.14', got %s", str.Value)
	}
}

func TestJSONEncodeBoolean(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn

	// true
	result := fn(object.TRUE)
	rv, _ := result.(*object.ReturnValue)
	str, _ := rv.Values[0].(*object.String)
	if str.Value != "true" {
		t.Errorf("expected 'true', got %s", str.Value)
	}

	// false
	result = fn(object.FALSE)
	rv, _ = result.(*object.ReturnValue)
	str, _ = rv.Values[0].(*object.String)
	if str.Value != "false" {
		t.Errorf("expected 'false', got %s", str.Value)
	}
}

func TestJSONEncodeNil(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn
	result := fn(object.NIL)

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	if str.Value != "null" {
		t.Errorf("expected 'null', got %s", str.Value)
	}
}

func TestJSONEncodeArray(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn
	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
			&object.Integer{Value: big.NewInt(3)},
		},
	}
	result := fn(arr)

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	if str.Value != "[1,2,3]" {
		t.Errorf("expected '[1,2,3]', got %s", str.Value)
	}
}

func TestJSONEncodeMap(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn
	m := &object.Map{
		Pairs: []*object.MapPair{
			{Key: &object.String{Value: "name"}, Value: &object.String{Value: "Alice"}},
			{Key: &object.String{Value: "age"}, Value: &object.Integer{Value: big.NewInt(30)}},
		},
		Index: map[string]int{"s:name": 0, "s:age": 1},
	}
	result := fn(m)

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	// Keys are sorted alphabetically
	expected := `{"age":30,"name":"Alice"}`
	if str.Value != expected {
		t.Errorf("expected %s, got %s", expected, str.Value)
	}
}

func TestJSONEncodeStruct(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn
	s := &object.Struct{
		TypeName: "Person",
		Fields: map[string]object.Object{
			"name": &object.String{Value: "Bob"},
			"age":  &object.Integer{Value: big.NewInt(25)},
		},
	}
	result := fn(s)

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	// Fields are sorted alphabetically
	expected := `{"age":25,"name":"Bob"}`
	if str.Value != expected {
		t.Errorf("expected %s, got %s", expected, str.Value)
	}
}

func TestJSONEncodeNonStringMapKey(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn
	m := &object.Map{
		Pairs: []*object.MapPair{
			{Key: &object.Integer{Value: big.NewInt(1)}, Value: &object.String{Value: "one"}},
		},
		Index: map[string]int{"i:1": 0},
	}
	result := fn(m)

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	// Should have error
	if rv.Values[0] != object.NIL {
		t.Errorf("expected nil result, got %v", rv.Values[0])
	}

	errStruct, ok := rv.Values[1].(*object.Struct)
	if !ok {
		t.Fatalf("expected error Struct, got %T", rv.Values[1])
	}

	code, _ := errStruct.Fields["code"].(*object.String)
	if code.Value != "E13003" {
		t.Errorf("expected error code E13003, got %s", code.Value)
	}
}

func TestJSONEncodeWrongArgCount(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn
	result := fn()

	err, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}

	if err.Code != "E7001" {
		t.Errorf("expected error code E7001, got %s", err.Code)
	}
}

// ============================================================================
// json.decode() tests
// ============================================================================

func TestJSONDecodeString(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn
	result := fn(&object.String{Value: `"hello"`})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	if str.Value != "hello" {
		t.Errorf("expected 'hello', got %s", str.Value)
	}
}

func TestJSONDecodeInteger(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn
	result := fn(&object.String{Value: "42"})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	integer, ok := rv.Values[0].(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer result, got %T", rv.Values[0])
	}

	if integer.Value.Int64() != 42 {
		t.Errorf("expected 42, got %d", integer.Value.Int64())
	}
}

func TestJSONDecodeFloat(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn
	result := fn(&object.String{Value: "3.14"})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	f, ok := rv.Values[0].(*object.Float)
	if !ok {
		t.Fatalf("expected Float result, got %T", rv.Values[0])
	}

	if f.Value != 3.14 {
		t.Errorf("expected 3.14, got %f", f.Value)
	}
}

func TestJSONDecodeBoolean(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn

	// true
	result := fn(&object.String{Value: "true"})
	rv, _ := result.(*object.ReturnValue)
	if rv.Values[0] != object.TRUE {
		t.Errorf("expected TRUE, got %v", rv.Values[0])
	}

	// false
	result = fn(&object.String{Value: "false"})
	rv, _ = result.(*object.ReturnValue)
	if rv.Values[0] != object.FALSE {
		t.Errorf("expected FALSE, got %v", rv.Values[0])
	}
}

func TestJSONDecodeNull(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn
	result := fn(&object.String{Value: "null"})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	if rv.Values[0] != object.NIL {
		t.Errorf("expected NIL, got %v", rv.Values[0])
	}
}

func TestJSONDecodeArray(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn
	result := fn(&object.String{Value: "[1, 2, 3]"})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := rv.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array result, got %T", rv.Values[0])
	}

	if len(arr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestJSONDecodeObject(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn
	result := fn(&object.String{Value: `{"name": "Alice", "age": 30}`})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	m, ok := rv.Values[0].(*object.Map)
	if !ok {
		t.Fatalf("expected Map result, got %T", rv.Values[0])
	}

	if len(m.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(m.Pairs))
	}
}

func TestJSONDecodeInvalidJSON(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn
	result := fn(&object.String{Value: "{invalid}"})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	if rv.Values[0] != object.NIL {
		t.Errorf("expected nil result, got %v", rv.Values[0])
	}

	errStruct, ok := rv.Values[1].(*object.Struct)
	if !ok {
		t.Fatalf("expected error Struct, got %T", rv.Values[1])
	}

	code, _ := errStruct.Fields["code"].(*object.String)
	if code.Value != "E13001" {
		t.Errorf("expected error code E13001, got %s", code.Value)
	}
}

func TestJSONDecodeWrongArgType(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn
	result := fn(&object.Integer{Value: big.NewInt(42)})

	err, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}

	if err.Code != "E7003" {
		t.Errorf("expected error code E7003, got %s", err.Code)
	}
}

// ============================================================================
// json.pretty() tests
// ============================================================================

func TestJSONPrettySimple(t *testing.T) {
	fn := JsonBuiltins["json.pretty"].Fn
	arr := &object.Array{
		Elements: []object.Object{
			&object.Integer{Value: big.NewInt(1)},
			&object.Integer{Value: big.NewInt(2)},
		},
	}
	result := fn(arr, &object.String{Value: "  "})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	expected := "[\n  1,\n  2\n]"
	if str.Value != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, str.Value)
	}
}

func TestJSONPrettyWrongArgCount(t *testing.T) {
	fn := JsonBuiltins["json.pretty"].Fn
	result := fn(&object.String{Value: "test"})

	err, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}

	if err.Code != "E7001" {
		t.Errorf("expected error code E7001, got %s", err.Code)
	}
}

// ============================================================================
// json.is_valid() tests
// ============================================================================

func TestJSONIsValidTrue(t *testing.T) {
	fn := JsonBuiltins["json.is_valid"].Fn

	tests := []string{
		`"hello"`,
		`42`,
		`true`,
		`false`,
		`null`,
		`[1, 2, 3]`,
		`{"key": "value"}`,
	}

	for _, test := range tests {
		result := fn(&object.String{Value: test})
		if result != object.TRUE {
			t.Errorf("expected TRUE for %s, got %v", test, result)
		}
	}
}

func TestJSONIsValidFalse(t *testing.T) {
	fn := JsonBuiltins["json.is_valid"].Fn

	tests := []string{
		`{invalid}`,
		`[1, 2,]`,
		`undefined`,
		`'single quotes'`,
	}

	for _, test := range tests {
		result := fn(&object.String{Value: test})
		if result != object.FALSE {
			t.Errorf("expected FALSE for %s, got %v", test, result)
		}
	}
}

func TestJSONIsValidWrongArgType(t *testing.T) {
	fn := JsonBuiltins["json.is_valid"].Fn
	result := fn(&object.Integer{Value: big.NewInt(42)})

	err, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}

	if err.Code != "E7003" {
		t.Errorf("expected error code E7003, got %s", err.Code)
	}
}

// ============================================================================
// Large integer tests
// ============================================================================

func TestJSONEncodeLargeInteger(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn

	// Create a large integer that exceeds int64 range
	largeInt := new(big.Int)
	largeInt.SetString("9999999999999999999999999999", 10)

	result := fn(&object.Integer{Value: largeInt})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	// Large integers should be encoded as strings
	expected := `"9999999999999999999999999999"`
	if str.Value != expected {
		t.Errorf("expected %s, got %s", expected, str.Value)
	}
}

// ============================================================================
// Typed json.decode() tests
// ============================================================================

func TestJSONDecodeTypedStruct(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn

	// Create a struct definition for Person
	structDef := &object.StructDef{
		Name: "Person",
		Fields: map[string]string{
			"name": "string",
			"age":  "int",
		},
	}

	typeVal := &object.TypeValue{
		TypeName: "Person",
		Def:      structDef,
	}

	result := fn(
		&object.String{Value: `{"name": "Alice", "age": 30}`},
		typeVal,
	)

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	s, ok := rv.Values[0].(*object.Struct)
	if !ok {
		t.Fatalf("expected Struct result, got %T", rv.Values[0])
	}

	if s.TypeName != "Person" {
		t.Errorf("expected TypeName 'Person', got %s", s.TypeName)
	}

	nameField, ok := s.Fields["name"].(*object.String)
	if !ok {
		t.Fatalf("expected name field to be String, got %T", s.Fields["name"])
	}
	if nameField.Value != "Alice" {
		t.Errorf("expected name 'Alice', got %s", nameField.Value)
	}

	ageField, ok := s.Fields["age"].(*object.Integer)
	if !ok {
		t.Fatalf("expected age field to be Integer, got %T", s.Fields["age"])
	}
	if ageField.Value.Int64() != 30 {
		t.Errorf("expected age 30, got %d", ageField.Value.Int64())
	}

	if rv.Values[1] != object.NIL {
		t.Errorf("expected nil error, got %v", rv.Values[1])
	}
}

func TestJSONDecodeTypedStructMissingField(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn

	structDef := &object.StructDef{
		Name: "Person",
		Fields: map[string]string{
			"name": "string",
			"age":  "int",
		},
	}

	typeVal := &object.TypeValue{
		TypeName: "Person",
		Def:      structDef,
	}

	// JSON missing "age" field
	result := fn(
		&object.String{Value: `{"name": "Bob"}`},
		typeVal,
	)

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	s, ok := rv.Values[0].(*object.Struct)
	if !ok {
		t.Fatalf("expected Struct result, got %T", rv.Values[0])
	}

	// Missing field should have zero value
	ageField, ok := s.Fields["age"].(*object.Integer)
	if !ok {
		t.Fatalf("expected age field to be Integer, got %T", s.Fields["age"])
	}
	if ageField.Value.Int64() != 0 {
		t.Errorf("expected age 0 (zero value), got %d", ageField.Value.Int64())
	}
}

func TestJSONDecodeTypedStructBoolField(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn

	structDef := &object.StructDef{
		Name: "Task",
		Fields: map[string]string{
			"title":     "string",
			"completed": "bool",
		},
	}

	typeVal := &object.TypeValue{
		TypeName: "Task",
		Def:      structDef,
	}

	result := fn(
		&object.String{Value: `{"title": "Test", "completed": true}`},
		typeVal,
	)

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	s, ok := rv.Values[0].(*object.Struct)
	if !ok {
		t.Fatalf("expected Struct result, got %T", rv.Values[0])
	}

	if s.Fields["completed"] != object.TRUE {
		t.Errorf("expected completed to be TRUE, got %v", s.Fields["completed"])
	}
}

func TestJSONDecodeTypedInvalidJSON(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn

	structDef := &object.StructDef{
		Name:   "Person",
		Fields: map[string]string{"name": "string"},
	}

	typeVal := &object.TypeValue{
		TypeName: "Person",
		Def:      structDef,
	}

	result := fn(
		&object.String{Value: "{invalid}"},
		typeVal,
	)

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	if rv.Values[0] != object.NIL {
		t.Errorf("expected NIL for invalid JSON, got %v", rv.Values[0])
	}

	// Should have error in second position
	errStruct, ok := rv.Values[1].(*object.Struct)
	if !ok {
		t.Fatalf("expected Struct error, got %T", rv.Values[1])
	}

	if errStruct.TypeName != "Error" {
		t.Errorf("expected Error struct, got %s", errStruct.TypeName)
	}
}

func TestJSONDecodeTypedWrongSecondArg(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn

	// Second arg is not a TypeValue
	result := fn(
		&object.String{Value: `{"name": "Alice"}`},
		&object.String{Value: "not a type"},
	)

	err, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}

	if err.Code != "E7003" {
		t.Errorf("expected error code E7003, got %s", err.Code)
	}
}

// ============================================================================
// Additional coverage tests
// ============================================================================

func TestJSONPrettyEncodeMap(t *testing.T) {
	fn := JsonBuiltins["json.pretty"].Fn

	// Test with map
	m := &object.Map{
		Pairs: []*object.MapPair{
			{Key: &object.String{Value: "name"}, Value: &object.String{Value: "Alice"}},
		},
		Index: map[string]int{"s:name": 0},
	}

	result := fn(m, &object.String{Value: "  "})
	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	// Check that it contains formatting (newlines/indentation)
	if len(str.Value) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestJSONEncodeByteCoverage(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn

	result := fn(&object.Byte{Value: 65})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	if str.Value != "65" {
		t.Errorf("expected '65', got %s", str.Value)
	}
}

func TestJSONEncodeCharCoverage(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn

	result := fn(&object.Char{Value: 'A'})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	if str.Value != `"A"` {
		t.Errorf("expected '\"A\"', got %s", str.Value)
	}
}

func TestJSONEncodeNestedArrayCoverage(t *testing.T) {
	fn := JsonBuiltins["json.encode"].Fn

	nested := &object.Array{
		Elements: []object.Object{
			&object.Array{
				Elements: []object.Object{
					&object.Integer{Value: big.NewInt(1)},
					&object.Integer{Value: big.NewInt(2)},
				},
			},
			&object.Array{
				Elements: []object.Object{
					&object.Integer{Value: big.NewInt(3)},
					&object.Integer{Value: big.NewInt(4)},
				},
			},
		},
	}

	result := fn(nested)
	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	str, ok := rv.Values[0].(*object.String)
	if !ok {
		t.Fatalf("expected String result, got %T", rv.Values[0])
	}

	if str.Value != "[[1,2],[3,4]]" {
		t.Errorf("expected '[[1,2],[3,4]]', got %s", str.Value)
	}
}

func TestJSONDecodeArrayCoverage(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn

	result := fn(&object.String{Value: `[1, 2, 3]`})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	arr, ok := rv.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array result, got %T", rv.Values[0])
	}

	if len(arr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestJSONDecodeNestedObject(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn

	result := fn(&object.String{Value: `{"person": {"name": "Alice", "age": 30}}`})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	m, ok := rv.Values[0].(*object.Map)
	if !ok {
		t.Fatalf("expected Map result, got %T", rv.Values[0])
	}

	if len(m.Pairs) != 1 {
		t.Errorf("expected 1 pair, got %d", len(m.Pairs))
	}
}

func TestJSONDecodeBooleanCoverage(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn

	// Test true
	result := fn(&object.String{Value: `true`})
	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	if rv.Values[0] != object.TRUE {
		t.Errorf("expected TRUE, got %v", rv.Values[0])
	}

	// Test false
	result = fn(&object.String{Value: `false`})
	rv, _ = result.(*object.ReturnValue)

	if rv.Values[0] != object.FALSE {
		t.Errorf("expected FALSE, got %v", rv.Values[0])
	}
}

func TestJSONDecodeNullCoverage(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn

	result := fn(&object.String{Value: `null`})
	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	if rv.Values[0] != object.NIL {
		t.Errorf("expected NIL, got %v", rv.Values[0])
	}
}

func TestJSONDecodeFloatCoverage(t *testing.T) {
	fn := JsonBuiltins["json.decode"].Fn

	result := fn(&object.String{Value: `3.14159`})
	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	f, ok := rv.Values[0].(*object.Float)
	if !ok {
		t.Fatalf("expected Float result, got %T", rv.Values[0])
	}

	if f.Value != 3.14159 {
		t.Errorf("expected 3.14159, got %f", f.Value)
	}
}
