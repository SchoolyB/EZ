package object

import (
	"strings"
	"testing"
)

// ============================================================================
// Object Type Constants
// ============================================================================

func TestObjectTypeConstants(t *testing.T) {
	tests := []struct {
		objType  ObjectType
		expected string
	}{
		{INTEGER_OBJ, "INTEGER"},
		{FLOAT_OBJ, "FLOAT"},
		{STRING_OBJ, "STRING"},
		{CHAR_OBJ, "CHAR"},
		{BOOLEAN_OBJ, "BOOLEAN"},
		{NIL_OBJ, "NIL"},
		{RETURN_VALUE_OBJ, "RETURN_VALUE"},
		{ERROR_OBJ, "ERROR"},
		{FUNCTION_OBJ, "FUNCTION"},
		{BUILTIN_OBJ, "BUILTIN"},
		{ARRAY_OBJ, "ARRAY"},
		{MAP_OBJ, "MAP"},
		{STRUCT_OBJ, "STRUCT"},
		{BREAK_OBJ, "BREAK"},
		{CONTINUE_OBJ, "CONTINUE"},
		{ENUM_OBJ, "ENUM"},
		{ENUM_VALUE_OBJ, "ENUM_VALUE"},
		{MODULE_OBJ, "MODULE"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.objType) != tt.expected {
				t.Errorf("ObjectType = %q, want %q", tt.objType, tt.expected)
			}
		})
	}
}

// ============================================================================
// Object Interface Tests
// ============================================================================

func TestIntegerObject(t *testing.T) {
	i := &Integer{Value: 42}
	if i.Type() != INTEGER_OBJ {
		t.Errorf("Type() = %s, want %s", i.Type(), INTEGER_OBJ)
	}
	if i.Inspect() != "42" {
		t.Errorf("Inspect() = %q, want %q", i.Inspect(), "42")
	}
}

func TestIntegerDeclaredType(t *testing.T) {
	t.Run("default type", func(t *testing.T) {
		i := &Integer{Value: 42}
		if i.GetDeclaredType() != "int" {
			t.Errorf("GetDeclaredType() = %q, want %q", i.GetDeclaredType(), "int")
		}
	})

	t.Run("explicit type", func(t *testing.T) {
		i := &Integer{Value: 42, DeclaredType: "i32"}
		if i.GetDeclaredType() != "i32" {
			t.Errorf("GetDeclaredType() = %q, want %q", i.GetDeclaredType(), "i32")
		}
	})
}

func TestFloatObject(t *testing.T) {
	tests := []struct {
		value    float64
		expected string
	}{
		{3.14, "3.14"},
		{1.0, "1.0"},
		{0.5, "0.5"},
		{100.0, "100.0"},
	}

	for _, tt := range tests {
		f := &Float{Value: tt.value}
		if f.Type() != FLOAT_OBJ {
			t.Errorf("Type() = %s, want %s", f.Type(), FLOAT_OBJ)
		}
		if f.Inspect() != tt.expected {
			t.Errorf("Inspect() = %q, want %q", f.Inspect(), tt.expected)
		}
	}
}

func TestStringObject(t *testing.T) {
	s := &String{Value: "hello", Mutable: true}
	if s.Type() != STRING_OBJ {
		t.Errorf("Type() = %s, want %s", s.Type(), STRING_OBJ)
	}
	if s.Inspect() != `"hello"` {
		t.Errorf("Inspect() = %q, want %q", s.Inspect(), `"hello"`)
	}
}

func TestCharObject(t *testing.T) {
	c := &Char{Value: 'A'}
	if c.Type() != CHAR_OBJ {
		t.Errorf("Type() = %s, want %s", c.Type(), CHAR_OBJ)
	}
	if c.Inspect() != "A" {
		t.Errorf("Inspect() = %q, want %q", c.Inspect(), "A")
	}
}

func TestBooleanObject(t *testing.T) {
	tests := []struct {
		value    bool
		expected string
	}{
		{true, "true"},
		{false, "false"},
	}

	for _, tt := range tests {
		b := &Boolean{Value: tt.value}
		if b.Type() != BOOLEAN_OBJ {
			t.Errorf("Type() = %s, want %s", b.Type(), BOOLEAN_OBJ)
		}
		if b.Inspect() != tt.expected {
			t.Errorf("Inspect() = %q, want %q", b.Inspect(), tt.expected)
		}
	}
}

func TestNilObject(t *testing.T) {
	n := &Nil{}
	if n.Type() != NIL_OBJ {
		t.Errorf("Type() = %s, want %s", n.Type(), NIL_OBJ)
	}
	if n.Inspect() != "nil" {
		t.Errorf("Inspect() = %q, want %q", n.Inspect(), "nil")
	}
}

func TestReturnValueObject(t *testing.T) {
	rv := &ReturnValue{
		Values: []Object{
			&Integer{Value: 1},
			&Integer{Value: 2},
		},
	}
	if rv.Type() != RETURN_VALUE_OBJ {
		t.Errorf("Type() = %s, want %s", rv.Type(), RETURN_VALUE_OBJ)
	}
	if rv.Inspect() != "1, 2" {
		t.Errorf("Inspect() = %q, want %q", rv.Inspect(), "1, 2")
	}
}

func TestErrorObject(t *testing.T) {
	t.Run("normal error", func(t *testing.T) {
		e := &Error{Message: "something went wrong", Code: "E5001"}
		if e.Type() != ERROR_OBJ {
			t.Errorf("Type() = %s, want %s", e.Type(), ERROR_OBJ)
		}
		if e.Inspect() != "ERROR: something went wrong" {
			t.Errorf("Inspect() = %q, want %q", e.Inspect(), "ERROR: something went wrong")
		}
	})

	t.Run("preformatted error", func(t *testing.T) {
		e := &Error{Message: "formatted message", PreFormatted: true}
		if e.Inspect() != "formatted message" {
			t.Errorf("Inspect() = %q, want %q", e.Inspect(), "formatted message")
		}
	})
}

func TestFunctionObject(t *testing.T) {
	f := &Function{}
	if f.Type() != FUNCTION_OBJ {
		t.Errorf("Type() = %s, want %s", f.Type(), FUNCTION_OBJ)
	}
	if f.Inspect() != "function" {
		t.Errorf("Inspect() = %q, want %q", f.Inspect(), "function")
	}
}

func TestBuiltinObject(t *testing.T) {
	b := &Builtin{Fn: func(args ...Object) Object { return NIL }}
	if b.Type() != BUILTIN_OBJ {
		t.Errorf("Type() = %s, want %s", b.Type(), BUILTIN_OBJ)
	}
	if b.Inspect() != "builtin function" {
		t.Errorf("Inspect() = %q, want %q", b.Inspect(), "builtin function")
	}
}

func TestArrayObject(t *testing.T) {
	a := &Array{
		Elements: []Object{
			&Integer{Value: 1},
			&Integer{Value: 2},
			&Integer{Value: 3},
		},
		Mutable: true,
	}
	if a.Type() != ARRAY_OBJ {
		t.Errorf("Type() = %s, want %s", a.Type(), ARRAY_OBJ)
	}
	if a.Inspect() != "{1, 2, 3}" {
		t.Errorf("Inspect() = %q, want %q", a.Inspect(), "{1, 2, 3}")
	}
}

func TestBreakObject(t *testing.T) {
	b := &Break{}
	if b.Type() != BREAK_OBJ {
		t.Errorf("Type() = %s, want %s", b.Type(), BREAK_OBJ)
	}
	if b.Inspect() != "break" {
		t.Errorf("Inspect() = %q, want %q", b.Inspect(), "break")
	}
}

func TestContinueObject(t *testing.T) {
	c := &Continue{}
	if c.Type() != CONTINUE_OBJ {
		t.Errorf("Type() = %s, want %s", c.Type(), CONTINUE_OBJ)
	}
	if c.Inspect() != "continue" {
		t.Errorf("Inspect() = %q, want %q", c.Inspect(), "continue")
	}
}

func TestEnumObject(t *testing.T) {
	e := &Enum{
		Name: "Status",
		Values: map[string]Object{
			"TODO": &Integer{Value: 0},
			"DONE": &Integer{Value: 1},
		},
	}
	if e.Type() != ENUM_OBJ {
		t.Errorf("Type() = %s, want %s", e.Type(), ENUM_OBJ)
	}
	// Inspect output order may vary due to map iteration
	if !strings.Contains(e.Inspect(), "enum Status") {
		t.Errorf("Inspect() should contain 'enum Status', got %q", e.Inspect())
	}
}

func TestEnumValueObject(t *testing.T) {
	ev := &EnumValue{
		EnumType: "Status",
		Name:     "TODO",
		Value:    &Integer{Value: 0},
	}
	if ev.Type() != ENUM_VALUE_OBJ {
		t.Errorf("Type() = %s, want %s", ev.Type(), ENUM_VALUE_OBJ)
	}
	if ev.Inspect() != "0" {
		t.Errorf("Inspect() = %q, want %q", ev.Inspect(), "0")
	}
}

func TestModuleObject(t *testing.T) {
	m := &ModuleObject{
		Name: "mymodule",
		Exports: map[string]Object{
			"helper": &Function{},
		},
	}
	if m.Type() != MODULE_OBJ {
		t.Errorf("Type() = %s, want %s", m.Type(), MODULE_OBJ)
	}
	if m.Inspect() != "<module mymodule>" {
		t.Errorf("Inspect() = %q, want %q", m.Inspect(), "<module mymodule>")
	}

	// Test Get
	obj, ok := m.Get("helper")
	if !ok {
		t.Error("Get(helper) should return true")
	}
	if obj == nil {
		t.Error("Get(helper) should return non-nil object")
	}

	_, ok = m.Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) should return false")
	}
}

func TestStructObject(t *testing.T) {
	s := &Struct{
		TypeName: "Person",
		Fields: map[string]Object{
			"name": &String{Value: "Alice"},
			"age":  &Integer{Value: 30},
		},
	}
	if s.Type() != STRUCT_OBJ {
		t.Errorf("Type() = %s, want %s", s.Type(), STRUCT_OBJ)
	}
	// Contains check since map order varies
	if !strings.Contains(s.Inspect(), "name:") && !strings.Contains(s.Inspect(), "age:") {
		t.Errorf("Inspect() should contain field info, got %q", s.Inspect())
	}
}

// ============================================================================
// Singleton Tests
// ============================================================================

func TestSingletonObjects(t *testing.T) {
	if NIL == nil {
		t.Error("NIL singleton should not be nil")
	}
	if TRUE == nil || !TRUE.Value {
		t.Error("TRUE singleton should be non-nil with Value=true")
	}
	if FALSE == nil || FALSE.Value {
		t.Error("FALSE singleton should be non-nil with Value=false")
	}
}

// ============================================================================
// Map Tests
// ============================================================================

func TestNewMap(t *testing.T) {
	m := NewMap()
	if m == nil {
		t.Fatal("NewMap() returned nil")
	}
	if len(m.Pairs) != 0 {
		t.Errorf("Pairs length = %d, want 0", len(m.Pairs))
	}
	if !m.Mutable {
		t.Error("Mutable should be true")
	}
}

func TestMapSetAndGet(t *testing.T) {
	m := NewMap()

	// Set values
	m.Set(&String{Value: "key1"}, &Integer{Value: 1})
	m.Set(&String{Value: "key2"}, &Integer{Value: 2})

	// Get values
	val, ok := m.Get(&String{Value: "key1"})
	if !ok {
		t.Error("Get(key1) should return true")
	}
	if val.(*Integer).Value != 1 {
		t.Errorf("Get(key1) = %d, want 1", val.(*Integer).Value)
	}

	// Update existing key
	m.Set(&String{Value: "key1"}, &Integer{Value: 100})
	val, _ = m.Get(&String{Value: "key1"})
	if val.(*Integer).Value != 100 {
		t.Errorf("Updated key1 = %d, want 100", val.(*Integer).Value)
	}

	// Get non-existent key
	_, ok = m.Get(&String{Value: "nonexistent"})
	if ok {
		t.Error("Get(nonexistent) should return false")
	}
}

func TestMapDelete(t *testing.T) {
	m := NewMap()
	m.Set(&String{Value: "a"}, &Integer{Value: 1})
	m.Set(&String{Value: "b"}, &Integer{Value: 2})
	m.Set(&String{Value: "c"}, &Integer{Value: 3})

	// Delete middle key
	ok := m.Delete(&String{Value: "b"})
	if !ok {
		t.Error("Delete(b) should return true")
	}

	// Verify deleted
	_, ok = m.Get(&String{Value: "b"})
	if ok {
		t.Error("Get(b) should return false after delete")
	}

	// Verify others still exist
	_, ok = m.Get(&String{Value: "a"})
	if !ok {
		t.Error("Get(a) should still return true")
	}
	_, ok = m.Get(&String{Value: "c"})
	if !ok {
		t.Error("Get(c) should still return true")
	}

	// Delete non-existent
	ok = m.Delete(&String{Value: "nonexistent"})
	if ok {
		t.Error("Delete(nonexistent) should return false")
	}
}

func TestMapInspect(t *testing.T) {
	m := NewMap()
	m.Set(&String{Value: "name"}, &String{Value: "Alice"})

	output := m.Inspect()
	if !strings.Contains(output, "name") {
		t.Errorf("Inspect() should contain 'name', got %q", output)
	}
}

func TestHashKey(t *testing.T) {
	tests := []struct {
		obj      Object
		expected string
		ok       bool
	}{
		{&String{Value: "hello"}, "s:hello", true},
		{&Integer{Value: 42}, "i:42", true},
		{&Boolean{Value: true}, "b:true", true},
		{&Boolean{Value: false}, "b:false", true},
		{&Char{Value: 'A'}, "c:65", true},
		{&Array{}, "", false}, // Arrays are not hashable
		{&Nil{}, "", false},   // Nil is not hashable
	}

	for _, tt := range tests {
		hash, ok := HashKey(tt.obj)
		if ok != tt.ok {
			t.Errorf("HashKey(%T) ok = %v, want %v", tt.obj, ok, tt.ok)
		}
		if ok && hash != tt.expected {
			t.Errorf("HashKey(%T) = %q, want %q", tt.obj, hash, tt.expected)
		}
	}
}

// ============================================================================
// Environment Tests
// ============================================================================

func TestNewEnvironment(t *testing.T) {
	env := NewEnvironment()
	if env == nil {
		t.Fatal("NewEnvironment() returned nil")
	}
	if env.outer != nil {
		t.Error("outer should be nil for new environment")
	}
	if env.loopDepth != 0 {
		t.Errorf("loopDepth = %d, want 0", env.loopDepth)
	}

	// Should have built-in Error struct
	def, ok := env.GetStructDef("Error")
	if !ok {
		t.Error("Error struct should be defined")
	}
	if def == nil {
		t.Error("Error struct def should not be nil")
	}
}

func TestNewEnclosedEnvironment(t *testing.T) {
	outer := NewEnvironment()
	outer.loopDepth = 2
	inner := NewEnclosedEnvironment(outer)

	if inner.outer != outer {
		t.Error("inner.outer should be outer")
	}
	if inner.loopDepth != 2 {
		t.Errorf("loopDepth = %d, want 2", inner.loopDepth)
	}
}

func TestEnvironmentGetSet(t *testing.T) {
	env := NewEnvironment()

	// Set value
	env.Set("x", &Integer{Value: 42}, true)

	// Get value
	val, ok := env.Get("x")
	if !ok {
		t.Error("Get(x) should return true")
	}
	if val.(*Integer).Value != 42 {
		t.Errorf("Get(x) = %d, want 42", val.(*Integer).Value)
	}

	// Get non-existent
	_, ok = env.Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) should return false")
	}
}

func TestEnvironmentScopeChain(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("x", &Integer{Value: 1}, true)

	inner := NewEnclosedEnvironment(outer)
	inner.Set("y", &Integer{Value: 2}, true)

	// Inner can see outer's variables
	val, ok := inner.Get("x")
	if !ok {
		t.Error("inner.Get(x) should return true")
	}
	if val.(*Integer).Value != 1 {
		t.Errorf("inner.Get(x) = %d, want 1", val.(*Integer).Value)
	}

	// Outer cannot see inner's variables
	_, ok = outer.Get("y")
	if ok {
		t.Error("outer.Get(y) should return false")
	}
}

func TestEnvironmentUpdate(t *testing.T) {
	env := NewEnvironment()
	env.Set("x", &Integer{Value: 1}, true)  // mutable
	env.Set("y", &Integer{Value: 2}, false) // immutable

	// Update mutable
	found, updated := env.Update("x", &Integer{Value: 10})
	if !found || !updated {
		t.Error("Update(x) should succeed")
	}
	val, _ := env.Get("x")
	if val.(*Integer).Value != 10 {
		t.Errorf("After update, x = %d, want 10", val.(*Integer).Value)
	}

	// Update immutable
	found, updated = env.Update("y", &Integer{Value: 20})
	if !found {
		t.Error("Update(y) should find the variable")
	}
	if updated {
		t.Error("Update(y) should fail (immutable)")
	}

	// Update non-existent
	found, _ = env.Update("z", &Integer{Value: 30})
	if found {
		t.Error("Update(z) should not find variable")
	}
}

func TestEnvironmentIsMutable(t *testing.T) {
	env := NewEnvironment()
	env.Set("mutable", &Integer{Value: 1}, true)
	env.Set("immutable", &Integer{Value: 2}, false)

	isMut, ok := env.IsMutable("mutable")
	if !ok || !isMut {
		t.Error("mutable should be mutable")
	}

	isMut, ok = env.IsMutable("immutable")
	if !ok || isMut {
		t.Error("immutable should not be mutable")
	}

	_, ok = env.IsMutable("nonexistent")
	if ok {
		t.Error("nonexistent should not be found")
	}
}

func TestEnvironmentLoopDepth(t *testing.T) {
	env := NewEnvironment()

	if env.InLoop() {
		t.Error("should not be in loop initially")
	}

	env.EnterLoop()
	if !env.InLoop() {
		t.Error("should be in loop after EnterLoop")
	}

	env.EnterLoop()
	if env.loopDepth != 2 {
		t.Errorf("loopDepth = %d, want 2", env.loopDepth)
	}

	env.ExitLoop()
	if env.loopDepth != 1 {
		t.Errorf("loopDepth = %d, want 1", env.loopDepth)
	}

	env.ExitLoop()
	if env.InLoop() {
		t.Error("should not be in loop after all exits")
	}

	// ExitLoop when not in loop should not go negative
	env.ExitLoop()
	if env.loopDepth != 0 {
		t.Errorf("loopDepth = %d, want 0", env.loopDepth)
	}
}

func TestEnvironmentImports(t *testing.T) {
	env := NewEnvironment()
	env.Import("arr", "arrays")

	module, ok := env.GetImport("arr")
	if !ok {
		t.Error("GetImport(arr) should return true")
	}
	if module != "arrays" {
		t.Errorf("GetImport(arr) = %q, want %q", module, "arrays")
	}

	_, ok = env.GetImport("nonexistent")
	if ok {
		t.Error("GetImport(nonexistent) should return false")
	}
}

func TestEnvironmentModules(t *testing.T) {
	env := NewEnvironment()
	mod := &ModuleObject{Name: "utils"}
	env.RegisterModule("utils", mod)

	m, ok := env.GetModule("utils")
	if !ok {
		t.Error("GetModule(utils) should return true")
	}
	if m != mod {
		t.Error("GetModule should return registered module")
	}

	if !env.HasModule("utils") {
		t.Error("HasModule(utils) should return true")
	}

	if env.HasModule("nonexistent") {
		t.Error("HasModule(nonexistent) should return false")
	}
}

func TestEnvironmentUsing(t *testing.T) {
	env := NewEnvironment()
	env.Use("std")
	env.Use("arrays")

	using := env.GetUsing()
	if len(using) != 2 {
		t.Errorf("GetUsing() length = %d, want 2", len(using))
	}
}

func TestEnvironmentStructDefs(t *testing.T) {
	env := NewEnvironment()
	def := &StructDef{
		Name:   "Point",
		Fields: map[string]string{"x": "int", "y": "int"},
	}
	env.RegisterStructDef("Point", def)

	d, ok := env.GetStructDef("Point")
	if !ok {
		t.Error("GetStructDef(Point) should return true")
	}
	if d != def {
		t.Error("GetStructDef should return registered def")
	}
}

func TestEnvironmentVisibility(t *testing.T) {
	env := NewEnvironment()
	env.SetWithVisibility("public", &Integer{Value: 1}, true, VisibilityPublic)
	env.SetWithVisibility("private", &Integer{Value: 2}, true, VisibilityPrivate)

	vis, ok := env.GetVisibility("public")
	if !ok || vis != VisibilityPublic {
		t.Error("public should have VisibilityPublic")
	}

	vis, ok = env.GetVisibility("private")
	if !ok || vis != VisibilityPrivate {
		t.Error("private should have VisibilityPrivate")
	}
}

func TestEnvironmentGetPublicBindings(t *testing.T) {
	env := NewEnvironment()
	env.SetWithVisibility("public1", &Integer{Value: 1}, true, VisibilityPublic)
	env.SetWithVisibility("public2", &Integer{Value: 2}, true, VisibilityPublic)
	env.SetWithVisibility("private", &Integer{Value: 3}, true, VisibilityPrivate)

	bindings := env.GetPublicBindings()
	if len(bindings) != 2 {
		t.Errorf("GetPublicBindings() length = %d, want 2", len(bindings))
	}
	if _, ok := bindings["private"]; ok {
		t.Error("private should not be in public bindings")
	}
}

// ============================================================================
// Visibility Constants
// ============================================================================

func TestVisibilityConstants(t *testing.T) {
	if VisibilityPublic != 0 {
		t.Errorf("VisibilityPublic = %d, want 0", VisibilityPublic)
	}
	if VisibilityPrivate != 1 {
		t.Errorf("VisibilityPrivate = %d, want 1", VisibilityPrivate)
	}
	if VisibilityPrivateModule != 2 {
		t.Errorf("VisibilityPrivateModule = %d, want 2", VisibilityPrivateModule)
	}
}
