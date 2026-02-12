package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// Helper to extract value and error from multi-value return
func extractValueAndError(obj object.Object) (object.Object, object.Object) {
	rv, ok := obj.(*object.ReturnValue)
	if !ok || len(rv.Values) != 2 {
		return nil, nil
	}
	return rv.Values[0], rv.Values[1]
}

// ============================================================================
// Base64 Tests
// ============================================================================

func TestBase64Encode(t *testing.T) {
	encodeFn := EncodingBuiltins["encoding.base64_encode"].Fn

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"Hello, World!", "SGVsbG8sIFdvcmxkIQ=="},
		{"hello", "aGVsbG8="},
		{"a", "YQ=="},
		{"ab", "YWI="},
		{"abc", "YWJj"},
		{"The quick brown fox", "VGhlIHF1aWNrIGJyb3duIGZveA=="},
	}

	for _, tt := range tests {
		result := encodeFn(&object.String{Value: tt.input})
		str, ok := result.(*object.String)
		if !ok {
			t.Fatalf("encoding.base64_encode(%q) returned %T, want String", tt.input, result)
		}
		if str.Value != tt.expected {
			t.Errorf("encoding.base64_encode(%q) = %s, want %s", tt.input, str.Value, tt.expected)
		}
	}
}

func TestBase64EncodeErrors(t *testing.T) {
	encodeFn := EncodingBuiltins["encoding.base64_encode"].Fn

	// No arguments
	result := encodeFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Wrong type
	result = encodeFn(&object.Integer{Value: nil})
	if !isErrorObject(result) {
		t.Error("expected error for non-string argument")
	}
}

func TestBase64Decode(t *testing.T) {
	decodeFn := EncodingBuiltins["encoding.base64_decode"].Fn

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"SGVsbG8sIFdvcmxkIQ==", "Hello, World!"},
		{"aGVsbG8=", "hello"},
		{"YQ==", "a"},
		{"YWI=", "ab"},
		{"YWJj", "abc"},
	}

	for _, tt := range tests {
		result := decodeFn(&object.String{Value: tt.input})
		val, err := extractValueAndError(result)
		if err != object.NIL {
			t.Errorf("encoding.base64_decode(%q) returned error", tt.input)
			continue
		}
		str, ok := val.(*object.String)
		if !ok {
			t.Fatalf("encoding.base64_decode(%q) value is %T, want String", tt.input, val)
		}
		if str.Value != tt.expected {
			t.Errorf("encoding.base64_decode(%q) = %s, want %s", tt.input, str.Value, tt.expected)
		}
	}
}

func TestBase64DecodeInvalid(t *testing.T) {
	decodeFn := EncodingBuiltins["encoding.base64_decode"].Fn

	invalidInputs := []string{
		"!!!invalid!!!",
		"not valid base64",
		"SGVsbG8=====", // too much padding
	}

	for _, input := range invalidInputs {
		result := decodeFn(&object.String{Value: input})
		_, err := extractValueAndError(result)
		if err == object.NIL {
			t.Errorf("encoding.base64_decode(%q) should return error", input)
		}
	}
}

func TestBase64DecodeErrors(t *testing.T) {
	decodeFn := EncodingBuiltins["encoding.base64_decode"].Fn

	// No arguments
	result := decodeFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Wrong type
	result = decodeFn(&object.Integer{Value: nil})
	if !isErrorObject(result) {
		t.Error("expected error for non-string argument")
	}
}

// ============================================================================
// Hex Tests
// ============================================================================

func TestHexEncode(t *testing.T) {
	encodeFn := EncodingBuiltins["encoding.hex_encode"].Fn

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"Hello, World!", "48656c6c6f2c20576f726c6421"},
		{"hello", "68656c6c6f"},
		{"abc", "616263"},
		{"\x00\xff", "00ff"},
	}

	for _, tt := range tests {
		result := encodeFn(&object.String{Value: tt.input})
		str, ok := result.(*object.String)
		if !ok {
			t.Fatalf("encoding.hex_encode(%q) returned %T, want String", tt.input, result)
		}
		if str.Value != tt.expected {
			t.Errorf("encoding.hex_encode(%q) = %s, want %s", tt.input, str.Value, tt.expected)
		}
	}
}

func TestHexEncodeErrors(t *testing.T) {
	encodeFn := EncodingBuiltins["encoding.hex_encode"].Fn

	// No arguments
	result := encodeFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Wrong type
	result = encodeFn(&object.Integer{Value: nil})
	if !isErrorObject(result) {
		t.Error("expected error for non-string argument")
	}
}

func TestHexDecode(t *testing.T) {
	decodeFn := EncodingBuiltins["encoding.hex_decode"].Fn

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"48656c6c6f2c20576f726c6421", "Hello, World!"},
		{"68656c6c6f", "hello"},
		{"616263", "abc"},
		{"00ff", "\x00\xff"},
		{"ABCDEF", "\xab\xcd\xef"}, // uppercase should work
	}

	for _, tt := range tests {
		result := decodeFn(&object.String{Value: tt.input})
		val, err := extractValueAndError(result)
		if err != object.NIL {
			t.Errorf("encoding.hex_decode(%q) returned error", tt.input)
			continue
		}
		str, ok := val.(*object.String)
		if !ok {
			t.Fatalf("encoding.hex_decode(%q) value is %T, want String", tt.input, val)
		}
		if str.Value != tt.expected {
			t.Errorf("encoding.hex_decode(%q) = %q, want %q", tt.input, str.Value, tt.expected)
		}
	}
}

func TestHexDecodeInvalid(t *testing.T) {
	decodeFn := EncodingBuiltins["encoding.hex_decode"].Fn

	invalidInputs := []string{
		"xyz",      // invalid hex chars
		"12345",    // odd length
		"ghijklmn", // invalid chars
	}

	for _, input := range invalidInputs {
		result := decodeFn(&object.String{Value: input})
		_, err := extractValueAndError(result)
		if err == object.NIL {
			t.Errorf("encoding.hex_decode(%q) should return error", input)
		}
	}
}

func TestHexDecodeErrors(t *testing.T) {
	decodeFn := EncodingBuiltins["encoding.hex_decode"].Fn

	// No arguments
	result := decodeFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Wrong type
	result = decodeFn(&object.Integer{Value: nil})
	if !isErrorObject(result) {
		t.Error("expected error for non-string argument")
	}
}

// ============================================================================
// URL Encoding Tests
// ============================================================================

func TestURLEncode(t *testing.T) {
	encodeFn := EncodingBuiltins["encoding.url_encode"].Fn

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "hello"},
		{"hello world", "hello+world"},
		{"foo=bar&baz=qux", "foo%3Dbar%26baz%3Dqux"},
		{"special chars: !@#$%", "special+chars%3A+%21%40%23%24%25"},
	}

	for _, tt := range tests {
		result := encodeFn(&object.String{Value: tt.input})
		str, ok := result.(*object.String)
		if !ok {
			t.Fatalf("encoding.url_encode(%q) returned %T, want String", tt.input, result)
		}
		if str.Value != tt.expected {
			t.Errorf("encoding.url_encode(%q) = %s, want %s", tt.input, str.Value, tt.expected)
		}
	}
}

func TestURLEncodeErrors(t *testing.T) {
	encodeFn := EncodingBuiltins["encoding.url_encode"].Fn

	// No arguments
	result := encodeFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Wrong type
	result = encodeFn(&object.Integer{Value: nil})
	if !isErrorObject(result) {
		t.Error("expected error for non-string argument")
	}
}

func TestURLDecode(t *testing.T) {
	decodeFn := EncodingBuiltins["encoding.url_decode"].Fn

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "hello"},
		{"hello+world", "hello world"},
		{"hello%20world", "hello world"},
		{"foo%3Dbar%26baz%3Dqux", "foo=bar&baz=qux"},
	}

	for _, tt := range tests {
		result := decodeFn(&object.String{Value: tt.input})
		val, err := extractValueAndError(result)
		if err != object.NIL {
			t.Errorf("encoding.url_decode(%q) returned error", tt.input)
			continue
		}
		str, ok := val.(*object.String)
		if !ok {
			t.Fatalf("encoding.url_decode(%q) value is %T, want String", tt.input, val)
		}
		if str.Value != tt.expected {
			t.Errorf("encoding.url_decode(%q) = %s, want %s", tt.input, str.Value, tt.expected)
		}
	}
}

func TestURLDecodeInvalid(t *testing.T) {
	decodeFn := EncodingBuiltins["encoding.url_decode"].Fn

	invalidInputs := []string{
		"%",   // incomplete escape
		"%x",  // incomplete escape
		"%zz", // invalid hex
	}

	for _, input := range invalidInputs {
		result := decodeFn(&object.String{Value: input})
		_, err := extractValueAndError(result)
		if err == object.NIL {
			t.Errorf("encoding.url_decode(%q) should return error", input)
		}
	}
}

func TestURLDecodeErrors(t *testing.T) {
	decodeFn := EncodingBuiltins["encoding.url_decode"].Fn

	// No arguments
	result := decodeFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Wrong type
	result = decodeFn(&object.Integer{Value: nil})
	if !isErrorObject(result) {
		t.Error("expected error for non-string argument")
	}
}

// ============================================================================
// Roundtrip Tests
// ============================================================================

func TestBase64Roundtrip(t *testing.T) {
	encodeFn := EncodingBuiltins["encoding.base64_encode"].Fn
	decodeFn := EncodingBuiltins["encoding.base64_decode"].Fn

	inputs := []string{"", "hello", "Hello, World!", "binary\x00data\xff"}

	for _, input := range inputs {
		encoded := encodeFn(&object.String{Value: input}).(*object.String)
		result := decodeFn(&object.String{Value: encoded.Value})
		decoded, _ := extractValueAndError(result)
		if decoded.(*object.String).Value != input {
			t.Errorf("base64 roundtrip failed for %q", input)
		}
	}
}

func TestHexRoundtrip(t *testing.T) {
	encodeFn := EncodingBuiltins["encoding.hex_encode"].Fn
	decodeFn := EncodingBuiltins["encoding.hex_decode"].Fn

	inputs := []string{"", "hello", "Hello, World!", "binary\x00data\xff"}

	for _, input := range inputs {
		encoded := encodeFn(&object.String{Value: input}).(*object.String)
		result := decodeFn(&object.String{Value: encoded.Value})
		decoded, _ := extractValueAndError(result)
		if decoded.(*object.String).Value != input {
			t.Errorf("hex roundtrip failed for %q", input)
		}
	}
}

func TestURLRoundtrip(t *testing.T) {
	encodeFn := EncodingBuiltins["encoding.url_encode"].Fn
	decodeFn := EncodingBuiltins["encoding.url_decode"].Fn

	inputs := []string{"", "hello", "hello world", "foo=bar&baz=qux"}

	for _, input := range inputs {
		encoded := encodeFn(&object.String{Value: input}).(*object.String)
		result := decodeFn(&object.String{Value: encoded.Value})
		decoded, _ := extractValueAndError(result)
		if decoded.(*object.String).Value != input {
			t.Errorf("URL roundtrip failed for %q", input)
		}
	}
}
