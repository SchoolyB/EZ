package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"math/big"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// ============================================================================
// random.float() Tests
// ============================================================================

func TestRandomFloat(t *testing.T) {
	floatFn := RandomBuiltins["random.float"].Fn

	// No args - should return float in [0.0, 1.0)
	for i := 0; i < 100; i++ {
		result := floatFn()
		f, ok := result.(*object.Float)
		if !ok {
			t.Fatalf("random.float() returned %T, want Float", result)
		}
		if f.Value < 0.0 || f.Value >= 1.0 {
			t.Errorf("random.float() = %f, want [0.0, 1.0)", f.Value)
		}
	}
}

func TestRandomFloatRange(t *testing.T) {
	floatFn := RandomBuiltins["random.float"].Fn

	// With min, max - should return float in [min, max)
	min := &object.Float{Value: 5.0}
	max := &object.Float{Value: 10.0}

	for i := 0; i < 100; i++ {
		result := floatFn(min, max)
		f, ok := result.(*object.Float)
		if !ok {
			t.Fatalf("random.float(5.0, 10.0) returned %T, want Float", result)
		}
		if f.Value < 5.0 || f.Value >= 10.0 {
			t.Errorf("random.float(5.0, 10.0) = %f, want [5.0, 10.0)", f.Value)
		}
	}
}

func TestRandomFloatErrors(t *testing.T) {
	floatFn := RandomBuiltins["random.float"].Fn

	// Wrong number of args
	result := floatFn(&object.Float{Value: 5.0})
	if !isErrorObject(result) {
		t.Error("expected error for 1 argument")
	}

	// max <= min
	result = floatFn(&object.Float{Value: 10.0}, &object.Float{Value: 5.0})
	if !isErrorObject(result) {
		t.Error("expected error for max <= min")
	}

	// max == min
	result = floatFn(&object.Float{Value: 5.0}, &object.Float{Value: 5.0})
	if !isErrorObject(result) {
		t.Error("expected error for max == min")
	}
}

// ============================================================================
// random.int() Tests
// ============================================================================

func TestRandomInt(t *testing.T) {
	intFn := RandomBuiltins["random.int"].Fn

	// With max - should return int in [0, max)
	max := &object.Integer{Value: big.NewInt(100)}

	for i := 0; i < 100; i++ {
		result := intFn(max)
		n, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("random.int(100) returned %T, want Integer", result)
		}
		if n.Value.Sign() < 0 || n.Value.Cmp(big.NewInt(100)) >= 0 {
			t.Errorf("random.int(100) = %s, want [0, 100)", n.Value.String())
		}
	}
}

func TestRandomIntRange(t *testing.T) {
	intFn := RandomBuiltins["random.int"].Fn

	// With min, max - should return int in [min, max)
	min := &object.Integer{Value: big.NewInt(10)}
	max := &object.Integer{Value: big.NewInt(20)}

	for i := 0; i < 100; i++ {
		result := intFn(min, max)
		n, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("random.int(10, 20) returned %T, want Integer", result)
		}
		if n.Value.Cmp(big.NewInt(10)) < 0 || n.Value.Cmp(big.NewInt(20)) >= 0 {
			t.Errorf("random.int(10, 20) = %s, want [10, 20)", n.Value.String())
		}
	}
}

func TestRandomIntErrors(t *testing.T) {
	intFn := RandomBuiltins["random.int"].Fn

	// No args
	result := intFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// max <= 0
	result = intFn(&object.Integer{Value: big.NewInt(0)})
	if !isErrorObject(result) {
		t.Error("expected error for max <= 0")
	}

	result = intFn(&object.Integer{Value: big.NewInt(-5)})
	if !isErrorObject(result) {
		t.Error("expected error for negative max")
	}

	// max <= min
	result = intFn(&object.Integer{Value: big.NewInt(20)}, &object.Integer{Value: big.NewInt(10)})
	if !isErrorObject(result) {
		t.Error("expected error for max <= min")
	}
}

// ============================================================================
// random.bool() Tests
// ============================================================================

func TestRandomBool(t *testing.T) {
	boolFn := RandomBuiltins["random.bool"].Fn

	trueCount := 0
	falseCount := 0

	// Run many times to verify both values are possible
	for i := 0; i < 1000; i++ {
		result := boolFn()
		b, ok := result.(*object.Boolean)
		if !ok {
			t.Fatalf("random.bool() returned %T, want Boolean", result)
		}
		if b.Value {
			trueCount++
		} else {
			falseCount++
		}
	}

	// Both should appear with reasonable frequency
	if trueCount == 0 {
		t.Error("random.bool() never returned true in 1000 calls")
	}
	if falseCount == 0 {
		t.Error("random.bool() never returned false in 1000 calls")
	}
}

func TestRandomBoolErrors(t *testing.T) {
	boolFn := RandomBuiltins["random.bool"].Fn

	// With args - should error
	result := boolFn(&object.Integer{Value: big.NewInt(1)})
	if !isErrorObject(result) {
		t.Error("expected error for arguments")
	}
}

// ============================================================================
// random.byte() Tests
// ============================================================================

func TestRandomByte(t *testing.T) {
	byteFn := RandomBuiltins["random.byte"].Fn

	for i := 0; i < 100; i++ {
		result := byteFn()
		b, ok := result.(*object.Byte)
		if !ok {
			t.Fatalf("random.byte() returned %T, want Byte", result)
		}
		// Byte is uint8, so always in [0, 255] - no range check needed
		_ = b.Value
	}
}

func TestRandomByteErrors(t *testing.T) {
	byteFn := RandomBuiltins["random.byte"].Fn

	// With args - should error
	result := byteFn(&object.Integer{Value: big.NewInt(1)})
	if !isErrorObject(result) {
		t.Error("expected error for arguments")
	}
}

// ============================================================================
// random.char() Tests
// ============================================================================

func TestRandomChar(t *testing.T) {
	charFn := RandomBuiltins["random.char"].Fn

	// No args - should return printable ASCII [32, 126]
	for i := 0; i < 100; i++ {
		result := charFn()
		c, ok := result.(*object.Char)
		if !ok {
			t.Fatalf("random.char() returned %T, want Char", result)
		}
		if c.Value < 32 || c.Value > 126 {
			t.Errorf("random.char() = %d (%c), want [32, 126]", c.Value, c.Value)
		}
	}
}

func TestRandomCharRange(t *testing.T) {
	charFn := RandomBuiltins["random.char"].Fn

	// With char range
	min := &object.Char{Value: 'a'}
	max := &object.Char{Value: 'z'}

	for i := 0; i < 100; i++ {
		result := charFn(min, max)
		c, ok := result.(*object.Char)
		if !ok {
			t.Fatalf("random.char('a', 'z') returned %T, want Char", result)
		}
		if c.Value < 'a' || c.Value > 'z' {
			t.Errorf("random.char('a', 'z') = %c, want [a-z]", c.Value)
		}
	}

	// With integer range
	minInt := &object.Integer{Value: big.NewInt(65)} // 'A'
	maxInt := &object.Integer{Value: big.NewInt(90)} // 'Z'

	for i := 0; i < 100; i++ {
		result := charFn(minInt, maxInt)
		c, ok := result.(*object.Char)
		if !ok {
			t.Fatalf("random.char(65, 90) returned %T, want Char", result)
		}
		if c.Value < 'A' || c.Value > 'Z' {
			t.Errorf("random.char(65, 90) = %c, want [A-Z]", c.Value)
		}
	}
}

func TestRandomCharErrors(t *testing.T) {
	charFn := RandomBuiltins["random.char"].Fn

	// Wrong number of args
	result := charFn(&object.Char{Value: 'a'})
	if !isErrorObject(result) {
		t.Error("expected error for 1 argument")
	}

	// max <= min
	result = charFn(&object.Char{Value: 'z'}, &object.Char{Value: 'a'})
	if !isErrorObject(result) {
		t.Error("expected error for max <= min")
	}

	// Invalid type
	result = charFn(&object.String{Value: "a"}, &object.String{Value: "z"})
	if !isErrorObject(result) {
		t.Error("expected error for string arguments")
	}
}

// ============================================================================
// random.choice() Tests
// ============================================================================

func TestRandomChoice(t *testing.T) {
	choiceFn := RandomBuiltins["random.choice"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(2)},
		&object.Integer{Value: big.NewInt(3)},
	}}

	counts := make(map[int64]int)

	for i := 0; i < 300; i++ {
		result := choiceFn(arr)
		n, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("random.choice() returned %T, want Integer", result)
		}
		counts[n.Value.Int64()]++
	}

	// All elements should be chosen at least once
	for _, val := range []int64{1, 2, 3} {
		if counts[val] == 0 {
			t.Errorf("random.choice() never chose %d in 300 calls", val)
		}
	}
}

func TestRandomChoiceErrors(t *testing.T) {
	choiceFn := RandomBuiltins["random.choice"].Fn

	// No args
	result := choiceFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Not an array
	result = choiceFn(&object.Integer{Value: big.NewInt(5)})
	if !isErrorObject(result) {
		t.Error("expected error for non-array argument")
	}

	// Empty array
	result = choiceFn(&object.Array{Elements: []object.Object{}})
	if !isErrorObject(result) {
		t.Error("expected error for empty array")
	}
}

// ============================================================================
// random.shuffle() Tests
// ============================================================================

func TestRandomShuffle(t *testing.T) {
	shuffleFn := RandomBuiltins["random.shuffle"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(2)},
		&object.Integer{Value: big.NewInt(3)},
		&object.Integer{Value: big.NewInt(4)},
		&object.Integer{Value: big.NewInt(5)},
	}}

	result := shuffleFn(arr)
	shuffled, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("random.shuffle() returned %T, want Array", result)
	}

	// Same length
	if len(shuffled.Elements) != len(arr.Elements) {
		t.Errorf("shuffled array has length %d, want %d", len(shuffled.Elements), len(arr.Elements))
	}

	// Contains same elements (count each)
	originalCounts := make(map[int64]int)
	shuffledCounts := make(map[int64]int)

	for _, el := range arr.Elements {
		originalCounts[el.(*object.Integer).Value.Int64()]++
	}
	for _, el := range shuffled.Elements {
		shuffledCounts[el.(*object.Integer).Value.Int64()]++
	}

	for k, v := range originalCounts {
		if shuffledCounts[k] != v {
			t.Errorf("element %d count mismatch: got %d, want %d", k, shuffledCounts[k], v)
		}
	}

	// Original should be unchanged
	for i, expected := range []int64{1, 2, 3, 4, 5} {
		if arr.Elements[i].(*object.Integer).Value.Int64() != expected {
			t.Errorf("original array was modified at index %d", i)
		}
	}
}

func TestRandomShuffleErrors(t *testing.T) {
	shuffleFn := RandomBuiltins["random.shuffle"].Fn

	// No args
	result := shuffleFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Not an array
	result = shuffleFn(&object.Integer{Value: big.NewInt(5)})
	if !isErrorObject(result) {
		t.Error("expected error for non-array argument")
	}
}

// ============================================================================
// random.sample() Tests
// ============================================================================

func TestRandomSample(t *testing.T) {
	sampleFn := RandomBuiltins["random.sample"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(2)},
		&object.Integer{Value: big.NewInt(3)},
		&object.Integer{Value: big.NewInt(4)},
		&object.Integer{Value: big.NewInt(5)},
	}}

	result := sampleFn(arr, &object.Integer{Value: big.NewInt(3)})
	sampled, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("random.sample() returned %T, want Array", result)
	}

	// Correct length
	if len(sampled.Elements) != 3 {
		t.Errorf("sampled array has length %d, want 3", len(sampled.Elements))
	}

	// All elements should be from original array
	validValues := map[int64]bool{1: true, 2: true, 3: true, 4: true, 5: true}
	for _, el := range sampled.Elements {
		val := el.(*object.Integer).Value.Int64()
		if !validValues[val] {
			t.Errorf("sampled element %d not in original array", val)
		}
	}

	// No duplicates
	seen := make(map[int64]bool)
	for _, el := range sampled.Elements {
		val := el.(*object.Integer).Value.Int64()
		if seen[val] {
			t.Errorf("duplicate element %d in sample", val)
		}
		seen[val] = true
	}
}

func TestRandomSampleZero(t *testing.T) {
	sampleFn := RandomBuiltins["random.sample"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(2)},
	}}

	result := sampleFn(arr, &object.Integer{Value: big.NewInt(0)})
	sampled, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("random.sample(arr, 0) returned %T, want Array", result)
	}

	if len(sampled.Elements) != 0 {
		t.Errorf("random.sample(arr, 0) returned %d elements, want 0", len(sampled.Elements))
	}
}

func TestRandomSampleAll(t *testing.T) {
	sampleFn := RandomBuiltins["random.sample"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(2)},
		&object.Integer{Value: big.NewInt(3)},
	}}

	result := sampleFn(arr, &object.Integer{Value: big.NewInt(3)})
	sampled, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("random.sample() returned %T, want Array", result)
	}

	if len(sampled.Elements) != 3 {
		t.Errorf("random.sample(arr, 3) returned %d elements, want 3", len(sampled.Elements))
	}
}

func TestRandomSampleErrors(t *testing.T) {
	sampleFn := RandomBuiltins["random.sample"].Fn

	arr := &object.Array{Elements: []object.Object{
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(2)},
	}}

	// No args
	result := sampleFn()
	if !isErrorObject(result) {
		t.Error("expected error for no arguments")
	}

	// Not an array
	result = sampleFn(&object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(1)})
	if !isErrorObject(result) {
		t.Error("expected error for non-array argument")
	}

	// n not an integer
	result = sampleFn(arr, &object.Float{Value: 1.5})
	if !isErrorObject(result) {
		t.Error("expected error for non-integer n")
	}

	// n < 0
	result = sampleFn(arr, &object.Integer{Value: big.NewInt(-1)})
	if !isErrorObject(result) {
		t.Error("expected error for negative n")
	}

	// n > len(array)
	result = sampleFn(arr, &object.Integer{Value: big.NewInt(5)})
	if !isErrorObject(result) {
		t.Error("expected error for n > array length")
	}
}
