package stdlib

import (
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/marshallburns/ez/pkg/object"
)

// DBBuiltings contains the db module functions for database operations
var DBBuiltins = map[string]*object.Builtin{
	// ============================================================================
	// Database Management
	// ============================================================================

	// Opens database using provided path
	// Creates new database file if it does not exist
	// Returns (database, error) tuple - error is nil on success
	"db.open": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "db.open() takes exactly 1 argument"}
			}

			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "db.open() requires a string path"}
			}

			if err := validatePath(path.Value, "db.open()"); err != nil {
				return &object.Error{Code: "E17001", Message: "db.open() requires valid path"}
			}

			if !strings.HasSuffix(path.Value, ".ezdb") {
				return &object.Error{Code: "E17001", Message: "db.open() requires a `.ezdb` file"}
			}

			info, statErr := os.Stat(path.Value)
			if statErr == nil && info.IsDir() {
				return &object.Error{Code: "E17002", Message: "db.open(): cannot read directory as file"}
			}

			if os.IsNotExist(statErr) {
				perms := os.FileMode(0644)
				if err := atomicWriteFile(path.Value, []byte("{}"), perms); err != nil {
					return &object.Error{Code: "E17001", Message: "db.open() failed to open database file"}
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
				return &object.Error{Code: "E17002", Message: "db.open(): could not read database file"}
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
				return &object.Error{Code: "E17004", Message: "db.open(): database file is corrupted"}
			}

			dbContent, ok := result.(*object.Map)
			if !ok {
				if _, isArray := result.(*object.Array); isArray {
					return &object.Error{Code: "E17004", Message: "db.open(): database file must contain a JSON object, not an array"}
				}
				return &object.Error{Code: "E17004", Message: "db.open(): database file is corrupted"}
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

	// Closes the database and prevents further actions on the database
	// Saves state of database to disk when closing
	// Returns (error) - error is nil on success
	"db.close": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "db.close() takes exactly 1 argument"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.close() requires a Database object as argument"}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: "db.close() cannot operate on closed database"}
			}

			jsonRes, err := encodeToJSON(&db.Store, make(map[uintptr]bool))
			if err != nil {
				return &object.Error{Code: "E17003", Message: "db.close() database contents not json encodable"}
			}

			perms := os.FileMode(0644)
			if err := atomicWriteFile(db.Path.Value, []byte(jsonRes), perms); err != nil {
				return &object.Error{Code: "E17003", Message: "db.close() failed to write to database"}
			}

			db.IsClosed = object.Boolean{Value: true}

			return &object.Nil{}
		},
	},

	// Manual saving of database to disk
	// Returns (error) - error is nil on success
	"db.save": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "db.save() takes exactly 1 argument"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.save() requires a Database object as argument"}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: "db.save() cannot operate on closed database"}
			}

			jsonRes, err := encodeToJSON(&db.Store, make(map[uintptr]bool))
			if err != nil {
				return &object.Error{Code: "E17003", Message: "db.save() database contents not json encodable"}
			}

			perms := os.FileMode(0644)
			if err := atomicWriteFile(db.Path.Value, []byte(jsonRes), perms); err != nil {
				return &object.Error{Code: "E17003", Message: "db.save() failed to write to database"}
			}

			return &object.Nil{}
		},
	},

	// ============================================================================
	// Database Operations
	// ============================================================================

	// Sets a key value pair in database
	// Returns nothing
	"db.set": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: "db.set() takes exactly 3 arguments"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.set() requires a Database object as first argument"}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: "db.set() cannot operate on closed database"}
			}

			key, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.set() requires a String as second argument"}
			}

			val, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.set() requires a String as third argument"}
			}

			db.Store.Set(key, val)

			return &object.Nil{}
		},
	},

	// Returns value associated with key if it exists
	// Returns (string, bool) - string is empty if key does not exist
	"db.get": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "db.get() takes exactly 2 arguments"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.get() requires a Database object as first argument"}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: "db.get() cannot operate on closed database"}
			}

			key, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.get() requires a String as second argument"}
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

	// Removes key value pair from database
	// Returns (bool) - false if key does not exist
	"db.remove": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "db.remove() takes exactly 2 arguments"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.remove() requires a Database object as first argument"}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: "db.remove() cannot operate on closed database"}
			}

			key, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.remove() requires a String as second argument"}
			}

			deleted := db.Store.Delete(key)
			return &object.Boolean{Value: deleted}
		},
	},

	// Checks if key exists in database
	// Returns (bool)
	"db.contains": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "db.contains() takes exactly 2 arguments"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.contains() requires a Database object as first argument"}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: "db.contains() cannot operate on closed database"}
			}

			key, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.contains() requires a String as second argument"}
			}

			_, exists := db.Store.Get(key)
			return &object.Boolean{Value: exists}
		},
	},

	// Fetches array of keys present in database
	// Returns ([string])
	"db.keys": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "db.keys() takes exactly 1 arguments"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.keys() requires a Database object as first argument"}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: "db.keys() cannot operate on closed database"}
			}

			var keys object.Array
			keys.ElementType = "string"
			for _, pair := range db.Store.Pairs {
				keys.Elements = append(keys.Elements, pair.Key)
			}

			return &keys
		},
	},

	// Fetches keys with prefix in database
	// Returns ([string])
	"db.prefix": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "db.prefix() takes exactly 2 arguments"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.prefix() requires a Database object as first argument"}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: "db.prefix() cannot operate on closed database"}
			}

			prefix, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "db.prefix() requires a String as second argument"}
			}

			var keys object.Array
			keys.ElementType = "string"
			for _, pair := range db.Store.Pairs {
				key, ok := pair.Key.(*object.String)
				if ok && strings.HasPrefix(key.Value, prefix.Value) {
					keys.Elements = append(keys.Elements, key)
				}
			}

			return &keys
		},
	},

	// Number of key value pairs in database
	// Returns (int)
	"db.count": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "db.count() takes exactly 1 arguments"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.count() requires a Database object as first argument"}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: "db.count() cannot operate on closed database"}
			}

			count := len(db.Store.Pairs)
			return &object.Integer{Value: big.NewInt(int64(count))}
		},
	},

	// Clears all key value pairs in database
	// Returns nothing
	"db.clear": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "db.clear() takes exactly 1 argument"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.clear() requires a Database object as argument"}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: "db.clear() cannot operate on closed database"}
			}

			db.Store = *object.NewMap()

			return &object.Nil{}
		},
	},

	// Checks if a database file exists at the given path
	// Returns (bool)
	"db.exists": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "db.exists() takes exactly 1 argument"}
			}

			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "db.exists() requires a string path"}
			}

			if !strings.HasSuffix(path.Value, ".ezdb") {
				return &object.Error{Code: "E17001", Message: "db.exists() requires a `.ezdb` file"}
			}

			_, err := os.Stat(path.Value)
			return &object.Boolean{Value: err == nil}
		},
	},

	// Renames a key in the database
	// Returns (bool) - false if old key does not exist
	"db.update_key_name": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: "db.update_key_name() takes exactly 3 arguments"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.update_key_name() requires a Database object as first argument"}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: "db.update_key_name() cannot operate on closed database"}
			}

			oldKey, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.update_key_name() requires a String as second argument"}
			}

			newKey, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.update_key_name() requires a String as third argument"}
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

	// Sorts database keys by specified order
	// Returns nothing
	"db.sort": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "db.sort() takes exactly 2 arguments"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.sort() requires a Database object as first argument"}
			}

			if db.IsClosed.Value {
				return &object.Error{Code: "E17005", Message: "db.sort() cannot operate on closed database"}
			}

			order, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.sort() requires a sort order constant as second argument (e.g., db.ALPHA)"}
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
				return &object.Error{Code: "E7003", Message: "db.sort() invalid order constant"}
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

	// Sort order: keys alphabetically A-Z
	"db.ALPHA": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(0)}
		},
		IsConstant: true,
	},

	// Sort order: keys alphabetically Z-A
	"db.ALPHA_DESC": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(1)}
		},
		IsConstant: true,
	},

	// Sort order: values alphabetically A-Z
	"db.VALUE_ALPHA": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(2)}
		},
		IsConstant: true,
	},

	// Sort order: values alphabetically Z-A
	"db.VALUE_ALPHA_DESC": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(3)}
		},
		IsConstant: true,
	},

	// Sort order: shortest keys first
	"db.KEY_LEN": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(4)}
		},
		IsConstant: true,
	},

	// Sort order: longest keys first
	"db.KEY_LEN_DESC": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(5)}
		},
		IsConstant: true,
	},

	// Sort order: shortest values first
	"db.VALUE_LEN": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(6)}
		},
		IsConstant: true,
	},

	// Sort order: longest values first
	"db.VALUE_LEN_DESC": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(7)}
		},
		IsConstant: true,
	},

	// Sort order: keys numerically ascending
	"db.NUMERIC": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(8)}
		},
		IsConstant: true,
	},

	// Sort order: keys numerically descending
	"db.NUMERIC_DESC": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(9)}
		},
		IsConstant: true,
	},
}
