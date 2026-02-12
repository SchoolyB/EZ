package stdlib

import (
	"math/big"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// Helper to create a map with string keys and int values
func createStringIntMap(pairs ...interface{}) *object.Map {
	m := object.NewMap()
	m.Mutable = true
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i].(string)
		val := pairs[i+1].(int64)
		m.Set(&object.String{Value: key}, &object.Integer{Value: big.NewInt(val)})
	}
	return m
}

// ============================================================================
// maps.is_empty
// ============================================================================

func TestMapsIsEmptyTrue(t *testing.T) {
	fn := MapsBuiltins["maps.is_empty"]
	m := object.NewMap()
	m.Mutable = true
	result := fn.Fn(m)
	if result != object.TRUE {
		t.Errorf("expected TRUE for empty map, got %v", result)
	}
}

func TestMapsIsEmptyFalse(t *testing.T) {
	fn := MapsBuiltins["maps.is_empty"]
	m := createStringIntMap("a", int64(1))
	result := fn.Fn(m)
	if result != object.FALSE {
		t.Errorf("expected FALSE for non-empty map, got %v", result)
	}
}

func TestMapsIsEmptyWrongType(t *testing.T) {
	fn := MapsBuiltins["maps.is_empty"]
	result := fn.Fn(&object.String{Value: "not a map"})
	if _, ok := result.(*object.Error); !ok {
		t.Errorf("expected error for non-map, got %T", result)
	}
}

// ============================================================================
// maps.get / maps.set
// ============================================================================

func TestMapsGetExists(t *testing.T) {
	fn := MapsBuiltins["maps.get"]
	m := createStringIntMap("key", int64(42))
	result := fn.Fn(m, &object.String{Value: "key"})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestMapsGetNotExists(t *testing.T) {
	fn := MapsBuiltins["maps.get"]
	m := createStringIntMap("key", int64(42))
	result := fn.Fn(m, &object.String{Value: "nonexistent"})
	// Should return nil or error
	if result != object.NIL {
		if _, ok := result.(*object.Error); !ok {
			t.Errorf("expected nil or error for missing key, got %T: %v", result, result)
		}
	}
}

func TestMapsSetNew(t *testing.T) {
	fn := MapsBuiltins["maps.set"]
	m := object.NewMap()
	m.Mutable = true
	result := fn.Fn(m, &object.String{Value: "key"}, &object.Integer{Value: big.NewInt(42)})
	if _, ok := result.(*object.Error); ok {
		t.Errorf("unexpected error: %v", result)
	}
	val, exists := m.Get(&object.String{Value: "key"})
	if !exists {
		t.Errorf("expected key to exist")
	}
	if intVal, ok := val.(*object.Integer); !ok || intVal.Value.Int64() != 42 {
		t.Errorf("expected map[key] = 42")
	}
}

func TestMapsSetOverwrite(t *testing.T) {
	fn := MapsBuiltins["maps.set"]
	m := createStringIntMap("key", int64(1))
	fn.Fn(m, &object.String{Value: "key"}, &object.Integer{Value: big.NewInt(99)})
	val, exists := m.Get(&object.String{Value: "key"})
	if !exists {
		t.Errorf("expected key to exist")
	}
	if intVal, ok := val.(*object.Integer); !ok || intVal.Value.Int64() != 99 {
		t.Errorf("expected map[key] = 99")
	}
}

// ============================================================================
// maps.contains / maps.contains_value
// ============================================================================

func TestMapsContainsTrue(t *testing.T) {
	fn := MapsBuiltins["maps.contains"]
	m := createStringIntMap("key", int64(42))
	result := fn.Fn(m, &object.String{Value: "key"})
	if result != object.TRUE {
		t.Errorf("expected TRUE, got %v", result)
	}
}

func TestMapsContainsFalse(t *testing.T) {
	fn := MapsBuiltins["maps.contains"]
	m := createStringIntMap("key", int64(42))
	result := fn.Fn(m, &object.String{Value: "other"})
	if result != object.FALSE {
		t.Errorf("expected FALSE, got %v", result)
	}
}

func TestMapsContainsValueTrue(t *testing.T) {
	fn := MapsBuiltins["maps.contains_value"]
	m := createStringIntMap("key", int64(42))
	result := fn.Fn(m, &object.Integer{Value: big.NewInt(42)})
	if result != object.TRUE {
		t.Errorf("expected TRUE, got %v", result)
	}
}

func TestMapsContainsValueFalse(t *testing.T) {
	fn := MapsBuiltins["maps.contains_value"]
	m := createStringIntMap("key", int64(42))
	result := fn.Fn(m, &object.Integer{Value: big.NewInt(99)})
	if result != object.FALSE {
		t.Errorf("expected FALSE, got %v", result)
	}
}

// ============================================================================
// maps.keys / maps.values
// ============================================================================

func TestMapsKeysSuccess(t *testing.T) {
	fn := MapsBuiltins["maps.keys"]
	m := createStringIntMap("a", int64(1), "b", int64(2))
	result := fn.Fn(m)
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 keys, got %d", len(arr.Elements))
	}
}

func TestMapsKeysEmptyUnit(t *testing.T) {
	fn := MapsBuiltins["maps.keys"]
	m := object.NewMap()
	m.Mutable = true
	result := fn.Fn(m)
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(arr.Elements) != 0 {
		t.Errorf("expected 0 keys, got %d", len(arr.Elements))
	}
}

func TestMapsValuesSuccess(t *testing.T) {
	fn := MapsBuiltins["maps.values"]
	m := createStringIntMap("a", int64(1), "b", int64(2))
	result := fn.Fn(m)
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 values, got %d", len(arr.Elements))
	}
}

// ============================================================================
// maps.remove
// ============================================================================

func TestMapsRemoveExists(t *testing.T) {
	fn := MapsBuiltins["maps.remove"]
	m := createStringIntMap("a", int64(1), "b", int64(2))
	result := fn.Fn(m, &object.String{Value: "a"})
	if _, ok := result.(*object.Error); ok {
		t.Errorf("unexpected error: %v", result)
	}
	_, exists := m.Get(&object.String{Value: "a"})
	if exists {
		t.Errorf("expected key 'a' to be removed")
	}
}

func TestMapsRemoveNotExists(t *testing.T) {
	fn := MapsBuiltins["maps.remove"]
	m := createStringIntMap("a", int64(1))
	result := fn.Fn(m, &object.String{Value: "nonexistent"})
	// Should not error, just do nothing
	if _, ok := result.(*object.Error); ok {
		t.Errorf("unexpected error: %v", result)
	}
}

// ============================================================================
// maps.clear
// ============================================================================

func TestMapsClear(t *testing.T) {
	fn := MapsBuiltins["maps.clear"]
	m := createStringIntMap("a", int64(1), "b", int64(2))
	result := fn.Fn(m)
	// maps.clear modifies in place and returns nil
	if result != object.NIL {
		t.Errorf("expected nil, got %T", result)
	}
	if len(m.Pairs) != 0 {
		t.Errorf("expected 0 pairs, got %d", len(m.Pairs))
	}
}

// ============================================================================
// maps.merge
// ============================================================================

func TestMapsMergeUnit(t *testing.T) {
	fn := MapsBuiltins["maps.merge"]
	m1 := createStringIntMap("a", int64(1))
	m2 := createStringIntMap("b", int64(2))
	result := fn.Fn(m1, m2)
	resultMap, ok := result.(*object.Map)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if len(resultMap.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(resultMap.Pairs))
	}
}

func TestMapsMergeOverwrite(t *testing.T) {
	fn := MapsBuiltins["maps.merge"]
	m1 := createStringIntMap("a", int64(1))
	m2 := createStringIntMap("a", int64(99))
	result := fn.Fn(m1, m2)
	resultMap, ok := result.(*object.Map)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	// m2 should overwrite m1
	val, exists := resultMap.Get(&object.String{Value: "a"})
	if !exists {
		t.Errorf("expected key 'a' to exist")
	}
	if intVal, ok := val.(*object.Integer); !ok || intVal.Value.Int64() != 99 {
		t.Errorf("expected 'a' to be 99, got %v", val)
	}
}

// ============================================================================
// maps.equals
// ============================================================================

func TestMapsEqualsTrue(t *testing.T) {
	fn := MapsBuiltins["maps.equals"]
	m1 := createStringIntMap("a", int64(1), "b", int64(2))
	m2 := createStringIntMap("a", int64(1), "b", int64(2))
	result := fn.Fn(m1, m2)
	if result != object.TRUE {
		t.Errorf("expected TRUE, got %v", result)
	}
}

func TestMapsEqualsFalse(t *testing.T) {
	fn := MapsBuiltins["maps.equals"]
	m1 := createStringIntMap("a", int64(1))
	m2 := createStringIntMap("a", int64(2))
	result := fn.Fn(m1, m2)
	if result != object.FALSE {
		t.Errorf("expected FALSE, got %v", result)
	}
}

// ============================================================================
// maps.update
// ============================================================================

func TestMapsUpdate(t *testing.T) {
	fn := MapsBuiltins["maps.update"]
	m1 := createStringIntMap("a", int64(1))
	m2 := createStringIntMap("a", int64(99), "b", int64(2))
	result := fn.Fn(m1, m2)
	if _, ok := result.(*object.Error); ok {
		t.Errorf("unexpected error: %v", result)
	}
	// m1 should be updated in place
	val, exists := m1.Get(&object.String{Value: "a"})
	if !exists {
		t.Errorf("expected key 'a' to exist")
	}
	if intVal, ok := val.(*object.Integer); !ok || intVal.Value.Int64() != 99 {
		t.Errorf("expected 'a' to be 99")
	}
	val, exists = m1.Get(&object.String{Value: "b"})
	if !exists {
		t.Errorf("expected key 'b' to exist")
	}
	if intVal, ok := val.(*object.Integer); !ok || intVal.Value.Int64() != 2 {
		t.Errorf("expected 'b' to be 2")
	}
}

// ============================================================================
// maps.to_array / maps.from_array
// ============================================================================

func TestMapsToArray(t *testing.T) {
	fn := MapsBuiltins["maps.to_array"]
	m := createStringIntMap("a", int64(1), "b", int64(2))
	result := fn.Fn(m)
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(arr.Elements))
	}
}

func TestMapsFromArray(t *testing.T) {
	fn := MapsBuiltins["maps.from_array"]
	// Create array of [key, value] pairs
	pair1 := &object.Array{Elements: []object.Object{
		&object.String{Value: "a"},
		&object.Integer{Value: big.NewInt(1)},
	}}
	pair2 := &object.Array{Elements: []object.Object{
		&object.String{Value: "b"},
		&object.Integer{Value: big.NewInt(2)},
	}}
	arr := &object.Array{Elements: []object.Object{pair1, pair2}}
	result := fn.Fn(arr)
	resultMap, ok := result.(*object.Map)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if len(resultMap.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(resultMap.Pairs))
	}
}

// ============================================================================
// maps.invert
// ============================================================================

func TestMapsInvert(t *testing.T) {
	fn := MapsBuiltins["maps.invert"]
	// Create map with string values for inversion
	m := object.NewMap()
	m.Mutable = true
	m.Set(&object.String{Value: "a"}, &object.String{Value: "1"})
	m.Set(&object.String{Value: "b"}, &object.String{Value: "2"})
	result := fn.Fn(m)
	resultMap, ok := result.(*object.Map)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	// Keys and values should be swapped
	_, exists := resultMap.Get(&object.String{Value: "1"})
	if !exists {
		t.Errorf("expected key '1' to exist")
	}
	_, exists = resultMap.Get(&object.String{Value: "2"})
	if !exists {
		t.Errorf("expected key '2' to exist")
	}
}

// ============================================================================
// maps.get_or_set
// ============================================================================

func TestMapsGetOrSetExists(t *testing.T) {
	fn := MapsBuiltins["maps.get_or_set"]
	m := createStringIntMap("key", int64(42))
	result := fn.Fn(m, &object.String{Value: "key"}, &object.Integer{Value: big.NewInt(99)})
	// Should return existing value, not set new one
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestMapsGetOrSetNotExists(t *testing.T) {
	fn := MapsBuiltins["maps.get_or_set"]
	m := object.NewMap()
	m.Mutable = true
	result := fn.Fn(m, &object.String{Value: "key"}, &object.Integer{Value: big.NewInt(42)})
	// Should set and return default value
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 42 {
		t.Errorf("expected 42, got %v", result)
	}
	// Check it was actually set
	val, exists := m.Get(&object.String{Value: "key"})
	if !exists {
		t.Errorf("expected key to exist")
	}
	if intVal, ok := val.(*object.Integer); !ok || intVal.Value.Int64() != 42 {
		t.Errorf("expected map[key] = 42")
	}
}
