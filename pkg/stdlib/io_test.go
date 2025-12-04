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
		// File reading/writing
		"io.read_file",
		"io.write_file",
		"io.append_file",
		"io.read_bytes",
		"io.write_bytes",
		// Convenience functions
		"io.read_lines",
		"io.append_line",
		"io.expand_path",
		// File system queries
		"io.exists",
		"io.is_file",
		"io.is_dir",
		// File/directory operations
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
		// Path manipulation
		"io.path_join",
		"io.path_base",
		"io.path_dir",
		"io.path_ext",
		"io.path_abs",
		"io.path_clean",
		"io.path_separator",
		// File handle constants
		"io.READ_ONLY",
		"io.WRITE_ONLY",
		"io.READ_WRITE",
		"io.APPEND",
		"io.CREATE",
		"io.TRUNCATE",
		"io.EXCLUSIVE",
		"io.SEEK_START",
		"io.SEEK_CURRENT",
		"io.SEEK_END",
		// File handle operations
		"io.open",
		"io.read",
		"io.read_all",
		"io.read_string",
		"io.write",
		"io.seek",
		"io.tell",
		"io.flush",
		"io.close",
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

// ============================================================================
// Byte I/O Tests (Phase 4)
// ============================================================================

func TestIOReadBytes(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	readBytesFn := IOBuiltins["io.read_bytes"].Fn

	t.Run("read binary file", func(t *testing.T) {
		// Create a file with known binary content
		content := []byte{0x00, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0xFF, 0x00}
		path := filepath.Join(dir, "binary.dat")
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result := readBytesFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		arr, ok := vals[0].(*object.Array)
		if !ok {
			t.Fatalf("expected Array, got %T", vals[0])
		}

		if len(arr.Elements) != len(content) {
			t.Errorf("expected %d bytes, got %d", len(content), len(arr.Elements))
		}

		// Verify each byte
		for i, elem := range arr.Elements {
			b, ok := elem.(*object.Byte)
			if !ok {
				t.Errorf("element %d: expected Byte, got %T", i, elem)
				continue
			}
			if b.Value != content[i] {
				t.Errorf("element %d: expected 0x%02x, got 0x%02x", i, content[i], b.Value)
			}
		}
	})

	t.Run("read nonexistent file", func(t *testing.T) {
		result := readBytesFn(&object.String{Value: filepath.Join(dir, "nonexistent.dat")})
		assertHasError(t, result)
	})

	t.Run("read directory as bytes", func(t *testing.T) {
		result := readBytesFn(&object.String{Value: dir})
		errStruct := assertHasError(t, result)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7042" {
			t.Errorf("expected E7042, got %s", code.Value)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		result := readBytesFn(&object.String{Value: ""})
		rv := result.(*object.ReturnValue)
		errStruct := rv.Values[1].(*object.Struct)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7040" {
			t.Errorf("expected E7040, got %s", code.Value)
		}
	})
}

func TestIOWriteBytes(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	writeBytesFn := IOBuiltins["io.write_bytes"].Fn
	readBytesFn := IOBuiltins["io.read_bytes"].Fn

	// Helper to create byte array
	makeByteArray := func(data []byte) *object.Array {
		elements := make([]object.Object, len(data))
		for i, b := range data {
			elements[i] = &object.Byte{Value: b}
		}
		return &object.Array{Elements: elements, ElementType: "byte"}
	}

	t.Run("write binary file", func(t *testing.T) {
		content := []byte{0x00, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0xFF, 0x00}
		path := filepath.Join(dir, "write_test.dat")

		result := writeBytesFn(&object.String{Value: path}, makeByteArray(content))
		assertNoError(t, result)

		// Verify by reading back
		readResult := readBytesFn(&object.String{Value: path})
		assertNoError(t, readResult)

		vals := getReturnValues(t, readResult)
		arr := vals[0].(*object.Array)

		if len(arr.Elements) != len(content) {
			t.Errorf("expected %d bytes, got %d", len(content), len(arr.Elements))
		}

		for i, elem := range arr.Elements {
			b := elem.(*object.Byte)
			if b.Value != content[i] {
				t.Errorf("byte %d: expected 0x%02x, got 0x%02x", i, content[i], b.Value)
			}
		}
	})

	t.Run("write with permissions", func(t *testing.T) {
		content := []byte{0x41, 0x42, 0x43}
		path := filepath.Join(dir, "perms_test.dat")

		// Write with custom permissions (0600 = owner read/write only)
		result := writeBytesFn(
			&object.String{Value: path},
			makeByteArray(content),
			&object.Integer{Value: 0600},
		)
		assertNoError(t, result)

		// Verify file exists and has correct content
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(data) != "ABC" {
			t.Errorf("expected ABC, got %s", string(data))
		}

		// Verify permissions (on Unix)
		if runtime.GOOS != "windows" {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("failed to stat file: %v", err)
			}
			perm := info.Mode().Perm()
			if perm != 0600 {
				t.Errorf("expected permissions 0600, got %o", perm)
			}
		}
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		path := filepath.Join(dir, "overwrite_test.dat")

		// Write initial content
		result := writeBytesFn(&object.String{Value: path}, makeByteArray([]byte{0x01, 0x02, 0x03}))
		assertNoError(t, result)

		// Overwrite with new content
		result = writeBytesFn(&object.String{Value: path}, makeByteArray([]byte{0xFF, 0xFE}))
		assertNoError(t, result)

		// Verify new content
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if len(data) != 2 || data[0] != 0xFF || data[1] != 0xFE {
			t.Errorf("expected [0xFF, 0xFE], got %v", data)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		result := writeBytesFn(&object.String{Value: ""}, makeByteArray([]byte{0x00}))
		rv := result.(*object.ReturnValue)
		errStruct := rv.Values[1].(*object.Struct)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7040" {
			t.Errorf("expected E7040, got %s", code.Value)
		}
	})

	t.Run("wrong argument type", func(t *testing.T) {
		path := filepath.Join(dir, "wrong_type.dat")
		// Pass string instead of byte array
		result := writeBytesFn(&object.String{Value: path}, &object.String{Value: "hello"})
		_, ok := result.(*object.Error)
		if !ok {
			t.Errorf("expected Error for wrong type, got %T", result)
		}
	})
}

func TestIOAtomicWrite(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	writeFileFn := IOBuiltins["io.write_file"].Fn
	readFileFn := IOBuiltins["io.read_file"].Fn

	t.Run("atomic write creates file", func(t *testing.T) {
		path := filepath.Join(dir, "atomic_test.txt")
		content := "Hello, atomic world!"

		result := writeFileFn(&object.String{Value: path}, &object.String{Value: content})
		assertNoError(t, result)

		// Verify content
		readResult := readFileFn(&object.String{Value: path})
		assertNoError(t, readResult)

		vals := getReturnValues(t, readResult)
		str := vals[0].(*object.String)
		if str.Value != content {
			t.Errorf("expected %q, got %q", content, str.Value)
		}
	})

	t.Run("atomic write overwrites existing", func(t *testing.T) {
		path := filepath.Join(dir, "atomic_overwrite.txt")

		// Create initial file
		result := writeFileFn(&object.String{Value: path}, &object.String{Value: "initial"})
		assertNoError(t, result)

		// Overwrite
		result = writeFileFn(&object.String{Value: path}, &object.String{Value: "overwritten"})
		assertNoError(t, result)

		// Verify
		readResult := readFileFn(&object.String{Value: path})
		assertNoError(t, readResult)

		vals := getReturnValues(t, readResult)
		str := vals[0].(*object.String)
		if str.Value != "overwritten" {
			t.Errorf("expected 'overwritten', got %q", str.Value)
		}
	})

	t.Run("no temp files left behind", func(t *testing.T) {
		path := filepath.Join(dir, "no_leftovers.txt")

		result := writeFileFn(&object.String{Value: path}, &object.String{Value: "test"})
		assertNoError(t, result)

		// Check for any .ez_tmp_ files
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read dir: %v", err)
		}

		for _, entry := range entries {
			if filepath.HasPrefix(entry.Name(), ".ez_tmp_") {
				t.Errorf("temp file left behind: %s", entry.Name())
			}
		}
	})
}

// ============================================================================
// File Handle Constants Tests (Phase 5)
// ============================================================================

func TestIOConstants(t *testing.T) {
	// Test file mode constants
	t.Run("READ_ONLY", func(t *testing.T) {
		fn := IOBuiltins["io.READ_ONLY"].Fn
		result := fn()
		val, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", result)
		}
		if val.Value != int64(os.O_RDONLY) {
			t.Errorf("expected %d (O_RDONLY), got %d", os.O_RDONLY, val.Value)
		}
	})

	t.Run("WRITE_ONLY", func(t *testing.T) {
		fn := IOBuiltins["io.WRITE_ONLY"].Fn
		result := fn()
		val, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", result)
		}
		if val.Value != int64(os.O_WRONLY) {
			t.Errorf("expected %d (O_WRONLY), got %d", os.O_WRONLY, val.Value)
		}
	})

	t.Run("READ_WRITE", func(t *testing.T) {
		fn := IOBuiltins["io.READ_WRITE"].Fn
		result := fn()
		val, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", result)
		}
		if val.Value != int64(os.O_RDWR) {
			t.Errorf("expected %d (O_RDWR), got %d", os.O_RDWR, val.Value)
		}
	})

	t.Run("APPEND", func(t *testing.T) {
		fn := IOBuiltins["io.APPEND"].Fn
		result := fn()
		val, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", result)
		}
		if val.Value != int64(os.O_APPEND) {
			t.Errorf("expected %d (O_APPEND), got %d", os.O_APPEND, val.Value)
		}
	})

	t.Run("CREATE", func(t *testing.T) {
		fn := IOBuiltins["io.CREATE"].Fn
		result := fn()
		val, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", result)
		}
		if val.Value != int64(os.O_CREATE) {
			t.Errorf("expected %d (O_CREATE), got %d", os.O_CREATE, val.Value)
		}
	})

	t.Run("TRUNCATE", func(t *testing.T) {
		fn := IOBuiltins["io.TRUNCATE"].Fn
		result := fn()
		val, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", result)
		}
		if val.Value != int64(os.O_TRUNC) {
			t.Errorf("expected %d (O_TRUNC), got %d", os.O_TRUNC, val.Value)
		}
	})

	t.Run("EXCLUSIVE", func(t *testing.T) {
		fn := IOBuiltins["io.EXCLUSIVE"].Fn
		result := fn()
		val, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", result)
		}
		if val.Value != int64(os.O_EXCL) {
			t.Errorf("expected %d (O_EXCL), got %d", os.O_EXCL, val.Value)
		}
	})

	// Test seek constants
	t.Run("SEEK_START", func(t *testing.T) {
		fn := IOBuiltins["io.SEEK_START"].Fn
		result := fn()
		val, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", result)
		}
		if val.Value != 0 {
			t.Errorf("expected 0 (SEEK_START), got %d", val.Value)
		}
	})

	t.Run("SEEK_CURRENT", func(t *testing.T) {
		fn := IOBuiltins["io.SEEK_CURRENT"].Fn
		result := fn()
		val, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", result)
		}
		if val.Value != 1 {
			t.Errorf("expected 1 (SEEK_CURRENT), got %d", val.Value)
		}
	})

	t.Run("SEEK_END", func(t *testing.T) {
		fn := IOBuiltins["io.SEEK_END"].Fn
		result := fn()
		val, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", result)
		}
		if val.Value != 2 {
			t.Errorf("expected 2 (SEEK_END), got %d", val.Value)
		}
	})
}

// ============================================================================
// File Handle Operations Tests (Phase 5)
// ============================================================================

func TestIOOpen(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := IOBuiltins["io.open"].Fn
	closeFn := IOBuiltins["io.close"].Fn

	t.Run("open existing file for reading", func(t *testing.T) {
		content := "Hello, World!"
		path := createTempFile(t, dir, "open_read.txt", content)

		result := openFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		handle, ok := vals[0].(*object.FileHandle)
		if !ok {
			t.Fatalf("expected FileHandle, got %T", vals[0])
		}
		if handle.Path != path {
			t.Errorf("expected path %q, got %q", path, handle.Path)
		}
		if handle.IsClosed {
			t.Error("handle should not be closed")
		}

		// Clean up
		closeFn(handle)
	})

	t.Run("open new file for writing", func(t *testing.T) {
		path := filepath.Join(dir, "open_write.txt")
		mode := &object.Integer{Value: int64(os.O_WRONLY | os.O_CREATE)}

		result := openFn(&object.String{Value: path}, mode)
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		handle, ok := vals[0].(*object.FileHandle)
		if !ok {
			t.Fatalf("expected FileHandle, got %T", vals[0])
		}

		closeFn(handle)

		// Verify file was created
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("file should have been created")
		}
	})

	t.Run("open with custom permissions", func(t *testing.T) {
		path := filepath.Join(dir, "open_perms.txt")
		mode := &object.Integer{Value: int64(os.O_WRONLY | os.O_CREATE)}
		perms := &object.Integer{Value: 0600}

		result := openFn(&object.String{Value: path}, mode, perms)
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		handle := vals[0].(*object.FileHandle)
		closeFn(handle)

		// Verify permissions on Unix
		if runtime.GOOS != "windows" {
			info, _ := os.Stat(path)
			if info.Mode().Perm() != 0600 {
				t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
			}
		}
	})

	t.Run("open nonexistent file for reading", func(t *testing.T) {
		result := openFn(&object.String{Value: filepath.Join(dir, "nonexistent.txt")})
		assertHasError(t, result)
	})

	t.Run("empty path error", func(t *testing.T) {
		result := openFn(&object.String{Value: ""})
		rv := result.(*object.ReturnValue)
		errStruct := rv.Values[1].(*object.Struct)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7040" {
			t.Errorf("expected E7040, got %s", code.Value)
		}
	})
}

func TestIOReadHandle(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := IOBuiltins["io.open"].Fn
	readFn := IOBuiltins["io.read"].Fn
	closeFn := IOBuiltins["io.close"].Fn

	t.Run("read bytes from file", func(t *testing.T) {
		content := "Hello, World!"
		path := createTempFile(t, dir, "read_test.txt", content)

		result := openFn(&object.String{Value: path})
		assertNoError(t, result)
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		// Read 5 bytes
		readResult := readFn(handle, &object.Integer{Value: 5})
		assertNoError(t, readResult)

		vals := getReturnValues(t, readResult)
		arr, ok := vals[0].(*object.Array)
		if !ok {
			t.Fatalf("expected Array, got %T", vals[0])
		}
		if len(arr.Elements) != 5 {
			t.Errorf("expected 5 bytes, got %d", len(arr.Elements))
		}

		// Verify content is "Hello"
		expected := "Hello"
		for i, elem := range arr.Elements {
			b := elem.(*object.Byte)
			if b.Value != expected[i] {
				t.Errorf("byte %d: expected %c, got %c", i, expected[i], b.Value)
			}
		}

		closeFn(handle)
	})

	t.Run("read from closed handle", func(t *testing.T) {
		path := createTempFile(t, dir, "read_closed.txt", "content")

		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)
		closeFn(handle)

		// Try to read from closed handle
		readResult := readFn(handle, &object.Integer{Value: 5})
		errStruct := assertHasError(t, readResult)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7050" {
			t.Errorf("expected E7050, got %s", code.Value)
		}
	})

	t.Run("negative byte count", func(t *testing.T) {
		path := createTempFile(t, dir, "read_negative.txt", "content")
		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		readResult := readFn(handle, &object.Integer{Value: -1})
		_, ok := readResult.(*object.Error)
		if !ok {
			t.Errorf("expected Error for negative count, got %T", readResult)
		}

		closeFn(handle)
	})
}

func TestIOReadAll(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := IOBuiltins["io.open"].Fn
	readAllFn := IOBuiltins["io.read_all"].Fn
	closeFn := IOBuiltins["io.close"].Fn

	t.Run("read all bytes", func(t *testing.T) {
		content := "Hello, World!"
		path := createTempFile(t, dir, "readall.txt", content)

		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		readResult := readAllFn(handle)
		assertNoError(t, readResult)

		vals := getReturnValues(t, readResult)
		arr := vals[0].(*object.Array)
		if len(arr.Elements) != len(content) {
			t.Errorf("expected %d bytes, got %d", len(content), len(arr.Elements))
		}

		closeFn(handle)
	})

	t.Run("read all from closed handle", func(t *testing.T) {
		path := createTempFile(t, dir, "readall_closed.txt", "content")
		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)
		closeFn(handle)

		readResult := readAllFn(handle)
		errStruct := assertHasError(t, readResult)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7050" {
			t.Errorf("expected E7050, got %s", code.Value)
		}
	})
}

func TestIOReadString(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := IOBuiltins["io.open"].Fn
	readStringFn := IOBuiltins["io.read_string"].Fn
	closeFn := IOBuiltins["io.close"].Fn

	t.Run("read string from file", func(t *testing.T) {
		content := "Hello, World!"
		path := createTempFile(t, dir, "readstring.txt", content)

		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		readResult := readStringFn(handle, &object.Integer{Value: 5})
		assertNoError(t, readResult)

		vals := getReturnValues(t, readResult)
		str, ok := vals[0].(*object.String)
		if !ok {
			t.Fatalf("expected String, got %T", vals[0])
		}
		if str.Value != "Hello" {
			t.Errorf("expected 'Hello', got %q", str.Value)
		}

		closeFn(handle)
	})
}

func TestIOWriteHandle(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := IOBuiltins["io.open"].Fn
	writeFn := IOBuiltins["io.write"].Fn
	closeFn := IOBuiltins["io.close"].Fn

	t.Run("write string to file", func(t *testing.T) {
		path := filepath.Join(dir, "write_string.txt")
		mode := &object.Integer{Value: int64(os.O_WRONLY | os.O_CREATE)}

		result := openFn(&object.String{Value: path}, mode)
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		writeResult := writeFn(handle, &object.String{Value: "Hello, World!"})
		assertNoError(t, writeResult)

		vals := getReturnValues(t, writeResult)
		bytesWritten, ok := vals[0].(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", vals[0])
		}
		if bytesWritten.Value != 13 {
			t.Errorf("expected 13 bytes written, got %d", bytesWritten.Value)
		}

		closeFn(handle)

		// Verify content
		data, _ := os.ReadFile(path)
		if string(data) != "Hello, World!" {
			t.Errorf("expected 'Hello, World!', got %q", string(data))
		}
	})

	t.Run("write byte array to file", func(t *testing.T) {
		path := filepath.Join(dir, "write_bytes.txt")
		mode := &object.Integer{Value: int64(os.O_WRONLY | os.O_CREATE)}

		result := openFn(&object.String{Value: path}, mode)
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		byteArr := &object.Array{
			Elements:    []object.Object{&object.Byte{Value: 0x48}, &object.Byte{Value: 0x69}},
			ElementType: "byte",
		}

		writeResult := writeFn(handle, byteArr)
		assertNoError(t, writeResult)

		closeFn(handle)

		// Verify content
		data, _ := os.ReadFile(path)
		if string(data) != "Hi" {
			t.Errorf("expected 'Hi', got %q", string(data))
		}
	})

	t.Run("write to closed handle", func(t *testing.T) {
		path := filepath.Join(dir, "write_closed.txt")
		mode := &object.Integer{Value: int64(os.O_WRONLY | os.O_CREATE)}

		result := openFn(&object.String{Value: path}, mode)
		handle := getReturnValues(t, result)[0].(*object.FileHandle)
		closeFn(handle)

		writeResult := writeFn(handle, &object.String{Value: "test"})
		errStruct := assertHasError(t, writeResult)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7050" {
			t.Errorf("expected E7050, got %s", code.Value)
		}
	})
}

func TestIOSeekAndTell(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := IOBuiltins["io.open"].Fn
	seekFn := IOBuiltins["io.seek"].Fn
	tellFn := IOBuiltins["io.tell"].Fn
	readStringFn := IOBuiltins["io.read_string"].Fn
	closeFn := IOBuiltins["io.close"].Fn

	t.Run("seek and read", func(t *testing.T) {
		content := "Hello, World!"
		path := createTempFile(t, dir, "seek_test.txt", content)

		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		// Seek to position 7
		seekResult := seekFn(handle, &object.Integer{Value: 7}, &object.Integer{Value: 0}) // SEEK_START
		assertNoError(t, seekResult)

		vals := getReturnValues(t, seekResult)
		pos := vals[0].(*object.Integer)
		if pos.Value != 7 {
			t.Errorf("expected position 7, got %d", pos.Value)
		}

		// Read from current position
		readResult := readStringFn(handle, &object.Integer{Value: 5})
		assertNoError(t, readResult)

		str := getReturnValues(t, readResult)[0].(*object.String)
		if str.Value != "World" {
			t.Errorf("expected 'World', got %q", str.Value)
		}

		closeFn(handle)
	})

	t.Run("tell position", func(t *testing.T) {
		content := "Hello"
		path := createTempFile(t, dir, "tell_test.txt", content)

		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		// Initial position should be 0
		tellResult := tellFn(handle)
		assertNoError(t, tellResult)
		pos := getReturnValues(t, tellResult)[0].(*object.Integer)
		if pos.Value != 0 {
			t.Errorf("expected position 0, got %d", pos.Value)
		}

		// Read 3 bytes
		readStringFn(handle, &object.Integer{Value: 3})

		// Position should now be 3
		tellResult = tellFn(handle)
		pos = getReturnValues(t, tellResult)[0].(*object.Integer)
		if pos.Value != 3 {
			t.Errorf("expected position 3, got %d", pos.Value)
		}

		closeFn(handle)
	})

	t.Run("seek from end", func(t *testing.T) {
		content := "Hello"
		path := createTempFile(t, dir, "seek_end.txt", content)

		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		// Seek to 2 bytes before end
		seekResult := seekFn(handle, &object.Integer{Value: -2}, &object.Integer{Value: 2}) // SEEK_END
		assertNoError(t, seekResult)

		pos := getReturnValues(t, seekResult)[0].(*object.Integer)
		if pos.Value != 3 {
			t.Errorf("expected position 3, got %d", pos.Value)
		}

		closeFn(handle)
	})

	t.Run("seek on closed handle", func(t *testing.T) {
		path := createTempFile(t, dir, "seek_closed.txt", "test")
		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)
		closeFn(handle)

		seekResult := seekFn(handle, &object.Integer{Value: 0}, &object.Integer{Value: 0})
		errStruct := assertHasError(t, seekResult)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7050" {
			t.Errorf("expected E7050, got %s", code.Value)
		}
	})

	t.Run("tell on closed handle", func(t *testing.T) {
		path := createTempFile(t, dir, "tell_closed.txt", "test")
		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)
		closeFn(handle)

		tellResult := tellFn(handle)
		errStruct := assertHasError(t, tellResult)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7050" {
			t.Errorf("expected E7050, got %s", code.Value)
		}
	})
}

func TestIOFlushAndClose(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := IOBuiltins["io.open"].Fn
	writeFn := IOBuiltins["io.write"].Fn
	flushFn := IOBuiltins["io.flush"].Fn
	closeFn := IOBuiltins["io.close"].Fn

	t.Run("flush written data", func(t *testing.T) {
		path := filepath.Join(dir, "flush_test.txt")
		mode := &object.Integer{Value: int64(os.O_WRONLY | os.O_CREATE)}

		result := openFn(&object.String{Value: path}, mode)
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		writeFn(handle, &object.String{Value: "test"})

		flushResult := flushFn(handle)
		assertNoError(t, flushResult)

		vals := getReturnValues(t, flushResult)
		success := vals[0].(*object.Boolean)
		if !success.Value {
			t.Error("flush should return true")
		}

		closeFn(handle)
	})

	t.Run("close file handle", func(t *testing.T) {
		path := createTempFile(t, dir, "close_test.txt", "content")

		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		if handle.IsClosed {
			t.Error("handle should not be closed initially")
		}

		closeResult := closeFn(handle)
		assertNoError(t, closeResult)

		if !handle.IsClosed {
			t.Error("handle should be closed after io.close()")
		}
	})

	t.Run("close already closed handle succeeds", func(t *testing.T) {
		path := createTempFile(t, dir, "close_twice.txt", "content")

		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		closeFn(handle)
		closeResult := closeFn(handle)
		assertNoError(t, closeResult)
	})

	t.Run("flush on closed handle", func(t *testing.T) {
		path := createTempFile(t, dir, "flush_closed.txt", "content")

		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)
		closeFn(handle)

		flushResult := flushFn(handle)
		errStruct := assertHasError(t, flushResult)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7050" {
			t.Errorf("expected E7050, got %s", code.Value)
		}
	})
}

func TestIOFileHandleInspect(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := IOBuiltins["io.open"].Fn
	closeFn := IOBuiltins["io.close"].Fn

	t.Run("open handle inspect", func(t *testing.T) {
		path := createTempFile(t, dir, "inspect.txt", "content")

		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		inspect := handle.Inspect()
		if inspect != "<FileHandle "+path+">" {
			t.Errorf("expected '<FileHandle %s>', got %q", path, inspect)
		}

		closeFn(handle)
	})

	t.Run("closed handle inspect", func(t *testing.T) {
		path := createTempFile(t, dir, "inspect_closed.txt", "content")

		result := openFn(&object.String{Value: path})
		handle := getReturnValues(t, result)[0].(*object.FileHandle)
		closeFn(handle)

		inspect := handle.Inspect()
		if inspect != "<FileHandle(closed) "+path+">" {
			t.Errorf("expected '<FileHandle(closed) %s>', got %q", path, inspect)
		}
	})
}

// ============================================================================
// File Handle Integration Tests
// ============================================================================

func TestIOFileHandleIntegration(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	openFn := IOBuiltins["io.open"].Fn
	readFn := IOBuiltins["io.read"].Fn
	writeFn := IOBuiltins["io.write"].Fn
	seekFn := IOBuiltins["io.seek"].Fn
	closeFn := IOBuiltins["io.close"].Fn

	t.Run("write then read with seek", func(t *testing.T) {
		path := filepath.Join(dir, "rw_test.txt")
		mode := &object.Integer{Value: int64(os.O_RDWR | os.O_CREATE)}

		// Open for read/write
		result := openFn(&object.String{Value: path}, mode)
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		// Write data
		writeFn(handle, &object.String{Value: "Hello, World!"})

		// Seek back to start
		seekFn(handle, &object.Integer{Value: 0}, &object.Integer{Value: 0})

		// Read data back
		readResult := readFn(handle, &object.Integer{Value: 5})
		assertNoError(t, readResult)

		arr := getReturnValues(t, readResult)[0].(*object.Array)
		result_str := ""
		for _, elem := range arr.Elements {
			result_str += string(elem.(*object.Byte).Value)
		}
		if result_str != "Hello" {
			t.Errorf("expected 'Hello', got %q", result_str)
		}

		closeFn(handle)
	})

	t.Run("append mode", func(t *testing.T) {
		path := createTempFile(t, dir, "append_test.txt", "Hello")

		// Open in append mode
		mode := &object.Integer{Value: int64(os.O_WRONLY | os.O_APPEND)}
		result := openFn(&object.String{Value: path}, mode)
		handle := getReturnValues(t, result)[0].(*object.FileHandle)

		// Write more data
		writeFn(handle, &object.String{Value: ", World!"})
		closeFn(handle)

		// Verify content
		data, _ := os.ReadFile(path)
		if string(data) != "Hello, World!" {
			t.Errorf("expected 'Hello, World!', got %q", string(data))
		}
	})
}

// ============================================================================
// Phase 6: Convenience Functions Tests
// ============================================================================

func TestIOReadLines(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	readLinesFn := IOBuiltins["io.read_lines"].Fn

	t.Run("read multiple lines", func(t *testing.T) {
		content := "line1\nline2\nline3"
		path := createTempFile(t, dir, "lines.txt", content)

		result := readLinesFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		arr, ok := vals[0].(*object.Array)
		if !ok {
			t.Fatalf("expected Array, got %T", vals[0])
		}
		if len(arr.Elements) != 3 {
			t.Errorf("expected 3 lines, got %d", len(arr.Elements))
		}
		if arr.Elements[0].(*object.String).Value != "line1" {
			t.Errorf("expected 'line1', got %q", arr.Elements[0].(*object.String).Value)
		}
	})

	t.Run("handle CRLF line endings", func(t *testing.T) {
		content := "line1\r\nline2\r\nline3"
		path := createTempFile(t, dir, "crlf.txt", content)

		result := readLinesFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		arr := vals[0].(*object.Array)
		if len(arr.Elements) != 3 {
			t.Errorf("expected 3 lines, got %d", len(arr.Elements))
		}
	})

	t.Run("handle trailing newline", func(t *testing.T) {
		content := "line1\nline2\n"
		path := createTempFile(t, dir, "trailing.txt", content)

		result := readLinesFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		arr := vals[0].(*object.Array)
		if len(arr.Elements) != 2 {
			t.Errorf("expected 2 lines, got %d", len(arr.Elements))
		}
	})

	t.Run("empty file", func(t *testing.T) {
		path := createTempFile(t, dir, "empty.txt", "")

		result := readLinesFn(&object.String{Value: path})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		arr := vals[0].(*object.Array)
		if len(arr.Elements) != 0 {
			t.Errorf("expected 0 lines, got %d", len(arr.Elements))
		}
	})

	t.Run("directory error", func(t *testing.T) {
		result := readLinesFn(&object.String{Value: dir})
		errStruct := assertHasError(t, result)
		code := errStruct.Fields["code"].(*object.String)
		if code.Value != "E7042" {
			t.Errorf("expected E7042, got %s", code.Value)
		}
	})
}

func TestIOAppendLine(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	appendLineFn := IOBuiltins["io.append_line"].Fn

	t.Run("append to new file", func(t *testing.T) {
		path := filepath.Join(dir, "newfile.txt")

		result := appendLineFn(&object.String{Value: path}, &object.String{Value: "first line"})
		assertNoError(t, result)

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(data) != "first line\n" {
			t.Errorf("expected 'first line\\n', got %q", string(data))
		}
	})

	t.Run("append multiple lines", func(t *testing.T) {
		path := filepath.Join(dir, "multiline.txt")

		appendLineFn(&object.String{Value: path}, &object.String{Value: "line1"})
		appendLineFn(&object.String{Value: path}, &object.String{Value: "line2"})
		appendLineFn(&object.String{Value: path}, &object.String{Value: "line3"})

		data, _ := os.ReadFile(path)
		expected := "line1\nline2\nline3\n"
		if string(data) != expected {
			t.Errorf("expected %q, got %q", expected, string(data))
		}
	})

	t.Run("with custom permissions", func(t *testing.T) {
		path := filepath.Join(dir, "perms.txt")

		result := appendLineFn(
			&object.String{Value: path},
			&object.String{Value: "test"},
			&object.Integer{Value: 0600},
		)
		assertNoError(t, result)

		if runtime.GOOS != "windows" {
			info, _ := os.Stat(path)
			if info.Mode().Perm() != 0600 {
				t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
			}
		}
	})
}

func TestIOExpandPath(t *testing.T) {
	expandPathFn := IOBuiltins["io.expand_path"].Fn

	t.Run("expand home directory", func(t *testing.T) {
		result := expandPathFn(&object.String{Value: "~/test"})
		str, ok := result.(*object.String)
		if !ok {
			t.Fatalf("expected String, got %T", result)
		}

		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, "test")
		if str.Value != expected {
			t.Errorf("expected %q, got %q", expected, str.Value)
		}
	})

	t.Run("clean path", func(t *testing.T) {
		result := expandPathFn(&object.String{Value: "/foo/bar/../baz"})
		str := result.(*object.String)
		expected := filepath.Clean("/foo/bar/../baz")
		if str.Value != expected {
			t.Errorf("expected %q, got %q", expected, str.Value)
		}
	})

	t.Run("no expansion needed", func(t *testing.T) {
		result := expandPathFn(&object.String{Value: "/absolute/path"})
		str := result.(*object.String)
		if str.Value != "/absolute/path" {
			t.Errorf("expected '/absolute/path', got %q", str.Value)
		}
	})
}

// ============================================================================
// Phase 6: Optional Permissions Tests
// ============================================================================

func TestIOWriteFileWithPerms(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	writeFileFn := IOBuiltins["io.write_file"].Fn

	t.Run("write with custom permissions", func(t *testing.T) {
		path := filepath.Join(dir, "perms_write.txt")

		result := writeFileFn(
			&object.String{Value: path},
			&object.String{Value: "test content"},
			&object.Integer{Value: 0600},
		)
		assertNoError(t, result)

		if runtime.GOOS != "windows" {
			info, _ := os.Stat(path)
			if info.Mode().Perm() != 0600 {
				t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
			}
		}
	})

	t.Run("default permissions", func(t *testing.T) {
		path := filepath.Join(dir, "default_perms.txt")

		result := writeFileFn(
			&object.String{Value: path},
			&object.String{Value: "test content"},
		)
		assertNoError(t, result)

		if runtime.GOOS != "windows" {
			info, _ := os.Stat(path)
			if info.Mode().Perm() != 0644 {
				t.Errorf("expected permissions 0644, got %o", info.Mode().Perm())
			}
		}
	})
}

func TestIOMkdirWithPerms(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	mkdirFn := IOBuiltins["io.mkdir"].Fn

	t.Run("mkdir with custom permissions", func(t *testing.T) {
		path := filepath.Join(dir, "custom_perms_dir")

		result := mkdirFn(
			&object.String{Value: path},
			&object.Integer{Value: 0700},
		)
		assertNoError(t, result)

		if runtime.GOOS != "windows" {
			info, _ := os.Stat(path)
			if info.Mode().Perm() != 0700 {
				t.Errorf("expected permissions 0700, got %o", info.Mode().Perm())
			}
		}
	})
}

func TestIOMkdirAllWithPerms(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	mkdirAllFn := IOBuiltins["io.mkdir_all"].Fn

	t.Run("mkdir_all with custom permissions", func(t *testing.T) {
		path := filepath.Join(dir, "a", "b", "c")

		result := mkdirAllFn(
			&object.String{Value: path},
			&object.Integer{Value: 0700},
		)
		assertNoError(t, result)

		if runtime.GOOS != "windows" {
			info, _ := os.Stat(path)
			if info.Mode().Perm() != 0700 {
				t.Errorf("expected permissions 0700, got %o", info.Mode().Perm())
			}
		}
	})
}

func TestIOCopyWithPerms(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	copyFn := IOBuiltins["io.copy"].Fn

	t.Run("copy with custom permissions", func(t *testing.T) {
		srcPath := createTempFile(t, dir, "source.txt", "content")
		dstPath := filepath.Join(dir, "dest.txt")

		result := copyFn(
			&object.String{Value: srcPath},
			&object.String{Value: dstPath},
			&object.Integer{Value: 0600},
		)
		assertNoError(t, result)

		if runtime.GOOS != "windows" {
			info, _ := os.Stat(dstPath)
			if info.Mode().Perm() != 0600 {
				t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
			}
		}
	})

	t.Run("copy preserves source permissions by default", func(t *testing.T) {
		srcPath := filepath.Join(dir, "source_perm.txt")
		os.WriteFile(srcPath, []byte("content"), 0755)
		dstPath := filepath.Join(dir, "dest_perm.txt")

		result := copyFn(
			&object.String{Value: srcPath},
			&object.String{Value: dstPath},
		)
		assertNoError(t, result)

		if runtime.GOOS != "windows" {
			srcInfo, _ := os.Stat(srcPath)
			dstInfo, _ := os.Stat(dstPath)
			if srcInfo.Mode().Perm() != dstInfo.Mode().Perm() {
				t.Errorf("expected permissions %o, got %o", srcInfo.Mode().Perm(), dstInfo.Mode().Perm())
			}
		}
	})
}

// ============================================================================
// Filesystem Utilities Tests
// ============================================================================

func TestIOGlob(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	globFn := IOBuiltins["io.glob"].Fn

	// Create test files
	createTempFile(t, dir, "file1.txt", "content1")
	createTempFile(t, dir, "file2.txt", "content2")
	createTempFile(t, dir, "file3.go", "content3")

	t.Run("glob matches txt files", func(t *testing.T) {
		pattern := filepath.Join(dir, "*.txt")
		result := globFn(&object.String{Value: pattern})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		arr, ok := vals[0].(*object.Array)
		if !ok {
			t.Fatalf("expected Array, got %T", vals[0])
		}

		if len(arr.Elements) != 2 {
			t.Errorf("expected 2 matches, got %d", len(arr.Elements))
		}
	})

	t.Run("glob matches go files", func(t *testing.T) {
		pattern := filepath.Join(dir, "*.go")
		result := globFn(&object.String{Value: pattern})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		arr := vals[0].(*object.Array)

		if len(arr.Elements) != 1 {
			t.Errorf("expected 1 match, got %d", len(arr.Elements))
		}
	})

	t.Run("glob no matches returns empty array", func(t *testing.T) {
		pattern := filepath.Join(dir, "*.xyz")
		result := globFn(&object.String{Value: pattern})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		arr := vals[0].(*object.Array)

		if len(arr.Elements) != 0 {
			t.Errorf("expected 0 matches, got %d", len(arr.Elements))
		}
	})

	t.Run("glob invalid pattern returns error", func(t *testing.T) {
		result := globFn(&object.String{Value: "["}) // Invalid pattern
		vals := getReturnValues(t, result)
		if vals[1] == object.NIL {
			t.Error("expected error for invalid pattern")
		}
	})
}

func TestIOWalk(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	walkFn := IOBuiltins["io.walk"].Fn

	// Create directory structure
	subdir := filepath.Join(dir, "subdir")
	os.Mkdir(subdir, 0755)

	createTempFile(t, dir, "root.txt", "root content")
	createTempFile(t, subdir, "nested.txt", "nested content")

	t.Run("walk finds all files", func(t *testing.T) {
		result := walkFn(&object.String{Value: dir})
		assertNoError(t, result)

		vals := getReturnValues(t, result)
		arr, ok := vals[0].(*object.Array)
		if !ok {
			t.Fatalf("expected Array, got %T", vals[0])
		}

		// Should find 2 files (root.txt and subdir/nested.txt)
		if len(arr.Elements) != 2 {
			t.Errorf("expected 2 files, got %d", len(arr.Elements))
		}
	})

	t.Run("walk non-existent directory returns error", func(t *testing.T) {
		result := walkFn(&object.String{Value: "/nonexistent/path"})
		vals := getReturnValues(t, result)
		if vals[1] == object.NIL {
			t.Error("expected error for non-existent directory")
		}
	})
}

func TestIOIsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests not reliable on Windows")
	}

	dir, cleanup := createTempDir(t)
	defer cleanup()

	isSymlinkFn := IOBuiltins["io.is_symlink"].Fn

	// Create a regular file
	regularFile := createTempFile(t, dir, "regular.txt", "content")

	// Create a symlink
	symlinkPath := filepath.Join(dir, "symlink.txt")
	os.Symlink(regularFile, symlinkPath)

	t.Run("regular file is not symlink", func(t *testing.T) {
		result := isSymlinkFn(&object.String{Value: regularFile})
		if result != object.FALSE {
			t.Error("expected regular file to not be symlink")
		}
	})

	t.Run("symlink is symlink", func(t *testing.T) {
		result := isSymlinkFn(&object.String{Value: symlinkPath})
		if result != object.TRUE {
			t.Error("expected symlink to be detected as symlink")
		}
	})

	t.Run("non-existent path returns false", func(t *testing.T) {
		result := isSymlinkFn(&object.String{Value: "/nonexistent/path"})
		if result != object.FALSE {
			t.Error("expected non-existent path to return false")
		}
	})

	t.Run("directory is not symlink", func(t *testing.T) {
		result := isSymlinkFn(&object.String{Value: dir})
		if result != object.FALSE {
			t.Error("expected directory to not be symlink")
		}
	})
}
