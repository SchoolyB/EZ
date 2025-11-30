package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

// This file re-exports types from the shared object package for backward compatibility.
// The actual type definitions are in pkg/object/object.go

import (
	"github.com/marshallburns/ez/pkg/object"
)

// Re-export types from object package
type (
	ObjectType      = object.ObjectType
	Object          = object.Object
	Integer         = object.Integer
	Float           = object.Float
	String          = object.String
	Char            = object.Char
	Boolean         = object.Boolean
	Nil             = object.Nil
	ReturnValue     = object.ReturnValue
	Error           = object.Error
	Function        = object.Function
	BuiltinFunction = object.BuiltinFunction
	Builtin         = object.Builtin
	Array           = object.Array
	Map             = object.Map
	MapPair         = object.MapPair
	Struct          = object.Struct
	Break           = object.Break
	Continue        = object.Continue
	StructDef       = object.StructDef
	Enum            = object.Enum
	EnumValue       = object.EnumValue
	Environment     = object.Environment
)

// Re-export constants
const (
	INTEGER_OBJ      = object.INTEGER_OBJ
	FLOAT_OBJ        = object.FLOAT_OBJ
	STRING_OBJ       = object.STRING_OBJ
	CHAR_OBJ         = object.CHAR_OBJ
	BOOLEAN_OBJ      = object.BOOLEAN_OBJ
	NIL_OBJ          = object.NIL_OBJ
	RETURN_VALUE_OBJ = object.RETURN_VALUE_OBJ
	ERROR_OBJ        = object.ERROR_OBJ
	FUNCTION_OBJ     = object.FUNCTION_OBJ
	BUILTIN_OBJ      = object.BUILTIN_OBJ
	ARRAY_OBJ        = object.ARRAY_OBJ
	MAP_OBJ          = object.MAP_OBJ
	STRUCT_OBJ       = object.STRUCT_OBJ
	BREAK_OBJ        = object.BREAK_OBJ
	CONTINUE_OBJ     = object.CONTINUE_OBJ
	ENUM_OBJ         = object.ENUM_OBJ
	ENUM_VALUE_OBJ   = object.ENUM_VALUE_OBJ
)

// Note: Singleton values (NIL, TRUE, FALSE) are defined in evaluator.go
// to maintain compatibility with existing code that uses them.

// Re-export constructor functions
var (
	NewEnvironment         = object.NewEnvironment
	NewEnclosedEnvironment = object.NewEnclosedEnvironment
	NewMap                 = object.NewMap
	HashKey                = object.HashKey
)
