package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"strings"

	"github.com/marshallburns/ez/pkg/object"
)

// =============================================================================
// Argument Count Validation
// =============================================================================

// ValidateArgCount checks that exactly 'expected' arguments were provided.
func ValidateArgCount(args []object.Object, expected int, funcName string) *object.Error {
	if len(args) != expected {
		return NewError("%s() takes exactly %d argument(s), got %d", funcName, expected, len(args))
	}
	return nil
}

// ValidateArgCountMin checks that at least 'min' arguments were provided.
func ValidateArgCountMin(args []object.Object, min int, funcName string) *object.Error {
	if len(args) < min {
		return NewError("%s() takes at least %d argument(s), got %d", funcName, min, len(args))
	}
	return nil
}

// ValidateArgCountRange checks that argument count is between min and max (inclusive).
func ValidateArgCountRange(args []object.Object, min, max int, funcName string) *object.Error {
	if len(args) < min || len(args) > max {
		return NewError("%s() takes %d-%d arguments, got %d", funcName, min, max, len(args))
	}
	return nil
}

// =============================================================================
// Type Assertion Helpers
// =============================================================================

// GetStringArg extracts a string argument at the given index.
func GetStringArg(args []object.Object, index int, funcName string) (*object.String, *object.Error) {
	if index >= len(args) {
		return nil, NewError("%s() missing argument %d", funcName, index+1)
	}
	if str, ok := args[index].(*object.String); ok {
		return str, nil
	}
	return nil, NewErrorWithCode("E7002", "%s() argument %d must be a string", funcName, index+1)
}

// GetIntArg extracts an integer argument at the given index.
func GetIntArg(args []object.Object, index int, funcName string) (*object.Integer, *object.Error) {
	if index >= len(args) {
		return nil, NewError("%s() missing argument %d", funcName, index+1)
	}
	if i, ok := args[index].(*object.Integer); ok {
		return i, nil
	}
	return nil, NewErrorWithCode("E7002", "%s() argument %d must be an integer", funcName, index+1)
}

// GetFloatArg extracts a float argument at the given index.
func GetFloatArg(args []object.Object, index int, funcName string) (*object.Float, *object.Error) {
	if index >= len(args) {
		return nil, NewError("%s() missing argument %d", funcName, index+1)
	}
	if f, ok := args[index].(*object.Float); ok {
		return f, nil
	}
	return nil, NewErrorWithCode("E7002", "%s() argument %d must be a float", funcName, index+1)
}

// GetArrayArg extracts an array argument at the given index.
func GetArrayArg(args []object.Object, index int, funcName string) (*object.Array, *object.Error) {
	if index >= len(args) {
		return nil, NewError("%s() missing argument %d", funcName, index+1)
	}
	if arr, ok := args[index].(*object.Array); ok {
		return arr, nil
	}
	return nil, NewErrorWithCode("E7002", "%s() argument %d must be an array", funcName, index+1)
}

// GetMapArg extracts a map argument at the given index.
func GetMapArg(args []object.Object, index int, funcName string) (*object.Map, *object.Error) {
	if index >= len(args) {
		return nil, NewError("%s() missing argument %d", funcName, index+1)
	}
	if m, ok := args[index].(*object.Map); ok {
		return m, nil
	}
	return nil, NewErrorWithCode("E7002", "%s() argument %d must be a map", funcName, index+1)
}

// GetBoolArg extracts a boolean argument at the given index.
func GetBoolArg(args []object.Object, index int, funcName string) (*object.Boolean, *object.Error) {
	if index >= len(args) {
		return nil, NewError("%s() missing argument %d", funcName, index+1)
	}
	if b, ok := args[index].(*object.Boolean); ok {
		return b, nil
	}
	return nil, NewErrorWithCode("E7002", "%s() argument %d must be a boolean", funcName, index+1)
}

// =============================================================================
// String Building Utilities
// =============================================================================

// JoinObjects joins object Inspect() values with a separator efficiently.
func JoinObjects(elements []object.Object, sep string) string {
	if len(elements) == 0 {
		return ""
	}
	if len(elements) == 1 {
		return elements[0].Inspect()
	}

	var sb strings.Builder
	sb.WriteString(elements[0].Inspect())
	for _, el := range elements[1:] {
		sb.WriteString(sep)
		sb.WriteString(el.Inspect())
	}
	return sb.String()
}
