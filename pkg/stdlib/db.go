package stdlib

import (
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/object"
)

// DBBuiltins contains the db module functions for database operations
var DBBuiltins = map[string]*object.Builtin{
	// ============================================================================
	// Database Management
	// ============================================================================

	// open opens or creates a database file at the given path.
	// Takes path string (.ezdb). Returns (Database, Error) tuple.
	"db.open": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("db.open()"))}
			}

			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s path", errors.Ident("db.open()"), errors.TypeExpected("string"))}
			}

			if err := validatePath(path.Value, "db.open()"); err != nil {
				return &object.Error{Code: "E17001", Message: fmt.Sprintf("%s requires valid path", errors.Ident("db.open()"))}
			}

			if !strings.HasSuffix(path.Value, ".ezdb") {
				return &object.Error{Code: "E17001", Message: fmt.Sprintf("%s requires a `.ezdb` file", errors.Ident("db.open()"))}
			}

			info, statErr := os.Stat(path.Value)
			if statErr == nil && info.IsDir() {
				return &object.Error{Code: "E17002", Message: fmt.Sprintf("%s cannot read directory as file", errors.Ident("db.open()"))}
			}

			if os.IsNotExist(statErr) {
				perms := os.FileMode(0644)
				if err := atomicWriteFile(path.Value, []byte("{}"), perms); err != nil {
					return &object.Error{Code: "E17001", Message: fmt.Sprintf("%s failed to open database file", errors.Ident("db.open()"))}
				}

				return &object.ReturnValue{Values: []object.Object{
					&object.Database{
						Path:     *path,
						Store:    *object.NewMap(),
						IsClosed: object.Boolean{Value: false},
					},
					object.NIL,
				}}
			}

			content, err := os.ReadFile(path.Value)
			if err != nil {
				return &object.Error{Code: "E17002", Message: fmt.Sprintf("%s could not read database file", errors.Ident("db.open()"))}
			}

			// Handle empty files the same as non-existent files - initialize with empty map
			if len(content) == 0 {
				return &object.ReturnValue{Values: []object.Object{
					&object.Database{
						Path:     *path,
						Store:    *object.NewMap(),
						IsClosed: object.Boolean{Value: false},
					},
					object.NIL,
				}}
			}

			result, err := decodeFromJSON(string(content))
			if err != nil {
				return &object.Error{Code: "E17004", Message: fmt.Sprintf("%s database file is corrupted", errors.Ident("db.open()"))}
			}

			dbContent, ok := result.(*object.Map)
			if !ok {
				if _, isArray := result.(*object.Array); isArray {
					return &object.Error{Code: "E17004", Message: fmt.Sprintf("%s database file must contain a JSON object, not an array", errors.Ident("db.open()"))}
				}
				return &object.Error{Code: "E17004", Message: fmt.Sprintf("%s database file is corrupted", errors.Ident("db.open()"))}
			}

			// Sort keys alphabetically on load for consistent ordering
			sort.Slice(dbContent.Pairs, func(i, j int) bool {
				keyI := dbContent.Pairs[i].Key.(*object.String).Value
				keyJ := dbContent.Pairs[j].Key.(*object.String).Value
				return keyI < keyJ
			})
			// Rebuild index after sorting
			for i, pair := range dbContent.Pairs {
				hash, _ := object.HashKey(pair.Key)
				dbContent.Index[hash] = i
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Database{
					Path:     *path,
					Store:    *dbContent,
					IsClosed: object.Boolean{Value: false},
				},
				object.NIL,
			}}
		},
	},

	// close closes a database and saves its state to disk.
	// Takes Database. Returns nil on success, prevents further operations.
	"db.close": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("db.close()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s object as argument", errors.Ident("db.close()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.close()"))}
			}

			jsonRes, err := encodeToJSON(&db.Store, make(map[uintptr]bool))
			if err != nil {
				return &object.Error{Code: "E17003", Message: fmt.Sprintf("%s database contents not json encodable", errors.Ident("db.close()"))}
			}

			perms := os.FileMode(0644)
			if err := atomicWriteFile(db.Path.Value, []byte(jsonRes), perms); err != nil {
				return &object.Error{Code: "E17003", Message: fmt.Sprintf("%s failed to write to database", errors.Ident("db.close()"))}
			}

			db.IsClosed = object.Boolean{Value: true}

			return &object.Nil{}
		},
	},

	// save manually persists the database state to disk.
	// Takes Database. Returns nil on success.
	"db.save": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("db.save()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s object as argument", errors.Ident("db.save()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.save()"))}
			}

			jsonRes, err := encodeToJSON(&db.Store, make(map[uintptr]bool))
			if err != nil {
				return &object.Error{Code: "E17003", Message: fmt.Sprintf("%s database contents not json encodable", errors.Ident("db.save()"))}
			}

			perms := os.FileMode(0644)
			if err := atomicWriteFile(db.Path.Value, []byte(jsonRes), perms); err != nil {
				return &object.Error{Code: "E17003", Message: fmt.Sprintf("%s failed to write to database", errors.Ident("db.save()"))}
			}

			return &object.Nil{}
		},
	},

	// ============================================================================
	// Database Operations
	// ============================================================================

	// set stores a key-value pair in the database.
	// Takes Database, key string, value string. Returns nil.
	"db.set": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 3 arguments", errors.Ident("db.set()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s object as first argument", errors.Ident("db.set()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.set()"))}
			}

			key, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("db.set()"), errors.TypeExpected("String"))}
			}

			val, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s as third argument", errors.Ident("db.set()"), errors.TypeExpected("String"))}
			}

			db.Store.Set(key, val)

			return &object.Nil{}
		},
	},

	// get retrieves a value by key from the database.
	// Takes Database, key. Returns (string, bool) where bool is found status.
	"db.get": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("db.get()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s object as first argument", errors.Ident("db.get()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.get()"))}
			}

			key, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("db.get()"), errors.TypeExpected("String"))}
			}

			val, exists := db.Store.Get(key)
			if exists {
				return &object.ReturnValue{Values: []object.Object{
					val.(*object.String),
					&object.Boolean{Value: true},
				}}
			} else {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					&object.Boolean{Value: false},
				}}
			}
		},
	},

	// remove deletes a key-value pair from the database.
	// Takes Database, key. Returns true if key existed.
	"db.remove": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("db.remove()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s object as first argument", errors.Ident("db.remove()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.remove()"))}
			}

			key, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("db.remove()"), errors.TypeExpected("String"))}
			}

			deleted := db.Store.Delete(key)
			return &object.Boolean{Value: deleted}
		},
	},

	// contains checks if a key exists in the database.
	// Takes Database, key. Returns true if key exists.
	"db.contains": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("db.contains()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s object as first argument", errors.Ident("db.contains()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.contains()"))}
			}

			key, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("db.contains()"), errors.TypeExpected("String"))}
			}

			_, exists := db.Store.Get(key)
			return &object.Boolean{Value: exists}
		},
	},

	// keys returns an array of all keys in the database.
	// Takes Database. Returns [string] array.
	"db.keys": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 arguments", errors.Ident("db.keys()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s object as first argument", errors.Ident("db.keys()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.keys()"))}
			}

			keys := &object.Array{
				Elements:    make([]object.Object, 0, len(db.Store.Pairs)),
				ElementType: "string",
			}
			for _, pair := range db.Store.Pairs {
				keys.Elements = append(keys.Elements, pair.Key)
			}

			return keys
		},
	},

	// values returns an array of all values in the database.
	// Takes Database. Returns array of values.
	"db.values": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("db.values()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("db.values()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.values()"))}
			}

			values := &object.Array{
				Elements:    make([]object.Object, 0, len(db.Store.Pairs)),
				ElementType: "any",
			}
			for _, pair := range db.Store.Pairs {
				values.Elements = append(values.Elements, pair.Value)
			}

			return values
		},
	},

	// entries returns all key-value pairs as Entry structs.
	// Takes Database. Returns array of {key, value} structs.
	"db.entries": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("db.entries()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s as first argument", errors.Ident("db.entries()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.entries()"))}
			}

			entries := &object.Array{
				Elements:    make([]object.Object, 0, len(db.Store.Pairs)),
				ElementType: "Entry",
			}
			for _, pair := range db.Store.Pairs {
				entry := &object.Struct{
					TypeName: "Entry",
					Mutable:  false,
					Fields: map[string]object.Object{
						"key":   pair.Key,
						"value": pair.Value,
					},
				}
				entries.Elements = append(entries.Elements, entry)
			}

			return entries
		},
	},

	// prefix returns all keys that start with a given prefix.
	// Takes Database, prefix string. Returns [string] array.
	"db.prefix": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("db.prefix()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s object as first argument", errors.Ident("db.prefix()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.prefix()"))}
			}

			prefix, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("db.prefix()"), errors.TypeExpected("String"))}
			}

			keys := &object.Array{
				Elements:    make([]object.Object, 0, len(db.Store.Pairs)/4+1),
				ElementType: "string",
			}
			for _, pair := range db.Store.Pairs {
				key, ok := pair.Key.(*object.String)
				if ok && strings.HasPrefix(key.Value, prefix.Value) {
					keys.Elements = append(keys.Elements, key)
				}
			}

			return keys
		},
	},

	// count returns the number of key-value pairs in the database.
	// Takes Database. Returns integer count.
	"db.count": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 arguments", errors.Ident("db.count()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s object as first argument", errors.Ident("db.count()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.count()"))}
			}

			count := len(db.Store.Pairs)
			return &object.Integer{Value: big.NewInt(int64(count))}
		},
	},

	// clear removes all key-value pairs from the database.
	// Takes Database. Returns nil.
	"db.clear": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("db.clear()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s object as argument", errors.Ident("db.clear()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.clear()"))}
			}

			db.Store = *object.NewMap()

			return &object.Nil{}
		},
	},

	// exists checks if a database file exists at the given path.
	// Takes path string. Returns true if file exists.
	"db.exists": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("db.exists()"))}
			}

			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s path", errors.Ident("db.exists()"), errors.TypeExpected("string"))}
			}

			if !strings.HasSuffix(path.Value, ".ezdb") {
				return &object.Error{Code: "E17001", Message: fmt.Sprintf("%s requires a `.ezdb` file", errors.Ident("db.exists()"))}
			}

			_, err := os.Stat(path.Value)
			return &object.Boolean{Value: err == nil}
		},
	},

	// update_key_name renames a key in the database.
	// Takes Database, old key, new key. Returns true if old key existed.
	"db.update_key_name": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 3 arguments", errors.Ident("db.update_key_name()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s object as first argument", errors.Ident("db.update_key_name()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.update_key_name()"))}
			}

			oldKey, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s as second argument", errors.Ident("db.update_key_name()"), errors.TypeExpected("String"))}
			}

			newKey, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s as third argument", errors.Ident("db.update_key_name()"), errors.TypeExpected("String"))}
			}

			// Get the value for the old key
			val, exists := db.Store.Get(oldKey)
			if !exists {
				return &object.Boolean{Value: false}
			}

			// Delete old key and set new key with the value
			db.Store.Delete(oldKey)
			db.Store.Set(newKey, val)

			return &object.Boolean{Value: true}
		},
	},

	// sort reorders database entries by the specified sort order.
	// Takes Database, order constant (e.g., db.ALPHA). Returns nil.
	"db.sort": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("db.sort()"))}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a %s object as first argument", errors.Ident("db.sort()"), errors.TypeExpected("Database"))}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: fmt.Sprintf("%s cannot operate on closed database", errors.Ident("db.sort()"))}
			}

			order, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s requires a sort order constant as second argument (e.g., %s)", errors.Ident("db.sort()"), errors.Ident("db.ALPHA"))}
			}

			switch order.Value.Int64() {
			case 0: // db.ALPHA - keys A-Z
				sort.Slice(db.Store.Pairs, func(i, j int) bool {
					keyI := db.Store.Pairs[i].Key.(*object.String).Value
					keyJ := db.Store.Pairs[j].Key.(*object.String).Value
					return keyI < keyJ
				})
			case 1: // db.ALPHA_DESC - keys Z-A
				sort.Slice(db.Store.Pairs, func(i, j int) bool {
					keyI := db.Store.Pairs[i].Key.(*object.String).Value
					keyJ := db.Store.Pairs[j].Key.(*object.String).Value
					return keyI > keyJ
				})
			case 2: // db.VALUE_ALPHA - values A-Z
				sort.Slice(db.Store.Pairs, func(i, j int) bool {
					valI := db.Store.Pairs[i].Value.(*object.String).Value
					valJ := db.Store.Pairs[j].Value.(*object.String).Value
					return valI < valJ
				})
			case 3: // db.VALUE_ALPHA_DESC - values Z-A
				sort.Slice(db.Store.Pairs, func(i, j int) bool {
					valI := db.Store.Pairs[i].Value.(*object.String).Value
					valJ := db.Store.Pairs[j].Value.(*object.String).Value
					return valI > valJ
				})
			case 4: // db.KEY_LEN - shortest keys first
				sort.Slice(db.Store.Pairs, func(i, j int) bool {
					keyI := db.Store.Pairs[i].Key.(*object.String).Value
					keyJ := db.Store.Pairs[j].Key.(*object.String).Value
					return len(keyI) < len(keyJ)
				})
			case 5: // db.KEY_LEN_DESC - longest keys first
				sort.Slice(db.Store.Pairs, func(i, j int) bool {
					keyI := db.Store.Pairs[i].Key.(*object.String).Value
					keyJ := db.Store.Pairs[j].Key.(*object.String).Value
					return len(keyI) > len(keyJ)
				})
			case 6: // db.VALUE_LEN - shortest values first
				sort.Slice(db.Store.Pairs, func(i, j int) bool {
					valI := db.Store.Pairs[i].Value.(*object.String).Value
					valJ := db.Store.Pairs[j].Value.(*object.String).Value
					return len(valI) < len(valJ)
				})
			case 7: // db.VALUE_LEN_DESC - longest values first
				sort.Slice(db.Store.Pairs, func(i, j int) bool {
					valI := db.Store.Pairs[i].Value.(*object.String).Value
					valJ := db.Store.Pairs[j].Value.(*object.String).Value
					return len(valI) > len(valJ)
				})
			case 8: // db.NUMERIC - keys numerically ascending
				sort.Slice(db.Store.Pairs, func(i, j int) bool {
					keyI := db.Store.Pairs[i].Key.(*object.String).Value
					keyJ := db.Store.Pairs[j].Key.(*object.String).Value
					numI, errI := strconv.ParseFloat(keyI, 64)
					numJ, errJ := strconv.ParseFloat(keyJ, 64)
					if errI != nil || errJ != nil {
						return keyI < keyJ // fall back to string comparison
					}
					return numI < numJ
				})
			case 9: // db.NUMERIC_DESC - keys numerically descending
				sort.Slice(db.Store.Pairs, func(i, j int) bool {
					keyI := db.Store.Pairs[i].Key.(*object.String).Value
					keyJ := db.Store.Pairs[j].Key.(*object.String).Value
					numI, errI := strconv.ParseFloat(keyI, 64)
					numJ, errJ := strconv.ParseFloat(keyJ, 64)
					if errI != nil || errJ != nil {
						return keyI > keyJ // fall back to string comparison
					}
					return numI > numJ
				})
			default:
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s invalid order constant", errors.Ident("db.sort()"))}
			}

			// Rebuild index after sorting
			for i, pair := range db.Store.Pairs {
				hash, _ := object.HashKey(pair.Key)
				db.Store.Index[hash] = i
			}

			return &object.Nil{}
		},
	},

	// ============================================================================
	// Database Constants
	// ============================================================================

	// ALPHA is a sort order constant for keys alphabetically A-Z.
	// Use with db.sort().
	"db.ALPHA": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(0)}
		},
		IsConstant: true,
	},

	// ALPHA_DESC is a sort order constant for keys alphabetically Z-A.
	// Use with db.sort().
	"db.ALPHA_DESC": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(1)}
		},
		IsConstant: true,
	},

	// VALUE_ALPHA is a sort order constant for values alphabetically A-Z.
	// Use with db.sort().
	"db.VALUE_ALPHA": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(2)}
		},
		IsConstant: true,
	},

	// VALUE_ALPHA_DESC is a sort order constant for values alphabetically Z-A.
	// Use with db.sort().
	"db.VALUE_ALPHA_DESC": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(3)}
		},
		IsConstant: true,
	},

	// KEY_LEN is a sort order constant for shortest keys first.
	// Use with db.sort().
	"db.KEY_LEN": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(4)}
		},
		IsConstant: true,
	},

	// KEY_LEN_DESC is a sort order constant for longest keys first.
	// Use with db.sort().
	"db.KEY_LEN_DESC": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(5)}
		},
		IsConstant: true,
	},

	// VALUE_LEN is a sort order constant for shortest values first.
	// Use with db.sort().
	"db.VALUE_LEN": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(6)}
		},
		IsConstant: true,
	},

	// VALUE_LEN_DESC is a sort order constant for longest values first.
	// Use with db.sort().
	"db.VALUE_LEN_DESC": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(7)}
		},
		IsConstant: true,
	},

	// NUMERIC is a sort order constant for keys numerically ascending.
	// Use with db.sort().
	"db.NUMERIC": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(8)}
		},
		IsConstant: true,
	},

	// NUMERIC_DESC is a sort order constant for keys numerically descending.
	// Use with db.sort().
	"db.NUMERIC_DESC": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(9)}
		},
		IsConstant: true,
	},
}
