package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/object"
)

// MapsBuiltins contains the maps module functions
var MapsBuiltins = map[string]*object.Builtin{
	// is_empty checks if a map has no entries.
	// Takes a map. Returns bool.
	"maps.is_empty": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("%s takes exactly 1 argument", errors.Ident("maps.is_empty()"))
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires a %s", errors.Ident("maps.is_empty()"), errors.TypeExpected("map"))}
			}
			if len(m.Pairs) == 0 {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// contains checks if a key exists in the map.
	// Takes map and key. Returns bool.
	"maps.contains": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("%s takes exactly 2 arguments (%s, key)", errors.Ident("maps.contains()"), errors.TypeExpected("map"))
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("maps.contains()"), errors.TypeExpected("map"))}
			}
			key := args[1]
			// Get() returns false for both unhashable keys and missing keys
			// Only check hashability on failure to provide better error message
			_, exists := m.Get(key)
			if !exists {
				if _, hashOk := object.HashKey(key); !hashOk {
					return &object.Error{Code: "E12001", Message: fmt.Sprintf("%s key must be a %s (string, int, bool, char)", errors.Ident("maps.contains()"), errors.TypeExpected("hashable type"))}
				}
				return object.FALSE
			}
			return object.TRUE
		},
	},

	// get retrieves a value by key, with optional default.
	// Takes map, key, and optional default. Returns value or nil/default.
	"maps.get": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return newError("%s takes 2 or 3 arguments (%s, key[, default])", errors.Ident("maps.get()"), errors.TypeExpected("map"))
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("maps.get()"), errors.TypeExpected("map"))}
			}
			key := args[1]
			// Get() returns false for both unhashable keys and missing keys
			value, exists := m.Get(key)
			if !exists {
				// Check hashability only on failure for better error message
				if _, hashOk := object.HashKey(key); !hashOk {
					return &object.Error{Code: "E12001", Message: fmt.Sprintf("%s key must be a %s (string, int, bool, char)", errors.Ident("maps.get()"), errors.TypeExpected("hashable type"))}
				}
				// Return default value if provided, otherwise NIL
				if len(args) == 3 {
					return args[2]
				}
				return object.NIL
			}
			return value
		},
	},

	// set adds or updates a key-value pair in the map.
	// Takes map, key, and value. Returns nil. Mutates the map.
	"maps.set": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return newError("%s takes exactly 3 arguments (%s, key, value)", errors.Ident("maps.set()"), errors.TypeExpected("map"))
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("maps.set()"), errors.TypeExpected("map"))}
			}
			if !m.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E12002",
				}
			}
			key := args[1]
			// Set() returns false for unhashable keys
			if !m.Set(key, args[2]) {
				return &object.Error{Code: "E12001", Message: fmt.Sprintf("%s key must be a %s (string, int, bool, char)", errors.Ident("maps.set()"), errors.TypeExpected("hashable type"))}
			}
			return object.NIL
		},
	},

	// clear removes all entries from the map.
	// Takes a map. Returns nil. Mutates the map.
	"maps.clear": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("%s takes exactly 1 argument", errors.Ident("maps.clear()"))
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires a %s", errors.Ident("maps.clear()"), errors.TypeExpected("map"))}
			}
			if !m.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E12002",
				}
			}
			m.Pairs = []*object.MapPair{}
			m.Index = make(map[string]int)
			return object.NIL
		},
	},

	// keys returns an array of all keys in the map.
	// Takes a map. Returns array of keys.
	"maps.keys": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("%s takes exactly 1 argument", errors.Ident("maps.keys()"))
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires a %s", errors.Ident("maps.keys()"), errors.TypeExpected("map"))}
			}
			keys := make([]object.Object, len(m.Pairs))
			for i, pair := range m.Pairs {
				keys[i] = pair.Key
			}
			return &object.Array{Elements: keys, Mutable: true, ElementType: m.KeyType}
		},
	},

	// values returns an array of all values in the map.
	// Takes a map. Returns array of values.
	"maps.values": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("%s takes exactly 1 argument", errors.Ident("maps.values()"))
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires a %s", errors.Ident("maps.values()"), errors.TypeExpected("map"))}
			}
			values := make([]object.Object, len(m.Pairs))
			for i, pair := range m.Pairs {
				values[i] = pair.Value
			}
			return &object.Array{Elements: values, Mutable: true, ElementType: m.ValueType}
		},
	},

	// merge combines multiple maps into a new map.
	// Takes two or more maps. Returns new merged map.
	"maps.merge": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("%s takes at least 2 arguments", errors.Ident("maps.merge()"))
			}
			// Create a new map with all entries from all input maps (non-destructive)
			result := object.NewMap()
			// Inherit type info from first map
			if firstMap, ok := args[0].(*object.Map); ok {
				result.KeyType = firstMap.KeyType
				result.ValueType = firstMap.ValueType
			}
			for _, arg := range args {
				m, ok := arg.(*object.Map)
				if !ok {
					return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires %s as arguments", errors.Ident("maps.merge()"), errors.TypeExpected("maps"))}
				}
				for _, pair := range m.Pairs {
					result.Set(pair.Key, pair.Value)
				}
			}
			return result
		},
	},

	// contains_value checks if a value exists in the map.
	// Takes map and value. Returns bool.
	"maps.contains_value": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("%s takes exactly 2 arguments (%s, value)", errors.Ident("maps.contains_value()"), errors.TypeExpected("map"))
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("maps.contains_value()"), errors.TypeExpected("map"))}
			}
			searchValue := args[1]
			for _, pair := range m.Pairs {
				if objectsEqual(pair.Value, searchValue) {
					return object.TRUE
				}
			}
			return object.FALSE
		},
	},

	// equals checks if two maps have identical key-value pairs.
	// Takes two maps. Returns bool.
	"maps.equals": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("%s takes exactly 2 arguments", errors.Ident("maps.equals()"))
			}
			m1, ok1 := args[0].(*object.Map)
			m2, ok2 := args[1].(*object.Map)
			if !ok1 || !ok2 {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires two %s", errors.Ident("maps.equals()"), errors.TypeExpected("maps"))}
			}
			// Check same length
			if len(m1.Pairs) != len(m2.Pairs) {
				return object.FALSE
			}
			// Check all key-value pairs match
			for _, pair := range m1.Pairs {
				val2, exists := m2.Get(pair.Key)
				if !exists || !objectsEqual(pair.Value, val2) {
					return object.FALSE
				}
			}
			return object.TRUE
		},
	},

	// get_or_set returns existing value or sets and returns default.
	// Takes map, key, and default. Returns value. May mutate the map.
	"maps.get_or_set": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return newError("%s takes exactly 3 arguments (%s, key, default)", errors.Ident("maps.get_or_set()"), errors.TypeExpected("map"))
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("maps.get_or_set()"), errors.TypeExpected("map"))}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E12001", Message: fmt.Sprintf("%s key must be a %s (string, int, bool, char)", errors.Ident("maps.get_or_set()"), errors.TypeExpected("hashable type"))}
			}
			// If key exists, return the value
			value, exists := m.Get(key)
			if exists {
				return value
			}
			// Key doesn't exist - set the default and return it
			if !m.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E12002",
				}
			}
			m.Set(key, args[2])
			return args[2]
		},
	},

	// remove deletes a key-value pair from the map.
	// Takes map and key. Returns bool indicating if key existed.
	"maps.remove": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("%s takes exactly 2 arguments (%s, key)", errors.Ident("maps.remove()"), errors.TypeExpected("map"))
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("maps.remove()"), errors.TypeExpected("map"))}
			}
			if !m.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E12002",
				}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E12001", Message: fmt.Sprintf("%s key must be a %s (string, int, bool, char)", errors.Ident("maps.remove()"), errors.TypeExpected("hashable type"))}
			}
			deleted := m.Delete(key)
			if deleted {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// update merges source maps into target map in-place.
	// Takes target map and one or more source maps. Returns nil. Mutates target.
	"maps.update": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("%s takes at least 2 arguments", errors.Ident("maps.update()"))
			}
			// First argument is the target map (destructive update)
			target, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires %s as arguments", errors.Ident("maps.update()"), errors.TypeExpected("maps"))}
			}
			if !target.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E12002",
				}
			}
			// Merge all source maps into target
			for i := 1; i < len(args); i++ {
				source, ok := args[i].(*object.Map)
				if !ok {
					return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires %s as arguments", errors.Ident("maps.update()"), errors.TypeExpected("maps"))}
				}
				for _, pair := range source.Pairs {
					target.Set(pair.Key, pair.Value)
				}
			}
			return object.NIL
		},
	},

	// to_array converts a map to an array of [key, value] pairs.
	// Takes a map. Returns array of arrays.
	"maps.to_array": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("%s takes exactly 1 argument", errors.Ident("maps.to_array()"))
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires a %s", errors.Ident("maps.to_array()"), errors.TypeExpected("map"))}
			}
			// Create array of [key, value] pairs
			pairs := make([]object.Object, len(m.Pairs))
			for i, pair := range m.Pairs {
				pairs[i] = &object.Array{
					Elements: []object.Object{pair.Key, pair.Value},
					Mutable:  true,
				}
			}
			return &object.Array{Elements: pairs, Mutable: true}
		},
	},

	// from_array creates a map from an array of [key, value] pairs.
	// Takes array of pairs. Returns new map.
	"maps.from_array": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("%s takes exactly 1 argument", errors.Ident("maps.from_array()"))
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires an %s", errors.Ident("maps.from_array()"), errors.TypeExpected("array"))}
			}
			// Each element should be an array of [key, value]
			result := object.NewMap()
			for i, elem := range arr.Elements {
				pair, ok := elem.(*object.Array)
				if !ok || len(pair.Elements) != 2 {
					return &object.Error{
						Code:    "E12004",
						Message: fmt.Sprintf("%s element %d must be a [key, value] pair", errors.Ident("maps.from_array()"), i),
					}
				}
				key := pair.Elements[0]
				if _, hashOk := object.HashKey(key); !hashOk {
					return &object.Error{
						Code:    "E12002",
						Message: fmt.Sprintf("%s key at index %d must be a %s (string, int, bool, char)", errors.Ident("maps.from_array()"), i, errors.TypeExpected("hashable type")),
					}
				}
				result.Set(key, pair.Elements[1])
			}
			return result
		},
	},

	// invert swaps keys and values, creating a new map.
	// Takes a map. Returns new map with values as keys.
	"maps.invert": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("%s takes exactly 1 argument", errors.Ident("maps.invert()"))
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: fmt.Sprintf("%s requires a %s", errors.Ident("maps.invert()"), errors.TypeExpected("map"))}
			}
			// Create new map with values as keys and keys as values
			result := object.NewMap()
			// Swap key and value types
			result.KeyType = m.ValueType
			result.ValueType = m.KeyType
			for _, pair := range m.Pairs {
				// Value must be hashable to become a key
				if _, hashOk := object.HashKey(pair.Value); !hashOk {
					return &object.Error{
						Code:    "E12005",
						Message: fmt.Sprintf("%s value %s is not hashable and cannot become a key", errors.Ident("maps.invert()"), pair.Value.Inspect()),
					}
				}
				result.Set(pair.Value, pair.Key)
			}
			return result
		},
	},
}
