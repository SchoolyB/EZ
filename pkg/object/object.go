package object

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/marshallburns/ez/pkg/ast"
)

type ObjectType string

const (
	INTEGER_OBJ      ObjectType = "INTEGER"
	FLOAT_OBJ        ObjectType = "FLOAT"
	STRING_OBJ       ObjectType = "STRING"
	CHAR_OBJ         ObjectType = "CHAR"
	BYTE_OBJ         ObjectType = "BYTE"
	BOOLEAN_OBJ      ObjectType = "BOOLEAN"
	NIL_OBJ          ObjectType = "NIL"
	RETURN_VALUE_OBJ ObjectType = "RETURN_VALUE"
	ERROR_OBJ        ObjectType = "ERROR"
	FUNCTION_OBJ     ObjectType = "FUNCTION"
	BUILTIN_OBJ      ObjectType = "BUILTIN"
	ARRAY_OBJ        ObjectType = "ARRAY"
	MAP_OBJ          ObjectType = "MAP"
	STRUCT_OBJ       ObjectType = "STRUCT"
	BREAK_OBJ        ObjectType = "BREAK"
	CONTINUE_OBJ     ObjectType = "CONTINUE"
	ENUM_OBJ         ObjectType = "ENUM"
	ENUM_VALUE_OBJ   ObjectType = "ENUM_VALUE"
	MODULE_OBJ       ObjectType = "MODULE"
	FILE_HANDLE_OBJ  ObjectType = "FILE_HANDLE"
	REFERENCE_OBJ    ObjectType = "REFERENCE"
	RANGE_OBJ        ObjectType = "RANGE"
	TYPE_OBJ         ObjectType = "TYPE"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

// Integer wraps big.Int for arbitrary precision integer support
type Integer struct {
	Value        *big.Int
	DeclaredType string
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return i.Value.String() }

func (i *Integer) GetDeclaredType() string {
	if i.DeclaredType == "" {
		return "int"
	}
	return i.DeclaredType
}

// Float wraps float64
type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string {
	// Use %g format which shows the shortest representation that
	// accurately represents the value - important for debugging
	// floating point precision issues (#640)
	s := strconv.FormatFloat(f.Value, 'g', -1, 64)
	// Ensure whole numbers still show as floats (e.g., "1" -> "1.0")
	if !strings.Contains(s, ".") && !strings.Contains(s, "e") {
		s += ".0"
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

// Byte wraps uint8 (0-255)
type Byte struct {
	Value uint8
}

func (b *Byte) Type() ObjectType { return BYTE_OBJ }
func (b *Byte) Inspect() string  { return fmt.Sprintf("%d", b.Value) }

// Range represents a range of values for use in membership checks and iteration
type Range struct {
	Start *big.Int
	End   *big.Int
	Step  *big.Int
}

func (r *Range) Type() ObjectType { return RANGE_OBJ }
func (r *Range) Inspect() string {
	if r.Step.Cmp(big.NewInt(1)) == 0 {
		return fmt.Sprintf("range(%s, %s)", r.Start.String(), r.End.String())
	}
	return fmt.Sprintf("range(%s, %s, %s)", r.Start.String(), r.End.String(), r.Step.String())
}

// Contains checks if a value is within the range (respecting step)
func (r *Range) Contains(value *big.Int) bool {
	if r.Step.Sign() > 0 {
		// Positive step: start <= value < end
		if value.Cmp(r.Start) < 0 || value.Cmp(r.End) >= 0 {
			return false
		}
		// Check if value is on a step: (value - start) % step == 0
		diff := new(big.Int).Sub(value, r.Start)
		mod := new(big.Int).Mod(diff, r.Step)
		return mod.Sign() == 0
	} else {
		// Negative step: end < value <= start
		if value.Cmp(r.Start) > 0 || value.Cmp(r.End) <= 0 {
			return false
		}
		// Check if value is on a step: (start - value) % abs(step) == 0
		diff := new(big.Int).Sub(r.Start, value)
		absStep := new(big.Int).Abs(r.Step)
		mod := new(big.Int).Mod(diff, absStep)
		return mod.Sign() == 0
	}
}

// FileHandle wraps an open file handle for streaming I/O
type FileHandle struct {
	File     *os.File
	Path     string // Original path used to open the file
	Mode     int    // Open mode flags
	IsClosed bool   // Track if handle has been closed
}

func (fh *FileHandle) Type() ObjectType { return FILE_HANDLE_OBJ }
func (fh *FileHandle) Inspect() string {
	if fh.IsClosed {
		return fmt.Sprintf("<FileHandle(closed) %s>", fh.Path)
	}
	return fmt.Sprintf("<FileHandle %s>", fh.Path)
}

// Reference represents a reference to a variable in another environment
// Used for mutable (&) parameters to allow modifications to persist to the caller
type Reference struct {
	Env  *Environment // The environment where the original variable lives
	Name string       // The variable name in that environment
}

func (r *Reference) Type() ObjectType { return REFERENCE_OBJ }
func (r *Reference) Inspect() string  { return fmt.Sprintf("<ref %s>", r.Name) }

// Deref returns the current value of the referenced variable
// Recursively dereferences if the target is also a Reference (for nested mutable param forwarding)
func (r *Reference) Deref() (Object, bool) {
	val, ok := r.Env.Get(r.Name)
	if !ok {
		return nil, false
	}
	// Chase through nested references
	if ref, isRef := val.(*Reference); isRef {
		return ref.Deref()
	}
	return val, ok
}

// SetValue updates the referenced variable's value
// Traverses the scope chain to find and update the variable
func (r *Reference) SetValue(val Object) bool {
	return r.Env.updateRef(r.Name, val)
}

// updateRef updates a variable's value, traversing outer scopes if needed
// This is used by Reference.SetValue to update through references
// If the target is itself a Reference, chase through to the ultimate target
func (e *Environment) updateRef(name string, val Object) bool {
	if current, ok := e.store[name]; ok {
		// If current value is a Reference, chase through it
		if ref, isRef := current.(*Reference); isRef {
			return ref.SetValue(val)
		}
		e.store[name] = val
		return true
	}
	if e.outer != nil {
		return e.outer.updateRef(name, val)
	}
	return false
}

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
	Message      string
	Code         string
	Line         int
	Column       int
	File         string // Source file where error occurred
	Help         string
	PreFormatted bool // If true, Message is already formatted and shouldn't be wrapped
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string {
	if e.PreFormatted {
		return e.Message
	}
	return "ERROR: " + e.Message
}

// Function represents a user-defined function
type Function struct {
	Parameters  []*ast.Parameter
	ReturnTypes []string
	Body        *ast.BlockStatement
	Env         *Environment
	File        string // Source file where function was defined
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string  { return "function" }

// BuiltinFunction is the signature for built-in functions
type BuiltinFunction func(args ...Object) Object

// Builtin represents a built-in function
type Builtin struct {
	Fn         BuiltinFunction
	IsConstant bool // If true, this is a constant (zero-arg function that returns a value)
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

// Array represents an array
type Array struct {
	Elements    []Object
	Mutable     bool
	ElementType string // Type of array elements (e.g., "int", "Task")
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	elements := make([]string, len(a.Elements))
	for i, e := range a.Elements {
		elements[i] = e.Inspect()
	}
	return "{" + strings.Join(elements, ", ") + "}"
}

// MapPair represents a key-value pair in a map
type MapPair struct {
	Key   Object
	Value Object
}

// Map represents a map/dictionary with ordered key-value pairs
type Map struct {
	Pairs   []*MapPair     // Ordered pairs for iteration
	Index   map[string]int // Maps key hash to index in Pairs for O(1) lookup
	Mutable bool
}

func (m *Map) Type() ObjectType { return MAP_OBJ }
func (m *Map) Inspect() string {
	pairs := make([]string, len(m.Pairs))
	for i, pair := range m.Pairs {
		pairs[i] = fmt.Sprintf("%s: %s", pair.Key.Inspect(), pair.Value.Inspect())
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

// HashKey returns a string hash for a key object (for map lookup)
func HashKey(obj Object) (string, bool) {
	switch o := obj.(type) {
	case *String:
		return "s:" + o.Value, true
	case *Integer:
		return fmt.Sprintf("i:%d", o.Value), true
	case *Boolean:
		if o.Value {
			return "b:true", true
		}
		return "b:false", true
	case *Char:
		return fmt.Sprintf("c:%d", o.Value), true
	case *Byte:
		return fmt.Sprintf("y:%d", o.Value), true
	case *EnumValue:
		// Enum values are hashable based on their type and name
		// Float-based enums are not allowed as map keys (checked at compile time)
		if _, isFloat := o.Value.(*Float); isFloat {
			return "", false
		}
		return fmt.Sprintf("e:%s.%s", o.EnumType, o.Name), true
	default:
		return "", false
	}
}

// Get retrieves a value from the map by key
func (m *Map) Get(key Object) (Object, bool) {
	hash, ok := HashKey(key)
	if !ok {
		return nil, false
	}
	idx, exists := m.Index[hash]
	if !exists {
		return nil, false
	}
	return m.Pairs[idx].Value, true
}

// Set adds or updates a key-value pair in the map
func (m *Map) Set(key, value Object) bool {
	hash, ok := HashKey(key)
	if !ok {
		return false
	}
	if idx, exists := m.Index[hash]; exists {
		// Update existing
		m.Pairs[idx].Value = value
	} else {
		// Add new
		m.Index[hash] = len(m.Pairs)
		m.Pairs = append(m.Pairs, &MapPair{Key: key, Value: value})
	}
	return true
}

// Delete removes a key-value pair from the map
func (m *Map) Delete(key Object) bool {
	hash, ok := HashKey(key)
	if !ok {
		return false
	}
	idx, exists := m.Index[hash]
	if !exists {
		return false
	}
	// Remove from pairs slice
	m.Pairs = append(m.Pairs[:idx], m.Pairs[idx+1:]...)
	// Rebuild index
	delete(m.Index, hash)
	for i := idx; i < len(m.Pairs); i++ {
		h, _ := HashKey(m.Pairs[i].Key)
		m.Index[h] = i
	}
	return true
}

// NewMap creates a new empty map
func NewMap() *Map {
	return &Map{
		Pairs:   []*MapPair{},
		Index:   make(map[string]int),
		Mutable: true,
	}
}

type StructFieldTags interface {
	Inspect() string
}

type EmptyTag struct{}

func (et *EmptyTag) Inspect() string {
	return "Tag: ``\n"
}

type JSONTag struct {
	Name           string
	Ignore         bool
	OmitEmpty      bool
	EncodeAsString bool
}

func (jt *JSONTag) Inspect() string {
	if jt.Ignore {
		return "Tag: `json:\"-\"`"
	}
	tag := fmt.Sprintf("Tag: `json:\"%s", jt.Name)
	if jt.OmitEmpty {
		tag += ",omitempty"
	}
	if jt.EncodeAsString {
		tag += ",string"
	}
	tag += "\"`"
	return tag
}

// Struct represents a struct instance
type Struct struct {
	TypeName  string
	Fields    map[string]Object
	FieldTags map[string]StructFieldTags
	Mutable   bool
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
	Name      string
	Fields    map[string]string
	FieldTags map[string]StructFieldTags
}

// TypeValue represents a type as a first-class value (for passing types to functions)
type TypeValue struct {
	TypeName string
	Def      *StructDef // nil for primitive types
}

func (t *TypeValue) Type() ObjectType { return TYPE_OBJ }
func (t *TypeValue) Inspect() string  { return fmt.Sprintf("<type %s>", t.TypeName) }

// Enum represents an enum with computed values
type Enum struct {
	Name   string
	Values map[string]Object
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
	EnumType string
	Name     string
	Value    Object
}

func (ev *EnumValue) Type() ObjectType { return ENUM_VALUE_OBJ }
func (ev *EnumValue) Inspect() string {
	return ev.Value.Inspect()
}

// ModuleObject represents a loaded module at runtime
type ModuleObject struct {
	Name       string                // Module name
	Exports    map[string]Object     // Exported (public) symbols
	StructDefs map[string]*StructDef // Exported struct definitions
}

func (m *ModuleObject) Type() ObjectType { return MODULE_OBJ }
func (m *ModuleObject) Inspect() string {
	return fmt.Sprintf("<module %s>", m.Name)
}

// Get retrieves an exported symbol from the module
func (m *ModuleObject) Get(name string) (Object, bool) {
	obj, ok := m.Exports[name]
	return obj, ok
}

// GetStructDef retrieves an exported struct definition from the module
func (m *ModuleObject) GetStructDef(name string) (*StructDef, bool) {
	if m.StructDefs == nil {
		return nil, false
	}
	def, ok := m.StructDefs[name]
	return def, ok
}

// Singleton values
var (
	NIL   = &Nil{}
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
)

// Visibility represents the access level of a symbol
type Visibility int

const (
	VisibilityPublic        Visibility = iota // Public (default) - accessible from anywhere
	VisibilityPrivate                         // Private to this file only
	VisibilityPrivateModule                   // Private to this module (all files in directory)
)

// Environment holds variable bindings
type Environment struct {
	store      map[string]Object
	mutable    map[string]bool
	visibility map[string]Visibility // Visibility of each binding
	structDefs map[string]*StructDef
	outer      *Environment
	imports    map[string]string        // Legacy: alias -> stdlib module name
	modules    map[string]*ModuleObject // User modules: alias -> module object
	using      []string
	loopDepth  int
}

func NewEnvironment() *Environment {
	env := &Environment{
		store:      make(map[string]Object),
		mutable:    make(map[string]bool),
		visibility: make(map[string]Visibility),
		structDefs: make(map[string]*StructDef),
		outer:      nil,
		imports:    make(map[string]string),
		modules:    make(map[string]*ModuleObject),
		using:      []string{},
		loopDepth:  0,
	}

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
	if outer != nil {
		env.loopDepth = outer.loopDepth
	}
	return env
}

func (e *Environment) Import(alias, module string) {
	e.imports[alias] = module
}

func (e *Environment) GetImport(alias string) (string, bool) {
	if module, ok := e.imports[alias]; ok {
		return module, true
	}
	if e.outer != nil {
		return e.outer.GetImport(alias)
	}
	return "", false
}

// RegisterModule registers a user module with the given alias
func (e *Environment) RegisterModule(alias string, mod *ModuleObject) {
	e.modules[alias] = mod
}

// GetModule retrieves a registered user module by alias
func (e *Environment) GetModule(alias string) (*ModuleObject, bool) {
	if mod, ok := e.modules[alias]; ok {
		return mod, true
	}
	if e.outer != nil {
		return e.outer.GetModule(alias)
	}
	return nil, false
}

// HasModule checks if a module is registered (either stdlib import or user module)
func (e *Environment) HasModule(alias string) bool {
	// Check user modules
	if _, ok := e.modules[alias]; ok {
		return true
	}
	// Check stdlib imports
	if _, ok := e.imports[alias]; ok {
		return true
	}
	if e.outer != nil {
		return e.outer.HasModule(alias)
	}
	return false
}

func (e *Environment) Use(alias string) {
	e.using = append(e.using, alias)
}

func (e *Environment) GetUsing() []string {
	result := make([]string, len(e.using))
	copy(result, e.using)
	if e.outer != nil {
		result = append(result, e.outer.GetUsing()...)
	}

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
	e.visibility[name] = VisibilityPublic // Default to public
	return val
}

// SetWithVisibility sets a value with explicit visibility
func (e *Environment) SetWithVisibility(name string, val Object, isMutable bool, vis Visibility) Object {
	e.store[name] = val
	e.mutable[name] = isMutable
	e.visibility[name] = vis
	return val
}

// GetVisibility returns the visibility of a binding
func (e *Environment) GetVisibility(name string) (Visibility, bool) {
	if vis, ok := e.visibility[name]; ok {
		return vis, true
	}
	if e.outer != nil {
		return e.outer.GetVisibility(name)
	}
	return VisibilityPublic, false
}

func (e *Environment) Update(name string, val Object) (bool, bool) {
	if _, ok := e.store[name]; ok {
		if !e.mutable[name] {
			return true, false
		}
		e.store[name] = val
		return true, true
	}
	if e.outer != nil {
		return e.outer.Update(name, val)
	}
	return false, false
}

func (e *Environment) IsMutable(name string) (bool, bool) {
	if isMut, ok := e.mutable[name]; ok {
		return isMut, true
	}
	if e.outer != nil {
		return e.outer.IsMutable(name)
	}
	return false, false
}

func (e *Environment) EnterLoop() {
	e.loopDepth++
}

func (e *Environment) ExitLoop() {
	if e.loopDepth > 0 {
		e.loopDepth--
	}
}

func (e *Environment) InLoop() bool {
	return e.loopDepth > 0
}

func (e *Environment) RegisterStructDef(name string, def *StructDef) {
	e.structDefs[name] = def
}

// RegisterStructDefWithVisibility registers a struct definition with explicit visibility
func (e *Environment) RegisterStructDefWithVisibility(name string, def *StructDef, vis Visibility) {
	e.structDefs[name] = def
	e.visibility[name] = vis
}

func (e *Environment) GetStructDef(name string) (*StructDef, bool) {
	if def, ok := e.structDefs[name]; ok {
		return def, true
	}
	if e.outer != nil {
		return e.outer.GetStructDef(name)
	}
	return nil, false
}

// GetPublicBindings returns only public bindings (for module exports)
func (e *Environment) GetPublicBindings() map[string]Object {
	result := make(map[string]Object)
	for name, obj := range e.store {
		if vis, ok := e.visibility[name]; !ok || vis == VisibilityPublic {
			result[name] = obj
		}
	}
	return result
}

// GetPublicStructDefs returns all public struct definitions in this environment
func (e *Environment) GetPublicStructDefs() map[string]*StructDef {
	result := make(map[string]*StructDef)
	for name, def := range e.structDefs {
		// Skip the built-in Error struct
		if name == "Error" {
			continue
		}
		// Check visibility - if not explicitly set, treat as public
		if vis, ok := e.visibility[name]; !ok || vis == VisibilityPublic {
			result[name] = def
		}
	}
	return result
}
