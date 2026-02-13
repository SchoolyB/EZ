package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

// Package stdlib provides the standard library functions for the EZ language.
// It exports builtins that are registered with the interpreter at initialization.

import (
	"github.com/marshallburns/ez/pkg/object"
)

// GetAllBuiltins returns a map of all standard library builtins.
func GetAllBuiltins() map[string]*object.Builtin {
	all := make(map[string]*object.Builtin)
	for _, module := range []map[string]*object.Builtin{
		StdBuiltins, MathBuiltins, ArraysBuiltins, StringsBuiltins,
		TimeBuiltins, MapsBuiltins, IOBuiltins, OSBuiltins,
		BytesBuiltins, RandomBuiltins, JsonBuiltins, BinaryBuiltins,
		DBBuiltins, UUIDBuiltins, EncodingBuiltins, CryptoBuiltins,
		HttpBuiltins, CsvBuiltins, RegexBuiltins, ServerBuiltins,
	} {
		for name, builtin := range module {
			all[name] = builtin
		}
	}
	return all
}
