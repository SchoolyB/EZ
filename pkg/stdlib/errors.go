package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
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
