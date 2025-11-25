package interpreter
// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"math"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var mathBuiltins = map[string]*Builtin{
	// Basic arithmetic
	"math.add": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("math.add() takes exactly 2 arguments")
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			if isFloat(args[0]) || isFloat(args[1]) {
				return &Float{Value: a + b}
			}
			return &Integer{Value: int64(a + b)}
		},
	},
	"math.sub": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("math.sub() takes exactly 2 arguments")
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			if isFloat(args[0]) || isFloat(args[1]) {
				return &Float{Value: a - b}
			}
			return &Integer{Value: int64(a - b)}
		},
	},
	"math.mul": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("math.mul() takes exactly 2 arguments")
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			if isFloat(args[0]) || isFloat(args[1]) {
				return &Float{Value: a * b}
			}
			return &Integer{Value: int64(a * b)}
		},
	},
	"math.div": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("math.div() takes exactly 2 arguments")
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			if b == 0 {
				return newError("division by zero")
			}
			return &Float{Value: a / b}
		},
	},
	"math.mod": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("math.mod() takes exactly 2 arguments")
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			if b == 0 {
				return newError("modulo by zero")
			}
			return &Float{Value: math.Mod(a, b)}
		},
	},

	// Absolute and sign
	"math.abs": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.abs() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if isFloat(args[0]) {
				return &Float{Value: math.Abs(val)}
			}
			if val < 0 {
				return &Integer{Value: int64(-val)}
			}
			return &Integer{Value: int64(val)}
		},
	},
	"math.sign": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.sign() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val > 0 {
				return &Integer{Value: 1}
			} else if val < 0 {
				return &Integer{Value: -1}
			}
			return &Integer{Value: 0}
		},
	},
	"math.neg": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.neg() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if isFloat(args[0]) {
				return &Float{Value: -val}
			}
			return &Integer{Value: int64(-val)}
		},
	},

	// Min/Max
	"math.min": {
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return newError("math.min() takes at least 2 arguments")
			}
			minVal, err := getNumber(args[0])
			if err != nil {
				return err
			}
			hasFloat := isFloat(args[0])
			for i := 1; i < len(args); i++ {
				val, err := getNumber(args[i])
				if err != nil {
					return err
				}
				if isFloat(args[i]) {
					hasFloat = true
				}
				if val < minVal {
					minVal = val
				}
			}
			if hasFloat {
				return &Float{Value: minVal}
			}
			return &Integer{Value: int64(minVal)}
		},
	},
	"math.max": {
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return newError("math.max() takes at least 2 arguments")
			}
			maxVal, err := getNumber(args[0])
			if err != nil {
				return err
			}
			hasFloat := isFloat(args[0])
			for i := 1; i < len(args); i++ {
				val, err := getNumber(args[i])
				if err != nil {
					return err
				}
				if isFloat(args[i]) {
					hasFloat = true
				}
				if val > maxVal {
					maxVal = val
				}
			}
			if hasFloat {
				return &Float{Value: maxVal}
			}
			return &Integer{Value: int64(maxVal)}
		},
	},
	"math.clamp": {
		Fn: func(args ...Object) Object {
			if len(args) != 3 {
				return newError("math.clamp() takes exactly 3 arguments (value, min, max)")
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
			if isFloat(args[0]) || isFloat(args[1]) || isFloat(args[2]) {
				return &Float{Value: result}
			}
			return &Integer{Value: int64(result)}
		},
	},

	// Rounding
	"math.floor": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.floor() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Integer{Value: int64(math.Floor(val))}
		},
	},
	"math.ceil": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.ceil() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Integer{Value: int64(math.Ceil(val))}
		},
	},
	"math.round": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.round() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Integer{Value: int64(math.Round(val))}
		},
	},
	"math.trunc": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.trunc() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Integer{Value: int64(math.Trunc(val))}
		},
	},

	// Powers and roots
	"math.pow": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("math.pow() takes exactly 2 arguments")
			}
			base, exp, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			result := math.Pow(base, exp)
			if isFloat(args[0]) || isFloat(args[1]) || exp < 0 {
				return &Float{Value: result}
			}
			return &Integer{Value: int64(result)}
		},
	},
	"math.sqrt": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.sqrt() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val < 0 {
				return newError("math.sqrt() cannot take negative number")
			}
			return &Float{Value: math.Sqrt(val)}
		},
	},
	"math.cbrt": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.cbrt() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Float{Value: math.Cbrt(val)}
		},
	},
	"math.hypot": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("math.hypot() takes exactly 2 arguments")
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			return &Float{Value: math.Hypot(a, b)}
		},
	},
	"math.exp": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.exp() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Float{Value: math.Exp(val)}
		},
	},
	"math.exp2": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.exp2() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Float{Value: math.Exp2(val)}
		},
	},

	// Logarithms
	"math.log": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.log() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val <= 0 {
				return newError("math.log() requires positive number")
			}
			return &Float{Value: math.Log(val)}
		},
	},
	"math.log2": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.log2() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val <= 0 {
				return newError("math.log2() requires positive number")
			}
			return &Float{Value: math.Log2(val)}
		},
	},
	"math.log10": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.log10() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val <= 0 {
				return newError("math.log10() requires positive number")
			}
			return &Float{Value: math.Log10(val)}
		},
	},

	// Trigonometry
	"math.sin": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.sin() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Float{Value: math.Sin(val)}
		},
	},
	"math.cos": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.cos() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Float{Value: math.Cos(val)}
		},
	},
	"math.tan": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.tan() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Float{Value: math.Tan(val)}
		},
	},
	"math.asin": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.asin() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val < -1 || val > 1 {
				return newError("math.asin() requires value between -1 and 1")
			}
			return &Float{Value: math.Asin(val)}
		},
	},
	"math.acos": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.acos() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if val < -1 || val > 1 {
				return newError("math.acos() requires value between -1 and 1")
			}
			return &Float{Value: math.Acos(val)}
		},
	},
	"math.atan": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.atan() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Float{Value: math.Atan(val)}
		},
	},
	"math.atan2": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("math.atan2() takes exactly 2 arguments")
			}
			y, x, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			return &Float{Value: math.Atan2(y, x)}
		},
	},

	// Hyperbolic
	"math.sinh": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.sinh() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Float{Value: math.Sinh(val)}
		},
	},
	"math.cosh": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.cosh() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Float{Value: math.Cosh(val)}
		},
	},
	"math.tanh": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.tanh() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Float{Value: math.Tanh(val)}
		},
	},

	// Angle conversion
	"math.deg_to_rad": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.deg_to_rad() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Float{Value: val * math.Pi / 180}
		},
	},
	"math.rad_to_deg": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.rad_to_deg() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			return &Float{Value: val * 180 / math.Pi}
		},
	},

	// Random
	"math.random": {
		Fn: func(args ...Object) Object {
			if len(args) == 0 {
				// Return float between 0 and 1
				return &Float{Value: rand.Float64()}
			} else if len(args) == 1 {
				// Return int between 0 and max-1
				max, err := getNumber(args[0])
				if err != nil {
					return err
				}
				if max <= 0 {
					return newError("math.random() max must be positive")
				}
				return &Integer{Value: int64(rand.Intn(int(max)))}
			} else if len(args) == 2 {
				// Return int between min and max
				min, max, err := getTwoNumbers(args)
				if err != nil {
					return err
				}
				if max <= min {
					return newError("math.random() max must be greater than min")
				}
				return &Integer{Value: int64(min) + int64(rand.Intn(int(max-min)))}
			}
			return newError("math.random() takes 0, 1, or 2 arguments")
		},
	},
	"math.random_float": {
		Fn: func(args ...Object) Object {
			if len(args) == 0 {
				return &Float{Value: rand.Float64()}
			} else if len(args) == 2 {
				min, max, err := getTwoNumbers(args)
				if err != nil {
					return err
				}
				return &Float{Value: min + rand.Float64()*(max-min)}
			}
			return newError("math.random_float() takes 0 or 2 arguments")
		},
	},

	// Constants
	"math.pi": {
		Fn: func(args ...Object) Object {
			return &Float{Value: math.Pi}
		},
	},
	"math.e": {
		Fn: func(args ...Object) Object {
			return &Float{Value: math.E}
		},
	},
	"math.phi": {
		Fn: func(args ...Object) Object {
			return &Float{Value: math.Phi}
		},
	},
	"math.sqrt2": {
		Fn: func(args ...Object) Object {
			return &Float{Value: math.Sqrt2}
		},
	},
	"math.ln2": {
		Fn: func(args ...Object) Object {
			return &Float{Value: math.Ln2}
		},
	},
	"math.ln10": {
		Fn: func(args ...Object) Object {
			return &Float{Value: math.Ln10}
		},
	},

	// Additional useful functions
	"math.sum": {
		Fn: func(args ...Object) Object {
			if len(args) == 0 {
				return &Integer{Value: 0}
			}
			var sum float64
			hasFloat := false
			for _, arg := range args {
				val, err := getNumber(arg)
				if err != nil {
					return err
				}
				if isFloat(arg) {
					hasFloat = true
				}
				sum += val
			}
			if hasFloat {
				return &Float{Value: sum}
			}
			return &Integer{Value: int64(sum)}
		},
	},
	"math.avg": {
		Fn: func(args ...Object) Object {
			if len(args) == 0 {
				return newError("math.avg() requires at least 1 argument")
			}
			var sum float64
			for _, arg := range args {
				val, err := getNumber(arg)
				if err != nil {
					return err
				}
				sum += val
			}
			return &Float{Value: sum / float64(len(args))}
		},
	},
	"math.factorial": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.factorial() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			n := int64(val)
			if n < 0 {
				return newError("math.factorial() requires non-negative integer")
			}
			if n > 20 {
				return newError("math.factorial() overflow for n > 20")
			}
			result := int64(1)
			for i := int64(2); i <= n; i++ {
				result *= i
			}
			return &Integer{Value: result}
		},
	},
	"math.gcd": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("math.gcd() takes exactly 2 arguments")
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			ai, bi := int64(math.Abs(a)), int64(math.Abs(b))
			for bi != 0 {
				ai, bi = bi, ai%bi
			}
			return &Integer{Value: ai}
		},
	},
	"math.lcm": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("math.lcm() takes exactly 2 arguments")
			}
			a, b, err := getTwoNumbers(args)
			if err != nil {
				return err
			}
			ai, bi := int64(math.Abs(a)), int64(math.Abs(b))
			if ai == 0 || bi == 0 {
				return &Integer{Value: 0}
			}
			// gcd
			ta, tb := ai, bi
			for tb != 0 {
				ta, tb = tb, ta%tb
			}
			return &Integer{Value: (ai * bi) / ta}
		},
	},
	"math.is_prime": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.is_prime() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			n := int64(val)
			if n < 2 {
				return FALSE
			}
			if n == 2 {
				return TRUE
			}
			if n%2 == 0 {
				return FALSE
			}
			for i := int64(3); i*i <= n; i += 2 {
				if n%i == 0 {
					return FALSE
				}
			}
			return TRUE
		},
	},
	"math.is_even": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.is_even() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if int64(val)%2 == 0 {
				return TRUE
			}
			return FALSE
		},
	},
	"math.is_odd": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("math.is_odd() takes exactly 1 argument")
			}
			val, err := getNumber(args[0])
			if err != nil {
				return err
			}
			if int64(val)%2 != 0 {
				return TRUE
			}
			return FALSE
		},
	},
	"math.lerp": {
		Fn: func(args ...Object) Object {
			if len(args) != 3 {
				return newError("math.lerp() takes exactly 3 arguments (a, b, t)")
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
			return &Float{Value: a + t*(b-a)}
		},
	},
	"math.map_range": {
		Fn: func(args ...Object) Object {
			if len(args) != 5 {
				return newError("math.map_range() takes exactly 5 arguments (value, in_min, in_max, out_min, out_max)")
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
			result := (value-inMin)*(outMax-outMin)/(inMax-inMin) + outMin
			return &Float{Value: result}
		},
	},
	"math.distance": {
		Fn: func(args ...Object) Object {
			if len(args) != 4 {
				return newError("math.distance() takes exactly 4 arguments (x1, y1, x2, y2)")
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
			return &Float{Value: math.Sqrt(dx*dx + dy*dy)}
		},
	},
}

// Helper functions
func getNumber(obj Object) (float64, *Error) {
	switch v := obj.(type) {
	case *Integer:
		return float64(v.Value), nil
	case *Float:
		return v.Value, nil
	default:
		return 0, newError("expected number, got %s", obj.Type())
	}
}

func getTwoNumbers(args []Object) (float64, float64, *Error) {
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

func isFloat(obj Object) bool {
	_, ok := obj.(*Float)
	return ok
}

func init() {
	for name, builtin := range mathBuiltins {
		builtins[name] = builtin
	}
}
