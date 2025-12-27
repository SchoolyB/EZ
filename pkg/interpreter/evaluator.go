package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/debugger"
	"github.com/marshallburns/ez/pkg/errors"
)

// Type bounds for integer types
var (
	// Signed integer bounds
	minInt8   = big.NewInt(math.MinInt8)
	maxInt8   = big.NewInt(math.MaxInt8)
	minInt16  = big.NewInt(math.MinInt16)
	maxInt16  = big.NewInt(math.MaxInt16)
	minInt32  = big.NewInt(math.MinInt32)
	maxInt32  = big.NewInt(math.MaxInt32)
	minInt64  = big.NewInt(math.MinInt64)
	maxInt64  = big.NewInt(math.MaxInt64)
	minInt128 = new(big.Int)
	maxInt128 = new(big.Int)
	minInt256 = new(big.Int)
	maxInt256 = new(big.Int)

	// Unsigned integer bounds
	maxUint8   = big.NewInt(math.MaxUint8)
	maxUint16  = big.NewInt(math.MaxUint16)
	maxUint32  = big.NewInt(math.MaxUint32)
	maxUint64  = new(big.Int).SetUint64(math.MaxUint64)
	maxUint128 = new(big.Int)
	maxUint256 = new(big.Int)

	zero = big.NewInt(0)
	one  = big.NewInt(1)
)

func init() {
	// i128: -2^127 to 2^127-1
	minInt128.Lsh(big.NewInt(-1), 127)
	maxInt128.Lsh(big.NewInt(1), 127)
	maxInt128.Sub(maxInt128, one)

	// i256: -2^255 to 2^255-1
	minInt256.Lsh(big.NewInt(-1), 255)
	maxInt256.Lsh(big.NewInt(1), 255)
	maxInt256.Sub(maxInt256, one)

	// u128: 0 to 2^128-1
	maxUint128.Lsh(big.NewInt(1), 128)
	maxUint128.Sub(maxUint128, one)

	// u256: 0 to 2^256-1
	maxUint256.Lsh(big.NewInt(1), 256)
	maxUint256.Sub(maxUint256, one)
}

// getTypeBounds returns the min and max values for a given integer type
func getTypeBounds(typeName string) (min, max *big.Int) {
	switch typeName {
	case "i8":
		return minInt8, maxInt8
	case "i16":
		return minInt16, maxInt16
	case "i32":
		return minInt32, maxInt32
	case "i64", "int", "":
		return minInt64, maxInt64
	case "i128":
		return minInt128, maxInt128
	case "i256":
		return minInt256, maxInt256
	case "u8":
		return zero, maxUint8
	case "u16":
		return zero, maxUint16
	case "u32":
		return zero, maxUint32
	case "u64", "uint":
		return zero, maxUint64
	case "u128":
		return zero, maxUint128
	case "u256":
		return zero, maxUint256
	default:
		// Default to int64 range for unknown types
		return minInt64, maxInt64
	}
}

// checkOverflow checks if a value is within bounds for a given type
func checkOverflow(result *big.Int, typeName string) bool {
	min, max := getTypeBounds(typeName)
	return result.Cmp(min) < 0 || result.Cmp(max) > 0
}

// getTypeRangeName returns a human-readable name for the type's range
func getTypeRangeName(typeName string) string {
	switch typeName {
	case "i8", "i16", "i32", "i64", "i128", "i256":
		return typeName
	case "u8", "u16", "u32", "u64", "u128", "u256":
		return typeName
	case "int", "":
		return "int64"
	case "uint":
		return "uint64"
	default:
		return "int64"
	}
}

var (
	NIL   = &Nil{}
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
)

// Call stack depth tracking for recursion limit
const MAX_CALL_DEPTH = 10000

var callDepth int

// EvalContext holds context for evaluation including the module loader
type EvalContext struct {
	Loader      *ModuleLoader
	CurrentFile string // Current file being evaluated (for relative imports)
}

// Global eval context (set when running a program)
var globalEvalContext *EvalContext

// validModules lists all available standard library modules
var validModules = map[string]bool{
	"std":     true, // Standard I/O functions (println, print, read_int)
	"math":    true, // Math functions
	"strings": true, // String utilities
	"arrays":  true, // Array utilities
	"maps":    true, // Map utilities
	"time":    true, // Time functions
	"io":      true, // File system and I/O operations
	"os":      true, // Operating system and environment
	"bytes":   true, // Binary data operations
	"random":  true, // Random number generation
	"json":    true, // JSON encoding/decoding
	"binary":  true, // Binary encoding/decoding for integers and floats
	"db":      true, // Simple key-value database
}

// isValidModule checks if a module name is valid (either standard library or user-created)
func isValidModule(moduleName string) bool {
	// Check standard library modules
	if validModules[moduleName] {
		return true
	}

	return false
}

// suggestModule returns a suggestion for a similar module name, or empty string if none found.
func suggestModule(invalidName string) string {
	// Check for common typos/variations
	suggestions := map[string]string{
		"string":   "strings",
		"array":    "arrays",
		"map":      "maps",
		"rand":     "random",
		"file":     "io",
		"files":    "io",
		"fs":       "io",
		"env":      "os",
		"system":   "os",
		"byte":     "bytes",
		"datetime": "time",
		"date":     "time",
	}

	if suggestion, ok := suggestions[invalidName]; ok {
		return suggestion
	}

	// Check for close matches using simple prefix/suffix matching
	for validName := range validModules {
		// Check if input is a prefix of a valid module (e.g., "str" -> "strings")
		if len(invalidName) >= 3 && strings.HasPrefix(validName, invalidName) {
			return validName
		}
		// Check if valid module is a prefix of input (e.g., "stringss" -> "strings")
		if len(validName) >= 3 && strings.HasPrefix(invalidName, validName) {
			return validName
		}
	}

	return ""
}

// convertVisibility converts AST visibility to object visibility
func convertVisibility(vis ast.Visibility) Visibility {
	switch vis {
	case ast.VisibilityPrivate:
		return VisibilityPrivate
	default:
		return VisibilityPublic
	}
}

// extractModuleName extracts the module name from a file path
// e.g., "./server" -> "server", "../utils" -> "utils", "./src/networking" -> "networking"
func extractModuleName(path string) string {
	// Remove leading ./ or ../
	for strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		if strings.HasPrefix(path, "./") {
			path = path[2:]
		} else if strings.HasPrefix(path, "../") {
			path = path[3:]
		}
	}
	// Get the last component
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

// SetEvalContext sets the global evaluation context
func SetEvalContext(ctx *EvalContext) {
	globalEvalContext = ctx
}

// GetEvalContext returns the global evaluation context
func GetEvalContext() *EvalContext {
	return globalEvalContext
}

// EvalWithContext evaluates a program with a given context
func EvalWithContext(node ast.Node, env *Environment, ctx *EvalContext) Object {
	oldCtx := globalEvalContext
	globalEvalContext = ctx
	result := Eval(node, env)
	globalEvalContext = oldCtx
	return result
}

// loadUserModule loads a user module from a file path and returns a ModuleObject
func loadUserModule(importPath string, line, column int, env *Environment) (*ModuleObject, Object) {
	if globalEvalContext == nil || globalEvalContext.Loader == nil {
		return nil, newErrorWithLocation("E6001", line, column, "module loader not initialized")
	}

	// Set the current file for relative path resolution
	globalEvalContext.Loader.SetCurrentFile(globalEvalContext.CurrentFile)

	// Load the module (this handles parsing and caching)
	mod, err := globalEvalContext.Loader.Load(importPath)
	if err != nil {
		// Check if this is a ModuleError with rich parse errors
		if modErr, ok := err.(*ModuleError); ok && modErr.EZErrors != nil && modErr.EZErrors.HasErrors() {
			// Return the formatted errors directly without wrapping
			return nil, &Error{Message: modErr.Error(), PreFormatted: true}
		}
		// Return error with location info for proper formatting
		if modErr, ok := err.(*ModuleError); ok {
			return nil, newErrorWithLocation(modErr.Code, line, column, modErr.Message)
		}
		return nil, newErrorWithLocation("E6001", line, column, err.Error())
	}

	// If module is already fully loaded, return cached ModuleObject
	if mod.State == ModuleLoaded && mod.ModuleObj != nil {
		return mod.ModuleObj, nil
	}

	// Check for circular import - if module already has a ModuleObj and is loading,
	// another call to loadUserModule is already evaluating this module.
	// Return the existing ModuleObj which will be populated when that call completes.
	if mod.State == ModuleLoading && mod.ModuleObj != nil {
		return mod.ModuleObj, nil
	}

	// We are the first to evaluate this module.
	// Create a new environment for the module
	moduleEnv := NewEnvironment()
	mod.Env = moduleEnv

	// Create ModuleObject early so circular imports can reference it
	// Since we're about to evaluate, we create the ModuleObj here
	mod.ModuleObj = &ModuleObject{
		Name:    mod.Name,
		Exports: make(map[string]Object),
	}

	// Save the current file and set the module file as current
	oldFile := globalEvalContext.CurrentFile
	if len(mod.Files) > 0 {
		globalEvalContext.CurrentFile = mod.Files[0]
	}

	// Evaluate the module's AST
	result := Eval(mod.AST, moduleEnv)

	// Restore the current file
	globalEvalContext.CurrentFile = oldFile

	if isError(result) {
		return nil, result
	}

	// Export only public symbols from the module environment
	for name, obj := range moduleEnv.GetPublicBindings() {
		mod.ModuleObj.Exports[name] = obj
	}

	// Export struct definitions from the module
	mod.ModuleObj.StructDefs = moduleEnv.GetPublicStructDefs()

	// Mark module as fully loaded
	mod.State = ModuleLoaded

	return mod.ModuleObj, nil
}

func Eval(node ast.Node, env *Environment) Object {
	// Debug hook: before evaluation (only for steppable statements)
	if dbg := debugger.GetGlobalDebugger(); dbg != nil && dbg.IsEnabled() {
		if isSteppableStatement(node) {
			dbg.BeforeEval(node, env)
		}
	}

	switch node := node.(type) {
	// Program
	case *ast.Program:
		return evalProgram(node, env)

	// Statements
	case *ast.ExpressionStatement:
		result := Eval(node.Expression, env)
		if isError(result) {
			return result
		}
		// Check for uncaptured return values from function calls
		if call, ok := node.Expression.(*ast.CallExpression); ok {
			if fn, ok := result.(*ReturnValue); ok && len(fn.Values) > 0 {
				return newErrorWithLocation("E4009", call.Token.Line, call.Token.Column,
					"return value from function not used (use _ to discard)")
			}
			// Also check if the function has declared return types but result is not NIL
			if result != NIL {
				// Check if it's a user function with return types
				if fnObj := getFunctionObject(call, env); fnObj != nil && len(fnObj.ReturnTypes) > 0 {
					return newErrorWithLocation("E4009", call.Token.Line, call.Token.Column,
						"return value from function not used (use _ to discard)")
				}
			}
		}
		return result

	case *ast.VariableDeclaration:
		return evalVariableDeclaration(node, env)

	case *ast.AssignmentStatement:
		return evalAssignment(node, env)

	case *ast.ReturnStatement:
		return evalReturn(node, env)

	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	case *ast.IfStatement:
		return evalIfStatement(node, env)

	case *ast.WhenStatement:
		return evalWhenStatement(node, env)

	case *ast.WhileStatement:
		return evalWhileStatement(node, env)

	case *ast.LoopStatement:
		return evalLoopStatement(node, env)

	case *ast.ForStatement:
		return evalForStatement(node, env)

	case *ast.ForEachStatement:
		return evalForEachStatement(node, env)

	case *ast.BreakStatement:
		if !env.InLoop() {
			return newErrorWithLocation("E5009", node.Token.Line, node.Token.Column,
				"break statement outside loop")
		}
		return &Break{}

	case *ast.ContinueStatement:
		if !env.InLoop() {
			return newErrorWithLocation("E5010", node.Token.Line, node.Token.Column,
				"continue statement outside loop")
		}
		return &Continue{}

	case *ast.FunctionDeclaration:
		return evalFunctionDeclaration(node, env)

	case *ast.StructDeclaration:
		// Register the struct type definition with visibility
		fields := make(map[string]string)
		tags := make(map[string]StructFieldTags)
		for _, field := range node.Fields {
			fields[field.Name.Value] = field.TypeName
			if strings.HasPrefix(field.Tag, "json:\"") {
				jsonTag := &JSONTag{Name: field.Name.Value}
				optionString, _ := strings.CutPrefix(field.Tag, "json:\"")
				optionString, _ = strings.CutSuffix(optionString, "\"")
				options := strings.Split(optionString, ",")
				switch options[0] {
				case "-":
					jsonTag.Ignore = true
				default:
					jsonTag.Name = options[0]
					for _, opt := range options[1:] {
						switch opt {
						case "omitempty":
							jsonTag.OmitEmpty = true
						case "string":
							jsonTag.EncodeAsString = true
						}
					}
				}
				tags[field.Name.Value] = jsonTag
			} else if field.Tag == "" {
				tags[field.Name.Value] = &EmptyTag{}
			}
		}
		vis := convertVisibility(node.Visibility)
		env.RegisterStructDefWithVisibility(node.Name.Value, &StructDef{
			Name:      node.Name.Value,
			Fields:    fields,
			FieldTags: tags,
		}, vis)
		return NIL

	case *ast.EnumDeclaration:
		return evalEnumDeclaration(node, env)

	case *ast.ImportStatement:
		// Register the imported module(s) with their aliases
		// The alias is what's used in code (e.g., str.upper())
		// The module is the actual library (e.g., strings)

		// Handle multiple imports (new comma-separated syntax)
		if len(node.Imports) > 0 {
			for _, item := range node.Imports {
				alias := item.Alias
				if alias == "" {
					if item.Module != "" {
						alias = item.Module
					} else {
						// Extract from path
						alias = extractModuleName(item.Path)
					}
				}

				if item.IsStdlib {
					// Standard library import
					if !isValidModule(item.Module) {
						if suggestion := suggestModule(item.Module); suggestion != "" {
							return newErrorWithLocation("E6002", node.Token.Line, node.Token.Column,
								"module '%s' not found. Did you mean @%s?", item.Module, suggestion)
						}
						return newErrorWithLocation("E6002", node.Token.Line, node.Token.Column,
							"module '%s' not found", item.Module)
					}
					env.Import(alias, item.Module)
					// Dual-name access: also register with original module name if alias differs
					if alias != item.Module {
						env.Import(item.Module, item.Module)
					}
				} else if item.Path != "" {
					// User module import - load the module
					moduleObj, loadErr := loadUserModule(item.Path, node.Token.Line, node.Token.Column, env)
					if loadErr != nil {
						return loadErr
					}
					if moduleObj == nil {
						return newErrorWithLocation("E6001", node.Token.Line, node.Token.Column,
							"failed to load module '%s'", item.Path)
					}
					// Register the module object so it can be accessed via alias.function()
					env.RegisterModule(alias, moduleObj)
					// Dual-name access: also register with original module name if alias differs
					originalName := extractModuleName(item.Path)
					if alias != originalName {
						env.RegisterModule(originalName, moduleObj)
					}
				}

				// Handle auto-use (import & use syntax)
				if node.AutoUse {
					env.Use(alias)
				}
			}
		} else {
			// Backward compatibility: handle single import using old fields
			if !isValidModule(node.Module) {
				if suggestion := suggestModule(node.Module); suggestion != "" {
					return newErrorWithLocation("E6002", node.Token.Line, node.Token.Column,
						"module '%s' not found. Did you mean @%s?", node.Module, suggestion)
				}
				return newErrorWithLocation("E6002", node.Token.Line, node.Token.Column,
					"module '%s' not found", node.Module)
			}

			alias := node.Alias
			if alias == "" {
				alias = node.Module
			}
			env.Import(alias, node.Module)

			// Handle auto-use
			if node.AutoUse {
				env.Use(alias)
			}
		}
		return NIL

	case *ast.UsingStatement:
		// Bring the module(s) functions into scope (function-scoped using)
		for _, module := range node.Modules {
			alias := module.Value
			// Verify the module was imported (check both stdlib and user modules)
			_, isStdlib := env.GetImport(alias)
			_, isUserModule := env.GetModule(alias)
			if !isStdlib && !isUserModule {
				return newErrorWithLocation("E6004", node.Token.Line, node.Token.Column,
					"cannot use '%s': module not imported", alias)
			}
			env.Use(alias)
		}
		return NIL

	// Expressions
	case *ast.IntegerValue:
		return &Integer{Value: new(big.Int).Set(node.Value)}

	case *ast.FloatValue:
		return &Float{Value: node.Value}

	case *ast.StringValue:
		return &String{Value: node.Value, Mutable: true}

	case *ast.InterpolatedString:
		return evalInterpolatedString(node, env)

	case *ast.CharValue:
		return &Char{Value: node.Value}

	case *ast.BooleanValue:
		if node.Value {
			return TRUE
		}
		return FALSE

	case *ast.NilValue:
		return NIL

	case *ast.ArrayValue:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &Array{Elements: elements, Mutable: true}

	case *ast.MapValue:
		return evalMapLiteral(node, env)

	case *ast.StructValue:
		return evalStructValue(node, env)

	case *ast.Label:
		return evalIdentifier(node, env)

	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		// Short-circuit evaluation for && and ||
		if node.Operator == "&&" {
			if !isTruthy(left) {
				return FALSE // Left is false, don't evaluate right
			}
			right := Eval(node.Right, env)
			if isError(right) {
				return right
			}
			return nativeBoolToBooleanObject(isTruthy(right))
		}
		if node.Operator == "||" {
			if isTruthy(left) {
				return TRUE // Left is true, don't evaluate right
			}
			right := Eval(node.Right, env)
			if isError(right) {
				return right
			}
			return nativeBoolToBooleanObject(isTruthy(right))
		}

		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right, node.Token.Line, node.Token.Column)

	case *ast.PostfixExpression:
		return evalPostfixExpression(node, env)

	case *ast.CallExpression:
		return evalCallExpression(node, env)

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}

		// Handle map indexing first (maps can use non-integer keys)
		if mapObj, ok := left.(*Map); ok {
			// Validate that the key is hashable
			if _, hashOk := HashKey(index); !hashOk {
				return newErrorWithLocation("E12001", node.Token.Line, node.Token.Column,
					"unusable as map key: %s", index.Type())
			}
			value, exists := mapObj.Get(index)
			if !exists {
				// Build helpful error message with available keys
				availableKeys := make([]string, len(mapObj.Pairs))
				for i, pair := range mapObj.Pairs {
					availableKeys[i] = pair.Key.Inspect()
				}
				keyList := ""
				if len(availableKeys) > 0 {
					keyList = fmt.Sprintf("\n\nAvailable keys: %v", availableKeys)
				}
				return newErrorWithLocation("E12003", node.Token.Line, node.Token.Column,
					"key %s not found in map%s", index.Inspect(), keyList)
			}
			return value
		}

		// For arrays and strings, index must be an integer
		idx, ok := index.(*Integer)
		if !ok {
			return newErrorWithLocation("E9003", node.Token.Line, node.Token.Column,
				"index must be an integer, got %s", index.Type())
		}

		switch obj := left.(type) {
		case *Array:
			arrLen := big.NewInt(int64(len(obj.Elements)))
			if idx.Value.Sign() < 0 || idx.Value.Cmp(arrLen) >= 0 {
				if arrLen.Sign() == 0 {
					return newErrorWithLocation("E9004", node.Token.Line, node.Token.Column,
						"index out of bounds: array is empty (length 0)\n\n"+
							"Attempted to access index %s, but array has no elements\n"+
							"Hint: Use arrays.append() to add elements before accessing by index", idx.Value.String())
				}
				return newErrorWithLocation("E9001", node.Token.Line, node.Token.Column,
					"index out of bounds: attempted to access index %s, but valid range is 0-%d",
					idx.Value.String(), arrLen.Int64()-1)
			}
			return obj.Elements[idx.Value.Int64()]

		case *String:
			// Convert to runes for proper UTF-8 character indexing
			runes := []rune(obj.Value)
			strLen := big.NewInt(int64(len(runes)))
			if idx.Value.Sign() < 0 || idx.Value.Cmp(strLen) >= 0 {
				if strLen.Sign() == 0 {
					return newErrorWithLocation("E10004", node.Token.Line, node.Token.Column,
						"index out of bounds: string is empty (length 0)\n\n"+
							"Attempted to access index %s", idx.Value.String())
				}
				return newErrorWithLocation("E10003", node.Token.Line, node.Token.Column,
					"index out of bounds: attempted to access index %s, but valid range is 0-%d",
					idx.Value.String(), strLen.Int64()-1)
			}
			return &Char{Value: runes[idx.Value.Int64()]}

		default:
			return newErrorWithLocation("E5015", node.Token.Line, node.Token.Column,
				"index operator not supported for %s", left.Type())
		}

	case *ast.MemberExpression:
		return evalMemberExpression(node, env)

	case *ast.NewExpression:
		return evalNewExpression(node, env)

	case *ast.RangeExpression:
		return evalRangeExpression(node, env)

	case *ast.CastExpression:
		return evalCastExpression(node, env)
	}

	return newError("unknown node type: %T", node)
}

func evalProgram(program *ast.Program, env *Environment) Object {
	var result Object

	// First, process import statements
	for _, stmt := range program.Statements {
		if importStmt, ok := stmt.(*ast.ImportStatement); ok {
			result = Eval(importStmt, env)
			if isError(result) {
				return result
			}
		}
	}

	// Then, process file-scoped using declarations
	for _, usingStmt := range program.FileUsing {
		for _, module := range usingStmt.Modules {
			alias := module.Value
			// Verify the module was imported (check both stdlib and user modules)
			_, isStdlib := env.GetImport(alias)
			_, isUserModule := env.GetModule(alias)
			if !isStdlib && !isUserModule {
				return newErrorWithLocation("E6004", usingStmt.Token.Line, usingStmt.Token.Column,
					"cannot use '%s': module not imported", alias)
			}
			env.Use(alias)
		}
	}

	// Finally, process all other statements
	for _, stmt := range program.Statements {
		// Skip imports (already processed)
		if _, ok := stmt.(*ast.ImportStatement); ok {
			continue
		}

		// Update current file context for accurate error reporting in multi-file modules
		if globalEvalContext != nil {
			if stmtFile := getStatementFile(stmt); stmtFile != "" {
				globalEvalContext.CurrentFile = stmtFile
			}
		}

		result = Eval(stmt, env)

		switch result := result.(type) {
		case *ReturnValue:
			if len(result.Values) > 0 {
				return result.Values[0]
			}
			return NIL
		case *Error:
			return result
		}
	}

	return result
}

func evalBlockStatement(block *ast.BlockStatement, env *Environment) Object {
	var result Object

	for _, stmt := range block.Statements {
		result = Eval(stmt, env)

		if result != nil {
			rt := result.Type()
			if rt == RETURN_VALUE_OBJ || rt == ERROR_OBJ || rt == BREAK_OBJ || rt == CONTINUE_OBJ {
				return result
			}
		}
	}

	return result
}

func evalVariableDeclaration(node *ast.VariableDeclaration, env *Environment) Object {
	var val Object = NIL

	if node.Value != nil {
		val = Eval(node.Value, env)
		if isError(val) {
			return val
		}

		// Copy-by-default for complex types (#661)
		// When assigning from another variable, deep copy structs/arrays/maps
		// UNLESS the value is a Reference (from ref() builtin)
		val = copyByDefault(val)

		// Handle multiple assignment FIRST: temp result, err = function()
		// This must happen before single-value type validation (#698)
		vis := convertVisibility(node.Visibility)
		if len(node.Names) > 1 {
			// Expect a ReturnValue with multiple values
			returnVal, ok := val.(*ReturnValue)
			if !ok {
				// Single value assigned to multiple variables - error
				return newError("expected %d values, got 1", len(node.Names))
			}

			if len(returnVal.Values) != len(node.Names) {
				return newError("expected %d values, got %d", len(node.Names), len(returnVal.Values))
			}

			// Validate types if TypeNames is provided
			for i, name := range node.Names {
				// Skip blank identifier (_)
				if name.Value == "_" {
					continue
				}

				unpackedVal := returnVal.Values[i]

				// Apply type validation if TypeNames is provided
				if i < len(node.TypeNames) && node.TypeNames[i] != "" {
					typeName := node.TypeNames[i]
					// Validate and potentially convert the value based on declared type
					validatedVal, err := validateAndConvertType(unpackedVal, typeName, node.Mutable, node.Token.Line, node.Token.Column)
					if err != nil {
						return err
					}
					unpackedVal = validatedVal
				}

				env.SetWithVisibility(name.Value, unpackedVal, node.Mutable, vis)
			}
			return NIL
		}

		// Validate type compatibility if a type is declared (single variable case)
		if node.TypeName != "" {
			// Check if declared type is an array type
			if len(node.TypeName) > 0 && node.TypeName[0] == '[' {
				// Check if const array has dynamic size (no fixed size specified)
				// const arrays must have a fixed size like [int, 3], not [int]
				if !node.Mutable && !strings.Contains(node.TypeName, ",") {
					// Extract element type from [type] -> type
					elemType := node.TypeName[1 : len(node.TypeName)-1]
					// Get the array length for the suggested fix
					arrayLen := 0
					if arr, ok := val.(*Array); ok {
						arrayLen = len(arr.Elements)
					}
					return newErrorWithLocation("E2032", node.Token.Line, node.Token.Column,
						"const array must have a fixed size\n\n"+
							"Dynamic arrays [%s] can change size, but const prevents modification.\n"+
							"Use 'temp' for dynamic arrays, or specify a fixed size for const.\n\n"+
							"Example: const arr [%s, %d] = %s",
						elemType, elemType, arrayLen, val.Inspect())
				}

				// Array type declared - value must be an array
				arr, ok := val.(*Array)
				if !ok {
					return newErrorWithLocation("E3018", node.Token.Line, node.Token.Column,
						"type mismatch: expected array type '%s', got %s\n\n"+
							"Array values must be enclosed in curly braces {}\n"+
							"Example: const arr %s = {%s}",
						node.TypeName, getEZTypeName(val), node.TypeName, val.Inspect())
				}
				// Set the element type on the array from the declared type
				// Extract element type from type name (e.g., "[int]" -> "int", "[int,5]" -> "int")
				elemType := node.TypeName[1:] // Remove leading '['
				if commaIdx := strings.Index(elemType, ","); commaIdx != -1 {
					elemType = elemType[:commaIdx] // Remove ",size]" part
				} else {
					elemType = elemType[:len(elemType)-1] // Remove trailing ']'
				}
				arr.ElementType = elemType
				// Set mutability based on temp vs const
				arr.Mutable = node.Mutable

				// Validate byte array elements are in range 0-255
				if elemType == "byte" {
					for i, elem := range arr.Elements {
						switch e := elem.(type) {
						case *Integer:
							if e.Value.Sign() < 0 || e.Value.Cmp(big.NewInt(255)) > 0 {
								return newErrorWithLocation("E3022", node.Token.Line, node.Token.Column,
									"byte array element [%d] value %s out of range: must be between 0 and 255", i, e.Value.String())
							}
							// Convert integer to byte
							arr.Elements[i] = &Byte{Value: uint8(e.Value.Int64())}
						case *Byte:
							// Already a byte, OK
						default:
							return newErrorWithLocation("E3022", node.Token.Line, node.Token.Column,
								"byte array element [%d] must be an integer value between 0 and 255", i)
						}
					}
				}
			}

			// Check if declared type is a map type
			if strings.HasPrefix(node.TypeName, "map[") {
				// Map type declared - value must be a map
				mapObj, ok := val.(*Map)
				if !ok {
					// Special case: empty {} is parsed as empty Array, convert to empty Map
					if arr, isArr := val.(*Array); isArr && len(arr.Elements) == 0 {
						mapObj = &Map{Pairs: []*MapPair{}, Index: make(map[string]int), Mutable: node.Mutable}
						val = mapObj
					} else {
						return newErrorWithLocation("E3019", node.Token.Line, node.Token.Column,
							"type mismatch: expected map type '%s', got %s\n\n"+
								"Map values must use key: value syntax\n"+
								"Example: temp m %s = {\"key\": value}",
							node.TypeName, getEZTypeName(val), node.TypeName)
					}
				} else {
					// Set mutability based on temp vs const
					mapObj.Mutable = node.Mutable
				}
			}

			// If we have a struct value, set mutability based on temp vs const
			if structObj, ok := val.(*Struct); ok {
				structObj.Mutable = node.Mutable
			}

			// If we have an integer value, set the declared type
			// and validate signed/unsigned compatibility
			if intVal, ok := val.(*Integer); ok {
				// Special handling for byte type - convert and validate range
				if node.TypeName == "byte" {
					if intVal.Value.Sign() < 0 || intVal.Value.Cmp(big.NewInt(255)) > 0 {
						return newErrorWithLocation("E3020", node.Token.Line, node.Token.Column,
							"cannot assign value %s to byte: value must be between 0 and 255", intVal.Value.String())
					}
					val = &Byte{Value: uint8(intVal.Value.Int64())}
				} else {
					// Check for negative value assigned to unsigned type
					if isUnsignedIntegerType(node.TypeName) && intVal.Value.Sign() < 0 {
						return newErrorWithLocation("E3020", node.Token.Line, node.Token.Column,
							"cannot assign negative value %s to unsigned type '%s'", intVal.Value.String(), node.TypeName)
					}
					// Set the declared type on the integer
					intVal.DeclaredType = node.TypeName
				}
			}
		}
	} else if node.TypeName != "" {
		// Variable declared with type but no value - provide appropriate default
		// Check if it's a dynamic array type (starts with '[' but doesn't contain ',')
		// Dynamic arrays: [int], [string], etc. - can be declared without values
		// Fixed-size arrays: [int,3], [string,5], etc. - MUST be initialized with values
		if len(node.TypeName) > 0 && node.TypeName[0] == '[' && !strings.Contains(node.TypeName, ",") {
			// Initialize dynamic array to empty array instead of NIL
			// Extract element type from type name (e.g., "[int]" -> "int")
			elementType := node.TypeName[1 : len(node.TypeName)-1]
			val = &Array{Elements: []Object{}, Mutable: node.Mutable, ElementType: elementType}
		} else if strings.HasPrefix(node.TypeName, "map[") {
			// Initialize map to empty map
			m := NewMap()
			m.Mutable = node.Mutable
			val = m
		} else if isIntegerType(node.TypeName) {
			// Handle all integer types (signed and unsigned)
			val = &Integer{Value: big.NewInt(0), DeclaredType: node.TypeName}
		} else {
			// Provide default values for other primitive types
			switch node.TypeName {
			case "float", "f32", "f64":
				val = &Float{Value: 0.0}
			case "string":
				val = &String{Value: "", Mutable: true}
			case "bool":
				val = FALSE // Use existing FALSE constant
			case "char":
				val = &Char{Value: '\x00'} // null character as default
			case "byte":
				val = &Byte{Value: 0}
			// For fixed-size arrays and other types, remain NIL
			default:
				val = NIL
			}
		}
	}

	// Set struct mutability based on temp vs const (for type inference case where TypeName == "")
	// This handles cases like: temp p = new(Person) or temp p = Person{name: "test"}
	// where no explicit type annotation is provided
	if structObj, ok := val.(*Struct); ok {
		structObj.Mutable = node.Mutable
	}

	// Single variable assignment
	vis := convertVisibility(node.Visibility)
	env.SetWithVisibility(node.Name.Value, val, node.Mutable, vis)
	return NIL
}

// validateAndConvertType validates a value against a declared type and converts if necessary
func validateAndConvertType(val Object, typeName string, mutable bool, line, col int) (Object, *Error) {
	// Handle array types
	if len(typeName) > 0 && typeName[0] == '[' {
		arr, ok := val.(*Array)
		if !ok {
			return nil, newErrorWithLocation("E3018", line, col,
				"type mismatch: expected array type '%s', got %s",
				typeName, getEZTypeName(val))
		}
		// Extract element type from type name
		elemType := typeName[1:]
		if commaIdx := strings.Index(elemType, ","); commaIdx != -1 {
			elemType = elemType[:commaIdx]
		} else {
			elemType = elemType[:len(elemType)-1]
		}
		arr.ElementType = elemType
		arr.Mutable = mutable
		return arr, nil
	}

	// Handle map types
	if strings.HasPrefix(typeName, "map[") {
		mapObj, ok := val.(*Map)
		if !ok {
			if arr, isArr := val.(*Array); isArr && len(arr.Elements) == 0 {
				mapObj = &Map{Pairs: []*MapPair{}, Index: make(map[string]int), Mutable: mutable}
				return mapObj, nil
			}
			return nil, newErrorWithLocation("E3019", line, col,
				"type mismatch: expected map type '%s', got %s",
				typeName, getEZTypeName(val))
		}
		mapObj.Mutable = mutable
		return mapObj, nil
	}

	// Handle struct types
	if structObj, ok := val.(*Struct); ok {
		structObj.Mutable = mutable
		return structObj, nil
	}

	// Handle integer types
	if intVal, ok := val.(*Integer); ok {
		if typeName == "byte" {
			if intVal.Value.Sign() < 0 || intVal.Value.Cmp(big.NewInt(255)) > 0 {
				return nil, newErrorWithLocation("E3020", line, col,
					"cannot assign value %s to byte: value must be between 0 and 255", intVal.Value.String())
			}
			return &Byte{Value: uint8(intVal.Value.Int64())}, nil
		}
		if isUnsignedIntegerType(typeName) && intVal.Value.Sign() < 0 {
			return nil, newErrorWithLocation("E3020", line, col,
				"cannot assign negative value %s to unsigned type '%s'", intVal.Value.String(), typeName)
		}
		intVal.DeclaredType = typeName
		return intVal, nil
	}

	// For other types (nil, Error, etc.), just return as-is
	return val, nil
}

func evalAssignment(node *ast.AssignmentStatement, env *Environment) Object {
	val := Eval(node.Value, env)
	if isError(val) {
		return val
	}

	// Copy-by-default for complex types on simple assignment (#661)
	if node.Operator == "=" {
		val = copyByDefault(val)
	}

	// Handle tuple unpacking assignment: a, b = func() (#699)
	if len(node.Names) > 1 {
		returnVal, ok := val.(*ReturnValue)
		if !ok {
			return newErrorWithLocation("E5012", node.Token.Line, node.Token.Column,
				"expected %d values, got 1", len(node.Names))
		}

		if len(returnVal.Values) != len(node.Names) {
			return newErrorWithLocation("E5012", node.Token.Line, node.Token.Column,
				"expected %d values, got %d", len(node.Names), len(returnVal.Values))
		}

		for i, name := range node.Names {
			// Skip blank identifier (_)
			if name.Value == "_" {
				continue
			}

			unpackedVal := copyByDefault(returnVal.Values[i])

			// Update the variable
			found, isMutable := env.Update(name.Value, unpackedVal)
			if !found {
				return newErrorWithLocation("E4001", node.Token.Line, node.Token.Column,
					"undefined variable '%s'", name.Value)
			}
			if !isMutable {
				return newErrorWithLocation("E5013", node.Token.Line, node.Token.Column,
					"cannot assign to immutable variable '%s' (declared as const)", name.Value)
			}
		}
		return NIL
	}

	// Check for multi-value return being assigned to single variable (#698)
	if retVal, ok := val.(*ReturnValue); ok && len(retVal.Values) > 1 {
		return newErrorWithLocation("E5012", node.Token.Line, node.Token.Column,
			"cannot assign %d values to single variable; use tuple unpacking: a, b = func()",
			len(retVal.Values))
	}

	switch target := node.Name.(type) {
	case *ast.Label:
		// Check if the variable is a Reference (for & params)
		if existingVal, ok := env.Get(target.Value); ok {
			if ref, isRef := existingVal.(*Reference); isRef {
				// Handle compound assignment for references
				if node.Operator != "=" {
					oldVal, ok := ref.Deref()
					if !ok {
						return newErrorWithLocation("E4001", node.Token.Line, node.Token.Column,
							"cannot dereference variable '%s'", target.Value)
					}
					val = evalCompoundAssignment(node.Operator, oldVal, val, node.Token.Line, node.Token.Column)
					if isError(val) {
						return val
					}
				}
				// Update through the reference
				ref.SetValue(val)
				return NIL
			}
		}

		// Handle compound assignment
		if node.Operator != "=" {
			oldVal, ok := env.Get(target.Value)
			if !ok {
				return newErrorWithLocation("E4001", node.Token.Line, node.Token.Column,
					"undefined variable '%s'", target.Value)
			}
			val = evalCompoundAssignment(node.Operator, oldVal, val, node.Token.Line, node.Token.Column)
			if isError(val) {
				return val
			}
		}

		found, isMutable := env.Update(target.Value, val)
		if !found {
			return newErrorWithLocation("E4001", node.Token.Line, node.Token.Column,
				"undefined variable '%s'", target.Value)
		}
		if !isMutable {
			return newErrorWithLocation("E5013", node.Token.Line, node.Token.Column,
				"cannot assign to immutable variable '%s' (declared as const)", target.Value)
		}

	case *ast.IndexExpression:
		// Array, string, or map index assignment
		// First check if the container variable is mutable
		if ident, ok := target.Left.(*ast.Label); ok {
			isMutable, exists := env.IsMutable(ident.Value)
			if exists && !isMutable {
				return newErrorWithLocation("E5006", node.Token.Line, node.Token.Column,
					"cannot modify immutable variable '%s' (declared as const)", ident.Value)
			}
		}

		container := Eval(target.Left, env)
		if isError(container) {
			return container
		}
		idx := Eval(target.Index, env)
		if isError(idx) {
			return idx
		}

		switch obj := container.(type) {
		case *Array:
			// Check if array is mutable
			if !obj.Mutable {
				return newErrorWithLocation("E5007", node.Token.Line, node.Token.Column,
					"cannot modify immutable array (declared as const)")
			}
			index, ok := idx.(*Integer)
			if !ok {
				return newErrorWithLocation("E3003", node.Token.Line, node.Token.Column,
					"array index must be integer, got %s", idx.Type())
			}
			arrLen := big.NewInt(int64(len(obj.Elements)))
			if index.Value.Sign() < 0 || index.Value.Cmp(arrLen) >= 0 {
				if arrLen.Sign() == 0 {
					return newErrorWithLocation("E9004", node.Token.Line, node.Token.Column,
						"index out of bounds: array is empty (length 0)\n\n"+
							"Attempted to assign to index %s, but array has no elements\n"+
							"Hint: Use arrays.append() to add elements before accessing by index", index.Value.String())
				}
				return newErrorWithLocation("E9001", node.Token.Line, node.Token.Column,
					"index out of bounds: attempted to assign to index %s, but valid range is 0-%d",
					index.Value.String(), arrLen.Int64()-1)
			}

			// Handle compound assignment
			if node.Operator != "=" {
				oldVal := obj.Elements[index.Value.Int64()]
				val = evalCompoundAssignment(node.Operator, oldVal, val, node.Token.Line, node.Token.Column)
				if isError(val) {
					return val
				}
			}

			obj.Elements[index.Value.Int64()] = val

		case *String:
			index, ok := idx.(*Integer)
			if !ok {
				return newErrorWithLocation("E3003", node.Token.Line, node.Token.Column,
					"string index must be integer, got %s", idx.Type())
			}
			// String mutation - verify the value is a character
			charObj, ok := val.(*Char)
			if !ok {
				return newErrorWithLocation("E3004", node.Token.Line, node.Token.Column,
					"can only assign character to string index, got %s", val.Type())
			}
			// Convert string to rune slice for proper UTF-8 character indexing
			runes := []rune(obj.Value)
			strLen := big.NewInt(int64(len(runes)))
			if index.Value.Sign() < 0 || index.Value.Cmp(strLen) >= 0 {
				if strLen.Sign() == 0 {
					return newErrorWithLocation("E5004", node.Token.Line, node.Token.Column,
						"index out of bounds: string is empty (length 0)\n\n"+
							"Attempted to assign to index %s", index.Value.String())
				}
				return newErrorWithLocation("E5003", node.Token.Line, node.Token.Column,
					"index out of bounds: attempted to assign to index %s, but valid range is 0-%d",
					index.Value.String(), strLen.Int64()-1)
			}
			// Modify rune slice and convert back
			runes[index.Value.Int64()] = charObj.Value
			obj.Value = string(runes)

		case *Map:
			// Map key assignment
			// Validate that the key is hashable
			if _, ok := HashKey(idx); !ok {
				return newErrorWithLocation("E12001", node.Token.Line, node.Token.Column,
					"map key must be a hashable type, got %s", idx.Type())
			}

			// Check if map is mutable
			if !obj.Mutable {
				return newErrorWithLocation("E12002", node.Token.Line, node.Token.Column,
					"cannot modify immutable map (declared as const)")
			}

			// Handle compound assignment
			if node.Operator != "=" {
				oldVal, exists := obj.Get(idx)
				if !exists {
					return newErrorWithLocation("E12003", node.Token.Line, node.Token.Column,
						"key not found in map for compound assignment")
				}
				val = evalCompoundAssignment(node.Operator, oldVal, val, node.Token.Line, node.Token.Column)
				if isError(val) {
					return val
				}
			}

			obj.Set(idx, val)

		default:
			return newErrorWithLocation("E3016", node.Token.Line, node.Token.Column,
				"index operator not supported: %s", container.Type())
		}

	case *ast.MemberExpression:
		// Check if this is a module member assignment (not allowed)
		if objIdent, ok := target.Object.(*ast.Label); ok {
			alias := objIdent.Value
			// Check if it's a user module
			if _, ok := env.GetModule(alias); ok {
				return newErrorWithLocation("E6008", node.Token.Line, node.Token.Column,
					"cannot assign to module member '%s.%s'\n\n"+
						"Module exports are read-only and cannot be modified from outside the module.",
					alias, target.Member.Value)
			}
			// Check if it's a stdlib import
			if _, ok := env.GetImport(alias); ok {
				return newErrorWithLocation("E6008", node.Token.Line, node.Token.Column,
					"cannot assign to module member '%s.%s'\n\n"+
						"Module exports are read-only and cannot be modified from outside the module.",
					alias, target.Member.Value)
			}
		}

		// Struct field assignment
		obj := Eval(target.Object, env)
		if isError(obj) {
			return obj
		}
		structObj, ok := obj.(*Struct)
		if !ok {
			return newErrorWithLocation("E4011", node.Token.Line, node.Token.Column,
				"member access not supported: %s", obj.Type())
		}

		// Check if struct is mutable
		// A struct is considered mutable if:
		// 1. It was declared with 'temp' (structObj.Mutable is true), OR
		// 2. It's accessed via an index on a mutable container (array/map)
		isMutableAccess := structObj.Mutable
		if !isMutableAccess {
			// Check if struct is accessed via mutable container
			if indexExpr, isIndex := target.Object.(*ast.IndexExpression); isIndex {
				container := Eval(indexExpr.Left, env)
				if !isError(container) {
					switch c := container.(type) {
					case *Array:
						isMutableAccess = c.Mutable
					case *Map:
						isMutableAccess = c.Mutable
					}
				}
			}
		}
		if !isMutableAccess {
			return newErrorWithLocation("E5017", node.Token.Line, node.Token.Column,
				"cannot modify field of immutable struct (declared as const)")
		}

		// Handle compound assignment
		if node.Operator != "=" {
			oldVal, exists := structObj.Fields[target.Member.Value]
			if !exists {
				return newError("field '%s' not found", target.Member.Value)
			}
			val = evalCompoundAssignment(node.Operator, oldVal, val, node.Token.Line, node.Token.Column)
			if isError(val) {
				return val
			}
		}

		structObj.Fields[target.Member.Value] = val
	}

	return NIL
}

func evalCompoundAssignment(op string, left, right Object, line, col int) Object {
	switch op {
	case "+=":
		return evalInfixExpression("+", left, right, line, col)
	case "-=":
		return evalInfixExpression("-", left, right, line, col)
	case "*=":
		return evalInfixExpression("*", left, right, line, col)
	case "/=":
		return evalInfixExpression("/", left, right, line, col)
	case "%=":
		return evalInfixExpression("%", left, right, line, col)
	default:
		return newError("unknown operator: %s", op)
	}
}

func evalReturn(node *ast.ReturnStatement, env *Environment) Object {
	values := make([]Object, len(node.Values))
	for i, v := range node.Values {
		val := Eval(v, env)
		if isError(val) {
			return val
		}
		values[i] = val
	}
	return &ReturnValue{Values: values}
}

func evalIfStatement(node *ast.IfStatement, env *Environment) Object {
	condition := Eval(node.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		// Create a new enclosed environment for the if block to support proper variable shadowing
		ifEnv := NewEnclosedEnvironment(env)
		return Eval(node.Consequence, ifEnv)
	} else if node.Alternative != nil {
		// Create a new enclosed environment for the else block to support proper variable shadowing
		elseEnv := NewEnclosedEnvironment(env)
		return Eval(node.Alternative, elseEnv)
	}

	return NIL
}

func evalWhenStatement(node *ast.WhenStatement, env *Environment) Object {
	// Evaluate the value being matched
	matchValue := Eval(node.Value, env)
	if isError(matchValue) {
		return matchValue
	}

	// Unwrap EnumValue to get the underlying value for comparisons
	// This allows matching enum values against integers and ranges
	if ev, ok := matchValue.(*EnumValue); ok {
		matchValue = ev.Value
	}

	// Try each case
	for _, whenCase := range node.Cases {
		matched := false

		for _, caseValue := range whenCase.Values {
			// Check if this is a range expression
			if rangeExpr, ok := caseValue.(*ast.RangeExpression); ok {
				// Evaluate the range expression to get a Range object
				rangeObj := evalRangeExpression(rangeExpr, env)
				if isError(rangeObj) {
					return rangeObj
				}

				// Check if match value is within range (respects step)
				if matchInt, ok := matchValue.(*Integer); ok {
					if r, ok := rangeObj.(*Range); ok {
						if r.Contains(matchInt.Value) {
							matched = true
							break
						}
					}
				}
				continue
			}

			// Evaluate the case value
			evalCaseValue := Eval(caseValue, env)
			if isError(evalCaseValue) {
				return evalCaseValue
			}

			// Unwrap EnumValue case values for comparison
			if ev, ok := evalCaseValue.(*EnumValue); ok {
				evalCaseValue = ev.Value
			}

			// Compare for equality
			if objectsEqual(matchValue, evalCaseValue) {
				matched = true
				break
			}
		}

		if matched {
			caseEnv := NewEnclosedEnvironment(env)
			return Eval(whenCase.Body, caseEnv)
		}
	}

	// No case matched, execute default if present
	if node.Default != nil {
		defaultEnv := NewEnclosedEnvironment(env)
		return Eval(node.Default, defaultEnv)
	}

	return NIL
}

// objectsEqual compares two objects for equality
func objectsEqual(a, b Object) bool {
	switch aVal := a.(type) {
	case *Integer:
		if bVal, ok := b.(*Integer); ok {
			return aVal.Value.Cmp(bVal.Value) == 0
		}
	case *String:
		if bVal, ok := b.(*String); ok {
			return aVal.Value == bVal.Value
		}
	case *Char:
		if bVal, ok := b.(*Char); ok {
			return aVal.Value == bVal.Value
		}
	case *Boolean:
		if bVal, ok := b.(*Boolean); ok {
			return aVal.Value == bVal.Value
		}
	case *EnumValue:
		if bVal, ok := b.(*EnumValue); ok {
			return aVal.EnumType == bVal.EnumType && aVal.Name == bVal.Name
		}
	}
	return false
}

func evalWhileStatement(node *ast.WhileStatement, env *Environment) Object {
	env.EnterLoop()
	defer env.ExitLoop()

	for {
		condition := Eval(node.Condition, env)
		if isError(condition) {
			return condition
		}

		if !isTruthy(condition) {
			break
		}

		// Create a new enclosed environment for each iteration to support proper variable shadowing
		whileEnv := NewEnclosedEnvironment(env)
		result := Eval(node.Body, whileEnv)
		if result != nil {
			if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
				return result
			}
			if result.Type() == BREAK_OBJ {
				break
			}
			// Continue just continues the loop
		}
	}

	return NIL
}

func evalLoopStatement(node *ast.LoopStatement, env *Environment) Object {
	env.EnterLoop()
	defer env.ExitLoop()

	for {
		// Create a new enclosed environment for each iteration to support proper variable shadowing
		loopEnv := NewEnclosedEnvironment(env)
		result := Eval(node.Body, loopEnv)
		if result != nil {
			if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
				return result
			}
			if result.Type() == BREAK_OBJ {
				break
			}
		}
	}

	return NIL
}

func evalRangeExpression(node *ast.RangeExpression, env *Environment) Object {
	line, col := node.Token.Line, node.Token.Column

	// Handle start - defaults to 0 if nil (single-argument form)
	start := big.NewInt(0)
	if node.Start != nil {
		startObj := Eval(node.Start, env)
		if isError(startObj) {
			return startObj
		}
		startInt, ok := startObj.(*Integer)
		if !ok {
			return newErrorWithLocation("E5013", line, col,
				"range start must be integer, got %s", startObj.Type())
		}
		start = new(big.Int).Set(startInt.Value)
	}

	// Handle end
	endObj := Eval(node.End, env)
	if isError(endObj) {
		return endObj
	}
	endInt, ok := endObj.(*Integer)
	if !ok {
		return newErrorWithLocation("E5014", line, col,
			"range end must be integer, got %s", endObj.Type())
	}
	end := new(big.Int).Set(endInt.Value)

	// Handle step - defaults to 1 (or -1 for descending ranges)
	step := big.NewInt(1)
	if node.Step != nil {
		stepObj := Eval(node.Step, env)
		if isError(stepObj) {
			return stepObj
		}
		stepInt, ok := stepObj.(*Integer)
		if !ok {
			return newErrorWithLocation("E5019", line, col,
				"range step must be integer, got %s", stepObj.Type())
		}
		step = new(big.Int).Set(stepInt.Value)
		if step.Sign() == 0 {
			return newErrorWithLocation("E9003", line, col,
				"range step cannot be zero")
		}
	} else if start.Cmp(end) > 0 {
		// Auto-detect descending range when no step is provided
		step = big.NewInt(-1)
	}

	return &Range{Start: start, End: end, Step: step}
}

// evalCastExpression evaluates cast(value, type) expressions for type conversion
func evalCastExpression(node *ast.CastExpression, env *Environment) Object {
	line, col := node.Token.Line, node.Token.Column

	// Evaluate the value expression
	value := Eval(node.Value, env)
	if isError(value) {
		return value
	}

	if node.IsArray {
		// Array cast: convert each element
		return evalArrayCast(value, node.ElementType, line, col)
	}

	// Single value cast
	return evalSingleCast(value, node.TargetType, line, col)
}

// evalArrayCast converts an array to a new array with elements of the target type
func evalArrayCast(value Object, elementType string, line, col int) Object {
	arr, ok := value.(*Array)
	if !ok {
		return newErrorWithLocation("E3001", line, col,
			"cast to array type requires array value, got %s", value.Type())
	}

	newElements := make([]Object, len(arr.Elements))
	for i, elem := range arr.Elements {
		converted := evalSingleCast(elem, elementType, line, col)
		if isError(converted) {
			// Add index info to the error
			errObj := converted.(*Error)
			return newErrorWithLocation(errObj.Code, line, col,
				"cast failed at index %d: %s", i, errObj.Message)
		}
		newElements[i] = converted
	}

	return &Array{
		Elements:    newElements,
		Mutable:     true,
		ElementType: elementType,
	}
}

// evalSingleCast converts a single value to the target type
func evalSingleCast(value Object, targetType string, line, col int) Object {
	// Get the appropriate builtin conversion function
	builtin, ok := builtins[targetType]
	if !ok {
		return newErrorWithLocation("E3001", line, col,
			"unknown cast target type: %s", targetType)
	}

	result := builtin.Fn(value)
	if errObj, isErr := result.(*Error); isErr {
		if errObj.Line == 0 && errObj.Column == 0 {
			errObj.Line = line
			errObj.Column = col
		}
	}
	return result
}

func evalForStatement(node *ast.ForStatement, env *Environment) Object {
	env.EnterLoop()
	defer env.ExitLoop()

	// Get range bounds
	rangeExpr, ok := node.Iterable.(*ast.RangeExpression)
	if !ok {
		return newErrorWithLocation("E5011", node.Token.Line, node.Token.Column,
			"for loop requires range() expression\n\n"+
				"Did you mean to use 'for_each' to iterate over a collection?\n\n"+
				"Use 'for' with range() for numeric iteration:\n"+
				"    for i in range(0, 10) { ... }\n\n"+
				"Use 'for_each' to iterate over arrays/strings:\n"+
				"    for_each item in collection { ... }")
	}

	// Handle start - defaults to 0 if nil (single-argument form)
	start := big.NewInt(0)
	if rangeExpr.Start != nil {
		startObj := Eval(rangeExpr.Start, env)
		if isError(startObj) {
			return startObj
		}
		startInt, ok := startObj.(*Integer)
		if !ok {
			return newError("range start must be integer")
		}
		start = new(big.Int).Set(startInt.Value)
	}

	// Handle end
	endObj := Eval(rangeExpr.End, env)
	if isError(endObj) {
		return endObj
	}
	endInt, ok := endObj.(*Integer)
	if !ok {
		return newError("range end must be integer")
	}
	end := endInt.Value

	// Handle step - defaults to 1 (or -1 for descending ranges)
	step := big.NewInt(1)
	if rangeExpr.Step != nil {
		stepObj := Eval(rangeExpr.Step, env)
		if isError(stepObj) {
			return stepObj
		}
		stepInt, ok := stepObj.(*Integer)
		if !ok {
			return newError("range step must be integer")
		}
		step = new(big.Int).Set(stepInt.Value)
		if step.Sign() == 0 {
			return newError("range step cannot be zero")
		}
	} else if start.Cmp(end) > 0 {
		// Auto-detect descending range when no step is provided
		step = big.NewInt(-1)
	}

	loopEnv := NewEnclosedEnvironment(env)

	// Handle positive and negative steps
	if step.Sign() > 0 {
		for i := new(big.Int).Set(start); i.Cmp(end) < 0; i.Add(i, step) {
			loopEnv.Set(node.Variable.Value, &Integer{Value: new(big.Int).Set(i)}, true)

			result := Eval(node.Body, loopEnv)
			if result != nil {
				if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
					return result
				}
				if result.Type() == BREAK_OBJ {
					break
				}
			}
		}
	} else {
		// Negative step: count down
		for i := new(big.Int).Set(start); i.Cmp(end) > 0; i.Add(i, step) {
			loopEnv.Set(node.Variable.Value, &Integer{Value: new(big.Int).Set(i)}, true)

			result := Eval(node.Body, loopEnv)
			if result != nil {
				if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
					return result
				}
				if result.Type() == BREAK_OBJ {
					break
				}
			}
		}
	}

	return NIL
}

func evalForEachStatement(node *ast.ForEachStatement, env *Environment) Object {
	env.EnterLoop()
	defer env.ExitLoop()

	collection := Eval(node.Collection, env)
	if isError(collection) {
		return collection
	}

	loopEnv := NewEnclosedEnvironment(env)

	// Handle arrays
	if arr, ok := collection.(*Array); ok {
		arr.IteratingCount++
		defer func() { arr.IteratingCount-- }()

		for _, elem := range arr.Elements {
			loopEnv.Set(node.Variable.Value, elem, true) // loop vars are mutable

			result := Eval(node.Body, loopEnv)
			if result != nil {
				if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
					return result
				}
				if result.Type() == BREAK_OBJ {
					break
				}
			}
		}
		return NIL
	}

	// Handle strings (iterate over characters)
	if str, ok := collection.(*String); ok {
		for _, ch := range str.Value {
			charObj := &Char{Value: ch}
			loopEnv.Set(node.Variable.Value, charObj, true) // loop vars are mutable

			result := Eval(node.Body, loopEnv)
			if result != nil {
				if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
					return result
				}
				if result.Type() == BREAK_OBJ {
					break
				}
			}
		}
		return NIL
	}

	return newErrorWithLocation("E3017", node.Token.Line, node.Token.Column,
		"for_each requires array or string, got %s", collection.Type())
}

func evalEnumDeclaration(node *ast.EnumDeclaration, env *Environment) Object {
	enum := &Enum{
		Name:   node.Name.Value,
		Values: make(map[string]Object),
	}

	// Get enum attributes (type, flags)
	typeName := "int" // default
	isFlags := false

	if node.Attributes != nil {
		typeName = node.Attributes.TypeName
		isFlags = node.Attributes.IsFlags
	}

	// Compute enum values
	currentInt := big.NewInt(0)
	var currentFloat float64 = 0.0
	flagValue := big.NewInt(1) // for @flags: 1, 2, 4, 8, ...

	for i, enumVal := range node.Values {
		if enumVal.Value != nil {
			// Explicit value assignment
			val := Eval(enumVal.Value, env)
			if isError(val) {
				return val
			}
			enum.Values[enumVal.Name.Value] = val

			// Update current value for next auto-increment
			switch v := val.(type) {
			case *Integer:
				if isFlags {
					// For flags, next value is double the current
					flagValue = new(big.Int).Lsh(v.Value, 1)
				} else {
					currentInt = new(big.Int).Add(v.Value, big.NewInt(1))
				}
			case *Float:
				currentFloat = v.Value + 1.0
			case *String:
				// Strings don't auto-increment
			}
		} else {
			// Auto-assign value based on type
			switch typeName {
			case "int":
				if isFlags {
					// @flags: use power-of-2 values (1, 2, 4, 8, ...)
					if i == 0 {
						enum.Values[enumVal.Name.Value] = &Integer{Value: big.NewInt(1)}
						flagValue = big.NewInt(2)
					} else {
						enum.Values[enumVal.Name.Value] = &Integer{Value: new(big.Int).Set(flagValue)}
						flagValue = new(big.Int).Lsh(flagValue, 1) // multiply by 2
					}
				} else {
					// Regular int enum: 0, 1, 2, 3, ...
					enum.Values[enumVal.Name.Value] = &Integer{Value: new(big.Int).Set(currentInt)}
					currentInt.Add(currentInt, big.NewInt(1))
				}
			case "float":
				enum.Values[enumVal.Name.Value] = &Float{Value: currentFloat}
				currentFloat += 1.0
			case "string":
				return newErrorWithLocation("E2031", enumVal.Name.Token.Line, enumVal.Name.Token.Column,
					"string enum '%s' requires explicit value for member '%s'\n\n"+
						"String enums do not auto-increment. Provide an explicit value like:\n"+
						"  %s = \"%s\"", node.Name.Value, enumVal.Name.Value, enumVal.Name.Value, strings.ToLower(enumVal.Name.Value))
			default:
				return newError("unsupported enum type: %s", typeName)
			}
		}
	}

	// Store the enum in the environment with visibility
	vis := convertVisibility(node.Visibility)
	env.SetWithVisibility(node.Name.Value, enum, false, vis) // enums are immutable
	return NIL
}

func evalFunctionDeclaration(node *ast.FunctionDeclaration, env *Environment) Object {
	// Get file from token (set by parser for multi-file modules) or from context
	file := node.Token.File
	if file == "" && globalEvalContext != nil {
		file = globalEvalContext.CurrentFile
	}
	fn := &Function{
		Parameters:  node.Parameters,
		ReturnTypes: node.ReturnTypes,
		Body:        node.Body,
		Env:         env,
		File:        file,
	}
	vis := convertVisibility(node.Visibility)
	env.SetWithVisibility(node.Name.Value, fn, false, vis) // functions are immutable
	return NIL
}

func evalIdentifier(node *ast.Label, env *Environment) Object {
	if val, ok := env.Get(node.Value); ok {
		// If the value is a Reference (for & params), dereference it
		if ref, isRef := val.(*Reference); isRef {
			if derefVal, ok := ref.Deref(); ok {
				return derefVal
			}
		}
		// Mutability is now set at declaration time, not at lookup time
		// This preserves the original object's mutability when passed to functions
		return val
	}

	// Check if the function is available via "using" modules
	// Detect ambiguity: if multiple modules have the same function, error
	var foundModules []string
	var foundBuiltin *Builtin
	var foundUserObj Object

	for _, alias := range env.GetUsing() {
		// Check stdlib modules
		if module, ok := env.GetImport(alias); ok {
			fullName := module + "." + node.Value
			if builtin, ok := builtins[fullName]; ok {
				foundModules = append(foundModules, module)
				foundBuiltin = builtin
			}
		}
		// Check user modules
		if moduleObj, ok := env.GetModule(alias); ok {
			if obj, ok := moduleObj.Get(node.Value); ok {
				foundModules = append(foundModules, alias)
				foundUserObj = obj
			}
		}
	}

	// Ambiguity check
	if len(foundModules) > 1 {
		err := newErrorWithLocation("E4008", node.Token.Line, node.Token.Column,
			"function '%s' found in multiple modules", node.Value)
		// Build helpful error message
		moduleList := strings.Join(foundModules, ", ")
		err.Help = fmt.Sprintf("use explicit module prefix: %s.%s()", foundModules[0], node.Value)
		err.Message = fmt.Sprintf("function '%s' found in multiple modules: %s", node.Value, moduleList)
		return err
	}

	// Found in exactly one module
	if len(foundModules) == 1 {
		if foundBuiltin != nil {
			// For constants (IsConstant=true), call immediately to get the value
			if foundBuiltin.IsConstant {
				return foundBuiltin.Fn()
			}
			return foundBuiltin
		}
		return foundUserObj
	}

	// Check global builtins (like len, typeof, etc.)
	if builtin, ok := builtins[node.Value]; ok {
		// For constants (IsConstant=true), call immediately to get the value
		if builtin.IsConstant {
			return builtin.Fn()
		}
		return builtin
	}

	// Check if it's a struct type (allows types to be passed as values to functions)
	if structDef, ok := env.GetStructDef(node.Value); ok {
		return &TypeValue{TypeName: node.Value, Def: structDef}
	}

	// Create error with potential suggestion
	err := newErrorWithLocation("E4001", node.Token.Line, node.Token.Column,
		"identifier not found: '%s'", node.Value)

	// Try to suggest a keyword or builtin
	if suggestion := errors.SuggestKeyword(node.Value); suggestion != "" {
		err.Help = fmt.Sprintf("did you mean '%s'?", suggestion)
	} else if suggestion := errors.SuggestBuiltin(node.Value); suggestion != "" {
		err.Help = fmt.Sprintf("did you mean '%s'?", suggestion)
	}

	return err
}

func evalExpressions(exps []ast.Expression, env *Environment) []Object {
	var result []Object

	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func evalPrefixExpression(operator string, right Object) Object {
	switch operator {
	case "!":
		return evalBangOperator(right)
	case "-":
		return evalMinusPrefixOperator(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalBangOperator(right Object) Object {
	// Use type assertion to check actual boolean value, not pointer identity.
	// This is necessary because stdlib functions may return object.TRUE/FALSE
	// which are different pointers than the evaluator's TRUE/FALSE constants.
	switch obj := right.(type) {
	case *Boolean:
		if obj.Value {
			return FALSE
		}
		return TRUE
	case *Nil:
		return TRUE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperator(right Object) Object {
	switch obj := right.(type) {
	case *Integer:
		result := new(big.Int).Neg(obj.Value)
		// Only check for overflow if a declared type is set
		// When no type is set, overflow will be checked at assignment time
		if obj.DeclaredType != "" && checkOverflow(result, obj.DeclaredType) {
			return newError("integer overflow: negating %s exceeds %s range", obj.Value.String(), getTypeRangeName(obj.DeclaredType))
		}
		return &Integer{Value: result, DeclaredType: obj.DeclaredType}
	case *Float:
		return &Float{Value: -obj.Value}
	default:
		return newError("unknown operator: -%s", right.Type())
	}
}

func evalInfixExpression(operator string, left, right Object, line, col int) Object {
	// Check for nil operands (except for == and != which can compare with nil)
	if operator != "==" && operator != "!=" {
		if left.Type() == NIL_OBJ {
			return newErrorWithLocation("E5006", line, col, "nil reference: cannot use nil with operator '%s'", operator)
		}
		if right.Type() == NIL_OBJ {
			return newErrorWithLocation("E5006", line, col, "nil reference: cannot use nil with operator '%s'", operator)
		}
	}

	// Unwrap EnumValue to get the underlying value for comparisons
	if ev, ok := left.(*EnumValue); ok {
		left = ev.Value
	}
	if ev, ok := right.(*EnumValue); ok {
		right = ev.Value
	}

	switch {
	// Handle 'in' and 'not_in' operators early - they work with any type + array
	case operator == "in":
		return evalInOperator(left, right, line, col)
	case operator == "not_in" || operator == "!in":
		result := evalInOperator(left, right, line, col)
		if result == TRUE {
			return FALSE
		}
		return TRUE
	case left.Type() == INTEGER_OBJ && right.Type() == INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right, line, col)
	case left.Type() == BYTE_OBJ && right.Type() == BYTE_OBJ:
		return evalByteInfixExpression(operator, left, right, line, col)
	case (left.Type() == BYTE_OBJ && right.Type() == INTEGER_OBJ) || (left.Type() == INTEGER_OBJ && right.Type() == BYTE_OBJ):
		// Promote byte to integer for mixed operations
		return evalByteIntegerInfixExpression(operator, left, right, line, col)
	case left.Type() == FLOAT_OBJ || right.Type() == FLOAT_OBJ:
		return evalFloatInfixExpression(operator, left, right, line, col)
	case left.Type() == STRING_OBJ && right.Type() == STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case left.Type() == CHAR_OBJ && right.Type() == CHAR_OBJ:
		return evalCharInfixExpression(operator, left, right, line, col)
	case left.Type() == BOOLEAN_OBJ && right.Type() == BOOLEAN_OBJ && (operator == "==" || operator == "!="):
		// Compare boolean values, not pointers (stdlib may return different Boolean objects)
		leftVal := left.(*Boolean).Value
		rightVal := right.(*Boolean).Value
		if operator == "==" {
			return nativeBoolToBooleanObject(leftVal == rightVal)
		}
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case operator == "&&":
		return nativeBoolToBooleanObject(isTruthy(left) && isTruthy(right))
	case operator == "||":
		return nativeBoolToBooleanObject(isTruthy(left) || isTruthy(right))
	default:
		return newErrorWithLocation("E3014", line, col, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIntegerInfixExpression(operator string, left, right Object, line, col int) Object {
	leftInt := left.(*Integer)
	rightInt := right.(*Integer)
	leftVal := leftInt.Value
	rightVal := rightInt.Value

	// Determine result type - use the wider type or the left operand's type
	resultType := leftInt.DeclaredType
	if resultType == "" {
		resultType = rightInt.DeclaredType
	}

	switch operator {
	case "+":
		result := new(big.Int).Add(leftVal, rightVal)
		if checkOverflow(result, resultType) {
			return newErrorWithLocation("E5005", line, col, "integer overflow: %s + %s exceeds %s range", leftVal.String(), rightVal.String(), getTypeRangeName(resultType))
		}
		return &Integer{Value: result, DeclaredType: resultType}
	case "-":
		result := new(big.Int).Sub(leftVal, rightVal)
		if checkOverflow(result, resultType) {
			return newErrorWithLocation("E5006", line, col, "integer overflow: %s - %s exceeds %s range", leftVal.String(), rightVal.String(), getTypeRangeName(resultType))
		}
		return &Integer{Value: result, DeclaredType: resultType}
	case "*":
		result := new(big.Int).Mul(leftVal, rightVal)
		if checkOverflow(result, resultType) {
			return newErrorWithLocation("E5007", line, col, "integer overflow: %s * %s exceeds %s range", leftVal.String(), rightVal.String(), getTypeRangeName(resultType))
		}
		return &Integer{Value: result, DeclaredType: resultType}
	case "/":
		if rightVal.Sign() == 0 {
			return newErrorWithLocation("E5001", line, col, "division by zero")
		}
		result := new(big.Int).Quo(leftVal, rightVal)
		// Check for overflow: MinInt / -1 would exceed MaxInt for signed types
		if checkOverflow(result, resultType) {
			return newErrorWithLocation("E5007", line, col, "integer overflow: %s / %s exceeds %s range", leftVal.String(), rightVal.String(), getTypeRangeName(resultType))
		}
		return &Integer{Value: result, DeclaredType: resultType}
	case "%":
		if rightVal.Sign() == 0 {
			return newErrorWithLocation("E5002", line, col, "modulo by zero")
		}
		result := new(big.Int).Rem(leftVal, rightVal)
		return &Integer{Value: result, DeclaredType: resultType}
	case "<":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) < 0)
	case ">":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) > 0)
	case "<=":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) <= 0)
	case ">=":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) >= 0)
	case "==":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) == 0)
	case "!=":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) != 0)
	default:
		return newErrorWithLocation("E3014", line, col, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalFloatInfixExpression(operator string, left, right Object, line, col int) Object {
	var leftVal, rightVal float64

	switch l := left.(type) {
	case *Float:
		leftVal = l.Value
	case *Integer:
		leftVal, _ = new(big.Float).SetInt(l.Value).Float64()
	}

	switch r := right.(type) {
	case *Float:
		rightVal = r.Value
	case *Integer:
		rightVal, _ = new(big.Float).SetInt(r.Value).Float64()
	}

	switch operator {
	case "+":
		result := leftVal + rightVal
		if math.IsInf(result, 0) {
			return newErrorWithLocation("E5005", line, col, "float overflow: %v + %v exceeds float range", leftVal, rightVal)
		}
		return &Float{Value: result}
	case "-":
		result := leftVal - rightVal
		if math.IsInf(result, 0) {
			return newErrorWithLocation("E5006", line, col, "float overflow: %v - %v exceeds float range", leftVal, rightVal)
		}
		return &Float{Value: result}
	case "*":
		result := leftVal * rightVal
		if math.IsInf(result, 0) {
			return newErrorWithLocation("E5007", line, col, "float overflow: %v * %v exceeds float range", leftVal, rightVal)
		}
		return &Float{Value: result}
	case "/":
		if rightVal == 0 {
			return newErrorWithLocation("E5001", line, col, "division by zero")
		}
		result := leftVal / rightVal
		if math.IsInf(result, 0) {
			return newErrorWithLocation("E5007", line, col, "float overflow: %v / %v exceeds float range", leftVal, rightVal)
		}
		return &Float{Value: result}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newErrorWithLocation("E3014", line, col, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalStringInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*String).Value
	rightVal := right.(*String).Value

	switch operator {
	case "+":
		return &String{Value: leftVal + rightVal, Mutable: true}
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalCharInfixExpression(operator string, left, right Object, line, col int) Object {
	leftVal := left.(*Char).Value
	rightVal := right.(*Char).Value

	switch operator {
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	default:
		return newErrorWithLocation("E3014", line, col, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalByteInfixExpression(operator string, left, right Object, line, col int) Object {
	leftVal := left.(*Byte).Value
	rightVal := right.(*Byte).Value

	switch operator {
	case "+":
		result := leftVal + rightVal
		if result < leftVal { // overflow: wrapped around
			return newErrorWithLocation("E5005", line, col, "byte overflow: %d + %d exceeds byte range (0-255)", leftVal, rightVal)
		}
		return &Byte{Value: result}
	case "-":
		if leftVal < rightVal { // underflow: would wrap around
			return newErrorWithLocation("E5006", line, col, "byte underflow: %d - %d exceeds byte range (0-255)", leftVal, rightVal)
		}
		return &Byte{Value: leftVal - rightVal}
	case "*":
		if rightVal != 0 {
			result := leftVal * rightVal
			if result/rightVal != leftVal { // overflow: wrapped around
				return newErrorWithLocation("E5007", line, col, "byte overflow: %d * %d exceeds byte range (0-255)", leftVal, rightVal)
			}
			return &Byte{Value: result}
		}
		return &Byte{Value: 0}
	case "/":
		if rightVal == 0 {
			return newErrorWithLocation("E5001", line, col, "division by zero")
		}
		return &Byte{Value: leftVal / rightVal}
	case "%":
		if rightVal == 0 {
			return newErrorWithLocation("E5002", line, col, "modulo by zero")
		}
		return &Byte{Value: leftVal % rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newErrorWithLocation("E3014", line, col, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalByteIntegerInfixExpression(operator string, left, right Object, line, col int) Object {
	var leftVal, rightVal *big.Int

	// Extract values, promoting byte to big.Int
	switch l := left.(type) {
	case *Byte:
		leftVal = big.NewInt(int64(l.Value))
	case *Integer:
		leftVal = l.Value
	}

	switch r := right.(type) {
	case *Byte:
		rightVal = big.NewInt(int64(r.Value))
	case *Integer:
		rightVal = r.Value
	}

	switch operator {
	case "+":
		return &Integer{Value: new(big.Int).Add(leftVal, rightVal)}
	case "-":
		return &Integer{Value: new(big.Int).Sub(leftVal, rightVal)}
	case "*":
		return &Integer{Value: new(big.Int).Mul(leftVal, rightVal)}
	case "/":
		if rightVal.Sign() == 0 {
			return newErrorWithLocation("E5001", line, col, "division by zero")
		}
		return &Integer{Value: new(big.Int).Quo(leftVal, rightVal)}
	case "%":
		if rightVal.Sign() == 0 {
			return newErrorWithLocation("E5002", line, col, "modulo by zero")
		}
		return &Integer{Value: new(big.Int).Rem(leftVal, rightVal)}
	case "<":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) < 0)
	case ">":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) > 0)
	case "<=":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) <= 0)
	case ">=":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) >= 0)
	case "==":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) == 0)
	case "!=":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) != 0)
	default:
		return newErrorWithLocation("E3014", line, col, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalInOperator(left, right Object, line, col int) Object {
	// Handle range membership check
	if r, ok := right.(*Range); ok {
		// Only integers can be checked against ranges
		leftInt, ok := left.(*Integer)
		if !ok {
			return newErrorWithLocation("E5020", line, col,
				"left operand of 'in range()' must be integer, got %s", left.Type())
		}
		if r.Contains(leftInt.Value) {
			return TRUE
		}
		return FALSE
	}

	// Handle array membership check
	arr, ok := right.(*Array)
	if !ok {
		return newErrorWithLocation("E3014", line, col,
			"right operand of 'in' must be array or range, got %s", right.Type())
	}

	for _, elem := range arr.Elements {
		if elementsEqual(left, elem) {
			return TRUE
		}
	}

	return FALSE
}

func elementsEqual(a, b Object) bool {
	switch av := a.(type) {
	case *Integer:
		if bv, ok := b.(*Integer); ok {
			return av.Value.Cmp(bv.Value) == 0
		}
	case *String:
		if bv, ok := b.(*String); ok {
			return av.Value == bv.Value
		}
	case *Boolean:
		if bv, ok := b.(*Boolean); ok {
			return av.Value == bv.Value
		}
	case *Byte:
		if bv, ok := b.(*Byte); ok {
			return av.Value == bv.Value
		}
	case *Char:
		if bv, ok := b.(*Char); ok {
			return av.Value == bv.Value
		}
	case *Float:
		if bv, ok := b.(*Float); ok {
			return av.Value == bv.Value
		}
	}
	return a == b
}

func evalPostfixExpression(node *ast.PostfixExpression, env *Environment) Object {
	ident, ok := node.Left.(*ast.Label)
	if !ok {
		return newErrorWithLocation("E5015", node.Token.Line, node.Token.Column,
			"postfix operator %s requires a variable identifier", node.Operator)
	}

	val, ok := env.Get(ident.Value)
	if !ok {
		return newError("identifier not found: %s", ident.Value)
	}

	intVal, ok := val.(*Integer)
	if !ok {
		return newErrorWithLocation("E5023", node.Token.Line, node.Token.Column,
			"postfix operator %s requires integer operand, got %s", node.Operator, val.Type())
	}

	var newVal *big.Int
	switch node.Operator {
	case "++":
		newVal = new(big.Int).Add(intVal.Value, one)
		if checkOverflow(newVal, intVal.DeclaredType) {
			return newErrorWithLocation("E5008", node.Token.Line, node.Token.Column,
				"integer overflow: %s++ exceeds %s range", intVal.Value.String(), getTypeRangeName(intVal.DeclaredType))
		}
	case "--":
		newVal = new(big.Int).Sub(intVal.Value, one)
		if checkOverflow(newVal, intVal.DeclaredType) {
			return newErrorWithLocation("E5009", node.Token.Line, node.Token.Column,
				"integer overflow: %s-- exceeds %s range", intVal.Value.String(), getTypeRangeName(intVal.DeclaredType))
		}
	default:
		return newError("unknown postfix operator: %s", node.Operator)
	}

	env.Update(ident.Value, &Integer{Value: newVal, DeclaredType: intVal.DeclaredType})
	return intVal // Return old value (postfix behavior)
}

func evalCallExpression(node *ast.CallExpression, env *Environment) Object {
	// Handle member calls like std.println
	if member, ok := node.Function.(*ast.MemberExpression); ok {
		return evalMemberCall(member, node.Arguments, env)
	}

	// Special handling for ref() builtin - needs access to environment (#661)
	if label, ok := node.Function.(*ast.Label); ok && label.Value == "ref" {
		return evalRefBuiltin(node.Arguments, env, node.Token.Line, node.Token.Column)
	}

	function := Eval(node.Function, env)
	if isError(function) {
		// Check if this is an "identifier not found" error and make it more specific
		if errObj, ok := function.(*Error); ok {
			if errObj.Code == "E4001" {
				// Change to "undefined function" error
				if label, ok := node.Function.(*ast.Label); ok {
					return newErrorWithLocation("E4002", label.Token.Line, label.Token.Column,
						"undefined function: '%s'", label.Value)
				}
			}
		}
		return function
	}

	// For user-defined functions with mutable (&) params, handle references specially
	if fn, ok := function.(*Function); ok {
		args := evalArgsWithReferences(node.Arguments, fn.Parameters, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(function, args, node.Token.Line, node.Token.Column)
	}

	// For builtins and other callables, evaluate arguments normally
	args := evalExpressions(node.Arguments, env)
	if len(args) == 1 && isError(args[0]) {
		return args[0]
	}

	return applyFunction(function, args, node.Token.Line, node.Token.Column)
}

// evalRefBuiltin handles the ref() builtin which creates a reference to a variable (#661)
// ref() allows explicit reference creation for shared state
func evalRefBuiltin(args []ast.Expression, env *Environment, line, column int) Object {
	if len(args) != 1 {
		return newErrorWithLocation("E7001", line, column,
			"ref() takes exactly 1 argument, got %d", len(args))
	}

	// The argument must be a variable (Label) to create a reference
	if label, ok := args[0].(*ast.Label); ok {
		// Verify the variable exists
		if _, ok := env.Get(label.Value); !ok {
			return newErrorWithLocation("E4001", label.Token.Line, label.Token.Column,
				"undefined variable: '%s'", label.Value)
		}
		// Create a reference to the variable
		return &Reference{Env: env, Name: label.Value}
	}

	// ref() requires a variable
	return newErrorWithLocation("E7003", line, column,
		"ref() argument must be a variable")
}

// evalArgsWithReferences evaluates arguments, creating References for mutable (&) params
func evalArgsWithReferences(argExprs []ast.Expression, params []*ast.Parameter, env *Environment) []Object {
	args := make([]Object, len(argExprs))

	for i, argExpr := range argExprs {
		// Check if this parameter is mutable
		if i < len(params) && params[i].Mutable {
			// Simple variable reference
			if label, ok := argExpr.(*ast.Label); ok {
				args[i] = &Reference{Env: env, Name: label.Value}
				continue
			}

			// Indexed expression (arr[i], map[k])
			if indexExpr, ok := argExpr.(*ast.IndexExpression); ok {
				container := Eval(indexExpr.Left, env)
				if isError(container) {
					return []Object{container}
				}
				index := Eval(indexExpr.Index, env)
				if isError(index) {
					return []Object{index}
				}
				// Get the variable name for display
				varName := ""
				if label, ok := indexExpr.Left.(*ast.Label); ok {
					varName = label.Value
				}
				args[i] = &Reference{
					Env:       env,
					Name:      varName,
					Container: container,
					Index:     index,
				}
				continue
			}

			// Member expression (s.field)
			if memberExpr, ok := argExpr.(*ast.MemberExpression); ok {
				container := Eval(memberExpr.Object, env)
				if isError(container) {
					return []Object{container}
				}
				// Get the variable name for display
				varName := ""
				if label, ok := memberExpr.Object.(*ast.Label); ok {
					varName = label.Value
				}
				args[i] = &Reference{
					Env:       env,
					Name:      varName,
					Container: container,
					Field:     memberExpr.Member.Value,
				}
				continue
			}
		}

		// Otherwise, evaluate normally
		evaluated := Eval(argExpr, env)
		if isError(evaluated) {
			return []Object{evaluated}
		}
		args[i] = evaluated
	}

	return args
}

func evalMemberCall(member *ast.MemberExpression, args []ast.Expression, env *Environment) Object {
	objIdent, ok := member.Object.(*ast.Label)
	if !ok {
		return newError("invalid member call")
	}

	alias := objIdent.Value

	// First check if it's a user module
	if moduleObj, ok := env.GetModule(alias); ok {
		memberName := member.Member.Value
		if fn, ok := moduleObj.Get(memberName); ok {
			// Evaluate arguments
			evalArgs := evalExpressions(args, env)
			if len(evalArgs) == 1 && isError(evalArgs[0]) {
				return evalArgs[0]
			}
			// Apply the function
			return applyFunction(fn, evalArgs, member.Token.Line, member.Token.Column)
		}
		return newErrorWithLocation("E4006", member.Token.Line, member.Token.Column,
			"'%s' not found in module '%s'", memberName, alias)
	}

	// Get the actual module name from the alias (stdlib)
	moduleName, ok := env.GetImport(alias)
	if !ok {
		return newError("module '%s' not imported", alias)
	}

	// Create a compound name like "strings.upper" using the actual module name
	fullName := moduleName + "." + member.Member.Value

	if builtin, ok := builtins[fullName]; ok {
		evalArgs := evalExpressions(args, env)
		if len(evalArgs) == 1 && isError(evalArgs[0]) {
			return evalArgs[0]
		}
		result := builtin.Fn(evalArgs...)
		// Add location info to errors from builtins
		if errObj, ok := result.(*Error); ok {
			if errObj.Line == 0 && errObj.Column == 0 {
				errObj.Line = member.Token.Line
				errObj.Column = member.Token.Column
			}
		}
		return result
	}

	// Provide helpful suggestions for common function name mistakes
	suggestions := map[string]string{
		"arrays.push":       "use arrays.append() instead",
		"strings.substring": "use strings.slice() instead",
		"strings.substr":    "use strings.slice() instead",
		"strings.length":    "use len() instead",
		"strings.size":      "use len() instead",
		"arrays.length":     "use len() instead",
		"arrays.size":       "use len() instead",
	}

	if suggestion, ok := suggestions[fullName]; ok {
		return newError("function not found: %s\n  help: %s", fullName, suggestion)
	}

	return newError("function not found: %s", fullName)
}

func applyFunction(fn Object, args []Object, line, col int) Object {
	// Check recursion depth limit
	callDepth++
	if callDepth > MAX_CALL_DEPTH {
		callDepth = 0 // Reset for future use
		return newErrorWithLocation("E5018", line, col,
			"maximum recursion depth exceeded (limit: %d)", MAX_CALL_DEPTH)
	}
	defer func() { callDepth-- }()

	switch fn := fn.(type) {
	case *Function:
		// Calculate minimum required arguments (parameters without defaults)
		minRequired := 0
		for _, param := range fn.Parameters {
			if param.DefaultValue == nil {
				minRequired++
			}
		}

		// Validate argument count
		if len(args) < minRequired || len(args) > len(fn.Parameters) {
			if minRequired == len(fn.Parameters) {
				return newErrorWithLocation("E5008", line, col,
					"wrong number of arguments: expected %d, got %d", len(fn.Parameters), len(args))
			}
			return newErrorWithLocation("E5008", line, col,
				"wrong number of arguments: expected %d to %d, got %d", minRequired, len(fn.Parameters), len(args))
		}
		extendedEnv := extendFunctionEnv(fn, args)

		// Debug hook: push call frame
		if dbg := debugger.GetGlobalDebugger(); dbg != nil && dbg.IsEnabled() {
			funcName := "<function>"
			loc := &ast.Location{
				File:   fn.File,
				Line:   line,
				Column: col,
			}
			dbg.PushFrame(funcName, fn.Body, extendedEnv, loc)
			defer dbg.PopFrame(nil)
		}

		// Save current file and set function's file as current for error reporting
		var oldFile string
		if globalEvalContext != nil && fn.File != "" {
			oldFile = globalEvalContext.CurrentFile
			globalEvalContext.CurrentFile = fn.File
		}

		evaluated := Eval(fn.Body, extendedEnv)

		// Restore current file
		if globalEvalContext != nil && fn.File != "" {
			globalEvalContext.CurrentFile = oldFile
		}

		result := unwrapReturnValue(evaluated)

		// Validate return type if function declares one
		if len(fn.ReturnTypes) > 0 && !isError(result) {
			if err := validateReturnType(result, fn.ReturnTypes, line, col); err != nil {
				return err
			}
		}
		return result

	case *Builtin:
		result := fn.Fn(args...)
		// Add location info to errors from builtins
		if errObj, ok := result.(*Error); ok {
			if errObj.Line == 0 && errObj.Column == 0 {
				errObj.Line = line
				errObj.Column = col
			}
		}
		return result

	default:
		return newErrorWithLocation("E3015", line, col, "not a function: %s", fn.Type())
	}
}

// validateReturnType checks if the returned value matches the declared return type
func validateReturnType(result Object, expectedTypes []string, line, col int) *Error {
	// Handle multiple return values
	if retVal, ok := result.(*ReturnValue); ok {
		if len(retVal.Values) != len(expectedTypes) {
			return newErrorWithLocation("E5008", line, col,
				"wrong number of return values: expected %d, got %d", len(expectedTypes), len(retVal.Values))
		}
		for i, val := range retVal.Values {
			// Check for negative literal to unsigned type first
			if err := checkNegativeToUnsigned(val, expectedTypes[i], line, col); err != nil {
				return err
			}
			if !typeMatches(val, expectedTypes[i]) {
				return createTypeMismatchError(val, expectedTypes[i], line, col)
			}
		}
		return nil
	}

	// Single return value
	if len(expectedTypes) == 1 {
		// Check for negative literal to unsigned type first
		if err := checkNegativeToUnsigned(result, expectedTypes[0], line, col); err != nil {
			return err
		}
		if !typeMatches(result, expectedTypes[0]) {
			return createTypeMismatchError(result, expectedTypes[0], line, col)
		}
	}
	return nil
}

// createTypeMismatchError creates an appropriate error for type mismatches
// with special handling for signed/unsigned integer mismatches
func createTypeMismatchError(val Object, expectedType string, line, col int) *Error {
	actualType := objectTypeToEZ(val)

	// Check for signed → unsigned mismatch
	if isSignedIntegerType(actualType) && isUnsignedIntegerType(expectedType) {
		return newErrorWithLocation("E3019", line, col,
			"cannot return signed type '%s' where unsigned type '%s' is expected (signed values may be negative)",
			actualType, expectedType)
	}

	// Generic type mismatch
	return newErrorWithLocation("E5012", line, col,
		"return type mismatch: expected %s, got %s", expectedType, actualType)
}

// checkNegativeToUnsigned checks if a negative value is being returned to an unsigned type
// This catches cases like `return -1` where the function returns u8
func checkNegativeToUnsigned(val Object, expectedType string, line, col int) *Error {
	if intVal, ok := val.(*Integer); ok {
		if isUnsignedIntegerType(expectedType) && intVal.Value.Sign() < 0 {
			return newErrorWithLocation("E3020", line, col,
				"cannot return negative value %s to unsigned type '%s'", intVal.Value.String(), expectedType)
		}
	}
	return nil
}

// isSignedIntegerType checks if a type is in the signed integer family
func isSignedIntegerType(typeName string) bool {
	switch typeName {
	case "i8", "i16", "i32", "i64", "i128", "i256", "int":
		return true
	}
	return false
}

// isUnsignedIntegerType checks if a type is in the unsigned integer family
func isUnsignedIntegerType(typeName string) bool {
	switch typeName {
	case "u8", "u16", "u32", "u64", "u128", "u256", "uint":
		return true
	}
	return false
}

// isIntegerType checks if a type is any integer type (signed or unsigned)
func isIntegerType(typeName string) bool {
	return isSignedIntegerType(typeName) || isUnsignedIntegerType(typeName)
}

// typeMatches checks if an object matches an EZ type name
// Implements signed/unsigned integer family compatibility rules:
// - Signed family: i8, i16, i32, i64, i128, i256, int (all compatible with each other)
// - Unsigned family: u8, u16, u32, u64, u128, u256, uint (all compatible with each other)
// - Unsigned → Signed: OK (unsigned is never negative, always safe)
// - Signed → Unsigned: ERROR (signed could be negative at runtime)
// - Positive literal → Either: OK (non-negative literals work for both)
func typeMatches(obj Object, ezType string) bool {
	actualType := objectTypeToEZ(obj)

	// nil is compatible with any struct type (like Error)
	if actualType == "nil" {
		// nil matches nil, or any non-primitive type
		// For now, we'll accept nil for any type except explicit primitives
		// This allows: return nil as Error
		return ezType == "nil" || ezType == "Error" || ezType == "error" || ezType == "array" ||
			(ezType != "int" && ezType != "float" && ezType != "string" &&
				ezType != "bool" && ezType != "char" && !isIntegerType(ezType))
	}

	// Exact match
	if actualType == ezType {
		return true
	}

	// error/Error are interchangeable (error is alias for Error struct)
	if (actualType == "error" && ezType == "Error") || (actualType == "Error" && ezType == "error") {
		return true
	}

	// Handle module-prefixed types (e.g., utils.Hero vs Hero)
	// Strip module prefix and compare base type names
	actualBase := stripModulePrefix(actualType)
	expectedBase := stripModulePrefix(ezType)
	if actualBase == expectedBase && (actualBase != actualType || expectedBase != ezType) {
		// Base names match and at least one had a module prefix
		return true
	}

	// Integer family compatibility rules
	if isIntegerType(actualType) && isIntegerType(ezType) {
		// Within same family: always OK
		if isSignedIntegerType(actualType) && isSignedIntegerType(ezType) {
			return true
		}
		if isUnsignedIntegerType(actualType) && isUnsignedIntegerType(ezType) {
			return true
		}

		// Unsigned → Signed: OK (unsigned values are always valid for signed)
		if isUnsignedIntegerType(actualType) && isSignedIntegerType(ezType) {
			return true
		}

		// Signed → Unsigned: Check if value is non-negative
		// Positive literals (or any non-negative value) can go to unsigned types
		if isSignedIntegerType(actualType) && isUnsignedIntegerType(ezType) {
			if intVal, ok := obj.(*Integer); ok {
				// Non-negative signed value is safe for unsigned
				if intVal.Value.Sign() >= 0 {
					return true
				}
			}
			// Negative value or unknown - not allowed
			return false
		}
	}

	return false
}

// stripModulePrefix removes the module prefix from a type name
// e.g., "utils.Hero" -> "Hero", "Hero" -> "Hero"
func stripModulePrefix(typeName string) string {
	if idx := strings.LastIndex(typeName, "."); idx != -1 {
		return typeName[idx+1:]
	}
	return typeName
}

// objectTypeToEZ converts Object type to EZ language type name
func objectTypeToEZ(obj Object) string {
	switch v := obj.(type) {
	case *Integer:
		return v.GetDeclaredType()
	case *Float:
		return "float"
	case *String:
		return "string"
	case *Boolean:
		return "bool"
	case *Array:
		// Return typed array format if element type is known
		if v.ElementType != "" {
			return "[" + v.ElementType + "]"
		}
		return "array"
	case *Struct:
		// Return the specific struct type name (e.g., "Person")
		if v.TypeName != "" {
			return v.TypeName
		}
		return "struct"
	case *EnumValue:
		// Return the enum type name (e.g., "COLOR")
		return v.EnumType
	case *Nil:
		return "nil"
	case *Function:
		return "function"
	case *ReturnValue:
		// Handle return values - format as tuple if multiple values
		if len(v.Values) == 1 {
			return objectTypeToEZ(v.Values[0])
		}
		// Multiple return values - format as tuple
		types := make([]string, len(v.Values))
		for i, val := range v.Values {
			types[i] = objectTypeToEZ(val)
		}
		return "(" + strings.Join(types, ", ") + ")"
	default:
		return string(obj.Type())
	}
}

func extendFunctionEnv(fn *Function, args []Object) *Environment {
	env := NewEnclosedEnvironment(fn.Env)

	for i, param := range fn.Parameters {
		var value Object
		if i < len(args) {
			// Use provided argument
			value = args[i]
		} else if param.DefaultValue != nil {
			// Evaluate default value in the function's closure environment
			value = Eval(param.DefaultValue, fn.Env)
		}
		if value != nil {
			// Use parameter's Mutable field: & params are mutable, non-& params are immutable
			env.Set(param.Name.Value, value, param.Mutable)
		}
	}

	return env
}

func unwrapReturnValue(obj Object) Object {
	if returnValue, ok := obj.(*ReturnValue); ok {
		if len(returnValue.Values) == 1 {
			return returnValue.Values[0]
		}
		// For multiple returns, keep the ReturnValue intact
		// so it can be unpacked by variable declaration
		return returnValue
	}
	return obj
}

func evalMapLiteral(node *ast.MapValue, env *Environment) Object {
	mapObj := NewMap()

	for _, pair := range node.Pairs {
		key := Eval(pair.Key, env)
		if isError(key) {
			return key
		}

		// Validate that the key is hashable
		if _, ok := HashKey(key); !ok {
			return newError("unusable as map key: %s", key.Type())
		}

		value := Eval(pair.Value, env)
		if isError(value) {
			return value
		}

		mapObj.Set(key, value)
	}

	return mapObj
}

func evalInterpolatedString(node *ast.InterpolatedString, env *Environment) Object {
	var result strings.Builder

	for _, part := range node.Parts {
		// Evaluate the part
		val := Eval(part, env)
		if isError(val) {
			return val
		}

		// Convert to string representation
		// For strings, use the raw value (not quoted Inspect())
		switch v := val.(type) {
		case *String:
			result.WriteString(v.Value)
		default:
			result.WriteString(val.Inspect())
		}
	}

	return &String{Value: result.String(), Mutable: true}
}

func evalStructValue(node *ast.StructValue, env *Environment) Object {
	typeName := node.Name.Value
	var structDef *StructDef
	var sourceModule *ModuleObject // Track source module for nested struct lookup
	var ok bool

	// Check for qualified type name (module.TypeName)
	if strings.Contains(typeName, ".") {
		parts := strings.SplitN(typeName, ".", 2)
		moduleName := parts[0]
		structName := parts[1]

		// Look up the module
		if moduleObj, modOk := env.GetModule(moduleName); modOk {
			structDef, ok = moduleObj.GetStructDef(structName)
			if !ok {
				return newErrorWithLocation("E3002", node.Token.Line, node.Token.Column,
					"undefined type '%s' in module '%s'", structName, moduleName)
			}
			sourceModule = moduleObj // Remember the source module for nested structs
		} else {
			return newErrorWithLocation("E4007", node.Token.Line, node.Token.Column,
				"module '%s' not imported", moduleName)
		}
	} else {
		// Look up the struct definition in the current environment
		structDef, ok = env.GetStructDef(typeName)
		if !ok {
			// Not found locally, check modules from "using" directives
			var foundModules []string
			var foundStructDef *StructDef
			var foundModule *ModuleObject

			for _, alias := range env.GetUsing() {
				if moduleObj, modOk := env.GetModule(alias); modOk {
					if sd, sdOk := moduleObj.GetStructDef(typeName); sdOk {
						foundModules = append(foundModules, alias)
						foundStructDef = sd
						foundModule = moduleObj
					}
				}
			}

			// Ambiguity check
			if len(foundModules) > 1 {
				moduleList := strings.Join(foundModules, ", ")
				return newErrorWithLocation("E3002", node.Token.Line, node.Token.Column,
					"type '%s' found in multiple modules: %s. Use explicit module prefix: %s.%s",
					typeName, moduleList, foundModules[0], typeName)
			}

			// Found in exactly one module
			if len(foundModules) == 1 {
				structDef = foundStructDef
				sourceModule = foundModule // Remember the source module
			} else {
				return newErrorWithLocation("E3002", node.Token.Line, node.Token.Column,
					"undefined type: '%s'", typeName)
			}
		}
	}

	// Create a new struct with default values for all fields first
	fields := make(map[string]Object)
	for fieldName, fieldType := range structDef.Fields {
		fields[fieldName] = getDefaultValueWithEnv(fieldType, env, sourceModule)
	}

	// Override with explicitly provided field values
	for fieldName, fieldExpr := range node.Fields {
		val := Eval(fieldExpr, env)
		if isError(val) {
			return val
		}
		// If the value is an empty array but the field type is a map, create an empty map instead
		if arr, ok := val.(*Array); ok && len(arr.Elements) == 0 {
			if fieldType, hasField := structDef.Fields[fieldName]; hasField {
				if strings.HasPrefix(fieldType, "map[") {
					val = &Map{Pairs: []*MapPair{}, Index: make(map[string]int), Mutable: true}
				}
			}
		}
		fields[fieldName] = val
	}

	return &Struct{
		TypeName:  structDef.Name,
		Fields:    fields,
		FieldTags: structDef.FieldTags,
	}
}

func evalNewExpression(node *ast.NewExpression, env *Environment) Object {
	typeName := node.TypeName.Value
	var structDef *StructDef
	var sourceModule *ModuleObject // Track source module for nested struct lookup
	var ok bool

	// Check for qualified type name (module.TypeName)
	if strings.Contains(typeName, ".") {
		parts := strings.SplitN(typeName, ".", 2)
		moduleName := parts[0]
		structName := parts[1]

		// Look up the module
		if moduleObj, modOk := env.GetModule(moduleName); modOk {
			structDef, ok = moduleObj.GetStructDef(structName)
			if !ok {
				return newErrorWithLocation("E3002", node.Token.Line, node.Token.Column,
					"undefined type '%s' in module '%s'", structName, moduleName)
			}
			sourceModule = moduleObj // Remember the source module for nested structs
		} else {
			return newErrorWithLocation("E4007", node.Token.Line, node.Token.Column,
				"module '%s' not imported", moduleName)
		}
	} else {
		// Look up the struct definition in the current environment
		structDef, ok = env.GetStructDef(typeName)
		if !ok {
			// Not found locally, check modules from "using" directives
			var foundModules []string
			var foundStructDef *StructDef
			var foundModule *ModuleObject

			for _, alias := range env.GetUsing() {
				if moduleObj, modOk := env.GetModule(alias); modOk {
					if sd, sdOk := moduleObj.GetStructDef(typeName); sdOk {
						foundModules = append(foundModules, alias)
						foundStructDef = sd
						foundModule = moduleObj
					}
				}
			}

			// Ambiguity check
			if len(foundModules) > 1 {
				moduleList := strings.Join(foundModules, ", ")
				return newErrorWithLocation("E3002", node.Token.Line, node.Token.Column,
					"type '%s' found in multiple modules: %s. Use explicit module prefix: %s.%s",
					typeName, moduleList, foundModules[0], typeName)
			}

			// Found in exactly one module
			if len(foundModules) == 1 {
				structDef = foundStructDef
				sourceModule = foundModule // Remember the source module
			} else {
				return newErrorWithLocation("E3002", node.Token.Line, node.Token.Column,
					"undefined type: '%s'", typeName)
			}
		}
	}

	// Create a new struct with default values for all fields
	fields := make(map[string]Object)
	for fieldName, fieldType := range structDef.Fields {
		fields[fieldName] = getDefaultValueWithEnv(fieldType, env, sourceModule)
	}

	return &Struct{
		TypeName: structDef.Name,
		Fields:   fields,
	}
}

// getDefaultValueWithEnv returns the default zero value for a given type,
// with access to the environment for looking up struct definitions.
// sourceModule is optional - if provided, nested struct types will be looked up
// in that module first (for cross-module struct initialization).
func getDefaultValueWithEnv(typeName string, env *Environment, sourceModule *ModuleObject) Object {
	// Check if it's a dynamic array type (starts with '[' but doesn't contain ',')
	if len(typeName) > 0 && typeName[0] == '[' && !strings.Contains(typeName, ",") {
		return &Array{Elements: []Object{}}
	}

	switch typeName {
	case "int", "i8", "i16", "i32", "i64", "i128", "i256":
		return &Integer{Value: big.NewInt(0)}
	case "uint", "u8", "u16", "u32", "u64", "u128", "u256", "byte":
		return &Integer{Value: big.NewInt(0)}
	case "float", "f32", "f64":
		return &Float{Value: 0.0}
	case "string":
		return &String{Value: "", Mutable: true}
	case "bool":
		return FALSE
	case "char":
		return &Char{Value: '\x00'}
	default:
		// First, check the source module if provided (for cross-module nested structs)
		if sourceModule != nil {
			if structDef, sdOk := sourceModule.GetStructDef(typeName); sdOk {
				fields := make(map[string]Object)
				for fieldName, fieldType := range structDef.Fields {
					fields[fieldName] = getDefaultValueWithEnv(fieldType, env, sourceModule)
				}
				return &Struct{
					TypeName: structDef.Name,
					Fields:   fields,
				}
			}
		}

		// Check if it's a struct type in the current environment
		if structDef, ok := env.GetStructDef(typeName); ok {
			fields := make(map[string]Object)
			for fieldName, fieldType := range structDef.Fields {
				fields[fieldName] = getDefaultValueWithEnv(fieldType, env, sourceModule)
			}
			return &Struct{
				TypeName: structDef.Name,
				Fields:   fields,
			}
		}

		// Check modules from "using" directives
		for _, alias := range env.GetUsing() {
			if moduleObj, modOk := env.GetModule(alias); modOk {
				if structDef, sdOk := moduleObj.GetStructDef(typeName); sdOk {
					fields := make(map[string]Object)
					for fieldName, fieldType := range structDef.Fields {
						fields[fieldName] = getDefaultValueWithEnv(fieldType, env, moduleObj)
					}
					return &Struct{
						TypeName: structDef.Name,
						Fields:   fields,
					}
				}
			}
		}

		// For unknown types, default to nil
		return NIL
	}
}

func evalMemberExpression(node *ast.MemberExpression, env *Environment) Object {
	// Check if this is a module member access
	if objIdent, ok := node.Object.(*ast.Label); ok {
		alias := objIdent.Value

		// First check if it's a user module
		if moduleObj, ok := env.GetModule(alias); ok {
			if member, ok := moduleObj.Get(node.Member.Value); ok {
				return member
			}
			return newErrorWithLocation("E4006", node.Token.Line, node.Token.Column,
				"'%s' not found in module '%s'", node.Member.Value, alias)
		}

		// Then check stdlib imports
		if moduleName, ok := env.GetImport(alias); ok {
			fullName := moduleName + "." + node.Member.Value
			if builtin, ok := builtins[fullName]; ok {
				// For constants (zero-arg functions), call them immediately
				return builtin.Fn()
			}
			return newErrorWithLocation("E4006", node.Token.Line, node.Token.Column,
				"'%s' not found in module '%s'", node.Member.Value, alias)
		}
	}

	obj := Eval(node.Object, env)
	if isError(obj) {
		return obj
	}

	// Check for nil reference
	if obj.Type() == NIL_OBJ {
		return newErrorWithLocation("E4010", node.Token.Line, node.Token.Column,
			"nil reference: cannot access member '%s' of nil", node.Member.Value)
	}

	if structObj, ok := obj.(*Struct); ok {
		if val, ok := structObj.Fields[node.Member.Value]; ok {
			// Propagate mutability to nested structs
			if nestedStruct, ok := val.(*Struct); ok {
				nestedStruct.Mutable = structObj.Mutable
			}
			// Propagate mutability to nested arrays
			if arr, ok := val.(*Array); ok {
				arr.Mutable = structObj.Mutable
			}
			return val
		}
		return newErrorWithLocation("E4003", node.Token.Line, node.Token.Column,
			"field '%s' not found", node.Member.Value)
	}

	// Check for enum value access (e.g., STATUS.ACTIVE)
	if enumObj, ok := obj.(*Enum); ok {
		if val, ok := enumObj.Values[node.Member.Value]; ok {
			// Wrap the value in an EnumValue to preserve type information
			return &EnumValue{
				EnumType: enumObj.Name,
				Name:     node.Member.Value,
				Value:    val,
			}
		}
		return newErrorWithLocation("E4004", node.Token.Line, node.Token.Column,
			"enum value '%s' not found in enum '%s'", node.Member.Value, enumObj.Name)
	}

	return newErrorWithLocation("E4011", node.Token.Line, node.Token.Column,
		"member access not supported on type %s", obj.Type())
}

func nativeBoolToBooleanObject(input bool) *Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func isTruthy(obj Object) bool {
	switch v := obj.(type) {
	case *Nil:
		return false
	case *Boolean:
		return v.Value
	default:
		return true
	}
}

// copyByDefault implements copy-by-default semantics for complex types (#661)
// Returns a deep copy of structs, arrays, and maps.
// References are returned as-is (that's the point of ref()).
// Primitives are returned as-is (they're immutable anyway).
func copyByDefault(val Object) Object {
	switch v := val.(type) {
	case *Reference:
		// References are NOT copied - that's the point of ref()
		return v
	case *Struct:
		// Deep copy struct
		newFields := make(map[string]Object)
		for key, fieldVal := range v.Fields {
			newFields[key] = copyByDefault(fieldVal)
		}
		return &Struct{
			TypeName:  v.TypeName,
			Fields:    newFields,
			FieldTags: v.FieldTags,
			Mutable:   v.Mutable,
		}
	case *Array:
		// Deep copy array
		newElements := make([]Object, len(v.Elements))
		for i, elem := range v.Elements {
			newElements[i] = copyByDefault(elem)
		}
		return &Array{
			Elements:    newElements,
			Mutable:     v.Mutable,
			ElementType: v.ElementType,
		}
	case *Map:
		// Deep copy map
		newMap := NewMap()
		for _, pair := range v.Pairs {
			newMap.Set(pair.Key, copyByDefault(pair.Value))
		}
		newMap.Mutable = v.Mutable
		return newMap
	default:
		// Primitives and other types are returned as-is
		return val
	}
}

func isError(obj Object) bool {
	if obj != nil {
		return obj.Type() == ERROR_OBJ
	}
	return false
}

// getFunctionObject retrieves the Function object from a call expression
func getFunctionObject(call *ast.CallExpression, env *Environment) *Function {
	if label, ok := call.Function.(*ast.Label); ok {
		if obj, ok := env.Get(label.Value); ok {
			if fn, ok := obj.(*Function); ok {
				return fn
			}
		}
	}
	return nil
}

func newError(format string, a ...interface{}) *Error {
	return &Error{Message: fmt.Sprintf(format, a...)}
}

// getStatementFile extracts the source file from a statement's token
// This is used to track the current file during multi-file module evaluation
func getStatementFile(stmt ast.Statement) string {
	switch s := stmt.(type) {
	case *ast.VariableDeclaration:
		return s.Token.File
	case *ast.FunctionDeclaration:
		return s.Token.File
	case *ast.StructDeclaration:
		return s.Token.File
	case *ast.EnumDeclaration:
		return s.Token.File
	case *ast.AssignmentStatement:
		return s.Token.File
	case *ast.ReturnStatement:
		return s.Token.File
	case *ast.ExpressionStatement:
		return s.Token.File
	case *ast.BlockStatement:
		return s.Token.File
	case *ast.IfStatement:
		return s.Token.File
	case *ast.WhenStatement:
		return s.Token.File
	case *ast.ForStatement:
		return s.Token.File
	case *ast.ForEachStatement:
		return s.Token.File
	case *ast.WhileStatement:
		return s.Token.File
	case *ast.LoopStatement:
		return s.Token.File
	case *ast.BreakStatement:
		return s.Token.File
	case *ast.ContinueStatement:
		return s.Token.File
	case *ast.ImportStatement:
		return s.Token.File
	case *ast.UsingStatement:
		return s.Token.File
	case *ast.ModuleDeclaration:
		return s.Token.File
	}
	return ""
}

// newErrorWithLocation creates an error with line/column info
func newErrorWithLocation(code string, line, column int, format string, a ...interface{}) *Error {
	file := ""
	if globalEvalContext != nil {
		file = globalEvalContext.CurrentFile
	}
	return &Error{
		Message: fmt.Sprintf(format, a...),
		Code:    code,
		Line:    line,
		Column:  column,
		File:    file,
	}
}

// isSteppableStatement returns true if the node represents a statement we should step through
func isSteppableStatement(node ast.Node) bool {
	switch node.(type) {
	case *ast.VariableDeclaration,
		*ast.AssignmentStatement,
		*ast.ExpressionStatement,
		*ast.ReturnStatement,
		*ast.IfStatement,
		*ast.WhileStatement,
		*ast.ForStatement,
		*ast.ForEachStatement,
		*ast.LoopStatement,
		*ast.BreakStatement,
		*ast.ContinueStatement:
		return true
	}
	return false
}
