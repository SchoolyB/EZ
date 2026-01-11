package typechecker

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"math/big"
	"sort"
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
	parent        *Scope
	variables     map[string]string // variable name -> type name
	mutability    map[string]bool   // variable name -> is mutable
	usingModules  map[string]bool   // modules imported via 'using'
	loopVariables map[string]bool   // loop variable names in this scope
}

// NewScope creates a new scope with an optional parent
func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:        parent,
		variables:     make(map[string]string),
		mutability:    make(map[string]bool),
		usingModules:  make(map[string]bool),
		loopVariables: make(map[string]bool),
	}
}

// Define adds a variable to the current scope (defaults to immutable)
func (s *Scope) Define(name, typeName string) {
	s.variables[name] = typeName
	s.mutability[name] = false
}

// DefineWithMutability adds a variable to the current scope with explicit mutability
func (s *Scope) DefineWithMutability(name, typeName string, mutable bool) {
	s.variables[name] = typeName
	s.mutability[name] = mutable
}

// IsMutable checks if a variable is mutable in the current scope or any parent scope
func (s *Scope) IsMutable(name string) (bool, bool) {
	if mutable, ok := s.mutability[name]; ok {
		return mutable, true
	}
	if s.parent != nil {
		return s.parent.IsMutable(name)
	}
	return false, false
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

// MarkAsLoopVariable marks a variable as a loop variable in the current scope
func (s *Scope) MarkAsLoopVariable(name string) {
	s.loopVariables[name] = true
}

// IsLoopVariableInOuterScope checks if a variable name is used as a loop variable
// in any parent scope (not the current scope)
func (s *Scope) IsLoopVariableInOuterScope(name string) bool {
	if s.parent == nil {
		return false
	}
	// Check parent scope and all its ancestors
	return s.parent.isLoopVariableInScopeChain(name)
}

// isLoopVariableInScopeChain checks if a variable is a loop variable in this scope or any parent
func (s *Scope) isLoopVariableInScopeChain(name string) bool {
	if s.loopVariables[name] {
		return true
	}
	if s.parent != nil {
		return s.parent.isLoopVariableInScopeChain(name)
	}
	return false
}

// Type represents a type in the EZ type system
type Type struct {
	Name         string
	Kind         TypeKind
	ElementType  *Type            // For arrays
	KeyType      *Type            // For maps
	ValueType    *Type            // For maps
	Fields       map[string]*Type // For structs
	Size         int              // For fixed-size arrays, -1 for dynamic
	EnumBaseType string           // For enums: "int", "string", or "float"
	EnumMembers  map[string]bool  // For enums: set of valid member names (#607)
}

// FunctionSignature represents a function's type signature
type FunctionSignature struct {
	Name        string
	Parameters  []*Parameter
	ReturnTypes []string
}

// Parameter represents a function parameter with type
type Parameter struct {
	Name       string
	Type       string
	Mutable    bool // true if declared with & prefix
	HasDefault bool // true if parameter has a default value
}

// ImportLocation stores the location of an import statement for error reporting
type ImportLocation struct {
	Line   int
	Column int
}

// TypeChecker validates types in an EZ program
type TypeChecker struct {
	types                map[string]*Type                         // All known types
	functions            map[string]*FunctionSignature            // All function signatures
	variables            map[string]string                        // Variable name -> type name (global scope)
	modules              map[string]bool                          // Imported module names
	usedModules          map[string]bool                          // Modules that are actually used (#639)
	importLocations      map[string]ImportLocation                // Module name -> import location for warnings (#639)
	stdlibAliases        map[string]string                        // Stdlib alias -> module name
	fileUsingModules     map[string]bool                          // File-level using modules
	moduleFunctions      map[string]map[string]*FunctionSignature // Module name -> function name -> signature
	moduleTypes          map[string]map[string]*Type              // Module name -> type name -> type
	moduleVariables      map[string]map[string]string             // Module name -> variable name -> type (#677)
	currentScope         *Scope                                   // Current scope for local variable tracking
	errors               *errors.EZErrorList
	source               string
	filename             string
	skipMainCheck        bool             // Skip main() function requirement (for module files)
	loopDepth            int              // Track nesting depth of loops for break/continue validation (#603)
	currentFuncAttrs     []*ast.Attribute // Current function's attributes for #suppress checking
	fileSuppressWarnings []string         // File-level suppressed warnings (from #suppress at file scope)
	currentModuleName    string           // Current module name for same-module symbol lookup
}

// NewTypeChecker creates a new type checker
func NewTypeChecker(source, filename string) *TypeChecker {
	tc := &TypeChecker{
		types:            make(map[string]*Type),
		functions:        make(map[string]*FunctionSignature),
		variables:        make(map[string]string),
		modules:          make(map[string]bool),
		usedModules:      make(map[string]bool),
		importLocations:  make(map[string]ImportLocation),
		stdlibAliases:    make(map[string]string),
		fileUsingModules: make(map[string]bool),
		moduleFunctions:  make(map[string]map[string]*FunctionSignature),
		moduleTypes:      make(map[string]map[string]*Type),
		moduleVariables:  make(map[string]map[string]string),
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

// SetCurrentModule sets the current module name for same-module symbol lookup
// This allows files in the same module to access each other's symbols without qualification
func (tc *TypeChecker) SetCurrentModule(moduleName string) {
	tc.currentModuleName = moduleName
}

// RegisterModuleFunction registers a function signature from an imported module
func (tc *TypeChecker) RegisterModuleFunction(moduleName, funcName string, sig *FunctionSignature) {
	if tc.moduleFunctions[moduleName] == nil {
		tc.moduleFunctions[moduleName] = make(map[string]*FunctionSignature)
	}
	tc.moduleFunctions[moduleName][funcName] = sig
}

// RegisterModuleType registers a type from an imported module
func (tc *TypeChecker) RegisterModuleType(moduleName, typeName string, t *Type) {
	if tc.moduleTypes[moduleName] == nil {
		tc.moduleTypes[moduleName] = make(map[string]*Type)
	}
	tc.moduleTypes[moduleName][typeName] = t
}

// RegisterModuleVariable registers a variable/constant from an imported module (#677)
func (tc *TypeChecker) RegisterModuleVariable(moduleName, varName, typeName string) {
	if tc.moduleVariables[moduleName] == nil {
		tc.moduleVariables[moduleName] = make(map[string]string)
	}
	tc.moduleVariables[moduleName][varName] = typeName
}

// GetModuleFunction retrieves a function signature from a module
func (tc *TypeChecker) GetModuleFunction(moduleName, funcName string) (*FunctionSignature, bool) {
	if funcs, ok := tc.moduleFunctions[moduleName]; ok {
		sig, exists := funcs[funcName]
		return sig, exists
	}
	return nil, false
}

// GetModuleVariable retrieves a variable type from a module (#677)
func (tc *TypeChecker) GetModuleVariable(moduleName, varName string) (string, bool) {
	if vars, ok := tc.moduleVariables[moduleName]; ok {
		typeName, exists := vars[varName]
		return typeName, exists
	}
	return "", false
}

// GetFunctions returns the functions map (for extracting signatures from module typechecker)
func (tc *TypeChecker) GetFunctions() map[string]*FunctionSignature {
	return tc.functions
}

// GetTypes returns the types map (for extracting types from module typechecker)
func (tc *TypeChecker) GetTypes() map[string]*Type {
	return tc.types
}

// GetVariables returns the variables map (for extracting constants from module typechecker) (#677)
func (tc *TypeChecker) GetVariables() map[string]string {
	return tc.variables
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
		"bool", "char", "string", "byte",
		// Special
		"void", "nil",
		// Internal types (not for user code - will be rejected by E3034)
		"any",
	}

	for _, name := range primitives {
		tc.types[name] = &Type{
			Name: name,
			Kind: PrimitiveType,
		}
	}

	// Register built-in Error struct (both "Error" and "error" alias)
	errorType := &Type{
		Name: "Error",
		Kind: StructType,
		Fields: map[string]*Type{
			"message": {Name: "string", Kind: PrimitiveType},
			"code":    {Name: "int", Kind: PrimitiveType},
		},
	}
	tc.types["Error"] = errorType
	tc.types["error"] = errorType // Alias for convenience

	// Built-in HTTP Response struct
	httpResponseType := &Type{
		Name: "HttpResponse",
		Kind: StructType,
		Fields: map[string]*Type{
			"status":  {Name: "int", Kind: PrimitiveType},
			"body":    {Name: "string", Kind: PrimitiveType},
			"headers": {Name: "map[string:[string]]", Kind: MapType},
		},
	}
	tc.types["HttpResponse"] = httpResponseType
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
		keyType, valueType, ok := tc.splitMapInnerType(inner)
		if ok {
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
			typeNamePart := parts[1]
			// Check if the module has been imported
			if tc.modules[moduleName] {
				return true
			}
			// Also check registered module types (#722 - self-referencing types)
			if modTypes, ok := tc.moduleTypes[moduleName]; ok {
				if _, exists := modTypes[typeNamePart]; exists {
					return true
				}
			}
		}
	}

	// Check local types first
	if _, exists := tc.types[typeName]; exists {
		return true
	}

	// Check if the type looks like a built-in sized type pattern (i<N>, u<N>, f<N>)
	// If it matches the pattern but wasn't found in tc.types, it's definitely invalid
	// and should NOT be deferred to module resolution
	if looksLikeBuiltinSizedType(typeName) {
		return false
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

// looksLikeBuiltinSizedType checks if a type name follows the pattern of built-in
// sized types (i8, i16, u32, f64, etc.) but is NOT a valid one.
// This prevents invalid types like i512 or u1024 from being deferred to module resolution.
func looksLikeBuiltinSizedType(typeName string) bool {
	if len(typeName) < 2 {
		return false
	}

	prefix := typeName[0]
	if prefix != 'i' && prefix != 'u' && prefix != 'f' {
		return false
	}

	// Check if the rest is all digits
	for _, c := range typeName[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}

	// It matches the pattern (e.g., i512, u1024, f128)
	// If we got here, it wasn't found in tc.types, so it's an invalid sized type
	return true
}

// RegisterType adds a user-defined type to the registry
func (tc *TypeChecker) RegisterType(name string, t *Type) {
	tc.types[name] = t
}

// RegisterFunction adds a function signature to the registry
func (tc *TypeChecker) RegisterFunction(name string, sig *FunctionSignature) {
	tc.functions[name] = sig
}

// RegisterVariable adds a variable/constant to the global scope (#722)
func (tc *TypeChecker) RegisterVariable(name, typeName string) {
	tc.variables[name] = typeName
}

// GetType retrieves a type by name
func (tc *TypeChecker) GetType(name string) (*Type, bool) {
	t, ok := tc.types[name]
	return t, ok
}

// getStructTypeIncludingModules looks up a struct type by name, checking both local
// and module types. For qualified names like "lib.Hero", it looks up in moduleTypes.
// For unqualified names like "Item", it also searches through all registered modules.
func (tc *TypeChecker) getStructTypeIncludingModules(typeName string) (*Type, bool) {
	// First check local types
	if t, exists := tc.types[typeName]; exists && t.Kind == StructType {
		return t, true
	}

	// Check if it's a qualified type (e.g., "lib.Hero")
	if strings.Contains(typeName, ".") {
		parts := strings.SplitN(typeName, ".", 2)
		if len(parts) == 2 {
			moduleName := parts[0]
			baseTypeName := parts[1]
			if moduleTypes, hasModule := tc.moduleTypes[moduleName]; hasModule {
				if t, exists := moduleTypes[baseTypeName]; exists && t.Kind == StructType {
					return t, true
				}
			}
		}
	}

	// For unqualified names, search through all registered modules
	// This handles cases where a struct field type like "[Item]" references
	// a type from the same module without qualification
	for _, moduleTypes := range tc.moduleTypes {
		if t, exists := moduleTypes[typeName]; exists && t.Kind == StructType {
			return t, true
		}
	}

	return nil, false
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
	// Store file-level suppressions
	tc.fileSuppressWarnings = program.FileSuppressWarnings

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
				if item.IsStdlib && item.Module != "" && moduleName != "" && moduleName != item.Module {
					tc.stdlibAliases[moduleName] = item.Module
				}

				// Store import location for unused import warnings (#639)
				tc.importLocations[moduleName] = ImportLocation{
					Line:   importStmt.Token.Line,
					Column: importStmt.Token.Column,
				}

				// If this is an "import & use" statement, also register for file-level using
				// This allows unqualified access to types from the module
				if importStmt.AutoUse {
					tc.fileUsingModules[moduleName] = true
					// Mark as used since "import & use" implies the module will be used
					tc.usedModules[moduleName] = true
				}
			}
		}
	}

	// Phase 0.5: Register file-level using modules
	for _, usingStmt := range program.FileUsing {
		for _, mod := range usingStmt.Modules {
			tc.fileUsingModules[mod.Value] = true
			// Mark as used since explicit 'using' statement implies usage
			tc.usedModules[mod.Value] = true
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

	// Phase 3: Check for invalid file-scope statements (#662)
	tc.checkFileScopeStatements(program.Statements)

	// Phase 4: Type check function bodies
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionDeclaration); ok {
			tc.checkFunctionBody(fn)
		}
	}

	// Phase 5: Validate that a main() function exists (unless skipped for module files)
	if !tc.skipMainCheck {
		tc.checkMainFunction()
	}

	// Phase 6: Check for unused imports (#639)
	tc.checkUnusedImports()

	errCount, _ := tc.errors.Count()
	return errCount == 0
}

// RegisterDeclarations performs a lightweight pass to register type and function
// declarations without full type checking. Used for multi-file modules to make
// types available before checking files that depend on them (#709).
func (tc *TypeChecker) RegisterDeclarations(program *ast.Program) {
	// Phase 0: Register all imported modules
	for _, stmt := range program.Statements {
		if importStmt, ok := stmt.(*ast.ImportStatement); ok {
			for _, item := range importStmt.Imports {
				moduleName := item.Alias
				if moduleName == "" {
					moduleName = item.Module
				}
				tc.modules[moduleName] = true
				if item.IsStdlib && item.Module != "" && moduleName != "" && moduleName != item.Module {
					tc.stdlibAliases[moduleName] = item.Module
				}
				if importStmt.AutoUse {
					tc.fileUsingModules[moduleName] = true
				}
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

	// Phase 1.5: Populate struct fields (without validation errors)
	for _, stmt := range program.Statements {
		if node, ok := stmt.(*ast.StructDeclaration); ok {
			tc.populateStructFields(node)
		}
	}

	// Phase 2: Register function signatures
	for _, stmt := range program.Statements {
		switch node := stmt.(type) {
		case *ast.FunctionDeclaration:
			sig := &FunctionSignature{
				Name:        node.Name.Value,
				Parameters:  []*Parameter{},
				ReturnTypes: node.ReturnTypes,
			}
			for _, param := range node.Parameters {
				sig.Parameters = append(sig.Parameters, &Parameter{
					Name:       param.Name.Value,
					Type:       param.TypeName,
					Mutable:    param.Mutable,
					HasDefault: param.DefaultValue != nil,
				})
			}
			tc.RegisterFunction(node.Name.Value, sig)
		case *ast.VariableDeclaration:
			// Register global constants/variables
			varType := node.TypeName
			if varType == "" {
				if inferred, ok := tc.inferExpressionType(node.Value); ok {
					varType = inferred
				}
			}
			tc.variables[node.Name.Value] = varType
		}
	}
}

// populateStructFields adds field types to a struct without validation errors
func (tc *TypeChecker) populateStructFields(node *ast.StructDeclaration) {
	structType, ok := tc.GetType(node.Name.Value)
	if !ok {
		return
	}

	for _, field := range node.Fields {
		// Create type for the field (skip validation - will be done in full check)
		var fieldType *Type
		if tc.isArrayType(field.TypeName) {
			fieldType = &Type{Name: field.TypeName, Kind: ArrayType}
		} else if tc.isMapType(field.TypeName) {
			fieldType = &Type{Name: field.TypeName, Kind: MapType}
		} else if existingType, exists := tc.GetType(field.TypeName); exists {
			fieldType = existingType
		} else {
			// Type doesn't exist yet - create placeholder
			fieldType = &Type{Name: field.TypeName, Kind: StructType}
		}
		structType.Fields[field.Name.Value] = fieldType
	}
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
	// Determine base type from attributes, default to "int"
	baseType := "int"
	if node.Attributes != nil && node.Attributes.TypeName != "" {
		baseType = node.Attributes.TypeName
	} else {
		// Infer base type from first value if no explicit attribute
		for _, member := range node.Values {
			if member.Value != nil {
				baseType = tc.getEnumValueType(member.Value)
				if baseType == "" {
					baseType = "int" // fallback
				}
				break
			}
		}
	}

	// Collect enum member names (#607)
	enumMembers := make(map[string]bool)
	for _, member := range node.Values {
		enumMembers[member.Name.Value] = true
	}

	enumType := &Type{
		Name:         node.Name.Value,
		Kind:         EnumType,
		EnumBaseType: baseType,
		EnumMembers:  enumMembers,
	}
	tc.RegisterType(node.Name.Value, enumType)
}

// checkStructDeclaration validates a struct's field types
func (tc *TypeChecker) checkStructDeclaration(node *ast.StructDeclaration) {
	structType, ok := tc.GetType(node.Name.Value)
	if !ok {
		return
	}

	for _, field := range node.Fields {
		// Check if field type exists
		if !tc.TypeExists(field.TypeName) {
			tc.addError(
				errors.E3009,
				fmt.Sprintf("undefined type '%s' in struct '%s'", field.TypeName, node.Name.Value),
				field.Name.Token.Line,
				field.Name.Token.Column,
			)
			continue
		}

		// Check if 'void' type is used (not allowed)
		if tc.containsVoidType(field.TypeName) {
			tc.addError(
				errors.E3038,
				fmt.Sprintf("'void' type cannot be used as struct field type for '%s'", field.Name.Value),
				field.Name.Token.Line,
				field.Name.Token.Column,
			)
			continue
		}

		// Add field to struct type
		fieldType, ok := tc.GetType(field.TypeName)
		if !ok {
			// For array/map types, create a Type on-the-fly since they're not in the registry
			if tc.isArrayType(field.TypeName) {
				fieldType = &Type{Name: field.TypeName, Kind: ArrayType}
			} else if tc.isMapType(field.TypeName) {
				fieldType = &Type{Name: field.TypeName, Kind: MapType}
			} else {
				// This shouldn't happen since TypeExists passed, but be safe
				continue
			}
		}
		structType.Fields[field.Name.Value] = fieldType
	}
}

// checkStructLiteral validates a struct literal's field values against the type definition
func (tc *TypeChecker) checkStructLiteral(structVal *ast.StructValue) {
	if structVal.Name == nil {
		return
	}

	structName := structVal.Name.Value
	// Use getStructTypeIncludingModules to handle both local types and
	// qualified imported types like "lib.Item"
	structType, exists := tc.getStructTypeIncludingModules(structName)
	if !exists {
		// Struct type doesn't exist - will be caught elsewhere
		return
	}

	// Check each field in the literal
	for fieldName, fieldValue := range structVal.Fields {
		// Check for type/function used as field value
		tc.checkValueExpression(fieldValue)
		tc.checkExpression(fieldValue)

		// Get the expected type for this field
		expectedType, fieldExists := structType.Fields[fieldName]
		if !fieldExists {
			// Field doesn't exist on struct - report error
			line, column := tc.getExpressionPosition(fieldValue)
			tc.addError(
				errors.E4003,
				fmt.Sprintf("struct '%s' has no field '%s'", structName, fieldName),
				line,
				column,
			)
			continue
		}

		// Infer the actual type of the field value
		actualType, ok := tc.inferExpressionType(fieldValue)
		if !ok {
			continue
		}

		// Check type compatibility
		if !tc.typesCompatible(expectedType.Name, actualType) {
			line, column := tc.getExpressionPosition(fieldValue)
			tc.addError(
				errors.E3001,
				fmt.Sprintf("struct field '%s' expects %s, got %s",
					fieldName, expectedType.Name, actualType),
				line,
				column,
			)
		}
	}
}

// checkArrayLiteral validates array literal elements have consistent types
func (tc *TypeChecker) checkArrayLiteral(arr *ast.ArrayValue) {
	if len(arr.Elements) == 0 {
		return // Empty array is OK
	}

	// Check each element for type/function used as value
	for _, elem := range arr.Elements {
		tc.checkValueExpression(elem)
		tc.checkExpression(elem)
	}

	// Get the type of the first element
	firstType, ok := tc.inferExpressionType(arr.Elements[0])
	if !ok {
		return // Can't determine type
	}

	// All other elements must have the same type
	for i := 1; i < len(arr.Elements); i++ {
		elemType, ok := tc.inferExpressionType(arr.Elements[i])
		if !ok {
			continue
		}

		if !tc.typesCompatible(firstType, elemType) {
			line, column := tc.getExpressionPosition(arr.Elements[i])
			tc.addError(
				errors.E3001,
				fmt.Sprintf("array element type mismatch: expected %s (from first element), got %s",
					firstType, elemType),
				line,
				column,
			)
		}
	}
}

// checkEnumDeclaration validates an enum declaration
func (tc *TypeChecker) checkEnumDeclaration(node *ast.EnumDeclaration) {
	// All enum members must have the same type
	// Determine the type from explicit values, or default to int for auto-assigned

	var firstType string
	var firstMemberName string

	// Track seen values to detect duplicates (#577)
	seenValues := make(map[string]string) // value string -> member name

	for _, member := range node.Values {
		if member.Value == nil {
			// No explicit value - will be auto-assigned as int
			if firstType == "" {
				firstType = "int"
				firstMemberName = member.Name.Value
			} else if firstType != "int" {
				tc.addError(
					errors.E3028,
					fmt.Sprintf("enum '%s' has mixed types: member '%s' is %s, but '%s' has no value (defaults to int)",
						node.Name.Value, firstMemberName, firstType, member.Name.Value),
					member.Name.Token.Line,
					member.Name.Token.Column,
				)
			}
			continue
		}

		// Determine the type of this member's value
		memberType := tc.getEnumValueType(member.Value)
		if memberType == "" {
			// Could not determine type - skip (parser should have caught invalid values)
			continue
		}

		if firstType == "" {
			// This is the first member with a determinable type
			firstType = memberType
			firstMemberName = member.Name.Value
		} else if memberType != firstType {
			// Type mismatch!
			tc.addError(
				errors.E3028,
				fmt.Sprintf("enum '%s' has mixed types: member '%s' is %s, but '%s' is %s",
					node.Name.Value, firstMemberName, firstType, member.Name.Value, memberType),
				member.Name.Token.Line,
				member.Name.Token.Column,
			)
		}

		// Check for duplicate values (#577)
		valueStr := tc.getEnumValueString(member.Value)
		if existingMember, exists := seenValues[valueStr]; exists {
			tc.addError(
				errors.E3033,
				fmt.Sprintf("enum '%s' has duplicate value: '%s' and '%s' both have value %s",
					node.Name.Value, existingMember, member.Name.Value, valueStr),
				member.Name.Token.Line,
				member.Name.Token.Column,
			)
		} else {
			seenValues[valueStr] = member.Name.Value
		}
	}
}

// getEnumValueType returns the type of an enum value expression
func (tc *TypeChecker) getEnumValueType(expr ast.Expression) string {
	switch expr.(type) {
	case *ast.IntegerValue:
		return "int"
	case *ast.FloatValue:
		return "float"
	case *ast.StringValue:
		return "string"
	case *ast.BooleanValue:
		return "bool"
	case *ast.CharValue:
		return "char"
	default:
		// For more complex expressions, we can't easily determine the type
		// This covers cases like enum values referencing other enums, etc.
		return ""
	}
}

// getEnumValueString returns a string representation of an enum value for duplicate detection
func (tc *TypeChecker) getEnumValueString(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.IntegerValue:
		return fmt.Sprintf("%d", e.Value)
	case *ast.FloatValue:
		return fmt.Sprintf("%v", e.Value)
	case *ast.StringValue:
		return fmt.Sprintf("\"%s\"", e.Value)
	case *ast.BooleanValue:
		return fmt.Sprintf("%v", e.Value)
	case *ast.CharValue:
		return fmt.Sprintf("'%c'", e.Value)
	default:
		return ""
	}
}

// checkGlobalVariableDeclaration validates a global variable declaration
func (tc *TypeChecker) checkGlobalVariableDeclaration(node *ast.VariableDeclaration) {
	// Check each variable in the declaration
	for _, name := range node.Names {
		varName := name.Value

		// Determine the type
		var typeName string
		if node.TypeName != "" {
			typeName = node.TypeName
		} else if node.Value != nil {
			// Type inference from value
			if inferredType, ok := tc.inferExpressionType(node.Value); ok {
				typeName = inferredType
			} else {
				continue
			}
		} else {
			continue
		}

		// Check if type exists (skip for inferred types that might be complex)
		if typeName != "" && !tc.TypeExists(typeName) && !strings.HasPrefix(typeName, "[") && !strings.HasPrefix(typeName, "map[") {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("undefined type '%s'", typeName),
				name.Token.Line,
				name.Token.Column,
			)
			continue
		}

		// Check if 'any' type is used (not allowed for user code)
		if typeName != "" && tc.containsAnyType(typeName) {
			tc.addError(
				errors.E3034,
				"'any' type cannot be used in variable declarations",
				name.Token.Line,
				name.Token.Column,
			)
			continue
		}

		// Register variable
		tc.variables[varName] = typeName

	}
}

// checkFunctionDeclaration validates a function's signature
func (tc *TypeChecker) checkFunctionDeclaration(node *ast.FunctionDeclaration) {
	sig := &FunctionSignature{
		Name:        node.Name.Value,
		Parameters:  []*Parameter{},
		ReturnTypes: node.ReturnTypes,
	}

	// Check parameter types and names
	for _, param := range node.Parameters {
		paramName := param.Name.Value

		// Check if parameter name shadows a user-defined type (struct/enum)
		if _, exists := tc.types[paramName]; exists {
			tc.addError(
				errors.E2033,
				fmt.Sprintf("'%s' is a type name and cannot be used as a parameter name", paramName),
				param.Name.Token.Line,
				param.Name.Token.Column,
			)
		}

		// Check if parameter name shadows a user-defined function
		if _, exists := tc.functions[paramName]; exists {
			tc.addError(
				errors.E2033,
				fmt.Sprintf("'%s' is a function name and cannot be used as a parameter name", paramName),
				param.Name.Token.Line,
				param.Name.Token.Column,
			)
		}

		if !tc.TypeExists(param.TypeName) {
			tc.addError(
				errors.E3010,
				fmt.Sprintf("undefined type '%s' for parameter '%s'", param.TypeName, param.Name.Value),
				param.Name.Token.Line,
				param.Name.Token.Column,
			)
		}

		// Check if 'any' type is used in parameter type (not allowed for user code)
		if tc.containsAnyType(param.TypeName) {
			tc.addError(
				errors.E3034,
				fmt.Sprintf("'any' type cannot be used as parameter type for '%s'", param.Name.Value),
				param.Name.Token.Line,
				param.Name.Token.Column,
			)
		}

		// Check if 'void' type is used in parameter type (not allowed)
		if param.TypeName == "void" {
			tc.addError(
				errors.E3038,
				fmt.Sprintf("'void' type cannot be used as parameter type for '%s'", param.Name.Value),
				param.Name.Token.Line,
				param.Name.Token.Column,
			)
		}

		// Check default value type matches parameter type (#582)
		if param.DefaultValue != nil {
			defaultType, ok := tc.inferExpressionType(param.DefaultValue)
			if ok && !tc.typesCompatible(param.TypeName, defaultType) {
				line, col := tc.getExpressionPosition(param.DefaultValue)
				tc.addError(
					errors.E3001,
					fmt.Sprintf("default value type mismatch: parameter '%s' expects %s, got %s",
						param.Name.Value, param.TypeName, defaultType),
					line,
					col,
				)
			}
		}

		sig.Parameters = append(sig.Parameters, &Parameter{
			Name:       param.Name.Value,
			Type:       param.TypeName,
			Mutable:    param.Mutable,
			HasDefault: param.DefaultValue != nil,
		})
	}

	// Check return types
	for _, returnType := range node.ReturnTypes {
		if !tc.TypeExists(returnType) {
			tc.addError(
				errors.E3011,
				fmt.Sprintf("undefined return type '%s' in function '%s'", returnType, node.Name.Value),
				node.Name.Token.Line,
				node.Name.Token.Column,
			)
		}
		// Check if 'any' type is used in return type (not allowed for user code)
		if tc.containsAnyType(returnType) {
			tc.addError(
				errors.E3034,
				fmt.Sprintf("'any' type cannot be used as return type in function '%s'", node.Name.Value),
				node.Name.Token.Line,
				node.Name.Token.Column,
			)
		}
		// Check if 'void' type is used as return type (not allowed - just omit return type for void functions)
		if returnType == "void" {
			tc.addError(
				errors.E3038,
				fmt.Sprintf("'void' cannot be used as return type in function '%s'; omit return type for void functions", node.Name.Value),
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

	// Track current function's attributes for #suppress checking
	prevAttrs := tc.currentFuncAttrs
	tc.currentFuncAttrs = node.Attributes
	defer func() { tc.currentFuncAttrs = prevAttrs }()

	// Add function parameters to scope with their mutability
	for _, param := range node.Parameters {
		tc.defineVariableWithMutability(param.Name.Value, param.TypeName, param.Mutable)
	}

	// Check if function body returns on all code paths (for functions with return types)
	if len(node.ReturnTypes) > 0 {
		if !tc.hasReturnStatement(node.Body) {
			// No return statement at all
			tc.addError(
				errors.E3024,
				fmt.Sprintf("Function '%s' declares return type(s) but has no return statement", node.Name.Value),
				node.Name.Token.Line,
				node.Name.Token.Column,
			)
		} else if !tc.allPathsReturn(node.Body) {
			// Has return statements, but not on all code paths
			tc.addError(
				errors.E3035,
				fmt.Sprintf("Function '%s' does not return a value on all code paths", node.Name.Value),
				node.Name.Token.Line,
				node.Name.Token.Column,
			)
		}
	}

	// Type check the function body
	tc.checkBlock(node.Body, node.ReturnTypes)
}

// checkMainFunction validates that a main() function exists as the program entry point
func (tc *TypeChecker) checkMainFunction() {
	if _, exists := tc.functions["main"]; !exists {
		tc.addError(
			errors.E4009,
			"program must define a main() function",
			1,
			1,
		)
	}
}

// markModuleUsed marks a module as being used in the code (#639)
func (tc *TypeChecker) markModuleUsed(moduleName string) {
	tc.usedModules[moduleName] = true
}

// checkUnusedImports warns about imported modules that are never used (#639)
func (tc *TypeChecker) checkUnusedImports() {
	for moduleName := range tc.modules {
		// Skip if module is used
		if tc.usedModules[moduleName] {
			continue
		}

		// Get import location for the warning
		loc, hasLoc := tc.importLocations[moduleName]
		if !hasLoc {
			// Should not happen, but be safe
			loc = ImportLocation{Line: 1, Column: 1}
		}

		tc.addWarning(
			errors.W1002,
			fmt.Sprintf("imported module '%s' is not used", moduleName),
			loc.Line,
			loc.Column,
		)
	}
}

// checkFileScopeStatements validates that only declarations are at file scope (#662)
// File scope should only allow: import, using, function declarations (do),
// type declarations (struct, enum), and variable declarations (const, temp).
// Control flow and executable statements should error.
func (tc *TypeChecker) checkFileScopeStatements(statements []ast.Statement) {
	for _, stmt := range statements {
		switch s := stmt.(type) {
		// These are allowed at file scope - do nothing
		case *ast.ImportStatement:
			// imports are allowed
		case *ast.UsingStatement:
			// using is allowed
		case *ast.FunctionDeclaration:
			// function declarations are allowed
		case *ast.StructDeclaration:
			// struct declarations are allowed
		case *ast.EnumDeclaration:
			// enum declarations are allowed
		case *ast.VariableDeclaration:
			// const/temp declarations are allowed at file scope
			// However, we might want to disallow mutable variable declarations
			// For now, allow both const and temp at file scope

		// Control flow statements - NOT allowed at file scope
		case *ast.IfStatement:
			tc.addError(
				errors.E2056,
				"'if' statement not allowed at file scope; move it inside a function",
				s.Token.Line,
				s.Token.Column,
			)
		case *ast.ForStatement:
			tc.addError(
				errors.E2056,
				"'for' statement not allowed at file scope; move it inside a function",
				s.Token.Line,
				s.Token.Column,
			)
		case *ast.ForEachStatement:
			tc.addError(
				errors.E2056,
				"'for_each' statement not allowed at file scope; move it inside a function",
				s.Token.Line,
				s.Token.Column,
			)
		case *ast.WhenStatement:
			tc.addError(
				errors.E2056,
				"'when' statement not allowed at file scope; move it inside a function",
				s.Token.Line,
				s.Token.Column,
			)
		case *ast.WhileStatement:
			tc.addError(
				errors.E2056,
				"'as_long_as' statement not allowed at file scope; move it inside a function",
				s.Token.Line,
				s.Token.Column,
			)
		case *ast.LoopStatement:
			tc.addError(
				errors.E2056,
				"'loop' statement not allowed at file scope; move it inside a function",
				s.Token.Line,
				s.Token.Column,
			)

		// Executable statements - NOT allowed at file scope
		case *ast.AssignmentStatement:
			tc.addError(
				errors.E2056,
				"assignment not allowed at file scope; move it inside a function",
				s.Token.Line,
				s.Token.Column,
			)
		case *ast.ExpressionStatement:
			tc.addError(
				errors.E2056,
				"expression statement not allowed at file scope; move it inside a function",
				s.Token.Line,
				s.Token.Column,
			)
		case *ast.ReturnStatement:
			tc.addError(
				errors.E2056,
				"'return' statement not allowed at file scope; can only be used inside a function",
				s.Token.Line,
				s.Token.Column,
			)
		case *ast.BreakStatement:
			tc.addError(
				errors.E2056,
				"'break' statement not allowed at file scope; can only be used inside a loop",
				s.Token.Line,
				s.Token.Column,
			)
		case *ast.ContinueStatement:
			tc.addError(
				errors.E2056,
				"'continue' statement not allowed at file scope; can only be used inside a loop",
				s.Token.Line,
				s.Token.Column,
			)
		case *ast.BlockStatement:
			tc.addError(
				errors.E2056,
				"block statement not allowed at file scope; move it inside a function",
				s.Token.Line,
				s.Token.Column,
			)
		case *ast.EnsureStatement:
			tc.addError(
				errors.E2056,
				"'ensure' statement not allowed at file scope; can only be used inside a function",
				s.Token.Line,
				s.Token.Column,
			)
		}
	}
}

// isSuppressed checks if a warning code is suppressed by function attributes or file-level #suppress
func (tc *TypeChecker) isSuppressed(warningCode string, attrs []*ast.Attribute) bool {
	// Check file-level suppressions first
	for _, code := range tc.fileSuppressWarnings {
		if code == "ALL" || code == warningCode {
			return true
		}
	}

	// Check function-level attributes
	if attrs == nil {
		return false
	}

	// Map warning codes to their alternate names
	alternateNames := map[string]string{
		"W3003": "array_size_mismatch",
	}

	for _, attr := range attrs {
		if attr.Name == "suppress" {
			for _, arg := range attr.Args {
				if arg == "ALL" || arg == warningCode {
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

// allPathsReturn checks if ALL code paths in a block return a value.
// This is stricter than hasReturnStatement which only checks if ANY path returns.
func (tc *TypeChecker) allPathsReturn(block *ast.BlockStatement) bool {
	if block == nil || len(block.Statements) == 0 {
		return false
	}

	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *ast.ReturnStatement:
			// Found a return at this level - this path returns
			return true

		case *ast.IfStatement:
			// For an if statement to guarantee a return on all paths:
			// 1. It must have an otherwise (else) clause
			// 2. Both the if branch AND the otherwise branch must all-paths-return
			if tc.ifAllPathsReturn(s) {
				return true
			}
			// If the if doesn't cover all paths, continue checking subsequent statements
		}
		// Loops (for, foreach, while) can't guarantee they execute,
		// so we can't count returns inside them as covering all paths.
		// Continue to next statement.
	}

	// Reached end of block without finding a guaranteed return
	return false
}

// ifAllPathsReturn checks if an if/or/otherwise chain returns on ALL paths
func (tc *TypeChecker) ifAllPathsReturn(ifStmt *ast.IfStatement) bool {
	// The consequence (if block) must return on all its paths
	if !tc.allPathsReturn(ifStmt.Consequence) {
		return false
	}

	// Must have an alternative (or/otherwise)
	if ifStmt.Alternative == nil {
		return false
	}

	// Check the alternative
	switch alt := ifStmt.Alternative.(type) {
	case *ast.IfStatement:
		// It's an "or" (else if) - recursively check
		return tc.ifAllPathsReturn(alt)
	case *ast.BlockStatement:
		// It's an "otherwise" (else) block
		return tc.allPathsReturn(alt)
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

	case *ast.WhenStatement:
		tc.checkWhenStatement(s, expectedReturnTypes)

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

	case *ast.BreakStatement:
		// Check that break is inside a loop (#603)
		if tc.loopDepth == 0 {
			tc.addError(
				errors.E5009,
				"break statement outside loop",
				s.Token.Line,
				s.Token.Column,
			)
		}

	case *ast.ContinueStatement:
		// Check that continue is inside a loop (#603)
		if tc.loopDepth == 0 {
			tc.addError(
				errors.E5009,
				"continue statement outside loop",
				s.Token.Line,
				s.Token.Column,
			)
		}

	case *ast.StructDeclaration:
		// Structs cannot be declared inside functions - must be at file level
		tc.addError(
			errors.E2053,
			fmt.Sprintf("struct '%s' cannot be declared inside a function; move it to file level", s.Name.Value),
			s.Token.Line,
			s.Token.Column,
		)

	case *ast.EnumDeclaration:
		// Enums cannot be declared inside functions - must be at file level
		tc.addError(
			errors.E2053,
			fmt.Sprintf("enum '%s' cannot be declared inside a function; move it to file level", s.Name.Value),
			s.Token.Line,
			s.Token.Column,
		)
	}
}

// checkVariableDeclaration validates a variable declaration (Phase 3)
func (tc *TypeChecker) checkVariableDeclaration(decl *ast.VariableDeclaration) {
	// Check if private is used inside a function (not allowed)
	if decl.Visibility == ast.VisibilityPrivate && tc.currentScope != nil {
		tc.addError(
			errors.E3037,
			"'private' modifier can only be used at module level, not inside functions",
			decl.Token.Line,
			decl.Token.Column,
		)
	}

	// Handle multiple names (for multi-return assignment)
	if len(decl.Names) > 1 {
		tc.checkMultiReturnDeclaration(decl)
		return
	}

	// Single variable declaration
	if decl.Name == nil {
		return
	}

	varName := decl.Name.Value
	declaredType := decl.TypeName

	// Check if 'void' type is used (not allowed for variables)
	if declaredType == "void" {
		tc.addError(
			errors.E3038,
			"'void' type cannot be used in variable declarations",
			decl.Name.Token.Line,
			decl.Name.Token.Column,
		)
		return
	}

	// Check if variable name shadows a type (enum/struct) - #571
	if t, exists := tc.types[varName]; exists && (t.Kind == EnumType || t.Kind == StructType) {
		kind := "enum"
		if t.Kind == StructType {
			kind = "struct"
		}
		tc.addError(
			errors.E4012,
			fmt.Sprintf("variable '%s' shadows %s type of the same name", varName, kind),
			decl.Name.Token.Line,
			decl.Name.Token.Column,
		)
	}

	// Check if variable name shadows a function - #572
	if _, exists := tc.functions[varName]; exists {
		tc.addError(
			errors.E4013,
			fmt.Sprintf("variable '%s' shadows function of the same name", varName),
			decl.Name.Token.Line,
			decl.Name.Token.Column,
		)
	}

	// Check if variable name shadows a global constant - #573
	if _, exists := tc.variables[varName]; exists && tc.currentScope != nil {
		// Only check if we're in a local scope (not global)
		// This catches local variables shadowing global constants
		tc.addWarning(
			errors.W2007,
			fmt.Sprintf("variable '%s' shadows global variable/constant of the same name", varName),
			decl.Name.Token.Line,
			decl.Name.Token.Column,
		)
	}

	// Check if variable name shadows an imported module - #579
	if _, exists := tc.modules[varName]; exists {
		tc.addError(
			errors.E4014,
			fmt.Sprintf("variable '%s' shadows imported module of the same name", varName),
			decl.Name.Token.Line,
			decl.Name.Token.Column,
		)
	}

	// Check if variable name shadows a function from a 'used' module - #616
	if shadowedModule := tc.getUsedModuleShadowingFunction(varName); shadowedModule != "" {
		tc.addError(
			errors.E4015,
			fmt.Sprintf("variable '%s' shadows function '%s.%s' from used module", varName, shadowedModule, varName),
			decl.Name.Token.Line,
			decl.Name.Token.Column,
		)
	}

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

	// Check if 'any' type is used (not allowed for user code)
	if declaredType != "" && tc.containsAnyType(declaredType) {
		tc.addError(
			errors.E3034,
			"'any' type cannot be used in variable declarations",
			decl.Name.Token.Line,
			decl.Name.Token.Column,
		)
		return
	}

	// Check if 'void' type is used (not allowed)
	if declaredType != "" && tc.containsVoidType(declaredType) {
		tc.addError(
			errors.E3038,
			"'void' type cannot be used in variable declarations",
			decl.Name.Token.Line,
			decl.Name.Token.Column,
		)
		return
	}

	// Check for float-based enum as map key (not allowed)
	if declaredType != "" && tc.isMapType(declaredType) {
		keyType := tc.extractMapKeyType(declaredType)
		if enumType, ok := tc.GetType(keyType); ok && enumType.Kind == EnumType {
			if enumType.EnumBaseType == "float" {
				tc.addError(
					errors.E3029,
					fmt.Sprintf("float-based enum '%s' cannot be used as map key", keyType),
					decl.Name.Token.Line,
					decl.Name.Token.Column,
				)
				return
			}
		}
	}

	// If there's an initial value, check type compatibility
	if decl.Value != nil {
		// Check for type/function used as value
		tc.checkValueExpression(decl.Value)

		// Validate the expression itself
		tc.checkExpression(decl.Value)

		// If no declared type, infer from value and register it
		if declaredType == "" {
			// Check if it's a multi-return function being assigned to a single variable
			// Must check BEFORE type inference since inferCallType returns first type for multi-return
			if tc.isMultiReturnCall(decl.Value) {
				tc.addError(
					errors.E3040,
					"cannot assign multi-return function result to single variable; use multiple variables or discard with _",
					decl.Name.Token.Line,
					decl.Name.Token.Column,
				)
				return
			}

			inferredType, ok := tc.inferExpressionType(decl.Value)
			if ok {
				if inferredType == "void" {
					tc.addError(
						errors.E3038,
						"cannot assign void function result to a variable",
						decl.Name.Token.Line,
						decl.Name.Token.Column,
					)
					return
				}
				if inferredType == "" {
					// Check if it's a multi-return function being assigned to a single variable
					if tc.isMultiReturnCall(decl.Value) {
						tc.addError(
							errors.E3040,
							"cannot assign multi-return function result to single variable; use multiple variables or discard with _",
							decl.Name.Token.Line,
							decl.Name.Token.Column,
						)
						return
					}
				}
				tc.defineVariableWithMutability(varName, inferredType, decl.Mutable)
			} else {
				// Still register with empty type so variable is in scope
				tc.defineVariableWithMutability(varName, "", decl.Mutable)
			}
			return
		}

		actualType, ok := tc.inferExpressionType(decl.Value)
		if ok {
			if actualType == "void" {
				tc.addError(
					errors.E3038,
					"cannot assign void function result to a variable",
					decl.Name.Token.Line,
					decl.Name.Token.Column,
				)
				return
			}

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

				if declaredType == "char" && actualType == "string" {
					sourceLine := errors.GetSourceLine(tc.source, decl.Name.Token.Line)

					mismatchError := errors.NewErrorWithHelp(
						errors.E3001,
						fmt.Sprintf("type mismatch: cannot assign %s to %s", actualType, declaredType),
						decl.Name.Token.File,
						decl.Name.Token.Line,
						decl.Name.Token.Column,
						sourceLine,
						fmt.Sprintf("use single quotes for char literals: '%s'", decl.Value.TokenLiteral()),
					)
					tc.errors.AddError(mismatchError)
				} else {
					tc.addError(
						errors.E3001,
						fmt.Sprintf("type mismatch: cannot assign %s to %s", actualType, declaredType),
						decl.Name.Token.Line,
						decl.Name.Token.Column,
					)
				}

				return
			}

			// Check byte value range for single byte declaration
			if declaredType == "byte" {
				if intLit, ok := decl.Value.(*ast.IntegerValue); ok {
					if intLit.Value.Sign() < 0 || intLit.Value.Cmp(big.NewInt(255)) > 0 {
						tc.addError(
							errors.E3025,
							fmt.Sprintf("byte value %s out of range: must be between 0 and 255", intLit.Value.String()),
							decl.Name.Token.Line,
							decl.Name.Token.Column,
						)
						return
					}
				}
				// Handle negative literals like -5 (parsed as prefix expression)
				if prefixExpr, ok := decl.Value.(*ast.PrefixExpression); ok {
					if prefixExpr.Operator == "-" {
						if intLit, ok := prefixExpr.Right.(*ast.IntegerValue); ok {
							tc.addError(
								errors.E3025,
								fmt.Sprintf("byte value -%s out of range: must be between 0 and 255", intLit.Value.String()),
								decl.Name.Token.Line,
								decl.Name.Token.Column,
							)
							return
						}
					}
				}
			}

			// Check sized integer type ranges (#666)
			if tc.isSizedIntegerType(declaredType) {
				tc.checkIntegerLiteralRange(decl.Value, declaredType, decl.Name.Token.Line, decl.Name.Token.Column)
			}

			// Check byte array element values
			if declaredType == "[byte]" || strings.HasPrefix(declaredType, "[byte,") {
				if arrLit, ok := decl.Value.(*ast.ArrayValue); ok {
					for i, elem := range arrLit.Elements {
						if intLit, ok := elem.(*ast.IntegerValue); ok {
							if intLit.Value.Sign() < 0 || intLit.Value.Cmp(big.NewInt(255)) > 0 {
								tc.addError(
									errors.E3026,
									fmt.Sprintf("byte array element [%d] value %s out of range: must be between 0 and 255", i, intLit.Value.String()),
									intLit.Token.Line,
									intLit.Token.Column,
								)
							}
						}
						// Handle negative literals like -5 (parsed as prefix expression)
						if prefixExpr, ok := elem.(*ast.PrefixExpression); ok {
							if prefixExpr.Operator == "-" {
								if intLit, ok := prefixExpr.Right.(*ast.IntegerValue); ok {
									tc.addError(
										errors.E3026,
										fmt.Sprintf("byte array element [%d] value -%s out of range: must be between 0 and 255", i, intLit.Value.String()),
										prefixExpr.Token.Line,
										prefixExpr.Token.Column,
									)
								}
							}
						}
					}
				}
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

	// Register variable in current scope with mutability (temp = mutable, const = immutable)
	if declaredType != "" {
		tc.defineVariableWithMutability(varName, declaredType, decl.Mutable)
	}
}

// isMultiReturnCall checks if an expression is a function call that returns multiple values
func (tc *TypeChecker) isMultiReturnCall(expr ast.Expression) bool {
	callExpr, ok := expr.(*ast.CallExpression)
	if !ok {
		return false
	}

	// Get the function name and module name (if applicable)
	var funcName string
	var moduleName string

	switch fn := callExpr.Function.(type) {
	case *ast.Label:
		funcName = fn.Value
	case *ast.MemberExpression:
		// Module.function call - get both module and function names
		funcName = fn.Member.Value
		if obj, ok := fn.Object.(*ast.Label); ok {
			moduleName = obj.Value
		}
	default:
		return false
	}

	// Look up the function signature
	if funcSig, exists := tc.functions[funcName]; exists {
		return len(funcSig.ReturnTypes) > 1
	}

	// Check if it's a module function with multiple return values
	if moduleName != "" {
		resolvedModuleName := tc.resolveStdlibModule(moduleName)
		if returnTypes := tc.getModuleMultiReturnTypes(resolvedModuleName, funcName, callExpr.Arguments); returnTypes != nil {
			return len(returnTypes) > 1
		}
	}

	// Check if it's a function from a user-defined module via 'using'
	for usingModuleName := range tc.fileUsingModules {
		if moduleFuncs, hasModule := tc.moduleFunctions[usingModuleName]; hasModule {
			if moduleSig, found := moduleFuncs[funcName]; found {
				return len(moduleSig.ReturnTypes) > 1
			}
		}
	}

	// Check if it's a builtin function with multiple return values
	if builtinReturnTypes := tc.getBuiltinMultiReturnTypes(funcName); builtinReturnTypes != nil {
		return len(builtinReturnTypes) > 1
	}

	return false
}

// checkMultiReturnDeclaration validates multi-return variable declarations
// e.g., temp x int, y string = getValues()
func (tc *TypeChecker) checkMultiReturnDeclaration(decl *ast.VariableDeclaration) {
	// Check for type/function used as value
	if decl.Value != nil {
		tc.checkValueExpression(decl.Value)
		tc.checkExpression(decl.Value)
	}

	// Get the function call to check return types (needed for both type checking and inference)
	callExpr, ok := decl.Value.(*ast.CallExpression)
	if !ok {
		// Value is not a function call - still register variables with unknown types
		for _, name := range decl.Names {
			if name != nil {
				tc.defineVariableWithMutability(name.Value, "", decl.Mutable)
			}
		}
		return
	}

	// Get the function name and module name (if applicable)
	var funcName string
	var moduleName string
	switch fn := callExpr.Function.(type) {
	case *ast.Label:
		funcName = fn.Value
	case *ast.MemberExpression:
		// Module.function call - get both module and function names
		funcName = fn.Member.Value
		if obj, ok := fn.Object.(*ast.Label); ok {
			moduleName = obj.Value
		}
	default:
		// Still register variables with unknown types
		for _, name := range decl.Names {
			if name != nil {
				tc.defineVariableWithMutability(name.Value, "", decl.Mutable)
			}
		}
		return
	}

	// Look up the function signature
	funcSig, exists := tc.functions[funcName]
	if !exists {
		// Check if it's a module function with multiple return values
		if moduleName != "" {
			resolvedModuleName := tc.resolveStdlibModule(moduleName)
			moduleReturnTypes := tc.getModuleMultiReturnTypes(resolvedModuleName, funcName, callExpr.Arguments)
			if moduleReturnTypes != nil {
				// Register variables with the correct module function return types
				for i, name := range decl.Names {
					if name != nil {
						inferredType := ""
						if i < len(moduleReturnTypes) {
							inferredType = moduleReturnTypes[i]
						}
						tc.defineVariableWithMutability(name.Value, inferredType, decl.Mutable)
					}
				}
				return
			}
		}

		// Check if it's a function from a user-defined module via 'using'
		for usingModuleName := range tc.fileUsingModules {
			if moduleFuncs, hasModule := tc.moduleFunctions[usingModuleName]; hasModule {
				if moduleSig, found := moduleFuncs[funcName]; found {
					funcSig = moduleSig
					exists = true
					break
				}
			}
		}
	}
	if !exists {
		// Check if it's a builtin function with multiple return values
		builtinReturnTypes := tc.getBuiltinMultiReturnTypes(funcName)
		if builtinReturnTypes != nil {
			// Register variables with the correct builtin return types
			for i, name := range decl.Names {
				if name != nil {
					inferredType := ""
					if i < len(builtinReturnTypes) {
						inferredType = builtinReturnTypes[i]
					}
					tc.defineVariableWithMutability(name.Value, inferredType, decl.Mutable)
				}
			}
			return
		}

		// Function not found - still register variables with unknown types
		// (undefined function error will be caught elsewhere)
		for _, name := range decl.Names {
			if name != nil {
				tc.defineVariableWithMutability(name.Value, "", decl.Mutable)
			}
		}
		return
	}

	// If no explicit types declared, infer from function return types and register
	if len(decl.TypeNames) == 0 {
		for i, name := range decl.Names {
			if name != nil {
				inferredType := ""
				if i < len(funcSig.ReturnTypes) {
					inferredType = funcSig.ReturnTypes[i]
				}
				tc.defineVariableWithMutability(name.Value, inferredType, decl.Mutable)
			}
		}
		return
	}

	// Check if 'any' type is used in any of the declared types (not allowed for user code)
	for i, declaredType := range decl.TypeNames {
		if tc.containsAnyType(declaredType) {
			line, col := 0, 0
			if i < len(decl.Names) && decl.Names[i] != nil {
				line = decl.Names[i].Token.Line
				col = decl.Names[i].Token.Column
			}
			tc.addError(
				errors.E3034,
				"'any' type cannot be used in variable declarations",
				line,
				col,
			)
			return
		}
	}

	// Check that the number of declared types matches the number of return types
	if len(decl.TypeNames) != len(funcSig.ReturnTypes) {
		// Count mismatch will be caught by E5012 at runtime
		return
	}

	// Check each declared type against the corresponding return type
	for i, declaredType := range decl.TypeNames {
		if i >= len(funcSig.ReturnTypes) {
			break
		}
		returnType := funcSig.ReturnTypes[i]

		if !tc.typesCompatible(declaredType, returnType) {
			// Get position and name from the variable at this index
			line, col := 0, 0
			varName := "<unknown>"
			if i < len(decl.Names) && decl.Names[i] != nil {
				varName = decl.Names[i].Value
				line = decl.Names[i].Token.Line
				col = decl.Names[i].Token.Column
			}
			tc.addError(
				errors.E3001,
				fmt.Sprintf("type mismatch in multi-return: variable '%s' declared as %s but function returns %s",
					varName, declaredType, returnType),
				line,
				col,
			)
		}
	}

	// Register variables in scope
	for i, name := range decl.Names {
		if name != nil && i < len(decl.TypeNames) {
			tc.defineVariableWithMutability(name.Value, decl.TypeNames[i], decl.Mutable)
		}
	}
}

// checkAssignment validates an assignment statement (Phase 3)
func (tc *TypeChecker) checkAssignment(assign *ast.AssignmentStatement) {
	// Also validate the value expression
	tc.checkExpression(assign.Value)

	// Check that the assignment target exists and is mutable
	if rootVar := tc.extractRootVariable(assign.Name); rootVar != "" {
		// First check if the variable exists (#665)
		_, varExists := tc.lookupVariable(rootVar)
		if !varExists {
			line, column := tc.getExpressionPosition(assign.Name)
			tc.addError(
				errors.E4001,
				fmt.Sprintf("undefined variable '%s'", rootVar),
				line,
				column,
			)
			return
		}

		// Check mutability - error if trying to modify an immutable variable
		isMutable, found := tc.isVariableMutable(rootVar)
		if found && !isMutable {
			line, column := tc.getExpressionPosition(assign.Name)
			// Check if this is a struct field assignment
			if _, isMember := assign.Name.(*ast.MemberExpression); isMember {
				tc.addError(
					errors.E5017,
					fmt.Sprintf("cannot modify field of immutable struct '%s' (declared as const)", rootVar),
					line,
					column,
				)
			} else {
				tc.addError(
					errors.E5016,
					fmt.Sprintf("cannot modify immutable variable '%s' (declared as const or as non-& parameter)", rootVar),
					line,
					column,
				)
			}
		}
	}

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

// extractRootVariable returns the root variable name from an expression
// For "x" returns "x", for "x.field" returns "x", for "arr[0]" returns "arr"
func (tc *TypeChecker) extractRootVariable(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.Label:
		return e.Value
	case *ast.MemberExpression:
		return tc.extractRootVariable(e.Object)
	case *ast.IndexExpression:
		return tc.extractRootVariable(e.Left)
	default:
		return ""
	}
}

// checkMemberAssignment validates struct field assignments
func (tc *TypeChecker) checkMemberAssignment(member *ast.MemberExpression, value ast.Expression) {
	// Get the object type
	objType, ok := tc.inferExpressionType(member.Object)
	if !ok || objType == "" {
		return
	}

	// Look up struct type (including module types for qualified names like "lib.Hero")
	structType, exists := tc.getStructTypeIncludingModules(objType)
	if !exists {
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

// checkMemberExpression validates member access expressions
func (tc *TypeChecker) checkMemberExpression(member *ast.MemberExpression) {
	// Get the object type
	objType, ok := tc.inferExpressionType(member.Object)
	if !ok {
		return
	}

	// Skip validation for unknown types (empty string)
	if objType == "" {
		return
	}

	// Skip module access - those are handled separately
	if _, isModule := tc.modules[objType]; isModule {
		return
	}
	if _, isUsedModule := tc.fileUsingModules[objType]; isUsedModule {
		return
	}

	// Warn about member access on error type which is commonly nil (#687)
	if objType == "error" || objType == "Error" {
		if !tc.isSuppressed("W2009", tc.currentFuncAttrs) {
			line, column := tc.getExpressionPosition(member.Object)
			tc.addWarning(
				errors.W2009,
				fmt.Sprintf("accessing member '%s' on error type which may be nil - consider checking for nil first", member.Member.Value),
				line,
				column,
			)
		}
	}

	// Warn about chained member access on nullable struct types (#689)
	// e.g., p.pos.x where p.pos is a struct that could be nil
	if _, isChained := member.Object.(*ast.MemberExpression); isChained {
		if tc.isNullableType(objType) && objType != "error" && objType != "Error" {
			line, column := tc.getExpressionPosition(member.Object)
			tc.addWarning(
				errors.W2010,
				fmt.Sprintf("accessing member '%s' on struct type '%s' which may be nil - consider checking for nil first", member.Member.Value, objType),
				line,
				column,
			)
		}
	}

	// Check if it's a struct type (including module types for qualified names like "lib.Hero")
	structType, exists := tc.getStructTypeIncludingModules(objType)
	if !exists {
		// Not a struct - member access is invalid
		line, column := tc.getExpressionPosition(member.Member)
		tc.addError(
			errors.E4011,
			fmt.Sprintf("cannot access member '%s' on type '%s' (not a struct)", member.Member.Value, objType),
			line,
			column,
		)
		return
	}

	// Check if the field exists
	if _, hasField := structType.Fields[member.Member.Value]; !hasField {
		line, column := tc.getExpressionPosition(member.Member)
		tc.addError(
			errors.E4003,
			fmt.Sprintf("struct '%s' has no field '%s'", objType, member.Member.Value),
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
			// Check if this is an undefined variable (simple identifier)
			if label, isLabel := val.(*ast.Label); isLabel {
				line, column := tc.getExpressionPosition(val)
				// Check if it's a type name being used as a value (common mistake)
				if _, isType := tc.types[label.Value]; isType {
					tc.addError(
						errors.E4001,
						fmt.Sprintf("cannot return type '%s' as a value; did you mean to return a variable?", label.Value),
						line,
						column,
					)
				} else if _, isFunc := tc.functions[label.Value]; !isFunc {
					// Not a type and not a function - truly undefined
					tc.addError(
						errors.E4001,
						fmt.Sprintf("undefined variable '%s'", label.Value),
						line,
						column,
					)
				}
			}
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

	// Check for bare identifiers that have no effect as statements
	// A bare function name or type name as a statement does nothing
	tc.checkValueExpression(exprStmt.Expression)

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
		allowsTypeArgs := tc.functionAllowsTypeArguments(e)
		for _, arg := range e.Arguments {
			if allowsTypeArgs {
				// Functions like json.decode accept type arguments
				tc.checkValueExpressionAllowTypes(arg) // Catch function used as argument (but allow types)
			} else {
				// Most functions should not accept types as arguments
				tc.checkValueExpression(arg) // Catch type/function used as argument
			}
			tc.checkExpression(arg)
		}

	case *ast.InfixExpression:
		tc.checkValueExpression(e.Left)  // Catch type/function in operator
		tc.checkValueExpression(e.Right) // Catch type/function in operator
		tc.checkInfixExpression(e)
		tc.checkExpression(e.Left)
		tc.checkExpression(e.Right)

	case *ast.PrefixExpression:
		tc.checkValueExpression(e.Right) // Catch type/function in operator
		tc.checkPrefixExpression(e)
		tc.checkExpression(e.Right)

	case *ast.IndexExpression:
		tc.checkIndexExpression(e)
		tc.checkExpression(e.Left)
		tc.checkExpression(e.Index)

	case *ast.MemberExpression:
		tc.checkExpression(e.Object)
		tc.checkMemberExpression(e)

	case *ast.ArrayValue:
		tc.checkArrayLiteral(e)

	case *ast.MapValue:
		// Track seen keys to detect duplicates (#641)
		seenKeys := make(map[string]int) // key string -> line number of first occurrence
		for _, pair := range e.Pairs {
			tc.checkExpression(pair.Key)
			tc.checkValueExpression(pair.Value) // Catch type/function used as map value
			tc.checkExpression(pair.Value)

			// Check for duplicate keys
			keyStr := tc.getEnumValueString(pair.Key) // Reuse enum helper for literal conversion
			if keyStr != "" {                         // Only check if we can get a string representation
				if firstLine, exists := seenKeys[keyStr]; exists {
					line, col := tc.getExpressionPosition(pair.Key)
					tc.addError(
						errors.E12006,
						fmt.Sprintf("duplicate key %s in map literal (first defined on line %d)", keyStr, firstLine),
						line,
						col,
					)
				} else {
					line, _ := tc.getExpressionPosition(pair.Key)
					seenKeys[keyStr] = line
				}
			}
		}

	case *ast.StructValue:
		tc.checkStructLiteral(e)

	case *ast.RangeExpression:
		tc.checkRangeExpression(e)

	case *ast.CastExpression:
		tc.checkCastExpression(e)

	case *ast.PostfixExpression:
		tc.checkExpression(e.Left)
		tc.checkPostfixExpression(e)

	case *ast.InterpolatedString:
		// Check all embedded expressions in the interpolated string (#684)
		for _, part := range e.Parts {
			// Skip string literal parts - only check embedded expressions
			if _, isString := part.(*ast.StringValue); !isString {
				tc.checkValueExpression(part) // Catch type/function used as interpolation value
				tc.checkExpression(part)
			}
		}

	case *ast.Label:
		// Check if the identifier is known (variable, function, type, enum, etc.)
		if !tc.isKnownIdentifier(e.Value) {
			line, col := tc.getExpressionPosition(e)
			tc.addError(
				errors.E4001,
				fmt.Sprintf("undefined variable '%s'", e.Value),
				line,
				col,
			)
		}
	}
}

// checkRangeExpression validates range bounds and argument types (#597)
func (tc *TypeChecker) checkRangeExpression(rangeExpr *ast.RangeExpression) {
	// Validate argument types - all must be integers
	if rangeExpr.Start != nil {
		startType, ok := tc.inferExpressionType(rangeExpr.Start)
		if ok && !tc.isIntegerType(startType) {
			line, col := tc.getExpressionPosition(rangeExpr.Start)
			tc.addError(
				errors.E3001,
				fmt.Sprintf("range() start must be integer, got %s", startType),
				line, col,
			)
		}
	}

	if rangeExpr.End != nil {
		endType, ok := tc.inferExpressionType(rangeExpr.End)
		if ok && !tc.isIntegerType(endType) {
			line, col := tc.getExpressionPosition(rangeExpr.End)
			tc.addError(
				errors.E3001,
				fmt.Sprintf("range() end must be integer, got %s", endType),
				line, col,
			)
		}
	}

	if rangeExpr.Step != nil {
		stepType, ok := tc.inferExpressionType(rangeExpr.Step)
		if ok && !tc.isIntegerType(stepType) {
			line, col := tc.getExpressionPosition(rangeExpr.Step)
			tc.addError(
				errors.E3001,
				fmt.Sprintf("range() step must be integer, got %s", stepType),
				line, col,
			)
		}
	}

	// Check if both start and end are integer literals
	// If so, verify start <= end
	if rangeExpr.Start == nil {
		// range(end) form, start defaults to 0
		// Also check subexpressions
		tc.checkExpression(rangeExpr.End)
		if rangeExpr.Step != nil {
			tc.checkExpression(rangeExpr.Step)
		}
		return
	}

	startInt, startOk := rangeExpr.Start.(*ast.IntegerValue)
	endInt, endOk := rangeExpr.End.(*ast.IntegerValue)

	if startOk && endOk {
		// Both are literals, we can check at compile time
		if startInt.Value.Cmp(endInt.Value) > 0 {
			tc.addError(
				errors.E9005,
				fmt.Sprintf("invalid range: start (%s) must be less than or equal to end (%s)",
					startInt.Value.String(), endInt.Value.String()),
				rangeExpr.Token.Line,
				rangeExpr.Token.Column,
			)
		}
	}

	// Also check subexpressions
	tc.checkExpression(rangeExpr.Start)
	tc.checkExpression(rangeExpr.End)
	if rangeExpr.Step != nil {
		tc.checkExpression(rangeExpr.Step)
	}
}

// checkCastExpression validates cast(value, type) expressions
func (tc *TypeChecker) checkCastExpression(castExpr *ast.CastExpression) {
	// Check the value expression
	tc.checkExpression(castExpr.Value)

	// Validate that the target type is a valid type
	targetType := castExpr.TargetType
	if castExpr.IsArray {
		// For array types like [u8], validate the element type
		if !tc.isValidCastTargetType(castExpr.ElementType) {
			tc.addError(
				errors.E3001,
				fmt.Sprintf("invalid cast target type: [%s]", castExpr.ElementType),
				castExpr.Token.Line,
				castExpr.Token.Column,
			)
			return
		}
	} else {
		// For simple types like u8, validate directly
		if !tc.isValidCastTargetType(targetType) {
			tc.addError(
				errors.E3001,
				fmt.Sprintf("invalid cast target type: %s", targetType),
				castExpr.Token.Line,
				castExpr.Token.Column,
			)
			return
		}
	}

	// Infer the source type and check if the conversion is valid
	sourceType, ok := tc.inferExpressionType(castExpr.Value)
	if !ok || sourceType == "" {
		return // Can't validate without knowing source type
	}

	// Check if the conversion is valid
	if castExpr.IsArray {
		// For array casts, source must be an array
		if !tc.isArrayType(sourceType) {
			tc.addError(
				errors.E3001,
				fmt.Sprintf("cannot cast non-array type %s to array type %s", sourceType, targetType),
				castExpr.Token.Line,
				castExpr.Token.Column,
			)
			return
		}

		// Get source element type and check if element conversion is valid
		sourceElemType := tc.extractArrayElementType(sourceType)
		if sourceElemType != "" && !tc.isValidCastConversion(sourceElemType, castExpr.ElementType) {
			tc.addWarning(
				errors.W2004,
				fmt.Sprintf("cast from [%s] to [%s] may fail at runtime", sourceElemType, castExpr.ElementType),
				castExpr.Token.Line,
				castExpr.Token.Column,
			)
		}
	} else {
		// For single value casts, check if conversion is valid
		if !tc.isValidCastConversion(sourceType, targetType) {
			tc.addWarning(
				errors.W2004,
				fmt.Sprintf("cast from %s to %s may fail at runtime", sourceType, targetType),
				castExpr.Token.Line,
				castExpr.Token.Column,
			)
		}
	}
}

// isValidCastTargetType checks if a type name is valid as a cast target
func (tc *TypeChecker) isValidCastTargetType(typeName string) bool {
	switch typeName {
	case "int", "i8", "i16", "i32", "i64", "i128", "i256",
		"uint", "u8", "u16", "u32", "u64", "u128", "u256",
		"float", "f32", "f64",
		"byte", "char", "string", "bool":
		return true
	default:
		return false
	}
}

// isValidCastConversion checks if a conversion from source to target type is valid
func (tc *TypeChecker) isValidCastConversion(source, target string) bool {
	// Any type can be cast to string (via Inspect)
	if target == "string" {
		return true
	}

	// Numeric types can be cast to other numeric types
	if tc.isNumericType(source) && tc.isNumericType(target) {
		return true
	}

	// char can be cast to int/numeric and vice versa
	if source == "char" && tc.isNumericType(target) {
		return true
	}
	if tc.isNumericType(source) && target == "char" {
		return true
	}

	// byte can be cast to other numeric types
	if source == "byte" && tc.isNumericType(target) {
		return true
	}
	if tc.isNumericType(source) && target == "byte" {
		return true
	}

	// string can be cast to numeric types (parsing)
	if source == "string" && tc.isNumericType(target) {
		return true
	}

	// bool conversions
	if target == "bool" {
		return source == "bool" || source == "int" || source == "string"
	}
	if source == "bool" && target == "int" {
		return true
	}

	return false
}

// checkPostfixExpression validates postfix operators (++ and --)
// These operators modify the operand, so we must check mutability and type (#598)
func (tc *TypeChecker) checkPostfixExpression(postfix *ast.PostfixExpression) {
	// Check that operand is an integer type
	operandType, ok := tc.inferExpressionType(postfix.Left)
	if ok && !tc.isIntegerType(operandType) {
		line, column := tc.getExpressionPosition(postfix.Left)
		tc.addError(
			errors.E3001,
			fmt.Sprintf("postfix operator %s requires integer operand, got %s", postfix.Operator, operandType),
			line,
			column,
		)
	}

	// Check mutability - error if trying to modify an immutable variable
	if rootVar := tc.extractRootVariable(postfix.Left); rootVar != "" {
		isMutable, found := tc.isVariableMutable(rootVar)
		if found && !isMutable {
			line, column := tc.getExpressionPosition(postfix.Left)
			tc.addError(
				errors.E5016,
				fmt.Sprintf("cannot modify immutable variable '%s' (declared as const or as non-& parameter)", rootVar),
				line,
				column,
			)
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

	// If either type is unknown (empty string), skip type checking
	// This can happen with stdlib multi-return functions where we don't have signatures
	if leftType == "" || rightType == "" {
		return
	}

	line, column := tc.getExpressionPosition(infix.Left)

	switch infix.Operator {
	case "+":
		// Valid for numbers or strings
		if tc.isNumericType(leftType) && tc.isNumericType(rightType) {
			// Check for potential overflow with literal values (#686)
			if tc.isIntegerType(leftType) && tc.isIntegerType(rightType) && leftType == rightType {
				leftVal, leftLit := tc.getLiteralIntValue(infix.Left)
				rightVal, rightLit := tc.getLiteralIntValue(infix.Right)
				if leftLit && rightLit {
					if overflows, msg := tc.checkArithmeticOverflow(leftVal, rightVal, "+", leftType); overflows {
						tc.addWarning(errors.W2008, msg, line, column)
					}
				}
			}
			// Warn about implicit type conversion when mixing byte with larger types
			if (leftType == "byte" && rightType != "byte" && tc.isNumericType(rightType)) ||
				(rightType == "byte" && leftType != "byte" && tc.isNumericType(leftType)) {
				tc.addWarning(
					errors.W2004,
					fmt.Sprintf("implicit type conversion: byte promoted to %s in arithmetic operation", tc.getPromotedType(leftType, rightType)),
					line,
					column,
				)
			}
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

	case "-", "*", "/":
		// Only valid for numbers
		if !tc.isNumericType(leftType) || !tc.isNumericType(rightType) {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("invalid operands for '%s': %s and %s (expected numeric)", infix.Operator, leftType, rightType),
				line,
				column,
			)
		} else {
			// Check for division by literal zero (#667)
			if infix.Operator == "/" && tc.isLiteralZero(infix.Right) {
				tc.addError(
					errors.E5001,
					"division by zero",
					line,
					column,
				)
			}
			// Check for potential overflow with literal values (#686)
			if infix.Operator != "/" && tc.isIntegerType(leftType) && tc.isIntegerType(rightType) && leftType == rightType {
				leftVal, leftLit := tc.getLiteralIntValue(infix.Left)
				rightVal, rightLit := tc.getLiteralIntValue(infix.Right)
				if leftLit && rightLit {
					if overflows, msg := tc.checkArithmeticOverflow(leftVal, rightVal, infix.Operator, leftType); overflows {
						tc.addWarning(errors.W2008, msg, line, column)
					}
				}
			}
			// Warn about implicit type conversion when mixing byte with larger types
			if (leftType == "byte" && rightType != "byte" && tc.isNumericType(rightType)) ||
				(rightType == "byte" && leftType != "byte" && tc.isNumericType(leftType)) {
				tc.addWarning(
					errors.W2004,
					fmt.Sprintf("implicit type conversion: byte promoted to %s in arithmetic operation", tc.getPromotedType(leftType, rightType)),
					line,
					column,
				)
			}
		}

	case "%":
		// Modulo only valid for integers (#601)
		if !tc.isIntegerType(leftType) || !tc.isIntegerType(rightType) {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("invalid operands for '%%': %s and %s (expected integer)", leftType, rightType),
				line,
				column,
			)
		} else {
			// Check for modulo by literal zero (#667)
			if tc.isLiteralZero(infix.Right) {
				tc.addError(
					errors.E5002,
					"modulo by zero",
					line,
					column,
				)
			}
		}

	case "==", "!=":
		// Check for comparing different enum types (#576)
		leftIsEnum := tc.isEnumType(leftType)
		rightIsEnum := tc.isEnumType(rightType)
		if leftIsEnum && rightIsEnum && leftType != rightType {
			tc.addError(
				errors.E3032,
				fmt.Sprintf("cannot compare different enum types: %s and %s", leftType, rightType),
				line,
				column,
			)
			return
		}

		// Allow int/float comparison like other comparison operators (#857)
		leftNumeric := tc.isNumericType(leftType)
		rightNumeric := tc.isNumericType(rightType)
		if leftNumeric && rightNumeric {
			// Both are numeric types - allow comparison
			return
		}

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
		// Check for string comparison explicitly to give helpful message (#887)
		if leftType == "string" && rightType == "string" {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("cannot use '%s' on strings; use strings.compare() instead", infix.Operator),
				line,
				column,
			)
			return
		}

		// Only valid for numbers, chars, or enums with numeric base type
		leftComparable := tc.isNumericType(leftType) || leftType == "char" || tc.isComparableEnumType(leftType)
		rightComparable := tc.isNumericType(rightType) || rightType == "char" || tc.isComparableEnumType(rightType)
		if !leftComparable {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("invalid operand for '%s': %s (expected numeric)", infix.Operator, leftType),
				line,
				column,
			)
		}
		if !rightComparable {
			tc.addError(
				errors.E3002,
				fmt.Sprintf("invalid operand for '%s': %s (expected numeric)", infix.Operator, rightType),
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
		// Validate that index type matches the map's declared key type (#608)
		if indexOk {
			mapKeyType := tc.extractMapKeyType(leftType)
			if mapKeyType != "" && !tc.typesCompatible(mapKeyType, indexType) {
				line, column := tc.getExpressionPosition(index.Index)
				tc.addError(
					errors.E3003,
					fmt.Sprintf("map key type mismatch: expected %s, got %s", mapKeyType, indexType),
					line,
					column,
				)
			}
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

	// Check array bounds for literal indices (#685)
	if leftOk && tc.isArrayType(leftType) {
		// Try to get the index as a literal value
		indexValue, isLiteral := tc.getLiteralIntValue(index.Index)
		if isLiteral {
			// Check for negative index
			if indexValue < 0 {
				line, column := tc.getExpressionPosition(index.Index)
				tc.addError(
					errors.E5003,
					fmt.Sprintf("array index %d is negative", indexValue),
					line,
					column,
				)
			} else {
				// Check against fixed array size if known
				arraySize := tc.extractArraySize(leftType)
				if arraySize > 0 && indexValue >= int64(arraySize) {
					line, column := tc.getExpressionPosition(index.Index)
					tc.addError(
						errors.E5003,
						fmt.Sprintf("array index %d out of bounds for array of size %d", indexValue, arraySize),
						line,
						column,
					)
				}
			}
		}
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
		// Check if this is a variable being called as a function (#602)
		if _, isVar := tc.lookupVariable(funcName); isVar {
			// It's a variable, not a function - error
			if _, isFunc := tc.functions[funcName]; !isFunc {
				line, column := tc.getExpressionPosition(call.Function)
				tc.addError(
					errors.E3015,
					fmt.Sprintf("cannot call non-function value '%s'", funcName),
					line,
					column,
				)
				return
			}
		}
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
		// Check if this function is from the same module (multi-file module support)
		if tc.currentModuleName != "" {
			if moduleFuncs, hasModule := tc.moduleFunctions[tc.currentModuleName]; hasModule {
				if moduleSig, found := moduleFuncs[funcName]; found {
					sig = moduleSig
					ok = true
				}
			}
		}
	}
	if !ok {
		// Check if this function is from a user-defined module via 'using'
		for moduleName := range tc.fileUsingModules {
			if moduleFuncs, hasModule := tc.moduleFunctions[moduleName]; hasModule {
				if moduleSig, found := moduleFuncs[funcName]; found {
					sig = moduleSig
					ok = true
					break
				}
			}
		}
	}
	if !ok {
		// Check if this function might be from a 'using' imported module
		if tc.checkDirectStdlibCall(funcName, call) {
			return
		}
		// Function not found anywhere - report error
		line, column := tc.getExpressionPosition(call.Function)
		tc.addError(
			errors.E4002,
			fmt.Sprintf("undefined function '%s'", funcName),
			line,
			column,
		)
		return
	}

	// Calculate minimum required arguments (parameters without defaults)
	minRequired := 0
	for _, param := range sig.Parameters {
		if !param.HasDefault {
			minRequired++
		}
	}

	// Check argument count
	if len(call.Arguments) < minRequired || len(call.Arguments) > len(sig.Parameters) {
		line, column := tc.getExpressionPosition(call.Function)
		var msg string
		if minRequired == len(sig.Parameters) {
			msg = fmt.Sprintf("wrong number of arguments to '%s': expected %d, got %d",
				funcName, len(sig.Parameters), len(call.Arguments))
		} else {
			msg = fmt.Sprintf("wrong number of arguments to '%s': expected %d to %d, got %d",
				funcName, minRequired, len(sig.Parameters), len(call.Arguments))
		}
		tc.addError(errors.E5008, msg, line, column)
		return
	}

	// Check argument types and mutability
	for i, arg := range call.Arguments {
		actualType, ok := tc.inferExpressionType(arg)
		if !ok {
			// Check if this is an undefined variable
			if label, isLabel := arg.(*ast.Label); isLabel {
				// Check if it's not a known function or type
				if _, isFunc := tc.functions[label.Value]; !isFunc {
					if _, isType := tc.types[label.Value]; !isType {
						line, column := tc.getExpressionPosition(arg)
						tc.addError(
							errors.E4001,
							fmt.Sprintf("undefined variable '%s'", label.Value),
							line,
							column,
						)
					}
				}
			}
			continue
		}

		expectedType := sig.Parameters[i].Type
		if !tc.typesCompatible(expectedType, actualType) {
			line, column := tc.getExpressionPosition(arg)
			displayType := actualType
			if displayType == "" {
				displayType = "<unknown type>"
			}
			tc.addError(
				errors.E3001,
				fmt.Sprintf("argument type mismatch in call to '%s': parameter '%s' expects %s, got %s",
					funcName, sig.Parameters[i].Name, expectedType, displayType),
				line,
				column,
			)
		}

		// Check for const -> & param error (E3023)
		// If the parameter is mutable (&), the argument must be a mutable variable
		if sig.Parameters[i].Mutable {
			// Check if argument is a simple variable (Label)
			if label, isLabel := arg.(*ast.Label); isLabel {
				// Check if this variable is mutable in scope
				isMutable, found := tc.isVariableMutable(label.Value)
				if found && !isMutable {
					line, column := tc.getExpressionPosition(arg)
					tc.addError(
						errors.E3027,
						fmt.Sprintf("cannot pass immutable variable '%s' to mutable parameter '&%s' in call to '%s'",
							label.Value, sig.Parameters[i].Name, funcName),
						line,
						column,
					)
				}
			}
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
				"cannot convert string to int at build-time (value may not be numeric)",
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
				"cannot convert string to float at build-time (value may not be numeric)",
				line,
				column,
			)
		}
		return true

	case "string", "bool", "char", "byte",
		"i8", "i16", "i32", "i64", "i128", "i256",
		"u8", "u16", "u32", "u64", "u128", "u256",
		"f32", "f64":
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
		// Skip type check if type is unknown (empty string)
		if ok && argType != "" && argType != "string" && !tc.isArrayType(argType) && !tc.isMapType(argType) {
			line, column := tc.getExpressionPosition(call.Arguments[0])
			tc.addError(errors.E3001,
				fmt.Sprintf("len() argument must be string, array, or map, got %s", argType),
				line, column)
		}
		return true

	case "typeof":
		// typeof() requires exactly 1 argument (any type except void)
		if len(call.Arguments) != 1 {
			line, column := tc.getExpressionPosition(call.Function)
			tc.addError(errors.E5008,
				fmt.Sprintf("typeof() requires exactly 1 argument, got %d", len(call.Arguments)),
				line, column)
		} else {
			// Check if argument is a void function call
			argType, ok := tc.inferExpressionType(call.Arguments[0])
			if ok && argType == "void" {
				line, column := tc.getExpressionPosition(call.Arguments[0])
				tc.addError(errors.E3038,
					"cannot use void function result as argument to typeof()",
					line, column)
			}
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

	case "copy":
		// copy() requires exactly 1 argument
		if len(call.Arguments) != 1 {
			line, column := tc.getExpressionPosition(call.Function)
			tc.addError(errors.E5008,
				fmt.Sprintf("copy() requires exactly 1 argument, got %d", len(call.Arguments)),
				line, column)
		}
		return true

	case "error":
		// error() requires exactly 1 argument (the error message)
		if len(call.Arguments) != 1 {
			line, column := tc.getExpressionPosition(call.Function)
			tc.addError(errors.E5008,
				fmt.Sprintf("error() requires exactly 1 argument, got %d", len(call.Arguments)),
				line, column)
		}
		return true

	case "append":
		// append() requires at least 2 arguments (array, value)
		if len(call.Arguments) < 2 {
			line, column := tc.getExpressionPosition(call.Function)
			tc.addError(errors.E5008,
				fmt.Sprintf("append() requires at least 2 arguments, got %d", len(call.Arguments)),
				line, column)
		}
		return true

	case "new":
		// new() requires exactly 1 argument (the type)
		if len(call.Arguments) != 1 {
			line, column := tc.getExpressionPosition(call.Function)
			tc.addError(errors.E5008,
				fmt.Sprintf("new() requires exactly 1 argument, got %d", len(call.Arguments)),
				line, column)
		}
		return true

	case "ref":
		// ref() requires exactly 1 argument
		if len(call.Arguments) != 1 {
			line, column := tc.getExpressionPosition(call.Function)
			tc.addError(errors.E5008,
				fmt.Sprintf("ref() requires exactly 1 argument, got %d", len(call.Arguments)),
				line, column)
		}
		return true

	case "range":
		// range() requires 2 or 3 arguments
		if len(call.Arguments) < 2 || len(call.Arguments) > 3 {
			line, column := tc.getExpressionPosition(call.Function)
			tc.addError(errors.E5008,
				fmt.Sprintf("range() requires 2 or 3 arguments, got %d", len(call.Arguments)),
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
	// Check for type/function used as condition
	tc.checkValueExpression(ifStmt.Condition)
	tc.checkExpression(ifStmt.Condition)

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

// checkWhenStatement validates a when/is/default statement
func (tc *TypeChecker) checkWhenStatement(whenStmt *ast.WhenStatement, expectedReturnTypes []string) {
	// Validate the when value expression (check for field access errors, etc.)
	tc.checkExpression(whenStmt.Value)

	// Infer the type of the value being matched
	valueType, ok := tc.inferExpressionType(whenStmt.Value)
	if !ok {
		// Type inference failed - check if the condition is a type name instead of a value
		if label, isLabel := whenStmt.Value.(*ast.Label); isLabel {
			// Check if this is a locally defined type (enum or struct)
			// This catches cases like `when COLOR { ... }` where COLOR is an enum type
			if _, isLocalType := tc.types[label.Value]; isLocalType {
				tc.addError(
					errors.E2047,
					fmt.Sprintf("when condition must be a value, not a type name '%s'", label.Value),
					whenStmt.Token.Line,
					whenStmt.Token.Column,
				)
			}
		}
		return
	}

	// Check that value type is allowed
	// Disallowed: bool, nil, arrays, maps
	if valueType == "bool" {
		tc.addError(
			errors.E2048,
			"when condition cannot be a boolean; use if/or/otherwise instead",
			whenStmt.Token.Line,
			whenStmt.Token.Column,
		)
		return
	}

	if valueType == "nil" {
		tc.addError(
			errors.E2049,
			"when condition cannot be nil; use if/otherwise to check for nil",
			whenStmt.Token.Line,
			whenStmt.Token.Column,
		)
		return
	}

	// Check for array or map types
	if tc.isArrayType(valueType) || tc.isMapType(valueType) {
		tc.addError(
			errors.E2050,
			fmt.Sprintf("when condition cannot be an array or map (got %s)", valueType),
			whenStmt.Token.Line,
			whenStmt.Token.Column,
		)
		return
	}

	// Track seen case values for duplicate detection
	seenCases := make(map[string]bool)

	// Check if this is an enum type for #strict validation
	enumTypeInfo, isEnumType := tc.GetType(valueType)
	if isEnumType && enumTypeInfo != nil && enumTypeInfo.Kind != EnumType {
		isEnumType = false
	}

	// Validate #strict is only used with enums
	if whenStmt.IsStrict && !isEnumType {
		tc.addError(
			errors.E2045,
			"#strict attribute only allowed on enum when statements",
			whenStmt.Token.Line,
			whenStmt.Token.Column,
		)
	}

	// Track handled enum cases for #strict exhaustiveness check
	handledEnumCases := make(map[string]bool)

	// Check each case
	for _, whenCase := range whenStmt.Cases {
		for _, caseValue := range whenCase.Values {
			// Check the case value expression (validates range bounds, etc.)
			tc.checkExpression(caseValue)

			// For #strict enum when statements, only allow enum member expressions
			if whenStmt.IsStrict && isEnumType {
				if !tc.isEnumMemberExpression(caseValue, valueType, enumTypeInfo) {
					line, col := tc.getExpressionPosition(caseValue)
					tc.addError(
						errors.E2054,
						"#strict when requires explicit enum member values, got non-enum expression",
						line,
						col,
					)
				} else {
					// Track this enum member as handled for exhaustiveness check
					if memberName := tc.getEnumMemberName(caseValue); memberName != "" {
						handledEnumCases[memberName] = true
					}
				}
			}

			// Skip range expressions for duplicate detection
			if _, isRange := caseValue.(*ast.RangeExpression); isRange {
				continue
			}

			// Get a string representation of the case value for duplicate detection
			caseKey := tc.getCaseValueKey(caseValue)
			if caseKey != "" {
				if seenCases[caseKey] {
					line, col := tc.getExpressionPosition(caseValue)
					tc.addError(
						errors.E2043,
						fmt.Sprintf("duplicate case value: %s", caseKey),
						line,
						col,
					)
				}
				seenCases[caseKey] = true
			}

			// Type check the case value matches the when value type
			caseType, ok := tc.inferExpressionType(caseValue)
			if ok && caseType != valueType && caseType != "unknown" && valueType != "unknown" {
				// Allow enum type matching
				if !strings.HasPrefix(caseType, valueType) && !strings.HasSuffix(caseType, "."+valueType) {
					// Allow int case values when matching against enum types (enums have int underlying values)
					if !(isEnumType && caseType == "int") {
						line, col := tc.getExpressionPosition(caseValue)
						tc.addError(
							errors.E3001,
							fmt.Sprintf("case value type %s does not match when value type %s", caseType, valueType),
							line,
							col,
						)
					}
				}
			}
		}

		// Check the case body
		tc.enterScope()
		tc.checkBlock(whenCase.Body, expectedReturnTypes)
		tc.exitScope()
	}

	// #strict enum exhaustiveness check - ensure all enum cases are handled
	if whenStmt.IsStrict && isEnumType && enumTypeInfo != nil && enumTypeInfo.EnumMembers != nil {
		var missingCases []string
		for enumMember := range enumTypeInfo.EnumMembers {
			if !handledEnumCases[enumMember] {
				missingCases = append(missingCases, enumMember)
			}
		}
		if len(missingCases) > 0 {
			// Sort for consistent error messages
			sort.Strings(missingCases)
			tc.addError(
				errors.E2046,
				fmt.Sprintf("#strict when statement missing enum cases: %s", strings.Join(missingCases, ", ")),
				whenStmt.Token.Line,
				whenStmt.Token.Column,
			)
		}
	}

	// Check the default block if present
	if whenStmt.Default != nil {
		tc.enterScope()
		tc.checkBlock(whenStmt.Default, expectedReturnTypes)
		tc.exitScope()
	}
}

// getEnumMemberName extracts the enum member name from a case value expression
// Returns the member name (e.g., "RED" from "Color.RED") or empty string if not an enum member
func (tc *TypeChecker) getEnumMemberName(expr ast.Expression) string {
	switch v := expr.(type) {
	case *ast.MemberExpression:
		// Handle EnumType.MEMBER pattern - return just the member name
		return v.Member.Value
	case *ast.Label:
		// Handle bare enum value (e.g., just RED)
		return v.Value
	}
	return ""
}

// isEnumMemberExpression checks if an expression is a valid enum member reference
// for use in #strict when statements. Valid forms are:
// - EnumType.MEMBER (e.g., Color.RED)
// - MEMBER (if it's a known enum value of the expected type)
func (tc *TypeChecker) isEnumMemberExpression(expr ast.Expression, enumTypeName string, enumTypeInfo *Type) bool {
	switch v := expr.(type) {
	case *ast.MemberExpression:
		// Check for EnumType.MEMBER pattern
		if obj, ok := v.Object.(*ast.Label); ok {
			// Get the base type name (strip module prefix if present)
			baseEnumName := enumTypeName
			if idx := strings.LastIndex(enumTypeName, "."); idx != -1 {
				baseEnumName = enumTypeName[idx+1:]
			}
			// Check if object matches the enum type name
			if obj.Value == baseEnumName || obj.Value == enumTypeName {
				// Verify the member is a valid enum value
				if enumTypeInfo != nil && enumTypeInfo.EnumMembers != nil {
					if enumTypeInfo.EnumMembers[v.Member.Value] {
						return true
					}
				}
			}
			// Also check module-prefixed enum types in moduleTypes
			for moduleName, moduleTypes := range tc.moduleTypes {
				if enumType, exists := moduleTypes[obj.Value]; exists && enumType.Kind == EnumType {
					if enumType.EnumMembers != nil && enumType.EnumMembers[v.Member.Value] {
						// Verify this is the right enum type
						fullTypeName := moduleName + "." + obj.Value
						if fullTypeName == enumTypeName || obj.Value == baseEnumName {
							return true
						}
					}
				}
			}
		}
	case *ast.Label:
		// Check if it's a bare enum value (e.g., just RED instead of Color.RED)
		if enumTypeInfo != nil && enumTypeInfo.EnumMembers != nil {
			if enumTypeInfo.EnumMembers[v.Value] {
				return true
			}
		}
	}
	return false
}

// getCaseValueKey returns a string key for a case value for duplicate detection
func (tc *TypeChecker) getCaseValueKey(expr ast.Expression) string {
	switch v := expr.(type) {
	case *ast.IntegerValue:
		return v.Value.String()
	case *ast.StringValue:
		return "\"" + v.Value + "\""
	case *ast.CharValue:
		return "'" + string(v.Value) + "'"
	case *ast.Label:
		return v.Value
	case *ast.MemberExpression:
		// Handle Enum.MEMBER
		if obj, ok := v.Object.(*ast.Label); ok {
			return obj.Value + "." + v.Member.Value
		}
	}
	return ""
}

// checkForStatement validates a for loop
func (tc *TypeChecker) checkForStatement(forStmt *ast.ForStatement, expectedReturnTypes []string) {
	tc.enterScope()
	tc.loopDepth++ // Track loop nesting for break/continue validation (#603)

	// Check the iterable expression (e.g., range())
	if forStmt.Iterable != nil {
		tc.checkExpression(forStmt.Iterable)
	}

	// Add loop variable to scope
	if forStmt.Variable != nil {
		varName := forStmt.Variable.Value

		// Check if this loop variable shadows an outer loop variable (#114)
		if tc.currentScope != nil && tc.currentScope.IsLoopVariableInOuterScope(varName) {
			line := forStmt.Variable.Token.Line
			column := forStmt.Variable.Token.Column
			tc.addError(
				errors.E4016,
				fmt.Sprintf("loop variable '%s' shadows outer loop variable", varName),
				line,
				column,
			)
		}

		varType := forStmt.VarType
		if varType == "" {
			varType = "int" // Default for range iteration
		}
		tc.defineVariable(varName, varType)

		// Mark as loop variable for shadowing detection in nested loops
		if tc.currentScope != nil {
			tc.currentScope.MarkAsLoopVariable(varName)
		}
	}

	tc.checkBlock(forStmt.Body, expectedReturnTypes)
	tc.loopDepth--
	tc.exitScope()
}

// checkForEachStatement validates a for_each loop
func (tc *TypeChecker) checkForEachStatement(forEach *ast.ForEachStatement, expectedReturnTypes []string) {
	tc.enterScope()
	tc.loopDepth++ // Track loop nesting for break/continue validation (#603)

	// Infer element type from collection and validate it's iterable (#595)
	if forEach.Variable != nil && forEach.Collection != nil {
		varName := forEach.Variable.Value

		// Check if this loop variable shadows an outer loop variable (#114)
		if tc.currentScope != nil && tc.currentScope.IsLoopVariableInOuterScope(varName) {
			line := forEach.Variable.Token.Line
			column := forEach.Variable.Token.Column
			tc.addError(
				errors.E4016,
				fmt.Sprintf("loop variable '%s' shadows outer loop variable", varName),
				line,
				column,
			)
		}

		collType, ok := tc.inferExpressionType(forEach.Collection)
		if ok {
			// Determine if the collection is mutable by checking the root variable
			// If iterating over a mutable variable's field (e.g., h.inventory where h is &),
			// the loop variable should also be mutable
			collectionMutable := false
			if rootVar := tc.extractRootVariable(forEach.Collection); rootVar != "" {
				if isMutable, found := tc.isVariableMutable(rootVar); found {
					collectionMutable = isMutable
				}
			}

			// For arrays, element type is inside []
			if tc.isArrayType(collType) {
				elemType := tc.extractArrayElementType(collType)
				// Loop variable inherits mutability from collection
				tc.defineVariableWithMutability(varName, elemType, collectionMutable)
			} else if collType == "string" {
				// Iterating over string gives char
				tc.defineVariableWithMutability(varName, "char", collectionMutable)
			} else {
				// Not an iterable type - produce error
				line, column := tc.getExpressionPosition(forEach.Collection)
				tc.addError(
					errors.E3017,
					fmt.Sprintf("for_each requires array or string, got %s", collType),
					line,
					column,
				)
			}
		}

		// Mark as loop variable for shadowing detection in nested loops
		if tc.currentScope != nil {
			tc.currentScope.MarkAsLoopVariable(varName)
		}
	}

	tc.checkBlock(forEach.Body, expectedReturnTypes)
	tc.loopDepth--
	tc.exitScope()
}

// checkWhileStatement validates an as_long_as loop
func (tc *TypeChecker) checkWhileStatement(whileStmt *ast.WhileStatement, expectedReturnTypes []string) {
	// Validate the condition expression (check for field access errors, etc.)
	tc.checkExpression(whileStmt.Condition)

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
	tc.loopDepth++ // Track loop nesting for break/continue validation (#603)
	tc.checkBlock(whileStmt.Body, expectedReturnTypes)
	tc.loopDepth--
	tc.exitScope()
}

// checkLoopStatement validates a loop statement
func (tc *TypeChecker) checkLoopStatement(loopStmt *ast.LoopStatement, expectedReturnTypes []string) {
	tc.enterScope()
	tc.loopDepth++ // Track loop nesting for break/continue validation (#603)
	tc.checkBlock(loopStmt.Body, expectedReturnTypes)
	tc.loopDepth--
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

// isSizedIntegerType checks if a type is a sized integer type (#666)
func (tc *TypeChecker) isSizedIntegerType(typeName string) bool {
	switch typeName {
	case "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64":
		return true
	}
	return false
}

// isLiteralZero checks if an expression is a literal zero value (#667)
func (tc *TypeChecker) isLiteralZero(expr ast.Expression) bool {
	// Check for integer literal 0
	if intLit, ok := expr.(*ast.IntegerValue); ok {
		return intLit.Value.Sign() == 0
	}
	// Check for float literal 0.0
	if floatLit, ok := expr.(*ast.FloatValue); ok {
		return floatLit.Value == 0.0
	}
	return false
}

// getIntegerTypeRange returns the min and max values for a sized integer type (#666)
func (tc *TypeChecker) getIntegerTypeRange(typeName string) (min, max *big.Int) {
	switch typeName {
	case "i8":
		return big.NewInt(-128), big.NewInt(127)
	case "i16":
		return big.NewInt(-32768), big.NewInt(32767)
	case "i32":
		return big.NewInt(-2147483648), big.NewInt(2147483647)
	case "i64":
		min = new(big.Int)
		max = new(big.Int)
		min.SetString("-9223372036854775808", 10)
		max.SetString("9223372036854775807", 10)
		return min, max
	case "u8":
		return big.NewInt(0), big.NewInt(255)
	case "u16":
		return big.NewInt(0), big.NewInt(65535)
	case "u32":
		return big.NewInt(0), big.NewInt(4294967295)
	case "u64":
		min = big.NewInt(0)
		max = new(big.Int)
		max.SetString("18446744073709551615", 10)
		return min, max
	}
	return nil, nil
}

// checkIntegerLiteralRange validates that an integer literal fits within the target type's range (#666)
// Extended in #686 to also handle simple arithmetic expressions with literal operands
func (tc *TypeChecker) checkIntegerLiteralRange(expr ast.Expression, targetType string, line, column int) {
	min, max := tc.getIntegerTypeRange(targetType)
	if min == nil || max == nil {
		return
	}

	var value *big.Int

	// Check for direct integer literal
	if intLit, ok := expr.(*ast.IntegerValue); ok {
		value = intLit.Value
	}

	// Check for negative literal (prefix expression with -)
	if prefixExpr, ok := expr.(*ast.PrefixExpression); ok {
		if prefixExpr.Operator == "-" {
			if intLit, ok := prefixExpr.Right.(*ast.IntegerValue); ok {
				value = new(big.Int).Neg(intLit.Value)
			}
		}
	}

	// Check for simple arithmetic expressions with literal operands (#686)
	if infixExpr, ok := expr.(*ast.InfixExpression); ok {
		leftVal, leftOk := tc.getLiteralIntValue(infixExpr.Left)
		rightVal, rightOk := tc.getLiteralIntValue(infixExpr.Right)
		if leftOk && rightOk {
			var result int64
			switch infixExpr.Operator {
			case "+":
				result = leftVal + rightVal
			case "-":
				result = leftVal - rightVal
			case "*":
				result = leftVal * rightVal
			case "/":
				if rightVal != 0 {
					result = leftVal / rightVal
				}
			default:
				// Unsupported operator, skip
				return
			}
			value = big.NewInt(result)
		}
	}

	if value == nil {
		return // Not a literal or computable expression, skip range check
	}

	// Check if value is within range
	if value.Cmp(min) < 0 || value.Cmp(max) > 0 {
		tc.addError(
			errors.E3036,
			fmt.Sprintf("value %s out of range for type %s (valid range: %s to %s)", value.String(), targetType, min.String(), max.String()),
			line,
			column,
		)
	}
}

// containsAnyType checks if a type string is or contains the 'any' type
// This catches: "any", "[any]", "map[string:any]", etc.
func (tc *TypeChecker) containsAnyType(typeName string) bool {
	if typeName == "any" {
		return true
	}
	// Check array element type
	if tc.isArrayType(typeName) {
		elemType := typeName[1 : len(typeName)-1]
		return tc.containsAnyType(elemType)
	}
	// Check map key and value types
	if tc.isMapType(typeName) {
		keyType := tc.extractMapKeyType(typeName)
		valueType := tc.extractMapValueType(typeName)
		return tc.containsAnyType(keyType) || tc.containsAnyType(valueType)
	}
	return false
}

// containsVoidType checks if a type string is or contains the 'void' type
// This catches: "void", "[void]", "map[string:void]", etc.
func (tc *TypeChecker) containsVoidType(typeName string) bool {
	if typeName == "void" {
		return true
	}
	// Check array element type
	if tc.isArrayType(typeName) {
		elemType := typeName[1 : len(typeName)-1]
		return tc.containsVoidType(elemType)
	}
	// Check map key and value types
	if tc.isMapType(typeName) {
		keyType := tc.extractMapKeyType(typeName)
		valueType := tc.extractMapValueType(typeName)
		return tc.containsVoidType(keyType) || tc.containsVoidType(valueType)
	}
	return false
}

// extractMapKeyType extracts the key type from a map type string
// For "map[Status:string]" returns "Status"
func (tc *TypeChecker) extractMapKeyType(mapType string) string {
	if !tc.isMapType(mapType) {
		return ""
	}
	// Extract "Status:string" from "map[Status:string]"
	inner := mapType[4 : len(mapType)-1]
	keyType, _, ok := tc.splitMapInnerType(inner)
	if !ok {
		return ""
	}
	return keyType
}

// extractMapValueType extracts the value type from a map type string
// For "map[string:int]" returns "int"
func (tc *TypeChecker) extractMapValueType(mapType string) string {
	if !tc.isMapType(mapType) {
		return ""
	}
	// Extract "string:int" from "map[string:int]"
	inner := mapType[4 : len(mapType)-1]
	_, valueType, ok := tc.splitMapInnerType(inner)
	if !ok {
		return ""
	}
	return valueType
}

// splitMapInnerType splits "key:value" while respecting nested map/array types.
func (tc *TypeChecker) splitMapInnerType(inner string) (string, string, bool) {
	depth := 0
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 {
				return inner[:i], inner[i+1:], true
			}
		}
	}
	return "", "", false
}

// isNullableType checks if a type can accept nil values
// error type and user-defined struct types can be nil in EZ
// Arrays, maps, and primitives cannot be nil
func (tc *TypeChecker) isNullableType(typeName string) bool {
	// error type can be nil (for error handling pattern)
	if typeName == "error" {
		return true
	}
	// User-defined struct types can be nil
	if t, exists := tc.types[typeName]; exists && t.Kind == StructType {
		return true
	}
	// Everything else (arrays, maps, primitives, enums) cannot be nil
	return false
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

// getLiteralIntValue extracts the integer value from a literal expression
// Returns the value and true if the expression is a literal integer (including negative)
// Returns 0 and false if the expression is not a literal
func (tc *TypeChecker) getLiteralIntValue(expr ast.Expression) (int64, bool) {
	switch e := expr.(type) {
	case *ast.IntegerValue:
		return e.Value.Int64(), true
	case *ast.PrefixExpression:
		// Handle negative literals like -5
		if e.Operator == "-" {
			if intVal, ok := e.Right.(*ast.IntegerValue); ok {
				return -intVal.Value.Int64(), true
			}
		}
	}
	return 0, false
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
	case *ast.NilValue:
		return e.Token.Line, e.Token.Column
	case *ast.MapValue:
		return e.Token.Line, e.Token.Column
	case *ast.StructValue:
		return e.Token.Line, e.Token.Column
	case *ast.NewExpression:
		return e.Token.Line, e.Token.Column
	case *ast.PostfixExpression:
		return e.Token.Line, e.Token.Column
	case *ast.RangeExpression:
		return e.Token.Line, e.Token.Column
	case *ast.CastExpression:
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

// defineVariable adds a variable to the current scope (defaults to immutable)
func (tc *TypeChecker) defineVariable(name, typeName string) {
	if tc.currentScope != nil {
		tc.currentScope.Define(name, typeName)
	}
}

// defineVariableWithMutability adds a variable to the current scope with explicit mutability
func (tc *TypeChecker) defineVariableWithMutability(name, typeName string, mutable bool) {
	if tc.currentScope != nil {
		tc.currentScope.DefineWithMutability(name, typeName, mutable)
	}
}

// isVariableMutable checks if a variable is mutable in the current scope
func (tc *TypeChecker) isVariableMutable(name string) (bool, bool) {
	if tc.currentScope != nil {
		return tc.currentScope.IsMutable(name)
	}
	return false, false
}

// lookupVariable finds a variable type in scope chain, global variables, or same-module variables
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
	// Then check same-module variables (multi-file module support)
	if tc.currentModuleName != "" {
		if moduleVars, hasModule := tc.moduleVariables[tc.currentModuleName]; hasModule {
			if typeName, found := moduleVars[name]; found {
				return typeName, true
			}
		}
	}
	// Finally check variables from user-defined modules via 'using'
	for moduleName := range tc.fileUsingModules {
		if moduleVars, hasModule := tc.moduleVariables[moduleName]; hasModule {
			if typeName, found := moduleVars[name]; found {
				return typeName, true
			}
		}
	}
	return "", false
}

// isBuiltinFunction returns true if the name is a builtin function
func (tc *TypeChecker) isBuiltinFunction(name string) bool {
	builtins := map[string]bool{
		// Type conversions
		"int": true, "float": true, "string": true, "bool": true, "char": true,
		"i8": true, "i16": true, "i32": true, "i64": true,
		"u8": true, "u16": true, "u32": true, "u64": true,
		"i128": true, "i256": true, "u128": true, "u256": true,
		"f32": true, "f64": true,
		"byte": true,
		// Core builtins
		"len": true, "typeof": true, "input": true, "copy": true, "error": true,
		"append": true, "new": true, "ref": true, "range": true,
		// Global builtins (no import needed)
		"exit": true, "panic": true, "assert": true,
	}
	return builtins[name]
}

// isBuiltinConstant returns true if the name is a builtin constant
func (tc *TypeChecker) isBuiltinConstant(name string) bool {
	constants := map[string]bool{
		"EXIT_SUCCESS": true,
		"EXIT_FAILURE": true,
		"nil":          true,
		"true":         true,
		"false":        true,
	}
	return constants[name]
}

// isStdlibFunction returns true if the name is a stdlib function (accessible via using)
func (tc *TypeChecker) isStdlibFunction(moduleName, funcName string) bool {
	stdFuncs := map[string]map[string]bool{
		"std": {
			"println": true, "printf": true, "print": true,
			"eprintln": true, "eprintf": true, "eprint": true,
			"sleep_milliseconds": true, "sleep_seconds": true, "sleep_nanoseconds": true,
			"read_int": true, "read_float": true, "read_string": true,
		},
	}
	if modFuncs, ok := stdFuncs[moduleName]; ok {
		return modFuncs[funcName]
	}
	return false
}

// isStdlibConstant returns true if the name is a stdlib constant (accessible via using)
func (tc *TypeChecker) isStdlibConstant(moduleName, constName string) bool {
	stdConsts := map[string]map[string]bool{
		"std": {
			"EXIT_SUCCESS": true, "EXIT_FAILURE": true,
		},
		"db": {
			"ALPHA": true, "ALPHA_DESC": true, "VALUE_ALPHA": true,
			"VALUE_ALPHA_DESC": true, "KEY_LEN": true, "KEY_LEN_DESC": true,
			"VALUE_LEN": true, "VALUE_LEN_DESC": true, "NUMERIC": true,
			"NUMERIC_DESC": true,
		},
		"http": {
			"OK": true, "CREATED": true, "ACCEPTED": true,
			"NO_CONTENT": true, "MOVED_PERMANENTLY": true, "FOUND": true,
			"NOT_MODIFIED": true, "TEMPORARY_REDIRECT": true, "PERMANENT_REDIRECT": true,
			"BAD_REQUEST": true, "UNAUTHORIZED": true, "PAYMENT_REQUIRED": true,
			"FORBIDDEN": true, "NOT_FOUND": true, "METHOD_NOT_ALLOWED": true,
			"CONFLICT": true, "INTERNAL_SERVER_ERROR": true, "BAD_GATEWAY": true,
			"SERVICE_UNAVAILABLE": true,
		},
		"io": {
			"READ_ONLY": true, "WRITE_ONLY": true, "READ_WRITE": true,
			"APPEND": true, "CREATE": true, "TRUNCATE": true,
			"EXCLUSIVE": true, "SEEK_START": true, "SEEK_CURRENT": true,
			"SEEK_END": true,
		},
		"math": {
			"PI": true, "E": true, "PHI": true,
			"SQRT2": true, "LN2": true, "LN10": true,
			"TAU": true, "INF": true, "NEG_INF": true,
		},
		"os": {
			"MAC_OS": true, "LINUX": true, "WINDOWS": true,
			"CURRENT_OS": true, "line_separator": true, "dev_null": true,
		},
		"time": {
			"SUNDAY": true, "MONDAY": true, "TUESDAY": true,
			"WEDNESDAY": true, "THURSDAY": true, "FRIDAY": true,
			"SATURDAY": true, "JANUARY": true, "FEBRUARY": true,
			"MARCH": true, "APRIL": true, "MAY": true,
			"JUNE": true, "JULY": true, "AUGUST": true,
			"SEPTEMBER": true, "OCTOBER": true, "NOVEMBER": true,
			"DECEMBER": true, "SECOND": true, "MINUTE": true,
			"HOUR": true, "DAY": true, "WEEK": true,
		},
		"uuid": {
			"NIL": true,
		},
	}
	if modConsts, ok := stdConsts[moduleName]; ok {
		return modConsts[constName]
	}
	return false
}

// isKnownIdentifier checks if a name is a known identifier (variable, function, type, module, etc.)
func (tc *TypeChecker) isKnownIdentifier(name string) bool {
	// Check if it's a variable
	if _, ok := tc.lookupVariable(name); ok {
		return true
	}
	// Check if it's a function
	if _, ok := tc.functions[name]; ok {
		return true
	}
	// Check if it's a type (struct/enum)
	if _, ok := tc.types[name]; ok {
		return true
	}
	// Check if it's an enum value (check all enum types for this member)
	for _, t := range tc.types {
		if t.Kind == EnumType && t.EnumMembers != nil {
			if t.EnumMembers[name] {
				return true
			}
		}
	}
	// Check if it's a builtin function
	if tc.isBuiltinFunction(name) {
		return true
	}
	// Check if it's a builtin constant (EXIT_SUCCESS, etc.)
	if tc.isBuiltinConstant(name) {
		return true
	}
	// Check if it's an imported module (e.g., math, strings, arrays)
	if tc.modules[name] {
		return true
	}
	// Check if it's a using module
	if tc.hasUsingModule(name) {
		return true
	}
	// Check if it's a stdlib function accessible via 'using'
	for moduleName := range tc.fileUsingModules {
		if tc.isStdlibFunction(moduleName, name) {
			return true
		}
	}
	if tc.currentScope != nil {
		for moduleName := range tc.currentScope.usingModules {
			if tc.isStdlibFunction(moduleName, name) {
				return true
			}
		}
	}

	// Check if it's a stdlib constant accessible via 'using'
	for moduleName := range tc.fileUsingModules {
		if tc.isStdlibConstant(moduleName, name) {
			return true
		}
	}
	if tc.currentScope != nil {
		for moduleName := range tc.currentScope.usingModules {
			if tc.isStdlibConstant(moduleName, name) {
				return true
			}
		}
	}

	// Check if it's a function from a user module accessible via 'using' (#671)
	for moduleName := range tc.fileUsingModules {
		if funcs, ok := tc.moduleFunctions[moduleName]; ok {
			if _, exists := funcs[name]; exists {
				return true
			}
		}
	}
	if tc.currentScope != nil {
		for moduleName := range tc.currentScope.usingModules {
			if funcs, ok := tc.moduleFunctions[moduleName]; ok {
				if _, exists := funcs[name]; exists {
					return true
				}
			}
		}
	}
	// Check if it's a type from a user module accessible via 'using' (#671)
	for moduleName := range tc.fileUsingModules {
		if types, ok := tc.moduleTypes[moduleName]; ok {
			if _, exists := types[name]; exists {
				return true
			}
		}
	}
	if tc.currentScope != nil {
		for moduleName := range tc.currentScope.usingModules {
			if types, ok := tc.moduleTypes[moduleName]; ok {
				if _, exists := types[name]; exists {
					return true
				}
			}
		}
	}
	// Check if it's a variable/constant from a user module accessible via 'using' (#677)
	for moduleName := range tc.fileUsingModules {
		if vars, ok := tc.moduleVariables[moduleName]; ok {
			if _, exists := vars[name]; exists {
				return true
			}
		}
	}
	if tc.currentScope != nil {
		for moduleName := range tc.currentScope.usingModules {
			if vars, ok := tc.moduleVariables[moduleName]; ok {
				if _, exists := vars[name]; exists {
					return true
				}
			}
		}
	}
	return false
}

// functionAllowsTypeArguments returns true if the function accepts type arguments.
// Some functions like json.decode need a type parameter to know what to decode into.
func (tc *TypeChecker) functionAllowsTypeArguments(call *ast.CallExpression) bool {
	// Check for module.function pattern (e.g., json.decode)
	if member, ok := call.Function.(*ast.MemberExpression); ok {
		if obj, ok := member.Object.(*ast.Label); ok {
			funcName := obj.Value + "." + member.Member.Value
			switch funcName {
			case "json.decode":
				return true
			}
		}
	}
	return false
}

// checkValueExpressionAllowTypes validates that an expression is not a function
// name being used as a value, but allows struct/enum types as values.
// This is used for function arguments where types can be passed (e.g., json.decode).
func (tc *TypeChecker) checkValueExpressionAllowTypes(expr ast.Expression) bool {
	label, isLabel := expr.(*ast.Label)
	if !isLabel {
		return false
	}

	// Allow struct/enum types as values when passed to functions
	if _, isType := tc.types[label.Value]; isType {
		return false // Types are allowed as function arguments
	}

	// Check if this label refers to a function (not being called)
	if _, isFunc := tc.functions[label.Value]; isFunc {
		line, column := tc.getExpressionPosition(expr)
		tc.addError(
			errors.E3031,
			fmt.Sprintf("function '%s' cannot be used as a value - functions must be called with ()",
				label.Value),
			line,
			column,
		)
		return true
	}

	return false
}

// checkValueExpression validates that an expression is not a type name or function
// name being used as a value. Returns true if an error was reported.
// This catches bugs like copy(StatusEnum) or copy(helperFunc).
func (tc *TypeChecker) checkValueExpression(expr ast.Expression) bool {
	label, isLabel := expr.(*ast.Label)
	if !isLabel {
		return false
	}

	// Check if this label refers to a type (enum or struct)
	if t, isType := tc.types[label.Value]; isType {
		// Only error for user-defined types (enum/struct), not primitives
		if t.Kind == EnumType {
			line, column := tc.getExpressionPosition(expr)
			tc.addError(
				errors.E3030,
				fmt.Sprintf("enum type '%s' cannot be used as a value - use a specific enum member like %s.MEMBER",
					label.Value, label.Value),
				line,
				column,
			)
			return true
		} else if t.Kind == StructType {
			line, column := tc.getExpressionPosition(expr)
			tc.addError(
				errors.E3030,
				fmt.Sprintf("struct type '%s' cannot be used as a value - create an instance with %s { field: value }",
					label.Value, label.Value),
				line,
				column,
			)
			return true
		}
	}

	// Check if this label refers to a function (not being called)
	if _, isFunc := tc.functions[label.Value]; isFunc {
		line, column := tc.getExpressionPosition(expr)
		tc.addError(
			errors.E3031,
			fmt.Sprintf("function '%s' cannot be used as a value - functions must be called with ()",
				label.Value),
			line,
			column,
		)
		return true
	}

	return false
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
		if varType, ok := tc.lookupVariable(e.Value); ok {
			return varType, true
		}
		// Check module variables accessible via 'using' (#677)
		for moduleName := range tc.fileUsingModules {
			if varType, ok := tc.GetModuleVariable(moduleName, e.Value); ok {
				return varType, true
			}
		}
		if tc.currentScope != nil {
			for moduleName := range tc.currentScope.usingModules {
				if varType, ok := tc.GetModuleVariable(moduleName, e.Value); ok {
					return varType, true
				}
			}
		}
		return "", false

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

	case *ast.CastExpression:
		// cast(value, type) returns the target type
		return e.TargetType, true

	case *ast.InterpolatedString:
		return "string", true

	case *ast.BlankIdentifier:
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
		// Check if this function is from the same module (multi-file module support)
		if tc.currentModuleName != "" {
			if moduleFuncs, hasModule := tc.moduleFunctions[tc.currentModuleName]; hasModule {
				if moduleSig, found := moduleFuncs[fn.Value]; found {
					if len(moduleSig.ReturnTypes) == 1 {
						return moduleSig.ReturnTypes[0], true
					} else if len(moduleSig.ReturnTypes) > 1 {
						return moduleSig.ReturnTypes[0], true
					}
					return "void", true
				}
			}
		}
		// Check if this function is from a user-defined module via file-level 'using'
		for moduleName := range tc.fileUsingModules {
			if moduleFuncs, hasModule := tc.moduleFunctions[moduleName]; hasModule {
				if moduleSig, found := moduleFuncs[fn.Value]; found {
					if len(moduleSig.ReturnTypes) == 1 {
						return moduleSig.ReturnTypes[0], true
					} else if len(moduleSig.ReturnTypes) > 1 {
						return moduleSig.ReturnTypes[0], true
					}
					return "void", true
				}
			}
		}
		// Check if this function is from a user-defined module via scope-level 'using'
		if tc.currentScope != nil {
			for moduleName := range tc.currentScope.usingModules {
				if moduleFuncs, hasModule := tc.moduleFunctions[moduleName]; hasModule {
					if moduleSig, found := moduleFuncs[fn.Value]; found {
						if len(moduleSig.ReturnTypes) == 1 {
							return moduleSig.ReturnTypes[0], true
						} else if len(moduleSig.ReturnTypes) > 1 {
							return moduleSig.ReturnTypes[0], true
						}
						return "void", true
					}
				}
			}
		}
		if inferredType, ok := tc.inferDirectStdlibCallType(fn.Value, call.Arguments); ok {
			return inferredType, true
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

// getBuiltinMultiReturnTypes returns the return types for built-in functions that return multiple values
// Returns nil if the function is not a known multi-return builtin
func (tc *TypeChecker) getBuiltinMultiReturnTypes(name string) []string {
	switch name {
	case "read_int":
		return []string{"int", "error"}
	default:
		return nil
	}
}

// getModuleMultiReturnTypes returns the return types for stdlib module functions that return multiple values
// Returns nil if the function is not a known multi-return module function
// The args parameter allows type inference based on function arguments (e.g., json.decode with a type hint)
func (tc *TypeChecker) getModuleMultiReturnTypes(moduleName, funcName string, args []ast.Expression) []string {
	switch moduleName {
	case "io":
		switch funcName {
		case "read_file", "read_lines", "read_bytes":
			// io.read_file returns (string, error)
			// io.read_lines returns ([string], error)
			// io.read_bytes returns ([byte], error)
			switch funcName {
			case "read_file":
				return []string{"string", "error"}
			case "read_lines":
				return []string{"[string]", "error"}
			case "read_bytes":
				return []string{"[byte]", "error"}
			}
		case "write_file", "append_file", "write_bytes":
			return []string{"bool", "error"}
		case "create_dir", "remove", "remove_dir", "rename", "copy_file":
			return []string{"bool", "error"}
		// Note: exists, is_dir, is_file return single bool (not tuple)
		case "file_size":
			return []string{"int", "error"}
		case "list_dir", "read_dir":
			return []string{"[string]", "error"}
		case "read_stdin":
			return []string{"string", "error"}
		case "open", "create":
			return []string{"File", "error"}
		case "fread", "fread_line", "fread_all":
			return []string{"string", "error"}
		case "fread_bytes":
			return []string{"[byte]", "error"}
		case "fwrite", "fwrite_line", "fwrite_bytes":
			return []string{"int", "error"}
		case "fseek", "ftell":
			return []string{"int", "error"}
		case "fclose", "fflush", "ftruncate":
			return []string{"bool", "error"}
		case "feof":
			return []string{"bool", "error"}
		case "temp_file", "temp_dir":
			return []string{"string", "error"}
		case "abs_path", "rel_path":
			return []string{"string", "error"}
		case "file_info":
			return []string{"FileInfo", "error"}
		}
	case "json":
		switch funcName {
		case "encode", "pretty":
			return []string{"string", "error"}
		case "decode":
			// Type argument is required - extract the type from the second argument
			if len(args) >= 2 {
				if label, ok := args[1].(*ast.Label); ok {
					return []string{label.Value, "error"}
				}
			}
			// If no type argument, return empty (error will be caught by checkJsonModuleCall)
			return nil
		}
	case "os":
		switch funcName {
		case "get_env":
			return []string{"string", "error"}
		case "exec":
			return []string{"int", "error"}
		case "exec_output":
			return []string{"string", "error"}
		case "set_env", "unset_env":
			return []string{"bool", "error"}
		}
	case "bytes":
		switch funcName {
		case "from_hex", "from_base64":
			return []string{"[byte]", "error"}
		case "read_u8", "read_i8":
			return []string{"int", "error"}
		case "read_u16", "read_u16_be", "read_i16", "read_i16_be":
			return []string{"int", "error"}
		case "read_u32", "read_u32_be", "read_i32", "read_i32_be":
			return []string{"int", "error"}
		case "read_u64", "read_u64_be", "read_i64", "read_i64_be":
			return []string{"int", "error"}
		case "read_f32", "read_f32_be", "read_f64", "read_f64_be":
			return []string{"float", "error"}
		}
	case "binary":
		// All binary encode functions return ([byte], error)
		// All binary decode functions return (typed_int_or_float, error)
		switch funcName {
		case "encode_i8", "encode_u8",
			"encode_i16_to_little_endian", "encode_u16_to_little_endian",
			"encode_i16_to_big_endian", "encode_u16_to_big_endian",
			"encode_i32_to_little_endian", "encode_u32_to_little_endian",
			"encode_i32_to_big_endian", "encode_u32_to_big_endian",
			"encode_i64_to_little_endian", "encode_u64_to_little_endian",
			"encode_i64_to_big_endian", "encode_u64_to_big_endian",
			"encode_i128_to_little_endian", "encode_u128_to_little_endian",
			"encode_i128_to_big_endian", "encode_u128_to_big_endian",
			"encode_i256_to_little_endian", "encode_u256_to_little_endian",
			"encode_i256_to_big_endian", "encode_u256_to_big_endian",
			"encode_f32_to_little_endian", "encode_f32_to_big_endian",
			"encode_f64_to_little_endian", "encode_f64_to_big_endian":
			return []string{"[byte]", "error"}
		case "decode_i8", "decode_u8",
			"decode_i16_from_little_endian", "decode_u16_from_little_endian",
			"decode_i16_from_big_endian", "decode_u16_from_big_endian",
			"decode_i32_from_little_endian", "decode_u32_from_little_endian",
			"decode_i32_from_big_endian", "decode_u32_from_big_endian",
			"decode_i64_from_little_endian", "decode_u64_from_little_endian",
			"decode_i64_from_big_endian", "decode_u64_from_big_endian",
			"decode_i128_from_little_endian", "decode_u128_from_little_endian",
			"decode_i128_from_big_endian", "decode_u128_from_big_endian",
			"decode_i256_from_little_endian", "decode_u256_from_little_endian",
			"decode_i256_from_big_endian", "decode_u256_from_big_endian":
			return []string{"int", "error"}
		case "decode_f32_from_little_endian", "decode_f32_from_big_endian",
			"decode_f64_from_little_endian", "decode_f64_from_big_endian":
			return []string{"float", "error"}
		}

	case "db":
		switch funcName {
		case "open":
			return []string{"Database", "error"}
		case "get":
			return []string{"string", "bool"}
		}

	case "http":
		switch funcName {
		case "get", "post", "put", "delete", "patch", "request":
			return []string{"HttpResponse", "error"}
		case "decode_url":
			return []string{"string", "error"}
		}
	}
	return nil
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
	case "byte":
		return "byte", true
	case "i8":
		return "i8", true
	case "i16":
		return "i16", true
	case "i32":
		return "i32", true
	case "i64":
		return "i64", true
	case "u8":
		return "u8", true
	case "u16":
		return "u16", true
	case "u32":
		return "u32", true
	case "u64":
		return "u64", true
	case "i128":
		return "i128", true
	case "i256":
		return "i256", true
	case "u128":
		return "u128", true
	case "u256":
		return "u256", true
	case "f32":
		return "f32", true
	case "f64":
		return "f64", true
	case "input":
		return "string", true
	case "read_int":
		return "int", true
	case "copy", "ref":
		// copy() and ref() return the same type as their argument
		if len(args) > 0 {
			if argType, ok := tc.inferExpressionType(args[0]); ok {
				return argType, true
			}
		}
		return "", false
	case "error":
		return "error", true
	case "new":
		// new() returns an instance of the type passed as argument
		if len(args) > 0 {
			if label, ok := args[0].(*ast.Label); ok {
				return label.Value, true
			}
		}
		return "", false
	case "range":
		return "[int]", true
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

	// Mark the module as used (#639)
	tc.markModuleUsed(moduleName)

	// Standard library function return types
	resolvedModuleName := tc.resolveStdlibModule(moduleName)
	if returnTypes := tc.getModuleMultiReturnTypes(resolvedModuleName, funcName, args); len(returnTypes) > 1 {
		// Multi-return stdlib calls cannot be inferred as a single value.
		return "", true
	}
	switch resolvedModuleName {
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
	case "maps":
		return tc.inferMapsCallType(funcName, args)
	case "io":
		return tc.inferIOCallType(funcName, args)
	case "os":
		return tc.inferOSCallType(funcName, args)
	case "json":
		return tc.inferJSONCallType(funcName, args)
	case "bytes":
		return tc.inferBytesCallType(funcName, args)
	case "encoding":
		return tc.inferEncodingCallType(funcName, args)
	case "crypto":
		return tc.inferCryptoCallType(funcName, args)
	case "random":
		return tc.inferRandomCallType(funcName, args)
	case "uuid":
		return tc.inferUUIDCallType(funcName, args)
	case "db":
		return tc.inferDBCallType(funcName, args)
	case "binary":
		return tc.inferBinaryCallType(funcName, args)
	default:
		// Check user-defined modules
		if sig, ok := tc.GetModuleFunction(moduleName, funcName); ok {
			if len(sig.ReturnTypes) >= 1 {
				return sig.ReturnTypes[0], true
			}
			return "void", true
		}
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
	case "abs", "min", "max":
		// Return type matches input type
		if len(args) > 0 {
			argType, ok := tc.inferExpressionType(args[0])
			if ok && (argType == "int" || argType == "float") {
				return argType, true
			}
		}
		return "float", true // fallback
	case "floor", "ceil", "round", "sqrt", "pow", "log", "log2", "log10",
		"sin", "cos", "tan", "asin", "acos", "atan", "exp", "avg",
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
		// Return type matches array element type
		if len(args) > 0 {
			arrType, ok := tc.inferExpressionType(args[0])
			if ok && len(arrType) > 2 && arrType[0] == '[' {
				elemType := arrType[1 : len(arrType)-1]
				// Handle fixed-size arrays like [int, 5]
				if commaIdx := strings.Index(elemType, ","); commaIdx != -1 {
					elemType = strings.TrimSpace(elemType[:commaIdx])
				}
				return elemType, true
			}
		}
		return "float", true // fallback
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
	case "len", "index", "last_index", "count", "to_int":
		return "int", true
	case "to_float":
		return "float", true
	case "contains", "starts_with", "ends_with", "is_empty", "to_bool":
		return "bool", true
	case "upper", "lower", "trim", "trim_left", "trim_right", "reverse",
		"replace", "substring", "repeat", "pad_left", "pad_right", "join":
		return "string", true
	case "split":
		return "[string]", true
	case "chars":
		return "[char]", true
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

// inferMapsCallType infers return types for @maps functions
func (tc *TypeChecker) inferMapsCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "is_empty", "has", "has_key", "has_value", "equals":
		return "bool", true
	case "delete", "remove":
		return "bool", true
	case "set", "clear", "update":
		return "void", true
	case "keys", "values":
		// Returns array - try to infer element type from map argument
		if len(args) > 0 {
			mapType, ok := tc.inferExpressionType(args[0])
			if ok && tc.isMapType(mapType) {
				keyType := tc.extractMapKeyType(mapType)
				valueType := tc.extractMapValueType(mapType)
				if keyType != "" && valueType != "" {
					if funcName == "keys" {
						return "[" + keyType + "]", true
					}
					return "[" + valueType + "]", true
				}
			}
		}
		return "[]", true
	case "get", "get_or_set":
		// Returns value type from map
		if len(args) > 0 {
			mapType, ok := tc.inferExpressionType(args[0])
			if ok && tc.isMapType(mapType) {
				valueType := tc.extractMapValueType(mapType)
				if valueType != "" {
					return valueType, true
				}
			}
		}
		return "", false
	case "merge":
		// Returns merged map with same type as first argument
		if len(args) > 0 {
			mapType, ok := tc.inferExpressionType(args[0])
			if ok && tc.isMapType(mapType) {
				return mapType, true
			}
		}
		return "map", true
	case "invert", "from_array":
		return "map", true
	case "to_array":
		return "[[]]", true
	default:
		return "", false
	}
}

// inferIOCallType infers return types for @io functions
func (tc *TypeChecker) inferIOCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "read_file", "read_string":
		return "string", true
	case "read_bytes", "read", "read_all":
		return "[byte]", true
	case "write_file", "write_bytes", "append_file", "append_line",
		"remove", "remove_dir", "remove_all", "rename", "copy",
		"mkdir", "mkdir_all", "flush", "close":
		return "bool", true
	case "read_lines", "read_dir", "glob", "walk":
		return "[string]", true
	case "expand_path", "path_join", "path_base", "path_dir",
		"path_ext", "path_clean", "path_separator", "path_abs":
		return "string", true
	case "exists", "is_file", "is_dir", "is_symlink":
		return "bool", true
	case "file_size", "file_mod_time", "write", "seek", "tell":
		return "int", true
	case "open":
		return "File", true
	case "READ_ONLY", "WRITE_ONLY", "READ_WRITE", "APPEND",
		"CREATE", "TRUNCATE", "EXCLUSIVE",
		"SEEK_START", "SEEK_CURRENT", "SEEK_END":
		return "int", true
	default:
		return "", false
	}
}

// inferOSCallType infers return types for @os functions
func (tc *TypeChecker) inferOSCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "get_env", "platform", "arch", "line_separator", "dev_null",
		"cwd", "home_dir", "hostname", "username", "temp_dir":
		return "string", true
	case "set_env", "unset_env", "chdir":
		return "bool", true
	case "env":
		return "map[string:string]", true
	case "args":
		return "[string]", true
	case "exit":
		return "void", true
	case "pid", "ppid", "num_cpu",
		"MAC_OS", "LINUX", "WINDOWS", "CURRENT_OS":
		return "int", true
	case "is_windows", "is_linux", "is_macos":
		return "bool", true
	case "exec":
		return "int", true
	case "exec_output":
		return "string", true
	default:
		return "", false
	}
}

// inferJSONCallType infers return types for @json functions
func (tc *TypeChecker) inferJSONCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "encode", "pretty":
		return "string", true
	case "decode":
		// Type argument is required - extract the type from the second argument
		if len(args) >= 2 {
			if label, ok := args[1].(*ast.Label); ok {
				return label.Value, true
			}
		}
		// If no type argument, return false (error will be caught by checkJsonModuleCall)
		return "", false
	case "is_valid":
		return "bool", true
	default:
		return "", false
	}
}

// inferBytesCallType infers return types for @bytes functions
func (tc *TypeChecker) inferBytesCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "from_array", "from_string", "from_hex", "from_base64",
		"slice", "concat", "join", "reverse", "repeat",
		"replace", "replace_n", "trim", "trim_left", "trim_right",
		"pad_left", "pad_right", "fill", "copy", "zero", "not",
		"and", "or", "xor":
		return "[byte]", true
	case "to_string", "to_hex", "to_hex_upper", "to_base64":
		return "string", true
	case "to_array":
		return "[int]", true
	case "split":
		return "[[byte]]", true
	case "contains", "equals", "is_empty", "starts_with", "ends_with":
		return "bool", true
	case "index", "last_index", "count", "compare":
		return "int", true
	default:
		return "", false
	}
}

// inferEncodingCallType infers return types for @encoding functions
func (tc *TypeChecker) inferEncodingCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "base64_encode", "hex_encode", "url_encode",
		"base64_decode", "hex_decode", "url_decode":
		return "string", true
	default:
		return "", false
	}
}

// inferCryptoCallType infers return types for @crypto functions
func (tc *TypeChecker) inferCryptoCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "sha256", "sha512", "md5", "random_hex":
		return "string", true
	case "random_bytes":
		return "[byte]", true
	default:
		return "", false
	}
}

// inferRandomCallType infers return types for @random functions
func (tc *TypeChecker) inferRandomCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "float":
		return "float", true
	case "int":
		return "int", true
	case "bool":
		return "bool", true
	case "byte":
		return "byte", true
	case "char":
		return "char", true
	case "choice":
		// Returns element type from array
		if len(args) > 0 {
			arrType, ok := tc.inferExpressionType(args[0])
			if ok && len(arrType) > 2 && arrType[0] == '[' {
				return arrType[1 : len(arrType)-1], true
			}
		}
		return "", false
	case "shuffle", "sample":
		// Returns same array type
		if len(args) > 0 {
			return tc.inferExpressionType(args[0])
		}
		return "[]", true
	default:
		return "", false
	}
}

// inferUUIDCallType infers return types for @uuid functions
func (tc *TypeChecker) inferUUIDCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "create", "create_compact", "NIL":
		return "string", true
	case "is_valid":
		return "bool", true
	default:
		return "", false
	}
}

// inferDBCallType infers return types for @db functions
func (tc *TypeChecker) inferDBCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	case "open":
		return "Database", true
	case "close", "save":
		return "error", true
	case "set", "clear", "sort":
		return "void", true
	case "get":
		return "string", true
	case "delete", "has", "exists", "update_key_name":
		return "bool", true
	case "keys", "prefix":
		return "[string]", true
	case "count",
		"ALPHA", "ALPHA_DESC", "VALUE_ALPHA", "VALUE_ALPHA_DESC",
		"KEY_LEN", "KEY_LEN_DESC", "VALUE_LEN", "VALUE_LEN_DESC",
		"NUMERIC", "NUMERIC_DESC":
		return "int", true
	default:
		return "", false
	}
}

// inferBinaryCallType infers return types for @binary functions
func (tc *TypeChecker) inferBinaryCallType(funcName string, args []ast.Expression) (string, bool) {
	switch funcName {
	// Encode functions return [byte]
	case "encode_i8", "encode_u8",
		"encode_i16_to_little_endian", "encode_i16_to_big_endian",
		"encode_u16_to_little_endian", "encode_u16_to_big_endian",
		"encode_i32_to_little_endian", "encode_i32_to_big_endian",
		"encode_u32_to_little_endian", "encode_u32_to_big_endian",
		"encode_i64_to_little_endian", "encode_i64_to_big_endian",
		"encode_u64_to_little_endian", "encode_u64_to_big_endian",
		"encode_i128_to_little_endian", "encode_i128_to_big_endian",
		"encode_u128_to_little_endian", "encode_u128_to_big_endian",
		"encode_i256_to_little_endian", "encode_i256_to_big_endian",
		"encode_u256_to_little_endian", "encode_u256_to_big_endian",
		"encode_f32_to_little_endian", "encode_f32_to_big_endian",
		"encode_f64_to_little_endian", "encode_f64_to_big_endian":
		return "[byte]", true
	// Integer decode functions return int
	case "decode_i8", "decode_u8",
		"decode_i16_from_little_endian", "decode_i16_from_big_endian",
		"decode_u16_from_little_endian", "decode_u16_from_big_endian",
		"decode_i32_from_little_endian", "decode_i32_from_big_endian",
		"decode_u32_from_little_endian", "decode_u32_from_big_endian",
		"decode_i64_from_little_endian", "decode_i64_from_big_endian",
		"decode_u64_from_little_endian", "decode_u64_from_big_endian",
		"decode_i128_from_little_endian", "decode_i128_from_big_endian",
		"decode_u128_from_little_endian", "decode_u128_from_big_endian",
		"decode_i256_from_little_endian", "decode_i256_from_big_endian",
		"decode_u256_from_little_endian", "decode_u256_from_big_endian":
		return "int", true
	// Float decode functions return float
	case "decode_f32_from_little_endian", "decode_f32_from_big_endian",
		"decode_f64_from_little_endian", "decode_f64_from_big_endian":
		return "float", true
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
		// Extract element type from [type] or [type,size]
		inner := leftType[1 : len(leftType)-1]
		// Handle fixed-size arrays: [int,3] -> int
		if commaIdx := strings.Index(inner, ","); commaIdx != -1 {
			return inner[:commaIdx], true
		}
		return inner, true
	}

	// Indexing into a map returns value type
	if tc.isMapType(leftType) {
		valueType := tc.extractMapValueType(leftType)
		if valueType != "" {
			return valueType, true
		}
	}

	return "", false
}

// inferMemberType infers the type of a member access expression
func (tc *TypeChecker) inferMemberType(member *ast.MemberExpression) (string, bool) {
	// Check if accessing enum member (e.g., Status.ACTIVE)
	if label, isLabel := member.Object.(*ast.Label); isLabel {
		if enumType, exists := tc.types[label.Value]; exists && enumType.Kind == EnumType {
			// Validate that the member exists (#607)
			memberName := member.Member.Value
			if enumType.EnumMembers != nil && !enumType.EnumMembers[memberName] {
				tc.addError(
					errors.E4004,
					fmt.Sprintf("enum '%s' has no member '%s'", label.Value, memberName),
					member.Member.Token.Line,
					member.Member.Token.Column,
				)
				return "", false
			}
			// Return the enum type name
			return label.Value, true
		}

		// Check if accessing module variable (e.g., lib.Numbers) (#677)
		moduleName := label.Value
		if tc.modules[moduleName] {
			memberName := member.Member.Value
			if varType, ok := tc.GetModuleVariable(moduleName, memberName); ok {
				// Mark the module as used (#639)
				tc.markModuleUsed(moduleName)
				return varType, true
			}
		}
	}

	// Check if accessing imported enum member (e.g., lib.Status.ACTIVE)
	// In this case, member.Object is itself a MemberExpression (lib.Status)
	if innerMember, isInnerMember := member.Object.(*ast.MemberExpression); isInnerMember {
		if moduleLabel, isModuleLabel := innerMember.Object.(*ast.Label); isModuleLabel {
			moduleName := moduleLabel.Value
			enumName := innerMember.Member.Value
			memberName := member.Member.Value

			// Check if this is a module.EnumType.MEMBER pattern
			if tc.modules[moduleName] {
				if moduleTypes, hasModule := tc.moduleTypes[moduleName]; hasModule {
					if enumType, hasEnum := moduleTypes[enumName]; hasEnum && enumType.Kind == EnumType {
						// Mark the module as used (#639)
						tc.markModuleUsed(moduleName)
						// Validate that the enum member exists
						if enumType.EnumMembers != nil && !enumType.EnumMembers[memberName] {
							tc.addError(
								errors.E4004,
								fmt.Sprintf("enum '%s.%s' has no member '%s'", moduleName, enumName, memberName),
								member.Member.Token.Line,
								member.Member.Token.Column,
							)
							return "", false
						}
						// Return the qualified enum type name
						return moduleName + "." + enumName, true
					}
				}
			}
		}
	}

	// Check if accessing struct field
	objType, ok := tc.inferExpressionType(member.Object)
	if !ok {
		return "", false
	}

	// Look up struct type (including module types for qualified names like "lib.Hero")
	if structType, exists := tc.getStructTypeIncludingModules(objType); exists {
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
		"float", "f32", "f64", "byte":
		return true
	default:
		return false
	}
}

// isEnumType checks if a type name refers to a user-defined enum type
func (tc *TypeChecker) isEnumType(typeName string) bool {
	if t, exists := tc.types[typeName]; exists {
		return t.Kind == EnumType
	}
	return false
}

// isComparableEnumType checks if a type is an enum with a comparable base type (numeric)
func (tc *TypeChecker) isComparableEnumType(typeName string) bool {
	if t, exists := tc.types[typeName]; exists && t.Kind == EnumType {
		return tc.isNumericType(t.EnumBaseType)
	}
	return false
}

// getPromotedType returns the resulting type when two numeric types are used together
// byte is promoted to the other type in mixed operations
func (tc *TypeChecker) getPromotedType(left, right string) string {
	if left == "byte" {
		return right
	}
	return left
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

// getIntegerBounds returns the min and max values for an integer type
// Returns (0, 0, false) if not a known integer type
func (tc *TypeChecker) getIntegerBounds(typeName string) (min, max int64, ok bool) {
	switch typeName {
	case "i8":
		return -128, 127, true
	case "i16":
		return -32768, 32767, true
	case "i32":
		return -2147483648, 2147483647, true
	case "i64", "int":
		return -9223372036854775808, 9223372036854775807, true
	case "u8", "byte":
		return 0, 255, true
	case "u16":
		return 0, 65535, true
	case "u32":
		return 0, 4294967295, true
	case "u64", "uint":
		// Note: max u64 exceeds int64, but we'll use int64 max for overflow checking
		return 0, 9223372036854775807, true
	default:
		return 0, 0, false
	}
}

// checkArithmeticOverflow checks if an arithmetic operation with literal values would overflow
// Returns true and a warning message if overflow is detected
func (tc *TypeChecker) checkArithmeticOverflow(left, right int64, operator, resultType string) (bool, string) {
	minVal, maxVal, ok := tc.getIntegerBounds(resultType)
	if !ok {
		return false, ""
	}

	var result int64
	var overflows bool

	switch operator {
	case "+":
		// Check for addition overflow
		if right > 0 && left > maxVal-right {
			overflows = true
		} else if right < 0 && left < minVal-right {
			overflows = true
		} else {
			result = left + right
		}
	case "-":
		// Check for subtraction overflow
		if right < 0 && left > maxVal+right {
			overflows = true
		} else if right > 0 && left < minVal+right {
			overflows = true
		} else {
			result = left - right
		}
	case "*":
		// Check for multiplication overflow
		if left != 0 && right != 0 {
			result = left * right
			if result/left != right {
				overflows = true
			}
		}
	default:
		return false, ""
	}

	if overflows || result > maxVal || result < minVal {
		return true, fmt.Sprintf("%s arithmetic with values %d %s %d overflows type %s (range %d to %d)",
			resultType, left, operator, right, resultType, minVal, maxVal)
	}
	return false, ""
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

	// error/Error are interchangeable (error is alias for Error struct)
	if (declared == "error" && actual == "Error") || (declared == "Error" && actual == "error") {
		return true
	}

	// Handle module-prefixed types (e.g., utils.Hero vs Hero, or Hero vs utils.Hero)
	// Strip module prefix and compare base type names
	declaredBase := tc.stripModulePrefix(declared)
	actualBase := tc.stripModulePrefix(actual)
	if declaredBase == actualBase && (declaredBase != declared || actualBase != actual) {
		// Base names match and at least one had a module prefix
		return true
	}

	// Handle enum-to-base-type compatibility
	// Enum values are compatible with their underlying base type
	if declaredType, exists := tc.types[declared]; exists && declaredType.Kind == EnumType {
		if declaredType.EnumBaseType == actual {
			return true
		}
	}
	if actualType, exists := tc.types[actual]; exists && actualType.Kind == EnumType {
		if actualType.EnumBaseType == declared {
			return true
		}
	}

	// nil is only compatible with reference types (arrays, maps, structs)
	// Primitive types (int, float, string, bool, char, byte) cannot be nil
	if actual == "nil" {
		return tc.isNullableType(declared)
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

		// Empty braces {} parsed as empty array [] should also be compatible with map types
		if actual == "[]" {
			return true
		}

		if tc.isMapType(actual) {
			declaredKey := tc.extractMapKeyType(declared)
			declaredValue := tc.extractMapValueType(declared)
			actualKey := tc.extractMapKeyType(actual)
			actualValue := tc.extractMapValueType(actual)
			if declaredKey != "" && declaredValue != "" && actualKey != "" && actualValue != "" {
				keyCompatible := tc.typesCompatible(declaredKey, actualKey)
				valueCompatible := tc.typesCompatible(declaredValue, actualValue)
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

	// Byte type compatibility - bytes are like unsigned 8-bit integers
	// Allow int to byte assignment (runtime validates 0-255 range)
	if declared == "byte" && (tc.isSignedIntegerType(actual) || tc.isUnsignedIntegerType(actual)) {
		return true
	}

	// Allow byte to integer/unsigned types
	if (tc.isSignedIntegerType(declared) || tc.isUnsignedIntegerType(declared)) && actual == "byte" {
		return true
	}

	return false
}

// stripModulePrefix removes the module prefix from a type name
// e.g., "utils.Hero" -> "Hero", "Hero" -> "Hero"
func (tc *TypeChecker) stripModulePrefix(typeName string) string {
	if idx := strings.LastIndex(typeName, "."); idx != -1 {
		return typeName[idx+1:]
	}
	return typeName
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
// hasUsingModule checks if a module is imported via 'using' at file or scope level
func (tc *TypeChecker) hasUsingModule(moduleName string) bool {
	// Check file-level using
	if tc.fileUsingModules[moduleName] {
		return true
	}
	// Check scope-level using
	if tc.currentScope != nil && tc.currentScope.HasUsingModule(moduleName) {
		return true
	}
	return false
}

func (tc *TypeChecker) resolveStdlibModule(moduleName string) string {
	if resolved, ok := tc.stdlibAliases[moduleName]; ok {
		return resolved
	}
	return moduleName
}

func (tc *TypeChecker) hasUsingStdlibModule(moduleName string) bool {
	if tc.hasUsingModule(moduleName) {
		return true
	}
	for alias, resolved := range tc.stdlibAliases {
		if resolved == moduleName && tc.hasUsingModule(alias) {
			return true
		}
	}
	return false
}

func (tc *TypeChecker) checkDirectStdlibCall(funcName string, call *ast.CallExpression) bool {
	line, column := tc.getExpressionPosition(call.Function)

	// Check std module
	if tc.hasUsingStdlibModule("std") {
		stdFuncs := map[string]bool{
			"println": true, "print": true, "printf": true,
			"eprintln": true, "eprint": true, "eprintf": true,
			"sleep_milliseconds": true, "sleep_seconds": true, "sleep_nanoseconds": true,
			"read_int": true, "read_float": true, "read_string": true,
		}
		if stdFuncs[funcName] {
			tc.checkStdModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check global builtins (no using required)
	globalBuiltins := map[string]bool{
		"exit": true, "panic": true, "assert": true,
	}
	if globalBuiltins[funcName] {
		return true // These are handled by runtime
	}

	// Check math module
	if tc.hasUsingStdlibModule("math") {
		if tc.isMathFunction(funcName) {
			tc.checkMathModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check arrays module
	if tc.hasUsingStdlibModule("arrays") {
		if tc.isArraysFunction(funcName) {
			tc.checkArraysModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check strings module
	if tc.hasUsingStdlibModule("strings") {
		if tc.isStringsFunction(funcName) {
			tc.checkStringsModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check time module
	if tc.hasUsingStdlibModule("time") {
		if tc.isTimeFunction(funcName) {
			tc.checkTimeModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check maps module
	if tc.hasUsingStdlibModule("maps") {
		if tc.isMapsFunction(funcName) {
			tc.checkMapsModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check io module
	if tc.hasUsingStdlibModule("io") {
		if tc.isIoFunction(funcName) {
			tc.checkIoModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check os module
	if tc.hasUsingStdlibModule("os") {
		if tc.isOsFunction(funcName) {
			tc.checkOsModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check random module
	if tc.hasUsingStdlibModule("random") {
		if tc.isRandomFunction(funcName) {
			tc.checkRandomModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check json module
	if tc.hasUsingStdlibModule("json") {
		if tc.isJsonFunction(funcName) {
			tc.checkJsonModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check bytes module
	if tc.hasUsingStdlibModule("bytes") {
		if tc.isBytesFunction(funcName) {
			tc.checkBytesModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check binary module
	if tc.hasUsingStdlibModule("binary") {
		if tc.isBinaryFunction(funcName) {
			tc.checkBinaryModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check db module
	if tc.hasUsingStdlibModule("db") {
		if tc.isDBFunction(funcName) {
			tc.checkDBModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check uuid module
	if tc.hasUsingStdlibModule("uuid") {
		if tc.isUuidFunction(funcName) {
			tc.checkUuidModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check encoding module
	if tc.hasUsingStdlibModule("encoding") {
		if tc.isEncodingFunction(funcName) {
			tc.checkEncodingModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check crypto module
	if tc.hasUsingStdlibModule("crypto") {
		if tc.isCryptoFunction(funcName) {
			tc.checkCryptoModuleCall(funcName, call, line, column)
			return true
		}
	}

	// Check user-defined module functions accessible via 'using' (#671)
	for moduleName := range tc.fileUsingModules {
		if funcs, ok := tc.moduleFunctions[moduleName]; ok {
			if sig, exists := funcs[funcName]; exists {
				// Validate argument count
				tc.validateCallArguments(funcName, call, sig, line, column)
				return true
			}
		}
	}
	if tc.currentScope != nil {
		for moduleName := range tc.currentScope.usingModules {
			if funcs, ok := tc.moduleFunctions[moduleName]; ok {
				if sig, exists := funcs[funcName]; exists {
					tc.validateCallArguments(funcName, call, sig, line, column)
					return true
				}
			}
		}
	}

	return false
}

func (tc *TypeChecker) inferDirectStdlibCallType(funcName string, args []ast.Expression) (string, bool) {
	if tc.hasUsingStdlibModule("std") {
		if inferred, ok := tc.inferStdCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("math") {
		if inferred, ok := tc.inferMathCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("arrays") {
		if inferred, ok := tc.inferArraysCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("strings") {
		if inferred, ok := tc.inferStringsCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("time") {
		if inferred, ok := tc.inferTimeCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("maps") {
		if inferred, ok := tc.inferMapsCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("io") {
		if returnTypes := tc.getModuleMultiReturnTypes("io", funcName, args); len(returnTypes) > 1 {
			return "", true
		}
		if inferred, ok := tc.inferIOCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("os") {
		if returnTypes := tc.getModuleMultiReturnTypes("os", funcName, args); len(returnTypes) > 1 {
			return "", true
		}
		if inferred, ok := tc.inferOSCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("random") {
		if inferred, ok := tc.inferRandomCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("json") {
		if returnTypes := tc.getModuleMultiReturnTypes("json", funcName, args); len(returnTypes) > 1 {
			return "", true
		}
		if inferred, ok := tc.inferJSONCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("bytes") {
		if returnTypes := tc.getModuleMultiReturnTypes("bytes", funcName, args); len(returnTypes) > 1 {
			return "", true
		}
		if inferred, ok := tc.inferBytesCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("binary") {
		if returnTypes := tc.getModuleMultiReturnTypes("binary", funcName, args); len(returnTypes) > 1 {
			return "", true
		}
		if inferred, ok := tc.inferBinaryCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("db") {
		if returnTypes := tc.getModuleMultiReturnTypes("db", funcName, args); len(returnTypes) > 1 {
			return "", true
		}
		if inferred, ok := tc.inferDBCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("uuid") {
		if inferred, ok := tc.inferUUIDCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("encoding") {
		if inferred, ok := tc.inferEncodingCallType(funcName, args); ok {
			return inferred, true
		}
	}
	if tc.hasUsingStdlibModule("crypto") {
		if inferred, ok := tc.inferCryptoCallType(funcName, args); ok {
			return inferred, true
		}
	}
	return "", false
}

// validateCallArguments validates argument count for a function call (#671)
func (tc *TypeChecker) validateCallArguments(funcName string, call *ast.CallExpression, sig *FunctionSignature, line, column int) {
	// Calculate minimum required arguments (parameters without defaults)
	minRequired := 0
	for _, param := range sig.Parameters {
		if !param.HasDefault {
			minRequired++
		}
	}

	// Check argument count
	if len(call.Arguments) < minRequired || len(call.Arguments) > len(sig.Parameters) {
		var msg string
		if minRequired == len(sig.Parameters) {
			msg = fmt.Sprintf("wrong number of arguments to '%s': expected %d, got %d",
				funcName, len(sig.Parameters), len(call.Arguments))
		} else {
			msg = fmt.Sprintf("wrong number of arguments to '%s': expected %d to %d, got %d",
				funcName, minRequired, len(sig.Parameters), len(call.Arguments))
		}
		tc.addError(errors.E5008, msg, line, column)
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
		"join": true, "replace": true, "to_int": true, "to_float": true, "to_bool": true,
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

// isMapsFunction checks if a function name exists in the maps module
func (tc *TypeChecker) isMapsFunction(name string) bool {
	mapsFuncs := map[string]bool{
		"len": true, "is_empty": true, "keys": true, "values": true, "clear": true,
		"to_array": true, "invert": true, "has": true, "has_key": true, "delete": true,
		"remove": true, "has_value": true, "get": true, "set": true, "get_or_set": true,
		"merge": true, "copy": true,
	}
	return mapsFuncs[name]
}

// isStdFunction checks if a function name exists in the std module
func (tc *TypeChecker) isStdFunction(name string) bool {
	stdFuncs := map[string]bool{
		"println": true, "print": true, "printf": true,
	}
	return stdFuncs[name]
}

// isIoFunction checks if a function name exists in the io module
func (tc *TypeChecker) isIoFunction(name string) bool {
	ioFuncs := map[string]bool{
		// File reading
		"read_file": true, "read_bytes": true, "read_lines": true,
		// File writing
		"write_file": true, "write_bytes": true, "append_file": true, "append_line": true,
		// Path utilities
		"expand_path": true, "path_join": true, "path_base": true, "path_dir": true,
		"path_ext": true, "path_abs": true, "path_clean": true, "path_separator": true,
		// File checks
		"exists": true, "is_file": true, "is_dir": true, "is_symlink": true,
		// File operations
		"remove": true, "remove_dir": true, "remove_all": true, "rename": true, "copy": true,
		// Directory operations
		"mkdir": true, "mkdir_all": true, "read_dir": true,
		// File metadata
		"file_size": true, "file_mod_time": true,
		// File handle operations
		"open": true, "read": true, "read_all": true, "read_string": true,
		"write": true, "seek": true, "tell": true, "flush": true, "close": true,
		// Filesystem utilities
		"glob": true, "walk": true,
		// Constants (accessed as functions)
		"READ_ONLY": true, "WRITE_ONLY": true, "READ_WRITE": true,
		"APPEND": true, "CREATE": true, "TRUNCATE": true, "EXCLUSIVE": true,
		"SEEK_START": true, "SEEK_CURRENT": true, "SEEK_END": true,
	}
	return ioFuncs[name]
}

// isOsFunction checks if a function name exists in the os module
func (tc *TypeChecker) isOsFunction(name string) bool {
	osFuncs := map[string]bool{
		// Environment variables
		"get_env": true, "set_env": true, "unset_env": true, "env": true, "args": true,
		// Process / System
		"exit": true, "cwd": true, "chdir": true, "hostname": true, "username": true,
		"home_dir": true, "temp_dir": true, "pid": true, "ppid": true,
		// Platform detection
		"platform": true, "arch": true, "is_windows": true, "is_linux": true, "is_macos": true,
		"num_cpu": true, "line_separator": true, "dev_null": true,
		// Command execution
		"exec": true, "exec_output": true,
		// Constants
		"MAC_OS": true, "LINUX": true, "WINDOWS": true, "CURRENT_OS": true,
	}
	return osFuncs[name]
}

// isRandomFunction checks if a function name exists in the random module
func (tc *TypeChecker) isRandomFunction(name string) bool {
	randomFuncs := map[string]bool{
		"float": true, "int": true, "bool": true, "byte": true, "char": true,
		"choice": true, "shuffle": true, "sample": true,
	}
	return randomFuncs[name]
}

// isJsonFunction checks if a function name exists in the json module
func (tc *TypeChecker) isJsonFunction(name string) bool {
	jsonFuncs := map[string]bool{
		"encode": true, "decode": true, "pretty": true, "is_valid": true,
	}
	return jsonFuncs[name]
}

func (tc *TypeChecker) isUuidFunction(name string) bool {
	uuidFuncs := map[string]bool{
		"create": true, "create_compact": true, "NIL": true, "is_valid": true,
	}
	return uuidFuncs[name]
}

func (tc *TypeChecker) isEncodingFunction(name string) bool {
	encodingFuncs := map[string]bool{
		"base64_encode": true, "hex_encode": true, "url_encode": true,
		"base64_decode": true, "hex_decode": true, "url_decode": true,
	}
	return encodingFuncs[name]
}

func (tc *TypeChecker) isCryptoFunction(name string) bool {
	cryptoFuncs := map[string]bool{
		"sha256": true, "sha512": true, "md5": true,
		"random_hex": true, "random_bytes": true,
	}
	return cryptoFuncs[name]
}

func (tc *TypeChecker) isDBFunction(name string) bool {
	dbFuncs := map[string]bool{
		// Creation
		"open": true,
		// Closing
		"close": true,
		// Saving
		"save": true,
		// Operations
		"set": true, "get": true, "delete": true, "has": true,
		"keys": true, "prefix": true, "count": true, "clear": true,
	}
	return dbFuncs[name]
}

func (tc *TypeChecker) isHttpFunction(name string) bool {
	httpFuncs := map[string]bool{
		"get": true, "post": true, "put": true, "delete": true,
		"patch": true,

		"request": true,

		"encode_url": true, "decode_url": true, "build_query": true,

		"json_body": true,
	}
	return httpFuncs[name]
}

// isBytesFunction checks if a function name exists in the bytes module
func (tc *TypeChecker) isBytesFunction(name string) bool {
	bytesFuncs := map[string]bool{
		// Creation
		"from_array": true, "from_string": true, "from_hex": true, "from_base64": true,
		// Conversion
		"to_string": true, "to_array": true, "to_hex": true, "to_hex_upper": true, "to_base64": true,
		// Slicing and combining
		"slice": true, "concat": true, "join": true, "split": true,
		// Search
		"contains": true, "index": true, "last_index": true, "count": true,
		// Comparison
		"compare": true, "equals": true, "is_empty": true, "starts_with": true, "ends_with": true,
		// Transformation
		"reverse": true, "repeat": true, "replace": true, "replace_n": true,
		"trim": true, "trim_left": true, "trim_right": true,
		"pad_left": true, "pad_right": true,
		// Bitwise
		"and": true, "or": true, "xor": true, "not": true,
		// Utilities
		"fill": true, "copy": true, "zero": true,
	}
	return bytesFuncs[name]
}

// isBinaryFunction checks if a function name exists in the binary module
func (tc *TypeChecker) isBinaryFunction(name string) bool {
	binaryFuncs := map[string]bool{
		// 8-bit (no endianness)
		"encode_i8": true, "decode_i8": true,
		"encode_u8": true, "decode_u8": true,
		// 16-bit little endian
		"encode_i16_to_little_endian": true, "decode_i16_from_little_endian": true,
		"encode_u16_to_little_endian": true, "decode_u16_from_little_endian": true,
		// 16-bit big endian
		"encode_i16_to_big_endian": true, "decode_i16_from_big_endian": true,
		"encode_u16_to_big_endian": true, "decode_u16_from_big_endian": true,
		// 32-bit little endian
		"encode_i32_to_little_endian": true, "decode_i32_from_little_endian": true,
		"encode_u32_to_little_endian": true, "decode_u32_from_little_endian": true,
		// 32-bit big endian
		"encode_i32_to_big_endian": true, "decode_i32_from_big_endian": true,
		"encode_u32_to_big_endian": true, "decode_u32_from_big_endian": true,
		// 64-bit little endian
		"encode_i64_to_little_endian": true, "decode_i64_from_little_endian": true,
		"encode_u64_to_little_endian": true, "decode_u64_from_little_endian": true,
		// 64-bit big endian
		"encode_i64_to_big_endian": true, "decode_i64_from_big_endian": true,
		"encode_u64_to_big_endian": true, "decode_u64_from_big_endian": true,
		// 128-bit little endian
		"encode_i128_to_little_endian": true, "decode_i128_from_little_endian": true,
		"encode_u128_to_little_endian": true, "decode_u128_from_little_endian": true,
		// 128-bit big endian
		"encode_i128_to_big_endian": true, "decode_i128_from_big_endian": true,
		"encode_u128_to_big_endian": true, "decode_u128_from_big_endian": true,
		// 256-bit little endian
		"encode_i256_to_little_endian": true, "decode_i256_from_little_endian": true,
		"encode_u256_to_little_endian": true, "decode_u256_from_little_endian": true,
		// 256-bit big endian
		"encode_i256_to_big_endian": true, "decode_i256_from_big_endian": true,
		"encode_u256_to_big_endian": true, "decode_u256_from_big_endian": true,
		// Float little endian
		"encode_f32_to_little_endian": true, "decode_f32_from_little_endian": true,
		"encode_f64_to_little_endian": true, "decode_f64_from_little_endian": true,
		// Float big endian
		"encode_f32_to_big_endian": true, "decode_f32_from_big_endian": true,
		"encode_f64_to_big_endian": true, "decode_f64_from_big_endian": true,
	}
	return binaryFuncs[name]
}

// getUsedModuleShadowingFunction checks if a name shadows a function from a used module
// Returns the module name if there's a shadow, empty string otherwise
func (tc *TypeChecker) getUsedModuleShadowingFunction(name string) string {
	// Check file-level using modules
	for moduleName := range tc.fileUsingModules {
		if tc.isModuleFunction(moduleName, name) {
			return moduleName
		}
	}

	// Check scope-level using modules
	if tc.currentScope != nil {
		for _, moduleName := range tc.currentScope.GetAllUsingModules() {
			if tc.isModuleFunction(moduleName, name) {
				return moduleName
			}
		}
	}

	return ""
}

// isModuleFunction checks if a function name exists in the specified module
func (tc *TypeChecker) isModuleFunction(moduleName, funcName string) bool {
	resolvedModuleName := tc.resolveStdlibModule(moduleName)
	switch resolvedModuleName {
	case "std":
		return tc.isStdFunction(funcName)
	case "math":
		return tc.isMathFunction(funcName)
	case "arrays":
		return tc.isArraysFunction(funcName)
	case "strings":
		return tc.isStringsFunction(funcName)
	case "time":
		return tc.isTimeFunction(funcName)
	case "maps":
		return tc.isMapsFunction(funcName)
	case "io":
		return tc.isIoFunction(funcName)
	case "os":
		return tc.isOsFunction(funcName)
	case "random":
		return tc.isRandomFunction(funcName)
	case "json":
		return tc.isJsonFunction(funcName)
	case "bytes":
		return tc.isBytesFunction(funcName)
	case "binary":
		return tc.isBinaryFunction(funcName)
	case "db":
		return tc.isDBFunction(funcName)
	case "http":
		return tc.isHttpFunction(funcName)
	case "uuid":
		return tc.isUuidFunction(funcName)
	case "encoding":
		return tc.isEncodingFunction(funcName)
	case "crypto":
		return tc.isCryptoFunction(funcName)
	default:
		// Check user-defined modules
		if funcs, ok := tc.moduleFunctions[moduleName]; ok {
			_, exists := funcs[funcName]
			return exists
		}
		return false
	}
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
	resolvedModuleName := tc.resolveStdlibModule(moduleName)

	// Check if the module was imported (for standard library modules)
	stdModules := map[string]bool{"std": true, "math": true, "arrays": true, "strings": true, "time": true, "maps": true, "io": true, "os": true, "bytes": true, "random": true, "json": true, "binary": true, "db": true, "uuid": true, "encoding": true, "crypto": true, "http": true}
	if stdModules[resolvedModuleName] && !tc.modules[moduleName] && !tc.modules[resolvedModuleName] {
		tc.addError(errors.E4007, fmt.Sprintf("module '%s' not imported; add 'import @%s'", moduleName, resolvedModuleName), line, column)
		return
	}

	// Mark the module as used (#639)
	tc.markModuleUsed(moduleName)

	switch resolvedModuleName {
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
	case "io":
		tc.checkIoModuleCall(funcName, call, line, column)
	case "os":
		tc.checkOsModuleCall(funcName, call, line, column)
	case "random":
		tc.checkRandomModuleCall(funcName, call, line, column)
	case "json":
		tc.checkJsonModuleCall(funcName, call, line, column)
	case "bytes":
		tc.checkBytesModuleCall(funcName, call, line, column)
	case "binary":
		tc.checkBinaryModuleCall(funcName, call, line, column)
	case "db":
		tc.checkDBModuleCall(funcName, call, line, column)
	case "uuid":
		tc.checkUuidModuleCall(funcName, call, line, column)
	case "encoding":
		tc.checkEncodingModuleCall(funcName, call, line, column)
	case "crypto":
		tc.checkCryptoModuleCall(funcName, call, line, column)
	case "http":
		tc.checkHttpModuleCall(funcName, call, line, column)
	default:
		// User-defined module - check if we have type info for it
		tc.checkUserModuleCall(moduleName, funcName, call, line, column)
	}
}

// checkUserModuleCall validates a user-defined module function call
func (tc *TypeChecker) checkUserModuleCall(moduleName, funcName string, call *ast.CallExpression, line, column int) {
	// Look up function signature in registered module functions
	sig, ok := tc.GetModuleFunction(moduleName, funcName)
	if !ok {
		// Check if the module itself is registered - if so, the function doesn't exist
		if tc.modules[moduleName] {
			// Module is registered but function doesn't exist - report error
			tc.addError(
				errors.E4002,
				fmt.Sprintf("undefined function '%s.%s'", moduleName, funcName),
				line,
				column,
			)
		}
		return
	}

	// Mark the module as used (#639)
	tc.markModuleUsed(moduleName)

	// Calculate minimum required arguments (parameters without defaults)
	minRequired := 0
	for _, param := range sig.Parameters {
		if !param.HasDefault {
			minRequired++
		}
	}

	// Check argument count
	if len(call.Arguments) < minRequired || len(call.Arguments) > len(sig.Parameters) {
		var msg string
		if minRequired == len(sig.Parameters) {
			msg = fmt.Sprintf("wrong number of arguments to '%s.%s': expected %d, got %d",
				moduleName, funcName, len(sig.Parameters), len(call.Arguments))
		} else {
			msg = fmt.Sprintf("wrong number of arguments to '%s.%s': expected %d to %d, got %d",
				moduleName, funcName, minRequired, len(sig.Parameters), len(call.Arguments))
		}
		tc.addError(errors.E5008, msg, line, column)
		return
	}

	// Check argument types
	for i, arg := range call.Arguments {
		if i >= len(sig.Parameters) {
			break
		}

		actualType, ok := tc.inferExpressionType(arg)
		if !ok {
			continue
		}

		expectedType := sig.Parameters[i].Type
		if !tc.typesCompatible(expectedType, actualType) {
			argLine, argColumn := tc.getExpressionPosition(arg)
			displayType := actualType
			if displayType == "" {
				displayType = "<unknown type>"
			}
			tc.addError(
				errors.E3001,
				fmt.Sprintf("argument type mismatch in call to '%s.%s': parameter '%s' expects %s, got %s",
					moduleName, funcName, sig.Parameters[i].Name, expectedType, displayType),
				argLine,
				argColumn,
			)
		}

		// Check for const -> & param error
		if sig.Parameters[i].Mutable {
			if label, isLabel := arg.(*ast.Label); isLabel {
				isMutable, found := tc.isVariableMutable(label.Value)
				if found && !isMutable {
					argLine, argColumn := tc.getExpressionPosition(arg)
					tc.addError(
						errors.E3027,
						fmt.Sprintf("cannot pass immutable variable '%s' to mutable parameter '&%s' in call to '%s.%s'",
							label.Value, sig.Parameters[i].Name, moduleName, funcName),
						argLine,
						argColumn,
					)
				}
			}
		}
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

		// Array + int + value (modify in-place, return void)
		"set":    {3, 3, []string{"array", "int", "any"}, "void"},
		"insert": {3, 3, []string{"array", "int", "any"}, "void"},

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

	// Additional element type validation for functions that modify arrays (#589, #590, #591)
	tc.checkArrayElementTypeCompatibility(funcName, call, line, column)
}

// checkArrayElementTypeCompatibility validates that elements being added to arrays
// have types compatible with the array's element type
func (tc *TypeChecker) checkArrayElementTypeCompatibility(funcName string, call *ast.CallExpression, line, column int) {
	if len(call.Arguments) < 2 {
		return
	}

	switch funcName {
	case "append", "unshift", "contains", "index_of", "last_index_of", "count", "remove", "remove_all", "fill":
		// First arg is array, remaining args are elements
		arrayType, ok := tc.inferExpressionType(call.Arguments[0])
		if !ok || !tc.isArrayType(arrayType) {
			return
		}
		elementType := tc.extractArrayElementType(arrayType)

		// Check each element argument (starting from index 1)
		for i := 1; i < len(call.Arguments); i++ {
			argType, ok := tc.inferExpressionType(call.Arguments[i])
			if !ok {
				continue
			}
			if !tc.typesCompatible(elementType, argType) {
				argLine, argCol := tc.getExpressionPosition(call.Arguments[i])
				tc.addError(errors.E3001,
					fmt.Sprintf("arrays.%s: element type mismatch - array has element type %s, but got %s",
						funcName, elementType, argType),
					argLine, argCol)
			}
		}

	case "insert", "set":
		// First arg is array, second is index, third is element
		if len(call.Arguments) < 3 {
			return
		}
		arrayType, ok := tc.inferExpressionType(call.Arguments[0])
		if !ok || !tc.isArrayType(arrayType) {
			return
		}
		elementType := tc.extractArrayElementType(arrayType)

		argType, ok := tc.inferExpressionType(call.Arguments[2])
		if !ok {
			return
		}
		if !tc.typesCompatible(elementType, argType) {
			argLine, argCol := tc.getExpressionPosition(call.Arguments[2])
			tc.addError(errors.E3001,
				fmt.Sprintf("arrays.%s: element type mismatch - array has element type %s, but got %s",
					funcName, elementType, argType),
				argLine, argCol)
		}

	case "concat", "zip":
		// All args are arrays - check they have compatible element types
		firstArrayType, ok := tc.inferExpressionType(call.Arguments[0])
		if !ok || !tc.isArrayType(firstArrayType) {
			return
		}
		firstElementType := tc.extractArrayElementType(firstArrayType)

		for i := 1; i < len(call.Arguments); i++ {
			argType, ok := tc.inferExpressionType(call.Arguments[i])
			if !ok || !tc.isArrayType(argType) {
				continue
			}
			argElementType := tc.extractArrayElementType(argType)
			if !tc.typesCompatible(firstElementType, argElementType) {
				argLine, argCol := tc.getExpressionPosition(call.Arguments[i])
				tc.addError(errors.E3001,
					fmt.Sprintf("arrays.%s: incompatible array element types - first array has %s, but argument %d has %s",
						funcName, firstElementType, i+1, argElementType),
					argLine, argCol)
			}
		}
	}
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

	// Additional key/value type validation (#593)
	tc.checkMapKeyValueTypeCompatibility(funcName, call, line, column)
}

// checkMapKeyValueTypeCompatibility validates that keys and values passed to map functions
// have types compatible with the map's declared key/value types
func (tc *TypeChecker) checkMapKeyValueTypeCompatibility(funcName string, call *ast.CallExpression, line, column int) {
	if len(call.Arguments) < 1 {
		return
	}

	// Get the map type from the first argument
	mapType, ok := tc.inferExpressionType(call.Arguments[0])
	if !ok || !tc.isMapType(mapType) {
		return
	}

	keyType := tc.extractMapKeyType(mapType)
	valueType := tc.extractMapValueType(mapType)

	switch funcName {
	case "has", "has_key", "delete", "remove":
		// Second arg is key
		if len(call.Arguments) < 2 {
			return
		}
		argType, ok := tc.inferExpressionType(call.Arguments[1])
		if !ok {
			return
		}
		if !tc.typesCompatible(keyType, argType) {
			argLine, argCol := tc.getExpressionPosition(call.Arguments[1])
			tc.addError(errors.E3001,
				fmt.Sprintf("maps.%s: key type mismatch - map has key type %s, but got %s",
					funcName, keyType, argType),
				argLine, argCol)
		}

	case "has_value":
		// Second arg is value
		if len(call.Arguments) < 2 {
			return
		}
		argType, ok := tc.inferExpressionType(call.Arguments[1])
		if !ok {
			return
		}
		if !tc.typesCompatible(valueType, argType) {
			argLine, argCol := tc.getExpressionPosition(call.Arguments[1])
			tc.addError(errors.E3001,
				fmt.Sprintf("maps.%s: value type mismatch - map has value type %s, but got %s",
					funcName, valueType, argType),
				argLine, argCol)
		}

	case "get":
		// Second arg is key, optional third arg is default value
		if len(call.Arguments) < 2 {
			return
		}
		keyArgType, ok := tc.inferExpressionType(call.Arguments[1])
		if ok && !tc.typesCompatible(keyType, keyArgType) {
			argLine, argCol := tc.getExpressionPosition(call.Arguments[1])
			tc.addError(errors.E3001,
				fmt.Sprintf("maps.%s: key type mismatch - map has key type %s, but got %s",
					funcName, keyType, keyArgType),
				argLine, argCol)
		}
		// Check default value type if provided
		if len(call.Arguments) >= 3 {
			defaultArgType, ok := tc.inferExpressionType(call.Arguments[2])
			if ok && !tc.typesCompatible(valueType, defaultArgType) {
				argLine, argCol := tc.getExpressionPosition(call.Arguments[2])
				tc.addError(errors.E3001,
					fmt.Sprintf("maps.%s: default value type mismatch - map has value type %s, but got %s",
						funcName, valueType, defaultArgType),
					argLine, argCol)
			}
		}

	case "set", "get_or_set":
		// Second arg is key, third arg is value
		if len(call.Arguments) < 3 {
			return
		}
		keyArgType, ok := tc.inferExpressionType(call.Arguments[1])
		if ok && !tc.typesCompatible(keyType, keyArgType) {
			argLine, argCol := tc.getExpressionPosition(call.Arguments[1])
			tc.addError(errors.E3001,
				fmt.Sprintf("maps.%s: key type mismatch - map has key type %s, but got %s",
					funcName, keyType, keyArgType),
				argLine, argCol)
		}
		valueArgType, ok := tc.inferExpressionType(call.Arguments[2])
		if ok && !tc.typesCompatible(valueType, valueArgType) {
			argLine, argCol := tc.getExpressionPosition(call.Arguments[2])
			tc.addError(errors.E3001,
				fmt.Sprintf("maps.%s: value type mismatch - map has value type %s, but got %s",
					funcName, valueType, valueArgType),
				argLine, argCol)
		}

	case "merge", "update", "equals":
		// All args are maps - check they have compatible key/value types
		for i := 1; i < len(call.Arguments); i++ {
			argType, ok := tc.inferExpressionType(call.Arguments[i])
			if !ok || !tc.isMapType(argType) {
				continue
			}
			argKeyType := tc.extractMapKeyType(argType)
			argValueType := tc.extractMapValueType(argType)
			if !tc.typesCompatible(keyType, argKeyType) {
				argLine, argCol := tc.getExpressionPosition(call.Arguments[i])
				tc.addError(errors.E3001,
					fmt.Sprintf("maps.%s: incompatible map key types - first map has %s, but argument %d has %s",
						funcName, keyType, i+1, argKeyType),
					argLine, argCol)
			}
			if !tc.typesCompatible(valueType, argValueType) {
				argLine, argCol := tc.getExpressionPosition(call.Arguments[i])
				tc.addError(errors.E3001,
					fmt.Sprintf("maps.%s: incompatible map value types - first map has %s, but argument %d has %s",
						funcName, valueType, i+1, argValueType),
					argLine, argCol)
			}
		}
	}
}

// checkStringsModuleCall validates strings module function calls
func (tc *TypeChecker) checkStringsModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// Single string arg
		"len":      {1, 1, []string{"string"}, "int"},
		"upper":    {1, 1, []string{"string"}, "string"},
		"lower":    {1, 1, []string{"string"}, "string"},
		"trim":     {1, 1, []string{"string"}, "string"},
		"to_int":   {1, 1, []string{"string"}, "int"},
		"to_float": {1, 1, []string{"string"}, "float"},
		"to_bool":  {1, 1, []string{"string"}, "bool"},

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

		// Format functions: format(format_string) or format(format_string, timestamp)
		"format": {1, 2, []string{"string", "int"}, "string"}, // format first, optional timestamp second
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

// checkIoModuleCall validates io module function calls
func (tc *TypeChecker) checkIoModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// File reading (1 path arg, returns tuple)
		"read_file":  {1, 1, []string{"string"}, "tuple"},
		"read_bytes": {1, 1, []string{"string"}, "tuple"},
		"read_lines": {1, 1, []string{"string"}, "tuple"},

		// File writing (2-3 args: path, content, optional perms)
		"write_file":  {2, 3, []string{"string", "string", "int"}, "tuple"},
		"write_bytes": {2, 3, []string{"string", "array", "int"}, "tuple"},
		"append_file": {2, 3, []string{"string", "string", "int"}, "tuple"},
		"append_line": {2, 3, []string{"string", "string", "int"}, "tuple"},

		// Path utilities
		"expand_path":    {1, 1, []string{"string"}, "string"},
		"path_join":      {1, -1, []string{"string"}, "string"},
		"path_base":      {1, 1, []string{"string"}, "string"},
		"path_dir":       {1, 1, []string{"string"}, "string"},
		"path_ext":       {1, 1, []string{"string"}, "string"},
		"path_abs":       {1, 1, []string{"string"}, "tuple"},
		"path_clean":     {1, 1, []string{"string"}, "string"},
		"path_separator": {0, 0, []string{}, "string"},

		// File checks (1 path arg, returns bool)
		"exists":     {1, 1, []string{"string"}, "bool"},
		"is_file":    {1, 1, []string{"string"}, "bool"},
		"is_dir":     {1, 1, []string{"string"}, "bool"},
		"is_symlink": {1, 1, []string{"string"}, "bool"},

		// File operations
		"remove":     {1, 1, []string{"string"}, "tuple"},
		"remove_dir": {1, 1, []string{"string"}, "tuple"},
		"remove_all": {1, 1, []string{"string"}, "tuple"},
		"rename":     {2, 2, []string{"string", "string"}, "tuple"},
		"copy":       {2, 3, []string{"string", "string", "int"}, "tuple"},

		// Directory operations
		"mkdir":     {1, 2, []string{"string", "int"}, "tuple"},
		"mkdir_all": {1, 2, []string{"string", "int"}, "tuple"},
		"read_dir":  {1, 1, []string{"string"}, "tuple"},

		// File metadata
		"file_size":     {1, 1, []string{"string"}, "tuple"},
		"file_mod_time": {1, 1, []string{"string"}, "tuple"},

		// File handle operations
		"open":        {1, 3, []string{"string", "int", "int"}, "tuple"},
		"read":        {2, 2, []string{"any", "int"}, "tuple"},
		"read_all":    {1, 1, []string{"any"}, "tuple"},
		"read_string": {2, 2, []string{"any", "int"}, "tuple"},
		"write":       {2, 2, []string{"any", "any"}, "tuple"},
		"seek":        {3, 3, []string{"any", "int", "int"}, "tuple"},
		"tell":        {1, 1, []string{"any"}, "tuple"},
		"flush":       {1, 1, []string{"any"}, "tuple"},
		"close":       {1, 1, []string{"any"}, "tuple"},

		// Filesystem utilities
		"glob": {1, 1, []string{"string"}, "tuple"},
		"walk": {1, 1, []string{"string"}, "tuple"},

		// Constants (no args)
		"READ_ONLY":    {0, 0, []string{}, "int"},
		"WRITE_ONLY":   {0, 0, []string{}, "int"},
		"READ_WRITE":   {0, 0, []string{}, "int"},
		"APPEND":       {0, 0, []string{}, "int"},
		"CREATE":       {0, 0, []string{}, "int"},
		"TRUNCATE":     {0, 0, []string{}, "int"},
		"EXCLUSIVE":    {0, 0, []string{}, "int"},
		"SEEK_START":   {0, 0, []string{}, "int"},
		"SEEK_CURRENT": {0, 0, []string{}, "int"},
		"SEEK_END":     {0, 0, []string{}, "int"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("io", funcName, call, sig, line, column)
}

// checkOsModuleCall validates os module function calls
func (tc *TypeChecker) checkOsModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// Environment variables
		"get_env":   {1, 1, []string{"string"}, "any"},
		"set_env":   {2, 2, []string{"string", "string"}, "tuple"},
		"unset_env": {1, 1, []string{"string"}, "tuple"},
		"env":       {0, 0, []string{}, "map"},
		"args":      {0, 0, []string{}, "array"},

		// Process / System
		"exit":     {0, 1, []string{"int"}, "void"},
		"cwd":      {0, 0, []string{}, "tuple"},
		"chdir":    {1, 1, []string{"string"}, "tuple"},
		"hostname": {0, 0, []string{}, "tuple"},
		"username": {0, 0, []string{}, "tuple"},
		"home_dir": {0, 0, []string{}, "tuple"},
		"temp_dir": {0, 0, []string{}, "string"},
		"pid":      {0, 0, []string{}, "int"},
		"ppid":     {0, 0, []string{}, "int"},

		// Platform detection
		"platform":       {0, 0, []string{}, "string"},
		"arch":           {0, 0, []string{}, "string"},
		"is_windows":     {0, 0, []string{}, "bool"},
		"is_linux":       {0, 0, []string{}, "bool"},
		"is_macos":       {0, 0, []string{}, "bool"},
		"num_cpu":        {0, 0, []string{}, "int"},
		"line_separator": {0, 0, []string{}, "string"},
		"dev_null":       {0, 0, []string{}, "string"},

		// Command execution
		"exec":        {1, 1, []string{"string"}, "tuple"},
		"exec_output": {1, 1, []string{"string"}, "tuple"},

		// Constants
		"MAC_OS":     {0, 0, []string{}, "int"},
		"LINUX":      {0, 0, []string{}, "int"},
		"WINDOWS":    {0, 0, []string{}, "int"},
		"CURRENT_OS": {0, 0, []string{}, "int"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("os", funcName, call, sig, line, column)
}

// checkRandomModuleCall validates random module function calls
func (tc *TypeChecker) checkRandomModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// random.float() or random.float(min, max)
		"float": {0, 2, []string{"numeric", "numeric"}, "float"},
		// random.int(max) or random.int(min, max)
		"int": {1, 2, []string{"numeric", "numeric"}, "int"},
		// random.bool()
		"bool": {0, 0, []string{}, "bool"},
		// random.byte()
		"byte": {0, 0, []string{}, "byte"},
		// random.char() or random.char(min, max)
		"char": {0, 2, []string{"any", "any"}, "char"},
		// random.choice(array)
		"choice": {1, 1, []string{"array"}, "any"},
		// random.shuffle(array)
		"shuffle": {1, 1, []string{"array"}, "array"},
		// random.sample(array, n)
		"sample": {2, 2, []string{"array", "int"}, "array"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("random", funcName, call, sig, line, column)
}

// checkJsonModuleCall validates json module function calls
func (tc *TypeChecker) checkJsonModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// json.encode(value)
		"encode": {1, 1, []string{"any"}, "tuple"},
		// json.decode(text, Type) - Type argument is REQUIRED
		"decode": {2, 2, []string{"string", "any"}, "tuple"},
		// json.pretty(value, indent)
		"pretty": {2, 2, []string{"any", "string"}, "tuple"},
		// json.is_valid(text)
		"is_valid": {1, 1, []string{"string"}, "bool"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	// Special error message for json.decode missing type argument
	if funcName == "decode" && len(call.Arguments) == 1 {
		tc.addError(
			errors.E13004,
			"json.decode() requires a type argument: json.decode(text, Type)",
			line,
			column,
		)
		return
	}

	tc.validateStdlibCall("json", funcName, call, sig, line, column)
}

// checkBytesModuleCall validates bytes module function calls
func (tc *TypeChecker) checkBytesModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// Creation functions
		"from_array":  {1, 1, []string{"array"}, "array"},
		"from_string": {1, 1, []string{"string"}, "array"},
		"from_hex":    {1, 1, []string{"string"}, "tuple"},
		"from_base64": {1, 1, []string{"string"}, "tuple"},

		// Conversion functions
		"to_string":    {1, 1, []string{"array"}, "string"},
		"to_array":     {1, 1, []string{"array"}, "array"},
		"to_hex":       {1, 1, []string{"array"}, "string"},
		"to_hex_upper": {1, 1, []string{"array"}, "string"},
		"to_base64":    {1, 1, []string{"array"}, "string"},

		// Slicing and combining
		"slice":  {2, 3, []string{"array", "int", "int"}, "array"},
		"concat": {2, -1, []string{"array"}, "array"},
		"join":   {2, 2, []string{"array", "array"}, "array"},
		"split":  {2, 2, []string{"array", "array"}, "array"},

		// Search functions
		"contains":   {2, 2, []string{"array", "array"}, "bool"},
		"index":      {2, 2, []string{"array", "array"}, "int"},
		"last_index": {2, 2, []string{"array", "array"}, "int"},
		"count":      {2, 2, []string{"array", "array"}, "int"},

		// Comparison functions
		"compare":     {2, 2, []string{"array", "array"}, "int"},
		"equals":      {2, 2, []string{"array", "array"}, "bool"},
		"is_empty":    {1, 1, []string{"array"}, "bool"},
		"starts_with": {2, 2, []string{"array", "array"}, "bool"},
		"ends_with":   {2, 2, []string{"array", "array"}, "bool"},

		// Transformation functions
		"reverse":    {1, 1, []string{"array"}, "array"},
		"repeat":     {2, 2, []string{"array", "int"}, "array"},
		"replace":    {3, 3, []string{"array", "array", "array"}, "array"},
		"replace_n":  {4, 4, []string{"array", "array", "array", "int"}, "array"},
		"trim":       {2, 2, []string{"array", "array"}, "array"},
		"trim_left":  {2, 2, []string{"array", "array"}, "array"},
		"trim_right": {2, 2, []string{"array", "array"}, "array"},
		"pad_left":   {3, 3, []string{"array", "int", "int"}, "array"},
		"pad_right":  {3, 3, []string{"array", "int", "int"}, "array"},

		// Bitwise functions
		"and": {2, 2, []string{"array", "array"}, "array"},
		"or":  {2, 2, []string{"array", "array"}, "array"},
		"xor": {2, 2, []string{"array", "array"}, "array"},
		"not": {1, 1, []string{"array"}, "array"},

		// Utility functions
		"fill": {2, 2, []string{"array", "int"}, "array"},
		"copy": {1, 1, []string{"array"}, "array"},
		"zero": {1, 1, []string{"array"}, "array"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("bytes", funcName, call, sig, line, column)
}

// checkBinaryModuleCall validates binary module function calls
func (tc *TypeChecker) checkBinaryModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// 8-bit (no endianness)
		"encode_i8": {1, 1, []string{"int"}, "tuple"},
		"decode_i8": {1, 1, []string{"array"}, "tuple"},
		"encode_u8": {1, 1, []string{"int"}, "tuple"},
		"decode_u8": {1, 1, []string{"array"}, "tuple"},

		// 16-bit little endian
		"encode_i16_to_little_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_i16_from_little_endian": {1, 1, []string{"array"}, "tuple"},
		"encode_u16_to_little_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_u16_from_little_endian": {1, 1, []string{"array"}, "tuple"},

		// 16-bit big endian
		"encode_i16_to_big_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_i16_from_big_endian": {1, 1, []string{"array"}, "tuple"},
		"encode_u16_to_big_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_u16_from_big_endian": {1, 1, []string{"array"}, "tuple"},

		// 32-bit little endian
		"encode_i32_to_little_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_i32_from_little_endian": {1, 1, []string{"array"}, "tuple"},
		"encode_u32_to_little_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_u32_from_little_endian": {1, 1, []string{"array"}, "tuple"},

		// 32-bit big endian
		"encode_i32_to_big_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_i32_from_big_endian": {1, 1, []string{"array"}, "tuple"},
		"encode_u32_to_big_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_u32_from_big_endian": {1, 1, []string{"array"}, "tuple"},

		// 64-bit little endian
		"encode_i64_to_little_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_i64_from_little_endian": {1, 1, []string{"array"}, "tuple"},
		"encode_u64_to_little_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_u64_from_little_endian": {1, 1, []string{"array"}, "tuple"},

		// 64-bit big endian
		"encode_i64_to_big_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_i64_from_big_endian": {1, 1, []string{"array"}, "tuple"},
		"encode_u64_to_big_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_u64_from_big_endian": {1, 1, []string{"array"}, "tuple"},

		// 128-bit little endian
		"encode_i128_to_little_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_i128_from_little_endian": {1, 1, []string{"array"}, "tuple"},
		"encode_u128_to_little_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_u128_from_little_endian": {1, 1, []string{"array"}, "tuple"},

		// 128-bit big endian
		"encode_i128_to_big_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_i128_from_big_endian": {1, 1, []string{"array"}, "tuple"},
		"encode_u128_to_big_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_u128_from_big_endian": {1, 1, []string{"array"}, "tuple"},

		// 256-bit little endian
		"encode_i256_to_little_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_i256_from_little_endian": {1, 1, []string{"array"}, "tuple"},
		"encode_u256_to_little_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_u256_from_little_endian": {1, 1, []string{"array"}, "tuple"},

		// 256-bit big endian
		"encode_i256_to_big_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_i256_from_big_endian": {1, 1, []string{"array"}, "tuple"},
		"encode_u256_to_big_endian":   {1, 1, []string{"int"}, "tuple"},
		"decode_u256_from_big_endian": {1, 1, []string{"array"}, "tuple"},

		// Float little endian
		"encode_f32_to_little_endian":   {1, 1, []string{"float"}, "tuple"},
		"decode_f32_from_little_endian": {1, 1, []string{"array"}, "tuple"},
		"encode_f64_to_little_endian":   {1, 1, []string{"float"}, "tuple"},
		"decode_f64_from_little_endian": {1, 1, []string{"array"}, "tuple"},

		// Float big endian
		"encode_f32_to_big_endian":   {1, 1, []string{"float"}, "tuple"},
		"decode_f32_from_big_endian": {1, 1, []string{"array"}, "tuple"},
		"encode_f64_to_big_endian":   {1, 1, []string{"float"}, "tuple"},
		"decode_f64_from_big_endian": {1, 1, []string{"array"}, "tuple"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("binary", funcName, call, sig, line, column)
}

func (tc *TypeChecker) checkDBModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// Database management
		"open":  {1, 1, []string{"string"}, "tuple"},
		"close": {1, 1, []string{"Database"}, "nil"},
		"save":  {1, 1, []string{"Database"}, "nil"},

		// Database operations
		"set":    {3, 3, []string{"Database", "string", "string"}, "nil"},
		"get":    {2, 2, []string{"Database", "string"}, "tuple"},
		"delete": {2, 2, []string{"Database", "string"}, "bool"},
		"has":    {2, 2, []string{"Database", "string"}, "bool"},
		"keys":   {1, 1, []string{"Database"}, "[string]"},
		"prefix": {2, 2, []string{"Database", "string"}, "[string]"},
		"count":  {1, 1, []string{"Database"}, "int"},
		"clear":  {1, 1, []string{"Database"}, "nil"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("db", funcName, call, sig, line, column)
}

// checkUuidModuleCall validates uuid module function calls
func (tc *TypeChecker) checkUuidModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// uuid.create() - generates a new UUID v4
		"create": {0, 0, []string{}, "string"},
		// uuid.create_compact() - generates a new UUID v4 without hyphens
		"create_compact": {0, 0, []string{}, "string"},
		// uuid.is_valid(str) - checks if a string is a valid UUID
		"is_valid": {1, 1, []string{"string"}, "bool"},
		// uuid.NIL - the nil UUID constant
		"NIL": {0, 0, []string{}, "string"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("uuid", funcName, call, sig, line, column)
}

// checkEncodingModuleCall validates encoding module function calls
func (tc *TypeChecker) checkEncodingModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// encoding.base64_encode(data) - encodes to base64
		"base64_encode": {1, 1, []string{"string"}, "string"},
		// encoding.base64_decode(data) - decodes from base64
		"base64_decode": {1, 1, []string{"string"}, "tuple"},
		// encoding.hex_encode(data) - encodes to hex
		"hex_encode": {1, 1, []string{"string"}, "string"},
		// encoding.hex_decode(data) - decodes from hex
		"hex_decode": {1, 1, []string{"string"}, "tuple"},
		// encoding.url_encode(data) - URL percent-encodes
		"url_encode": {1, 1, []string{"string"}, "string"},
		// encoding.url_decode(data) - decodes URL encoding
		"url_decode": {1, 1, []string{"string"}, "tuple"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("encoding", funcName, call, sig, line, column)
}

func (tc *TypeChecker) checkCryptoModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		// crypto.sha256(data) - SHA-256 hash
		"sha256": {1, 1, []string{"string"}, "string"},
		// crypto.sha512(data) - SHA-512 hash
		"sha512": {1, 1, []string{"string"}, "string"},
		// crypto.md5(data) - MD5 hash (legacy)
		"md5": {1, 1, []string{"string"}, "string"},
		// crypto.random_bytes(length) - secure random bytes
		"random_bytes": {1, 1, []string{"int"}, "array"},
		// crypto.random_hex(length) - secure random hex string
		"random_hex": {1, 1, []string{"int"}, "string"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("crypto", funcName, call, sig, line, column)
}

func (tc *TypeChecker) checkHttpModuleCall(funcName string, call *ast.CallExpression, line, column int) {
	signatures := map[string]StdlibFuncSig{
		"get":    {1, 1, []string{"string"}, "tuple"},
		"post":   {2, 2, []string{"string", "string"}, "tuple"},
		"put":    {2, 2, []string{"string", "string"}, "tuple"},
		"delete": {1, 1, []string{"string"}, "tuple"},
		"patch":  {2, 2, []string{"string", "string"}, "tuple"},

		"request": {5, 5, []string{"string", "string", "string", "map[string:string]", "int"}, "tuple"},

		"encode_url":  {1, 1, []string{"string"}, "string"},
		"decode_url":  {1, 1, []string{"string"}, "tuple"},
		"build_query": {1, 1, []string{"map[string:string]"}, "string"},

		"json_body": {1, 1, []string{"map"}, "string"},
	}

	sig, exists := signatures[funcName]
	if !exists {
		return
	}

	tc.validateStdlibCall("http", funcName, call, sig, line, column)
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
	// Unknown type (empty string) - skip validation, let runtime catch it
	if actual == "" {
		return true
	}

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
