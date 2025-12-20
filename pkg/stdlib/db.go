package stdlib

import (
	"math/big"
	"os"
	"strings"

	"github.com/marshallburns/ez/pkg/object"
)

var DbBuiltins = map[string]*object.Builtin{
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
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					createDBError("E17001",  "db.open() requires a valid path"),
				}}
			}

			if !strings.HasSuffix(path.Value, ".ezdb") {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					createDBError("E17001",  "db.open() requires a `.ezdb` file"),
				}}
			}

			info, statErr := os.Stat(path.Value)
			if statErr == nil && info.IsDir() {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					createDBError("E17002", "db.open(): cannot read directory as file"),
				}}
			}

			if os.IsNotExist(statErr) {
				perms := os.FileMode(0644)
				if err := atomicWriteFile(path.Value, []byte(""), perms); err != nil {
					return &object.ReturnValue{Values: []object.Object{
						&object.Error{Code: "E17001", Message: "db.open() failed to open database file"},
					}}
				}
				return &object.ReturnValue{Values: []object.Object{
					&object.Database{
						Path: *path,
						Store: *object.NewMap(),
						IsClosed: object.Boolean{Value: false},
					},
					object.NIL,
				}}
			}

			content, err := os.ReadFile(path.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					createDBError("E17002", "db.open(): could not read database file"),
				}}
			}

			result, err := decodeFromJSON(string(content))
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					createDBError("E17004", "db.open(): database file is corrupted"),
				}}
			}

			dbContent, ok := result.(*object.Map)
			if !ok {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					createDBError("E17004", "db.open(): database file is corrupted"),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Database{
					Path: *path,
					Store: *dbContent,
					IsClosed: object.Boolean{Value: false},
				},
				object.NIL,
			}}
		},
	},

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
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17005", Message: "db.save() cannot operate on closed database"},
				}}
			}

			jsonRes, err := encodeToJSON(&db.Store, make(map[uintptr]bool))
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17003", Message: "db.save() database contents not json encodable"},
				}}
			}

			perms := os.FileMode(0644)
			if err := atomicWriteFile(db.Path.Value, []byte(jsonRes), perms); err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17003", Message: "db.save() failed to write to database"},
				}}
			}

			db.IsClosed = object.Boolean{Value: true}

			return &object.ReturnValue{Values: []object.Object{
				&object.Nil{},
			}}
		},
	},

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
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17005", Message: "db.set() cannot operate on closed database"},
				}}
			}

			key, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.set() requires a String as second argument"}
			}

			db.Store.Set(key, args[2])

			return &object.Nil{}
		},
	},

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
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17005", Message: "db.get() cannot operate on closed database"},
				}}
			}

			key, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.get() requires a String as second argument"}
			}

			val, exists := db.Store.Get(key)
			return &object.ReturnValue{Values: []object.Object{
				val,
				&object.Boolean{Value: exists},
			}}
		},
	},

	"db.delete": {},

	"db.has": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "db.has() takes exactly 2 arguments"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.has() requires a Database object as first argument"}
			}

			if db.IsClosed.Value {
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17005", Message: "db.has() cannot operate on closed database"},
				}}
			}

			key, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.has() requires a String as second argument"}
			}

			_, exists := db.Store.Get(key)
			return &object.ReturnValue{Values: []object.Object{
				&object.Boolean{Value: exists},
			}}
		},
	},

	"db.keys": {},

	"db.prefix": {},

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
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17005", Message: "db.count() cannot operate on closed database"},
				}}
			}

			count := len(db.Store.Pairs)
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(int64(count))},
			}}
		},
	},

	"db.clear": {},

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
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17005", Message: "db.save() cannot operate on closed database"},
				}}
			}

			jsonRes, err := encodeToJSON(&db.Store, make(map[uintptr]bool))
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17003", Message: "db.save() database contents not json encodable"},
				}}
			}

			perms := os.FileMode(0644)
			if err := atomicWriteFile(db.Path.Value, []byte(jsonRes), perms); err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17003", Message: "db.save() failed to write to database"},
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Nil{},
			}}
		},
	},
}

func createDBError(code, message string) *object.Struct {
	return &object.Struct{
		TypeName: "Error",
		Fields: map[string]object.Object{
			"message": &object.String{Value: message},
			"code": &object.String{Value: code},
		},
	}
}
