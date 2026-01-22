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
	// Basic arithmetic
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

	// Absolute and sign
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

	// Min/Max
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

	// Rounding
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

	// Powers and roots
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

	// Logarithms
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

	// Trigonometry
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

	// Hyperbolic
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

	// Angle conversion
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

	// Random
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

	// Constants
	"math.PI": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Pi}
		},
		IsConstant: true,
	},
	"math.E": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.E}
		},
		IsConstant: true,
	},
	"math.PHI": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Phi}
		},
		IsConstant: true,
	},
	"math.SQRT2": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Sqrt2}
		},
		IsConstant: true,
	},
	"math.LN2": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Ln2}
		},
		IsConstant: true,
	},
	"math.LN10": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Ln10}
		},
		IsConstant: true,
	},
	"math.TAU": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: 2 * math.Pi}
		},
		IsConstant: true,
	},
	"math.INF": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Inf(1)}
		},
		IsConstant: true,
	},
	"math.NEG_INF": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Inf(-1)}
		},
		IsConstant: true,
	},

	// Special value checks
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

	// Additional useful functions
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
	"math.factorial": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "math.factorial() takes exactly 1 argument"}
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			n := int64(val)
			if n < 0 {
				return &object.Error{Code: "E8004", Message: "math.factorial() requires non-negative integer"}
			}
			if n > 20 {
				return &object.Error{Code: "E8005", Message: "math.factorial() overflow for n > 20"}
			}
			result := int64(1)
			for i := int64(2); i <= n; i++ {
				result *= i
			}
			return &object.Integer{Value: big.NewInt(result)}
		},
	},
	"math.gcd": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.gcd() takes exactly 2 arguments"}
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
	"math.lcm": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "math.lcm() takes exactly 2 arguments"}
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
		return 0, &object.Error{Code: "E7005", Message: fmt.Sprintf("expected number, got %s", getEZTypeName(obj))}
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
