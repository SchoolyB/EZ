package interpreter

import (
	"fmt"
	"strings"

	"github.com/marshallburns/ez/pkg/ast"
)

type ObjectType string

const (
	INTEGER_OBJ      ObjectType = "INTEGER"
	FLOAT_OBJ        ObjectType = "FLOAT"
	STRING_OBJ       ObjectType = "STRING"
	CHAR_OBJ         ObjectType = "CHAR"
	BOOLEAN_OBJ      ObjectType = "BOOLEAN"
	NIL_OBJ          ObjectType = "NIL"
	RETURN_VALUE_OBJ ObjectType = "RETURN_VALUE"
	ERROR_OBJ        ObjectType = "ERROR"
	FUNCTION_OBJ     ObjectType = "FUNCTION"
	BUILTIN_OBJ      ObjectType = "BUILTIN"
	ARRAY_OBJ        ObjectType = "ARRAY"
	STRUCT_OBJ       ObjectType = "STRUCT"
	BREAK_OBJ        ObjectType = "BREAK"
	CONTINUE_OBJ     ObjectType = "CONTINUE"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

// Integer wraps int64
type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

// Float wraps float64
type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string  { return fmt.Sprintf("%g", f.Value) }

// String wraps string
type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

// Char wraps rune
type Char struct {
	Value rune
}

func (c *Char) Type() ObjectType { return CHAR_OBJ }
func (c *Char) Inspect() string  { return string(c.Value) }

// Boolean wraps bool
type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

// Nil represents nil
type Nil struct{}

func (n *Nil) Type() ObjectType { return NIL_OBJ }
func (n *Nil) Inspect() string  { return "nil" }

// ReturnValue wraps a return value
type ReturnValue struct {
	Values []Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string {
	vals := make([]string, len(rv.Values))
	for i, v := range rv.Values {
		vals[i] = v.Inspect()
	}
	return strings.Join(vals, ", ")
}

// Error represents an error
type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

// Function represents a user-defined function
type Function struct {
	Parameters []*ast.Parameter
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string  { return "function" }

// BuiltinFunction is the signature for built-in functions
type BuiltinFunction func(args ...Object) Object

// Builtin represents a built-in function
type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

// Array represents an array
type Array struct {
	Elements []Object
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	elements := make([]string, len(a.Elements))
	for i, e := range a.Elements {
		elements[i] = e.Inspect()
	}
	return "{" + strings.Join(elements, ", ") + "}"
}

// Struct represents a struct instance
type Struct struct {
	Fields map[string]Object
}

func (s *Struct) Type() ObjectType { return STRUCT_OBJ }
func (s *Struct) Inspect() string {
	pairs := make([]string, 0, len(s.Fields))
	for k, v := range s.Fields {
		pairs = append(pairs, fmt.Sprintf("%s: %s", k, v.Inspect()))
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

// Break represents a break statement
type Break struct{}

func (b *Break) Type() ObjectType { return BREAK_OBJ }
func (b *Break) Inspect() string  { return "break" }

// Continue represents a continue statement
type Continue struct{}

func (c *Continue) Type() ObjectType { return CONTINUE_OBJ }
func (c *Continue) Inspect() string  { return "continue" }

// Environment holds variable bindings
type Environment struct {
	store   map[string]Object
	outer   *Environment
	imports map[string]string // alias -> module mapping
	using   []string          // modules brought into scope (by alias)
}

func NewEnvironment() *Environment {
	return &Environment{
		store:   make(map[string]Object),
		outer:   nil,
		imports: make(map[string]string),
		using:   []string{},
	}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// Import registers a module with an alias
func (e *Environment) Import(alias, module string) {
	e.imports[alias] = module
}

// GetImport returns the module name for an alias (checks parent scopes too)
func (e *Environment) GetImport(alias string) (string, bool) {
	if module, ok := e.imports[alias]; ok {
		return module, true
	}
	if e.outer != nil {
		return e.outer.GetImport(alias)
	}
	return "", false
}

// Use brings a module's functions into scope (by alias)
func (e *Environment) Use(alias string) {
	e.using = append(e.using, alias)
}

// GetUsing returns all modules in scope (checks parent scopes too)
func (e *Environment) GetUsing() []string {
	result := make([]string, len(e.using))
	copy(result, e.using)
	if e.outer != nil {
		result = append(result, e.outer.GetUsing()...)
	}
	return result
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

// Update updates an existing variable (for assignments)
func (e *Environment) Update(name string, val Object) bool {
	if _, ok := e.store[name]; ok {
		e.store[name] = val
		return true
	}
	if e.outer != nil {
		return e.outer.Update(name, val)
	}
	return false
}
