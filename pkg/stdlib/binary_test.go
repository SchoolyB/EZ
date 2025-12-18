package stdlib

import (
	"math"
	"math/big"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// Helper to create byte array from slice
func makeBinaryByteArray(bytes []byte) *object.Array {
	elements := make([]object.Object, len(bytes))
	for i, b := range bytes {
		elements[i] = &object.Byte{Value: b}
	}
	return &object.Array{Elements: elements, ElementType: "byte"}
}

// Helper to extract bytes from return value
func extractBytesFromReturn(t *testing.T, result object.Object) []byte {
	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}
	if len(rv.Values) != 2 {
		t.Fatalf("expected 2 values in tuple, got %d", len(rv.Values))
	}
	if rv.Values[1] != object.NIL {
		t.Fatalf("expected nil error, got %v", rv.Values[1])
	}
	arr, ok := rv.Values[0].(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", rv.Values[0])
	}
	bytes := make([]byte, len(arr.Elements))
	for i, elem := range arr.Elements {
		b, ok := elem.(*object.Byte)
		if !ok {
			t.Fatalf("expected Byte at index %d, got %T", i, elem)
		}
		bytes[i] = b.Value
	}
	return bytes
}

// Helper to extract integer from return value
func extractIntFromReturn(t *testing.T, result object.Object) *big.Int {
	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}
	if len(rv.Values) != 2 {
		t.Fatalf("expected 2 values in tuple, got %d", len(rv.Values))
	}
	if rv.Values[1] != object.NIL {
		t.Fatalf("expected nil error, got %v", rv.Values[1])
	}
	intVal, ok := rv.Values[0].(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", rv.Values[0])
	}
	return intVal.Value
}

// Helper to extract float from return value
func extractFloatFromReturn(t *testing.T, result object.Object) float64 {
	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}
	if len(rv.Values) != 2 {
		t.Fatalf("expected 2 values in tuple, got %d", len(rv.Values))
	}
	if rv.Values[1] != object.NIL {
		t.Fatalf("expected nil error, got %v", rv.Values[1])
	}
	floatVal, ok := rv.Values[0].(*object.Float)
	if !ok {
		t.Fatalf("expected Float, got %T", rv.Values[0])
	}
	return floatVal.Value
}

func TestBinaryEncodeDecodeI8(t *testing.T) {
	tests := []struct {
		input    int64
		expected byte
	}{
		{0, 0x00},
		{127, 0x7F},
		{-128, 0x80},
		{-1, 0xFF},
	}

	for _, tt := range tests {
		// Encode
		encodeResult := BinaryBuiltins["binary.encode_i8"].Fn(&object.Integer{Value: big.NewInt(tt.input)})
		bytes := extractBytesFromReturn(t, encodeResult)
		if len(bytes) != 1 || bytes[0] != tt.expected {
			t.Errorf("encode_i8(%d): expected [0x%02X], got %v", tt.input, tt.expected, bytes)
		}

		// Decode
		decodeResult := BinaryBuiltins["binary.decode_i8"].Fn(makeBinaryByteArray([]byte{tt.expected}))
		val := extractIntFromReturn(t, decodeResult)
		if val.Int64() != tt.input {
			t.Errorf("decode_i8([0x%02X]): expected %d, got %d", tt.expected, tt.input, val.Int64())
		}
	}
}

func TestBinaryEncodeDecodeU8(t *testing.T) {
	tests := []struct {
		input    int64
		expected byte
	}{
		{0, 0x00},
		{127, 0x7F},
		{255, 0xFF},
	}

	for _, tt := range tests {
		encodeResult := BinaryBuiltins["binary.encode_u8"].Fn(&object.Integer{Value: big.NewInt(tt.input)})
		bytes := extractBytesFromReturn(t, encodeResult)
		if len(bytes) != 1 || bytes[0] != tt.expected {
			t.Errorf("encode_u8(%d): expected [0x%02X], got %v", tt.input, tt.expected, bytes)
		}

		decodeResult := BinaryBuiltins["binary.decode_u8"].Fn(makeBinaryByteArray([]byte{tt.expected}))
		val := extractIntFromReturn(t, decodeResult)
		if val.Int64() != tt.input {
			t.Errorf("decode_u8([0x%02X]): expected %d, got %d", tt.expected, tt.input, val.Int64())
		}
	}
}

func TestBinaryEncodeDecodeI32LE(t *testing.T) {
	tests := []struct {
		input    int64
		expected []byte
	}{
		{0, []byte{0x00, 0x00, 0x00, 0x00}},
		{1, []byte{0x01, 0x00, 0x00, 0x00}},
		{256, []byte{0x00, 0x01, 0x00, 0x00}},
		{12345, []byte{0x39, 0x30, 0x00, 0x00}},
		{-1, []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{2147483647, []byte{0xFF, 0xFF, 0xFF, 0x7F}},
		{-2147483648, []byte{0x00, 0x00, 0x00, 0x80}},
	}

	for _, tt := range tests {
		encodeResult := BinaryBuiltins["binary.encode_i32_to_little_endian"].Fn(&object.Integer{Value: big.NewInt(tt.input)})
		bytes := extractBytesFromReturn(t, encodeResult)
		if len(bytes) != 4 {
			t.Errorf("encode_i32_to_little_endian(%d): expected 4 bytes, got %d", tt.input, len(bytes))
			continue
		}
		for i, b := range tt.expected {
			if bytes[i] != b {
				t.Errorf("encode_i32_to_little_endian(%d): byte %d expected 0x%02X, got 0x%02X", tt.input, i, b, bytes[i])
			}
		}

		decodeResult := BinaryBuiltins["binary.decode_i32_from_little_endian"].Fn(makeBinaryByteArray(tt.expected))
		val := extractIntFromReturn(t, decodeResult)
		if val.Int64() != tt.input {
			t.Errorf("decode_i32_from_little_endian: expected %d, got %d", tt.input, val.Int64())
		}
	}
}

func TestBinaryEncodeDecodeI32BE(t *testing.T) {
	tests := []struct {
		input    int64
		expected []byte
	}{
		{0, []byte{0x00, 0x00, 0x00, 0x00}},
		{1, []byte{0x00, 0x00, 0x00, 0x01}},
		{256, []byte{0x00, 0x00, 0x01, 0x00}},
		{12345, []byte{0x00, 0x00, 0x30, 0x39}},
		{-1, []byte{0xFF, 0xFF, 0xFF, 0xFF}},
	}

	for _, tt := range tests {
		encodeResult := BinaryBuiltins["binary.encode_i32_to_big_endian"].Fn(&object.Integer{Value: big.NewInt(tt.input)})
		bytes := extractBytesFromReturn(t, encodeResult)
		for i, b := range tt.expected {
			if bytes[i] != b {
				t.Errorf("encode_i32_to_big_endian(%d): byte %d expected 0x%02X, got 0x%02X", tt.input, i, b, bytes[i])
			}
		}

		decodeResult := BinaryBuiltins["binary.decode_i32_from_big_endian"].Fn(makeBinaryByteArray(tt.expected))
		val := extractIntFromReturn(t, decodeResult)
		if val.Int64() != tt.input {
			t.Errorf("decode_i32_from_big_endian: expected %d, got %d", tt.input, val.Int64())
		}
	}
}

func TestBinaryEncodeDecodeU64LE(t *testing.T) {
	tests := []struct {
		input    uint64
		expected []byte
	}{
		{0, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{1, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{18446744073709551615, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}},
	}

	for _, tt := range tests {
		encodeResult := BinaryBuiltins["binary.encode_u64_to_little_endian"].Fn(&object.Integer{Value: new(big.Int).SetUint64(tt.input)})
		bytes := extractBytesFromReturn(t, encodeResult)
		for i, b := range tt.expected {
			if bytes[i] != b {
				t.Errorf("encode_u64_to_little_endian(%d): byte %d expected 0x%02X, got 0x%02X", tt.input, i, b, bytes[i])
			}
		}

		decodeResult := BinaryBuiltins["binary.decode_u64_from_little_endian"].Fn(makeBinaryByteArray(tt.expected))
		val := extractIntFromReturn(t, decodeResult)
		if val.Uint64() != tt.input {
			t.Errorf("decode_u64_from_little_endian: expected %d, got %d", tt.input, val.Uint64())
		}
	}
}

func TestBinaryEncodeDecodeF32LE(t *testing.T) {
	tests := []float32{
		0.0,
		1.0,
		-1.0,
		3.14159,
		float32(math.MaxFloat32),
	}

	for _, tt := range tests {
		encodeResult := BinaryBuiltins["binary.encode_f32_to_little_endian"].Fn(&object.Float{Value: float64(tt)})
		bytes := extractBytesFromReturn(t, encodeResult)
		if len(bytes) != 4 {
			t.Errorf("encode_f32_to_little_endian(%f): expected 4 bytes, got %d", tt, len(bytes))
			continue
		}

		decodeResult := BinaryBuiltins["binary.decode_f32_from_little_endian"].Fn(makeBinaryByteArray(bytes))
		val := extractFloatFromReturn(t, decodeResult)
		if float32(val) != tt {
			t.Errorf("f32 round-trip: expected %f, got %f", tt, val)
		}
	}
}

func TestBinaryEncodeDecodeF64LE(t *testing.T) {
	tests := []float64{
		0.0,
		1.0,
		-1.0,
		3.141592653589793,
		math.MaxFloat64,
	}

	for _, tt := range tests {
		encodeResult := BinaryBuiltins["binary.encode_f64_to_little_endian"].Fn(&object.Float{Value: tt})
		bytes := extractBytesFromReturn(t, encodeResult)
		if len(bytes) != 8 {
			t.Errorf("encode_f64_to_little_endian(%f): expected 8 bytes, got %d", tt, len(bytes))
			continue
		}

		decodeResult := BinaryBuiltins["binary.decode_f64_from_little_endian"].Fn(makeBinaryByteArray(bytes))
		val := extractFloatFromReturn(t, decodeResult)
		if val != tt {
			t.Errorf("f64 round-trip: expected %f, got %f", tt, val)
		}
	}
}

func TestBinaryEncodeDecodeI128LE(t *testing.T) {
	// Test zero
	encodeResult := BinaryBuiltins["binary.encode_i128_to_little_endian"].Fn(&object.Integer{Value: big.NewInt(0)})
	bytes := extractBytesFromReturn(t, encodeResult)
	if len(bytes) != 16 {
		t.Errorf("encode_i128_to_little_endian(0): expected 16 bytes, got %d", len(bytes))
	}

	decodeResult := BinaryBuiltins["binary.decode_i128_from_little_endian"].Fn(makeBinaryByteArray(bytes))
	val := extractIntFromReturn(t, decodeResult)
	if val.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("decode_i128_from_little_endian(0): expected 0, got %s", val.String())
	}

	// Test positive value
	testVal := big.NewInt(12345678901234567)
	encodeResult = BinaryBuiltins["binary.encode_i128_to_little_endian"].Fn(&object.Integer{Value: testVal})
	bytes = extractBytesFromReturn(t, encodeResult)
	decodeResult = BinaryBuiltins["binary.decode_i128_from_little_endian"].Fn(makeBinaryByteArray(bytes))
	val = extractIntFromReturn(t, decodeResult)
	if val.Cmp(testVal) != 0 {
		t.Errorf("i128 round-trip: expected %s, got %s", testVal.String(), val.String())
	}

	// Test negative value
	negVal := big.NewInt(-12345678901234567)
	encodeResult = BinaryBuiltins["binary.encode_i128_to_little_endian"].Fn(&object.Integer{Value: negVal})
	bytes = extractBytesFromReturn(t, encodeResult)
	decodeResult = BinaryBuiltins["binary.decode_i128_from_little_endian"].Fn(makeBinaryByteArray(bytes))
	val = extractIntFromReturn(t, decodeResult)
	if val.Cmp(negVal) != 0 {
		t.Errorf("i128 round-trip (negative): expected %s, got %s", negVal.String(), val.String())
	}
}

func TestBinaryRangeErrors(t *testing.T) {
	// i8 out of range
	result := BinaryBuiltins["binary.encode_i8"].Fn(&object.Integer{Value: big.NewInt(128)})
	rv, ok := result.(*object.ReturnValue)
	if !ok || rv.Values[1] == object.NIL {
		t.Error("encode_i8(128) should return error")
	}

	// u8 negative
	result = BinaryBuiltins["binary.encode_u8"].Fn(&object.Integer{Value: big.NewInt(-1)})
	rv, ok = result.(*object.ReturnValue)
	if !ok || rv.Values[1] == object.NIL {
		t.Error("encode_u8(-1) should return error")
	}

	// Wrong byte array length
	result = BinaryBuiltins["binary.decode_i32_from_little_endian"].Fn(makeBinaryByteArray([]byte{0x00, 0x00}))
	rv, ok = result.(*object.ReturnValue)
	if !ok || rv.Values[1] == object.NIL {
		t.Error("decode_i32_from_little_endian with 2 bytes should return error")
	}
}

func TestBinaryWrongArgCount(t *testing.T) {
	// No args
	result := BinaryBuiltins["binary.encode_i32_to_little_endian"].Fn()
	if _, ok := result.(*object.Error); !ok {
		t.Error("encode_i32_to_little_endian() with no args should return error")
	}

	// Too many args
	result = BinaryBuiltins["binary.encode_i32_to_little_endian"].Fn(
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(2)},
	)
	if _, ok := result.(*object.Error); !ok {
		t.Error("encode_i32_to_little_endian() with 2 args should return error")
	}
}
