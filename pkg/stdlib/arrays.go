package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/marshallburns/ez/pkg/object"
)

// ArraysBuiltins contains the arrays module functions
var ArraysBuiltins = map[string]*object.Builtin{
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
					Code:    "E4005",
				}
			}
			arr.Elements = append(arr.Elements, args[1:]...)
			return object.NIL
		},
	},

	"arrays.unshift": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("arrays.unshift() takes at least 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return newError("arrays.unshift() requires an array as first argument")
			}
			newElements := make([]object.Object, 0, len(arr.Elements)+len(args)-1)
			newElements = append(newElements, args[1:]...)
			newElements = append(newElements, arr.Elements...)
			return &object.Array{Elements: newElements}
		},
	},

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
			idx, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "arrays.insert() requires an integer index"}
			}
			index := int(idx.Value)
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
			if len(arr.Elements) == 0 {
				return &object.Error{Code: "E9001", Message: "arrays.pop() cannot pop from empty array"}
			}
			lastElement := arr.Elements[len(arr.Elements)-1]
			arr.Elements = arr.Elements[:len(arr.Elements)-1]
			return lastElement
		},
	},

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
			if len(arr.Elements) == 0 {
				return &object.Error{Code: "E9001", Message: "arrays.shift() cannot shift from empty array"}
			}
			firstElement := arr.Elements[0]
			arr.Elements = arr.Elements[1:]
			return firstElement
		},
	},

	"arrays.remove_at": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.remove_at() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.remove_at() requires an array"}
			}
			idx, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "arrays.remove_at() requires an integer index"}
			}
			index := int(idx.Value)
			if index < 0 || index >= len(arr.Elements) {
				return &object.Error{Code: "E5003", Message: "arrays.remove_at() index out of bounds"}
			}
			newElements := make([]object.Object, len(arr.Elements)-1)
			copy(newElements[:index], arr.Elements[:index])
			copy(newElements[index:], arr.Elements[index+1:])
			return &object.Array{Elements: newElements}
		},
	},

	// arrays.remove(arr, index) - Removes element at the specified INDEX (not value!)
	// Use arrays.remove_value() if you need to remove by value instead.
	// Modifies array in-place, returns NIL.
	"arrays.remove": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.remove() takes exactly 2 arguments (array, index)")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.remove() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			idx, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "arrays.remove() requires an integer index"}
			}
			index := int(idx.Value)
			if index < 0 || index >= len(arr.Elements) {
				return &object.Error{Code: "E9001", Message: fmt.Sprintf("arrays.remove() index out of bounds: %d (array length: %d)", index, len(arr.Elements))}
			}
			// Remove element at index by modifying the array in-place
			arr.Elements = append(arr.Elements[:index], arr.Elements[index+1:]...)
			return object.NIL
		},
	},

	// arrays.remove_value(arr, value) - Removes the FIRST occurrence of the specified VALUE.
	// Use arrays.remove() if you need to remove by index instead.
	// Modifies array in-place, returns NIL. Does nothing if value not found.
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

	"arrays.remove_all": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.remove_all() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.remove_all() requires an array"}
			}
			newElements := []object.Object{}
			for _, el := range arr.Elements {
				if !objectsEqual(el, args[1]) {
					newElements = append(newElements, el)
				}
			}
			return &object.Array{Elements: newElements}
		},
	},

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
			arr.Elements = []object.Object{}
			return object.NIL
		},
	},

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
			index := int(idx.Value)
			if index < 0 || index >= len(arr.Elements) {
				return &object.Error{Code: "E5003", Message: "arrays.get() index out of bounds"}
			}
			return arr.Elements[index]
		},
	},

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

	"arrays.set": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return newError("arrays.set() takes exactly 3 arguments (array, index, value)")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.set() requires an array"}
			}
			idx, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "arrays.set() requires an integer index"}
			}
			index := int(idx.Value)
			if index < 0 || index >= len(arr.Elements) {
				return &object.Error{Code: "E5003", Message: "arrays.set() index out of bounds"}
			}
			newElements := make([]object.Object, len(arr.Elements))
			copy(newElements, arr.Elements)
			newElements[index] = args[2]
			return &object.Array{Elements: newElements}
		},
	},

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
			startIdx := int(start.Value)
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
				endIdx = int(end.Value)
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
				return &object.Array{Elements: []object.Object{}}
			}

			newElements := make([]object.Object, endIdx-startIdx)
			copy(newElements, arr.Elements[startIdx:endIdx])
			return &object.Array{Elements: newElements}
		},
	},

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
			count := int(n.Value)
			if count < 0 {
				count = 0
			}
			if count > len(arr.Elements) {
				count = len(arr.Elements)
			}
			newElements := make([]object.Object, count)
			copy(newElements, arr.Elements[:count])
			return &object.Array{Elements: newElements}
		},
	},

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
			count := int(n.Value)
			if count < 0 {
				count = 0
			}
			if count > len(arr.Elements) {
				count = len(arr.Elements)
			}
			newElements := make([]object.Object, len(arr.Elements)-count)
			copy(newElements, arr.Elements[count:])
			return &object.Array{Elements: newElements}
		},
	},

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
					return &object.Integer{Value: int64(i)}
				}
			}
			return &object.Integer{Value: -1}
		},
	},

	"arrays.index_of": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.index_of() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.index_of() requires an array"}
			}
			for i, el := range arr.Elements {
				if objectsEqual(el, args[1]) {
					return &object.Integer{Value: int64(i)}
				}
			}
			return &object.Integer{Value: -1}
		},
	},

	"arrays.last_index_of": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.last_index_of() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.last_index_of() requires an array"}
			}
			for i := len(arr.Elements) - 1; i >= 0; i-- {
				if objectsEqual(arr.Elements[i], args[1]) {
					return &object.Integer{Value: int64(i)}
				}
			}
			return &object.Integer{Value: -1}
		},
	},

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
			return &object.Integer{Value: int64(count)}
		},
	},

	"arrays.reverse": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("arrays.reverse() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "arrays.reverse() requires an array"}
			}
			if !arr.Mutable {
				return &object.Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			// Reverse in-place and return the array
			for i, j := 0, len(arr.Elements)-1; i < j; i, j = i+1, j-1 {
				arr.Elements[i], arr.Elements[j] = arr.Elements[j], arr.Elements[i]
			}
			return arr
		},
	},

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

			// Shuffle in-place (Fisher-Yates)
			for i := len(arr.Elements) - 1; i > 0; i-- {
				j := int(randomInt(int64(i + 1)))
				arr.Elements[i], arr.Elements[j] = arr.Elements[j], arr.Elements[i]
			}

			return object.NIL
		},
	},

	"arrays.concat": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("arrays.concat() takes at least 2 arguments")
			}
			var newElements []object.Object
			for _, arg := range args {
				arr, ok := arg.(*object.Array)
				if !ok {
					return newError("arrays.concat() requires arrays")
				}
				newElements = append(newElements, arr.Elements...)
			}
			return &object.Array{Elements: newElements}
		},
	},

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
				newElements[i] = &object.Array{Elements: []object.Object{arr1.Elements[i], arr2.Elements[i]}}
			}
			return &object.Array{Elements: newElements}
		},
	},

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
			for _, el := range arr.Elements {
				if innerArr, ok := el.(*object.Array); ok {
					newElements = append(newElements, innerArr.Elements...)
				} else {
					newElements = append(newElements, el)
				}
			}
			return &object.Array{Elements: newElements}
		},
	},

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
			return &object.Array{Elements: newElements}
		},
	},

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
			return &object.Array{Elements: newElements}
		},
	},

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
					sum += float64(v.Value)
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
			return &object.Integer{Value: int64(sum)}
		},
	},

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
				return &object.Integer{Value: 1}
			}

			product := 1.0
			hasFloat := false
			for _, el := range arr.Elements {
				switch v := el.(type) {
				case *object.Integer:
					product *= float64(v.Value)
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
			return &object.Integer{Value: int64(product)}
		},
	},

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
					sum += float64(v.Value)
				case *object.Float:
					sum += v.Value
				default:
					return &object.Error{Code: "E9002", Message: "arrays.avg() requires numeric array"}
				}
			}
			return &object.Float{Value: sum / float64(len(arr.Elements))}
		},
	},

	"arrays.range": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 3 {
				return newError("arrays.range() takes 1 to 3 arguments")
			}

			var start, end, step int64

			if len(args) == 1 {
				end = args[0].(*object.Integer).Value
				start = 0
				step = 1
			} else if len(args) == 2 {
				start = args[0].(*object.Integer).Value
				end = args[1].(*object.Integer).Value
				step = 1
			} else {
				start = args[0].(*object.Integer).Value
				end = args[1].(*object.Integer).Value
				step = args[2].(*object.Integer).Value
			}

			if step == 0 {
				return &object.Error{Code: "E9003", Message: "arrays.range() step cannot be zero"}
			}

			var elements []object.Object
			if step > 0 {
				for i := start; i < end; i += step {
					elements = append(elements, &object.Integer{Value: i})
				}
			} else {
				for i := start; i > end; i += step {
					elements = append(elements, &object.Integer{Value: i})
				}
			}
			return &object.Array{Elements: elements}
		},
	},

	"arrays.repeat": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("arrays.repeat() takes exactly 2 arguments")
			}
			n, ok := args[1].(*object.Integer)
			if !ok {
				return newError("arrays.repeat() requires integer count")
			}
			count := int(n.Value)
			if count < 0 {
				count = 0
			}
			elements := make([]object.Object, count)
			for i := 0; i < count; i++ {
				elements[i] = args[0]
			}
			return &object.Array{Elements: elements}
		},
	},

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
			for i := range arr.Elements {
				arr.Elements[i] = args[1]
			}
			return object.NIL
		},
	},

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

			result := arr.Elements[0].Inspect()
			for _, el := range arr.Elements[1:] {
				result += sep.Value + el.Inspect()
			}
			return &object.String{Value: result}
		},
	},

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
			if size.Value <= 0 {
				return &object.Error{Code: "E9004", Message: "arrays.chunk() size must be greater than 0"}
			}

			chunkSize := int(size.Value)
			var chunks []object.Object

			for i := 0; i < len(arr.Elements); i += chunkSize {
				end := i + chunkSize
				if end > len(arr.Elements) {
					end = len(arr.Elements)
				}
				chunk := make([]object.Object, end-i)
				copy(chunk, arr.Elements[i:end])
				chunks = append(chunks, &object.Array{Elements: chunk, Mutable: true})
			}

			return &object.Array{Elements: chunks, Mutable: true}
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
			if ai.Value < bi.Value {
				return -1
			} else if ai.Value > bi.Value {
				return 1
			}
			return 0
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
