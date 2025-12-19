package stdlib

import (
	"strings"
	"os"

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

			db := JsonBuiltins["json.decode"].Fn(&object.String{Value: string(content)}).(*object.ReturnValue)
			if db.Values[1] != object.NIL {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					createDBError("E17004", "db.open(): database file is corrupted"),
				}}
			}

			dbContent, ok := db.Values[0].(*object.Map)
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

	"db.close": {},
	"db.set": {},
	"db.get": {},
	"db.delete": {},
	"db.has": {},
	"db.keys": {},
	"db.prefix": {},
	"db.count": {},
	"db.clear": {},

	"db.save": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "db.save() takes exactly 1 argument"}
			}

			db, ok := args[0].(*object.Database)
			if !ok {
				return &object.Error{Code: "E7001", Message: "db.save() requires a Database struct as argument"}
			}
			
			if db.IsClosed.Value {
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17005", Message: "db.save() cannot operate on closed database"},
				}}
			}

			jsonRes := JsonBuiltins["json.encode"].Fn(&db.Store).(*object.ReturnValue)
			if jsonRes.Values[1] != object.NIL {
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17003", Message: "db.save() database contents not json encodable"},
				}}
			}

			encodedStr := jsonRes.Values[0].(*object.String)
			ioRes := IOBuiltins["io.write_file"].Fn(&db.Path, encodedStr).(*object.ReturnValue)
			if ioRes.Values[1] != object.NIL {
				return &object.ReturnValue{Values: []object.Object{
					&object.Error{Code: "E17003", Message: "db.save() failed to write to database"},
				}}
			}

			return &object.Nil{}
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
