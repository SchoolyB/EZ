package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"github.com/marshallburns/ez/pkg/object"
)

// MapsBuiltins contains the maps module functions
var MapsBuiltins = map[string]*object.Builtin{
	"maps.len": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("maps.len() takes exactly 1 argument")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E9025", Message: "maps.len() requires a map"}
			}
			return &object.Integer{Value: int64(len(m.Pairs))}
		},
	},

	"maps.is_empty": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("maps.is_empty() takes exactly 1 argument")
			}
			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E9025", Message: "maps.is_empty() requires a map"}
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
				return &object.Error{Code: "E9025", Message: "maps.has() requires a map as first argument"}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E9020", Message: "maps.has() key must be a hashable type (string, int, bool, char)"}
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
				return &object.Error{Code: "E9025", Message: "maps.get() requires a map as first argument"}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E9020", Message: "maps.get() key must be a hashable type (string, int, bool, char)"}
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
				return &object.Error{Code: "E9025", Message: "maps.set() requires a map as first argument"}
			}
			if !m.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E4005",
				}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E9020", Message: "maps.set() key must be a hashable type (string, int, bool, char)"}
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
				return &object.Error{Code: "E9025", Message: "maps.delete() requires a map as first argument"}
			}
			if !m.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E4005",
				}
			}
			key := args[1]
			if _, hashOk := object.HashKey(key); !hashOk {
				return &object.Error{Code: "E9020", Message: "maps.delete() key must be a hashable type (string, int, bool, char)"}
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
				return &object.Error{Code: "E9025", Message: "maps.clear() requires a map"}
			}
			if !m.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E4005",
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
				return &object.Error{Code: "E9025", Message: "maps.keys() requires a map"}
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
				return &object.Error{Code: "E9025", Message: "maps.values() requires a map"}
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
				return &object.Error{Code: "E9025", Message: "maps.copy() requires a map"}
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
			// First argument is the target map
			target, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E9025", Message: "maps.merge() requires maps as arguments"}
			}
			if !target.Mutable {
				return &object.Error{
					Message: "cannot modify immutable map (declared as const)",
					Code:    "E4005",
				}
			}
			// Merge all source maps into target
			for i := 1; i < len(args); i++ {
				source, ok := args[i].(*object.Map)
				if !ok {
					return &object.Error{Code: "E9025", Message: "maps.merge() requires maps as arguments"}
				}
				for _, pair := range source.Pairs {
					target.Set(pair.Key, pair.Value)
				}
			}
			return object.NIL
		},
	},
}
