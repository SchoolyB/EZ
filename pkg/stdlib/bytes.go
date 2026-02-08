package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/marshallburns/ez/pkg/object"
)

// BytesBuiltins contains the bytes module functions for binary data operations
var BytesBuiltins = map[string]*object.Builtin{
	// ============================================================================
	// Creation Functions
	// ============================================================================

	// from_array creates a byte array from an array of integers.
	// Takes array of integers (0-255). Returns byte array.
	"bytes.from_array": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.from_array() takes exactly 1 argument (array)"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.from_array() requires an array argument"}
			}

			elements := make([]object.Object, len(arr.Elements))
			for i, elem := range arr.Elements {
				var val int64
				switch e := elem.(type) {
				case *object.Integer:
					val = e.Value.Int64()
				case *object.Byte:
					val = int64(e.Value)
				default:
					return &object.Error{Code: "E7004", Message: fmt.Sprintf("bytes.from_array() element %d must be an integer", i)}
				}
				if val < 0 || val > 255 {
					return &object.Error{Code: "E3022", Message: fmt.Sprintf("bytes.from_array() element %d value %d out of range (0-255)", i, val)}
				}
				elements[i] = &object.Byte{Value: uint8(val)}
			}
			return &object.Array{Elements: elements, ElementType: "byte"}
		},
	},

	// from_string creates a byte array from a UTF-8 encoded string.
	// Takes a string. Returns byte array with UTF-8 bytes.
	"bytes.from_string": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.from_string() takes exactly 1 argument (string)"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "bytes.from_string() requires a string argument"}
			}

			data := []byte(str.Value)
			elements := make([]object.Object, len(data))
			for i, b := range data {
				elements[i] = &object.Byte{Value: b}
			}
			return &object.Array{Elements: elements, ElementType: "byte"}
		},
	},

	// from_hex creates a byte array from a hexadecimal string.
	// Takes hex string. Returns ([byte], Error) tuple.
	"bytes.from_hex": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.from_hex() takes exactly 1 argument (hex string)"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "bytes.from_hex() requires a string argument"}
			}

			data, err := hex.DecodeString(str.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7014", fmt.Sprintf("bytes.from_hex() invalid hex string: %s", err.Error())),
				}}
			}

			elements := make([]object.Object, len(data))
			for i, b := range data {
				elements[i] = &object.Byte{Value: b}
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.Array{Elements: elements, ElementType: "byte"},
				object.NIL,
			}}
		},
	},

	// from_base64 creates a byte array from a base64 encoded string.
	// Takes base64 string. Returns ([byte], Error) tuple.
	"bytes.from_base64": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.from_base64() takes exactly 1 argument (base64 string)"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "bytes.from_base64() requires a string argument"}
			}

			data, err := base64.StdEncoding.DecodeString(str.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7014", fmt.Sprintf("bytes.from_base64() invalid base64 string: %s", err.Error())),
				}}
			}

			elements := make([]object.Object, len(data))
			for i, b := range data {
				elements[i] = &object.Byte{Value: b}
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.Array{Elements: elements, ElementType: "byte"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// Conversion Functions
	// ============================================================================

	// to_string converts a byte array to a UTF-8 string.
	// Takes byte array. Returns string.
	"bytes.to_string": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.to_string() takes exactly 1 argument (bytes)"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.to_string() requires a byte array argument"}
			}

			data := make([]byte, len(arr.Elements))
			for i, elem := range arr.Elements {
				b, ok := elem.(*object.Byte)
				if !ok {
					// Try integer for backwards compatibility
					if intVal, ok := elem.(*object.Integer); ok {
						val := intVal.Value.Int64()
						if val < 0 || val > 255 {
							return &object.Error{Code: "E7005", Message: fmt.Sprintf("byte value at index %d out of range: %d (must be 0-255)", i, val)}
						}
						data[i] = byte(val)
						continue
					}
					return &object.Error{Code: "E7002", Message: "bytes.to_string() requires a byte array"}
				}
				data[i] = b.Value
			}
			return &object.String{Value: string(data)}
		},
	},

	// to_array converts a byte array to an array of integers.
	// Takes byte array. Returns integer array.
	"bytes.to_array": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.to_array() takes exactly 1 argument (bytes)"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.to_array() requires a byte array argument"}
			}

			elements := make([]object.Object, len(arr.Elements))
			for i, elem := range arr.Elements {
				switch e := elem.(type) {
				case *object.Byte:
					elements[i] = &object.Integer{Value: big.NewInt(int64(e.Value))}
				case *object.Integer:
					elements[i] = e
				default:
					return &object.Error{Code: "E7002", Message: "bytes.to_array() requires a byte array"}
				}
			}
			return &object.Array{Elements: elements, ElementType: "int"}
		},
	},

	// to_hex converts a byte array to a lowercase hexadecimal string.
	// Takes byte array. Returns hex string.
	"bytes.to_hex": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.to_hex() takes exactly 1 argument (bytes)"}
			}
			data, err := bytesArgToSlice(args[0], "bytes.to_hex()")
			if err != nil {
				return err
			}
			return &object.String{Value: hex.EncodeToString(data)}
		},
	},

	// to_hex_upper converts a byte array to an uppercase hexadecimal string.
	// Takes byte array. Returns uppercase hex string.
	"bytes.to_hex_upper": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.to_hex_upper() takes exactly 1 argument (bytes)"}
			}
			data, err := bytesArgToSlice(args[0], "bytes.to_hex_upper()")
			if err != nil {
				return err
			}
			return &object.String{Value: strings.ToUpper(hex.EncodeToString(data))}
		},
	},

	// to_base64 converts a byte array to a base64 encoded string.
	// Takes byte array. Returns base64 string.
	"bytes.to_base64": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.to_base64() takes exactly 1 argument (bytes)"}
			}
			data, err := bytesArgToSlice(args[0], "bytes.to_base64()")
			if err != nil {
				return err
			}
			return &object.String{Value: base64.StdEncoding.EncodeToString(data)}
		},
	},

	// ============================================================================
	// Operations
	// ============================================================================

	// slice extracts a portion of bytes from start to end index.
	// Takes bytes, start, end. Supports negative indices. End is exclusive.
	"bytes.slice": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: "bytes.slice() takes exactly 3 arguments (bytes, start, end)"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.slice() requires a byte array as first argument"}
			}
			startArg, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "bytes.slice() requires an integer start index"}
			}
			endArg, ok := args[2].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "bytes.slice() requires an integer end index"}
			}

			length := int64(len(arr.Elements))
			start := startArg.Value.Int64()
			end := endArg.Value.Int64()

			// Handle negative indices
			if start < 0 {
				start = length + start
			}
			if end < 0 {
				end = length + end
			}

			// Clamp to valid range
			if start < 0 {
				start = 0
			}
			if end > length {
				end = length
			}
			if start > end {
				start = end
			}

			elements := make([]object.Object, end-start)
			copy(elements, arr.Elements[start:end])
			return &object.Array{Elements: elements, ElementType: "byte"}
		},
	},

	// concat joins two byte arrays into one.
	// Takes two byte arrays. Returns combined byte array.
	"bytes.concat": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.concat() takes exactly 2 arguments (bytes1, bytes2)"}
			}
			arr1, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.concat() requires byte arrays"}
			}
			arr2, ok := args[1].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.concat() requires byte arrays"}
			}

			elements := make([]object.Object, len(arr1.Elements)+len(arr2.Elements))
			copy(elements, arr1.Elements)
			copy(elements[len(arr1.Elements):], arr2.Elements)
			return &object.Array{Elements: elements, ElementType: "byte"}
		},
	},

	// join combines multiple byte arrays with a separator.
	// Takes array of byte arrays and separator. Returns combined bytes.
	"bytes.join": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.join() takes exactly 2 arguments (array, separator)"}
			}
			partsArr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.join() requires an array of byte arrays as first argument"}
			}
			sep, ok := args[1].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.join() requires a byte array separator"}
			}

			if len(partsArr.Elements) == 0 {
				return &object.Array{Elements: []object.Object{}, ElementType: "byte"}
			}

			var result []object.Object
			for i, part := range partsArr.Elements {
				partArr, ok := part.(*object.Array)
				if !ok {
					return &object.Error{Code: "E7002", Message: "bytes.join() array elements must be byte arrays"}
				}
				result = append(result, partArr.Elements...)
				if i < len(partsArr.Elements)-1 {
					result = append(result, sep.Elements...)
				}
			}
			return &object.Array{Elements: result, ElementType: "byte"}
		},
	},

	// split divides bytes into parts using a separator.
	// Takes bytes and separator. Returns array of byte arrays.
	"bytes.split": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.split() takes exactly 2 arguments (bytes, separator)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.split()")
			if errObj != nil {
				return errObj
			}
			sep, errObj := bytesArgToSlice(args[1], "bytes.split()")
			if errObj != nil {
				return errObj
			}

			parts := bytes.Split(data, sep)
			elements := make([]object.Object, len(parts))
			for i, part := range parts {
				partElements := make([]object.Object, len(part))
				for j, b := range part {
					partElements[j] = &object.Byte{Value: b}
				}
				elements[i] = &object.Array{Elements: partElements, ElementType: "byte"}
			}
			return &object.Array{Elements: elements, ElementType: "[byte]"}
		},
	},

	// contains checks if bytes contain a byte pattern.
	// Takes bytes and pattern. Returns true if found.
	"bytes.contains": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.contains() takes exactly 2 arguments (bytes, pattern)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.contains()")
			if errObj != nil {
				return errObj
			}
			pattern, errObj := bytesArgToSlice(args[1], "bytes.contains()")
			if errObj != nil {
				return errObj
			}

			if bytes.Contains(data, pattern) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// index finds the first occurrence of a pattern in bytes.
	// Takes bytes and pattern. Returns index or -1 if not found.
	"bytes.index": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.index() takes exactly 2 arguments (bytes, pattern)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.index()")
			if errObj != nil {
				return errObj
			}
			pattern, errObj := bytesArgToSlice(args[1], "bytes.index()")
			if errObj != nil {
				return errObj
			}

			return &object.Integer{Value: big.NewInt(int64(bytes.Index(data, pattern)))}
		},
	},

	// last_index finds the last occurrence of a pattern in bytes.
	// Takes bytes and pattern. Returns index or -1 if not found.
	"bytes.last_index": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.last_index() takes exactly 2 arguments (bytes, pattern)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.last_index()")
			if errObj != nil {
				return errObj
			}
			pattern, errObj := bytesArgToSlice(args[1], "bytes.last_index()")
			if errObj != nil {
				return errObj
			}

			return &object.Integer{Value: big.NewInt(int64(bytes.LastIndex(data, pattern)))}
		},
	},

	// count returns the number of non-overlapping pattern occurrences.
	// Takes bytes and pattern. Returns integer count.
	"bytes.count": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.count() takes exactly 2 arguments (bytes, pattern)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.count()")
			if errObj != nil {
				return errObj
			}
			pattern, errObj := bytesArgToSlice(args[1], "bytes.count()")
			if errObj != nil {
				return errObj
			}

			return &object.Integer{Value: big.NewInt(int64(bytes.Count(data, pattern)))}
		},
	},

	// compare lexicographically compares two byte arrays.
	// Takes two byte arrays. Returns -1, 0, or 1.
	"bytes.compare": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.compare() takes exactly 2 arguments (bytes1, bytes2)"}
			}
			data1, errObj := bytesArgToSlice(args[0], "bytes.compare()")
			if errObj != nil {
				return errObj
			}
			data2, errObj := bytesArgToSlice(args[1], "bytes.compare()")
			if errObj != nil {
				return errObj
			}

			return &object.Integer{Value: big.NewInt(int64(bytes.Compare(data1, data2)))}
		},
	},

	// equals checks if two byte arrays are identical.
	// Takes two byte arrays. Returns true if equal.
	"bytes.equals": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.equals() takes exactly 2 arguments (bytes1, bytes2)"}
			}
			data1, errObj := bytesArgToSlice(args[0], "bytes.equals()")
			if errObj != nil {
				return errObj
			}
			data2, errObj := bytesArgToSlice(args[1], "bytes.equals()")
			if errObj != nil {
				return errObj
			}

			if bytes.Equal(data1, data2) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// ============================================================================
	// Inspection Functions
	// ============================================================================

	// is_empty checks if a byte array has no elements.
	// Takes byte array. Returns true if empty.
	"bytes.is_empty": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.is_empty() takes exactly 1 argument (bytes)"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.is_empty() requires a byte array argument"}
			}

			if len(arr.Elements) == 0 {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// starts_with checks if bytes begin with a prefix.
	// Takes bytes and prefix. Returns true if bytes start with prefix.
	"bytes.starts_with": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.starts_with() takes exactly 2 arguments (bytes, prefix)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.starts_with()")
			if errObj != nil {
				return errObj
			}
			prefix, errObj := bytesArgToSlice(args[1], "bytes.starts_with()")
			if errObj != nil {
				return errObj
			}

			if bytes.HasPrefix(data, prefix) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// ends_with checks if bytes end with a suffix.
	// Takes bytes and suffix. Returns true if bytes end with suffix.
	"bytes.ends_with": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.ends_with() takes exactly 2 arguments (bytes, suffix)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.ends_with()")
			if errObj != nil {
				return errObj
			}
			suffix, errObj := bytesArgToSlice(args[1], "bytes.ends_with()")
			if errObj != nil {
				return errObj
			}

			if bytes.HasSuffix(data, suffix) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// ============================================================================
	// Manipulation Functions
	// ============================================================================

	// reverse returns a new byte array with elements in reverse order.
	// Takes byte array. Returns reversed copy.
	"bytes.reverse": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.reverse() takes exactly 1 argument (bytes)"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.reverse() requires a byte array argument"}
			}

			length := len(arr.Elements)
			elements := make([]object.Object, length)
			for i, elem := range arr.Elements {
				elements[length-1-i] = elem
			}
			return &object.Array{Elements: elements, ElementType: "byte"}
		},
	},

	// repeat creates a new byte array by repeating bytes n times.
	// Takes bytes and count. Returns repeated bytes.
	"bytes.repeat": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.repeat() takes exactly 2 arguments (bytes, count)"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.repeat() requires a byte array as first argument"}
			}
			count, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "bytes.repeat() requires an integer count"}
			}
			countVal := count.Value.Int64()
			if countVal < 0 {
				return &object.Error{Code: "E7011", Message: "bytes.repeat() count cannot be negative"}
			}

			elements := make([]object.Object, 0, len(arr.Elements)*int(countVal))
			for i := int64(0); i < countVal; i++ {
				elements = append(elements, arr.Elements...)
			}
			return &object.Array{Elements: elements, ElementType: "byte"}
		},
	},

	// replace replaces all occurrences of old bytes with new bytes.
	// Takes bytes, old pattern, new pattern. Returns modified bytes.
	"bytes.replace": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: "bytes.replace() takes exactly 3 arguments (bytes, old, new)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.replace()")
			if errObj != nil {
				return errObj
			}
			old, errObj := bytesArgToSlice(args[1], "bytes.replace()")
			if errObj != nil {
				return errObj
			}
			newBytes, errObj := bytesArgToSlice(args[2], "bytes.replace()")
			if errObj != nil {
				return errObj
			}

			result := bytes.ReplaceAll(data, old, newBytes)
			return sliceToByteArray(result)
		},
	},

	// replace_n replaces the first n occurrences of old bytes with new bytes.
	// Takes bytes, old, new, count. Returns modified bytes.
	"bytes.replace_n": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 4 {
				return &object.Error{Code: "E7001", Message: "bytes.replace_n() takes exactly 4 arguments (bytes, old, new, n)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.replace_n()")
			if errObj != nil {
				return errObj
			}
			old, errObj := bytesArgToSlice(args[1], "bytes.replace_n()")
			if errObj != nil {
				return errObj
			}
			newBytes, errObj := bytesArgToSlice(args[2], "bytes.replace_n()")
			if errObj != nil {
				return errObj
			}
			n, ok := args[3].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "bytes.replace_n() requires an integer count"}
			}

			result := bytes.Replace(data, old, newBytes, int(n.Value.Int64()))
			return sliceToByteArray(result)
		},
	},

	// trim removes leading and trailing bytes that appear in cutset.
	// Takes bytes and cutset. Returns trimmed bytes.
	"bytes.trim": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.trim() takes exactly 2 arguments (bytes, cutset)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.trim()")
			if errObj != nil {
				return errObj
			}
			cutset, errObj := bytesArgToSlice(args[1], "bytes.trim()")
			if errObj != nil {
				return errObj
			}

			result := bytes.Trim(data, string(cutset))
			return sliceToByteArray(result)
		},
	},

	// trim_left removes leading bytes that appear in cutset.
	// Takes bytes and cutset. Returns left-trimmed bytes.
	"bytes.trim_left": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.trim_left() takes exactly 2 arguments (bytes, cutset)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.trim_left()")
			if errObj != nil {
				return errObj
			}
			cutset, errObj := bytesArgToSlice(args[1], "bytes.trim_left()")
			if errObj != nil {
				return errObj
			}

			result := bytes.TrimLeft(data, string(cutset))
			return sliceToByteArray(result)
		},
	},

	// trim_right removes trailing bytes that appear in cutset.
	// Takes bytes and cutset. Returns right-trimmed bytes.
	"bytes.trim_right": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.trim_right() takes exactly 2 arguments (bytes, cutset)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.trim_right()")
			if errObj != nil {
				return errObj
			}
			cutset, errObj := bytesArgToSlice(args[1], "bytes.trim_right()")
			if errObj != nil {
				return errObj
			}

			result := bytes.TrimRight(data, string(cutset))
			return sliceToByteArray(result)
		},
	},

	// pad_left pads bytes on the left to reach specified length.
	// Takes bytes, target length, pad byte. Returns padded bytes.
	"bytes.pad_left": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: "bytes.pad_left() takes exactly 3 arguments (bytes, length, pad_byte)"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.pad_left() requires a byte array as first argument"}
			}
			length, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "bytes.pad_left() requires an integer length"}
			}
			padByte, err := getByteValue(args[2], "bytes.pad_left()")
			if err != nil {
				return err
			}

			lengthVal := length.Value.Int64()
			currentLen := int64(len(arr.Elements))
			if currentLen >= lengthVal {
				// Return copy of original
				elements := make([]object.Object, len(arr.Elements))
				copy(elements, arr.Elements)
				return &object.Array{Elements: elements, ElementType: "byte"}
			}

			padCount := lengthVal - currentLen
			elements := make([]object.Object, lengthVal)
			for i := int64(0); i < padCount; i++ {
				elements[i] = &object.Byte{Value: padByte}
			}
			copy(elements[padCount:], arr.Elements)
			return &object.Array{Elements: elements, ElementType: "byte"}
		},
	},

	// pad_right pads bytes on the right to reach specified length.
	// Takes bytes, target length, pad byte. Returns padded bytes.
	"bytes.pad_right": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: "bytes.pad_right() takes exactly 3 arguments (bytes, length, pad_byte)"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.pad_right() requires a byte array as first argument"}
			}
			length, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "bytes.pad_right() requires an integer length"}
			}
			padByte, err := getByteValue(args[2], "bytes.pad_right()")
			if err != nil {
				return err
			}

			lengthVal := length.Value.Int64()
			currentLen := int64(len(arr.Elements))
			if currentLen >= lengthVal {
				// Return copy of original
				elements := make([]object.Object, len(arr.Elements))
				copy(elements, arr.Elements)
				return &object.Array{Elements: elements, ElementType: "byte"}
			}

			elements := make([]object.Object, lengthVal)
			copy(elements, arr.Elements)
			for i := currentLen; i < lengthVal; i++ {
				elements[i] = &object.Byte{Value: padByte}
			}
			return &object.Array{Elements: elements, ElementType: "byte"}
		},
	},

	// ============================================================================
	// Bitwise Operations
	// ============================================================================

	// and performs bitwise AND on two equal-length byte arrays.
	// Takes two byte arrays. Returns ([byte], Error) tuple.
	"bytes.and": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.and() takes exactly 2 arguments (bytes1, bytes2)"}
			}
			data1, errObj := bytesArgToSlice(args[0], "bytes.and()")
			if errObj != nil {
				return errObj
			}
			data2, errObj := bytesArgToSlice(args[1], "bytes.and()")
			if errObj != nil {
				return errObj
			}

			if len(data1) != len(data2) {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7010", "bytes.and() requires byte arrays of equal length"),
				}}
			}

			result := make([]byte, len(data1))
			for i := range data1 {
				result[i] = data1[i] & data2[i]
			}
			return &object.ReturnValue{Values: []object.Object{
				sliceToByteArray(result),
				object.NIL,
			}}
		},
	},

	// or performs bitwise OR on two equal-length byte arrays.
	// Takes two byte arrays. Returns ([byte], Error) tuple.
	"bytes.or": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.or() takes exactly 2 arguments (bytes1, bytes2)"}
			}
			data1, errObj := bytesArgToSlice(args[0], "bytes.or()")
			if errObj != nil {
				return errObj
			}
			data2, errObj := bytesArgToSlice(args[1], "bytes.or()")
			if errObj != nil {
				return errObj
			}

			if len(data1) != len(data2) {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7010", "bytes.or() requires byte arrays of equal length"),
				}}
			}

			result := make([]byte, len(data1))
			for i := range data1 {
				result[i] = data1[i] | data2[i]
			}
			return &object.ReturnValue{Values: []object.Object{
				sliceToByteArray(result),
				object.NIL,
			}}
		},
	},

	// xor performs bitwise XOR on two equal-length byte arrays.
	// Takes two byte arrays. Returns ([byte], Error) tuple.
	"bytes.xor": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.xor() takes exactly 2 arguments (bytes1, bytes2)"}
			}
			data1, errObj := bytesArgToSlice(args[0], "bytes.xor()")
			if errObj != nil {
				return errObj
			}
			data2, errObj := bytesArgToSlice(args[1], "bytes.xor()")
			if errObj != nil {
				return errObj
			}

			if len(data1) != len(data2) {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7010", "bytes.xor() requires byte arrays of equal length"),
				}}
			}

			result := make([]byte, len(data1))
			for i := range data1 {
				result[i] = data1[i] ^ data2[i]
			}
			return &object.ReturnValue{Values: []object.Object{
				sliceToByteArray(result),
				object.NIL,
			}}
		},
	},

	// not performs bitwise NOT (complement) on a byte array.
	// Takes byte array. Returns complemented bytes.
	"bytes.not": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.not() takes exactly 1 argument (bytes)"}
			}
			data, errObj := bytesArgToSlice(args[0], "bytes.not()")
			if errObj != nil {
				return errObj
			}

			result := make([]byte, len(data))
			for i := range data {
				result[i] = ^data[i]
			}
			return sliceToByteArray(result)
		},
	},

	// ============================================================================
	// Utility Functions
	// ============================================================================

	// fill creates a new byte array with all bytes set to a value.
	// Takes byte array and fill value. Returns filled bytes.
	"bytes.fill": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "bytes.fill() takes exactly 2 arguments (bytes, value)"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.fill() requires a byte array as first argument"}
			}
			fillByte, err := getByteValue(args[1], "bytes.fill()")
			if err != nil {
				return err
			}

			elements := make([]object.Object, len(arr.Elements))
			for i := range elements {
				elements[i] = &object.Byte{Value: fillByte}
			}
			return &object.Array{Elements: elements, ElementType: "byte"}
		},
	},

	// copy creates a duplicate of a byte array.
	// Takes byte array. Returns new byte array copy.
	"bytes.copy": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.copy() takes exactly 1 argument (bytes)"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.copy() requires a byte array argument"}
			}

			elements := make([]object.Object, len(arr.Elements))
			copy(elements, arr.Elements)
			return &object.Array{Elements: elements, ElementType: "byte"}
		},
	},

	// zero creates a new byte array with all bytes set to zero.
	// Takes byte array. Returns zeroed bytes (for clearing sensitive data).
	"bytes.zero": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "bytes.zero() takes exactly 1 argument (bytes)"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7002", Message: "bytes.zero() requires a byte array argument"}
			}

			elements := make([]object.Object, len(arr.Elements))
			for i := range elements {
				elements[i] = &object.Byte{Value: 0}
			}
			return &object.Array{Elements: elements, ElementType: "byte"}
		},
	},
}

// Helper function to convert a byte array argument to []byte
func bytesArgToSlice(arg object.Object, funcName string) ([]byte, *object.Error) {
	arr, ok := arg.(*object.Array)
	if !ok {
		return nil, &object.Error{Code: "E7002", Message: fmt.Sprintf("%s requires a byte array argument", funcName)}
	}

	data := make([]byte, len(arr.Elements))
	for i, elem := range arr.Elements {
		switch e := elem.(type) {
		case *object.Byte:
			data[i] = e.Value
		case *object.Integer:
			val := e.Value.Int64()
			if val < 0 || val > 255 {
				return nil, &object.Error{Code: "E7005", Message: fmt.Sprintf("byte value at index %d out of range: %d (must be 0-255)", i, val)}
			}
			data[i] = byte(val)
		default:
			return nil, &object.Error{Code: "E7002", Message: fmt.Sprintf("%s requires a byte array", funcName)}
		}
	}
	return data, nil
}

// Helper function to convert []byte to object.Array of Bytes
func sliceToByteArray(data []byte) *object.Array {
	elements := make([]object.Object, len(data))
	for i, b := range data {
		elements[i] = &object.Byte{Value: b}
	}
	return &object.Array{Elements: elements, ElementType: "byte"}
}

// Helper function to get a byte value from an object
func getByteValue(arg object.Object, funcName string) (uint8, *object.Error) {
	switch v := arg.(type) {
	case *object.Byte:
		return v.Value, nil
	case *object.Integer:
		intVal := v.Value.Int64()
		if intVal < 0 || intVal > 255 {
			return 0, &object.Error{Code: "E3021", Message: fmt.Sprintf("%s byte value %s out of range (0-255)", funcName, v.Value.String())}
		}
		return uint8(intVal), nil
	default:
		return 0, &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires a byte or integer value", funcName)}
	}
}
