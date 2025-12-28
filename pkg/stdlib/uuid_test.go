package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"regexp"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var compactUUIDRegex = regexp.MustCompile(`^[0-9a-f]{32}$`)

func TestUUIDCreate(t *testing.T) {
	createFn := UUIDBuiltins["uuid.create"].Fn

	// Generate multiple UUIDs and verify format
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		result := createFn()
		str, ok := result.(*object.String)
		if !ok {
			t.Fatalf("uuid.create() returned %T, want String", result)
		}

		// Verify UUID v4 format
		if !uuidRegex.MatchString(str.Value) {
			t.Errorf("uuid.create() = %s, not a valid UUID v4 format", str.Value)
		}

		// Check uniqueness
		if seen[str.Value] {
			t.Errorf("uuid.create() produced duplicate: %s", str.Value)
		}
		seen[str.Value] = true
	}
}

func TestUUIDCreateErrors(t *testing.T) {
	createFn := UUIDBuiltins["uuid.create"].Fn

	// Should error with arguments
	result := createFn(&object.String{Value: "arg"})
	if !isErrorObject(result) {
		t.Error("expected error when passing arguments to uuid.create()")
	}
}

func TestUUIDCreateCompact(t *testing.T) {
	compactFn := UUIDBuiltins["uuid.create_compact"].Fn

	// Generate multiple compact UUIDs
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		result := compactFn()
		str, ok := result.(*object.String)
		if !ok {
			t.Fatalf("uuid.create_compact() returned %T, want String", result)
		}

		// Verify compact format (32 hex chars, no hyphens)
		if !compactUUIDRegex.MatchString(str.Value) {
			t.Errorf("uuid.create_compact() = %s, not a valid compact UUID format", str.Value)
		}

		// Should be 32 characters
		if len(str.Value) != 32 {
			t.Errorf("uuid.create_compact() length = %d, want 32", len(str.Value))
		}

		// Check uniqueness
		if seen[str.Value] {
			t.Errorf("uuid.create_compact() produced duplicate: %s", str.Value)
		}
		seen[str.Value] = true
	}
}

func TestUUIDCreateCompactErrors(t *testing.T) {
	compactFn := UUIDBuiltins["uuid.create_compact"].Fn

	// Should error with arguments
	result := compactFn(&object.String{Value: "arg"})
	if !isErrorObject(result) {
		t.Error("expected error when passing arguments to uuid.create_compact()")
	}
}

func TestUUIDIsValid(t *testing.T) {
	isValidFn := UUIDBuiltins["uuid.is_valid"].Fn

	tests := []struct {
		input    string
		expected bool
	}{
		// Valid UUIDs
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"6ba7b810-9dad-11d1-80b4-00c04fd430c8", true},
		{"00000000-0000-0000-0000-000000000000", true},
		{"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF", true},
		// Invalid UUIDs
		{"", false},
		{"not-a-uuid", false},
		{"550e8400-e29b-41d4-a716", false},                     // too short
		{"550e8400-e29b-41d4-a716-4466554400001234", false},    // too long
		{"550e8400e29b41d4a716446655440000", false},            // missing hyphens
		{"550e8400-e29b-41d4-a716-44665544000g", false},        // invalid char
		{"550e8400-e29b-41d4-a716_446655440000", false},        // wrong separator
	}

	for _, tt := range tests {
		result := isValidFn(&object.String{Value: tt.input})
		b, ok := result.(*object.Boolean)
		if !ok {
			t.Fatalf("uuid.is_valid(%q) returned %T, want Boolean", tt.input, result)
		}
		if b.Value != tt.expected {
			t.Errorf("uuid.is_valid(%q) = %v, want %v", tt.input, b.Value, tt.expected)
		}
	}
}

func TestUUIDIsValidErrors(t *testing.T) {
	isValidFn := UUIDBuiltins["uuid.is_valid"].Fn

	// No arguments
	result := isValidFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Wrong type
	result = isValidFn(&object.Integer{Value: nil})
	if !isErrorObject(result) {
		t.Error("expected error for non-string argument")
	}

	// Too many arguments
	result = isValidFn(&object.String{Value: "a"}, &object.String{Value: "b"})
	if !isErrorObject(result) {
		t.Error("expected error for too many arguments")
	}
}

func TestUUIDNil(t *testing.T) {
	nilFn := UUIDBuiltins["uuid.NIL"].Fn

	result := nilFn()
	str, ok := result.(*object.String)
	if !ok {
		t.Fatalf("uuid.NIL() returned %T, want String", result)
	}

	expected := "00000000-0000-0000-0000-000000000000"
	if str.Value != expected {
		t.Errorf("uuid.NIL() = %s, want %s", str.Value, expected)
	}
}

func TestUUIDNilErrors(t *testing.T) {
	nilFn := UUIDBuiltins["uuid.NIL"].Fn

	// Should error with arguments
	result := nilFn(&object.String{Value: "arg"})
	if !isErrorObject(result) {
		t.Error("expected error when passing arguments to uuid.NIL()")
	}
}
