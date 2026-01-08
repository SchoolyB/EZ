package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

// This file bridges the stdlib builtins into the interpreter package.
// The actual builtin implementations live in pkg/stdlib/builtins.go.
// We also keep getEZTypeName here since it uses the interpreter's type aliases.

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

// getEZTypeName returns the EZ language type name for an object
// For integers, returns the declared type (e.g., "u64", "i32") instead of generic "INTEGER"
// For arrays and maps, returns the typed format (e.g., "[string]", "map[string:int]")
func getEZTypeName(obj Object) string {
	switch v := obj.(type) {
	case *Integer:
		return v.GetDeclaredType()
	case *Float:
		return "float"
	case *String:
		return "string"
	case *Boolean:
		return "bool"
	case *Char:
		return "char"
	case *Byte:
		return "byte"
	case *Array:
		if v.ElementType != "" {
			return "[" + v.ElementType + "]"
		}
		return "array"
	case *Map:
		if v.KeyType != "" && v.ValueType != "" {
			return "map[" + v.KeyType + ":" + v.ValueType + "]"
		}
		return "map"
	case *Struct:
		if v.TypeName != "" {
			return v.TypeName
		}
		return "struct"
	case *EnumValue:
		return v.EnumType
	case *Nil:
		return "nil"
	case *Function:
		return "function"
	default:
		return string(obj.Type())
	}
}
