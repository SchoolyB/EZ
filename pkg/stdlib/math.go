package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"

	"github.com/marshallburns/ez/pkg/object"
)

// MathBuiltins contains the math module functions
var MathBuiltins = map[string]*object.Builtin{
	// ============================================================================
	// Basic Arithmetic
	// add, sub, mul, div, mod perform basic math operations.
	// ============================================================================

	// add adds two numbers together.
	// Takes two numbers. Returns sum (int if both int, else float).
	"math.add": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.add() takes exactly 2 arguments"}
			}
			// Use big.Int arithmetic for integer-only operations to preserve precision
			if intA, intB := getTwoIntegers(args); intA != nil && intB != nil {
				return &object.Integer{Value: new(big.Int).Add(intA, intB)}
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			return &object.Float{Value: a + b}
		},
	},

	// sub subtracts the second number from the first.
	// Takes two numbers. Returns difference.
	"math.sub": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.sub() takes exactly 2 arguments"}
			}
			// Use big.Int arithmetic for integer-only operations to preserve precision
			if intA, intB := getTwoIntegers(args); intA != nil && intB != nil {
				return &object.Integer{Value: new(big.Int).Sub(intA, intB)}
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			return &object.Float{Value: a - b}
		},
	},

	// mul multiplies two numbers.
	// Takes two numbers. Returns product.
	"math.mul": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.mul() takes exactly 2 arguments"}
			}
			// Use big.Int arithmetic for integer-only operations to preserve precision
			if intA, intB := getTwoIntegers(args); intA != nil && intB != nil {
				return &object.Integer{Value: new(big.Int).Mul(intA, intB)}
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			return &object.Float{Value: a * b}
		},
	},

	// div divides the first number by the second.
	// Takes two numbers. Returns quotient as float.
	"math.div": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.div() takes exactly 2 arguments"}
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			if b == 0 {
				return &object.Error{Code: "E5001", Message: "division by zero"}
			}
			return &object.Float{Value: a / b}
		},
	},

	// mod returns the remainder of division.
	// Takes two numbers. Returns modulo.
	"math.mod": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.mod() takes exactly 2 arguments"}
			}
			// Use big.Int arithmetic for integer-only operations to preserve precision
			if intA, intB := getTwoIntegers(args); intA != nil && intB != nil {
				if intB.Sign() == 0 {
					return &object.Error{Code: "E5002", Message: "modulo by zero"}
				}
				return &object.Integer{Value: new(big.Int).Mod(intA, intB)}
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			if b == 0 {
				return &object.Error{Code: "E5002", Message: "modulo by zero"}
			}
			return &object.Float{Value: math.Mod(a, b)}
		},
	},

	// ============================================================================
	// Absolute Value and Sign
	// ============================================================================

	// abs returns the absolute value of a number.
	// Takes one number. Returns non-negative value.
	"math.abs": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.abs() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if isFloat(args[0]) {
				return &object.Float{Value: math.Abs(val)}
			}
			// For big.Int, we can always compute absolute value
			intVal := args[0].(*object.Integer).Value
			result := new(big.Int).Abs(intVal)
			return &object.Integer{Value: result}
		},
	},

	// sign returns the sign of a number: -1, 0, or 1.
	// Takes one number. Returns int.
	"math.sign": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.sign() takes exactly 1 argument"}
			}
			// Use big.Int.Sign() directly for integers to avoid precision loss
			if intVal := getInteger(args[0]); intVal != nil {
				return &object.Integer{Value: big.NewInt(int64(intVal.Sign()))}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val > 0 {
				return &object.Integer{Value: big.NewInt(1)}
			} else if val < 0 {
				return &object.Integer{Value: big.NewInt(-1)}
			}
			return &object.Integer{Value: big.NewInt(0)}
		},
	},

	// neg returns the negation of a number.
	// Takes one number. Returns negated value.
	"math.neg": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.neg() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if isFloat(args[0]) {
				return &object.Float{Value: -val}
			}
			// For big.Int, we can always negate
			intVal := args[0].(*object.Integer).Value
			result := new(big.Int).Neg(intVal)
			return &object.Integer{Value: result}
		},
	},

	// ============================================================================
	// Min/Max/Clamp
	// ============================================================================

	// min returns the smallest of the given numbers.
	// Takes two or more numbers. Returns minimum value.
	"math.min": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return &object.Error{Code: "E7001", Message: "math.min() takes at least 2 arguments"}
			}
			// Check if all arguments are integers for precision-preserving arithmetic
			allInts := true
			for _, arg := range args {
				if !isInteger(arg) {
					allInts = false
					break
				}
			}
			if allInts {
				minVal := args[0].(*object.Integer).Value
				for i := 1; i < len(args); i++ {
					val := args[i].(*object.Integer).Value
					if val.Cmp(minVal) < 0 {
						minVal = val
					}
				}
				return &object.Integer{Value: new(big.Int).Set(minVal)}
			}
			// Fall back to float64 if any floats
			minVal, err := getNumber(args[0])
			if err != nil {
				return err
			}
			for i := 1; i < len(args); i++ {
				val, err := getNumber(args[i])
				if err != nil {
					return err
				}
				if val < minVal {
					minVal = val
				}
			}
			return &object.Float{Value: minVal}
		},
	},

	// max returns the largest of the given numbers.
	// Takes two or more numbers. Returns maximum value.
	"math.max": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return &object.Error{Code: "E7001", Message: "math.max() takes at least 2 arguments"}
			}
			// Check if all arguments are integers for precision-preserving arithmetic
			allInts := true
			for _, arg := range args {
				if !isInteger(arg) {
					allInts = false
					break
				}
			}
			if allInts {
				maxVal := args[0].(*object.Integer).Value
				for i := 1; i < len(args); i++ {
					val := args[i].(*object.Integer).Value
					if val.Cmp(maxVal) > 0 {
						maxVal = val
					}
				}
				return &object.Integer{Value: new(big.Int).Set(maxVal)}
			}
			// Fall back to float64 if any floats
			maxVal, err := getNumber(args[0])
			if err != nil {
				return err
			}
			for i := 1; i < len(args); i++ {
				val, err := getNumber(args[i])
				if err != nil {
					return err
				}
				if val > maxVal {
					maxVal = val
				}
			}
			return &object.Float{Value: maxVal}
		},
	},

	// clamp constrains a value between min and max bounds.
	// Takes value, min, max. Returns clamped value.
	"math.clamp": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: "math.clamp() takes exactly 3 arguments (value, min, max)"}
			}
			// Use big.Int arithmetic if all arguments are integers
			if isInteger(args[0]) && isInteger(args[1]) && isInteger(args[2]) {
				val := args[0].(*object.Integer).Value
				minVal := args[1].(*object.Integer).Value
				maxVal := args[2].(*object.Integer).Value
				result := new(big.Int).Set(val)
				if result.Cmp(minVal) < 0 {
					result.Set(minVal)
				}
				if result.Cmp(maxVal) > 0 {
					result.Set(maxVal)
				}
				return &object.Integer{Value: result}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			minVal, err := getNumber(args[1])
			if err != nil {
				return err
			}
			maxVal, err := getNumber(args[2])
			if err != nil {
				return err
			}
			result := math.Max(minVal, math.Min(val, maxVal))
			return &object.Float{Value: result}
		},
	},

	// ============================================================================
	// Rounding
	// floor, ceil, round, trunc convert floats to integers.
	// ============================================================================

	// floor rounds down to the nearest integer.
	// Takes one number. Returns int.
	"math.floor": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.floor() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Integer{Value: big.NewInt(int64(math.Floor(val)))}
		},
	},

	// ceil rounds up to the nearest integer.
	// Takes one number. Returns int.
	"math.ceil": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.ceil() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Integer{Value: big.NewInt(int64(math.Ceil(val)))}
		},
	},

	// round rounds to the nearest integer.
	// Takes one number. Returns int.
	"math.round": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.round() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Integer{Value: big.NewInt(int64(math.Round(val)))}
		},
	},

	// trunc truncates toward zero.
	// Takes one number. Returns int.
	"math.trunc": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.trunc() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Integer{Value: big.NewInt(int64(math.Trunc(val)))}
		},
	},

	// ============================================================================
	// Powers and Roots
	// pow, sqrt, cbrt, hypot, exp, exp2 for exponentiation.
	// ============================================================================

	// pow raises base to the power of exponent.
	// Takes base and exponent. Returns result.
	"math.pow": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.pow() takes exactly 2 arguments"}
			}
			// Use big.Int.Exp for integer base and non-negative integer exponent
			if intBase, intExp := getTwoIntegers(args); intBase != nil && intExp != nil {
				if intExp.Sign() >= 0 {
					result := new(big.Int).Exp(intBase, intExp, nil)
					return &object.Integer{Value: result}
				}
				// Negative exponent - fall through to float
			}
			base, exp, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			result := math.Pow(base, exp)
			if math.IsInf(result, 0) || math.IsNaN(result) {
				return &object.Error{Code: "E5004", Message: "math.pow() overflow or invalid result"}
			}
			return &object.Float{Value: result}
		},
	},

	// sqrt returns the square root of a number.
	// Takes one non-negative number. Returns float.
	"math.sqrt": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.sqrt() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val < 0 {
				return &object.Error{Code: "E8001", Message: "math.sqrt() cannot take negative number"}
			}
			return &object.Float{Value: math.Sqrt(val)}
		},
	},

	// cbrt returns the cube root of a number.
	// Takes one number. Returns float.
	"math.cbrt": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.cbrt() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Float{Value: math.Cbrt(val)}
		},
	},

	// hypot returns sqrt(a*a + b*b), the hypotenuse.
	// Takes two numbers. Returns float.
	"math.hypot": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.hypot() takes exactly 2 arguments"}
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			return &object.Float{Value: math.Hypot(a, b)}
		},
	},

	// exp returns e raised to the power x.
	// Takes one number. Returns float.
	"math.exp": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.exp() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Float{Value: math.Exp(val)}
		},
	},

	// exp2 returns 2 raised to the power x.
	// Takes one number. Returns float.
	"math.exp2": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.exp2() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Float{Value: math.Exp2(val)}
		},
	},

	// ============================================================================
	// Logarithms
	// log, log2, log10, log_base for logarithmic functions.
	// ============================================================================

	// log returns the natural logarithm (base e).
	// Takes one positive number. Returns float.
	"math.log": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.log() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val <= 0 {
				return &object.Error{Code: "E8002", Message: "math.log() requires positive number"}
			}
			return &object.Float{Value: math.Log(val)}
		},
	},

	// log2 returns the base-2 logarithm.
	// Takes one positive number. Returns float.
	"math.log2": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.log2() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val <= 0 {
				return &object.Error{Code: "E8002", Message: "math.log2() requires positive number"}
			}
			return &object.Float{Value: math.Log2(val)}
		},
	},

	// log10 returns the base-10 logarithm.
	// Takes one positive number. Returns float.
	"math.log10": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.log10() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val <= 0 {
				return &object.Error{Code: "E8002", Message: "math.log10() requires positive number"}
			}
			return &object.Float{Value: math.Log10(val)}
		},
	},

	// log_base returns the logarithm with arbitrary base.
	// Takes value and base. Returns float.
	"math.log_base": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.log_base() takes exactly 2 arguments (value, base)"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			base, err := getNumber(args[1])
			if err != nil {
				return err
			}
			if val <= 0 {
				return &object.Error{Code: "E8002", Message: "math.log_base() requires positive value"}
			}
			if base <= 0 || base == 1 {
				return &object.Error{Code: "E8002", Message: "math.log_base() requires positive base != 1"}
			}
			// log_base(x, b) = log(x) / log(b)
			return &object.Float{Value: math.Log(val) / math.Log(base)}
		},
	},

	// ============================================================================
	// Trigonometry
	// sin, cos, tan, asin, acos, atan, atan2 (radians).
	// ============================================================================

	// sin returns the sine of an angle in radians.
	// Takes one number. Returns float.
	"math.sin": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.sin() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Float{Value: math.Sin(val)}
		},
	},

	// cos returns the cosine of an angle in radians.
	// Takes one number. Returns float.
	"math.cos": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.cos() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Float{Value: math.Cos(val)}
		},
	},

	// tan returns the tangent of an angle in radians.
	// Takes one number. Returns float.
	"math.tan": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.tan() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Float{Value: math.Tan(val)}
		},
	},

	// asin returns the arcsine in radians.
	// Takes value between -1 and 1. Returns float.
	"math.asin": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.asin() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val < -1 || val > 1 {
				return &object.Error{Code: "E8003", Message: "math.asin() requires value between -1 and 1"}
			}
			return &object.Float{Value: math.Asin(val)}
		},
	},

	// acos returns the arccosine in radians.
	// Takes value between -1 and 1. Returns float.
	"math.acos": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.acos() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val < -1 || val > 1 {
				return &object.Error{Code: "E8003", Message: "math.acos() requires value between -1 and 1"}
			}
			return &object.Float{Value: math.Acos(val)}
		},
	},

	// atan returns the arctangent in radians.
	// Takes one number. Returns float.
	"math.atan": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.atan() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Float{Value: math.Atan(val)}
		},
	},

	// atan2 returns the angle from the x-axis to point (x, y).
	// Takes y and x coordinates. Returns float in radians.
	"math.atan2": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.atan2() takes exactly 2 arguments"}
			}
			y, x, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			return &object.Float{Value: math.Atan2(y, x)}
		},
	},

	// ============================================================================
	// Hyperbolic Functions
	// sinh, cosh, tanh for hyperbolic trigonometry.
	// ============================================================================

	// sinh returns the hyperbolic sine.
	// Takes one number. Returns float.
	"math.sinh": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.sinh() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Float{Value: math.Sinh(val)}
		},
	},

	// cosh returns the hyperbolic cosine.
	// Takes one number. Returns float.
	"math.cosh": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.cosh() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Float{Value: math.Cosh(val)}
		},
	},

	// tanh returns the hyperbolic tangent.
	// Takes one number. Returns float.
	"math.tanh": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.tanh() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Float{Value: math.Tanh(val)}
		},
	},

	// ============================================================================
	// Angle Conversion
	// ============================================================================

	// deg_to_rad converts degrees to radians.
	// Takes degrees. Returns radians as float.
	"math.deg_to_rad": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.deg_to_rad() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Float{Value: val * math.Pi / 180}
		},
	},

	// rad_to_deg converts radians to degrees.
	// Takes radians. Returns degrees as float.
	"math.rad_to_deg": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.rad_to_deg() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &object.Float{Value: val * 180 / math.Pi}
		},
	},

	// ============================================================================
	// Random Number Generation
	// ============================================================================

	// random generates a random number.
	// No args: float [0,1). One arg: int [0,max). Two args: int [min,max).
	"math.random": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) == 0 {
				return &object.Float{Value: rand.Float64()}
			} else if len(args) == 1 {
				max, err := getNumber(args[0])
				if err != nil {
					return err
				}
				if max <= 0 {
					return &object.Error{Code: "E8006", Message: "math.random() max must be positive"}
				}
				return &object.Integer{Value: big.NewInt(int64(rand.Intn(int(max))))}
			} else if len(args) == 2 {
				min, max, err := getTwoNumbers(args)
				if err != nil {
					return err
				}
				if max <= min {
					return &object.Error{Code: "E8006", Message: "math.random() max must be greater than min"}
				}
				return &object.Integer{Value: big.NewInt(int64(min) + int64(rand.Intn(int(max-min))))}
			}
			return &object.Error{Code: "E7001", Message: "math.random() takes 0, 1, or 2 arguments"}
		},
	},

	// random_float generates a random float.
	// No args: float [0,1). Two args: float [min,max).
	"math.random_float": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) == 0 {
				return &object.Float{Value: rand.Float64()}
			} else if len(args) == 2 {
				min, max, err := getTwoNumbers(args)
				if err != nil {
					return err
				}
				return &object.Float{Value: min + rand.Float64()*(max-min)}
			}
			return &object.Error{Code: "E7001", Message: "math.random_float() takes 0 or 2 arguments"}
		},
	},

	// ============================================================================
	// Mathematical Constants
	// PI, E, PHI, SQRT2, LN2, LN10, TAU, INF, NEG_INF, EPSILON, MAX_FLOAT, MIN_FLOAT.
	// ============================================================================

	// PI is the ratio of circle circumference to diameter (~3.14159).
	"math.PI": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Pi}
		},
		IsConstant: true,
	},

	// E is Euler's number, the base of natural logarithms (~2.71828).
	"math.E": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.E}
		},
		IsConstant: true,
	},

	// PHI is the golden ratio (~1.61803).
	"math.PHI": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Phi}
		},
		IsConstant: true,
	},

	// SQRT2 is the square root of 2 (~1.41421).
	"math.SQRT2": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Sqrt2}
		},
		IsConstant: true,
	},

	// LN2 is the natural logarithm of 2 (~0.69315).
	"math.LN2": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Ln2}
		},
		IsConstant: true,
	},

	// LN10 is the natural logarithm of 10 (~2.30259).
	"math.LN10": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Ln10}
		},
		IsConstant: true,
	},

	// TAU is 2*PI, the ratio of circumference to radius (~6.28319).
	"math.TAU": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: 2 * math.Pi}
		},
		IsConstant: true,
	},

	// INF is positive infinity.
	"math.INF": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Inf(1)}
		},
		IsConstant: true,
	},

	// NEG_INF is negative infinity.
	"math.NEG_INF": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Inf(-1)}
		},
		IsConstant: true,
	},

	// EPSILON is the smallest float64 difference from 1.
	"math.EPSILON": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Nextafter(1.0, 2.0) - 1.0}
		},
		IsConstant: true,
	},

	// MAX_FLOAT is the largest representable float64.
	"math.MAX_FLOAT": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.MaxFloat64}
		},
		IsConstant: true,
	},

	// MIN_FLOAT is the smallest positive non-zero float64.
	"math.MIN_FLOAT": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.SmallestNonzeroFloat64}
		},
		IsConstant: true,
	},

	// ============================================================================
	// Special Value Checks
	// ============================================================================

	// is_inf checks if a number is infinite.
	// Takes one number. Returns bool.
	"math.is_inf": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.is_inf() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if math.IsInf(val, 0) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// is_nan checks if a number is NaN (not a number).
	// Takes one number. Returns bool.
	"math.is_nan": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.is_nan() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if math.IsNaN(val) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// is_finite checks if a number is finite (not infinite or NaN).
	// Takes one number. Returns bool.
	"math.is_finite": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.is_finite() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if !math.IsNaN(val) && !math.IsInf(val, 0) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// ============================================================================
	// Utility Functions
	// sum, avg, factorial, gcd, lcm, is_prime, is_even, is_odd, lerp, map_range, distance.
	// ============================================================================

	// sum adds all given numbers together.
	// Takes zero or more numbers. Returns sum.
	"math.sum": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) == 0 {
				return &object.Integer{Value: big.NewInt(0)}
			}
			// Check if all arguments are integers for precision-preserving arithmetic
			allInts := true
			for _, arg := range args {
				if !isInteger(arg) {
					allInts = false
					break
				}
			}
			if allInts {
				sum := big.NewInt(0)
				for _, arg := range args {
					sum.Add(sum, arg.(*object.Integer).Value)
				}
				return &object.Integer{Value: sum}
			}
			// Fall back to float64 if any floats
			var sum float64
			for _, arg := range args {
				val, err := getNumber(arg)
				if err != nil {
					return err
				}
				sum += val
			}
			return &object.Float{Value: sum}
		},
	},

	// avg returns the arithmetic mean of the given numbers.
	// Takes one or more numbers. Returns float.
	"math.avg": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) == 0 {
				return &object.Error{Code: "E7001", Message: "math.avg() requires at least 1 argument"}
			}
			var sum float64
			for _, arg := range args {
				val, err := getNumber(arg)
				if err != nil {
					return err
				}
				sum += val
			}
			return &object.Float{Value: sum / float64(len(args))}
		},
	},

	// factorial computes n! (n factorial).
	// Takes non-negative integer. Returns int.
	"math.factorial": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.factorial() takes exactly 1 argument"}
			}
			// Use big.Int for arbitrary precision factorial
			if intVal := getInteger(args[0]); intVal != nil {
				if intVal.Sign() < 0 {
					return &object.Error{Code: "E8004", Message: "math.factorial() requires non-negative integer"}
				}
				result := big.NewInt(1)
				n := intVal.Int64()
				for i := int64(2); i <= n; i++ {
					result.Mul(result, big.NewInt(i))
				}
				return &object.Integer{Value: result}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			n := int64(val)
			if n < 0 {
				return &object.Error{Code: "E8004", Message: "math.factorial() requires non-negative integer"}
			}
			result := big.NewInt(1)
			for i := int64(2); i <= n; i++ {
				result.Mul(result, big.NewInt(i))
			}
			return &object.Integer{Value: result}
		},
	},

	// gcd returns the greatest common divisor of two numbers.
	// Takes two numbers. Returns int.
	"math.gcd": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.gcd() takes exactly 2 arguments"}
			}
			// Use big.Int.GCD for arbitrary precision
			if intA, intB := getTwoIntegers(args); intA != nil && intB != nil {
				a := new(big.Int).Abs(intA)
				b := new(big.Int).Abs(intB)
				result := new(big.Int).GCD(nil, nil, a, b)
				return &object.Integer{Value: result}
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			ai, bi := int64(math.Abs(a)), int64(math.Abs(b))
			for bi != 0 {
				ai, bi = bi, ai%bi
			}
			return &object.Integer{Value: big.NewInt(ai)}
		},
	},

	// lcm returns the least common multiple of two numbers.
	// Takes two numbers. Returns int.
	"math.lcm": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.lcm() takes exactly 2 arguments"}
			}
			// Use big.Int for arbitrary precision: lcm(a,b) = |a*b| / gcd(a,b)
			if intA, intB := getTwoIntegers(args); intA != nil && intB != nil {
				a := new(big.Int).Abs(intA)
				b := new(big.Int).Abs(intB)
				if a.Sign() == 0 || b.Sign() == 0 {
					return &object.Integer{Value: big.NewInt(0)}
				}
				gcd := new(big.Int).GCD(nil, nil, a, b)
				// lcm = (a * b) / gcd
				result := new(big.Int).Mul(a, b)
				result.Div(result, gcd)
				return &object.Integer{Value: result}
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			ai, bi := int64(math.Abs(a)), int64(math.Abs(b))
			if ai == 0 || bi == 0 {
				return &object.Integer{Value: big.NewInt(0)}
			}
			ta, tb := ai, bi
			for tb != 0 {
				ta, tb = tb, ta%tb
			}
			return &object.Integer{Value: big.NewInt((ai * bi) / ta)}
		},
	},

	// is_prime checks if a number is prime.
	// Takes one number. Returns bool.
	"math.is_prime": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.is_prime() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			n := int64(val)
			if n < 2 {
				return object.FALSE
			}
			if n == 2 {
				return object.TRUE
			}
			if n%2 == 0 {
				return object.FALSE
			}
			for i := int64(3); i*i <= n; i += 2 {
				if n%i == 0 {
					return object.FALSE
				}
			}
			return object.TRUE
		},
	},

	// is_even checks if a number is even.
	// Takes one number. Returns bool.
	"math.is_even": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.is_even() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if int64(val)%2 == 0 {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// is_odd checks if a number is odd.
	// Takes one number. Returns bool.
	"math.is_odd": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.is_odd() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if int64(val)%2 != 0 {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// lerp performs linear interpolation between two values.
	// Takes a, b, and t (0-1). Returns a + t*(b-a).
	"math.lerp": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: "math.lerp() takes exactly 3 arguments (a, b, t)"}
			}
			a, err := getNumber(args[0])
			if err != nil {
				return err
			}
			b, err := getNumber(args[1])
			if err != nil {
				return err
			}
			t, err := getNumber(args[2])
			if err != nil {
				return err
			}
			return &object.Float{Value: a + t*(b-a)}
		},
	},

	// map_range maps a value from one range to another.
	// Takes value, in_min, in_max, out_min, out_max. Returns mapped float.
	"math.map_range": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 5 {
				return &object.Error{Code: "E7001", Message: "math.map_range() takes exactly 5 arguments (value, in_min, in_max, out_min, out_max)"}
			}
			value, err := getNumber(args[0])
			if err != nil {
				return err
			}
			inMin, err := getNumber(args[1])
			if err != nil {
				return err
			}
			inMax, err := getNumber(args[2])
			if err != nil {
				return err
			}
			outMin, err := getNumber(args[3])
			if err != nil {
				return err
			}
			outMax, err := getNumber(args[4])
			if err != nil {
				return err
			}
			if inMax == inMin {
				return &object.Error{Code: "E8007", Message: "math.map_range() requires in_min != in_max (division by zero)"}
			}
			result := (value-inMin)*(outMax-outMin)/(inMax-inMin) + outMin
			return &object.Float{Value: result}
		},
	},

	// distance calculates the Euclidean distance between two 2D points.
	// Takes x1, y1, x2, y2. Returns float.
	"math.distance": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 4 {
				return &object.Error{Code: "E7001", Message: "math.distance() takes exactly 4 arguments (x1, y1, x2, y2)"}
			}
			x1, err := getNumber(args[0])
			if err != nil {
				return err
			}
			y1, err := getNumber(args[1])
			if err != nil {
				return err
			}
			x2, err := getNumber(args[2])
			if err != nil {
				return err
			}
			y2, err := getNumber(args[3])
			if err != nil {
				return err
			}
			dx := x2 - x1
			dy := y2 - y1
			return &object.Float{Value: math.Sqrt(dx*dx + dy*dy)}
		},
	},
}

// Helper functions for math operations
func getNumber(obj object.Object) (float64, *object.Error) {
	switch v := obj.(type) {
	case *object.Integer:
		f, _ := new(big.Float).SetInt(v.Value).Float64()
		return f, nil
	case *object.Float:
		return v.Value, nil
	default:
		return 0, &object.Error{Code: "E7005", Message: fmt.Sprintf("expected number, got %s", object.GetEZTypeName(obj))}
	}
}

func getTwoNumbers(args []object.Object) (float64, float64, *object.Error) {
	a, err := getNumber(args[0])
	if err != nil {
		return 0, 0, err
	}
	b, err := getNumber(args[1])
	if err != nil {
		return 0, 0, err
	}
	return a, b, nil
}

func isFloat(obj object.Object) bool {
	_, ok := obj.(*object.Float)
	return ok
}

func isInteger(obj object.Object) bool {
	_, ok := obj.(*object.Integer)
	return ok
}

// getTwoIntegers returns the big.Int values if both args are integers, nil otherwise
func getTwoIntegers(args []object.Object) (*big.Int, *big.Int) {
	intA, okA := args[0].(*object.Integer)
	intB, okB := args[1].(*object.Integer)
	if okA && okB {
		return intA.Value, intB.Value
	}
	return nil, nil
}

// getInteger returns the big.Int value if the arg is an integer, nil otherwise
func getInteger(obj object.Object) *big.Int {
	if intVal, ok := obj.(*object.Integer); ok {
		return intVal.Value
	}
	return nil
}
