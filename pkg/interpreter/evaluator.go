package interpreter

import (
	"fmt"

	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/errors"
)

var (
	NIL   = &Nil{}
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
)

func Eval(node ast.Node, env *Environment) Object {
	switch node := node.(type) {
	// Program
	case *ast.Program:
		return evalProgram(node, env)

	// Statements
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

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
		return &Break{}

	case *ast.ContinueStatement:
		return &Continue{}

	case *ast.FunctionDeclaration:
		return evalFunctionDeclaration(node, env)

	case *ast.ImportStatement:
		// Register the imported module with its alias
		// The alias is what's used in code (e.g., str.upper())
		// The module is the actual library (e.g., strings)
		alias := node.Alias
		if alias == "" {
			alias = node.Module
		}
		env.Import(alias, node.Module)
		return NIL

	case *ast.UsingStatement:
		// Bring the module's functions into scope
		alias := node.Module.Value
		// Verify the module was imported
		if _, ok := env.GetImport(alias); !ok {
			return newError("cannot use '%s': module not imported", alias)
		}
		env.Use(alias)
		return NIL

	// Expressions
	case *ast.IntegerValue:
		return &Integer{Value: node.Value}

	case *ast.FloatValue:
		return &Float{Value: node.Value}

	case *ast.StringValue:
		return &String{Value: node.Value}

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
		return &Array{Elements: elements}

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
		return evalInfixExpression(node.Operator, left, right)

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
		return evalIndexExpression(left, index)

	case *ast.MemberExpression:
		return evalMemberExpression(node, env)

	case *ast.RangeExpression:
		// Range is typically used in for loops, not standalone
		return newError("range() can only be used in for loops")
	}

	return newError("unknown node type: %T", node)
}

func evalProgram(program *ast.Program, env *Environment) Object {
	var result Object

	for _, stmt := range program.Statements {
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
			env.Set(name.Value, returnVal.Values[i])
		}
		return NIL
	}

	// Single variable assignment
	env.Set(node.Name.Value, val)
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
				return newError("identifier not found: %s", target.Value)
			}
			val = evalCompoundAssignment(node.Operator, oldVal, val)
			if isError(val) {
				return val
			}
		}

		if !env.Update(target.Value, val) {
			return newError("identifier not found: %s", target.Value)
		}

	case *ast.IndexExpression:
		// Array index assignment
		arr := Eval(target.Left, env)
		if isError(arr) {
			return arr
		}
		idx := Eval(target.Index, env)
		if isError(idx) {
			return idx
		}

		arrayObj, ok := arr.(*Array)
		if !ok {
			return newError("index operator not supported: %s", arr.Type())
		}
		index, ok := idx.(*Integer)
		if !ok {
			return newError("index must be integer, got %s", idx.Type())
		}
		if index.Value < 0 || index.Value >= int64(len(arrayObj.Elements)) {
			return newError("index out of bounds: %d", index.Value)
		}
		arrayObj.Elements[index.Value] = val

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
		structObj.Fields[target.Member.Value] = val
	}

	return NIL
}

func evalCompoundAssignment(op string, left, right Object) Object {
	switch op {
	case "+=":
		return evalInfixExpression("+", left, right)
	case "-=":
		return evalInfixExpression("-", left, right)
	case "*=":
		return evalInfixExpression("*", left, right)
	case "/=":
		return evalInfixExpression("/", left, right)
	case "%=":
		return evalInfixExpression("%", left, right)
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
		return Eval(node.Consequence, env)
	} else if node.Alternative != nil {
		return Eval(node.Alternative, env)
	}

	return NIL
}

func evalWhileStatement(node *ast.WhileStatement, env *Environment) Object {
	for {
		condition := Eval(node.Condition, env)
		if isError(condition) {
			return condition
		}

		if !isTruthy(condition) {
			break
		}

		result := Eval(node.Body, env)
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
	for {
		result := Eval(node.Body, env)
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
	// Get range bounds
	rangeExpr, ok := node.Iterable.(*ast.RangeExpression)
	if !ok {
		return newError("for loop requires range() expression")
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

	for i := start.Value; i <= end.Value; i++ { // Inclusive range
		loopEnv.Set(node.Variable.Value, &Integer{Value: i})

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
	collection := Eval(node.Collection, env)
	if isError(collection) {
		return collection
	}

	arr, ok := collection.(*Array)
	if !ok {
		return newError("for_each requires array, got %s", collection.Type())
	}

	loopEnv := NewEnclosedEnvironment(env)

	for _, elem := range arr.Elements {
		loopEnv.Set(node.Variable.Value, elem)

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

func evalFunctionDeclaration(node *ast.FunctionDeclaration, env *Environment) Object {
	fn := &Function{
		Parameters: node.Parameters,
		Body:       node.Body,
		Env:        env,
	}
	env.Set(node.Name.Value, fn)
	return NIL
}

func evalIdentifier(node *ast.Label, env *Environment) Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}

	// Check if the function is available via "using" modules
	for _, alias := range env.GetUsing() {
		if module, ok := env.GetImport(alias); ok {
			fullName := module + "." + node.Value
			if builtin, ok := builtins[fullName]; ok {
				return builtin
			}
		}
	}

	// Check global builtins (like len, typeof, etc.)
	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	// Create error with potential suggestion
	err := newErrorWithLocation("E3001", node.Token.Line, node.Token.Column,
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

func evalInfixExpression(operator string, left, right Object) Object {
	switch {
	case left.Type() == INTEGER_OBJ && right.Type() == INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case left.Type() == FLOAT_OBJ || right.Type() == FLOAT_OBJ:
		return evalFloatInfixExpression(operator, left, right)
	case left.Type() == STRING_OBJ && right.Type() == STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
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
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIntegerInfixExpression(operator string, left, right Object) Object {
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
			return newError("division by zero")
		}
		return &Integer{Value: leftVal / rightVal}
	case "%":
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
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalFloatInfixExpression(operator string, left, right Object) Object {
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
			return newError("division by zero")
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
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalStringInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*String).Value
	rightVal := right.(*String).Value

	switch operator {
	case "+":
		return &String{Value: leftVal + rightVal}
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
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
			if errObj.Code == "E3001" {
				// Change to "undefined function" error
				if label, ok := node.Function.(*ast.Label); ok {
					return newErrorWithLocation("E3002", label.Token.Line, label.Token.Column,
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

	return applyFunction(function, args)
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
		return builtin.Fn(evalArgs...)
	}

	return newError("function not found: %s", fullName)
}

func applyFunction(fn Object, args []Object) Object {
	switch fn := fn.(type) {
	case *Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)

	case *Builtin:
		return fn.Fn(args...)

	default:
		return newError("not a function: %s", fn.Type())
	}
}

func extendFunctionEnv(fn *Function, args []Object) *Environment {
	env := NewEnclosedEnvironment(fn.Env)

	for i, param := range fn.Parameters {
		if i < len(args) {
			env.Set(param.Name.Value, args[i])
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
			return newError("'%s' not found in module '%s'", node.Member.Value, alias)
		}
	}

	obj := Eval(node.Object, env)
	if isError(obj) {
		return obj
	}

	if structObj, ok := obj.(*Struct); ok {
		if val, ok := structObj.Fields[node.Member.Value]; ok {
			return val
		}
		return newError("field not found: %s", node.Member.Value)
	}

	return newError("member access not supported: %s", obj.Type())
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
