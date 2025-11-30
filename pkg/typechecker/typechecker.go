package typechecker

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"strings"

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

// Scope represents a lexical scope for variable tracking
type Scope struct {
	parent    *Scope
	variables map[string]string // variable name -> type name
}

// NewScope creates a new scope with an optional parent
func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:    parent,
		variables: make(map[string]string),
	}
}

// Define adds a variable to the current scope
func (s *Scope) Define(name, typeName string) {
	s.variables[name] = typeName
}

// Lookup finds a variable in the current scope or any parent scope
func (s *Scope) Lookup(name string) (string, bool) {
	if typeName, ok := s.variables[name]; ok {
		return typeName, true
	}
	if s.parent != nil {
		return s.parent.Lookup(name)
	}
	return "", false
}

// Type represents a type in the EZ type system
type Type struct {
	Name        string
	Kind        TypeKind
	ElementType *Type            // For arrays
	Fields      map[string]*Type // For structs
	Size        int              // For fixed-size arrays, -1 for dynamic
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
	types        map[string]*Type              // All known types
	functions    map[string]*FunctionSignature // All function signatures
	variables    map[string]string             // Variable name -> type name (global scope)
	modules      map[string]bool               // Imported module names
	currentScope *Scope                        // Current scope for local variable tracking
	errors       *errors.EZErrorList
	source       string
	filename     string
}

// NewTypeChecker creates a new type checker
func NewTypeChecker(source, filename string) *TypeChecker {
	tc := &TypeChecker{
		types:     make(map[string]*Type),
		functions: make(map[string]*FunctionSignature),
		variables: make(map[string]string),
		modules:   make(map[string]bool),
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
		"i8", "i16", "i32", "i64", "i128", "i256", "int",
		// Unsigned integers
		"u8", "u16", "u32", "u64", "u128", "u256", "uint",
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

// addWarning adds a type warning
func (tc *TypeChecker) addWarning(code errors.ErrorCode, message string, line, column int) {
	sourceLine := ""
	if tc.source != "" {
		sourceLine = errors.GetSourceLine(tc.source, line)
	}

	warn := errors.NewErrorWithSource(
		code,
		message,
		tc.filename,
		line,
		column,
		sourceLine,
	)
	tc.errors.AddWarning(warn)
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

	// Phase 4: Validate that a main() function exists
	tc.checkMainFunction()

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
				errors.E3002,
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
				errors.E3002,
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
				errors.E3002,
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
				errors.E3002,
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
	// Create a new scope for this function
	tc.enterScope()
	defer tc.exitScope()

	// Add function parameters to scope
	for _, param := range node.Parameters {
		tc.defineVariable(param.Name.Value, param.TypeName)
	}

	// Check if function body contains at least one return statement (for functions with return types)
	if len(node.ReturnTypes) > 0 {
		// Check if W2003 warning is suppressed
		if !tc.isSuppressed("W2003", node.Attributes) {
			if !tc.hasReturnStatement(node.Body) {
				tc.addWarning(
					errors.W2003,
					fmt.Sprintf("Function '%s' declares return type(s) but has no return statement", node.Name.Value),
					node.Name.Token.Line,
					node.Name.Token.Column,
				)
			}
		}
	}

	// Type check the function body
	tc.checkBlock(node.Body, node.ReturnTypes)
}

// checkMainFunction validates that a main() function exists as the program entry point
func (tc *TypeChecker) checkMainFunction() {
	if _, exists := tc.functions["main"]; !exists {
		tc.addError(
			errors.E3007,
			"Program must define a main() function",
			1,
			1,
		)
	}
}

// isSuppressed checks if a warning code is suppressed by function attributes
func (tc *TypeChecker) isSuppressed(warningCode string, attrs []*ast.Attribute) bool {
	if attrs == nil {
		return false
	}

	for _, attr := range attrs {
		if attr.Name == "suppress" {
			for _, arg := range attr.Args {
				if arg == warningCode || arg == "missing_return" {
					return true
				}
			}
		}
	}
	return false
}

// hasReturnStatement recursively checks if a block contains a return statement
func (tc *TypeChecker) hasReturnStatement(block *ast.BlockStatement) bool {
	if block == nil {
		return false
	}

	for _, stmt := range block.Statements {
		// Check if this statement is a return statement
		if _, ok := stmt.(*ast.ReturnStatement); ok {
			return true
		}

		// Check nested blocks in control flow statements
		switch s := stmt.(type) {
		case *ast.IfStatement:
			if tc.hasReturnInIfStatement(s) {
				return true
			}

		case *ast.ForStatement:
			if tc.hasReturnStatement(s.Body) {
				return true
			}

		case *ast.ForEachStatement:
			if tc.hasReturnStatement(s.Body) {
				return true
			}

		case *ast.WhileStatement:
			if tc.hasReturnStatement(s.Body) {
				return true
			}
		}
	}

	return false
}

// hasReturnInIfStatement recursively checks if/or/otherwise chains for return statements
func (tc *TypeChecker) hasReturnInIfStatement(ifStmt *ast.IfStatement) bool {
	// Check the consequence block
	if tc.hasReturnStatement(ifStmt.Consequence) {
		return true
	}

	// Check the alternative (can be another IfStatement or BlockStatement)
	if ifStmt.Alternative != nil {
		if altIf, ok := ifStmt.Alternative.(*ast.IfStatement); ok {
			// Recursively check the next if in the chain
			return tc.hasReturnInIfStatement(altIf)
		} else if altBlock, ok := ifStmt.Alternative.(*ast.BlockStatement); ok {
			// Check the otherwise block
			return tc.hasReturnStatement(altBlock)
		}
	}

	return false
}

// ============================================================================
// Phase 3, 4, 5: Statement Type Checking
// ============================================================================

// checkBlock validates all statements in a block
func (tc *TypeChecker) checkBlock(block *ast.BlockStatement, expectedReturnTypes []string) {
	if block == nil {
		return
	}

	for _, stmt := range block.Statements {
		tc.checkStatement(stmt, expectedReturnTypes)
	}
}

// checkStatement validates a single statement
func (tc *TypeChecker) checkStatement(stmt ast.Statement, expectedReturnTypes []string) {
	switch s := stmt.(type) {
	case *ast.VariableDeclaration:
		tc.checkVariableDeclaration(s)

	case *ast.AssignmentStatement:
		tc.checkAssignment(s)

	case *ast.ReturnStatement:
		tc.checkReturnStatement(s, expectedReturnTypes)

	case *ast.ExpressionStatement:
		tc.checkExpressionStatement(s)

	case *ast.IfStatement:
		tc.checkIfStatement(s, expectedReturnTypes)

	case *ast.ForStatement:
		tc.checkForStatement(s, expectedReturnTypes)

	case *ast.ForEachStatement:
		tc.checkForEachStatement(s, expectedReturnTypes)

	case *ast.WhileStatement:
		tc.checkWhileStatement(s, expectedReturnTypes)

	case *ast.LoopStatement:
		tc.checkLoopStatement(s, expectedReturnTypes)

	case *ast.BlockStatement:
		tc.enterScope()
		tc.checkBlock(s, expectedReturnTypes)
		tc.exitScope()
	}
}

// checkVariableDeclaration validates a variable declaration (Phase 3)
func (tc *TypeChecker) checkVariableDeclaration(decl *ast.VariableDeclaration) {
	// Handle multiple names (for multi-return assignment)
	if len(decl.Names) > 1 {
		// Multi-return assignment like: temp result, err = divide(10, 2)
		// For now, we can't easily infer types for multi-return
		// The runtime will handle this
		return
	}

	// Single variable declaration
	if decl.Name == nil {
		return
	}

	varName := decl.Name.Value
	declaredType := decl.TypeName

	// Check if declared type exists
	if declaredType != "" && !tc.TypeExists(declaredType) {
		tc.addError(
			errors.E3008,
			fmt.Sprintf("undefined type '%s'", declaredType),
			decl.Name.Token.Line,
			decl.Name.Token.Column,
		)
		return
	}

	// If there's an initial value, check type compatibility
	if decl.Value != nil {
		// Validate the expression itself
		tc.checkExpression(decl.Value)

		if declaredType == "" {
			return // No type to check against
		}

		actualType, ok := tc.inferExpressionType(decl.Value)
		if ok {
			// Check if it's an array type mismatch (assigning scalar to array)
			if tc.isArrayType(declaredType) && !tc.isArrayType(actualType) && actualType != "nil" {
				tc.addError(
					errors.E3018,
					fmt.Sprintf("cannot assign %s to array type %s - array type requires value in {} format", actualType, declaredType),
					decl.Name.Token.Line,
					decl.Name.Token.Column,
				)
				return
			}

			// Check for type mismatch
			if !tc.typesCompatible(declaredType, actualType) {
				tc.addError(
					errors.E3001,
					fmt.Sprintf("type mismatch: cannot assign %s to %s", actualType, declaredType),
					decl.Name.Token.Line,
					decl.Name.Token.Column,
				)
				return
			}
		}
	}

	// Register variable in current scope
	if declaredType != "" {
		tc.defineVariable(varName, declaredType)
	}
}

// checkAssignment validates an assignment statement (Phase 3)
func (tc *TypeChecker) checkAssignment(assign *ast.AssignmentStatement) {
	// Also validate the value expression
	tc.checkExpression(assign.Value)

	// Get the target type
	targetType, targetOk := tc.inferExpressionType(assign.Name)
	if !targetOk {
		// Try to get more specific type info for member expressions
		if member, ok := assign.Name.(*ast.MemberExpression); ok {
			tc.checkMemberAssignment(member, assign.Value)
		}
		return
	}

	// Get the value type
	valueType, valueOk := tc.inferExpressionType(assign.Value)
	if !valueOk {
		return // Can't determine value type
	}

	// Check compatibility
	if !tc.typesCompatible(targetType, valueType) {
		line, column := tc.getExpressionPosition(assign.Name)
		tc.addError(
			errors.E3001,
			fmt.Sprintf("type mismatch: cannot assign %s to %s", valueType, targetType),
			line,
			column,
		)
	}

	// For index expressions, also validate the index
	if indexExpr, ok := assign.Name.(*ast.IndexExpression); ok {
		tc.checkIndexExpression(indexExpr)
	}
}

// checkMemberAssignment validates struct field assignments
func (tc *TypeChecker) checkMemberAssignment(member *ast.MemberExpression, value ast.Expression) {
	// Get the object type
	objType, ok := tc.inferExpressionType(member.Object)
	if !ok {
		return
	}

	// Look up struct type
	structType, exists := tc.types[objType]
	if !exists || structType.Kind != StructType {
		return // Not a struct, can't check field types
	}

	// Get the field type
	fieldType, hasField := structType.Fields[member.Member.Value]
	if !hasField {
		line, column := tc.getExpressionPosition(member.Member)
		tc.addError(
			errors.E4003,
			fmt.Sprintf("struct '%s' has no field '%s'", objType, member.Member.Value),
			line,
			column,
		)
		return
	}

	// Get the value type
	valueType, ok := tc.inferExpressionType(value)
	if !ok {
		return
	}

	// Check compatibility
	if !tc.typesCompatible(fieldType.Name, valueType) {
		line, column := tc.getExpressionPosition(member.Member)
		tc.addError(
			errors.E3001,
			fmt.Sprintf("type mismatch: cannot assign %s to field '%s' of type %s",
				valueType, member.Member.Value, fieldType.Name),
			line,
			column,
		)
	}
}

// checkReturnStatement validates a return statement (Phase 4)
func (tc *TypeChecker) checkReturnStatement(ret *ast.ReturnStatement, expectedTypes []string) {
	// Validate all return value expressions
	for _, val := range ret.Values {
		tc.checkExpression(val)
	}

	// No return type expected
	if len(expectedTypes) == 0 {
		if len(ret.Values) > 0 {
			tc.addError(
				errors.E3012,
				"unexpected return value in void function",
				ret.Token.Line,
				ret.Token.Column,
			)
		}
		return
	}

	// Check return value count
	if len(ret.Values) != len(expectedTypes) {
		tc.addError(
			errors.E3013,
			fmt.Sprintf("wrong number of return values: expected %d, got %d", len(expectedTypes), len(ret.Values)),
			ret.Token.Line,
			ret.Token.Column,
		)
		return
	}

	// Check each return value type
	for i, val := range ret.Values {
		actualType, ok := tc.inferExpressionType(val)
		if !ok {
			continue // Can't determine type
		}

		expectedType := expectedTypes[i]
		if !tc.typesCompatible(expectedType, actualType) {
			tc.addError(
				errors.E3012,
				fmt.Sprintf("return type mismatch: expected %s, got %s", expectedType, actualType),
				ret.Token.Line,
				ret.Token.Column,
			)
		}
	}
}

// checkExpressionStatement validates an expression statement
func (tc *TypeChecker) checkExpressionStatement(exprStmt *ast.ExpressionStatement) {
	if exprStmt.Expression == nil {
		return
	}

	// Validate the entire expression tree
	tc.checkExpression(exprStmt.Expression)
}

// checkExpression recursively validates an expression and its sub-expressions
func (tc *TypeChecker) checkExpression(expr ast.Expression) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.CallExpression:
		tc.checkFunctionCall(e)
		// Also check arguments
		for _, arg := range e.Arguments {
			tc.checkExpression(arg)
		}

	case *ast.InfixExpression:
		tc.checkInfixExpression(e)
		tc.checkExpression(e.Left)
		tc.checkExpression(e.Right)

	case *ast.PrefixExpression:
		tc.checkPrefixExpression(e)
		tc.checkExpression(e.Right)

	case *ast.IndexExpression:
		tc.checkIndexExpression(e)
		tc.checkExpression(e.Left)
		tc.checkExpression(e.Index)

	case *ast.MemberExpression:
		tc.checkExpression(e.Object)

	case *ast.ArrayValue:
		for _, elem := range e.Elements {
			tc.checkExpression(elem)
		}
	}
}

// checkInfixExpression validates binary operator usage (Phase 6)
func (tc *TypeChecker) checkInfixExpression(infix *ast.InfixExpression) {
	leftType, leftOk := tc.inferExpressionType(infix.Left)
	rightType, rightOk := tc.inferExpressionType(infix.Right)

	if !leftOk || !rightOk {
		return // Can't determine types
	}

	line, column := tc.getExpressionPosition(infix.Left)

	switch infix.Operator {
	case "+":
		// Valid for numbers or strings
		if tc.isNumericType(leftType) && tc.isNumericType(rightType) {
			return // OK
		}
		if leftType == "string" && rightType == "string" {
			return // OK - string concatenation
		}
		tc.addError(
			errors.E3002,
			fmt.Sprintf("invalid operands for '+': %s and %s (expected numeric or string)", leftType, rightType),
			line,
			column,
		)

	case "-", "*", "/", "%":
		// Only valid for numbers
		if !tc.isNumericType(leftType) || !tc.isNumericType(rightType) {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("invalid operands for '%s': %s and %s (expected numeric)", infix.Operator, leftType, rightType),
				line,
				column,
			)
		}

	case "==", "!=":
		// Valid for any matching types
		if !tc.typesCompatible(leftType, rightType) && !tc.typesCompatible(rightType, leftType) {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("cannot compare %s with %s using '%s'", leftType, rightType, infix.Operator),
				line,
				column,
			)
		}

	case "<", ">", "<=", ">=":
		// Only valid for numbers or strings
		if !tc.isNumericType(leftType) && leftType != "string" {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("invalid operand for '%s': %s (expected numeric or string)", infix.Operator, leftType),
				line,
				column,
			)
		}
		if !tc.isNumericType(rightType) && rightType != "string" {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("invalid operand for '%s': %s (expected numeric or string)", infix.Operator, rightType),
				line,
				column,
			)
		}

	case "&&", "||":
		// Only valid for booleans
		if leftType != "bool" {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("invalid operand for '%s': %s (expected bool)", infix.Operator, leftType),
				line,
				column,
			)
		}
		if rightType != "bool" {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("invalid operand for '%s': %s (expected bool)", infix.Operator, rightType),
				line,
				column,
			)
		}

	case "in", "!in":
		// Right side must be array or string
		if !tc.isArrayType(rightType) && rightType != "string" {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("invalid operand for '%s': %s (expected array or string)", infix.Operator, rightType),
				line,
				column,
			)
		}
	}
}

// checkPrefixExpression validates unary operator usage
func (tc *TypeChecker) checkPrefixExpression(prefix *ast.PrefixExpression) {
	operandType, ok := tc.inferExpressionType(prefix.Right)
	if !ok {
		return
	}

	line, column := tc.getExpressionPosition(prefix.Right)

	switch prefix.Operator {
	case "!":
		if operandType != "bool" {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("invalid operand for '!': %s (expected bool)", operandType),
				line,
				column,
			)
		}

	case "-":
		if !tc.isNumericType(operandType) {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("invalid operand for unary '-': %s (expected numeric)", operandType),
				line,
				column,
			)
		}
	}
}

// checkIndexExpression validates array/string indexing
func (tc *TypeChecker) checkIndexExpression(index *ast.IndexExpression) {
	// Check that the index is an integer
	indexType, ok := tc.inferExpressionType(index.Index)
	if ok && !tc.isIntegerType(indexType) {
		line, column := tc.getExpressionPosition(index.Index)
		tc.addError(
			errors.E3003,
			fmt.Sprintf("index must be an integer, got %s", indexType),
			line,
			column,
		)
	}

	// Check that the left side is indexable
	leftType, ok := tc.inferExpressionType(index.Left)
	if ok && !tc.isArrayType(leftType) && leftType != "string" {
		line, column := tc.getExpressionPosition(index.Left)
		tc.addError(
			errors.E3016,
			fmt.Sprintf("cannot index into %s (expected array or string)", leftType),
			line,
			column,
		)
	}
}

// checkFunctionCall validates function call argument types (Phase 5)
func (tc *TypeChecker) checkFunctionCall(call *ast.CallExpression) {
	// Get function name
	var funcName string
	switch fn := call.Function.(type) {
	case *ast.Label:
		funcName = fn.Value
	case *ast.MemberExpression:
		// Module function call - let runtime handle for now
		return
	default:
		return
	}

	// Check builtin type conversion functions
	if tc.checkBuiltinTypeConversion(funcName, call) {
		return // Handled as builtin
	}

	// Look up function signature
	sig, ok := tc.functions[funcName]
	if !ok {
		// Unknown function - let runtime handle
		return
	}

	// Check argument count
	if len(call.Arguments) != len(sig.Parameters) {
		line, column := tc.getExpressionPosition(call.Function)
		tc.addError(
			errors.E5008,
			fmt.Sprintf("wrong number of arguments to '%s': expected %d, got %d",
				funcName, len(sig.Parameters), len(call.Arguments)),
			line,
			column,
		)
		return
	}

	// Check argument types
	for i, arg := range call.Arguments {
		actualType, ok := tc.inferExpressionType(arg)
		if !ok {
			continue
		}

		expectedType := sig.Parameters[i].Type
		if !tc.typesCompatible(expectedType, actualType) {
			line, column := tc.getExpressionPosition(arg)
			tc.addError(
				errors.E3001,
				fmt.Sprintf("argument type mismatch in call to '%s': parameter '%s' expects %s, got %s",
					funcName, sig.Parameters[i].Name, expectedType, actualType),
				line,
				column,
			)
		}
	}
}

// checkBuiltinTypeConversion validates builtin type conversion functions
// Returns true if this was a builtin conversion, false otherwise
func (tc *TypeChecker) checkBuiltinTypeConversion(funcName string, call *ast.CallExpression) bool {
	switch funcName {
	case "int":
		if len(call.Arguments) != 1 {
			return false // Let runtime handle arg count
		}
		argType, ok := tc.inferExpressionType(call.Arguments[0])
		if !ok {
			return true
		}
		// int() accepts: numeric types, bool, but NOT string variables
		if argType == "string" {
			// Check if it's a string literal that looks numeric
			if strVal, isLiteral := call.Arguments[0].(*ast.StringValue); isLiteral {
				if tc.isNumericString(strVal.Value) {
					return true // OK - numeric string literal
				}
			}
			line, column := tc.getExpressionPosition(call.Arguments[0])
			tc.addError(
				errors.E3005,
				fmt.Sprintf("cannot convert string to int at build-time (value may not be numeric)"),
				line,
				column,
			)
		}
		return true

	case "float":
		if len(call.Arguments) != 1 {
			return false
		}
		argType, ok := tc.inferExpressionType(call.Arguments[0])
		if !ok {
			return true
		}
		// float() accepts: numeric types, bool, but NOT string variables
		if argType == "string" {
			// Check if it's a string literal that looks numeric
			if strVal, isLiteral := call.Arguments[0].(*ast.StringValue); isLiteral {
				if tc.isNumericString(strVal.Value) {
					return true // OK - numeric string literal
				}
			}
			line, column := tc.getExpressionPosition(call.Arguments[0])
			tc.addError(
				errors.E3006,
				fmt.Sprintf("cannot convert string to float at build-time (value may not be numeric)"),
				line,
				column,
			)
		}
		return true

	case "string", "bool", "char":
		// These conversions are generally safe
		return true

	case "len", "typeof":
		// Other builtins - let them through
		return true

	default:
		return false // Not a builtin we handle
	}
}

// isNumericString checks if a string looks like a valid number
func (tc *TypeChecker) isNumericString(s string) bool {
	if len(s) == 0 {
		return false
	}
	hasDigit := false
	hasDot := false
	for i, ch := range s {
		if ch == '-' || ch == '+' {
			if i != 0 {
				return false
			}
			continue
		}
		if ch == '.' {
			if hasDot {
				return false
			}
			hasDot = true
			continue
		}
		if ch >= '0' && ch <= '9' {
			hasDigit = true
			continue
		}
		// Allow underscore separators
		if ch == '_' {
			continue
		}
		return false
	}
	return hasDigit
}

// checkIfStatement validates an if statement
func (tc *TypeChecker) checkIfStatement(ifStmt *ast.IfStatement, expectedReturnTypes []string) {
	// Check that condition is boolean
	condType, ok := tc.inferExpressionType(ifStmt.Condition)
	if ok && condType != "bool" {
		line, column := tc.getExpressionPosition(ifStmt.Condition)
		tc.addError(
			errors.E3001,
			fmt.Sprintf("if condition must be bool, got %s", condType),
			line,
			column,
		)
	}

	// Check consequence block
	tc.enterScope()
	tc.checkBlock(ifStmt.Consequence, expectedReturnTypes)
	tc.exitScope()

	// Check alternative (else/or/otherwise)
	if ifStmt.Alternative != nil {
		switch alt := ifStmt.Alternative.(type) {
		case *ast.IfStatement:
			tc.checkIfStatement(alt, expectedReturnTypes)
		case *ast.BlockStatement:
			tc.enterScope()
			tc.checkBlock(alt, expectedReturnTypes)
			tc.exitScope()
		}
	}
}

// checkForStatement validates a for loop
func (tc *TypeChecker) checkForStatement(forStmt *ast.ForStatement, expectedReturnTypes []string) {
	tc.enterScope()

	// Add loop variable to scope
	if forStmt.Variable != nil {
		varType := forStmt.VarType
		if varType == "" {
			varType = "int" // Default for range iteration
		}
		tc.defineVariable(forStmt.Variable.Value, varType)
	}

	tc.checkBlock(forStmt.Body, expectedReturnTypes)
	tc.exitScope()
}

// checkForEachStatement validates a for_each loop
func (tc *TypeChecker) checkForEachStatement(forEach *ast.ForEachStatement, expectedReturnTypes []string) {
	tc.enterScope()

	// Infer element type from collection
	if forEach.Variable != nil && forEach.Collection != nil {
		collType, ok := tc.inferExpressionType(forEach.Collection)
		if ok {
			// For arrays, element type is inside []
			if len(collType) > 2 && collType[0] == '[' {
				elemType := collType[1 : len(collType)-1]
				tc.defineVariable(forEach.Variable.Value, elemType)
			} else if collType == "string" {
				// Iterating over string gives char
				tc.defineVariable(forEach.Variable.Value, "char")
			}
		}
	}

	tc.checkBlock(forEach.Body, expectedReturnTypes)
	tc.exitScope()
}

// checkWhileStatement validates an as_long_as loop
func (tc *TypeChecker) checkWhileStatement(whileStmt *ast.WhileStatement, expectedReturnTypes []string) {
	// Check that condition is boolean
	condType, ok := tc.inferExpressionType(whileStmt.Condition)
	if ok && condType != "bool" {
		line, column := tc.getExpressionPosition(whileStmt.Condition)
		tc.addError(
			errors.E3001,
			fmt.Sprintf("while condition must be bool, got %s", condType),
			line,
			column,
		)
	}

	tc.enterScope()
	tc.checkBlock(whileStmt.Body, expectedReturnTypes)
	tc.exitScope()
}

// checkLoopStatement validates a loop statement
func (tc *TypeChecker) checkLoopStatement(loopStmt *ast.LoopStatement, expectedReturnTypes []string) {
	tc.enterScope()
	tc.checkBlock(loopStmt.Body, expectedReturnTypes)
	tc.exitScope()
}

// isArrayType checks if a type string represents an array type
func (tc *TypeChecker) isArrayType(typeName string) bool {
	return len(typeName) >= 2 && typeName[0] == '[' && typeName[len(typeName)-1] == ']'
}

// extractArrayElementType extracts the element type from an array type string
// Handles both [int] and [int, 3] (fixed-size) formats
func (tc *TypeChecker) extractArrayElementType(arrType string) string {
	if len(arrType) < 3 || arrType[0] != '[' {
		return arrType
	}

	// Remove the outer brackets
	inner := arrType[1 : len(arrType)-1]

	// Check for comma (fixed-size array like "int, 3")
	for i, ch := range inner {
		if ch == ',' {
			return strings.TrimSpace(inner[:i])
		}
	}

	return inner
}

// getExpressionPosition returns the line and column of an expression
func (tc *TypeChecker) getExpressionPosition(expr ast.Expression) (int, int) {
	switch e := expr.(type) {
	case *ast.Label:
		return e.Token.Line, e.Token.Column
	case *ast.IntegerValue:
		return e.Token.Line, e.Token.Column
	case *ast.FloatValue:
		return e.Token.Line, e.Token.Column
	case *ast.StringValue:
		return e.Token.Line, e.Token.Column
	case *ast.BooleanValue:
		return e.Token.Line, e.Token.Column
	case *ast.CharValue:
		return e.Token.Line, e.Token.Column
	case *ast.ArrayValue:
		return e.Token.Line, e.Token.Column
	case *ast.CallExpression:
		return e.Token.Line, e.Token.Column
	case *ast.InfixExpression:
		return e.Token.Line, e.Token.Column
	case *ast.PrefixExpression:
		return e.Token.Line, e.Token.Column
	case *ast.IndexExpression:
		return e.Token.Line, e.Token.Column
	case *ast.MemberExpression:
		return e.Token.Line, e.Token.Column
	default:
		return 1, 1
	}
}

// ============================================================================
// Phase 1 & 2: Expression Type Inference with Scope Tracking
// ============================================================================

// enterScope creates a new child scope and makes it current
func (tc *TypeChecker) enterScope() {
	tc.currentScope = NewScope(tc.currentScope)
}

// exitScope returns to the parent scope
func (tc *TypeChecker) exitScope() {
	if tc.currentScope != nil && tc.currentScope.parent != nil {
		tc.currentScope = tc.currentScope.parent
	}
}

// defineVariable adds a variable to the current scope
func (tc *TypeChecker) defineVariable(name, typeName string) {
	if tc.currentScope != nil {
		tc.currentScope.Define(name, typeName)
	}
}

// lookupVariable finds a variable type in scope chain or global variables
func (tc *TypeChecker) lookupVariable(name string) (string, bool) {
	// First check local scopes
	if tc.currentScope != nil {
		if typeName, ok := tc.currentScope.Lookup(name); ok {
			return typeName, true
		}
	}
	// Then check global variables
	if typeName, ok := tc.variables[name]; ok {
		return typeName, true
	}
	return "", false
}

// inferExpressionType determines the type of an expression at build-time
// Returns the type name and whether the type could be determined
func (tc *TypeChecker) inferExpressionType(expr ast.Expression) (string, bool) {
	if expr == nil {
		return "", false
	}

	switch e := expr.(type) {
	case *ast.IntegerValue:
		return "int", true

	case *ast.FloatValue:
		return "float", true

	case *ast.StringValue:
		return "string", true

	case *ast.CharValue:
		return "char", true

	case *ast.BooleanValue:
		return "bool", true

	case *ast.NilValue:
		return "nil", true

	case *ast.Label:
		// Variable lookup
		return tc.lookupVariable(e.Value)

	case *ast.ArrayValue:
		return tc.inferArrayType(e)

	case *ast.StructValue:
		// Struct literal - type is the struct name
		if e.Name != nil {
			return e.Name.Value, true
		}
		return "", false

	case *ast.PrefixExpression:
		return tc.inferPrefixType(e)

	case *ast.InfixExpression:
		return tc.inferInfixType(e)

	case *ast.PostfixExpression:
		// Postfix operators (++ and --) return the operand's type
		return tc.inferExpressionType(e.Left)

	case *ast.CallExpression:
		return tc.inferCallType(e)

	case *ast.IndexExpression:
		return tc.inferIndexType(e)

	case *ast.MemberExpression:
		return tc.inferMemberType(e)

	case *ast.NewExpression:
		// new(Type) returns the type
		if e.TypeName != nil {
			return e.TypeName.Value, true
		}
		return "", false

	case *ast.RangeExpression:
		// range(start, end) returns [int]
		return "[int]", true

	case *ast.InterpolatedString:
		return "string", true

	case *ast.IgnoreValue:
		return "void", true

	default:
		return "", false
	}
}

// inferArrayType infers the type of an array literal
func (tc *TypeChecker) inferArrayType(arr *ast.ArrayValue) (string, bool) {
	if len(arr.Elements) == 0 {
		// Empty array - can't infer element type
		return "[]", true
	}

	// Infer type from first element
	firstType, ok := tc.inferExpressionType(arr.Elements[0])
	if !ok {
		return "", false
	}

	return fmt.Sprintf("[%s]", firstType), true
}

// inferPrefixType infers the type of a prefix expression
func (tc *TypeChecker) inferPrefixType(prefix *ast.PrefixExpression) (string, bool) {
	operandType, ok := tc.inferExpressionType(prefix.Right)
	if !ok {
		return "", false
	}

	switch prefix.Operator {
	case "!":
		// Logical NOT always returns bool
		return "bool", true
	case "-":
		// Unary minus returns the operand's numeric type
		if tc.isNumericType(operandType) {
			return operandType, true
		}
		return "", false
	default:
		return operandType, true
	}
}

// inferInfixType infers the type of an infix/binary expression
func (tc *TypeChecker) inferInfixType(infix *ast.InfixExpression) (string, bool) {
	leftType, leftOk := tc.inferExpressionType(infix.Left)
	rightType, rightOk := tc.inferExpressionType(infix.Right)

	if !leftOk || !rightOk {
		return "", false
	}

	switch infix.Operator {
	// Comparison operators always return bool
	case "==", "!=", "<", ">", "<=", ">=":
		return "bool", true

	// Logical operators always return bool
	case "&&", "||":
		return "bool", true

	// Membership operators return bool
	case "in", "!in":
		return "bool", true

	// Arithmetic operators
	case "+":
		// String concatenation
		if leftType == "string" && rightType == "string" {
			return "string", true
		}
		// Numeric addition
		if tc.isNumericType(leftType) && tc.isNumericType(rightType) {
			return tc.promoteNumericTypes(leftType, rightType), true
		}
		return "", false

	case "-", "*", "/", "%":
		// Numeric operations
		if tc.isNumericType(leftType) && tc.isNumericType(rightType) {
			return tc.promoteNumericTypes(leftType, rightType), true
		}
		return "", false

	default:
		return "", false
	}
}

// inferCallType infers the return type of a function call
func (tc *TypeChecker) inferCallType(call *ast.CallExpression) (string, bool) {
	switch fn := call.Function.(type) {
	case *ast.Label:
		// Direct function call like foo()
		if sig, ok := tc.functions[fn.Value]; ok {
			if len(sig.ReturnTypes) == 1 {
				return sig.ReturnTypes[0], true
			} else if len(sig.ReturnTypes) > 1 {
				// Multi-return - return first type for now
				// Full multi-return handling requires special treatment
				return sig.ReturnTypes[0], true
			}
			return "void", true
		}
		// Check built-in functions
		return tc.inferBuiltinCallType(fn.Value, call.Arguments)

	case *ast.MemberExpression:
		// Module function call like std.println()
		return tc.inferModuleCallType(fn, call.Arguments)

	default:
		return "", false
	}
}

// inferBuiltinCallType infers the return type of built-in functions
func (tc *TypeChecker) inferBuiltinCallType(name string, args []ast.Expression) (string, bool) {
	switch name {
	case "len":
		return "int", true
	case "typeof":
		return "string", true
	case "int":
		return "int", true
	case "float":
		return "float", true
	case "string":
		return "string", true
	case "bool":
		return "bool", true
	case "char":
		return "char", true
	default:
		return "", false
	}
}

// inferModuleCallType infers the return type of module function calls
func (tc *TypeChecker) inferModuleCallType(member *ast.MemberExpression, args []ast.Expression) (string, bool) {
	// Get module name
	moduleName := ""
	if label, ok := member.Object.(*ast.Label); ok {
		moduleName = label.Value
	} else {
		return "", false
	}

	funcName := member.Member.Value

	// Standard library function return types
	switch moduleName {
	case "std":
		return tc.inferStdCallType(funcName, args)
	case "math":
		return tc.inferMathCallType(funcName, args)
	case "arrays":
		return tc.inferArraysCallType(funcName, args)
	case "strings":
		return tc.inferStringsCallType(funcName, args)
	case "time":
		return tc.inferTimeCallType(funcName, args)
	default:
		return "", false
	}
}

// inferStdCallType infers return types for @std functions
func (tc *TypeChecker) inferStdCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "println", "print", "printf":
		return "void", true
	case "input":
		return "string", true
	default:
		return "", false
	}
}

// inferMathCallType infers return types for @math functions
func (tc *TypeChecker) inferMathCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "abs", "floor", "ceil", "round", "sqrt", "pow", "log", "log2", "log10",
		"sin", "cos", "tan", "asin", "acos", "atan", "exp", "min", "max", "avg",
		"random_float":
		return "float", true
	case "random", "factorial":
		return "int", true
	default:
		return "", false
	}
}

// inferArraysCallType infers return types for @arrays functions
func (tc *TypeChecker) inferArraysCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "len", "index_of", "last_index_of":
		return "int", true
	case "contains", "is_empty":
		return "bool", true
	case "join":
		return "string", true
	case "push", "unshift", "clear", "remove_at", "set":
		return "void", true
	case "pop", "shift", "get", "first", "last":
		// Returns element type - need array type to determine
		if len(args) > 0 {
			arrType, ok := tc.inferExpressionType(args[0])
			if ok && len(arrType) > 2 && arrType[0] == '[' {
				// Extract element type from [type]
				return arrType[1 : len(arrType)-1], true
			}
		}
		return "", false
	case "sum", "product", "min", "max":
		// Could be int or float depending on array type
		return "float", true
	case "avg":
		return "float", true
	case "reverse", "slice", "copy", "concat", "unique", "sorted", "filter", "map":
		// Returns array of same/similar type
		if len(args) > 0 {
			return tc.inferExpressionType(args[0])
		}
		return "", false
	case "repeat", "range":
		return "[int]", true
	case "zip":
		return "[[]]", true // Array of arrays
	default:
		return "", false
	}
}

// inferStringsCallType infers return types for @strings functions
func (tc *TypeChecker) inferStringsCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "len", "index", "last_index", "count":
		return "int", true
	case "contains", "starts_with", "ends_with", "is_empty":
		return "bool", true
	case "upper", "lower", "trim", "trim_left", "trim_right", "reverse",
		"replace", "substring", "repeat", "pad_left", "pad_right", "join":
		return "string", true
	case "split", "chars":
		return "[string]", true
	default:
		return "", false
	}
}

// inferTimeCallType infers return types for @time functions
func (tc *TypeChecker) inferTimeCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "now", "now_ms", "tick", "make", "add_seconds", "add_minutes",
		"add_hours", "add_days", "add_months", "add_years", "diff":
		return "int", true
	case "format", "format_date", "format_time":
		return "string", true
	case "year", "month", "day", "hour", "minute", "second", "weekday",
		"day_of_year", "days_in_month":
		return "int", true
	case "is_leap_year":
		return "bool", true
	case "sleep", "sleep_ms":
		return "void", true
	case "elapsed_ms":
		return "int", true
	default:
		return "", false
	}
}

// inferIndexType infers the type when indexing into an array or string
func (tc *TypeChecker) inferIndexType(index *ast.IndexExpression) (string, bool) {
	leftType, ok := tc.inferExpressionType(index.Left)
	if !ok {
		return "", false
	}

	// Indexing into a string returns char
	if leftType == "string" {
		return "char", true
	}

	// Indexing into an array returns element type
	if len(leftType) > 2 && leftType[0] == '[' {
		// Extract element type from [type]
		return leftType[1 : len(leftType)-1], true
	}

	return "", false
}

// inferMemberType infers the type of a member access expression
func (tc *TypeChecker) inferMemberType(member *ast.MemberExpression) (string, bool) {
	// Check if accessing struct field
	objType, ok := tc.inferExpressionType(member.Object)
	if !ok {
		return "", false
	}

	// Look up struct type
	if structType, exists := tc.types[objType]; exists && structType.Kind == StructType {
		if fieldType, hasField := structType.Fields[member.Member.Value]; hasField {
			return fieldType.Name, true
		}
	}

	// Could be module access - return unknown for now
	// Module function calls are handled in inferCallType
	return "", false
}

// isNumericType checks if a type is numeric
func (tc *TypeChecker) isNumericType(typeName string) bool {
	switch typeName {
	case "int", "i8", "i16", "i32", "i64", "i128", "i256",
		"uint", "u8", "u16", "u32", "u64", "u128", "u256",
		"float", "f32", "f64":
		return true
	default:
		return false
	}
}

// isIntegerType checks if a type is an integer type
func (tc *TypeChecker) isIntegerType(typeName string) bool {
	switch typeName {
	case "int", "i8", "i16", "i32", "i64", "i128", "i256",
		"uint", "u8", "u16", "u32", "u64", "u128", "u256":
		return true
	default:
		return false
	}
}

// isSignedIntegerType checks if a type is a signed integer
func (tc *TypeChecker) isSignedIntegerType(typeName string) bool {
	switch typeName {
	case "int", "i8", "i16", "i32", "i64", "i128", "i256":
		return true
	default:
		return false
	}
}

// isUnsignedIntegerType checks if a type is an unsigned integer
func (tc *TypeChecker) isUnsignedIntegerType(typeName string) bool {
	switch typeName {
	case "uint", "u8", "u16", "u32", "u64", "u128", "u256":
		return true
	default:
		return false
	}
}

// promoteNumericTypes returns the "wider" type for mixed numeric operations
func (tc *TypeChecker) promoteNumericTypes(left, right string) string {
	// Float always wins
	if left == "float" || left == "f32" || left == "f64" ||
		right == "float" || right == "f32" || right == "f64" {
		return "float"
	}
	// Otherwise return the left type (could be more sophisticated)
	return left
}

// typesCompatible checks if two types are compatible for assignment
func (tc *TypeChecker) typesCompatible(declared, actual string) bool {
	// Exact match
	if declared == actual {
		return true
	}

	// nil is compatible with any reference type
	if actual == "nil" {
		return true
	}

	// Handle array type compatibility
	if len(declared) > 2 && declared[0] == '[' {
		// Empty array [] is compatible with any array type
		if actual == "[]" {
			return true
		}

		if len(actual) > 2 && actual[0] == '[' {
			// Extract element types, handling fixed-size arrays like [int, 3]
			declaredElem := tc.extractArrayElementType(declared)
			actualElem := tc.extractArrayElementType(actual)
			return tc.typesCompatible(declaredElem, actualElem)
		}
	}

	// Integer family compatibility rules
	// Signed integers can be assigned to other signed integers (with potential truncation)
	if tc.isSignedIntegerType(declared) && tc.isSignedIntegerType(actual) {
		return true
	}

	// Unsigned integers can be assigned to other unsigned integers
	if tc.isUnsignedIntegerType(declared) && tc.isUnsignedIntegerType(actual) {
		return true
	}

	// Unsigned to signed is safe - the value will always fit
	if tc.isSignedIntegerType(declared) && tc.isUnsignedIntegerType(actual) {
		return true
	}

	// Float family compatibility
	if (declared == "float" || declared == "f32" || declared == "f64") &&
		(actual == "float" || actual == "f32" || actual == "f64") {
		return true
	}

	// Integer literals (actual == "int") can be assigned to unsigned if value is non-negative
	// This is handled at runtime since we can't know the value at build-time in all cases
	// For now, we allow int to unsigned assignment and let runtime catch negative values
	if tc.isUnsignedIntegerType(declared) && tc.isSignedIntegerType(actual) {
		// We'll be permissive here - the runtime already catches negative values
		return true
	}

	return false
}
