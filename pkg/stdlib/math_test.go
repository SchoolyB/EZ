package stdlib

import (
	"math"
	"math/big"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// ============================================================================
// Basic arithmetic: add, sub, mul, div, mod
// ============================================================================

func TestMathAddIntegers(t *testing.T) {
	fn := MathBuiltins["math.add"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(3)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 8 {
		t.Errorf("expected 8, got %v", result)
	}
}

func TestMathAddFloats(t *testing.T) {
	fn := MathBuiltins["math.add"]
	result := fn.Fn(&object.Float{Value: 2.5}, &object.Float{Value: 1.5})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 4.0 {
		t.Errorf("expected 4.0, got %v", result)
	}
}

func TestMathSubIntegers(t *testing.T) {
	fn := MathBuiltins["math.sub"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(10)}, &object.Integer{Value: big.NewInt(3)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 7 {
		t.Errorf("expected 7, got %v", result)
	}
}

func TestMathMulIntegers(t *testing.T) {
	fn := MathBuiltins["math.mul"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(4)}, &object.Integer{Value: big.NewInt(5)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 20 {
		t.Errorf("expected 20, got %v", result)
	}
}

func TestMathDivIntegers(t *testing.T) {
	fn := MathBuiltins["math.div"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(20)}, &object.Integer{Value: big.NewInt(4)})
	switch v := result.(type) {
	case *object.Integer:
		if v.Value.Int64() != 5 {
			t.Errorf("expected 5, got %v", result)
		}
	case *object.Float:
		if v.Value != 5.0 {
			t.Errorf("expected 5.0, got %v", result)
		}
	default:
		t.Errorf("expected Integer or Float, got %T", result)
	}
}

func TestMathDivByZero(t *testing.T) {
	fn := MathBuiltins["math.div"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(10)}, &object.Integer{Value: big.NewInt(0)})
	if _, ok := result.(*object.Error); !ok {
		t.Errorf("expected error for division by zero, got %T", result)
	}
}

func TestMathModIntegers(t *testing.T) {
	fn := MathBuiltins["math.mod"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(10)}, &object.Integer{Value: big.NewInt(3)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 1 {
		t.Errorf("expected 1, got %v", result)
	}
}

// ============================================================================
// abs, neg, sign
// ============================================================================

func TestMathAbsPositive(t *testing.T) {
	fn := MathBuiltins["math.abs"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(5)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 5 {
		t.Errorf("expected 5, got %v", result)
	}
}

func TestMathAbsNegative(t *testing.T) {
	fn := MathBuiltins["math.abs"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(-5)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 5 {
		t.Errorf("expected 5, got %v", result)
	}
}

func TestMathAbsNegativeFloatUnit(t *testing.T) {
	fn := MathBuiltins["math.abs"]
	result := fn.Fn(&object.Float{Value: -3.14})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 3.14 {
		t.Errorf("expected 3.14, got %v", result)
	}
}

func TestMathNegUnit(t *testing.T) {
	fn := MathBuiltins["math.neg"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(5)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != -5 {
		t.Errorf("expected -5, got %v", result)
	}
}

func TestMathSignPositive(t *testing.T) {
	fn := MathBuiltins["math.sign"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(42)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 1 {
		t.Errorf("expected 1, got %v", result)
	}
}

func TestMathSignNegative(t *testing.T) {
	fn := MathBuiltins["math.sign"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(-42)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != -1 {
		t.Errorf("expected -1, got %v", result)
	}
}

func TestMathSignZero(t *testing.T) {
	fn := MathBuiltins["math.sign"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(0)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 0 {
		t.Errorf("expected 0, got %v", result)
	}
}

// ============================================================================
// min, max, clamp
// ============================================================================

func TestMathMinIntegers(t *testing.T) {
	fn := MathBuiltins["math.min"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(3)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 3 {
		t.Errorf("expected 3, got %v", result)
	}
}

func TestMathMaxIntegers(t *testing.T) {
	fn := MathBuiltins["math.max"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(5)}, &object.Integer{Value: big.NewInt(3)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 5 {
		t.Errorf("expected 5, got %v", result)
	}
}

func TestMathClampWithin(t *testing.T) {
	fn := MathBuiltins["math.clamp"]
	result := fn.Fn(
		&object.Integer{Value: big.NewInt(5)},
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(10)},
	)
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 5 {
		t.Errorf("expected 5, got %v", result)
	}
}

func TestMathClampBelow(t *testing.T) {
	fn := MathBuiltins["math.clamp"]
	result := fn.Fn(
		&object.Integer{Value: big.NewInt(-5)},
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(10)},
	)
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 1 {
		t.Errorf("expected 1, got %v", result)
	}
}

func TestMathClampAbove(t *testing.T) {
	fn := MathBuiltins["math.clamp"]
	result := fn.Fn(
		&object.Integer{Value: big.NewInt(100)},
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(10)},
	)
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 10 {
		t.Errorf("expected 10, got %v", result)
	}
}

// ============================================================================
// pow, sqrt, cbrt
// ============================================================================

func TestMathPowIntegers(t *testing.T) {
	fn := MathBuiltins["math.pow"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(2)}, &object.Integer{Value: big.NewInt(10)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 1024 {
		t.Errorf("expected 1024, got %v", result)
	}
}

func TestMathPowFloatsUnit(t *testing.T) {
	fn := MathBuiltins["math.pow"]
	result := fn.Fn(&object.Float{Value: 2.0}, &object.Float{Value: 0.5})
	floatVal, ok := result.(*object.Float)
	if !ok || math.Abs(floatVal.Value-math.Sqrt(2)) > 0.0001 {
		t.Errorf("expected sqrt(2), got %v", result)
	}
}

func TestMathSqrtUnit(t *testing.T) {
	fn := MathBuiltins["math.sqrt"]
	result := fn.Fn(&object.Float{Value: 16.0})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 4.0 {
		t.Errorf("expected 4.0, got %v", result)
	}
}

func TestMathSqrtNegativeUnit(t *testing.T) {
	fn := MathBuiltins["math.sqrt"]
	result := fn.Fn(&object.Float{Value: -1.0})
	// Implementation may return NaN or an error for sqrt of negative
	switch v := result.(type) {
	case *object.Float:
		if !math.IsNaN(v.Value) {
			t.Errorf("expected NaN for sqrt(-1), got %v", result)
		}
	case *object.Error:
		// Error is also acceptable
	default:
		t.Errorf("expected NaN or error for sqrt(-1), got %T", result)
	}
}

func TestMathCbrt(t *testing.T) {
	fn := MathBuiltins["math.cbrt"]
	result := fn.Fn(&object.Float{Value: 27.0})
	floatVal, ok := result.(*object.Float)
	if !ok || math.Abs(floatVal.Value-3.0) > 0.0001 {
		t.Errorf("expected 3.0, got %v", result)
	}
}

// ============================================================================
// floor, ceil, round, trunc
// ============================================================================

func TestMathFloor(t *testing.T) {
	fn := MathBuiltins["math.floor"]
	result := fn.Fn(&object.Float{Value: 3.7})
	switch v := result.(type) {
	case *object.Float:
		if v.Value != 3.0 {
			t.Errorf("expected 3.0, got %v", result)
		}
	case *object.Integer:
		if v.Value.Int64() != 3 {
			t.Errorf("expected 3, got %v", result)
		}
	default:
		t.Errorf("expected Float or Integer, got %T", result)
	}
}

func TestMathCeil(t *testing.T) {
	fn := MathBuiltins["math.ceil"]
	result := fn.Fn(&object.Float{Value: 3.2})
	switch v := result.(type) {
	case *object.Float:
		if v.Value != 4.0 {
			t.Errorf("expected 4.0, got %v", result)
		}
	case *object.Integer:
		if v.Value.Int64() != 4 {
			t.Errorf("expected 4, got %v", result)
		}
	default:
		t.Errorf("expected Float or Integer, got %T", result)
	}
}

func TestMathRoundUnit(t *testing.T) {
	fn := MathBuiltins["math.round"]
	tests := []struct {
		input    float64
		expected int64
	}{
		{3.4, 3},
		{3.5, 4},
		{3.6, 4},
		{-3.5, -4},
	}
	for _, tt := range tests {
		result := fn.Fn(&object.Float{Value: tt.input})
		switch v := result.(type) {
		case *object.Float:
			if int64(v.Value) != tt.expected {
				t.Errorf("round(%v): expected %v, got %v", tt.input, tt.expected, result)
			}
		case *object.Integer:
			if v.Value.Int64() != tt.expected {
				t.Errorf("round(%v): expected %v, got %v", tt.input, tt.expected, result)
			}
		default:
			t.Errorf("round(%v): expected numeric, got %T", tt.input, result)
		}
	}
}

func TestMathTrunc(t *testing.T) {
	fn := MathBuiltins["math.trunc"]
	tests := []struct {
		input    float64
		expected int64
	}{
		{3.7, 3},
		{-3.7, -3},
	}
	for _, tt := range tests {
		result := fn.Fn(&object.Float{Value: tt.input})
		switch v := result.(type) {
		case *object.Float:
			if int64(v.Value) != tt.expected {
				t.Errorf("trunc(%v): expected %v, got %v", tt.input, tt.expected, result)
			}
		case *object.Integer:
			if v.Value.Int64() != tt.expected {
				t.Errorf("trunc(%v): expected %v, got %v", tt.input, tt.expected, result)
			}
		default:
			t.Errorf("trunc(%v): expected numeric, got %T", tt.input, result)
		}
	}
}

// ============================================================================
// Trigonometry: sin, cos, tan, asin, acos, atan, atan2
// ============================================================================

func TestMathSinZero(t *testing.T) {
	fn := MathBuiltins["math.sin"]
	result := fn.Fn(&object.Float{Value: 0.0})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 0.0 {
		t.Errorf("expected 0.0, got %v", result)
	}
}

func TestMathCosZero(t *testing.T) {
	fn := MathBuiltins["math.cos"]
	result := fn.Fn(&object.Float{Value: 0.0})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 1.0 {
		t.Errorf("expected 1.0, got %v", result)
	}
}

func TestMathTanZero(t *testing.T) {
	fn := MathBuiltins["math.tan"]
	result := fn.Fn(&object.Float{Value: 0.0})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 0.0 {
		t.Errorf("expected 0.0, got %v", result)
	}
}

func TestMathAtan2(t *testing.T) {
	fn := MathBuiltins["math.atan2"]
	result := fn.Fn(&object.Float{Value: 1.0}, &object.Float{Value: 1.0})
	floatVal, ok := result.(*object.Float)
	if !ok || math.Abs(floatVal.Value-math.Pi/4) > 0.0001 {
		t.Errorf("expected pi/4, got %v", result)
	}
}

// ============================================================================
// Hyperbolic: sinh, cosh, tanh
// ============================================================================

func TestMathSinhZero(t *testing.T) {
	fn := MathBuiltins["math.sinh"]
	result := fn.Fn(&object.Float{Value: 0.0})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 0.0 {
		t.Errorf("expected 0.0, got %v", result)
	}
}

func TestMathCoshZero(t *testing.T) {
	fn := MathBuiltins["math.cosh"]
	result := fn.Fn(&object.Float{Value: 0.0})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 1.0 {
		t.Errorf("expected 1.0, got %v", result)
	}
}

func TestMathTanhZero(t *testing.T) {
	fn := MathBuiltins["math.tanh"]
	result := fn.Fn(&object.Float{Value: 0.0})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 0.0 {
		t.Errorf("expected 0.0, got %v", result)
	}
}

// ============================================================================
// Logarithms: log, log2, log10, log_base, exp, exp2
// ============================================================================

func TestMathLogE(t *testing.T) {
	fn := MathBuiltins["math.log"]
	result := fn.Fn(&object.Float{Value: math.E})
	floatVal, ok := result.(*object.Float)
	if !ok || math.Abs(floatVal.Value-1.0) > 0.0001 {
		t.Errorf("expected 1.0, got %v", result)
	}
}

func TestMathLog2(t *testing.T) {
	fn := MathBuiltins["math.log2"]
	result := fn.Fn(&object.Float{Value: 8.0})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 3.0 {
		t.Errorf("expected 3.0, got %v", result)
	}
}

func TestMathLog10Unit(t *testing.T) {
	fn := MathBuiltins["math.log10"]
	result := fn.Fn(&object.Float{Value: 1000.0})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 3.0 {
		t.Errorf("expected 3.0, got %v", result)
	}
}

func TestMathExpUnit(t *testing.T) {
	fn := MathBuiltins["math.exp"]
	result := fn.Fn(&object.Float{Value: 0.0})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 1.0 {
		t.Errorf("expected 1.0, got %v", result)
	}
}

func TestMathExp2(t *testing.T) {
	fn := MathBuiltins["math.exp2"]
	result := fn.Fn(&object.Float{Value: 3.0})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 8.0 {
		t.Errorf("expected 8.0, got %v", result)
	}
}

// ============================================================================
// Number theory: is_prime, is_even, is_odd, gcd, lcm, factorial
// ============================================================================

func TestMathIsPrimeTrue(t *testing.T) {
	fn := MathBuiltins["math.is_prime"]
	primes := []int64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
	for _, p := range primes {
		result := fn.Fn(&object.Integer{Value: big.NewInt(p)})
		if result != object.TRUE {
			t.Errorf("expected %d to be prime", p)
		}
	}
}

func TestMathIsPrimeFalse(t *testing.T) {
	fn := MathBuiltins["math.is_prime"]
	nonPrimes := []int64{0, 1, 4, 6, 8, 9, 10, 12, 15}
	for _, n := range nonPrimes {
		result := fn.Fn(&object.Integer{Value: big.NewInt(n)})
		if result != object.FALSE {
			t.Errorf("expected %d to not be prime", n)
		}
	}
}

func TestMathIsEvenUnit(t *testing.T) {
	fn := MathBuiltins["math.is_even"]
	if fn.Fn(&object.Integer{Value: big.NewInt(4)}) != object.TRUE {
		t.Error("expected 4 to be even")
	}
	if fn.Fn(&object.Integer{Value: big.NewInt(5)}) != object.FALSE {
		t.Error("expected 5 to not be even")
	}
}

func TestMathIsOddUnit(t *testing.T) {
	fn := MathBuiltins["math.is_odd"]
	if fn.Fn(&object.Integer{Value: big.NewInt(5)}) != object.TRUE {
		t.Error("expected 5 to be odd")
	}
	if fn.Fn(&object.Integer{Value: big.NewInt(4)}) != object.FALSE {
		t.Error("expected 4 to not be odd")
	}
}

func TestMathGCDUnit(t *testing.T) {
	fn := MathBuiltins["math.gcd"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(48)}, &object.Integer{Value: big.NewInt(18)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 6 {
		t.Errorf("expected 6, got %v", result)
	}
}

func TestMathLCMUnit(t *testing.T) {
	fn := MathBuiltins["math.lcm"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(4)}, &object.Integer{Value: big.NewInt(6)})
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 12 {
		t.Errorf("expected 12, got %v", result)
	}
}

func TestMathFactorialUnit(t *testing.T) {
	fn := MathBuiltins["math.factorial"]
	tests := []struct {
		n        int64
		expected int64
	}{
		{0, 1},
		{1, 1},
		{5, 120},
		{10, 3628800},
	}
	for _, tt := range tests {
		result := fn.Fn(&object.Integer{Value: big.NewInt(tt.n)})
		intVal, ok := result.(*object.Integer)
		if !ok || intVal.Value.Int64() != tt.expected {
			t.Errorf("factorial(%d): expected %d, got %v", tt.n, tt.expected, result)
		}
	}
}

// ============================================================================
// Utility: sum, avg, hypot, distance, lerp, map_range
// ============================================================================

func TestMathSum(t *testing.T) {
	fn := MathBuiltins["math.sum"]
	// math.sum takes multiple number arguments, not an array
	result := fn.Fn(
		&object.Integer{Value: big.NewInt(1)},
		&object.Integer{Value: big.NewInt(2)},
		&object.Integer{Value: big.NewInt(3)},
		&object.Integer{Value: big.NewInt(4)},
	)
	intVal, ok := result.(*object.Integer)
	if !ok || intVal.Value.Int64() != 10 {
		t.Errorf("expected 10, got %v", result)
	}
}

func TestMathAvg(t *testing.T) {
	fn := MathBuiltins["math.avg"]
	// math.avg takes multiple number arguments, not an array
	result := fn.Fn(
		&object.Integer{Value: big.NewInt(10)},
		&object.Integer{Value: big.NewInt(20)},
		&object.Integer{Value: big.NewInt(30)},
	)
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 20.0 {
		t.Errorf("expected 20.0, got %v", result)
	}
}

func TestMathHypot(t *testing.T) {
	fn := MathBuiltins["math.hypot"]
	result := fn.Fn(&object.Float{Value: 3.0}, &object.Float{Value: 4.0})
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 5.0 {
		t.Errorf("expected 5.0, got %v", result)
	}
}

func TestMathDistance(t *testing.T) {
	fn := MathBuiltins["math.distance"]
	result := fn.Fn(
		&object.Float{Value: 0.0}, &object.Float{Value: 0.0},
		&object.Float{Value: 3.0}, &object.Float{Value: 4.0},
	)
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 5.0 {
		t.Errorf("expected 5.0, got %v", result)
	}
}

func TestMathLerp(t *testing.T) {
	fn := MathBuiltins["math.lerp"]
	result := fn.Fn(
		&object.Float{Value: 0.0},
		&object.Float{Value: 10.0},
		&object.Float{Value: 0.5},
	)
	floatVal, ok := result.(*object.Float)
	if !ok || floatVal.Value != 5.0 {
		t.Errorf("expected 5.0, got %v", result)
	}
}

// ============================================================================
// Angle conversion: deg_to_rad, rad_to_deg
// ============================================================================

func TestMathDegToRad(t *testing.T) {
	fn := MathBuiltins["math.deg_to_rad"]
	result := fn.Fn(&object.Float{Value: 180.0})
	floatVal, ok := result.(*object.Float)
	if !ok || math.Abs(floatVal.Value-math.Pi) > 0.0001 {
		t.Errorf("expected pi, got %v", result)
	}
}

func TestMathRadToDeg(t *testing.T) {
	fn := MathBuiltins["math.rad_to_deg"]
	result := fn.Fn(&object.Float{Value: math.Pi})
	floatVal, ok := result.(*object.Float)
	if !ok || math.Abs(floatVal.Value-180.0) > 0.0001 {
		t.Errorf("expected 180, got %v", result)
	}
}

// ============================================================================
// Special value checks: is_nan, is_inf, is_finite
// ============================================================================

func TestMathIsNaN(t *testing.T) {
	fn := MathBuiltins["math.is_nan"]
	if fn.Fn(&object.Float{Value: math.NaN()}) != object.TRUE {
		t.Error("expected TRUE for NaN")
	}
	if fn.Fn(&object.Float{Value: 0.0}) != object.FALSE {
		t.Error("expected FALSE for 0.0")
	}
}

func TestMathIsInf(t *testing.T) {
	fn := MathBuiltins["math.is_inf"]
	if fn.Fn(&object.Float{Value: math.Inf(1)}) != object.TRUE {
		t.Error("expected TRUE for +Inf")
	}
	if fn.Fn(&object.Float{Value: math.Inf(-1)}) != object.TRUE {
		t.Error("expected TRUE for -Inf")
	}
	if fn.Fn(&object.Float{Value: 0.0}) != object.FALSE {
		t.Error("expected FALSE for 0.0")
	}
}

func TestMathIsFinite(t *testing.T) {
	fn := MathBuiltins["math.is_finite"]
	if fn.Fn(&object.Float{Value: 42.0}) != object.TRUE {
		t.Error("expected TRUE for 42.0")
	}
	if fn.Fn(&object.Float{Value: math.Inf(1)}) != object.FALSE {
		t.Error("expected FALSE for Inf")
	}
	if fn.Fn(&object.Float{Value: math.NaN()}) != object.FALSE {
		t.Error("expected FALSE for NaN")
	}
}

// ============================================================================
// Constants
// ============================================================================

func TestMathConstantPI(t *testing.T) {
	fn := MathBuiltins["math.PI"]
	result := fn.Fn()
	floatVal, ok := result.(*object.Float)
	if !ok || math.Abs(floatVal.Value-math.Pi) > 0.0001 {
		t.Errorf("expected pi, got %v", result)
	}
}

func TestMathConstantE(t *testing.T) {
	fn := MathBuiltins["math.E"]
	result := fn.Fn()
	floatVal, ok := result.(*object.Float)
	if !ok || math.Abs(floatVal.Value-math.E) > 0.0001 {
		t.Errorf("expected e, got %v", result)
	}
}

func TestMathConstantTAU(t *testing.T) {
	fn := MathBuiltins["math.TAU"]
	result := fn.Fn()
	floatVal, ok := result.(*object.Float)
	if !ok || math.Abs(floatVal.Value-2*math.Pi) > 0.0001 {
		t.Errorf("expected 2*pi, got %v", result)
	}
}

func TestMathConstantPHI(t *testing.T) {
	fn := MathBuiltins["math.PHI"]
	result := fn.Fn()
	floatVal, ok := result.(*object.Float)
	phi := (1 + math.Sqrt(5)) / 2
	if !ok || math.Abs(floatVal.Value-phi) > 0.0001 {
		t.Errorf("expected phi, got %v", result)
	}
}
