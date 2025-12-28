package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

func TestCryptoSHA256(t *testing.T) {
	sha256Fn := CryptoBuiltins["crypto.sha256"].Fn

	tests := []struct {
		input    string
		expected string
	}{
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"hello", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
		{"Hello, World!", "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"},
	}

	for _, tt := range tests {
		result := sha256Fn(&object.String{Value: tt.input})
		str, ok := result.(*object.String)
		if !ok {
			t.Fatalf("crypto.sha256(%q) returned %T, want String", tt.input, result)
		}
		if str.Value != tt.expected {
			t.Errorf("crypto.sha256(%q) = %s, want %s", tt.input, str.Value, tt.expected)
		}
	}
}

func TestCryptoSHA256Errors(t *testing.T) {
	sha256Fn := CryptoBuiltins["crypto.sha256"].Fn

	// No args
	result := sha256Fn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Wrong type
	result = sha256Fn(&object.Integer{Value: big.NewInt(123)})
	if !isErrorObject(result) {
		t.Error("expected error for non-string argument")
	}

	// Too many args
	result = sha256Fn(&object.String{Value: "a"}, &object.String{Value: "b"})
	if !isErrorObject(result) {
		t.Error("expected error for too many arguments")
	}
}

func TestCryptoSHA512(t *testing.T) {
	sha512Fn := CryptoBuiltins["crypto.sha512"].Fn

	tests := []struct {
		input    string
		expected string
	}{
		{"", "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"},
		{"hello", "9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca72323c3d99ba5c11d7c7acc6e14b8c5da0c4663475c2e5c3adef46f73bcdec043"},
	}

	for _, tt := range tests {
		result := sha512Fn(&object.String{Value: tt.input})
		str, ok := result.(*object.String)
		if !ok {
			t.Fatalf("crypto.sha512(%q) returned %T, want String", tt.input, result)
		}
		if str.Value != tt.expected {
			t.Errorf("crypto.sha512(%q) = %s, want %s", tt.input, str.Value, tt.expected)
		}
	}
}

func TestCryptoSHA512Errors(t *testing.T) {
	sha512Fn := CryptoBuiltins["crypto.sha512"].Fn

	// No args
	result := sha512Fn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Wrong type
	result = sha512Fn(&object.Integer{Value: big.NewInt(123)})
	if !isErrorObject(result) {
		t.Error("expected error for non-string argument")
	}
}

func TestCryptoMD5(t *testing.T) {
	md5Fn := CryptoBuiltins["crypto.md5"].Fn

	tests := []struct {
		input    string
		expected string
	}{
		{"", "d41d8cd98f00b204e9800998ecf8427e"},
		{"hello", "5d41402abc4b2a76b9719d911017c592"},
		{"Hello, World!", "65a8e27d8879283831b664bd8b7f0ad4"},
	}

	for _, tt := range tests {
		result := md5Fn(&object.String{Value: tt.input})
		str, ok := result.(*object.String)
		if !ok {
			t.Fatalf("crypto.md5(%q) returned %T, want String", tt.input, result)
		}
		if str.Value != tt.expected {
			t.Errorf("crypto.md5(%q) = %s, want %s", tt.input, str.Value, tt.expected)
		}
	}
}

func TestCryptoMD5Errors(t *testing.T) {
	md5Fn := CryptoBuiltins["crypto.md5"].Fn

	// No args
	result := md5Fn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Wrong type
	result = md5Fn(&object.Integer{Value: big.NewInt(123)})
	if !isErrorObject(result) {
		t.Error("expected error for non-string argument")
	}
}

func TestCryptoRandomBytes(t *testing.T) {
	randomBytesFn := CryptoBuiltins["crypto.random_bytes"].Fn

	// Test various lengths
	for _, length := range []int64{0, 1, 16, 32, 64} {
		result := randomBytesFn(&object.Integer{Value: big.NewInt(length)})
		arr, ok := result.(*object.Array)
		if !ok {
			t.Fatalf("crypto.random_bytes(%d) returned %T, want Array", length, result)
		}
		if int64(len(arr.Elements)) != length {
			t.Errorf("crypto.random_bytes(%d) returned array of length %d", length, len(arr.Elements))
		}
		// Verify all elements are bytes
		for i, elem := range arr.Elements {
			if _, ok := elem.(*object.Byte); !ok {
				t.Errorf("crypto.random_bytes(%d)[%d] is %T, want Byte", length, i, elem)
			}
		}
	}

	// Test randomness - two calls should (almost certainly) produce different results
	result1 := randomBytesFn(&object.Integer{Value: big.NewInt(32)})
	result2 := randomBytesFn(&object.Integer{Value: big.NewInt(32)})
	arr1 := result1.(*object.Array)
	arr2 := result2.(*object.Array)

	same := true
	for i := range arr1.Elements {
		if arr1.Elements[i].(*object.Byte).Value != arr2.Elements[i].(*object.Byte).Value {
			same = false
			break
		}
	}
	if same {
		t.Error("crypto.random_bytes() produced identical results twice in a row")
	}
}

func TestCryptoRandomBytesErrors(t *testing.T) {
	randomBytesFn := CryptoBuiltins["crypto.random_bytes"].Fn

	// No args
	result := randomBytesFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Wrong type
	result = randomBytesFn(&object.String{Value: "16"})
	if !isErrorObject(result) {
		t.Error("expected error for non-integer argument")
	}

	// Negative length
	result = randomBytesFn(&object.Integer{Value: big.NewInt(-1)})
	if !isErrorObject(result) {
		t.Error("expected error for negative length")
	}
}

func TestCryptoRandomHex(t *testing.T) {
	randomHexFn := CryptoBuiltins["crypto.random_hex"].Fn

	// Test various lengths
	for _, length := range []int64{0, 1, 16, 32} {
		result := randomHexFn(&object.Integer{Value: big.NewInt(length)})
		str, ok := result.(*object.String)
		if !ok {
			t.Fatalf("crypto.random_hex(%d) returned %T, want String", length, result)
		}
		expectedLen := int(length * 2) // hex encoding doubles the length
		if len(str.Value) != expectedLen {
			t.Errorf("crypto.random_hex(%d) returned string of length %d, want %d", length, len(str.Value), expectedLen)
		}
		// Verify it's valid hex
		if length > 0 {
			if _, err := hex.DecodeString(str.Value); err != nil {
				t.Errorf("crypto.random_hex(%d) returned invalid hex: %s", length, str.Value)
			}
		}
	}

	// Test randomness
	result1 := randomHexFn(&object.Integer{Value: big.NewInt(32)})
	result2 := randomHexFn(&object.Integer{Value: big.NewInt(32)})
	if result1.(*object.String).Value == result2.(*object.String).Value {
		t.Error("crypto.random_hex() produced identical results twice in a row")
	}
}

func TestCryptoRandomHexErrors(t *testing.T) {
	randomHexFn := CryptoBuiltins["crypto.random_hex"].Fn

	// No args
	result := randomHexFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Wrong type
	result = randomHexFn(&object.String{Value: "16"})
	if !isErrorObject(result) {
		t.Error("expected error for non-integer argument")
	}

	// Negative length
	result = randomHexFn(&object.Integer{Value: big.NewInt(-1)})
	if !isErrorObject(result) {
		t.Error("expected error for negative length")
	}
}
