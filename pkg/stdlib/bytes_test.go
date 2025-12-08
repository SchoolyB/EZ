package stdlib

import (
	"math/big"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// Helper to create a byte array from Go []byte
func makeByteArray(data []byte) *object.Array {
	elements := make([]object.Object, len(data))
	for i, b := range data {
		elements[i] = &object.Byte{Value: b}
	}
	return &object.Array{Elements: elements, ElementType: "byte"}
}

// Helper to create an integer array
func makeIntArray(data []int64) *object.Array {
	elements := make([]object.Object, len(data))
	for i, v := range data {
		elements[i] = &object.Integer{Value: big.NewInt(v)}
	}
	return &object.Array{Elements: elements}
}

// Helper to extract byte slice from result
func getByteSlice(obj object.Object) []byte {
	arr, ok := obj.(*object.Array)
	if !ok {
		return nil
	}
	result := make([]byte, len(arr.Elements))
	for i, elem := range arr.Elements {
		switch e := elem.(type) {
		case *object.Byte:
			result[i] = e.Value
		case *object.Integer:
			result[i] = byte(e.Value.Int64())
		}
	}
	return result
}

// ============================================================================
// Creation Functions Tests
// ============================================================================

func TestBytesFromArray(t *testing.T) {
	fn := BytesBuiltins["bytes.from_array"].Fn

	tests := []struct {
		name     string
		input    []int64
		expected []byte
		wantErr  bool
	}{
		{"empty array", []int64{}, []byte{}, false},
		{"simple bytes", []int64{72, 101, 108, 108, 111}, []byte("Hello"), false},
		{"boundary values", []int64{0, 127, 255}, []byte{0, 127, 255}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(makeIntArray(tt.input))
			if tt.wantErr {
				if _, ok := result.(*object.Error); !ok {
					t.Errorf("expected error, got %T", result)
				}
				return
			}
			got := getByteSlice(result)
			if string(got) != string(tt.expected) {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBytesFromArrayErrors(t *testing.T) {
	fn := BytesBuiltins["bytes.from_array"].Fn

	// Value out of range
	result := fn(makeIntArray([]int64{256}))
	if _, ok := result.(*object.Error); !ok {
		t.Errorf("expected error for value > 255, got %T", result)
	}

	// Negative value
	result = fn(makeIntArray([]int64{-1}))
	if _, ok := result.(*object.Error); !ok {
		t.Errorf("expected error for negative value, got %T", result)
	}

	// Wrong argument type
	result = fn(&object.String{Value: "not an array"})
	if _, ok := result.(*object.Error); !ok {
		t.Errorf("expected error for wrong type, got %T", result)
	}

	// Wrong argument count
	result = fn()
	if _, ok := result.(*object.Error); !ok {
		t.Errorf("expected error for no args, got %T", result)
	}
}

func TestBytesFromString(t *testing.T) {
	fn := BytesBuiltins["bytes.from_string"].Fn

	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{"empty string", "", []byte{}},
		{"hello", "Hello", []byte("Hello")},
		{"unicode", "こんにちは", []byte("こんにちは")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(&object.String{Value: tt.input})
			got := getByteSlice(result)
			if string(got) != string(tt.expected) {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBytesFromHex(t *testing.T) {
	fn := BytesBuiltins["bytes.from_hex"].Fn

	// Valid hex
	result := fn(&object.String{Value: "48656c6c6f"})
	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}
	got := getByteSlice(rv.Values[0])
	if string(got) != "Hello" {
		t.Errorf("got %s, want Hello", string(got))
	}
	if rv.Values[1] != object.NIL {
		t.Errorf("expected nil error, got %v", rv.Values[1])
	}

	// Invalid hex
	result = fn(&object.String{Value: "GGGG"})
	rv, ok = result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}
	if rv.Values[0] != object.NIL {
		t.Errorf("expected nil value for invalid hex")
	}
	if _, ok := rv.Values[1].(*object.Struct); !ok {
		t.Errorf("expected error struct, got %T", rv.Values[1])
	}
}

func TestBytesFromBase64(t *testing.T) {
	fn := BytesBuiltins["bytes.from_base64"].Fn

	// Valid base64
	result := fn(&object.String{Value: "SGVsbG8="})
	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}
	got := getByteSlice(rv.Values[0])
	if string(got) != "Hello" {
		t.Errorf("got %s, want Hello", string(got))
	}

	// Invalid base64
	result = fn(&object.String{Value: "!!!!"})
	rv, ok = result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}
	if rv.Values[0] != object.NIL {
		t.Errorf("expected nil value for invalid base64")
	}
}

// ============================================================================
// Conversion Functions Tests
// ============================================================================

func TestBytesToString(t *testing.T) {
	fn := BytesBuiltins["bytes.to_string"].Fn

	result := fn(makeByteArray([]byte("Hello")))
	str, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if str.Value != "Hello" {
		t.Errorf("got %s, want Hello", str.Value)
	}
}

func TestBytesToArray(t *testing.T) {
	fn := BytesBuiltins["bytes.to_array"].Fn

	result := fn(makeByteArray([]byte{72, 101}))
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}
	if len(arr.Elements) != 2 {
		t.Errorf("got len %d, want 2", len(arr.Elements))
	}
	if arr.Elements[0].(*object.Integer).Value.Cmp(big.NewInt(72)) != 0 {
		t.Errorf("got %s, want 72", arr.Elements[0].(*object.Integer).Value.String())
	}
}

func TestBytesToHex(t *testing.T) {
	fn := BytesBuiltins["bytes.to_hex"].Fn

	result := fn(makeByteArray([]byte("Hello")))
	str, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if str.Value != "48656c6c6f" {
		t.Errorf("got %s, want 48656c6c6f", str.Value)
	}
}

func TestBytesToHexUpper(t *testing.T) {
	fn := BytesBuiltins["bytes.to_hex_upper"].Fn

	result := fn(makeByteArray([]byte("Hello")))
	str, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if str.Value != "48656C6C6F" {
		t.Errorf("got %s, want 48656C6C6F", str.Value)
	}
}

func TestBytesToBase64(t *testing.T) {
	fn := BytesBuiltins["bytes.to_base64"].Fn

	result := fn(makeByteArray([]byte("Hello")))
	str, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if str.Value != "SGVsbG8=" {
		t.Errorf("got %s, want SGVsbG8=", str.Value)
	}
}

// ============================================================================
// Operations Tests
// ============================================================================

func TestBytesSlice(t *testing.T) {
	fn := BytesBuiltins["bytes.slice"].Fn

	input := makeByteArray([]byte("Hello"))

	// Normal slice
	result := fn(input, &object.Integer{Value: big.NewInt(1)}, &object.Integer{Value: big.NewInt(4)})
	got := getByteSlice(result)
	if string(got) != "ell" {
		t.Errorf("got %s, want ell", string(got))
	}

	// Negative indices
	result = fn(input, &object.Integer{Value: big.NewInt(-3)}, &object.Integer{Value: big.NewInt(-1)})
	got = getByteSlice(result)
	if string(got) != "ll" {
		t.Errorf("got %s, want ll", string(got))
	}
}

func TestBytesConcat(t *testing.T) {
	fn := BytesBuiltins["bytes.concat"].Fn

	a := makeByteArray([]byte("Hello"))
	b := makeByteArray([]byte(" World"))

	result := fn(a, b)
	got := getByteSlice(result)
	if string(got) != "Hello World" {
		t.Errorf("got %s, want Hello World", string(got))
	}
}

func TestBytesContains(t *testing.T) {
	fn := BytesBuiltins["bytes.contains"].Fn

	haystack := makeByteArray([]byte("Hello World"))
	needle := makeByteArray([]byte("World"))
	missing := makeByteArray([]byte("xyz"))

	result := fn(haystack, needle)
	if result != object.TRUE {
		t.Errorf("expected true for contains")
	}

	result = fn(haystack, missing)
	if result != object.FALSE {
		t.Errorf("expected false for not contains")
	}
}

func TestBytesIndex(t *testing.T) {
	fn := BytesBuiltins["bytes.index"].Fn

	haystack := makeByteArray([]byte("Hello World"))
	needle := makeByteArray([]byte("World"))
	missing := makeByteArray([]byte("xyz"))

	result := fn(haystack, needle)
	idx := result.(*object.Integer).Value
	if idx.Cmp(big.NewInt(6)) != 0 {
		t.Errorf("got %s, want 6", idx.String())
	}

	result = fn(haystack, missing)
	idx = result.(*object.Integer).Value
	if idx.Cmp(big.NewInt(-1)) != 0 {
		t.Errorf("got %s, want -1", idx.String())
	}
}

func TestBytesLastIndex(t *testing.T) {
	fn := BytesBuiltins["bytes.last_index"].Fn

	haystack := makeByteArray([]byte("abcabc"))
	needle := makeByteArray([]byte("bc"))

	result := fn(haystack, needle)
	idx := result.(*object.Integer).Value
	if idx.Cmp(big.NewInt(4)) != 0 {
		t.Errorf("got %s, want 4", idx.String())
	}
}

func TestBytesCount(t *testing.T) {
	fn := BytesBuiltins["bytes.count"].Fn

	haystack := makeByteArray([]byte("abababab"))
	needle := makeByteArray([]byte("ab"))

	result := fn(haystack, needle)
	count := result.(*object.Integer).Value
	if count.Cmp(big.NewInt(4)) != 0 {
		t.Errorf("got %s, want 4", count.String())
	}
}

func TestBytesCompare(t *testing.T) {
	fn := BytesBuiltins["bytes.compare"].Fn

	a := makeByteArray([]byte("aaa"))
	b := makeByteArray([]byte("bbb"))

	result := fn(a, b)
	if result.(*object.Integer).Value.Cmp(big.NewInt(-1)) != 0 {
		t.Errorf("expected -1 for a < b")
	}

	result = fn(b, a)
	if result.(*object.Integer).Value.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("expected 1 for b > a")
	}

	result = fn(a, a)
	if result.(*object.Integer).Value.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("expected 0 for a == a")
	}
}

func TestBytesEquals(t *testing.T) {
	fn := BytesBuiltins["bytes.equals"].Fn

	a := makeByteArray([]byte("Hello"))
	b := makeByteArray([]byte("Hello"))
	c := makeByteArray([]byte("World"))

	if fn(a, b) != object.TRUE {
		t.Errorf("expected true for equal bytes")
	}

	if fn(a, c) != object.FALSE {
		t.Errorf("expected false for different bytes")
	}
}

// ============================================================================
// Inspection Tests
// ============================================================================

func TestBytesIsEmpty(t *testing.T) {
	fn := BytesBuiltins["bytes.is_empty"].Fn

	empty := makeByteArray([]byte{})
	notEmpty := makeByteArray([]byte("Hi"))

	if fn(empty) != object.TRUE {
		t.Errorf("expected true for empty")
	}

	if fn(notEmpty) != object.FALSE {
		t.Errorf("expected false for not empty")
	}
}

func TestBytesStartsWith(t *testing.T) {
	fn := BytesBuiltins["bytes.starts_with"].Fn

	data := makeByteArray([]byte("Hello World"))
	prefix := makeByteArray([]byte("Hello"))
	notPrefix := makeByteArray([]byte("World"))

	if fn(data, prefix) != object.TRUE {
		t.Errorf("expected true for starts_with")
	}

	if fn(data, notPrefix) != object.FALSE {
		t.Errorf("expected false for not starts_with")
	}
}

func TestBytesEndsWith(t *testing.T) {
	fn := BytesBuiltins["bytes.ends_with"].Fn

	data := makeByteArray([]byte("Hello World"))
	suffix := makeByteArray([]byte("World"))
	notSuffix := makeByteArray([]byte("Hello"))

	if fn(data, suffix) != object.TRUE {
		t.Errorf("expected true for ends_with")
	}

	if fn(data, notSuffix) != object.FALSE {
		t.Errorf("expected false for not ends_with")
	}
}

// ============================================================================
// Manipulation Tests
// ============================================================================

func TestBytesReverse(t *testing.T) {
	fn := BytesBuiltins["bytes.reverse"].Fn

	input := makeByteArray([]byte("Hello"))
	result := fn(input)
	got := getByteSlice(result)
	if string(got) != "olleH" {
		t.Errorf("got %s, want olleH", string(got))
	}
}

func TestBytesRepeat(t *testing.T) {
	fn := BytesBuiltins["bytes.repeat"].Fn

	input := makeByteArray([]byte("ab"))

	result := fn(input, &object.Integer{Value: big.NewInt(3)})
	got := getByteSlice(result)
	if string(got) != "ababab" {
		t.Errorf("got %s, want ababab", string(got))
	}

	result = fn(input, &object.Integer{Value: big.NewInt(0)})
	got = getByteSlice(result)
	if len(got) != 0 {
		t.Errorf("got len %d, want 0", len(got))
	}
}

func TestBytesReplace(t *testing.T) {
	fn := BytesBuiltins["bytes.replace"].Fn

	input := makeByteArray([]byte("hello hello"))
	old := makeByteArray([]byte("hello"))
	new := makeByteArray([]byte("hi"))

	result := fn(input, old, new)
	got := getByteSlice(result)
	if string(got) != "hi hi" {
		t.Errorf("got %s, want hi hi", string(got))
	}
}

func TestBytesReplaceN(t *testing.T) {
	fn := BytesBuiltins["bytes.replace_n"].Fn

	input := makeByteArray([]byte("hello hello hello"))
	old := makeByteArray([]byte("hello"))
	new := makeByteArray([]byte("hi"))

	result := fn(input, old, new, &object.Integer{Value: big.NewInt(2)})
	got := getByteSlice(result)
	if string(got) != "hi hi hello" {
		t.Errorf("got %s, want hi hi hello", string(got))
	}
}

func TestBytesTrim(t *testing.T) {
	fn := BytesBuiltins["bytes.trim"].Fn

	input := makeByteArray([]byte{0, 0, 65, 66, 0, 0})
	cutset := makeByteArray([]byte{0})

	result := fn(input, cutset)
	got := getByteSlice(result)
	if string(got) != "AB" {
		t.Errorf("got %v, want AB", got)
	}
}

func TestBytesPadLeft(t *testing.T) {
	fn := BytesBuiltins["bytes.pad_left"].Fn

	input := makeByteArray([]byte("Hi"))

	result := fn(input, &object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(0)})
	got := getByteSlice(result)
	expected := []byte{0, 0, 0, 'H', 'i'}
	if string(got) != string(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}

	// No padding needed
	result = fn(input, &object.Integer{Value: big.NewInt(2)}, &object.Integer{Value: big.NewInt(0)})
	got = getByteSlice(result)
	if string(got) != "Hi" {
		t.Errorf("got %s, want Hi", string(got))
	}
}

func TestBytesPadRight(t *testing.T) {
	fn := BytesBuiltins["bytes.pad_right"].Fn

	input := makeByteArray([]byte("Hi"))

	result := fn(input, &object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(0)})
	got := getByteSlice(result)
	expected := []byte{'H', 'i', 0, 0, 0}
	if string(got) != string(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

// ============================================================================
// Bitwise Operations Tests
// ============================================================================

func TestBytesAnd(t *testing.T) {
	fn := BytesBuiltins["bytes.and"].Fn

	a := makeByteArray([]byte{0xFF, 0x0F})
	b := makeByteArray([]byte{0xF0, 0xFF})

	result := fn(a, b)
	rv := result.(*object.ReturnValue)
	got := getByteSlice(rv.Values[0])
	expected := []byte{0xF0, 0x0F}
	if got[0] != expected[0] || got[1] != expected[1] {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestBytesAndLengthMismatch(t *testing.T) {
	fn := BytesBuiltins["bytes.and"].Fn

	a := makeByteArray([]byte{0xFF})
	b := makeByteArray([]byte{0xF0, 0xFF})

	result := fn(a, b)
	rv := result.(*object.ReturnValue)
	if rv.Values[0] != object.NIL {
		t.Errorf("expected nil for length mismatch")
	}
	if _, ok := rv.Values[1].(*object.Struct); !ok {
		t.Errorf("expected error struct, got %T", rv.Values[1])
	}
}

func TestBytesOr(t *testing.T) {
	fn := BytesBuiltins["bytes.or"].Fn

	a := makeByteArray([]byte{0xF0, 0x00})
	b := makeByteArray([]byte{0x0F, 0xFF})

	result := fn(a, b)
	rv := result.(*object.ReturnValue)
	got := getByteSlice(rv.Values[0])
	expected := []byte{0xFF, 0xFF}
	if got[0] != expected[0] || got[1] != expected[1] {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestBytesXor(t *testing.T) {
	fn := BytesBuiltins["bytes.xor"].Fn

	a := makeByteArray([]byte{0xFF, 0x00})
	b := makeByteArray([]byte{0xF0, 0x0F})

	result := fn(a, b)
	rv := result.(*object.ReturnValue)
	got := getByteSlice(rv.Values[0])
	expected := []byte{0x0F, 0x0F}
	if got[0] != expected[0] || got[1] != expected[1] {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestBytesNot(t *testing.T) {
	fn := BytesBuiltins["bytes.not"].Fn

	a := makeByteArray([]byte{0xFF, 0x00})

	result := fn(a)
	got := getByteSlice(result)
	expected := []byte{0x00, 0xFF}
	if got[0] != expected[0] || got[1] != expected[1] {
		t.Errorf("got %v, want %v", got, expected)
	}
}

// ============================================================================
// Utility Tests
// ============================================================================

func TestBytesFill(t *testing.T) {
	fn := BytesBuiltins["bytes.fill"].Fn

	input := makeByteArray([]byte{1, 2, 3, 4, 5})

	result := fn(input, &object.Integer{Value: big.NewInt(0xFF)})
	got := getByteSlice(result)
	for i, b := range got {
		if b != 0xFF {
			t.Errorf("got[%d] = %d, want 255", i, b)
		}
	}
}

func TestBytesCopy(t *testing.T) {
	fn := BytesBuiltins["bytes.copy"].Fn

	input := makeByteArray([]byte("Hello"))

	result := fn(input)
	arr := result.(*object.Array)

	// Verify it's a copy (different slice)
	if &arr.Elements == &input.Elements {
		t.Errorf("expected different slice, got same reference")
	}

	got := getByteSlice(result)
	if string(got) != "Hello" {
		t.Errorf("got %s, want Hello", string(got))
	}
}

func TestBytesZero(t *testing.T) {
	fn := BytesBuiltins["bytes.zero"].Fn

	input := makeByteArray([]byte("secret"))

	result := fn(input)
	got := getByteSlice(result)
	for i, b := range got {
		if b != 0 {
			t.Errorf("got[%d] = %d, want 0", i, b)
		}
	}
}

// ============================================================================
// Split and Join Tests
// ============================================================================

func TestBytesSplit(t *testing.T) {
	fn := BytesBuiltins["bytes.split"].Fn

	input := makeByteArray([]byte("a,b,c"))
	sep := makeByteArray([]byte(","))

	result := fn(input, sep)
	arr := result.(*object.Array)
	if len(arr.Elements) != 3 {
		t.Errorf("got %d parts, want 3", len(arr.Elements))
	}
}

func TestBytesJoin(t *testing.T) {
	fn := BytesBuiltins["bytes.join"].Fn

	parts := &object.Array{
		Elements: []object.Object{
			makeByteArray([]byte("a")),
			makeByteArray([]byte("b")),
			makeByteArray([]byte("c")),
		},
	}
	sep := makeByteArray([]byte("-"))

	result := fn(parts, sep)
	got := getByteSlice(result)
	if string(got) != "a-b-c" {
		t.Errorf("got %s, want a-b-c", string(got))
	}
}
