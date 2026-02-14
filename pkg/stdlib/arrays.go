package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"math/big"
	"math/rand"
	"sort"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/object"
)

// checkIterating returns an error if the array is being iterated over
func checkIterating(arr *object.Array, funcName string) *object.Error {
	if arr.IsIterating() {
		return &object.Error{
			Code:    "E9006",
			Message: fmt.Sprintf("%s() cannot modify array during for_each iteration", errors.Ident(funcName)),
		}
	}
	return nil
}

// ArraysBuiltins contains the arrays module functions
var ArraysBuiltins = map[string]*object.Builtin{
	// is_empty checks if an array has no elements.
	// Returns true if the array is empty, false otherwise.
	"arrays.is_empty": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.is_empty() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.is_empty() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// append adds one or more elements to the end of an array.
	// Modifies array in-place. Takes array and values to append.
	"arrays.append": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("arrays.append() takes at least 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return newError("arrays.append() requires an array as first argument")
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E5007",
				}
			}
			if err := checkIterating(arr, "arrays.append"); err != nil {
				return err
			}
			arr.Elements = append(arr.Elements, args[1:]...)
			return object.NIL
		},
	},

	// unshift adds one or more elements to the beginning of an array.
	// Modifies array in-place. Takes array and values to prepend.
	"arrays.unshift": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("arrays.unshift() takes at least 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return newError("arrays.unshift() requires an array as first argument")
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E5007",
				}
			}
			if err := checkIterating(arr, "arrays.unshift"); err != nil {
				return err
			}
			// Prepend elements in-place
			newElements := make([]object.Object, 0, len(arr.Elements)+len(args)-1)
			newElements = append(newElements, args[1:]...)
			newElements = append(newElements, arr.Elements...)
			arr.Elements = newElements
			return object.NIL
		},
	},

	// insert adds an element at a specific index in the array.
	// Takes array, index, and value. Modifies array in-place.
	"arrays.insert": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return newError("arrays.insert() takes exactly 3 arguments (array, index, value)")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return newError("arrays.insert() requires an array as first argument")
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if err := checkIterating(arr, "arrays.insert"); err != nil {
				return err
			}
			idx, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "arrays.insert() requires an integer index"}
			}
			index := int(idx.Value.Int64())
			if index < 0 || index > len(arr.Elements) {
				return &object.Error{Code: "E5003", Message: "arrays.insert() index out of bounds"}
			}
			// Insert element at index by modifying the array in-place
			arr.Elements = append(arr.Elements, nil)
			copy(arr.Elements[index+1:], arr.Elements[index:])
			arr.Elements[index] = args[2]
			return object.NIL
		},
	},

	// pop removes and returns the last element of an array.
	// Modifies array in-place. Returns error if array is empty.
	"arrays.pop": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.pop() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.pop() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if err := checkIterating(arr, "arrays.pop"); err != nil {
				return err
			}
			if len(arr.Elements) == 0 {
				return &object.Error{Code: "E9001", Message: "arrays.pop() cannot pop from empty array"}
			}
			lastElement := arr.Elements[len(arr.Elements)-1]
			arr.Elements = arr.Elements[:len(arr.Elements)-1]
			return lastElement
		},
	},

	// shift removes and returns the first element of an array.
	// Modifies array in-place. Returns error if array is empty.
	"arrays.shift": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.shift() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.shift() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if err := checkIterating(arr, "arrays.shift"); err != nil {
				return err
			}
			if len(arr.Elements) == 0 {
				return &object.Error{Code: "E9001", Message: "arrays.shift() cannot shift from empty array"}
			}
			firstElement := arr.Elements[0]
			arr.Elements = arr.Elements[1:]
			return firstElement
		},
	},

	// remove_at removes the element at a specific index.
	// Takes array and index. Modifies array in-place.
	"arrays.remove_at": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.remove_at() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.remove_at() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if err := checkIterating(arr, "arrays.remove_at"); err != nil {
				return err
			}
			idx, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "arrays.remove_at() requires an integer index"}
			}
			index := int(idx.Value.Int64())
			if index < 0 || index >= len(arr.Elements) {
				return &object.Error{Code: "E5003", Message: fmt.Sprintf("arrays.remove_at() index out of bounds: %d (array length: %d)", index, len(arr.Elements))}
			}
			// Remove element at index by modifying the array in-place
			arr.Elements = append(arr.Elements[:index], arr.Elements[index+1:]...)
			return object.NIL
		},
	},

	// remove_value removes the first occurrence of a value from an array.
	// Takes array and value. Modifies array in-place, does nothing if not found.
	"arrays.remove_value": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.remove_value() takes exactly 2 arguments (array, value)")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.remove_value() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if err := checkIterating(arr, "arrays.remove_value"); err != nil {
				return err
			}
			for i, el := range arr.Elements {
				if objectsEqual(el, args[1]) {
					// Remove first occurrence of value by modifying the array in-place
					arr.Elements = append(arr.Elements[:i], arr.Elements[i+1:]...)
					return object.NIL
				}
			}
			// Value not found - do nothing
			return object.NIL
		},
	},

	// remove_all removes all occurrences of a value from an array.
	// Takes array and value. Modifies array in-place.
	"arrays.remove_all": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.remove_all() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.remove_all() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if err := checkIterating(arr, "arrays.remove_all"); err != nil {
				return err
			}
			newElements := []object.Object{}
			for _, el := range arr.Elements {
				if !objectsEqual(el, args[1]) {
					newElements = append(newElements, el)
				}
			}
			arr.Elements = newElements
			return object.NIL
		},
	},

	// clear removes all elements from an array.
	// Takes an array. Modifies array in-place.
	"arrays.clear": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.clear() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.clear() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if err := checkIterating(arr, "arrays.clear"); err != nil {
				return err
			}
			arr.Elements = []object.Object{}
			return object.NIL
		},
	},

	// get returns the element at a specific index.
	// Takes array and index. Returns error if index is out of bounds.
	"arrays.get": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.get() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.get() requires an array"}
			}
			idx, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "arrays.get() requires an integer index"}
			}
			index := int(idx.Value.Int64())
			if index < 0 || index >= len(arr.Elements) {
				return &object.Error{Code: "E5003", Message: "arrays.get() index out of bounds"}
			}
			return arr.Elements[index]
		},
	},

	// first returns the first element of an array.
	// Takes an array. Returns nil if array is empty.
	"arrays.first": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.first() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.first() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return object.NIL
			}
			return arr.Elements[0]
		},
	},

	// last returns the last element of an array.
	// Takes an array. Returns nil if array is empty.
	"arrays.last": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.last() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.last() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return object.NIL
			}
			return arr.Elements[len(arr.Elements)-1]
		},
	},

	// set replaces the element at a specific index with a new value.
	// Takes array, index, and value. Modifies array in-place.
	"arrays.set": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return newError("arrays.set() takes exactly 3 arguments (array, index, value)")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.set() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if err := checkIterating(arr, "arrays.set"); err != nil {
				return err
			}
			idx, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "arrays.set() requires an integer index"}
			}
			index := int(idx.Value.Int64())
			if index < 0 || index >= len(arr.Elements) {
				return &object.Error{Code: "E5003", Message: "arrays.set() index out of bounds"}
			}
			arr.Elements[index] = args[2]
			return object.NIL
		},
	},

	// slice returns a new array containing elements from start to end index.
	// Takes array, start index, and optional end index. Supports negative indices.
	"arrays.slice": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return newError("arrays.slice() takes 2 or 3 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.slice() requires an array"}
			}
			start, ok := args[1].(*object.Integer)
			if !ok {
				return newError("arrays.slice() requires integer indices")
			}
			startIdx := int(start.Value.Int64())
			if startIdx < 0 {
				startIdx = len(arr.Elements) + startIdx
			}
			if startIdx < 0 {
				startIdx = 0
			}

			endIdx := len(arr.Elements)
			if len(args) == 3 {
				end, ok := args[2].(*object.Integer)
				if !ok {
					return newError("arrays.slice() requires integer indices")
				}
				endIdx = int(end.Value.Int64())
				if endIdx < 0 {
					endIdx = len(arr.Elements) + endIdx
				}
			}

			if startIdx > len(arr.Elements) {
				startIdx = len(arr.Elements)
			}
			if endIdx > len(arr.Elements) {
				endIdx = len(arr.Elements)
			}
			if startIdx > endIdx {
				return &object.Array{Elements: []object.Object{}, ElementType: arr.ElementType}
			}

			newElements := make([]object.Object, endIdx-startIdx)
			copy(newElements, arr.Elements[startIdx:endIdx])
			return &object.Array{Elements: newElements, ElementType: arr.ElementType}
		},
	},

	// take returns a new array with the first n elements.
	// Takes array and count. Returns fewer elements if array is smaller.
	"arrays.take": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.take() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.take() requires an array"}
			}
			n, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "arrays.take() requires an integer"}
			}
			count := int(n.Value.Int64())
			if count < 0 {
				count = 0
			}
			if count > len(arr.Elements) {
				count = len(arr.Elements)
			}
			newElements := make([]object.Object, count)
			copy(newElements, arr.Elements[:count])
			return &object.Array{Elements: newElements, ElementType: arr.ElementType}
		},
	},

	// drop returns a new array with the first n elements removed.
	// Takes array and count. Returns empty array if count exceeds length.
	"arrays.drop": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.drop() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.drop() requires an array"}
			}
			n, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "arrays.drop() requires an integer"}
			}
			count := int(n.Value.Int64())
			if count < 0 {
				count = 0
			}
			if count > len(arr.Elements) {
				count = len(arr.Elements)
			}
			newElements := make([]object.Object, len(arr.Elements)-count)
			copy(newElements, arr.Elements[count:])
			return &object.Array{Elements: newElements, ElementType: arr.ElementType}
		},
	},

	// contains checks if an array contains a specific value.
	// Takes array and value. Returns true if found, false otherwise.
	"arrays.contains": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.contains() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.contains() requires an array"}
			}
			for _, el := range arr.Elements {
				if objectsEqual(el, args[1]) {
					return object.TRUE
				}
			}
			return object.FALSE
		},
	},

	// index returns the index of the first occurrence of a value.
	// Takes array and value. Returns -1 if not found.
	"arrays.index": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.index() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.index() requires an array"}
			}
			for i, el := range arr.Elements {
				if objectsEqual(el, args[1]) {
					return &object.Integer{Value: big.NewInt(int64(i))}
				}
			}
			return &object.Integer{Value: big.NewInt(-1)}
		},
	},

	// last_index returns the index of the last occurrence of a value.
	// Takes array and value. Returns -1 if not found.
	"arrays.last_index": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.last_index() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.last_index() requires an array"}
			}
			for i := len(arr.Elements) - 1; i >= 0; i-- {
				if objectsEqual(arr.Elements[i], args[1]) {
					return &object.Integer{Value: big.NewInt(int64(i))}
				}
			}
			return &object.Integer{Value: big.NewInt(-1)}
		},
	},

	// count returns the number of occurrences of a value in an array.
	// Takes array and value. Returns integer count.
	"arrays.count": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.count() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.count() requires an array"}
			}
			count := 0
			for _, el := range arr.Elements {
				if objectsEqual(el, args[1]) {
					count++
				}
			}
			return &object.Integer{Value: big.NewInt(int64(count))}
		},
	},

	// reverse returns a new array with elements in reverse order.
	// Takes an array. Does not modify the original array.
	"arrays.reverse": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.reverse() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.reverse() requires an array"}
			}
			// Create a new array with reversed elements (don't mutate original)
			n := len(arr.Elements)
			reversed := make([]object.Object, n)
			for i := 0; i < n; i++ {
				reversed[i] = arr.Elements[n-1-i]
			}
			return &object.Array{Elements: reversed, Mutable: true, ElementType: arr.ElementType}
		},
	},

	// sort sorts an array in ascending order.
	// Takes an array. Modifies array in-place.
	"arrays.sort": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.sort() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.sort() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if err := checkIterating(arr, "arrays.sort"); err != nil {
				return err
			}
			if len(arr.Elements) == 0 {
				return object.NIL
			}

			// Sort in-place
			sort.Slice(arr.Elements, func(i, j int) bool {
				return compareObjects(arr.Elements[i], arr.Elements[j]) < 0
			})

			return object.NIL
		},
	},

	// sort_desc sorts an array in descending order.
	// Takes an array. Modifies array in-place.
	"arrays.sort_desc": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.sort_desc() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.sort_desc() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if err := checkIterating(arr, "arrays.sort_desc"); err != nil {
				return err
			}
			if len(arr.Elements) == 0 {
				return object.NIL
			}

			// Sort in-place (descending)
			sort.Slice(arr.Elements, func(i, j int) bool {
				return compareObjects(arr.Elements[i], arr.Elements[j]) > 0
			})

			return object.NIL
		},
	},

	// shuffle randomly reorders the elements of an array.
	// Takes an array. Modifies array in-place using Fisher-Yates algorithm.
	"arrays.shuffle": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.shuffle() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.shuffle() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if err := checkIterating(arr, "arrays.shuffle"); err != nil {
				return err
			}

			// Shuffle in-place (Fisher-Yates)
			for i := len(arr.Elements) - 1; i > 0; i-- {
				j := int(randomInt(int64(i + 1)))
				arr.Elements[i], arr.Elements[j] = arr.Elements[j], arr.Elements[i]
			}

			return object.NIL
		},
	},

	// concat combines two or more arrays into a new array.
	// Takes two or more arrays. Returns a new array with all elements.
	"arrays.concat": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("arrays.concat() takes at least 2 arguments")
			}
			var newElements []object.Object
			var elementType string
			for i, arg := range args {
				arr, ok := arg.(*object.Array)
				if !ok {
					return newError("arrays.concat() requires arrays")
				}
				if i == 0 {
					elementType = arr.ElementType
				}
				newElements = append(newElements, arr.Elements...)
			}
			return &object.Array{Elements: newElements, ElementType: elementType}
		},
	},

	// zip combines two arrays into an array of pairs.
	// Takes two arrays. Returns array of [a, b] pairs up to shorter length.
	"arrays.zip": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.zip() takes exactly 2 arguments")
			}
			arr1, ok := args[0].(*object.Array)
			if !ok {
				return newError("arrays.zip() requires arrays")
			}
			arr2, ok := args[1].(*object.Array)
			if !ok {
				return newError("arrays.zip() requires arrays")
			}

			minLen := len(arr1.Elements)
			if len(arr2.Elements) < minLen {
				minLen = len(arr2.Elements)
			}

			newElements := make([]object.Object, minLen)
			for i := 0; i < minLen; i++ {
				newElements[i] = &object.Array{Elements: []object.Object{arr1.Elements[i], arr2.Elements[i]}, ElementType: "any"}
			}
			return &object.Array{Elements: newElements, ElementType: "array"}
		},
	},

	// flatten flattens a nested array by one level.
	// Takes an array of arrays. Returns a new flat array.
	"arrays.flatten": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.flatten() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.flatten() requires an array"}
			}

			var newElements []object.Object
			var elementType string
			// Try to extract element type from outer array's ElementType
			// If it's like "[int]", the flattened result should be "int"
			if len(arr.ElementType) > 2 && arr.ElementType[0] == '[' && arr.ElementType[len(arr.ElementType)-1] == ']' {
				elementType = arr.ElementType[1 : len(arr.ElementType)-1]
			}
			for _, el := range arr.Elements {
				if innerArr, ok := el.(*object.Array); ok {
					newElements = append(newElements, innerArr.Elements...)
					// Prefer inner array's ElementType if available
					if elementType == "" && innerArr.ElementType != "" {
						elementType = innerArr.ElementType
					}
				} else {
					newElements = append(newElements, el)
				}
			}
			return &object.Array{Elements: newElements, ElementType: elementType}
		},
	},

	// unique returns a new array with duplicate values removed.
	// Takes an array. Preserves the order of first occurrences.
	"arrays.unique": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.unique() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.unique() requires an array"}
			}

			var newElements []object.Object
			seen := make(map[string]bool)
			for _, el := range arr.Elements {
				key := el.Inspect()
				if !seen[key] {
					seen[key] = true
					newElements = append(newElements, el)
				}
			}
			return &object.Array{Elements: newElements, ElementType: arr.ElementType}
		},
	},

	// duplicates returns a new array containing only values that appear more than once.
	// Takes an array. Returns one copy of each duplicate value.
	"arrays.duplicates": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.duplicates() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.duplicates() requires an array"}
			}

			var newElements []object.Object
			count := make(map[string]int)
			added := make(map[string]bool)

			for _, el := range arr.Elements {
				key := el.Inspect()
				count[key]++
			}

			for _, el := range arr.Elements {
				key := el.Inspect()
				if count[key] > 1 && !added[key] {
					added[key] = true
					newElements = append(newElements, el)
				}
			}
			return &object.Array{Elements: newElements, ElementType: arr.ElementType}
		},
	},

	// sum returns the sum of all numeric elements in an array.
	// Takes a numeric array. Returns int if all integers, float otherwise.
	"arrays.sum": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.sum() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.sum() requires an array"}
			}

			var sum float64
			hasFloat := false
			for _, el := range arr.Elements {
				switch v := el.(type) {
				case *object.Integer:
					f, _ := new(big.Float).SetInt(v.Value).Float64()
					sum += f
				case *object.Float:
					sum += v.Value
					hasFloat = true
				default:
					return &object.Error{Code: "E9002", Message: "arrays.sum() requires numeric array"}
				}
			}
			if hasFloat {
				return &object.Float{Value: sum}
			}
			return &object.Integer{Value: big.NewInt(int64(sum))}
		},
	},

	// product returns the product of all numeric elements in an array.
	// Takes a numeric array. Returns 1 for empty arrays.
	"arrays.product": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.product() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.product() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return &object.Integer{Value: big.NewInt(1)}
			}

			product := 1.0
			hasFloat := false
			for _, el := range arr.Elements {
				switch v := el.(type) {
				case *object.Integer:
					f, _ := new(big.Float).SetInt(v.Value).Float64()
					product *= f
				case *object.Float:
					product *= v.Value
					hasFloat = true
				default:
					return &object.Error{Code: "E9002", Message: "arrays.product() requires numeric array"}
				}
			}
			if hasFloat {
				return &object.Float{Value: product}
			}
			return &object.Integer{Value: big.NewInt(int64(product))}
		},
	},

	// min returns the minimum value in an array.
	// Takes a non-empty array. Returns error if array is empty.
	"arrays.min": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.min() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.min() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return &object.Error{Code: "E9001", Message: "arrays.min() requires non-empty array"}
			}

			minVal := arr.Elements[0]
			for _, el := range arr.Elements[1:] {
				if compareObjects(el, minVal) < 0 {
					minVal = el
				}
			}
			return minVal
		},
	},

	// max returns the maximum value in an array.
	// Takes a non-empty array. Returns error if array is empty.
	"arrays.max": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.max() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.max() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return &object.Error{Code: "E9001", Message: "arrays.max() requires non-empty array"}
			}

			maxVal := arr.Elements[0]
			for _, el := range arr.Elements[1:] {
				if compareObjects(el, maxVal) > 0 {
					maxVal = el
				}
			}
			return maxVal
		},
	},

	// avg returns the average of all numeric elements in an array.
	// Takes a non-empty numeric array. Returns a float.
	"arrays.avg": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.avg() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.avg() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return &object.Error{Code: "E9001", Message: "arrays.avg() requires non-empty array"}
			}

			var sum float64
			for _, el := range arr.Elements {
				switch v := el.(type) {
				case *object.Integer:
					f, _ := new(big.Float).SetInt(v.Value).Float64()
					sum += f
				case *object.Float:
					sum += v.Value
				default:
					return &object.Error{Code: "E9002", Message: "arrays.avg() requires numeric array"}
				}
			}
			return &object.Float{Value: sum / float64(len(arr.Elements))}
		},
	},

	// repeat creates a new array with a value repeated n times.
	// Takes value and count. Returns array with n copies of value.
	"arrays.repeat": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.repeat() takes exactly 2 arguments")
			}
			n, ok := args[1].(*object.Integer)
			if !ok {
				return newError("arrays.repeat() requires integer count")
			}
			count := int(n.Value.Int64())
			if count < 0 {
				count = 0
			}
			elements := make([]object.Object, count)
			for i := 0; i < count; i++ {
				elements[i] = args[0]
			}
			return &object.Array{Elements: elements, ElementType: object.GetEZTypeName(args[0])}
		},
	},

	// fill replaces all elements in an array with a specified value.
	// Takes array and value. Modifies array in-place.
	"arrays.fill": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.fill() takes exactly 2 arguments (array, value)")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.fill() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if err := checkIterating(arr, "arrays.fill"); err != nil {
				return err
			}
			for i := range arr.Elements {
				arr.Elements[i] = args[1]
			}
			return object.NIL
		},
	},

	// join concatenates array elements into a string with a separator.
	// Takes array and separator string. Returns combined string.
	"arrays.join": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.join() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.join() requires an array"}
			}
			sep, ok := args[1].(*object.String)
			if !ok {
				return newError("arrays.join() requires a string separator")
			}

			if len(arr.Elements) == 0 {
				return &object.String{Value: ""}
			}

			return &object.String{Value: JoinObjects(arr.Elements, sep.Value)}
		},
	},

	// all_equal checks if all elements in an array are equal.
	// Takes an array. Returns true for empty or single-element arrays.
	"arrays.all_equal": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.all_equal() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.all_equal() requires an array"}
			}
			if len(arr.Elements) <= 1 {
				return object.TRUE
			}
			first := arr.Elements[0]
			for _, el := range arr.Elements[1:] {
				if !objectsEqual(el, first) {
					return object.FALSE
				}
			}
			return object.TRUE
		},
	},

	// chunk splits an array into smaller arrays of a specified size.
	// Takes array and size. Returns array of arrays.
	"arrays.chunk": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.chunk() takes exactly 2 arguments (array, size)")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.chunk() requires an array"}
			}
			size, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "arrays.chunk() requires an integer size"}
			}
			if size.Value.Sign() <= 0 {
				return &object.Error{Code: "E9004", Message: "arrays.chunk() size must be greater than 0"}
			}

			chunkSize := int(size.Value.Int64())
			var chunks []object.Object

			for i := 0; i < len(arr.Elements); i += chunkSize {
				end := i + chunkSize
				if end > len(arr.Elements) {
					end = len(arr.Elements)
				}
				chunk := make([]object.Object, end-i)
				copy(chunk, arr.Elements[i:end])
				chunks = append(chunks, &object.Array{Elements: chunk, Mutable: true, ElementType: arr.ElementType})
			}

			return &object.Array{Elements: chunks, Mutable: true, ElementType: "array"}
		},
	},

	// equals checks if two arrays have the same elements in the same order.
	// Takes two arrays. Returns true if equal, false otherwise.
	"arrays.equals": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.equals() takes exactly 2 arguments (array1, array2)")
			}
			a1, ok1 := args[0].(*object.Array)
			a2, ok2 := args[1].(*object.Array)
			if !ok1 || !ok2 {
				return &object.Error{Code: "E7002", Message: "arrays.equals() requires two arrays"}
			}
			// Check same length
			if len(a1.Elements) != len(a2.Elements) {
				return object.FALSE
			}
			// Check all elements match
			for i, elem := range a1.Elements {
				if !objectsEqual(elem, a2.Elements[i]) {
					return object.FALSE
				}
			}
			return object.TRUE
		},
	},
}

// Helper functions for arrays
func newError(format string, args ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, args...)}
}

func objectsEqual(a, b object.Object) bool {
	if a.Type() != b.Type() {
		return false
	}
	return a.Inspect() == b.Inspect()
}

func compareObjects(a, b object.Object) int {
	if ai, ok := a.(*object.Integer); ok {
		if bi, ok := b.(*object.Integer); ok {
			return ai.Value.Cmp(bi.Value)
		}
	}
	if af, ok := a.(*object.Float); ok {
		if bf, ok := b.(*object.Float); ok {
			if af.Value < bf.Value {
				return -1
			} else if af.Value > bf.Value {
				return 1
			}
			return 0
		}
	}
	if as, ok := a.(*object.String); ok {
		if bs, ok := b.(*object.String); ok {
			if as.Value < bs.Value {
				return -1
			} else if as.Value > bs.Value {
				return 1
			}
			return 0
		}
	}
	if a.Inspect() < b.Inspect() {
		return -1
	} else if a.Inspect() > b.Inspect() {
		return 1
	}
	return 0
}

func randomInt(max int64) int64 {
	if max <= 0 {
		return 0
	}
	return rand.Int63n(max)
}
