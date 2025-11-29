package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"sort"
	"time"
)

var arraysBuiltins = map[string]*Builtin{

	/*
		Func: arrays.len()
		Desc: Returns the length of an array as an integer.

		Format: arrays.len(yourArray)

		Usage:

		temp arr [int] = {1,2,3}
		temp arrLen = arrays.len(arr)
		std.println(arrLen) # Output: 3
	*/
	"arrays.len": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.len() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.len() requires an array"}
			}
			return &Integer{Value: int64(len(arr.Elements))}
		},
	},

	// Returns a boolean indicating if the array is empty or not.
	"arrays.is_empty": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.is_empty() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.is_empty() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return TRUE
			}
			return FALSE
		},
	},
	/*
		Func: arrays.push()
		Desc: Adds one or more elements to the end of an array, Modifies the original array(Meaning no new array is returned).

		Format: arrays.push(yourArray, valueToAdd1, valueToAdd2, moreValues)

		Usage:

		temp arr [int] = {1,2,3}
		arrays.push(arr, 4, 5)
		std.println(arr) # Output: {1, 2, 3, 4, 5}
	*/
	"arrays.push": {
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return newError("arrays.push() takes at least 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return newError("arrays.push() requires an array as first argument")
			}
			// Check if array is mutable
			if !arr.Mutable {
				return &Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			// Modify the array in-place by appending to its Elements slice
			arr.Elements = append(arr.Elements, args[1:]...)
			return NIL
		},
	},

	/*
		Func: arrays.unshift()
		Desc: Adds one or more elements to the beginning of an array, Returns a new array.

		Format: arrays.unshift(yourArray, valueToAdd1, valueToAdd2, moreValues)

		Usage:

		temp oldArray [int] = {2, 3, 4}
		temp newArr = arrays.unshift(oldArray, 0, 1)
		std.println(newArr) # Output: {0, 1, 2, 3, 4}
	*/
	"arrays.unshift": {
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return newError("arrays.unshift() takes at least 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return newError("arrays.unshift() requires an array as first argument")
			}
			newElements := make([]Object, 0, len(arr.Elements)+len(args)-1)
			newElements = append(newElements, args[1:]...)
			newElements = append(newElements, arr.Elements...)
			return &Array{Elements: newElements}
		},
	},

	/*
		Func: arrays.insert()
		Desc: Inserts an element at a specified index in an array, Returns a new array.

		Format: arrays.insert(yourArray, indexToInsertAt, valueToInsert)
		Usage:

		temp oldArray [int] = {10, 30, 40}
		temp newArr = arrays.insert(oldArray, 1, 20)
		std.println(newArr) # Output: {10, 20, 30, 40}
	*/
	"arrays.insert": {
		Fn: func(args ...Object) Object {
			if len(args) != 3 {
				return newError("arrays.insert() takes exactly 3 arguments (array, index, value)")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return newError("arrays.insert() requires an array as first argument")
			}
			idx, ok := args[1].(*Integer)
			if !ok {
				return &Error{Code: "E9003", Message: "arrays.insert() requires an integer index"}
			}
			index := int(idx.Value)
			if index < 0 || index > len(arr.Elements) {
				return &Error{Code: "E9001", Message: "arrays.insert() index out of bounds"}
			}
			newElements := make([]Object, len(arr.Elements)+1)
			copy(newElements[:index], arr.Elements[:index])
			newElements[index] = args[2]
			copy(newElements[index+1:], arr.Elements[index:])
			return &Array{Elements: newElements}
		},
	},
	/*
		Func: arrays.pop()
		Desc: Removes and returns the last element from an array, Modifies the original array(Meaning no new array is returned).

		Format: arrays.pop(yourArray)
		Usage:

		temp arr [char] = {'A', 'B', 'C'}
		temp lastElement = arrays.pop(arr)
		std.println(arr) # Output: {'A', 'B'}
		std.println(lastElement) # Output: 'C'
	*/
	"arrays.pop": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.pop() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.pop() requires an array"}
			}
			// Check if array is mutable
			if !arr.Mutable {
				return &Error{
					Message: "cannot modify immutable array (declared as const)",
					Code:    "E4005",
				}
			}
			if len(arr.Elements) == 0 {
				return &Error{Code: "E9008", Message: "arrays.pop() cannot pop from empty array"}
			}
			// Get the last element
			lastElement := arr.Elements[len(arr.Elements)-1]
			// Remove it from the array in-place
			arr.Elements = arr.Elements[:len(arr.Elements)-1]
			return lastElement
		},
	},
	/*
		Func: arrays.shift()
		Desc: Removes and returns the first element from an array, Returns the first element.

		Format: arrays.shift(yourArray)
		Usage:

		temp arr [int] = {10, 20, 30, 40}
		temp firstElement = arrays.shift(arr)
		std.println(firstElement) # Output: 10
	*/
	"arrays.shift": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.shift() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.shift() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return &Error{Code: "E9009", Message: "arrays.shift() cannot shift from empty array"}
			}
			return arr.Elements[0]
		},
	},
	"arrays.remove_at": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.remove_at() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.remove_at() requires an array"}
			}
			idx, ok := args[1].(*Integer)
			if !ok {
				return &Error{Code: "E9003", Message: "arrays.remove_at() requires an integer index"}
			}
			index := int(idx.Value)
			if index < 0 || index >= len(arr.Elements) {
				return &Error{Code: "E9001", Message: "arrays.remove_at() index out of bounds"}
			}
			newElements := make([]Object, len(arr.Elements)-1)
			copy(newElements[:index], arr.Elements[:index])
			copy(newElements[index:], arr.Elements[index+1:])
			return &Array{Elements: newElements}
		},
	},
	"arrays.remove": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.remove() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.remove() requires an array"}
			}
			// Find and remove first occurrence
			for i, el := range arr.Elements {
				if objectsEqual(el, args[1]) {
					newElements := make([]Object, len(arr.Elements)-1)
					copy(newElements[:i], arr.Elements[:i])
					copy(newElements[i:], arr.Elements[i+1:])
					return &Array{Elements: newElements}
				}
			}
			// Not found, return original
			return arr
		},
	},
	"arrays.remove_all": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.remove_all() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.remove_all() requires an array"}
			}
			newElements := []Object{}
			for _, el := range arr.Elements {
				if !objectsEqual(el, args[1]) {
					newElements = append(newElements, el)
				}
			}
			return &Array{Elements: newElements}
		},
	},
	"arrays.clear": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.clear() takes exactly 1 argument")
			}
			_, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.clear() requires an array"}
			}
			return &Array{Elements: []Object{}}
		},
	},

	// Accessing elements
	"arrays.get": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.get() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.get() requires an array"}
			}
			idx, ok := args[1].(*Integer)
			if !ok {
				return &Error{Code: "E9003", Message: "arrays.get() requires an integer index"}
			}
			index := int(idx.Value)
			if index < 0 || index >= len(arr.Elements) {
				return &Error{Code: "E9001", Message: "arrays.get() index out of bounds"}
			}
			return arr.Elements[index]
		},
	},
	"arrays.first": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.first() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.first() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return NIL
			}
			return arr.Elements[0]
		},
	},
	"arrays.last": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.last() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.last() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return NIL
			}
			return arr.Elements[len(arr.Elements)-1]
		},
	},
	"arrays.set": {
		Fn: func(args ...Object) Object {
			if len(args) != 3 {
				return newError("arrays.set() takes exactly 3 arguments (array, index, value)")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.set() requires an array"}
			}
			idx, ok := args[1].(*Integer)
			if !ok {
				return &Error{Code: "E9003", Message: "arrays.set() requires an integer index"}
			}
			index := int(idx.Value)
			if index < 0 || index >= len(arr.Elements) {
				return &Error{Code: "E9001", Message: "arrays.set() index out of bounds"}
			}
			newElements := make([]Object, len(arr.Elements))
			copy(newElements, arr.Elements)
			newElements[index] = args[2]
			return &Array{Elements: newElements}
		},
	},

	// Slicing
	"arrays.slice": {
		Fn: func(args ...Object) Object {
			if len(args) < 2 || len(args) > 3 {
				return newError("arrays.slice() takes 2 or 3 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.slice() requires an array"}
			}
			start, ok := args[1].(*Integer)
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
				end, ok := args[2].(*Integer)
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
				return &Array{Elements: []Object{}}
			}

			newElements := make([]Object, endIdx-startIdx)
			copy(newElements, arr.Elements[startIdx:endIdx])
			return &Array{Elements: newElements}
		},
	},
	"arrays.take": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.take() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.take() requires an array"}
			}
			n, ok := args[1].(*Integer)
			if !ok {
				return &Error{Code: "E9003", Message: "arrays.take() requires an integer"}
			}
			count := int(n.Value)
			if count < 0 {
				count = 0
			}
			if count > len(arr.Elements) {
				count = len(arr.Elements)
			}
			newElements := make([]Object, count)
			copy(newElements, arr.Elements[:count])
			return &Array{Elements: newElements}
		},
	},
	"arrays.drop": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.drop() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.drop() requires an array"}
			}
			n, ok := args[1].(*Integer)
			if !ok {
				return &Error{Code: "E9003", Message: "arrays.drop() requires an integer"}
			}
			count := int(n.Value)
			if count < 0 {
				count = 0
			}
			if count > len(arr.Elements) {
				count = len(arr.Elements)
			}
			newElements := make([]Object, len(arr.Elements)-count)
			copy(newElements, arr.Elements[count:])
			return &Array{Elements: newElements}
		},
	},

	// Searching
	"arrays.contains": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.contains() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.contains() requires an array"}
			}
			for _, el := range arr.Elements {
				if objectsEqual(el, args[1]) {
					return TRUE
				}
			}
			return FALSE
		},
	},
	"arrays.index_of": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.index_of() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.index_of() requires an array"}
			}
			for i, el := range arr.Elements {
				if objectsEqual(el, args[1]) {
					return &Integer{Value: int64(i)}
				}
			}
			return &Integer{Value: -1}
		},
	},
	"arrays.last_index_of": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.last_index_of() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.last_index_of() requires an array"}
			}
			for i := len(arr.Elements) - 1; i >= 0; i-- {
				if objectsEqual(arr.Elements[i], args[1]) {
					return &Integer{Value: int64(i)}
				}
			}
			return &Integer{Value: -1}
		},
	},
	"arrays.count": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.count() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.count() requires an array"}
			}
			count := 0
			for _, el := range arr.Elements {
				if objectsEqual(el, args[1]) {
					count++
				}
			}
			return &Integer{Value: int64(count)}
		},
	},

	// Reordering
	"arrays.reverse": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.reverse() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.reverse() requires an array"}
			}
			newElements := make([]Object, len(arr.Elements))
			for i, el := range arr.Elements {
				newElements[len(arr.Elements)-1-i] = el
			}
			return &Array{Elements: newElements}
		},
	},
	"arrays.sort": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.sort() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.sort() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return &Array{Elements: []Object{}}
			}

			newElements := make([]Object, len(arr.Elements))
			copy(newElements, arr.Elements)

			sort.Slice(newElements, func(i, j int) bool {
				return compareObjects(newElements[i], newElements[j]) < 0
			})

			return &Array{Elements: newElements}
		},
	},
	"arrays.sort_desc": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.sort_desc() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.sort_desc() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return &Array{Elements: []Object{}}
			}

			newElements := make([]Object, len(arr.Elements))
			copy(newElements, arr.Elements)

			sort.Slice(newElements, func(i, j int) bool {
				return compareObjects(newElements[i], newElements[j]) > 0
			})

			return &Array{Elements: newElements}
		},
	},
	"arrays.shuffle": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.shuffle() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.shuffle() requires an array"}
			}

			newElements := make([]Object, len(arr.Elements))
			copy(newElements, arr.Elements)

			// Fisher-Yates shuffle
			for i := len(newElements) - 1; i > 0; i-- {
				j := int(randomInt(int64(i + 1)))
				newElements[i], newElements[j] = newElements[j], newElements[i]
			}

			return &Array{Elements: newElements}
		},
	},

	// Combining arrays
	"arrays.concat": {
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return newError("arrays.concat() takes at least 2 arguments")
			}
			var newElements []Object
			for _, arg := range args {
				arr, ok := arg.(*Array)
				if !ok {
					return newError("arrays.concat() requires arrays")
				}
				newElements = append(newElements, arr.Elements...)
			}
			return &Array{Elements: newElements}
		},
	},
	"arrays.zip": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.zip() takes exactly 2 arguments")
			}
			arr1, ok := args[0].(*Array)
			if !ok {
				return newError("arrays.zip() requires arrays")
			}
			arr2, ok := args[1].(*Array)
			if !ok {
				return newError("arrays.zip() requires arrays")
			}

			minLen := len(arr1.Elements)
			if len(arr2.Elements) < minLen {
				minLen = len(arr2.Elements)
			}

			newElements := make([]Object, minLen)
			for i := 0; i < minLen; i++ {
				newElements[i] = &Array{Elements: []Object{arr1.Elements[i], arr2.Elements[i]}}
			}
			return &Array{Elements: newElements}
		},
	},
	"arrays.flatten": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.flatten() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.flatten() requires an array"}
			}

			var newElements []Object
			for _, el := range arr.Elements {
				if innerArr, ok := el.(*Array); ok {
					newElements = append(newElements, innerArr.Elements...)
				} else {
					newElements = append(newElements, el)
				}
			}
			return &Array{Elements: newElements}
		},
	},

	// Unique and duplicates
	"arrays.unique": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.unique() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.unique() requires an array"}
			}

			var newElements []Object
			seen := make(map[string]bool)
			for _, el := range arr.Elements {
				key := el.Inspect()
				if !seen[key] {
					seen[key] = true
					newElements = append(newElements, el)
				}
			}
			return &Array{Elements: newElements}
		},
	},
	"arrays.duplicates": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.duplicates() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.duplicates() requires an array"}
			}

			var newElements []Object
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
			return &Array{Elements: newElements}
		},
	},

	// Math on arrays
	"arrays.sum": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.sum() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.sum() requires an array"}
			}

			var sum float64
			hasFloat := false
			for _, el := range arr.Elements {
				switch v := el.(type) {
				case *Integer:
					sum += float64(v.Value)
				case *Float:
					sum += v.Value
					hasFloat = true
				default:
					return &Error{Code: "E9012", Message: "arrays.sum() requires numeric array"}
				}
			}
			if hasFloat {
				return &Float{Value: sum}
			}
			return &Integer{Value: int64(sum)}
		},
	},
	"arrays.product": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.product() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.product() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return &Integer{Value: 1}
			}

			product := 1.0
			hasFloat := false
			for _, el := range arr.Elements {
				switch v := el.(type) {
				case *Integer:
					product *= float64(v.Value)
				case *Float:
					product *= v.Value
					hasFloat = true
				default:
					return &Error{Code: "E9012", Message: "arrays.product() requires numeric array"}
				}
			}
			if hasFloat {
				return &Float{Value: product}
			}
			return &Integer{Value: int64(product)}
		},
	},
	"arrays.min": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.min() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.min() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return &Error{Code: "E9014", Message: "arrays.min() requires non-empty array"}
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
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.max() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.max() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return &Error{Code: "E9014", Message: "arrays.max() requires non-empty array"}
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
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.avg() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.avg() requires an array"}
			}
			if len(arr.Elements) == 0 {
				return &Error{Code: "E9014", Message: "arrays.avg() requires non-empty array"}
			}

			var sum float64
			for _, el := range arr.Elements {
				switch v := el.(type) {
				case *Integer:
					sum += float64(v.Value)
				case *Float:
					sum += v.Value
				default:
					return &Error{Code: "E9012", Message: "arrays.avg() requires numeric array"}
				}
			}
			return &Float{Value: sum / float64(len(arr.Elements))}
		},
	},

	// Creation helpers
	"arrays.range": {
		Fn: func(args ...Object) Object {
			if len(args) < 1 || len(args) > 3 {
				return newError("arrays.range() takes 1 to 3 arguments")
			}

			var start, end, step int64

			if len(args) == 1 {
				end = args[0].(*Integer).Value
				start = 0
				step = 1
			} else if len(args) == 2 {
				start = args[0].(*Integer).Value
				end = args[1].(*Integer).Value
				step = 1
			} else {
				start = args[0].(*Integer).Value
				end = args[1].(*Integer).Value
				step = args[2].(*Integer).Value
			}

			if step == 0 {
				return &Error{Code: "E9018", Message: "arrays.range() step cannot be zero"}
			}

			var elements []Object
			if step > 0 {
				for i := start; i < end; i += step {
					elements = append(elements, &Integer{Value: i})
				}
			} else {
				for i := start; i > end; i += step {
					elements = append(elements, &Integer{Value: i})
				}
			}
			return &Array{Elements: elements}
		},
	},
	"arrays.repeat": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.repeat() takes exactly 2 arguments")
			}
			n, ok := args[1].(*Integer)
			if !ok {
				return newError("arrays.repeat() requires integer count")
			}
			count := int(n.Value)
			if count < 0 {
				count = 0
			}
			elements := make([]Object, count)
			for i := 0; i < count; i++ {
				elements[i] = args[0]
			}
			return &Array{Elements: elements}
		},
	},
	"arrays.fill": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.fill() takes exactly 2 arguments (array, value)")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.fill() requires an array"}
			}
			elements := make([]Object, len(arr.Elements))
			for i := range elements {
				elements[i] = args[1]
			}
			return &Array{Elements: elements}
		},
	},

	// Copying
	"arrays.copy": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.copy() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.copy() requires an array"}
			}
			newElements := make([]Object, len(arr.Elements))
			copy(newElements, arr.Elements)
			return &Array{Elements: newElements}
		},
	},

	// String conversion
	"arrays.join": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("arrays.join() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.join() requires an array"}
			}
			sep, ok := args[1].(*String)
			if !ok {
				return newError("arrays.join() requires a string separator")
			}

			if len(arr.Elements) == 0 {
				return &String{Value: ""}
			}

			result := arr.Elements[0].Inspect()
			for _, el := range arr.Elements[1:] {
				result += sep.Value + el.Inspect()
			}
			return &String{Value: result}
		},
	},

	// Boolean checks
	"arrays.all_equal": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("arrays.all_equal() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Code: "E9005", Message: "arrays.all_equal() requires an array"}
			}
			if len(arr.Elements) <= 1 {
				return TRUE
			}
			first := arr.Elements[0]
			for _, el := range arr.Elements[1:] {
				if !objectsEqual(el, first) {
					return FALSE
				}
			}
			return TRUE
		},
	},
}

// Helper functions
func objectsEqual(a, b Object) bool {
	if a.Type() != b.Type() {
		return false
	}
	return a.Inspect() == b.Inspect()
}

func compareObjects(a, b Object) int {
	// Compare integers
	if ai, ok := a.(*Integer); ok {
		if bi, ok := b.(*Integer); ok {
			if ai.Value < bi.Value {
				return -1
			} else if ai.Value > bi.Value {
				return 1
			}
			return 0
		}
	}
	// Compare floats
	if af, ok := a.(*Float); ok {
		if bf, ok := b.(*Float); ok {
			if af.Value < bf.Value {
				return -1
			} else if af.Value > bf.Value {
				return 1
			}
			return 0
		}
	}
	// Compare strings
	if as, ok := a.(*String); ok {
		if bs, ok := b.(*String); ok {
			if as.Value < bs.Value {
				return -1
			} else if as.Value > bs.Value {
				return 1
			}
			return 0
		}
	}
	// Default: compare by string representation
	if a.Inspect() < b.Inspect() {
		return -1
	} else if a.Inspect() > b.Inspect() {
		return 1
	}
	return 0
}

func randomInt(max int64) int64 {
	// Simple random using time - already seeded in lib_math
	return int64(float64(max) * float64(time.Now().UnixNano()%1000) / 1000.0)
}

func init() {
	for name, builtin := range arraysBuiltins {
		builtins[name] = builtin
	}
}
