package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"math/big"
	"math/rand"

	"github.com/marshallburns/ez/pkg/object"
)

// RandomBuiltins contains the random module functions
var RandomBuiltins = map[string]*object.Builtin{
	// float generates a random floating-point number.
	// No args: returns [0.0, 1.0). Two args: returns [min, max).
	"random.float": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) == 0 {
				return &object.Float{Value: rand.Float64(), DeclaredType: "float"}
			} else if len(args) == 2 {
				min, err := getRandomNumber(args[0])
				if err != nil {
					return err
				}
				max, err := getRandomNumber(args[1])
				if err != nil {
					return err
				}
				if max <= min {
					return &object.Error{Code: "E8006", Message: "random.float() max must be greater than min"}
				}
				return &object.Float{Value: min + rand.Float64()*(max-min), DeclaredType: "float"}
			}
			return &object.Error{Code: "E7001", Message: "random.float() takes 0 or 2 arguments"}
		},
	},

	// int generates a random integer.
	// One arg: returns [0, max). Two args: returns [min, max).
	"random.int": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) == 1 {
				max, err := getRandomNumber(args[0])
				if err != nil {
					return err
				}
				if max <= 0 {
					return &object.Error{Code: "E8006", Message: "random.int() max must be positive"}
				}
				return &object.Integer{Value: big.NewInt(int64(rand.Intn(int(max))))}
			} else if len(args) == 2 {
				min, err := getRandomNumber(args[0])
				if err != nil {
					return err
				}
				max, err := getRandomNumber(args[1])
				if err != nil {
					return err
				}
				if max <= min {
					return &object.Error{Code: "E8006", Message: "random.int() max must be greater than min"}
				}
				return &object.Integer{Value: big.NewInt(int64(min) + int64(rand.Intn(int(max-min))))}
			}
			return &object.Error{Code: "E7001", Message: "random.int() takes 1 or 2 arguments"}
		},
	},

	// bool returns true or false with 50/50 probability.
	// Takes no arguments. Returns bool.
	"random.bool": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 0 {
				return &object.Error{Code: "E7001", Message: "random.bool() takes no arguments"}
			}
			if rand.Intn(2) == 0 {
				return object.FALSE
			}
			return object.TRUE
		},
	},

	// byte returns a random byte value [0, 255].
	// Takes no arguments. Returns byte.
	"random.byte": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 0 {
				return &object.Error{Code: "E7001", Message: "random.byte() takes no arguments"}
			}
			return &object.Byte{Value: uint8(rand.Intn(256))}
		},
	},

	// char returns a random character.
	// No args: printable ASCII [32,126]. Two args: char in [min,max].
	"random.char": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) == 0 {
				// Printable ASCII range: 32 (space) to 126 (~)
				return &object.Char{Value: rune(32 + rand.Intn(95))}
			} else if len(args) == 2 {
				var minVal, maxVal rune

				// Get min value
				switch v := args[0].(type) {
				case *object.Char:
					minVal = v.Value
				case *object.Integer:
					minVal = rune(v.Value.Int64())
				default:
					return &object.Error{Code: "E7003", Message: "random.char() requires char or integer arguments"}
				}

				// Get max value
				switch v := args[1].(type) {
				case *object.Char:
					maxVal = v.Value
				case *object.Integer:
					maxVal = rune(v.Value.Int64())
				default:
					return &object.Error{Code: "E7003", Message: "random.char() requires char or integer arguments"}
				}

				if maxVal <= minVal {
					return &object.Error{Code: "E8006", Message: "random.char() max must be greater than min"}
				}

				return &object.Char{Value: minVal + rune(rand.Intn(int(maxVal-minVal+1)))}
			}
			return &object.Error{Code: "E7001", Message: "random.char() takes 0 or 2 arguments"}
		},
	},

	// choice returns a random element from an array.
	// Takes an array. Returns random element.
	"random.choice": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "random.choice() takes exactly 1 argument"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "random.choice() requires an array argument"}
			}
			if len(arr.Elements) == 0 {
				return &object.Error{Code: "E9007", Message: "random.choice() cannot select from empty array"}
			}
			return arr.Elements[rand.Intn(len(arr.Elements))]
		},
	},

	// shuffle returns a new array with elements in random order.
	// Takes an array. Returns new shuffled array.
	"random.shuffle": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "random.shuffle() takes exactly 1 argument"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "random.shuffle() requires an array argument"}
			}

			// Create a copy of elements
			newElements := make([]object.Object, len(arr.Elements))
			copy(newElements, arr.Elements)

			// Fisher-Yates shuffle
			for i := len(newElements) - 1; i > 0; i-- {
				j := rand.Intn(i + 1)
				newElements[i], newElements[j] = newElements[j], newElements[i]
			}

			return &object.Array{Elements: newElements, Mutable: true, ElementType: arr.ElementType}
		},
	},

	// sample returns n unique random elements from an array.
	// Takes array and count. Returns new array of sampled elements.
	"random.sample": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "random.sample() takes exactly 2 arguments"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "random.sample() requires an array as first argument"}
			}
			nObj, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "random.sample() requires an integer as second argument"}
			}
			n := int(nObj.Value.Int64())

			if n < 0 {
				return &object.Error{Code: "E10001", Message: "random.sample() count cannot be negative"}
			}
			if n > len(arr.Elements) {
				return &object.Error{Code: "E9008", Message: "random.sample() count exceeds array length"}
			}

			// Create index slice and shuffle it
			indices := make([]int, len(arr.Elements))
			for i := range indices {
				indices[i] = i
			}
			// Partial Fisher-Yates - only need first n elements
			for i := 0; i < n; i++ {
				j := i + rand.Intn(len(indices)-i)
				indices[i], indices[j] = indices[j], indices[i]
			}

			// Select first n elements using shuffled indices
			result := make([]object.Object, n)
			for i := 0; i < n; i++ {
				result[i] = arr.Elements[indices[i]]
			}

			return &object.Array{Elements: result, Mutable: true, ElementType: arr.ElementType}
		},
	},
}

// Helper function for random module
func getRandomNumber(obj object.Object) (float64, *object.Error) {
	switch v := obj.(type) {
	case *object.Integer:
		f, _ := new(big.Float).SetInt(v.Value).Float64()
		return f, nil
	case *object.Float:
		return v.Value, nil
	default:
		return 0, &object.Error{Code: "E7005", Message: "expected number argument"}
	}
}
