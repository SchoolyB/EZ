package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// ============================================================================
// Test Helpers for IO Module
// ============================================================================

// createTempDir creates a temporary directory for testing and returns its path
// and a cleanup function
func createTempDir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "ez_io_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	return dir, func() {
		os.RemoveAll(dir)
	}
}

// createTempFile creates a temporary file with the given content and returns its path
func createTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return path
}

// getReturnValues extracts values from a ReturnValue object
func getReturnValues(t *testing.T, obj object.Object) []object.Object {
	t.Helper()
	rv, ok := obj.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", obj)
	}
	return rv.Values
}

// assertNoError checks that the second return value is nil (no error)
func assertNoError(t *testing.T, obj object.Object) {
	t.Helper()
	vals := getReturnValues(t, obj)
	if len(vals) < 2 {
		t.Fatalf("expected at least 2 return values, got %d", len(vals))
	}
	if vals[1] != object.NIL {
		if errStruct, ok := vals[1].(*object.Struct); ok {
			if msg, ok := errStruct.Fields["message"].(*object.String); ok {
				t.Fatalf("expected no error, got: %s", msg.Value)
			}
		}
		t.Fatalf("expected nil error, got %T", vals[1])
	}
}

// assertHasError checks that the second return value is an error
func assertHasError(t *testing.T, obj object.Object) *object.Struct {
	t.Helper()
	vals := getReturnValues(t, obj)
	if len(vals) < 2 {
		t.Fatalf("expected at least 2 return values, got %d", len(vals))
	}
	errStruct, ok := vals[1].(*object.Struct)
	if !ok || vals[1] == object.NIL {
		t.Fatalf("expected error struct, got %T", vals[1])
	}
	return errStruct
}

// ============================================================================
// File Reading Tests
// ============================================================================

func TestIOReadFile(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	readFileFn := IOBuiltins["io.read_file"].Fn

	t.Run("read existing file", func(t *testing.T) {
		content := "Hello, EZ!"
		path := createTempFile(t, dir, "test.txt", content)

		result := readFileFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		str, ok := vals[0].(*object.String)
		if !ok {
			t.Fatalf("expected String, got %T", vals[0])
		}
		if str.Value != content {
			t.Errorf("expected %q, got %q", content, str.Value)
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		result := readFileFn(&object.String{Value: filepath.Join(dir, "nonexistent.txt")})
		errStruct := assertHasError(t, result)

		code, ok := errStruct.Fields["code"].(*object.String)
		if !ok || code.Value != "E7016" {
			t.Errorf("expected error code E7016, got %v", errStruct.Fields["code"])
		}
	})

	t.Run("wrong argument count", func(t *testing.T) {
		result := readFileFn()
		if !isErrorObject(result) {
			t.Error("expected error for no arguments")
		}
	})

	t.Run("wrong argument type", func(t *testing.T) {
		result := readFileFn(&object.Integer{Value: 123})
		if !isErrorObject(result) {
			t.Error("expected error for wrong argument type")
		}
	})
}

// ============================================================================
// File Writing Tests
// ============================================================================

func TestIOWriteFile(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	writeFileFn := IOBuiltins["io.write_file"].Fn

	t.Run("write new file", func(t *testing.T) {
		path := filepath.Join(dir, "new_file.txt")
		content := "Test content"

		result := writeFileFn(&object.String{Value: path}, &object.String{Value: content})
		assertNoError(t, result)

		// Verify file was created
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}
		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		path := createTempFile(t, dir, "existing.txt", "old content")
		newContent := "new content"

		result := writeFileFn(&object.String{Value: path}, &object.String{Value: newContent})
		assertNoError(t, result)

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(data) != newContent {
			t.Errorf("expected %q, got %q", newContent, string(data))
		}
	})

	t.Run("wrong argument count", func(t *testing.T) {
		result := writeFileFn(&object.String{Value: "path"})
		if !isErrorObject(result) {
			t.Error("expected error for missing content argument")
		}
	})
}

func TestIOAppendFile(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	appendFileFn := IOBuiltins["io.append_file"].Fn

	t.Run("append to existing file", func(t *testing.T) {
		path := createTempFile(t, dir, "append_test.txt", "Hello")

		result := appendFileFn(&object.String{Value: path}, &object.String{Value: " World"})
		assertNoError(t, result)

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(data) != "Hello World" {
			t.Errorf("expected %q, got %q", "Hello World", string(data))
		}
	})

	t.Run("append to new file", func(t *testing.T) {
		path := filepath.Join(dir, "new_append.txt")

		result := appendFileFn(&object.String{Value: path}, &object.String{Value: "New content"})
		assertNoError(t, result)

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(data) != "New content" {
			t.Errorf("expected %q, got %q", "New content", string(data))
		}
	})
}

// ============================================================================
// Path Existence Tests
// ============================================================================

func TestIOExists(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	existsFn := IOBuiltins["io.exists"].Fn

	t.Run("existing file", func(t *testing.T) {
		path := createTempFile(t, dir, "exists.txt", "content")
		result := existsFn(&object.String{Value: path})
		testBooleanObject(t, result, true)
	})

	t.Run("existing directory", func(t *testing.T) {
		result := existsFn(&object.String{Value: dir})
		testBooleanObject(t, result, true)
	})

	t.Run("non-existent path", func(t *testing.T) {
		result := existsFn(&object.String{Value: filepath.Join(dir, "nonexistent")})
		testBooleanObject(t, result, false)
	})
}

func TestIOIsFile(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	isFileFn := IOBuiltins["io.is_file"].Fn

	t.Run("regular file", func(t *testing.T) {
		path := createTempFile(t, dir, "file.txt", "content")
		result := isFileFn(&object.String{Value: path})
		testBooleanObject(t, result, true)
	})

	t.Run("directory", func(t *testing.T) {
		result := isFileFn(&object.String{Value: dir})
		testBooleanObject(t, result, false)
	})

	t.Run("non-existent", func(t *testing.T) {
		result := isFileFn(&object.String{Value: filepath.Join(dir, "nonexistent")})
		testBooleanObject(t, result, false)
	})
}

func TestIOIsDir(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	isDirFn := IOBuiltins["io.is_dir"].Fn

	t.Run("directory", func(t *testing.T) {
		result := isDirFn(&object.String{Value: dir})
		testBooleanObject(t, result, true)
	})

	t.Run("regular file", func(t *testing.T) {
		path := createTempFile(t, dir, "file.txt", "content")
		result := isDirFn(&object.String{Value: path})
		testBooleanObject(t, result, false)
	})

	t.Run("non-existent", func(t *testing.T) {
		result := isDirFn(&object.String{Value: filepath.Join(dir, "nonexistent")})
		testBooleanObject(t, result, false)
	})
}

// ============================================================================
// File Operation Tests
// ============================================================================

func TestIORemove(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	removeFn := IOBuiltins["io.remove"].Fn

	t.Run("remove existing file", func(t *testing.T) {
		path := createTempFile(t, dir, "to_remove.txt", "content")

		result := removeFn(&object.String{Value: path})
		assertNoError(t, result)

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("file should have been removed")
		}
	})

	t.Run("remove non-existent file", func(t *testing.T) {
		result := removeFn(&object.String{Value: filepath.Join(dir, "nonexistent.txt")})
		assertHasError(t, result)
	})

	t.Run("cannot remove directory", func(t *testing.T) {
		subdir := filepath.Join(dir, "subdir")
		os.Mkdir(subdir, 0755)

		result := removeFn(&object.String{Value: subdir})
		errStruct := assertHasError(t, result)

		code, _ := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7018" {
			t.Errorf("expected E7018, got %s", code.Value)
		}
	})
}

func TestIORemoveDir(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	removeDirFn := IOBuiltins["io.remove_dir"].Fn

	t.Run("remove empty directory", func(t *testing.T) {
		subdir := filepath.Join(dir, "empty_dir")
		os.Mkdir(subdir, 0755)

		result := removeDirFn(&object.String{Value: subdir})
		assertNoError(t, result)

		if _, err := os.Stat(subdir); !os.IsNotExist(err) {
			t.Error("directory should have been removed")
		}
	})

	t.Run("cannot remove file", func(t *testing.T) {
		path := createTempFile(t, dir, "file.txt", "content")

		result := removeDirFn(&object.String{Value: path})
		errStruct := assertHasError(t, result)

		code, _ := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7019" {
			t.Errorf("expected E7019, got %s", code.Value)
		}
	})

	t.Run("cannot remove non-empty directory", func(t *testing.T) {
		subdir := filepath.Join(dir, "non_empty")
		os.Mkdir(subdir, 0755)
		createTempFile(t, subdir, "file.txt", "content")

		result := removeDirFn(&object.String{Value: subdir})
		assertHasError(t, result)
	})
}

func TestIORemoveAll(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	removeAllFn := IOBuiltins["io.remove_all"].Fn

	t.Run("remove directory recursively", func(t *testing.T) {
		subdir := filepath.Join(dir, "recursive")
		os.MkdirAll(filepath.Join(subdir, "nested"), 0755)
		createTempFile(t, subdir, "file1.txt", "content")
		createTempFile(t, filepath.Join(subdir, "nested"), "file2.txt", "content")

		result := removeAllFn(&object.String{Value: subdir})
		assertNoError(t, result)

		if _, err := os.Stat(subdir); !os.IsNotExist(err) {
			t.Error("directory should have been removed recursively")
		}
	})

	t.Run("safety check for root", func(t *testing.T) {
		result := removeAllFn(&object.String{Value: "/"})
		errStruct := assertHasError(t, result)

		code, _ := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7020" {
			t.Errorf("expected E7020 safety error, got %s", code.Value)
		}
	})
}

func TestIORename(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	renameFn := IOBuiltins["io.rename"].Fn

	t.Run("rename file", func(t *testing.T) {
		oldPath := createTempFile(t, dir, "old_name.txt", "content")
		newPath := filepath.Join(dir, "new_name.txt")

		result := renameFn(&object.String{Value: oldPath}, &object.String{Value: newPath})
		assertNoError(t, result)

		if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
			t.Error("old file should not exist")
		}
		if _, err := os.Stat(newPath); err != nil {
			t.Error("new file should exist")
		}
	})

	t.Run("rename non-existent file", func(t *testing.T) {
		result := renameFn(
			&object.String{Value: filepath.Join(dir, "nonexistent.txt")},
			&object.String{Value: filepath.Join(dir, "new.txt")},
		)
		assertHasError(t, result)
	})
}

func TestIOCopy(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	copyFn := IOBuiltins["io.copy"].Fn

	t.Run("copy file", func(t *testing.T) {
		srcPath := createTempFile(t, dir, "source.txt", "Hello, Copy!")
		dstPath := filepath.Join(dir, "destination.txt")

		result := copyFn(&object.String{Value: srcPath}, &object.String{Value: dstPath})
		assertNoError(t, result)

		// Both files should exist
		srcData, _ := os.ReadFile(srcPath)
		dstData, _ := os.ReadFile(dstPath)

		if string(srcData) != string(dstData) {
			t.Error("copied file content should match source")
		}
	})

	t.Run("cannot copy directory", func(t *testing.T) {
		subdir := filepath.Join(dir, "subdir")
		os.Mkdir(subdir, 0755)

		result := copyFn(&object.String{Value: subdir}, &object.String{Value: filepath.Join(dir, "copy")})
		errStruct := assertHasError(t, result)

		code, _ := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7021" {
			t.Errorf("expected E7021, got %s", code.Value)
		}
	})
}

// ============================================================================
// Directory Operation Tests
// ============================================================================

func TestIOMkdir(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	mkdirFn := IOBuiltins["io.mkdir"].Fn

	t.Run("create directory", func(t *testing.T) {
		path := filepath.Join(dir, "new_dir")

		result := mkdirFn(&object.String{Value: path})
		assertNoError(t, result)

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("directory should exist: %v", err)
		}
		if !info.IsDir() {
			t.Error("should be a directory")
		}
	})

	t.Run("cannot create nested without parent", func(t *testing.T) {
		path := filepath.Join(dir, "parent", "child")

		result := mkdirFn(&object.String{Value: path})
		assertHasError(t, result)
	})
}

func TestIOMkdirAll(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	mkdirAllFn := IOBuiltins["io.mkdir_all"].Fn

	t.Run("create nested directories", func(t *testing.T) {
		path := filepath.Join(dir, "parent", "child", "grandchild")

		result := mkdirAllFn(&object.String{Value: path})
		assertNoError(t, result)

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("directory should exist: %v", err)
		}
		if !info.IsDir() {
			t.Error("should be a directory")
		}
	})
}

func TestIOReadDir(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	readDirFn := IOBuiltins["io.read_dir"].Fn

	t.Run("list directory contents", func(t *testing.T) {
		createTempFile(t, dir, "file1.txt", "content")
		createTempFile(t, dir, "file2.txt", "content")
		os.Mkdir(filepath.Join(dir, "subdir"), 0755)

		result := readDirFn(&object.String{Value: dir})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		arr, ok := vals[0].(*object.Array)
		if !ok {
			t.Fatalf("expected Array, got %T", vals[0])
		}

		if len(arr.Elements) != 3 {
			t.Errorf("expected 3 entries, got %d", len(arr.Elements))
		}
	})

	t.Run("read non-existent directory", func(t *testing.T) {
		result := readDirFn(&object.String{Value: filepath.Join(dir, "nonexistent")})
		assertHasError(t, result)
	})
}

// ============================================================================
// File Metadata Tests
// ============================================================================

func TestIOFileSize(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	fileSizeFn := IOBuiltins["io.file_size"].Fn

	t.Run("get file size", func(t *testing.T) {
		content := "Hello, World!"
		path := createTempFile(t, dir, "sized.txt", content)

		result := fileSizeFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		size, ok := vals[0].(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", vals[0])
		}
		if size.Value != int64(len(content)) {
			t.Errorf("expected %d, got %d", len(content), size.Value)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		result := fileSizeFn(&object.String{Value: filepath.Join(dir, "nonexistent.txt")})
		assertHasError(t, result)
	})
}

func TestIOFileModTime(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	fileModTimeFn := IOBuiltins["io.file_mod_time"].Fn

	t.Run("get modification time", func(t *testing.T) {
		path := createTempFile(t, dir, "timed.txt", "content")

		result := fileModTimeFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		timestamp, ok := vals[0].(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", vals[0])
		}
		if timestamp.Value <= 0 {
			t.Error("timestamp should be positive")
		}
	})
}

// ============================================================================
// Path Utility Tests
// ============================================================================

func TestIOPathJoin(t *testing.T) {
	pathJoinFn := IOBuiltins["io.path_join"].Fn

	t.Run("join multiple parts", func(t *testing.T) {
		result := pathJoinFn(
			&object.String{Value: "home"},
			&object.String{Value: "user"},
			&object.String{Value: "documents"},
		)

		str, ok := result.(*object.String)
		if !ok {
			t.Fatalf("expected String, got %T", result)
		}

		expected := filepath.Join("home", "user", "documents")
		if str.Value != expected {
			t.Errorf("expected %q, got %q", expected, str.Value)
		}
	})

	t.Run("no arguments", func(t *testing.T) {
		result := pathJoinFn()
		if !isErrorObject(result) {
			t.Error("expected error for no arguments")
		}
	})
}

func TestIOPathBase(t *testing.T) {
	pathBaseFn := IOBuiltins["io.path_base"].Fn

	tests := []struct {
		input    string
		expected string
	}{
		{"/home/user/file.txt", "file.txt"},
		{"/home/user/", "user"},
		{"file.txt", "file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := pathBaseFn(&object.String{Value: tt.input})
			testStringObject(t, result, tt.expected)
		})
	}
}

func TestIOPathDir(t *testing.T) {
	pathDirFn := IOBuiltins["io.path_dir"].Fn

	t.Run("get directory", func(t *testing.T) {
		result := pathDirFn(&object.String{Value: "/home/user/file.txt"})
		str, ok := result.(*object.String)
		if !ok {
			t.Fatalf("expected String, got %T", result)
		}
		// Normalize for comparison
		expected := filepath.Dir("/home/user/file.txt")
		if str.Value != expected {
			t.Errorf("expected %q, got %q", expected, str.Value)
		}
	})
}

func TestIOPathExt(t *testing.T) {
	pathExtFn := IOBuiltins["io.path_ext"].Fn

	tests := []struct {
		input    string
		expected string
	}{
		{"file.txt", ".txt"},
		{"file.tar.gz", ".gz"},
		{"file", ""},
		{".hidden", ".hidden"}, // Go's filepath.Ext treats .hidden as extension
		{"noext", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := pathExtFn(&object.String{Value: tt.input})
			testStringObject(t, result, tt.expected)
		})
	}
}

func TestIOPathAbs(t *testing.T) {
	pathAbsFn := IOBuiltins["io.path_abs"].Fn

	t.Run("get absolute path", func(t *testing.T) {
		result := pathAbsFn(&object.String{Value: "."})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		str, ok := vals[0].(*object.String)
		if !ok {
			t.Fatalf("expected String, got %T", vals[0])
		}
		if !filepath.IsAbs(str.Value) {
			t.Error("result should be an absolute path")
		}
	})
}

func TestIOPathClean(t *testing.T) {
	pathCleanFn := IOBuiltins["io.path_clean"].Fn

	tests := []struct {
		input    string
		expected string
	}{
		{"./foo/bar/../baz", filepath.Clean("./foo/bar/../baz")},
		{"foo//bar", filepath.Clean("foo//bar")},
		{"/a/b/../c", filepath.Clean("/a/b/../c")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := pathCleanFn(&object.String{Value: tt.input})
			testStringObject(t, result, tt.expected)
		})
	}
}

func TestIOPathSeparator(t *testing.T) {
	pathSeparatorFn := IOBuiltins["io.path_separator"].Fn

	result := pathSeparatorFn()
	str, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	expected := string(filepath.Separator)
	if str.Value != expected {
		t.Errorf("expected %q, got %q", expected, str.Value)
	}

	// Platform-specific check
	if runtime.GOOS == "windows" {
		if str.Value != "\\" {
			t.Error("expected backslash on Windows")
		}
	} else {
		if str.Value != "/" {
			t.Error("expected forward slash on Unix")
		}
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestIOIntegration(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	t.Run("write then read", func(t *testing.T) {
		writeFileFn := IOBuiltins["io.write_file"].Fn
		readFileFn := IOBuiltins["io.read_file"].Fn

		path := filepath.Join(dir, "integration.txt")
		content := "Integration test content"

		// Write
		writeResult := writeFileFn(&object.String{Value: path}, &object.String{Value: content})
		assertNoError(t, writeResult)

		// Read
		readResult := readFileFn(&object.String{Value: path})
		assertNoError(t, readResult)

		vals := getReturnValues(t, readResult)
		str, _ := vals[0].(*object.String)
		if str.Value != content {
			t.Errorf("read content doesn't match written content")
		}
	})

	t.Run("mkdir then read_dir", func(t *testing.T) {
		mkdirFn := IOBuiltins["io.mkdir"].Fn
		readDirFn := IOBuiltins["io.read_dir"].Fn
		writeFileFn := IOBuiltins["io.write_file"].Fn

		subdir := filepath.Join(dir, "subdir_test")

		// Create directory
		mkdirResult := mkdirFn(&object.String{Value: subdir})
		assertNoError(t, mkdirResult)

		// Create files in directory
		writeFileFn(&object.String{Value: filepath.Join(subdir, "a.txt")}, &object.String{Value: "a"})
		writeFileFn(&object.String{Value: filepath.Join(subdir, "b.txt")}, &object.String{Value: "b"})

		// Read directory
		readResult := readDirFn(&object.String{Value: subdir})
		assertNoError(t, readResult)

		vals := getReturnValues(t, readResult)
		arr, _ := vals[0].(*object.Array)
		if len(arr.Elements) != 2 {
			t.Errorf("expected 2 files, got %d", len(arr.Elements))
		}
	})
}

// ============================================================================
// GetAllBuiltins Integration Test
// ============================================================================

func TestIOBuiltinsRegistered(t *testing.T) {
	builtins := GetAllBuiltins()

	expectedFunctions := []string{
		"io.read_file",
		"io.write_file",
		"io.append_file",
		"io.exists",
		"io.is_file",
		"io.is_dir",
		"io.remove",
		"io.remove_dir",
		"io.remove_all",
		"io.rename",
		"io.copy",
		"io.mkdir",
		"io.mkdir_all",
		"io.read_dir",
		"io.file_size",
		"io.file_mod_time",
		"io.path_join",
		"io.path_base",
		"io.path_dir",
		"io.path_ext",
		"io.path_abs",
		"io.path_clean",
		"io.path_separator",
	}

	for _, name := range expectedFunctions {
		if builtins[name] == nil {
			t.Errorf("expected builtin %q not found in GetAllBuiltins()", name)
		}
	}
}

// ============================================================================
// Security Validation Tests
// ============================================================================

func TestIOSecurityEmptyPath(t *testing.T) {
	// Test that empty paths are rejected with E7040
	tests := []struct {
		name string
		fn   string
	}{
		{"io.read_file", "io.read_file"},
		{"io.write_file", "io.write_file"},
		{"io.append_file", "io.append_file"},
		{"io.remove", "io.remove"},
		{"io.remove_dir", "io.remove_dir"},
		{"io.remove_all", "io.remove_all"},
		{"io.mkdir", "io.mkdir"},
		{"io.mkdir_all", "io.mkdir_all"},
		{"io.read_dir", "io.read_dir"},
		{"io.file_size", "io.file_size"},
		{"io.file_mod_time", "io.file_mod_time"},
		{"io.path_abs", "io.path_abs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := IOBuiltins[tt.fn].Fn
			var result object.Object

			// Call with empty path - functions need different arg counts
			switch tt.fn {
			case "io.write_file", "io.append_file":
				result = fn(&object.String{Value: ""}, &object.String{Value: "content"})
			default:
				result = fn(&object.String{Value: ""})
			}

			// Should return error tuple with E7040
			rv, ok := result.(*object.ReturnValue)
			if !ok {
				t.Fatalf("expected ReturnValue, got %T", result)
			}
			if len(rv.Values) < 2 {
				t.Fatalf("expected 2 return values, got %d", len(rv.Values))
			}
			errStruct, ok := rv.Values[1].(*object.Struct)
			if !ok {
				t.Fatalf("expected error struct, got %T", rv.Values[1])
			}
			code, ok := errStruct.Fields["code"].(*object.String)
			if !ok || code.Value != "E7040" {
				t.Errorf("expected error code E7040, got %v", errStruct.Fields["code"])
			}
		})
	}

	// Test two-argument functions with empty paths
	t.Run("io.rename empty old path", func(t *testing.T) {
		fn := IOBuiltins["io.rename"].Fn
		result := fn(&object.String{Value: ""}, &object.String{Value: "/tmp/new"})
		rv := result.(*object.ReturnValue)
		errStruct := rv.Values[1].(*object.Struct)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7040" {
			t.Errorf("expected error code E7040, got %s", code.Value)
		}
	})

	t.Run("io.rename empty new path", func(t *testing.T) {
		dir, cleanup := createTempDir(t)
		defer cleanup()
		path := createTempFile(t, dir, "test.txt", "content")

		fn := IOBuiltins["io.rename"].Fn
		result := fn(&object.String{Value: path}, &object.String{Value: ""})
		rv := result.(*object.ReturnValue)
		errStruct := rv.Values[1].(*object.Struct)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7040" {
			t.Errorf("expected error code E7040, got %s", code.Value)
		}
	})

	t.Run("io.copy empty src path", func(t *testing.T) {
		fn := IOBuiltins["io.copy"].Fn
		result := fn(&object.String{Value: ""}, &object.String{Value: "/tmp/dst"})
		rv := result.(*object.ReturnValue)
		errStruct := rv.Values[1].(*object.Struct)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7040" {
			t.Errorf("expected error code E7040, got %s", code.Value)
		}
	})

	t.Run("io.copy empty dst path", func(t *testing.T) {
		dir, cleanup := createTempDir(t)
		defer cleanup()
		path := createTempFile(t, dir, "test.txt", "content")

		fn := IOBuiltins["io.copy"].Fn
		result := fn(&object.String{Value: path}, &object.String{Value: ""})
		rv := result.(*object.ReturnValue)
		errStruct := rv.Values[1].(*object.Struct)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7040" {
			t.Errorf("expected error code E7040, got %s", code.Value)
		}
	})
}

func TestIOSecurityNullByte(t *testing.T) {
	// Test that paths with null bytes are rejected with E7041
	nullPath := "test\x00file.txt"

	tests := []struct {
		name string
		fn   string
	}{
		{"io.read_file", "io.read_file"},
		{"io.write_file", "io.write_file"},
		{"io.append_file", "io.append_file"},
		{"io.remove", "io.remove"},
		{"io.remove_dir", "io.remove_dir"},
		{"io.remove_all", "io.remove_all"},
		{"io.mkdir", "io.mkdir"},
		{"io.mkdir_all", "io.mkdir_all"},
		{"io.read_dir", "io.read_dir"},
		{"io.file_size", "io.file_size"},
		{"io.file_mod_time", "io.file_mod_time"},
		{"io.path_abs", "io.path_abs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := IOBuiltins[tt.fn].Fn
			var result object.Object

			switch tt.fn {
			case "io.write_file", "io.append_file":
				result = fn(&object.String{Value: nullPath}, &object.String{Value: "content"})
			default:
				result = fn(&object.String{Value: nullPath})
			}

			rv, ok := result.(*object.ReturnValue)
			if !ok {
				t.Fatalf("expected ReturnValue, got %T", result)
			}
			errStruct, ok := rv.Values[1].(*object.Struct)
			if !ok {
				t.Fatalf("expected error struct, got %T", rv.Values[1])
			}
			code, ok := errStruct.Fields["code"].(*object.String)
			if !ok || code.Value != "E7041" {
				t.Errorf("expected error code E7041, got %v", errStruct.Fields["code"])
			}
		})
	}

	// Test path utility functions that return object.Error for null bytes
	pathUtilTests := []string{
		"io.path_join",
		"io.path_base",
		"io.path_dir",
		"io.path_ext",
		"io.path_clean",
	}

	for _, fnName := range pathUtilTests {
		t.Run(fnName+" null byte", func(t *testing.T) {
			fn := IOBuiltins[fnName].Fn
			var result object.Object

			if fnName == "io.path_join" {
				result = fn(&object.String{Value: "valid"}, &object.String{Value: nullPath})
			} else {
				result = fn(&object.String{Value: nullPath})
			}

			errObj, ok := result.(*object.Error)
			if !ok {
				t.Fatalf("expected Error, got %T", result)
			}
			if errObj.Code != "E7041" {
				t.Errorf("expected error code E7041, got %s", errObj.Code)
			}
		})
	}
}

func TestIOSecurityBoolFunctionsInvalidPaths(t *testing.T) {
	// Test that boolean functions return false for invalid paths (don't error)
	tests := []struct {
		name string
		fn   string
	}{
		{"io.exists empty", "io.exists"},
		{"io.is_file empty", "io.is_file"},
		{"io.is_dir empty", "io.is_dir"},
	}

	for _, tt := range tests {
		t.Run(tt.name+" empty path", func(t *testing.T) {
			fn := IOBuiltins[tt.fn].Fn
			result := fn(&object.String{Value: ""})
			if result != object.FALSE {
				t.Errorf("expected FALSE for empty path, got %T", result)
			}
		})

		t.Run(tt.name+" null byte path", func(t *testing.T) {
			fn := IOBuiltins[tt.fn].Fn
			result := fn(&object.String{Value: "test\x00file"})
			if result != object.FALSE {
				t.Errorf("expected FALSE for null byte path, got %T", result)
			}
		})
	}
}

func TestIOSecurityReadDirectoryAsFile(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	// Create a subdirectory
	subdir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	fn := IOBuiltins["io.read_file"].Fn
	result := fn(&object.String{Value: subdir})

	rv, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", result)
	}

	// First value should be nil
	if rv.Values[0] != object.NIL {
		t.Errorf("expected nil first value, got %T", rv.Values[0])
	}

	// Second value should be error with E7042
	errStruct, ok := rv.Values[1].(*object.Struct)
	if !ok {
		t.Fatalf("expected error struct, got %T", rv.Values[1])
	}
	code, ok := errStruct.Fields["code"].(*object.String)
	if !ok || code.Value != "E7042" {
		t.Errorf("expected error code E7042, got %v", errStruct.Fields["code"])
	}
}
