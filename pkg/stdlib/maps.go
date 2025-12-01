package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"

	"github.com/marshallburns/ez/pkg/object"
)

// MapsBuiltins contains the maps module functions
var MapsBuiltins = map[string]*object.Builtin{
	"maps.is_empty": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("maps.is_empty() takes exactly 1 argument")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.is_empty() requires a map"}
			}
			if len(m.Pairs) == 0 {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	"maps.has": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("maps.has() takes exactly 2 arguments (map, key)")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.has() requires a map as first argument"}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E12001", Message: "maps.has() key must be a hashable type (string, int, bool, char)"}
			}
			_, exists := m.Get(key)
			if exists {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	"maps.get": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return newError("maps.get() takes 2 or 3 arguments (map, key[, default])")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.get() requires a map as first argument"}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E12001", Message: "maps.get() key must be a hashable type (string, int, bool, char)"}
			}
			value, exists := m.Get(key)
			if exists {
				return value
			}
			// Return default value if provided, otherwise NIL
			if len(args) == 3 {
				return args[2]
			}
			return object.NIL
		},
	},

	"maps.set": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return newError("maps.set() takes exactly 3 arguments (map, key, value)")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.set() requires a map as first argument"}
			}
			if !m.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E12003",
				}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E12001", Message: "maps.set() key must be a hashable type (string, int, bool, char)"}
			}
			m.Set(key, args[2])
			return object.NIL
		},
	},

	"maps.delete": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("maps.delete() takes exactly 2 arguments (map, key)")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.delete() requires a map as first argument"}
			}
			if !m.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E12003",
				}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E12001", Message: "maps.delete() key must be a hashable type (string, int, bool, char)"}
			}
			deleted := m.Delete(key)
			if deleted {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	"maps.clear": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("maps.clear() takes exactly 1 argument")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.clear() requires a map"}
			}
			if !m.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E12003",
				}
			}
			m.Pairs = []*object.MapPair{}
			m.Index = make(map[string]int)
			return object.NIL
		},
	},

	"maps.keys": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("maps.keys() takes exactly 1 argument")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.keys() requires a map"}
			}
			keys := make([]object.Object, len(m.Pairs))
			for i, pair := range m.Pairs {
				keys[i] = pair.Key
			}
			return &object.Array{Elements: keys, Mutable: true}
		},
	},

	"maps.values": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("maps.values() takes exactly 1 argument")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.values() requires a map"}
			}
			values := make([]object.Object, len(m.Pairs))
			for i, pair := range m.Pairs {
				values[i] = pair.Value
			}
			return &object.Array{Elements: values, Mutable: true}
		},
	},

	"maps.copy": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("maps.copy() takes exactly 1 argument")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.copy() requires a map"}
			}
			// Create a shallow copy of the map
			newMap := object.NewMap()
			for _, pair := range m.Pairs {
				newMap.Set(pair.Key, pair.Value)
			}
			return newMap
		},
	},

	"maps.merge": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("maps.merge() takes at least 2 arguments")
			}
			// Create a new map with all entries from all input maps (non-destructive)
			result := object.NewMap()
			for _, arg := range args {
				m, ok := arg.(*object.Map)
				if !ok {
					return &object.Error{Code: "E7007", Message: "maps.merge() requires maps as arguments"}
				}
				for _, pair := range m.Pairs {
					result.Set(pair.Key, pair.Value)
				}
			}
			return result
		},
	},

	// has_key is an alias for has
	"maps.has_key": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("maps.has_key() takes exactly 2 arguments (map, key)")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.has_key() requires a map as first argument"}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E12001", Message: "maps.has_key() key must be a hashable type (string, int, bool, char)"}
			}
			_, exists := m.Get(key)
			if exists {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	"maps.has_value": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("maps.has_value() takes exactly 2 arguments (map, value)")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.has_value() requires a map as first argument"}
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

	"maps.equals": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("maps.equals() takes exactly 2 arguments")
			}
			m1, ok1 := args[0].(*object.Map)
			m2, ok2 := args[1].(*object.Map)
			if !ok1 || !ok2 {
				return &object.Error{Code: "E7007", Message: "maps.equals() requires two maps"}
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

	"maps.get_or_set": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return newError("maps.get_or_set() takes exactly 3 arguments (map, key, default)")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.get_or_set() requires a map as first argument"}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E12001", Message: "maps.get_or_set() key must be a hashable type (string, int, bool, char)"}
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
					Code:    "E12003",
				}
			}
			m.Set(key, args[2])
			return args[2]
		},
	},

	// remove is an alias for delete
	"maps.remove": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("maps.remove() takes exactly 2 arguments (map, key)")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.remove() requires a map as first argument"}
			}
			if !m.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E12003",
				}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E12001", Message: "maps.remove() key must be a hashable type (string, int, bool, char)"}
			}
			deleted := m.Delete(key)
			if deleted {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	"maps.update": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("maps.update() takes at least 2 arguments")
			}
			// First argument is the target map (destructive update)
			target, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.update() requires maps as arguments"}
			}
			if !target.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E12003",
				}
			}
			// Merge all source maps into target
			for i := 1; i < len(args); i++ {
				source, ok := args[i].(*object.Map)
				if !ok {
					return &object.Error{Code: "E7007", Message: "maps.update() requires maps as arguments"}
				}
				for _, pair := range source.Pairs {
					target.Set(pair.Key, pair.Value)
				}
			}
			return object.NIL
		},
	},

	"maps.to_array": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("maps.to_array() takes exactly 1 argument")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.to_array() requires a map"}
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

	"maps.from_array": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("maps.from_array() takes exactly 1 argument")
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.from_array() requires an array"}
			}
			// Each element should be an array of [key, value]
			result := object.NewMap()
			for i, elem := range arr.Elements {
				pair, ok := elem.(*object.Array)
				if !ok || len(pair.Elements) != 2 {
					return &object.Error{
						Code:    "E12004",
						Message: fmt.Sprintf("maps.from_array() element %d must be a [key, value] pair", i),
					}
				}
				key := pair.Elements[0]
				if _, hashOk := object.HashKey(key); !hashOk {
					return &object.Error{
						Code:    "E12002",
						Message: fmt.Sprintf("maps.from_array() key at index %d must be a hashable type (string, int, bool, char)", i),
					}
				}
				result.Set(key, pair.Elements[1])
			}
			return result
		},
	},

	"maps.invert": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("maps.invert() takes exactly 1 argument")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "maps.invert() requires a map"}
			}
			// Create new map with values as keys and keys as values
			result := object.NewMap()
			for _, pair := range m.Pairs {
				// Value must be hashable to become a key
				if _, hashOk := object.HashKey(pair.Value); !hashOk {
					return &object.Error{
						Code:    "E12005",
						Message: fmt.Sprintf("maps.invert() value %s is not hashable and cannot become a key", pair.Value.Inspect()),
					}
				}
				result.Set(pair.Value, pair.Key)
			}
			return result
		},
	},
}
