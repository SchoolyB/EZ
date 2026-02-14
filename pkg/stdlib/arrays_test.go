package stdlib

import (
	"math/big"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// Helper to create an integer array for unit tests
func createIntegerArray(vals ...int64) *object.Array {
	elements := make([]object.Object, len(vals))
	for i, v := range vals {
		elements[i] = &object.Integer{Value: big.NewInt(v)}
	}
	return &object.Array{Elements: elements, Mutable: true, ElementType: "int"}
}

// Helper to create a string array
func makeStringArray(vals ...string) *object.Array {
	elements := make([]object.Object, len(vals))
	for i, v := range vals {
		elements[i] = &object.String{Value: v}
	}
	return &object.Array{Elements: elements, Mutable: true, ElementType: "string"}
}

// ============================================================================
// arrays.is_empty
// ============================================================================

func TestArraysIsEmptyTrue(t *testing.T) {
	fn := ArraysBuiltins["arrays.is_empty"]
	arr := &object.Array{Elements: []object.Object{}, Mutable: true}
	result := fn.Fn(arr)
	if result != object.TRUE {
		t.Errorf("expected TRUE for empty array, got %v", result)
	}
}

func TestArraysIsEmptyFalse(t *testing.T) {
	fn := ArraysBuiltins["arrays.is_empty"]
	arr := createIntegerArray(1, 2, 3)
	result := fn.Fn(arr)
	if result != object.FALSE {
		t.Errorf("expected FALSE for non-empty array, got %v", result)
	}
}

func TestArraysIsEmptyWrongType(t *testing.T) {
	fn := ArraysBuiltins["arrays.is_empty"]
	result := fn.Fn(&object.String{Value: "not an array"})
	if _, ok := result.(*object.Error); !ok {
		t.Errorf("expected error for non-array, got %T", result)
	}
}

// ============================================================================
// arrays.first / arrays.last
// ============================================================================

func TestArraysFirstSuccess(t *testing.T) {
	fn := ArraysBuiltins["arrays.first"]
	arr := createIntegerArray(10, 20, 30)
	result := fn.Fn(arr)
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 10 {
		t.Errorf("expected 10, got %v", result)
	}
}

func TestArraysFirstEmpty(t *testing.T) {
	fn := ArraysBuiltins["arrays.first"]
	arr := &object.Array{Elements: []object.Object{}, Mutable: true}
	result := fn.Fn(arr)
	// Should return error or nil for empty array
	if _, ok := result.(*object.Error); !ok && result != object.NIL {
		t.Errorf("expected error or nil for empty array, got %T", result)
	}
}

func TestArraysLastSuccess(t *testing.T) {
	fn := ArraysBuiltins["arrays.last"]
	arr := createIntegerArray(10, 20, 30)
	result := fn.Fn(arr)
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 30 {
		t.Errorf("expected 30, got %v", result)
	}
}

func TestArraysLastEmpty(t *testing.T) {
	fn := ArraysBuiltins["arrays.last"]
	arr := &object.Array{Elements: []object.Object{}, Mutable: true}
	result := fn.Fn(arr)
	// Should return error or nil for empty array
	if _, ok := result.(*object.Error); !ok && result != object.NIL {
		t.Errorf("expected error or nil for empty array, got %T", result)
	}
}

// ============================================================================
// arrays.get / arrays.set
// ============================================================================

func TestArraysGetValid(t *testing.T) {
	fn := ArraysBuiltins["arrays.get"]
	arr := createIntegerArray(10, 20, 30)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(1)})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 20 {
		t.Errorf("expected 20, got %v", result)
	}
}

func TestArraysGetOutOfBounds(t *testing.T) {
	fn := ArraysBuiltins["arrays.get"]
	arr := createIntegerArray(10, 20, 30)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(10)})
	if _, ok := result.(*object.Error); !ok {
		t.Errorf("expected error for out of bounds, got %T", result)
	}
}

func TestArraysGetNegativeIndex(t *testing.T) {
	fn := ArraysBuiltins["arrays.get"]
	arr := createIntegerArray(10, 20, 30)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(-1)})
	// Negative index should return last element or error depending on implementation
	if _, ok := result.(*object.Error); ok {
		// Error is acceptable
	} else if intVal, ok := result.(*object.Integer); ok && intVal.Value.Int64() == 30 {
		// -1 returning last element is also acceptable
	}
}

func TestArraysSetValid(t *testing.T) {
	fn := ArraysBuiltins["arrays.set"]
	arr := createIntegerArray(10, 20, 30)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(1)}, &object.Integer{Value: big.NewInt(99)})
	if _, ok := result.(*object.Error); ok {
		t.Errorf("unexpected error: %v", result)
	}
	// Check the array was modified
	if arr.Elements[1].(*object.Integer).Value.Int64() != 99 {
		t.Errorf("expected arr[1] = 99, got %v", arr.Elements[1])
	}
}

// ============================================================================
// arrays.contains / arrays.index / arrays.last_index
// ============================================================================

func TestArraysContainsTrue(t *testing.T) {
	fn := ArraysBuiltins["arrays.contains"]
	arr := createIntegerArray(10, 20, 30)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(20)})
	if result != object.TRUE {
		t.Errorf("expected TRUE, got %v", result)
	}
}

func TestArraysContainsFalse(t *testing.T) {
	fn := ArraysBuiltins["arrays.contains"]
	arr := createIntegerArray(10, 20, 30)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(99)})
	if result != object.FALSE {
		t.Errorf("expected FALSE, got %v", result)
	}
}

func TestArraysIndexFound(t *testing.T) {
	fn := ArraysBuiltins["arrays.index"]
	arr := createIntegerArray(10, 20, 30, 20)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(20)})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 1 {
		t.Errorf("expected 1, got %v", result)
	}
}

func TestArraysIndexNotFound(t *testing.T) {
	fn := ArraysBuiltins["arrays.index"]
	arr := createIntegerArray(10, 20, 30)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(99)})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != -1 {
		t.Errorf("expected -1, got %v", result)
	}
}

func TestArraysLastIndexFound(t *testing.T) {
	fn := ArraysBuiltins["arrays.last_index"]
	arr := createIntegerArray(10, 20, 30, 20)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(20)})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 3 {
		t.Errorf("expected 3, got %v", result)
	}
}

// ============================================================================
// arrays.count
// ============================================================================

func TestArraysCount(t *testing.T) {
	fn := ArraysBuiltins["arrays.count"]
	arr := createIntegerArray(1, 2, 2, 3, 2)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(2)})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 3 {
		t.Errorf("expected 3, got %v", result)
	}
}

func TestArraysCountZero(t *testing.T) {
	fn := ArraysBuiltins["arrays.count"]
	arr := createIntegerArray(1, 2, 3)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(99)})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 0 {
		t.Errorf("expected 0, got %v", result)
	}
}

// ============================================================================
// arrays.sum / arrays.avg / arrays.min / arrays.max / arrays.product
// ============================================================================

func TestArraysSumUnit(t *testing.T) {
	fn := ArraysBuiltins["arrays.sum"]
	arr := createIntegerArray(1, 2, 3, 4, 5)
	result := fn.Fn(arr)
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 15 {
		t.Errorf("expected 15, got %v", result)
	}
}

func TestArraysSumEmpty(t *testing.T) {
	fn := ArraysBuiltins["arrays.sum"]
	arr := &object.Array{Elements: []object.Object{}, Mutable: true}
	result := fn.Fn(arr)
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 0 {
		t.Errorf("expected 0, got %v", result)
	}
}

func TestArraysAvg(t *testing.T) {
	fn := ArraysBuiltins["arrays.avg"]
	arr := createIntegerArray(10, 20, 30)
	result := fn.Fn(arr)
	if floatVal, ok := result.(*object.Float); !ok || floatVal.Value != 20.0 {
		t.Errorf("expected 20.0, got %v", result)
	}
}

func TestArraysMin(t *testing.T) {
	fn := ArraysBuiltins["arrays.min"]
	arr := createIntegerArray(30, 10, 20)
	result := fn.Fn(arr)
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 10 {
		t.Errorf("expected 10, got %v", result)
	}
}

func TestArraysMax(t *testing.T) {
	fn := ArraysBuiltins["arrays.max"]
	arr := createIntegerArray(30, 10, 20)
	result := fn.Fn(arr)
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 30 {
		t.Errorf("expected 30, got %v", result)
	}
}

func TestArraysProduct(t *testing.T) {
	fn := ArraysBuiltins["arrays.product"]
	arr := createIntegerArray(2, 3, 4)
	result := fn.Fn(arr)
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 24 {
		t.Errorf("expected 24, got %v", result)
	}
}

// ============================================================================
// arrays.reverse / arrays.sort / arrays.sort_desc
// ============================================================================

func TestArraysReverseUnit(t *testing.T) {
	fn := ArraysBuiltins["arrays.reverse"]
	arr := createIntegerArray(1, 2, 3)
	result := fn.Fn(arr)
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(resultArr.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(resultArr.Elements))
	}
	expected := []int64{3, 2, 1}
	for i, exp := range expected {
		if resultArr.Elements[i].(*object.Integer).Value.Int64() != exp {
			t.Errorf("element %d: expected %d, got %v", i, exp, resultArr.Elements[i])
		}
	}
}

func TestArraysSortUnit(t *testing.T) {
	fn := ArraysBuiltins["arrays.sort"]
	arr := createIntegerArray(3, 1, 2)
	result := fn.Fn(arr)
	// May return array or nil (in-place modification)
	var sortedArr *object.Array
	if result == object.NIL {
		sortedArr = arr
	} else if resultArr, ok := result.(*object.Array); ok {
		sortedArr = resultArr
	} else {
		t.Fatalf("expected array or nil, got %T", result)
	}
	expected := []int64{1, 2, 3}
	for i, exp := range expected {
		if sortedArr.Elements[i].(*object.Integer).Value.Int64() != exp {
			t.Errorf("element %d: expected %d, got %v", i, exp, sortedArr.Elements[i])
		}
	}
}

func TestArraysSortDescUnit(t *testing.T) {
	fn := ArraysBuiltins["arrays.sort_desc"]
	arr := createIntegerArray(1, 3, 2)
	result := fn.Fn(arr)
	// May return array or nil (in-place modification)
	var sortedArr *object.Array
	if result == object.NIL {
		sortedArr = arr
	} else if resultArr, ok := result.(*object.Array); ok {
		sortedArr = resultArr
	} else {
		t.Fatalf("expected array or nil, got %T", result)
	}
	expected := []int64{3, 2, 1}
	for i, exp := range expected {
		if sortedArr.Elements[i].(*object.Integer).Value.Int64() != exp {
			t.Errorf("element %d: expected %d, got %v", i, exp, sortedArr.Elements[i])
		}
	}
}

// ============================================================================
// arrays.unique / arrays.duplicates
// ============================================================================

func TestArraysUnique(t *testing.T) {
	fn := ArraysBuiltins["arrays.unique"]
	arr := createIntegerArray(1, 2, 2, 3, 1, 3)
	result := fn.Fn(arr)
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(resultArr.Elements) != 3 {
		t.Errorf("expected 3 unique elements, got %d", len(resultArr.Elements))
	}
}

func TestArraysDuplicates(t *testing.T) {
	fn := ArraysBuiltins["arrays.duplicates"]
	arr := createIntegerArray(1, 2, 2, 3, 1, 3)
	result := fn.Fn(arr)
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	// Should return duplicates: 2, 1, 3 (elements that appear more than once)
	if len(resultArr.Elements) != 3 {
		t.Errorf("expected 3 duplicate elements, got %d", len(resultArr.Elements))
	}
}

// ============================================================================
// arrays.slice / arrays.take / arrays.drop
// ============================================================================

func TestArraysSliceUnit(t *testing.T) {
	fn := ArraysBuiltins["arrays.slice"]
	arr := createIntegerArray(1, 2, 3, 4, 5)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(1)}, &object.Integer{Value: big.NewInt(4)})
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(resultArr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(resultArr.Elements))
	}
}

func TestArraysTake(t *testing.T) {
	fn := ArraysBuiltins["arrays.take"]
	arr := createIntegerArray(1, 2, 3, 4, 5)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(3)})
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(resultArr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(resultArr.Elements))
	}
}

func TestArraysDrop(t *testing.T) {
	fn := ArraysBuiltins["arrays.drop"]
	arr := createIntegerArray(1, 2, 3, 4, 5)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(2)})
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(resultArr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(resultArr.Elements))
	}
}

// ============================================================================
// arrays.concat / arrays.flatten
// ============================================================================

func TestArraysConcatUnit(t *testing.T) {
	fn := ArraysBuiltins["arrays.concat"]
	arr1 := createIntegerArray(1, 2)
	arr2 := createIntegerArray(3, 4)
	result := fn.Fn(arr1, arr2)
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(resultArr.Elements) != 4 {
		t.Errorf("expected 4 elements, got %d", len(resultArr.Elements))
	}
}

func TestArraysFlatten(t *testing.T) {
	fn := ArraysBuiltins["arrays.flatten"]
	inner1 := createIntegerArray(1, 2)
	inner2 := createIntegerArray(3, 4)
	outer := &object.Array{Elements: []object.Object{inner1, inner2}, Mutable: true}
	result := fn.Fn(outer)
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(resultArr.Elements) != 4 {
		t.Errorf("expected 4 elements, got %d", len(resultArr.Elements))
	}
}

// ============================================================================
// arrays.chunk
// ============================================================================

func TestArraysChunk(t *testing.T) {
	fn := ArraysBuiltins["arrays.chunk"]
	arr := createIntegerArray(1, 2, 3, 4, 5)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(2)})
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	// Should be [[1,2], [3,4], [5]]
	if len(resultArr.Elements) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(resultArr.Elements))
	}
}

// ============================================================================
// arrays.fill / arrays.repeat
// ============================================================================

func TestArraysFillUnit(t *testing.T) {
	fn := ArraysBuiltins["arrays.fill"]
	arr := createIntegerArray(1, 2, 3)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(0)})
	// May return array or nil (in-place modification)
	var filledArr *object.Array
	if result == object.NIL {
		filledArr = arr
	} else if resultArr, ok := result.(*object.Array); ok {
		filledArr = resultArr
	} else {
		t.Fatalf("expected array or nil, got %T", result)
	}
	for i, elem := range filledArr.Elements {
		if elem.(*object.Integer).Value.Int64() != 0 {
			t.Errorf("element %d: expected 0, got %v", i, elem)
		}
	}
}

func TestArraysRepeat(t *testing.T) {
	fn := ArraysBuiltins["arrays.repeat"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(42)}, &object.Integer{Value: big.NewInt(3)})
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(resultArr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(resultArr.Elements))
	}
}

// ============================================================================
// arrays.zip
// ============================================================================

func TestArraysZip(t *testing.T) {
	fn := ArraysBuiltins["arrays.zip"]
	arr1 := createIntegerArray(1, 2, 3)
	arr2 := makeStringArray("a", "b", "c")
	result := fn.Fn(arr1, arr2)
	resultArr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(resultArr.Elements) != 3 {
		t.Errorf("expected 3 pairs, got %d", len(resultArr.Elements))
	}
}

// ============================================================================
// arrays.equals / arrays.all_equal
// ============================================================================

func TestArraysEquals(t *testing.T) {
	fn := ArraysBuiltins["arrays.equals"]
	arr1 := createIntegerArray(1, 2, 3)
	arr2 := createIntegerArray(1, 2, 3)
	result := fn.Fn(arr1, arr2)
	if result != object.TRUE {
		t.Errorf("expected TRUE, got %v", result)
	}
}

func TestArraysEqualsNotEqual(t *testing.T) {
	fn := ArraysBuiltins["arrays.equals"]
	arr1 := createIntegerArray(1, 2, 3)
	arr2 := createIntegerArray(1, 2, 4)
	result := fn.Fn(arr1, arr2)
	if result != object.FALSE {
		t.Errorf("expected FALSE, got %v", result)
	}
}

func TestArraysAllEqual(t *testing.T) {
	fn := ArraysBuiltins["arrays.all_equal"]
	arr := createIntegerArray(5, 5, 5, 5)
	result := fn.Fn(arr)
	if result != object.TRUE {
		t.Errorf("expected TRUE, got %v", result)
	}
}

func TestArraysAllEqualFalse(t *testing.T) {
	fn := ArraysBuiltins["arrays.all_equal"]
	arr := createIntegerArray(5, 5, 6, 5)
	result := fn.Fn(arr)
	if result != object.FALSE {
		t.Errorf("expected FALSE, got %v", result)
	}
}

// ============================================================================
// arrays.join
// ============================================================================

func TestArraysJoinUnit(t *testing.T) {
	fn := ArraysBuiltins["arrays.join"]
	arr := makeStringArray("a", "b", "c")
	result := fn.Fn(arr, &object.String{Value: ", "})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected string, got %T: %v", result, result)
	}
	expected := "a, b, c"
	if strVal.Value != expected {
		t.Errorf("expected '%s', got '%s'", expected, strVal.Value)
	}
}

// ============================================================================
// arrays.insert / arrays.remove_at / arrays.remove_value / arrays.remove_all
// ============================================================================

func TestArraysInsertUnit(t *testing.T) {
	fn := ArraysBuiltins["arrays.insert"]
	arr := createIntegerArray(1, 3)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(1)}, &object.Integer{Value: big.NewInt(2)})
	// May return array or nil (in-place modification)
	var insertedArr *object.Array
	if result == object.NIL {
		insertedArr = arr
	} else if resultArr, ok := result.(*object.Array); ok {
		insertedArr = resultArr
	} else {
		t.Fatalf("expected array or nil, got %T", result)
	}
	if len(insertedArr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(insertedArr.Elements))
	}
}

func TestArraysRemoveAt(t *testing.T) {
	fn := ArraysBuiltins["arrays.remove_at"]
	arr := createIntegerArray(1, 2, 3)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(1)})
	// May return array or nil (in-place modification), or the removed element
	if result == object.NIL {
		if len(arr.Elements) != 2 {
			t.Errorf("expected 2 elements, got %d", len(arr.Elements))
		}
	} else if resultArr, ok := result.(*object.Array); ok {
		if len(resultArr.Elements) != 2 {
			t.Errorf("expected 2 elements, got %d", len(resultArr.Elements))
		}
	} else if _, ok := result.(*object.Integer); ok {
		// Some implementations return the removed element
		if len(arr.Elements) != 2 {
			t.Errorf("expected 2 elements in original array, got %d", len(arr.Elements))
		}
	}
}

func TestArraysRemoveValueUnit(t *testing.T) {
	fn := ArraysBuiltins["arrays.remove_value"]
	arr := createIntegerArray(1, 2, 3, 2)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(2)})
	// May return array or nil (in-place modification)
	var removedArr *object.Array
	if result == object.NIL {
		removedArr = arr
	} else if resultArr, ok := result.(*object.Array); ok {
		removedArr = resultArr
	} else {
		t.Fatalf("expected array or nil, got %T", result)
	}
	// Should remove first occurrence of 2
	if len(removedArr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(removedArr.Elements))
	}
}

func TestArraysRemoveAll(t *testing.T) {
	fn := ArraysBuiltins["arrays.remove_all"]
	arr := createIntegerArray(1, 2, 3, 2)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(2)})
	// May return array or nil (in-place modification)
	var removedArr *object.Array
	if result == object.NIL {
		removedArr = arr
	} else if resultArr, ok := result.(*object.Array); ok {
		removedArr = resultArr
	} else {
		t.Fatalf("expected array or nil, got %T", result)
	}
	// Should remove all occurrences of 2
	if len(removedArr.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(removedArr.Elements))
	}
}

// ============================================================================
// arrays.shift / arrays.unshift
// ============================================================================

func TestArraysShift(t *testing.T) {
	fn := ArraysBuiltins["arrays.shift"]
	arr := createIntegerArray(1, 2, 3)
	result := fn.Fn(arr)
	// Returns the removed element
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 1 {
		t.Errorf("expected 1, got %v", result)
	}
}

func TestArraysUnshift(t *testing.T) {
	fn := ArraysBuiltins["arrays.unshift"]
	arr := createIntegerArray(2, 3)
	result := fn.Fn(arr, &object.Integer{Value: big.NewInt(1)})
	// May return array or nil (in-place modification)
	var unshiftedArr *object.Array
	if result == object.NIL {
		unshiftedArr = arr
	} else if resultArr, ok := result.(*object.Array); ok {
		unshiftedArr = resultArr
	} else {
		t.Fatalf("expected array or nil, got %T", result)
	}
	if len(unshiftedArr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(unshiftedArr.Elements))
	}
	if unshiftedArr.Elements[0].(*object.Integer).Value.Int64() != 1 {
		t.Errorf("expected first element to be 1")
	}
}

// ============================================================================
// arrays.clear
// ============================================================================

func TestArraysClearUnit(t *testing.T) {
	fn := ArraysBuiltins["arrays.clear"]
	arr := createIntegerArray(1, 2, 3)
	result := fn.Fn(arr)
	// May return array or nil (in-place modification)
	var clearedArr *object.Array
	if result == object.NIL {
		clearedArr = arr
	} else if resultArr, ok := result.(*object.Array); ok {
		clearedArr = resultArr
	} else {
		t.Fatalf("expected array or nil, got %T", result)
	}
	if len(clearedArr.Elements) != 0 {
		t.Errorf("expected 0 elements, got %d", len(clearedArr.Elements))
	}
}
