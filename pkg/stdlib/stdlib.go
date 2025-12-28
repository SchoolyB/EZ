package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

// Package stdlib provides the standard library functions for the EZ language.
// It exports builtins that are registered with the interpreter at initialization.

import (
	"github.com/marshallburns/ez/pkg/object"
)

// GetAllBuiltins returns a map of all standard library builtins.
// This is called by the interpreter to register all stdlib functions.
func GetAllBuiltins() map[string]*object.Builtin {
	all := make(map[string]*object.Builtin)

	// Merge all module builtins
	for name, builtin := range StdBuiltins {
		all[name] = builtin
	}
	for name, builtin := range MathBuiltins {
		all[name] = builtin
	}
	for name, builtin := range ArraysBuiltins {
		all[name] = builtin
	}
	for name, builtin := range StringsBuiltins {
		all[name] = builtin
	}
	for name, builtin := range TimeBuiltins {
		all[name] = builtin
	}
	for name, builtin := range MapsBuiltins {
		all[name] = builtin
	}
	for name, builtin := range IOBuiltins {
		all[name] = builtin
	}
	for name, builtin := range OSBuiltins {
		all[name] = builtin
	}
	for name, builtin := range BytesBuiltins {
		all[name] = builtin
	}
	for name, builtin := range RandomBuiltins {
		all[name] = builtin
	}
	for name, builtin := range JsonBuiltins {
		all[name] = builtin
	}
	for name, builtin := range BinaryBuiltins {
		all[name] = builtin
	}
	for name, builtin := range DBBuiltins {
		all[name] = builtin
	}
	for name, builtin := range UUIDBuiltins {
		all[name] = builtin
	}

	return all
}
