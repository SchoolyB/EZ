package typechecker
// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"

	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/errors"
)

// TypeKind represents the category of a type
type TypeKind int

const (
	PrimitiveType TypeKind = iota
	ArrayType
	StructType
	EnumType
	FunctionType
	VoidType
)

// Type represents a type in the EZ type system
type Type struct {
	Name       string
	Kind       TypeKind
	ElementType *Type              // For arrays
	Fields     map[string]*Type   // For structs
	Size       int                // For fixed-size arrays, -1 for dynamic
}

// FunctionSignature represents a function's type signature
type FunctionSignature struct {
	Name        string
	Parameters  []*Parameter
	ReturnTypes []string
}

// Parameter represents a function parameter with type
type Parameter struct {
	Name string
	Type string
}

// TypeChecker validates types in an EZ program
type TypeChecker struct {
	types      map[string]*Type              // All known types
	functions  map[string]*FunctionSignature // All function signatures
	variables  map[string]string             // Variable name -> type name (global scope)
	errors     *errors.EZErrorList
	source     string
	filename   string
}

// NewTypeChecker creates a new type checker
func NewTypeChecker(source, filename string) *TypeChecker {
	tc := &TypeChecker{
		types:     make(map[string]*Type),
		functions: make(map[string]*FunctionSignature),
		variables: make(map[string]string),
		errors:    errors.NewErrorList(),
		source:    source,
		filename:  filename,
	}

	// Register built-in primitive types
	tc.registerBuiltinTypes()

	return tc
}

// registerBuiltinTypes adds all built-in types to the registry
func (tc *TypeChecker) registerBuiltinTypes() {
	primitives := []string{
		// Signed integers
		"i8", "i16", "i32", "i64", "int",
		// Unsigned integers
		"u8", "u16", "u32", "u64", "uint",
		// Floats
		"f32", "f64", "float",
		// Other primitives
		"bool", "char", "string",
		// Special
		"void", "nil",
	}

	for _, name := range primitives {
		tc.types[name] = &Type{
			Name: name,
			Kind: PrimitiveType,
		}
	}

	// Register built-in Error struct
	tc.types["Error"] = &Type{
		Name: "Error",
		Kind: StructType,
		Fields: map[string]*Type{
			"message": {Name: "string", Kind: PrimitiveType},
			"code":    {Name: "int", Kind: PrimitiveType},
		},
	}
}

// TypeExists checks if a type name is registered
func (tc *TypeChecker) TypeExists(typeName string) bool {
	// Check for array types: [type] or [type, size]
	if len(typeName) > 2 && typeName[0] == '[' {
		// For now, just check if it's an array syntax
		// Full validation will happen in CheckArrayType
		return true
	}

	_, exists := tc.types[typeName]
	return exists
}

// RegisterType adds a user-defined type to the registry
func (tc *TypeChecker) RegisterType(name string, t *Type) {
	tc.types[name] = t
}

// RegisterFunction adds a function signature to the registry
func (tc *TypeChecker) RegisterFunction(name string, sig *FunctionSignature) {
	tc.functions[name] = sig
}

// GetType retrieves a type by name
func (tc *TypeChecker) GetType(name string) (*Type, bool) {
	t, ok := tc.types[name]
	return t, ok
}

// Errors returns the error list
func (tc *TypeChecker) Errors() *errors.EZErrorList {
	return tc.errors
}

// addError adds a type error
func (tc *TypeChecker) addError(code errors.ErrorCode, message string, line, column int) {
	sourceLine := ""
	if tc.source != "" {
		sourceLine = errors.GetSourceLine(tc.source, line)
	}

	err := errors.NewErrorWithSource(
		code,
		message,
		tc.filename,
		line,
		column,
		sourceLine,
	)
	tc.errors.AddError(err)
}

// CheckProgram performs type checking on the entire program
func (tc *TypeChecker) CheckProgram(program *ast.Program) bool {
	// Phase 1: Register all user-defined types (structs, enums)
	for _, stmt := range program.Statements {
		switch node := stmt.(type) {
		case *ast.StructDeclaration:
			tc.registerStructType(node)
		case *ast.EnumDeclaration:
			tc.registerEnumType(node)
		}
	}

	// Phase 2: Validate all global declarations
	for _, stmt := range program.Statements {
		switch node := stmt.(type) {
		case *ast.StructDeclaration:
			tc.checkStructDeclaration(node)
		case *ast.EnumDeclaration:
			tc.checkEnumDeclaration(node)
		case *ast.VariableDeclaration:
			tc.checkGlobalVariableDeclaration(node)
		case *ast.FunctionDeclaration:
			tc.checkFunctionDeclaration(node)
		}
	}

	// Phase 3: Type check function bodies
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionDeclaration); ok {
			tc.checkFunctionBody(fn)
		}
	}

	errCount, _ := tc.errors.Count()
	return errCount == 0
}

// registerStructType registers a struct type name (without validating fields yet)
func (tc *TypeChecker) registerStructType(node *ast.StructDeclaration) {
	structType := &Type{
		Name:   node.Name.Value,
		Kind:   StructType,
		Fields: make(map[string]*Type),
	}
	tc.RegisterType(node.Name.Value, structType)
}

// registerEnumType registers an enum type name
func (tc *TypeChecker) registerEnumType(node *ast.EnumDeclaration) {
	enumType := &Type{
		Name: node.Name.Value,
		Kind: EnumType,
	}
	tc.RegisterType(node.Name.Value, enumType)
}

// checkStructDeclaration validates a struct's field types
func (tc *TypeChecker) checkStructDeclaration(node *ast.StructDeclaration) {
	structType, _ := tc.GetType(node.Name.Value)

	for _, field := range node.Fields {
		// Check if field type exists
		if !tc.TypeExists(field.TypeName) {
			tc.addError(
				errors.E2006,
				fmt.Sprintf("undefined type '%s' in struct '%s'", field.TypeName, node.Name.Value),
				field.Name.Token.Line,
				field.Name.Token.Column,
			)
			continue
		}

		// Add field to struct type
		fieldType, _ := tc.GetType(field.TypeName)
		structType.Fields[field.Name.Value] = fieldType
	}
}

// checkEnumDeclaration validates an enum declaration
func (tc *TypeChecker) checkEnumDeclaration(node *ast.EnumDeclaration) {
	// TODO: Validate enum base type if specified
	// For now, enums are just registered as types
}

// checkGlobalVariableDeclaration validates a global variable declaration
func (tc *TypeChecker) checkGlobalVariableDeclaration(node *ast.VariableDeclaration) {
	// Check each variable in the declaration
	for i, name := range node.Names {
		varName := name.Value

		// Determine the type
		var typeName string
		if node.TypeName != "" {
			typeName = node.TypeName
		} else if node.Value != nil {
			// Type inference from value (for multi-return assignments)
			// For now, we'll handle this in the function body checking
			continue
		}

		// Check if type exists
		if !tc.TypeExists(typeName) {
			tc.addError(
				errors.E2006,
				fmt.Sprintf("undefined type '%s'", typeName),
				name.Token.Line,
				name.Token.Column,
			)
			continue
		}

		// Register variable
		tc.variables[varName] = typeName

		// If there's an initial value, check type compatibility
		if node.Value != nil && i == 0 {
			// TODO: Check that value type matches declared type
			// This requires expression type inference
		}
	}
}

// checkFunctionDeclaration validates a function's signature
func (tc *TypeChecker) checkFunctionDeclaration(node *ast.FunctionDeclaration) {
	sig := &FunctionSignature{
		Name:        node.Name.Value,
		Parameters:  []*Parameter{},
		ReturnTypes: node.ReturnTypes,
	}

	// Check parameter types
	for _, param := range node.Parameters {
		if !tc.TypeExists(param.TypeName) {
			tc.addError(
				errors.E2006,
				fmt.Sprintf("undefined type '%s' for parameter '%s'", param.TypeName, param.Name.Value),
				param.Name.Token.Line,
				param.Name.Token.Column,
			)
		}
		sig.Parameters = append(sig.Parameters, &Parameter{
			Name: param.Name.Value,
			Type: param.TypeName,
		})
	}

	// Check return types
	for _, returnType := range node.ReturnTypes {
		if !tc.TypeExists(returnType) {
			tc.addError(
				errors.E2006,
				fmt.Sprintf("undefined return type '%s' in function '%s'", returnType, node.Name.Value),
				node.Name.Token.Line,
				node.Name.Token.Column,
			)
		}
	}

	tc.RegisterFunction(node.Name.Value, sig)
}

// checkFunctionBody validates the body of a function
func (tc *TypeChecker) checkFunctionBody(node *ast.FunctionDeclaration) {
	// TODO: Implement function body type checking
	// This will check:
	// - Local variable declarations
	// - Assignments
	// - Expressions
	// - Function calls
	// - Return statements
}
