package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

// This file bridges the stdlib builtins into the interpreter package.
// The actual builtin implementations live in pkg/stdlib/builtins.go.

import (
	"github.com/marshallburns/ez/pkg/object"
	"github.com/marshallburns/ez/pkg/stdlib"
)

// builtins maps function names to their implementations.
// The actual implementations are in pkg/stdlib/.
var builtins map[string]*object.Builtin

func init() {
	// Get all builtins from the stdlib package
	builtins = stdlib.GetAllBuiltins()
}
