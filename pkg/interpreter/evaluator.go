package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"strings"

	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/errors"
)

var (
	NIL   = &Nil{}
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
)

// validModules lists all available standard library modules
var validModules = map[string]bool{
	"std":     true, // Standard I/O functions (println, print, read_int)
	"math":    true, // Math functions (upcoming)
	"string":  true, // String manipulation (upcoming)
	"strings": true, // String utilities (upcoming)
	"arrays":  true, // Array utilities (upcoming)
	"time":    true, // Time functions (upcoming)
}

// isValidModule checks if a module name is valid (either standard library or user-created)
func isValidModule(moduleName string) bool {
	// Check standard library modules
	if validModules[moduleName] {
		return true
	}

	// TODO: Check for user-created modules (e.g., local .ez files)
	// For now, we only validate against standard library

	return false
}

func Eval(node ast.Node, env *Environment) Object {
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
					"return value from function not used (use @ignore to discard)")
			}
			// Also check if the function has declared return types but result is not NIL
			if result != NIL {
				// Check if it's a user function with return types
				if fnObj := getFunctionObject(call, env); fnObj != nil && len(fnObj.ReturnTypes) > 0 {
					return newErrorWithLocation("E4009", call.Token.Line, call.Token.Column,
						"return value from function not used (use @ignore to discard)")
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
		// Register the struct type definition
		fields := make(map[string]string)
		for _, field := range node.Fields {
			fields[field.Name.Value] = field.TypeName
		}
		env.RegisterStructDef(node.Name.Value, &StructDef{
			Name:   node.Name.Value,
			Fields: fields,
		})
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
				// Validate that the module exists
				if !isValidModule(item.Module) {
					return newErrorWithLocation("E6002", node.Token.Line, node.Token.Column,
						"module '%s' not found", item.Module)
				}

				alias := item.Alias
				if alias == "" {
					alias = item.Module
				}
				env.Import(alias, item.Module)
			}
		} else {
			// Backward compatibility: handle single import using old fields
			if !isValidModule(node.Module) {
				return newErrorWithLocation("E6002", node.Token.Line, node.Token.Column,
					"module '%s' not found", node.Module)
			}

			alias := node.Alias
			if alias == "" {
				alias = node.Module
			}
			env.Import(alias, node.Module)
		}
		return NIL

	case *ast.UsingStatement:
		// Bring the module(s) functions into scope (function-scoped using)
		for _, module := range node.Modules {
			alias := module.Value
			// Verify the module was imported
			if _, ok := env.GetImport(alias); !ok {
				return newErrorWithLocation("E6004", node.Token.Line, node.Token.Column,
					"cannot use '%s': module not imported", alias)
			}
			env.Use(alias)
		}
		return NIL

	// Expressions
	case *ast.IntegerValue:
		return &Integer{Value: node.Value}

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

		// Handle index access with improved error messages
		idx, ok := index.(*Integer)
		if !ok {
			return newErrorWithLocation("E9003", node.Token.Line, node.Token.Column,
				"index must be an integer, got %s", index.Type())
		}

		switch obj := left.(type) {
		case *Array:
			arrLen := int64(len(obj.Elements))
			if idx.Value < 0 || idx.Value >= arrLen {
				if arrLen == 0 {
					return newErrorWithLocation("E9004", node.Token.Line, node.Token.Column,
						"index out of bounds: array is empty (length 0)\n\n"+
							"Attempted to access index %d, but array has no elements\n"+
							"Hint: Use arrays.push() to add elements before accessing by index", idx.Value)
				}
				return newErrorWithLocation("E9001", node.Token.Line, node.Token.Column,
					"index out of bounds: attempted to access index %d, but valid range is 0-%d",
					idx.Value, arrLen-1)
			}
			return obj.Elements[idx.Value]

		case *String:
			strLen := int64(len(obj.Value))
			if idx.Value < 0 || idx.Value >= strLen {
				if strLen == 0 {
					return newErrorWithLocation("E10004", node.Token.Line, node.Token.Column,
						"index out of bounds: string is empty (length 0)\n\n"+
							"Attempted to access index %d", idx.Value)
				}
				return newErrorWithLocation("E10003", node.Token.Line, node.Token.Column,
					"index out of bounds: attempted to access index %d, but valid range is 0-%d",
					idx.Value, strLen-1)
			}
			return &Char{Value: rune(obj.Value[idx.Value])}

		default:
			return newErrorWithLocation("E5015", node.Token.Line, node.Token.Column,
				"index operator not supported for %s", left.Type())
		}

	case *ast.MemberExpression:
		return evalMemberExpression(node, env)

	case *ast.NewExpression:
		return evalNewExpression(node, env)

	case *ast.RangeExpression:
		// Range is typically used in for loops, not standalone
		return newError("range() can only be used in for loops")
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
			// Verify the module was imported
			if _, ok := env.GetImport(alias); !ok {
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

		result = Eval(stmt, env)

		switch result := result.(type) {
		case *ReturnValue:
			return result.Values[0]
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

		// Validate type compatibility if a type is declared
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
				if _, ok := val.(*Array); !ok {
					return newErrorWithLocation("E3018", node.Token.Line, node.Token.Column,
						"type mismatch: expected array type '%s', got %s\n\n"+
							"Array values must be enclosed in curly braces {}\n"+
							"Example: const arr %s = {%s}",
						node.TypeName, getEZTypeName(val), node.TypeName, val.Inspect())
				}
			}

			// If we have an integer value, set the declared type
			// and validate signed/unsigned compatibility
			if intVal, ok := val.(*Integer); ok {
				// Check for negative value assigned to unsigned type
				if isUnsignedIntegerType(node.TypeName) && intVal.Value < 0 {
					return newErrorWithLocation("E3020", node.Token.Line, node.Token.Column,
						"cannot assign negative value %d to unsigned type '%s'", intVal.Value, node.TypeName)
				}
				// Set the declared type on the integer
				intVal.DeclaredType = node.TypeName
			}
		}
	} else if node.TypeName != "" {
		// Variable declared with type but no value - provide appropriate default
		// Check if it's a dynamic array type (starts with '[' but doesn't contain ',')
		// Dynamic arrays: [int], [string], etc. - can be declared without values
		// Fixed-size arrays: [int,3], [string,5], etc. - MUST be initialized with values
		if len(node.TypeName) > 0 && node.TypeName[0] == '[' && !strings.Contains(node.TypeName, ",") {
			// Initialize dynamic array to empty array instead of NIL
			val = &Array{Elements: []Object{}, Mutable: true}
		} else if isIntegerType(node.TypeName) {
			// Handle all integer types (signed and unsigned)
			val = &Integer{Value: 0, DeclaredType: node.TypeName}
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
			// For fixed-size arrays and other types, remain NIL
			default:
				val = NIL
			}
		}
	}

	// Handle multiple assignment: temp result, err = function()
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

		for i, name := range node.Names {
			// Skip @ignore
			if name.Value == "@ignore" {
				continue
			}
			env.Set(name.Value, returnVal.Values[i], node.Mutable)
		}
		return NIL
	}

	// Single variable assignment
	env.Set(node.Name.Value, val, node.Mutable)
	return NIL
}

func evalAssignment(node *ast.AssignmentStatement, env *Environment) Object {
	val := Eval(node.Value, env)
	if isError(val) {
		return val
	}

	switch target := node.Name.(type) {
	case *ast.Label:
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
		// Array or string index assignment
		// First check if the container variable is mutable
		if ident, ok := target.Left.(*ast.Label); ok {
			isMutable, exists := env.IsMutable(ident.Value)
			if exists && !isMutable {
				return newErrorWithLocation("E5014", node.Token.Line, node.Token.Column,
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

		index, ok := idx.(*Integer)
		if !ok {
			return newError("index must be integer, got %s", idx.Type())
		}

		switch obj := container.(type) {
		case *Array:
			arrLen := int64(len(obj.Elements))
			if index.Value < 0 || index.Value >= arrLen {
				if arrLen == 0 {
					return newErrorWithLocation("E9004", node.Token.Line, node.Token.Column,
						"index out of bounds: array is empty (length 0)\n\n"+
							"Attempted to assign to index %d, but array has no elements\n"+
							"Hint: Use arrays.push() to add elements before accessing by index", index.Value)
				}
				return newErrorWithLocation("E9001", node.Token.Line, node.Token.Column,
					"index out of bounds: attempted to assign to index %d, but valid range is 0-%d",
					index.Value, arrLen-1)
			}

			// Handle compound assignment
			if node.Operator != "=" {
				oldVal := obj.Elements[index.Value]
				val = evalCompoundAssignment(node.Operator, oldVal, val, node.Token.Line, node.Token.Column)
				if isError(val) {
					return val
				}
			}

			obj.Elements[index.Value] = val

		case *String:
			// String mutation - verify the value is a character
			charObj, ok := val.(*Char)
			if !ok {
				return newErrorWithLocation("E3001", node.Token.Line, node.Token.Column,
					"can only assign character to string index, got %s", val.Type())
			}
			strLen := int64(len(obj.Value))
			if index.Value < 0 || index.Value >= strLen {
				if strLen == 0 {
					return newErrorWithLocation("E10004", node.Token.Line, node.Token.Column,
						"index out of bounds: string is empty (length 0)\n\n"+
							"Attempted to assign to index %d", index.Value)
				}
				return newErrorWithLocation("E10003", node.Token.Line, node.Token.Column,
					"index out of bounds: attempted to assign to index %d, but valid range is 0-%d",
					index.Value, strLen-1)
			}
			// Convert string to rune slice, modify, convert back
			runes := []rune(obj.Value)
			runes[index.Value] = charObj.Value
			obj.Value = string(runes)

		default:
			return newError("index operator not supported: %s", container.Type())
		}

	case *ast.MemberExpression:
		// Struct field assignment
		obj := Eval(target.Object, env)
		if isError(obj) {
			return obj
		}
		structObj, ok := obj.(*Struct)
		if !ok {
			return newError("member access not supported: %s", obj.Type())
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

	startObj := Eval(rangeExpr.Start, env)
	if isError(startObj) {
		return startObj
	}
	endObj := Eval(rangeExpr.End, env)
	if isError(endObj) {
		return endObj
	}

	start, ok := startObj.(*Integer)
	if !ok {
		return newError("range start must be integer")
	}
	end, ok := endObj.(*Integer)
	if !ok {
		return newError("range end must be integer")
	}

	loopEnv := NewEnclosedEnvironment(env)

	for i := start.Value; i < end.Value; i++ { // Exclusive end
		loopEnv.Set(node.Variable.Value, &Integer{Value: i}, true) // loop vars are mutable

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

	return newError("for_each requires array or string, got %s", collection.Type())
}

func evalEnumDeclaration(node *ast.EnumDeclaration, env *Environment) Object {
	enum := &Enum{
		Name:   node.Name.Value,
		Values: make(map[string]Object),
	}

	// Get enum attributes (type, skip, increment)
	typeName := "int"     // default
	increment := int64(1) // default increment
	var floatIncrement float64 = 1.0

	if node.Attributes != nil {
		typeName = node.Attributes.TypeName
		if node.Attributes.Skip && node.Attributes.Increment != nil {
			// Evaluate the increment expression
			incVal := Eval(node.Attributes.Increment, env)
			if intVal, ok := incVal.(*Integer); ok {
				increment = intVal.Value
				floatIncrement = float64(intVal.Value)
			} else if floatVal, ok := incVal.(*Float); ok {
				floatIncrement = floatVal.Value
			}
		}
	}

	// Compute enum values
	var currentInt int64 = 0
	var currentFloat float64 = 0.0

	for _, enumVal := range node.Values {
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
				currentInt = v.Value + increment
			case *Float:
				currentFloat = v.Value + floatIncrement
			case *String:
				// Strings don't auto-increment
			}
		} else {
			// Auto-assign value based on type
			switch typeName {
			case "int":
				enum.Values[enumVal.Name.Value] = &Integer{Value: currentInt}
				currentInt += increment
			case "float":
				enum.Values[enumVal.Name.Value] = &Float{Value: currentFloat}
				currentFloat += floatIncrement
			case "string":
				return newError("string enums require explicit values for all members")
			default:
				return newError("unsupported enum type: %s", typeName)
			}
		}
	}

	// Store the enum in the environment
	env.Set(node.Name.Value, enum, false) // enums are immutable
	return NIL
}

func evalFunctionDeclaration(node *ast.FunctionDeclaration, env *Environment) Object {
	fn := &Function{
		Parameters:  node.Parameters,
		ReturnTypes: node.ReturnTypes,
		Body:        node.Body,
		Env:         env,
	}
	env.Set(node.Name.Value, fn, false) // functions are immutable
	return NIL
}

func evalIdentifier(node *ast.Label, env *Environment) Object {
	if val, ok := env.Get(node.Value); ok {
		// Set mutability flag on arrays and strings based on variable declaration
		isMutable, _ := env.IsMutable(node.Value)
		switch obj := val.(type) {
		case *Array:
			obj.Mutable = isMutable
		case *String:
			obj.Mutable = isMutable
		}
		return val
	}

	// Check if the function is available via "using" modules
	// Detect ambiguity: if multiple modules have the same function, error
	var foundModules []string
	var foundBuiltin *Builtin

	for _, alias := range env.GetUsing() {
		if module, ok := env.GetImport(alias); ok {
			fullName := module + "." + node.Value
			if builtin, ok := builtins[fullName]; ok {
				foundModules = append(foundModules, module)
				foundBuiltin = builtin
			}
		}
	}

	// Ambiguity check
	if len(foundModules) > 1 {
		err := newErrorWithLocation("E4002", node.Token.Line, node.Token.Column,
			"function '%s' found in multiple modules", node.Value)
		// Build helpful error message
		moduleList := strings.Join(foundModules, ", ")
		err.Help = fmt.Sprintf("use explicit module prefix: %s.%s()", foundModules[0], node.Value)
		err.Message = fmt.Sprintf("function '%s' found in multiple modules: %s", node.Value, moduleList)
		return err
	}

	// Found in exactly one module
	if len(foundModules) == 1 {
		return foundBuiltin
	}

	// Check global builtins (like len, typeof, etc.)
	if builtin, ok := builtins[node.Value]; ok {
		return builtin
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
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NIL:
		return TRUE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperator(right Object) Object {
	switch obj := right.(type) {
	case *Integer:
		return &Integer{Value: -obj.Value}
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

	switch {
	case left.Type() == INTEGER_OBJ && right.Type() == INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right, line, col)
	case left.Type() == FLOAT_OBJ || right.Type() == FLOAT_OBJ:
		return evalFloatInfixExpression(operator, left, right, line, col)
	case left.Type() == STRING_OBJ && right.Type() == STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case left.Type() == CHAR_OBJ && right.Type() == CHAR_OBJ:
		return evalCharInfixExpression(operator, left, right, line, col)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case operator == "&&":
		return nativeBoolToBooleanObject(isTruthy(left) && isTruthy(right))
	case operator == "||":
		return nativeBoolToBooleanObject(isTruthy(left) || isTruthy(right))
	case operator == "in":
		return evalInOperator(left, right)
	case operator == "not_in" || operator == "!in":
		result := evalInOperator(left, right)
		if result == TRUE {
			return FALSE
		}
		return TRUE
	default:
		return newErrorWithLocation("E5015", line, col, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIntegerInfixExpression(operator string, left, right Object, line, col int) Object {
	leftVal := left.(*Integer).Value
	rightVal := right.(*Integer).Value

	switch operator {
	case "+":
		return &Integer{Value: leftVal + rightVal}
	case "-":
		return &Integer{Value: leftVal - rightVal}
	case "*":
		return &Integer{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newErrorWithLocation("E5001", line, col, "division by zero")
		}
		return &Integer{Value: leftVal / rightVal}
	case "%":
		if rightVal == 0 {
			return newErrorWithLocation("E5002", line, col, "modulo by zero")
		}
		return &Integer{Value: leftVal % rightVal}
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
		return newErrorWithLocation("E5015", line, col, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalFloatInfixExpression(operator string, left, right Object, line, col int) Object {
	var leftVal, rightVal float64

	switch l := left.(type) {
	case *Float:
		leftVal = l.Value
	case *Integer:
		leftVal = float64(l.Value)
	}

	switch r := right.(type) {
	case *Float:
		rightVal = r.Value
	case *Integer:
		rightVal = float64(r.Value)
	}

	switch operator {
	case "+":
		return &Float{Value: leftVal + rightVal}
	case "-":
		return &Float{Value: leftVal - rightVal}
	case "*":
		return &Float{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newErrorWithLocation("E5001", line, col, "division by zero")
		}
		return &Float{Value: leftVal / rightVal}
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
		return newErrorWithLocation("E5015", line, col, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
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
		return newErrorWithLocation("E5015", line, col, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalInOperator(left, right Object) Object {
	arr, ok := right.(*Array)
	if !ok {
		return newError("right operand of 'in' must be array, got %s", right.Type())
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
			return av.Value == bv.Value
		}
	case *String:
		if bv, ok := b.(*String); ok {
			return av.Value == bv.Value
		}
	case *Boolean:
		if bv, ok := b.(*Boolean); ok {
			return av.Value == bv.Value
		}
	}
	return a == b
}

func evalPostfixExpression(node *ast.PostfixExpression, env *Environment) Object {
	ident, ok := node.Left.(*ast.Label)
	if !ok {
		return newError("postfix operator requires identifier")
	}

	val, ok := env.Get(ident.Value)
	if !ok {
		return newError("identifier not found: %s", ident.Value)
	}

	intVal, ok := val.(*Integer)
	if !ok {
		return newError("postfix operator requires integer")
	}

	var newVal int64
	switch node.Operator {
	case "++":
		newVal = intVal.Value + 1
	case "--":
		newVal = intVal.Value - 1
	default:
		return newError("unknown postfix operator: %s", node.Operator)
	}

	env.Update(ident.Value, &Integer{Value: newVal})
	return intVal // Return old value (postfix behavior)
}

func evalCallExpression(node *ast.CallExpression, env *Environment) Object {
	// Handle member calls like std.println
	if member, ok := node.Function.(*ast.MemberExpression); ok {
		return evalMemberCall(member, node.Arguments, env)
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

	args := evalExpressions(node.Arguments, env)
	if len(args) == 1 && isError(args[0]) {
		return args[0]
	}

	return applyFunction(function, args, node.Token.Line, node.Token.Column)
}

func evalMemberCall(member *ast.MemberExpression, args []ast.Expression, env *Environment) Object {
	objIdent, ok := member.Object.(*ast.Label)
	if !ok {
		return newError("invalid member call")
	}

	alias := objIdent.Value

	// Get the actual module name from the alias
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

	return newError("function not found: %s", fullName)
}

func applyFunction(fn Object, args []Object, line, col int) Object {
	switch fn := fn.(type) {
	case *Function:
		// Validate argument count
		if len(args) != len(fn.Parameters) {
			return newErrorWithLocation("E5004", line, col,
				"wrong number of arguments: expected %d, got %d", len(fn.Parameters), len(args))
		}
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
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
		return newErrorWithLocation("E5015", line, col, "not a function: %s", fn.Type())
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
		if isUnsignedIntegerType(expectedType) && intVal.Value < 0 {
			return newErrorWithLocation("E3020", line, col,
				"cannot return negative value %d to unsigned type '%s'", intVal.Value, expectedType)
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
		return ezType == "nil" || ezType == "Error" || ezType == "array" ||
			(ezType != "int" && ezType != "float" && ezType != "string" &&
				ezType != "bool" && ezType != "char" && !isIntegerType(ezType))
	}

	// Exact match
	if actualType == ezType {
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
				if intVal.Value >= 0 {
					return true
				}
			}
			// Negative value or unknown - not allowed
			return false
		}
	}

	return false
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
	default:
		return string(obj.Type())
	}
}

func extendFunctionEnv(fn *Function, args []Object) *Environment {
	env := NewEnclosedEnvironment(fn.Env)

	for i, param := range fn.Parameters {
		if i < len(args) {
			env.Set(param.Name.Value, args[i], false) // params are immutable (const)
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

func evalIndexExpression(left, index Object) Object {
	switch {
	case left.Type() == ARRAY_OBJ && index.Type() == INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == STRING_OBJ && index.Type() == INTEGER_OBJ:
		return evalStringIndexExpression(left, index)
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

func evalArrayIndexExpression(array, index Object) Object {
	arrayObject := array.(*Array)
	idx := index.(*Integer).Value

	if idx < 0 || idx >= int64(len(arrayObject.Elements)) {
		return newError("index out of bounds: %d", idx)
	}

	return arrayObject.Elements[idx]
}

func evalStringIndexExpression(str, index Object) Object {
	stringObject := str.(*String)
	idx := index.(*Integer).Value

	if idx < 0 || idx >= int64(len(stringObject.Value)) {
		return newError("index out of bounds: %d", idx)
	}

	return &Char{Value: rune(stringObject.Value[idx])}
}

func evalInterpolatedString(node *ast.InterpolatedString, env *Environment) Object {
	var result strings.Builder

	for _, part := range node.Parts {
		// Evaluate the part
		val := Eval(part, env)
		if isError(val) {
			return val
		}

		// Convert to string using Inspect()
		result.WriteString(val.Inspect())
	}

	return &String{Value: result.String(), Mutable: true}
}

func evalStructValue(node *ast.StructValue, env *Environment) Object {
	// Create a new struct with the given fields
	fields := make(map[string]Object)

	// Evaluate each field expression
	for fieldName, fieldExpr := range node.Fields {
		val := Eval(fieldExpr, env)
		if isError(val) {
			return val
		}
		fields[fieldName] = val
	}

	return &Struct{
		TypeName: node.Name.Value,
		Fields:   fields,
	}
}

func evalNewExpression(node *ast.NewExpression, env *Environment) Object {
	// Look up the struct definition
	structDef, ok := env.GetStructDef(node.TypeName.Value)
	if !ok {
		return newErrorWithLocation("E3002", node.Token.Line, node.Token.Column,
			"undefined type: '%s'", node.TypeName.Value)
	}

	// Create a new struct with default values for all fields
	fields := make(map[string]Object)
	for fieldName, fieldType := range structDef.Fields {
		fields[fieldName] = getDefaultValue(fieldType)
	}

	return &Struct{
		TypeName: structDef.Name,
		Fields:   fields,
	}
}

// getDefaultValue returns the default zero value for a given type
func getDefaultValue(typeName string) Object {
	// Check if it's a dynamic array type (starts with '[' but doesn't contain ',')
	if len(typeName) > 0 && typeName[0] == '[' && !strings.Contains(typeName, ",") {
		return &Array{Elements: []Object{}}
	}

	switch typeName {
	case "int":
		return &Integer{Value: 0}
	case "float":
		return &Float{Value: 0.0}
	case "string":
		return &String{Value: "", Mutable: true}
	case "bool":
		return FALSE
	case "char":
		return &Char{Value: '\x00'}
	default:
		// For other types (structs, fixed-size arrays, etc.), default to nil
		return NIL
	}
}

func evalMemberExpression(node *ast.MemberExpression, env *Environment) Object {
	// Check if this is a module constant access (like math.pi)
	if objIdent, ok := node.Object.(*ast.Label); ok {
		alias := objIdent.Value
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
		return newErrorWithLocation("E5006", node.Token.Line, node.Token.Column,
			"nil reference: cannot access member '%s' of nil", node.Member.Value)
	}

	if structObj, ok := obj.(*Struct); ok {
		if val, ok := structObj.Fields[node.Member.Value]; ok {
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

	return newErrorWithLocation("E5015", node.Token.Line, node.Token.Column,
		"member access not supported on type %s", obj.Type())
}

func nativeBoolToBooleanObject(input bool) *Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func isTruthy(obj Object) bool {
	switch obj {
	case NIL:
		return false
	case FALSE:
		return false
	default:
		return true
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

// newErrorWithLocation creates an error with line/column info
func newErrorWithLocation(code string, line, column int, format string, a ...interface{}) *Error {
	return &Error{
		Message: fmt.Sprintf(format, a...),
		Code:    code,
		Line:    line,
		Column:  column,
	}
}
