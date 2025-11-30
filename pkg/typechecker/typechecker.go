package typechecker

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/errors"
)

// TypeKind represents the category of a type
type TypeKind int

const (
	PrimitiveType TypeKind = iota
	ArrayType
	MapType
	StructType
	EnumType
	FunctionType
	VoidType
)

// Scope represents a lexical scope for variable tracking
type Scope struct {
	parent       *Scope
	variables    map[string]string // variable name -> type name
	usingModules map[string]bool   // modules imported via 'using'
}

// NewScope creates a new scope with an optional parent
func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:       parent,
		variables:    make(map[string]string),
		usingModules: make(map[string]bool),
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

// AddUsingModule adds a module to the current scope's using list
func (s *Scope) AddUsingModule(moduleName string) {
	s.usingModules[moduleName] = true
}

// HasUsingModule checks if a module is in scope via 'using'
func (s *Scope) HasUsingModule(moduleName string) bool {
	if s.usingModules[moduleName] {
		return true
	}
	if s.parent != nil {
		return s.parent.HasUsingModule(moduleName)
	}
	return false
}

// GetAllUsingModules returns all modules imported via 'using' in the current scope and parent scopes
func (s *Scope) GetAllUsingModules() []string {
	result := make([]string, 0)
	seen := make(map[string]bool)

	// Collect from current scope
	for moduleName := range s.usingModules {
		if !seen[moduleName] {
			result = append(result, moduleName)
			seen[moduleName] = true
		}
	}

	// Collect from parent scopes
	if s.parent != nil {
		for _, moduleName := range s.parent.GetAllUsingModules() {
			if !seen[moduleName] {
				result = append(result, moduleName)
				seen[moduleName] = true
			}
		}
	}

	return result
}

// Type represents a type in the EZ type system
type Type struct {
	Name        string
	Kind        TypeKind
	ElementType *Type            // For arrays
	KeyType     *Type            // For maps
	ValueType   *Type            // For maps
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
	types            map[string]*Type              // All known types
	functions        map[string]*FunctionSignature // All function signatures
	variables        map[string]string             // Variable name -> type name (global scope)
	modules          map[string]bool               // Imported module names
	fileUsingModules map[string]bool               // File-level using modules
	currentScope     *Scope                        // Current scope for local variable tracking
	errors           *errors.EZErrorList
	source           string
	filename         string
	skipMainCheck    bool // Skip main() function requirement (for module files)
}

// NewTypeChecker creates a new type checker
func NewTypeChecker(source, filename string) *TypeChecker {
	tc := &TypeChecker{
		types:            make(map[string]*Type),
		functions:        make(map[string]*FunctionSignature),
		variables:        make(map[string]string),
		modules:          make(map[string]bool),
		fileUsingModules: make(map[string]bool),
		errors:           errors.NewErrorList(),
		source:           source,
		filename:         filename,
	}

	// Register built-in primitive types
	tc.registerBuiltinTypes()

	return tc
}

// SetSkipMainCheck sets whether to skip the main() function requirement
// Use this for module files that don't need a main() function
func (tc *TypeChecker) SetSkipMainCheck(skip bool) {
	tc.skipMainCheck = skip
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

	// Check for map types: map[keyType:valueType]
	if strings.HasPrefix(typeName, "map[") && strings.HasSuffix(typeName, "]") {
		// Validate the map type has proper format
		inner := typeName[4 : len(typeName)-1] // Extract keyType:valueType
		parts := strings.Split(inner, ":")
		if len(parts) == 2 {
			keyType := parts[0]
			valueType := parts[1]
			// Both key and value types must exist
			return tc.TypeExists(keyType) && tc.TypeExists(valueType)
		}
		return false
	}

	// Check for qualified type names (module.TypeName)
	// These are validated at runtime when the module is loaded
	if strings.Contains(typeName, ".") {
		parts := strings.SplitN(typeName, ".", 2)
		if len(parts) == 2 {
			moduleName := parts[0]
			// Check if the module has been imported
			if tc.modules[moduleName] {
				return true
			}
		}
	}

	// Check local types first
	if _, exists := tc.types[typeName]; exists {
		return true
	}

	// Check if the type might be available via file-level 'using' directive
	// For unqualified type names, if a module is imported via 'using',
	// the type will be validated at runtime when the module is loaded
	for moduleName := range tc.fileUsingModules {
		// If the module has been imported and is in file-level 'using', the type is considered valid
		// Actual type existence is validated at runtime
		if tc.modules[moduleName] {
			return true
		}
	}

	// Also check scope-level 'using' modules
	if tc.currentScope != nil {
		for _, moduleName := range tc.currentScope.GetAllUsingModules() {
			// If the module has been imported and is in 'using', the type is considered valid
			// Actual type existence is validated at runtime
			if tc.modules[moduleName] {
				return true
			}
		}
	}

	return false
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
	// Phase 0: Register all imported modules
	for _, stmt := range program.Statements {
		if importStmt, ok := stmt.(*ast.ImportStatement); ok {
			for _, item := range importStmt.Imports {
				// Register the module (use alias if provided, otherwise module name)
				moduleName := item.Alias
				if moduleName == "" {
					moduleName = item.Module
				}
				tc.modules[moduleName] = true
			}
		}
	}

	// Phase 0.5: Register file-level using modules
	for _, usingStmt := range program.FileUsing {
		for _, mod := range usingStmt.Modules {
			tc.fileUsingModules[mod.Value] = true
		}
	}

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

	// Phase 4: Validate that a main() function exists (unless skipped for module files)
	if !tc.skipMainCheck {
		tc.checkMainFunction()
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

	// Map warning codes to their alternate names
	alternateNames := map[string]string{
		"W2003": "missing_return",
		"W3003": "array_size_mismatch",
	}

	for _, attr := range attrs {
		if attr.Name == "suppress" {
			for _, arg := range attr.Args {
				if arg == warningCode {
					return true
				}
				// Check if the alternate name matches
				if altName, ok := alternateNames[warningCode]; ok && arg == altName {
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

	case *ast.UsingStatement:
		// Track which modules are available via 'using'
		for _, mod := range s.Modules {
			if tc.currentScope != nil {
				tc.currentScope.AddUsingModule(mod.Value)
			}
		}
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

			// Check for fixed-size array size mismatch (W3003)
			if tc.isArrayType(declaredType) {
				declaredSize := tc.extractArraySize(declaredType)
				if declaredSize > 0 {
					// Check if the value is an array literal
					if arrLit, ok := decl.Value.(*ast.ArrayValue); ok {
						actualSize := len(arrLit.Elements)
						if actualSize < declaredSize {
							if !tc.isSuppressed("W3003", decl.Attributes) {
								tc.addWarning(
									errors.W3003,
									fmt.Sprintf("fixed-size array not fully initialized: declared size %d but only %d element(s) provided",
										declaredSize, actualSize),
									decl.Name.Token.Line,
									decl.Name.Token.Column,
								)
							}
						}
					}
				}
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

	case *ast.MapValue:
		for _, pair := range e.Pairs {
			tc.checkExpression(pair.Key)
			tc.checkExpression(pair.Value)
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

// checkIndexExpression validates array/string/map indexing
func (tc *TypeChecker) checkIndexExpression(index *ast.IndexExpression) {
	// Get the left side type to determine what kind of index is valid
	leftType, leftOk := tc.inferExpressionType(index.Left)
	indexType, indexOk := tc.inferExpressionType(index.Index)

	// Check if left side is a map type
	if leftOk && tc.isMapType(leftType) {
		// Maps can be indexed with string, int, bool, or char (hashable types)
		if indexOk && !tc.isHashableType(indexType) {
			line, column := tc.getExpressionPosition(index.Index)
			tc.addError(
				errors.E3003,
				fmt.Sprintf("map key must be a hashable type (string, int, bool, char), got %s", indexType),
				line,
				column,
			)
		}
		return
	}

	// For arrays and strings, index must be an integer
	if indexOk && !tc.isIntegerType(indexType) {
		line, column := tc.getExpressionPosition(index.Index)
		tc.addError(
			errors.E3003,
			fmt.Sprintf("index must be an integer, got %s", indexType),
			line,
			column,
		)
	}

	// Check that the left side is indexable
	if leftOk && !tc.isArrayType(leftType) && leftType != "string" && !tc.isMapType(leftType) {
		line, column := tc.getExpressionPosition(index.Left)
		tc.addError(
			errors.E3016,
			fmt.Sprintf("cannot index into %s (expected array, string, or map)", leftType),
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
		// Module function call - validate stdlib calls
		tc.checkStdlibCall(fn, call)
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
		// Check if this function might be from a 'using' imported module
		tc.checkDirectStdlibCall(funcName, call)
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
		// These conversions are generally safe - but validate arg count
		if len(call.Arguments) != 1 {
			line, column := tc.getExpressionPosition(call.Function)
			tc.addError(errors.E5008,
				fmt.Sprintf("%s() requires exactly 1 argument, got %d", funcName, len(call.Arguments)),
				line, column)
		}
		return true

	case "len":
		// len() requires exactly 1 argument that is a string or array
		if len(call.Arguments) != 1 {
			line, column := tc.getExpressionPosition(call.Function)
			tc.addError(errors.E5008,
				fmt.Sprintf("len() requires exactly 1 argument, got %d", len(call.Arguments)),
				line, column)
			return true
		}
		argType, ok := tc.inferExpressionType(call.Arguments[0])
		if ok && argType != "string" && !tc.isArrayType(argType) && !tc.isMapType(argType) {
			line, column := tc.getExpressionPosition(call.Arguments[0])
			tc.addError(errors.E3001,
				fmt.Sprintf("len() argument must be string, array, or map, got %s", argType),
				line, column)
		}
		return true

	case "typeof":
		// typeof() requires exactly 1 argument (any type)
		if len(call.Arguments) != 1 {
			line, column := tc.getExpressionPosition(call.Function)
			tc.addError(errors.E5008,
				fmt.Sprintf("typeof() requires exactly 1 argument, got %d", len(call.Arguments)),
				line, column)
		}
		return true

	case "input":
		// input() takes 0 arguments
		if len(call.Arguments) != 0 {
			line, column := tc.getExpressionPosition(call.Function)
			tc.addError(errors.E5008,
				fmt.Sprintf("input() takes 0 arguments, got %d", len(call.Arguments)),
				line, column)
		}
		return true

	case "read_int":
		// read_int() takes 0 arguments
		if len(call.Arguments) != 0 {
			line, column := tc.getExpressionPosition(call.Function)
			tc.addError(errors.E5008,
				fmt.Sprintf("read_int() takes 0 arguments, got %d", len(call.Arguments)),
				line, column)
		}
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

// isMapType checks if a type string represents a map type
func (tc *TypeChecker) isMapType(typeName string) bool {
	return strings.HasPrefix(typeName, "map[") && strings.HasSuffix(typeName, "]")
}

// isHashableType checks if a type can be used as a map key
func (tc *TypeChecker) isHashableType(typeName string) bool {
	switch typeName {
	case "string", "int", "bool", "char":
		return true
	}
	// Also accept integer variants
	return tc.isIntegerType(typeName)
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

// extractArraySize extracts the size from a fixed-size array type string
// Returns -1 for dynamic arrays (e.g., "[int]") or the size for fixed arrays (e.g., "[int, 10]")
func (tc *TypeChecker) extractArraySize(arrType string) int {
	if len(arrType) < 3 || arrType[0] != '[' {
		return -1
	}

	// Remove the outer brackets
	inner := arrType[1 : len(arrType)-1]

	// Look for comma (fixed-size array like "int, 3")
	for i, ch := range inner {
		if ch == ',' {
			sizeStr := strings.TrimSpace(inner[i+1:])
			size, err := strconv.Atoi(sizeStr)
			if err != nil {
				return -1
			}
			return size
		}
	}

	return -1 // Dynamic array
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

	case *ast.MapValue:
		return tc.inferMapType(e)

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

// inferMapType infers the type of a map literal
func (tc *TypeChecker) inferMapType(mapLit *ast.MapValue) (string, bool) {
	if len(mapLit.Pairs) == 0 {
		// Empty map - can't infer types
		return "map[]", true
	}

	// Infer types from first key-value pair
	firstPair := mapLit.Pairs[0]
	keyType, keyOk := tc.inferExpressionType(firstPair.Key)
	valueType, valueOk := tc.inferExpressionType(firstPair.Value)

	if !keyOk || !valueOk {
		return "", false
	}

	return fmt.Sprintf("map[%s:%s]", keyType, valueType), true
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
	case "append", "unshift", "clear", "remove_at", "set":
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

// inferIndexType infers the type when indexing into an array, string, or map
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

	// Indexing into a map returns value type
	if tc.isMapType(leftType) {
		// Extract value type from map[keyType:valueType]
		inner := leftType[4 : len(leftType)-1] // Remove "map[" and "]"
		parts := strings.Split(inner, ":")
		if len(parts) == 2 {
			return parts[1], true
		}
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

	// Handle map type compatibility
	if tc.isMapType(declared) {
		// Empty map map[] is compatible with any map type
		if actual == "map[]" {
			return true
		}

		if tc.isMapType(actual) {
			// Extract key and value types
			declaredInner := declared[4 : len(declared)-1]
			actualInner := actual[4 : len(actual)-1]
			declaredParts := strings.Split(declaredInner, ":")
			actualParts := strings.Split(actualInner, ":")
			if len(declaredParts) == 2 && len(actualParts) == 2 {
				keyCompatible := tc.typesCompatible(declaredParts[0], actualParts[0])
				valueCompatible := tc.typesCompatible(declaredParts[1], actualParts[1])
				return keyCompatible && valueCompatible
			}
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

// ============================================================================
// Standard Library Argument Validation
// ============================================================================

// StdlibFuncSig defines a standard library function signature for validation
type StdlibFuncSig struct {
	MinArgs    int      // Minimum number of arguments
	MaxArgs    int      // Maximum number of arguments (-1 for variadic)
	ArgTypes   []string // Expected argument types (use "any" for any type, "numeric" for numbers, "array" for arrays)
	ReturnType string   // Return type
}

// checkDirectStdlibCall validates a direct function call that might be from an imported module
// This handles cases like: using math; sqrt(4) instead of math.sqrt(4)
func (tc *TypeChecker) checkDirectStdlibCall(funcName string, call *ast.CallExpression) {
	line, column := tc.getExpressionPosition(call.Function)

	// Check which modules are imported via 'using' and see if the function exists there
	if tc.currentScope == nil {
		return
	}

	// Check std module
	if tc.currentScope.HasUsingModule("std") {
		if funcName == "println" || funcName == "print" || funcName == "printf" {
			tc.checkStdModuleCall(funcName, call, line, column)
			return
		}
	}

	// Check math module
	if tc.currentScope.HasUsingModule("math") {
		if tc.isMathFunction(funcName) {
			tc.checkMathModuleCall(funcName, call, line, column)
			return
		}
	}

	// Check arrays module
	if tc.currentScope.HasUsingModule("arrays") {
		if tc.isArraysFunction(funcName) {
			tc.checkArraysModuleCall(funcName, call, line, column)
			return
		}
	}

	// Check strings module
	if tc.currentScope.HasUsingModule("strings") {
		if tc.isStringsFunction(funcName) {
			tc.checkStringsModuleCall(funcName, call, line, column)
			return
		}
	}

	// Check time module
	if tc.currentScope.HasUsingModule("time") {
		if tc.isTimeFunction(funcName) {
			tc.checkTimeModuleCall(funcName, call, line, column)
			return
		}
	}
}

// isMathFunction checks if a function name exists in the math module
func (tc *TypeChecker) isMathFunction(name string) bool {
	mathFuncs := map[string]bool{
		"add": true, "sub": true, "mul": true, "div": true, "mod": true,
		"abs": true, "sign": true, "neg": true, "floor": true, "ceil": true,
		"round": true, "trunc": true, "sqrt": true, "cbrt": true, "exp": true,
		"exp2": true, "log": true, "log2": true, "log10": true, "sin": true,
		"cos": true, "tan": true, "asin": true, "acos": true, "atan": true,
		"sinh": true, "cosh": true, "tanh": true, "pow": true, "hypot": true,
		"atan2": true, "gcd": true, "lcm": true, "deg_to_rad": true, "rad_to_deg": true,
		"min": true, "max": true, "sum": true, "avg": true, "clamp": true,
		"lerp": true, "map_range": true, "distance": true, "factorial": true,
		"is_prime": true, "is_even": true, "is_odd": true, "random": true,
		"random_float": true, "pi": true, "e": true, "phi": true, "sqrt2": true,
		"ln2": true, "ln10": true,
	}
	return mathFuncs[name]
}

// isArraysFunction checks if a function name exists in the arrays module
func (tc *TypeChecker) isArraysFunction(name string) bool {
	arraysFuncs := map[string]bool{
		"len": true, "is_empty": true, "first": true, "last": true, "pop": true,
		"shift": true, "clear": true, "copy": true, "reverse": true, "sort": true,
		"sort_desc": true, "shuffle": true, "unique": true, "duplicates": true,
		"flatten": true, "sum": true, "product": true, "min": true, "max": true,
		"avg": true, "all_equal": true, "append": true, "unshift": true, "contains": true,
		"index_of": true, "last_index_of": true, "count": true, "remove": true,
		"remove_all": true, "fill": true, "get": true, "remove_at": true, "take": true,
		"drop": true, "set": true, "insert": true, "slice": true, "join": true,
		"zip": true, "concat": true, "range": true, "repeat": true,
	}
	return arraysFuncs[name]
}

// isStringsFunction checks if a function name exists in the strings module
func (tc *TypeChecker) isStringsFunction(name string) bool {
	stringsFuncs := map[string]bool{
		"len": true, "upper": true, "lower": true, "trim": true, "contains": true,
		"starts_with": true, "ends_with": true, "index": true, "split": true,
		"join": true, "replace": true,
	}
	return stringsFuncs[name]
}

// isTimeFunction checks if a function name exists in the time module
func (tc *TypeChecker) isTimeFunction(name string) bool {
	timeFuncs := map[string]bool{
		"now": true, "now_ms": true, "now_ns": true, "tick": true, "timezone": true,
		"utc_offset": true, "year": true, "month": true, "day": true, "hour": true,
		"minute": true, "second": true, "weekday": true, "weekday_name": true,
		"month_name": true, "day_of_year": true, "is_leap_year": true, "start_of_day": true,
		"end_of_day": true, "start_of_month": true, "end_of_month": true, "start_of_year": true,
		"end_of_year": true, "iso": true, "date": true, "clock": true, "format": true,
		"parse": true, "sleep": true, "sleep_ms": true, "add_seconds": true,
		"add_minutes": true, "add_hours": true, "add_days": true, "diff": true,
		"diff_days": true, "is_before": true, "is_after": true, "make": true,
		"days_in_month": true, "elapsed_ms": true,
	}
	return timeFuncs[name]
}

// checkStdlibCall validates a standard library module function call
func (tc *TypeChecker) checkStdlibCall(member *ast.MemberExpression, call *ast.CallExpression) {
	// Get module name
	moduleName := ""
	if label, ok := member.Object.(*ast.Label); ok {
		moduleName = label.Value
	} else {
		return
	}

	funcName := member.Member.Value
	line, column := tc.getExpressionPosition(member.Member)

	switch moduleName {
	case "std":
		tc.checkStdModuleCall(funcName, call, line, column)
	case "math":
		tc.checkMathModuleCall(funcName, call, line, column)
	case "arrays":
		tc.checkArraysModuleCall(funcName, call, line, column)
	case "strings":
		tc.checkStringsModuleCall(funcName, call, line, column)
	case "time":
		tc.checkTimeModuleCall(funcName, call, line, column)
	case "maps":
		tc.checkMapsModuleCall(funcName, call, line, column)
	}
}

// checkStdModuleCall validates std module function calls
func (tc *TypeChecker) checkStdModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	switch funcName {
	case "println", "print":
		// Accept any arguments (variadic, any type)
		return
	case "printf":
		// First argument must be a string (format string)
		if len(call.Arguments) < 1 {
			tc.addError(errors.E5008, fmt.Sprintf("std.%s requires at least 1 argument (format string)", funcName), line, column)
			return
		}
		argType, ok := tc.inferExpressionType(call.Arguments[0])
		if ok && argType != "string" {
			tc.addError(errors.E3001, fmt.Sprintf("std.%s format argument must be string, got %s", funcName, argType), line, column)
		}
	}
}

// checkMathModuleCall validates math module function calls
func (tc *TypeChecker) checkMathModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	// Define expected signatures
	signatures := map[string]StdlibFuncSig{
		// Basic arithmetic (2 numeric args)
		"add": {2, 2, []string{"numeric", "numeric"}, "float"},
		"sub": {2, 2, []string{"numeric", "numeric"}, "float"},
		"mul": {2, 2, []string{"numeric", "numeric"}, "float"},
		"div": {2, 2, []string{"numeric", "numeric"}, "float"},
		"mod": {2, 2, []string{"numeric", "numeric"}, "float"},

		// Single numeric arg
		"abs":   {1, 1, []string{"numeric"}, "float"},
		"sign":  {1, 1, []string{"numeric"}, "int"},
		"neg":   {1, 1, []string{"numeric"}, "float"},
		"floor": {1, 1, []string{"numeric"}, "int"},
		"ceil":  {1, 1, []string{"numeric"}, "int"},
		"round": {1, 1, []string{"numeric"}, "int"},
		"trunc": {1, 1, []string{"numeric"}, "int"},
		"sqrt":  {1, 1, []string{"numeric"}, "float"},
		"cbrt":  {1, 1, []string{"numeric"}, "float"},
		"exp":   {1, 1, []string{"numeric"}, "float"},
		"exp2":  {1, 1, []string{"numeric"}, "float"},
		"log":   {1, 1, []string{"numeric"}, "float"},
		"log2":  {1, 1, []string{"numeric"}, "float"},
		"log10": {1, 1, []string{"numeric"}, "float"},

		// Trigonometry
		"sin":  {1, 1, []string{"numeric"}, "float"},
		"cos":  {1, 1, []string{"numeric"}, "float"},
		"tan":  {1, 1, []string{"numeric"}, "float"},
		"asin": {1, 1, []string{"numeric"}, "float"},
		"acos": {1, 1, []string{"numeric"}, "float"},
		"atan": {1, 1, []string{"numeric"}, "float"},
		"sinh": {1, 1, []string{"numeric"}, "float"},
		"cosh": {1, 1, []string{"numeric"}, "float"},
		"tanh": {1, 1, []string{"numeric"}, "float"},

		// Two numeric args
		"pow":        {2, 2, []string{"numeric", "numeric"}, "float"},
		"hypot":      {2, 2, []string{"numeric", "numeric"}, "float"},
		"atan2":      {2, 2, []string{"numeric", "numeric"}, "float"},
		"gcd":        {2, 2, []string{"int", "int"}, "int"},
		"lcm":        {2, 2, []string{"int", "int"}, "int"},
		"deg_to_rad": {1, 1, []string{"numeric"}, "float"},
		"rad_to_deg": {1, 1, []string{"numeric"}, "float"},

		// Variadic numeric
		"min": {2, -1, []string{"numeric"}, "float"},
		"max": {2, -1, []string{"numeric"}, "float"},
		"sum": {1, -1, []string{"numeric"}, "float"},
		"avg": {1, -1, []string{"numeric"}, "float"},

		// Three args
		"clamp": {3, 3, []string{"numeric", "numeric", "numeric"}, "float"},
		"lerp":  {3, 3, []string{"numeric", "numeric", "numeric"}, "float"},

		// Five args
		"map_range": {5, 5, []string{"numeric", "numeric", "numeric", "numeric", "numeric"}, "float"},
		"distance":  {4, 4, []string{"numeric", "numeric", "numeric", "numeric"}, "float"},

		// Integer-only
		"factorial": {1, 1, []string{"int"}, "int"},
		"is_prime":  {1, 1, []string{"int"}, "bool"},
		"is_even":   {1, 1, []string{"int"}, "bool"},
		"is_odd":    {1, 1, []string{"int"}, "bool"},

		// Random (variable args)
		"random":       {0, 2, []string{"int", "int"}, "int"},
		"random_float": {0, 2, []string{"numeric", "numeric"}, "float"},

		// Constants (no args)
		"pi":    {0, 0, []string{}, "float"},
		"e":     {0, 0, []string{}, "float"},
		"phi":   {0, 0, []string{}, "float"},
		"sqrt2": {0, 0, []string{}, "float"},
		"ln2":   {0, 0, []string{}, "float"},
		"ln10":  {0, 0, []string{}, "float"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return // Unknown function, let runtime handle
	}

	tc.validateStdlibCall("math", funcName, call, sig, line, column)
}

// checkArraysModuleCall validates arrays module function calls
func (tc *TypeChecker) checkArraysModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// Single array arg
		"len":        {1, 1, []string{"array"}, "int"},
		"is_empty":   {1, 1, []string{"array"}, "bool"},
		"first":      {1, 1, []string{"array"}, "any"},
		"last":       {1, 1, []string{"array"}, "any"},
		"pop":        {1, 1, []string{"array"}, "any"},
		"shift":      {1, 1, []string{"array"}, "any"},
		"clear":      {1, 1, []string{"array"}, "array"},
		"copy":       {1, 1, []string{"array"}, "array"},
		"reverse":    {1, 1, []string{"array"}, "array"},
		"sort":       {1, 1, []string{"array"}, "array"},
		"sort_desc":  {1, 1, []string{"array"}, "array"},
		"shuffle":    {1, 1, []string{"array"}, "array"},
		"unique":     {1, 1, []string{"array"}, "array"},
		"duplicates": {1, 1, []string{"array"}, "array"},
		"flatten":    {1, 1, []string{"array"}, "array"},
		"sum":        {1, 1, []string{"array"}, "float"},
		"product":    {1, 1, []string{"array"}, "float"},
		"min":        {1, 1, []string{"array"}, "any"},
		"max":        {1, 1, []string{"array"}, "any"},
		"avg":        {1, 1, []string{"array"}, "float"},
		"all_equal":  {1, 1, []string{"array"}, "bool"},

		// Array + value
		"append":        {2, -1, []string{"array", "any"}, "void"},
		"unshift":       {2, -1, []string{"array", "any"}, "array"},
		"contains":      {2, 2, []string{"array", "any"}, "bool"},
		"index_of":      {2, 2, []string{"array", "any"}, "int"},
		"last_index_of": {2, 2, []string{"array", "any"}, "int"},
		"count":         {2, 2, []string{"array", "any"}, "int"},
		"remove":        {2, 2, []string{"array", "any"}, "array"},
		"remove_all":    {2, 2, []string{"array", "any"}, "array"},
		"fill":          {2, 2, []string{"array", "any"}, "array"},

		// Array + int
		"get":       {2, 2, []string{"array", "int"}, "any"},
		"remove_at": {2, 2, []string{"array", "int"}, "array"},
		"take":      {2, 2, []string{"array", "int"}, "array"},
		"drop":      {2, 2, []string{"array", "int"}, "array"},

		// Array + int + value
		"set":    {3, 3, []string{"array", "int", "any"}, "array"},
		"insert": {3, 3, []string{"array", "int", "any"}, "array"},

		// Array + int + int (optional)
		"slice": {2, 3, []string{"array", "int", "int"}, "array"},

		// Array + string
		"join": {2, 2, []string{"array", "string"}, "string"},

		// Two arrays
		"zip":    {2, 2, []string{"array", "array"}, "array"},
		"concat": {1, -1, []string{"array"}, "array"},

		// Creation functions
		"range":  {1, 3, []string{"int", "int", "int"}, "array"},
		"repeat": {2, 2, []string{"any", "int"}, "array"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("arrays", funcName, call, sig, line, column)
}

// checkMapsModuleCall validates maps module function calls
func (tc *TypeChecker) checkMapsModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// Single map arg
		"len":      {1, 1, []string{"map"}, "int"},
		"is_empty": {1, 1, []string{"map"}, "bool"},
		"keys":     {1, 1, []string{"map"}, "array"},
		"values":   {1, 1, []string{"map"}, "array"},
		"clear":    {1, 1, []string{"map"}, "void"},
		"copy":     {1, 1, []string{"map"}, "map"},
		"to_array": {1, 1, []string{"map"}, "array"},
		"invert":   {1, 1, []string{"map"}, "map"},

		// Map + key
		"has":     {2, 2, []string{"map", "any"}, "bool"},
		"has_key": {2, 2, []string{"map", "any"}, "bool"},
		"delete":  {2, 2, []string{"map", "any"}, "bool"},
		"remove":  {2, 2, []string{"map", "any"}, "bool"},

		// Map + value
		"has_value": {2, 2, []string{"map", "any"}, "bool"},

		// Map + key + optional default
		"get": {2, 3, []string{"map", "any", "any"}, "any"},

		// Map + key + value
		"set":        {3, 3, []string{"map", "any", "any"}, "void"},
		"get_or_set": {3, 3, []string{"map", "any", "any"}, "any"},

		// Two maps
		"equals": {2, 2, []string{"map", "map"}, "bool"},

		// Variadic map operations (map, map, ...)
		"merge":  {2, -1, []string{"map"}, "map"},
		"update": {2, -1, []string{"map"}, "void"},

		// Array to map
		"from_array": {1, 1, []string{"array"}, "map"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("maps", funcName, call, sig, line, column)
}

// checkStringsModuleCall validates strings module function calls
func (tc *TypeChecker) checkStringsModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// Single string arg
		"len":   {1, 1, []string{"string"}, "int"},
		"upper": {1, 1, []string{"string"}, "string"},
		"lower": {1, 1, []string{"string"}, "string"},
		"trim":  {1, 1, []string{"string"}, "string"},

		// String + string
		"contains":    {2, 2, []string{"string", "string"}, "bool"},
		"starts_with": {2, 2, []string{"string", "string"}, "bool"},
		"ends_with":   {2, 2, []string{"string", "string"}, "bool"},
		"index":       {2, 2, []string{"string", "string"}, "int"},
		"split":       {2, 2, []string{"string", "string"}, "array"},

		// Array + string (for join)
		"join": {2, 2, []string{"array", "string"}, "string"},

		// String + string + string
		"replace": {3, 3, []string{"string", "string", "string"}, "string"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("strings", funcName, call, sig, line, column)
}

// checkTimeModuleCall validates time module function calls
func (tc *TypeChecker) checkTimeModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// No args
		"now":        {0, 0, []string{}, "int"},
		"now_ms":     {0, 0, []string{}, "int"},
		"now_ns":     {0, 0, []string{}, "int"},
		"tick":       {0, 0, []string{}, "int"},
		"timezone":   {0, 0, []string{}, "string"},
		"utc_offset": {0, 0, []string{}, "int"},

		// Optional timestamp (0 or 1 int arg)
		"year":           {0, 1, []string{"int"}, "int"},
		"month":          {0, 1, []string{"int"}, "int"},
		"day":            {0, 1, []string{"int"}, "int"},
		"hour":           {0, 1, []string{"int"}, "int"},
		"minute":         {0, 1, []string{"int"}, "int"},
		"second":         {0, 1, []string{"int"}, "int"},
		"weekday":        {0, 1, []string{"int"}, "int"},
		"weekday_name":   {0, 1, []string{"int"}, "string"},
		"month_name":     {0, 1, []string{"int"}, "string"},
		"day_of_year":    {0, 1, []string{"int"}, "int"},
		"is_leap_year":   {0, 1, []string{"int"}, "bool"},
		"start_of_day":   {0, 1, []string{"int"}, "int"},
		"end_of_day":     {0, 1, []string{"int"}, "int"},
		"start_of_month": {0, 1, []string{"int"}, "int"},
		"end_of_month":   {0, 1, []string{"int"}, "int"},
		"start_of_year":  {0, 1, []string{"int"}, "int"},
		"end_of_year":    {0, 1, []string{"int"}, "int"},
		"iso":            {0, 1, []string{"int"}, "string"},
		"date":           {0, 1, []string{"int"}, "string"},
		"clock":          {0, 1, []string{"int"}, "string"},

		// Format functions
		"format": {1, 2, []string{"string", "int"}, "string"},
		"parse":  {2, 2, []string{"string", "string"}, "int"},

		// Sleep (numeric arg - int or float)
		"sleep":    {1, 1, []string{"numeric"}, "void"},
		"sleep_ms": {1, 1, []string{"int"}, "void"},

		// Arithmetic (timestamp + value)
		"add_seconds": {2, 2, []string{"int", "int"}, "int"},
		"add_minutes": {2, 2, []string{"int", "int"}, "int"},
		"add_hours":   {2, 2, []string{"int", "int"}, "int"},
		"add_days":    {2, 2, []string{"int", "int"}, "int"},

		// Difference
		"diff":      {2, 2, []string{"int", "int"}, "int"},
		"diff_days": {2, 2, []string{"int", "int"}, "int"},

		// Comparisons
		"is_before": {2, 2, []string{"int", "int"}, "bool"},
		"is_after":  {2, 2, []string{"int", "int"}, "bool"},

		// Creation
		"make": {3, 6, []string{"int", "int", "int", "int", "int", "int"}, "int"},

		// days_in_month (0-2 args)
		"days_in_month": {0, 2, []string{"int", "int"}, "int"},

		// elapsed_ms
		"elapsed_ms": {1, 1, []string{"int"}, "float"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("time", funcName, call, sig, line, column)
}

// validateStdlibCall performs the actual validation of a stdlib call
func (tc *TypeChecker) validateStdlibCall(moduleName, funcName string, call *ast.CallExpression, sig StdlibFuncSig, line, column int) {
	argCount := len(call.Arguments)

	// Check argument count
	if argCount < sig.MinArgs {
		tc.addError(errors.E5008,
			fmt.Sprintf("%s.%s requires at least %d argument(s), got %d", moduleName, funcName, sig.MinArgs, argCount),
			line, column)
		return
	}

	if sig.MaxArgs >= 0 && argCount > sig.MaxArgs {
		tc.addError(errors.E5008,
			fmt.Sprintf("%s.%s accepts at most %d argument(s), got %d", moduleName, funcName, sig.MaxArgs, argCount),
			line, column)
		return
	}

	// Check argument types
	for i, arg := range call.Arguments {
		actualType, ok := tc.inferExpressionType(arg)
		if !ok {
			continue // Can't determine type
		}

		// Determine expected type for this argument
		var expectedType string
		if i < len(sig.ArgTypes) {
			expectedType = sig.ArgTypes[i]
		} else if len(sig.ArgTypes) > 0 {
			// For variadic functions, use the last type pattern
			expectedType = sig.ArgTypes[len(sig.ArgTypes)-1]
		} else {
			continue // No type constraints
		}

		// Validate the type
		if !tc.typeMatchesExpected(actualType, expectedType) {
			argLine, argColumn := tc.getExpressionPosition(arg)
			tc.addError(errors.E3001,
				fmt.Sprintf("%s.%s argument %d: expected %s, got %s", moduleName, funcName, i+1, expectedType, actualType),
				argLine, argColumn)
		}
	}
}

// typeMatchesExpected checks if an actual type matches an expected type constraint
func (tc *TypeChecker) typeMatchesExpected(actual, expected string) bool {
	switch expected {
	case "any":
		return true
	case "numeric":
		return tc.isNumericType(actual)
	case "int":
		return tc.isIntegerType(actual)
	case "float":
		return actual == "float" || actual == "f32" || actual == "f64"
	case "string":
		return actual == "string"
	case "bool":
		return actual == "bool"
	case "array":
		return tc.isArrayType(actual)
	case "map":
		return tc.isMapType(actual)
	default:
		return tc.typesCompatible(expected, actual)
	}
}
