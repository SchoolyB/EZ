package stdlib

import (
	"math/big"
	"os"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)


// ============================================================================
// Test Helpers for DB Module
// ============================================================================

// ============================================================================
// Database Opening Tests
// ============================================================================

func TestDBOpen(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := DbBuiltins["db.open"].Fn

	t.Run("opening existing file", func(t *testing.T) {
		content := "{\"name\":\"Alice\"}"
		path := createTempFile(t, dir, "mydb.ezdb", content)

		result := openFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		if _, ok := vals[0].(*object.Database); !ok {
			t.Fatalf("expected Database, got %T", vals[0])
		}
	})

	t.Run("opening non-existent file", func(t *testing.T) {
		path := dir + "mydb.ezdb"
		result := openFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		if _, ok := vals[0].(*object.Database); !ok {
			t.Fatalf("expected Database, got %T", vals[0])
		}
	})

	t.Run("wrong argument count", func(t *testing.T) {
		result := openFn()
		if !isErrorObject(result) {
			t.Error("expected error for no arguments")
		}
	})

	t.Run("wrong argument type", func(t *testing.T) {
		result := openFn(&object.Integer{Value: big.NewInt(123)})
		if !isErrorObject(result) {
			t.Error("expected error for wrong argument type")
		}
	})
}

// ============================================================================
// Database Closing Tests
// ============================================================================

func TestDBClose(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := DbBuiltins["db.open"].Fn
	closeFn := DbBuiltins["db.close"].Fn
	setFn := DbBuiltins["db.set"].Fn

	t.Run("closing existing database file", func(t *testing.T) {
		content := "{\"name\":\"Alice\"}"
		path := createTempFile(t, dir, "mydb.ezdb", content)

		result := openFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		db, ok := vals[0].(*object.Database)
		if !ok {
			t.Fatalf("expected Database, got %T", vals[0])
		}

		result = closeFn(db)
		if isErrorObject(result) {
			t.Fatalf("expected no errors, got %T", result)
		}
	})

	t.Run("closing new database file", func(t *testing.T) {
		path := dir + "mydb.ezdb"
		result := openFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		db, ok := vals[0].(*object.Database)
		if !ok {
			t.Fatalf("expected Database, got %T", vals[0])
		}

		result = closeFn(db)
		if isErrorObject(result) {
			t.Fatalf("expected no errors, got %T", result)
		}
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		result := closeFn()
		if !isErrorObject(result) {
			t.Error("expected error for no arguments")
		}
	})

	t.Run("wrong argument type", func(t *testing.T) {
		result := closeFn(&object.Integer{Value: big.NewInt(123)})
		if !isErrorObject(result) {
			t.Error("expected error for wrong argument type")
		}
	})

	t.Run("save database to disk after close", func(t *testing.T) {
		path := dir + "mydb.ezdb"
		result := openFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		db, ok := vals[0].(*object.Database)
		if !ok {
			t.Fatalf("expected Database, got %T", vals[0])
		}

		setFn(db, &object.String{Value: "x"}, &object.String{Value: "42"})

		result = closeFn(db)
		if isErrorObject(result) {
			t.Fatalf("unexpected error while closing database")
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read written database file: %v", err)
		}

		content := "{\"x\":\"42\"}"
		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}
	})
}

// ============================================================================
// Database Set and Get Tests
// ============================================================================

func TestDBSetAndGet(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := DbBuiltins["db.open"].Fn
	setFn := DbBuiltins["db.set"].Fn
	getFn := DbBuiltins["db.get"].Fn

	path := createTempFile(t, dir, "mydb.ezdb", "{}")
	openRes := openFn(&object.String{Value: path})
	vals := getReturnValues(t, openRes)
	db := vals[0].(*object.Database)

	t.Run("set and get value", func(t *testing.T) {
		key := &object.String{Value: "name"}
		val := &object.String{Value: "Alice"}

		res := setFn(db, key, val)
		if isErrorObject(res) {
			t.Fatalf("unexpected error from db.set")
		}

		res = getFn(db, key)
		vals := getReturnValues(t, res)

		if vals[0].(*object.String).Value != "Alice" {
			t.Fatalf("expected 'Alice', got %v", vals[0])
		}
		if !vals[1].(*object.Boolean).Value {
			t.Fatalf("expected exists=true")
		}
	})

	t.Run("get missing key", func(t *testing.T) {
		key := &object.String{Value: "age"}
		res := getFn(db, key)
		vals := getReturnValues(t, res)

		if vals[0].(*object.String).Value != "" {
			t.Fatalf("expected empty String for missing key")
		}
		if vals[1].(*object.Boolean).Value {
			t.Fatalf("expected second return to be false")
		}
	})
}

// ============================================================================
// Database Has and Delete Tests
// ============================================================================

func TestDBDeleteAndHas(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := DbBuiltins["db.open"].Fn
	setFn := DbBuiltins["db.set"].Fn
	delFn := DbBuiltins["db.delete"].Fn
	hasFn := DbBuiltins["db.has"].Fn

	path := createTempFile(t, dir, "mydb.ezdb", "{}")
	db := getReturnValues(t, openFn(&object.String{Value: path}))[0].(*object.Database)

	key := &object.String{Value: "k"}
	val := &object.String{Value: "v"}
	setFn(db, key, val)

	t.Run("has existing key", func(t *testing.T) {
		res := hasFn(db, key)
		vals := getReturnValues(t, res)

		if !vals[0].(*object.Boolean).Value {
			t.Fatalf("expected true")
		}
	})

	t.Run("delete existing key", func(t *testing.T) {
		res := delFn(db, key)
		vals := getReturnValues(t, res)

		if !vals[0].(*object.Boolean).Value {
			t.Fatalf("expected deletion to succeed")
		}
	})

	t.Run("has deleted key", func(t *testing.T) {
		res := hasFn(db, key)
		vals := getReturnValues(t, res)

		if vals[0].(*object.Boolean).Value {
			t.Fatalf("expected false after deletion")
		}
	})
}

// ============================================================================
// Database Keys and Prefix Tests
// ============================================================================

func TestDBKeysAndPrefix(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := DbBuiltins["db.open"].Fn
	setFn := DbBuiltins["db.set"].Fn
	keysFn := DbBuiltins["db.keys"].Fn
	prefixFn := DbBuiltins["db.prefix"].Fn

	path := createTempFile(t, dir, "mydb.ezdb", "{}")
	db := getReturnValues(t, openFn(&object.String{Value: path}))[0].(*object.Database)

	setFn(db, &object.String{Value: "user:1"}, &object.String{Value: "A"})
	setFn(db, &object.String{Value: "user:2"}, &object.String{Value: "B"})
	setFn(db, &object.String{Value: "admin:1"}, &object.String{Value: "C"})

	t.Run("keys returns all keys", func(t *testing.T) {
		res := keysFn(db)
		arr := res.(*object.Array)

		if len(arr.Elements) != 3 {
			t.Fatalf("expected 3 keys, got %d", len(arr.Elements))
		}
	})

	t.Run("prefix filters keys", func(t *testing.T) {
		res := prefixFn(db, &object.String{Value: "user:"})
		arr := res.(*object.Array)

		if len(arr.Elements) != 2 {
			t.Fatalf("expected 2 keys, got %d", len(arr.Elements))
		}
	})
}

// ============================================================================
// Database Count and Clear Tests
// ============================================================================

func TestDBCountAndClear(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := DbBuiltins["db.open"].Fn
	setFn := DbBuiltins["db.set"].Fn
	countFn := DbBuiltins["db.count"].Fn
	clearFn := DbBuiltins["db.clear"].Fn

	path := createTempFile(t, dir, "mydb.ezdb", "{}")
	db := getReturnValues(t, openFn(&object.String{Value: path}))[0].(*object.Database)

	setFn(db, &object.String{Value: "a"}, &object.String{Value: "1"})
	setFn(db, &object.String{Value: "b"}, &object.String{Value: "2"})

	t.Run("count entries", func(t *testing.T) {
		res := countFn(db)
		vals := getReturnValues(t, res)

		if vals[0].(*object.Integer).Value.Int64() != 2 {
			t.Fatalf("expected count=2")
		}
	})

	t.Run("clear database", func(t *testing.T) {
		clearFn(db)
		res := countFn(db)
		vals := getReturnValues(t, res)

		if vals[0].(*object.Integer).Value.Int64() != 0 {
			t.Fatalf("expected count=0 after clear")
		}
	})
}

// ============================================================================
// Database Saving Tests
// ============================================================================

func TestDBSave(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := DbBuiltins["db.open"].Fn
	setFn := DbBuiltins["db.set"].Fn
	saveFn := DbBuiltins["db.save"].Fn
	closeFn := DbBuiltins["db.close"].Fn

	path := dir + "mydb.ezdb"

	db := getReturnValues(t, openFn(&object.String{Value: path}))[0].(*object.Database)
	setFn(db, &object.String{Value: "x"}, &object.String{Value: "42"})

	t.Run("save database to disk", func(t *testing.T) {
		res := saveFn(db)
		if isErrorObject(res) {
			t.Fatalf("unexpected error during save")
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read written database file: %v", err)
		}

		content := "{\"x\":\"42\"}"
		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}
	})

	t.Run("save on closed database", func(t *testing.T) {
		res := closeFn(db)
		if isErrorObject(res) {
			t.Fatalf("unexpected error during close")
		}

		res = saveFn(db)
		vals := getReturnValues(t, res)
		if !isErrorObject(vals[0]) {
			t.Fatalf("expected error on saving after closing database")
		}
	})
}

