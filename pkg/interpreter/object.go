package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

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
	ENUM_OBJ         ObjectType = "ENUM"
	ENUM_VALUE_OBJ   ObjectType = "ENUM_VALUE"
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
func (f *Float) Inspect() string {
	// Format float, ensuring we always show at least .0 for whole numbers
	s := fmt.Sprintf("%.10f", f.Value)
	// Trim trailing zeros, but keep at least one digit after decimal point
	s = strings.TrimRight(s, "0")
	if strings.HasSuffix(s, ".") {
		s += "0"
	}
	return s
}

// String wraps string
type String struct {
	Value   string
	Mutable bool
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return fmt.Sprintf("%q", s.Value) }

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
	// Extended error info for formatted output
	Code   string // Error code like "E3001"
	Line   int
	Column int
	Help   string // Optional help suggestion
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

// Function represents a user-defined function
type Function struct {
	Parameters  []*ast.Parameter
	ReturnTypes []string // declared return types for validation
	Body        *ast.BlockStatement
	Env         *Environment
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
	Mutable  bool
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
	TypeName string // e.g., "Person"
	Fields   map[string]Object
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

// StructDef holds the definition of a struct type
type StructDef struct {
	Name   string
	Fields map[string]string // field name -> type name
}

// Enum represents an enum with computed values
type Enum struct {
	Name   string
	Values map[string]Object // value name -> actual value (Integer, Float, or String)
}

func (e *Enum) Type() ObjectType { return ENUM_OBJ }
func (e *Enum) Inspect() string {
	var out strings.Builder
	out.WriteString("enum ")
	out.WriteString(e.Name)
	out.WriteString(" { ")
	first := true
	for name, val := range e.Values {
		if !first {
			out.WriteString(", ")
		}
		first = false
		out.WriteString(name)
		out.WriteString(" = ")
		out.WriteString(val.Inspect())
	}
	out.WriteString(" }")
	return out.String()
}

// EnumValue wraps an enum member value with its type information
type EnumValue struct {
	EnumType string // The enum type name (e.g., "COLOR")
	Name     string // The value name (e.g., "RED")
	Value    Object // The underlying value (Integer, Float, or String)
}

func (ev *EnumValue) Type() ObjectType { return ENUM_VALUE_OBJ }
func (ev *EnumValue) Inspect() string {
	return ev.Value.Inspect()
}

// Environment holds variable bindings
type Environment struct {
	store      map[string]Object
	mutable    map[string]bool       // tracks if variable is mutable (temp) or immutable (const)
	structDefs map[string]*StructDef // struct type definitions
	outer      *Environment
	imports    map[string]string // alias -> module mapping
	using      []string          // modules brought into scope (by alias)
	loopDepth  int               // tracks nested loop depth for break/continue validation
}

func NewEnvironment() *Environment {
	env := &Environment{
		store:      make(map[string]Object),
		mutable:    make(map[string]bool),
		structDefs: make(map[string]*StructDef),
		outer:      nil,
		imports:    make(map[string]string),
		using:      []string{},
		loopDepth:  0,
	}

	// Register built-in Error type
	env.structDefs["Error"] = &StructDef{
		Name: "Error",
		Fields: map[string]string{
			"message": "string",
			"code":    "int",
		},
	}

	return env
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	// Inherit loop depth from parent
	if outer != nil {
		env.loopDepth = outer.loopDepth
	}
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
// Deduplicates modules to avoid ambiguity when same module is used in multiple scopes
func (e *Environment) GetUsing() []string {
	result := make([]string, len(e.using))
	copy(result, e.using)
	if e.outer != nil {
		result = append(result, e.outer.GetUsing()...)
	}

	// Deduplicate the modules
	seen := make(map[string]bool)
	deduped := make([]string, 0, len(result))
	for _, module := range result {
		if !seen[module] {
			seen[module] = true
			deduped = append(deduped, module)
		}
	}

	return deduped
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object, isMutable bool) Object {
	e.store[name] = val
	e.mutable[name] = isMutable
	return val
}

// Update updates an existing variable (for assignments)
// Returns: (success bool, isMutable bool)
func (e *Environment) Update(name string, val Object) (bool, bool) {
	if _, ok := e.store[name]; ok {
		// Check if variable is mutable
		if !e.mutable[name] {
			return true, false // found but immutable
		}
		e.store[name] = val
		return true, true // found and updated
	}
	if e.outer != nil {
		return e.outer.Update(name, val)
	}
	return false, false // not found
}

// IsMutable checks if a variable is mutable
func (e *Environment) IsMutable(name string) (bool, bool) {
	if isMut, ok := e.mutable[name]; ok {
		return isMut, true
	}
	if e.outer != nil {
		return e.outer.IsMutable(name)
	}
	return false, false
}

// EnterLoop increments the loop depth
func (e *Environment) EnterLoop() {
	e.loopDepth++
}

// ExitLoop decrements the loop depth
func (e *Environment) ExitLoop() {
	if e.loopDepth > 0 {
		e.loopDepth--
	}
}

// InLoop returns true if currently inside a loop
func (e *Environment) InLoop() bool {
	return e.loopDepth > 0
}

// RegisterStructDef registers a struct type definition
func (e *Environment) RegisterStructDef(name string, def *StructDef) {
	e.structDefs[name] = def
}

// GetStructDef retrieves a struct type definition (checks parent scopes too)
func (e *Environment) GetStructDef(name string) (*StructDef, bool) {
	if def, ok := e.structDefs[name]; ok {
		return def, true
	}
	if e.outer != nil {
		return e.outer.GetStructDef(name)
	}
	return nil, false
}
