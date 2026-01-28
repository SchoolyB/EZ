package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"

	"github.com/marshallburns/ez/pkg/object"
)

// CreateStdlibError creates an Error struct for recoverable errors in tuple returns.
// Use this for stdlib functions that return (result, error) tuples.
func CreateStdlibError(code, message string) *object.Struct {
	return &object.Struct{
		TypeName: "Error",
		Fields: map[string]object.Object{
			"message": &object.String{Value: message},
			"code":    &object.String{Value: code},
		},
	}
}

// NewError creates a simple error with a formatted message.
// Use this for validation errors that halt execution.
func NewError(format string, args ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, args...)}
}

// NewErrorWithCode creates an error with both code and formatted message.
func NewErrorWithCode(code, format string, args ...interface{}) *object.Error {
	return &object.Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}
