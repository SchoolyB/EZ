package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/marshallburns/ez/pkg/object"
)

// validatePath checks for invalid path inputs (empty or containing null bytes)
// Returns nil if valid, or an error tuple if invalid
func validatePath(path string, funcName string) *object.ReturnValue {
	if path == "" {
		return &object.ReturnValue{Values: []object.Object{
			object.NIL,
			CreateStdlibError("E7040", fmt.Sprintf("%s: path cannot be empty", funcName)),
		}}
	}
	if strings.ContainsRune(path, '\x00') {
		return &object.ReturnValue{Values: []object.Object{
			object.NIL,
			CreateStdlibError("E7041", fmt.Sprintf("%s: path contains null byte", funcName)),
		}}
	}
	return nil
}

// validatePathBool is like validatePath but returns false instead of nil for boolean-returning functions
func validatePathBool(path string) bool {
	return path != "" && !strings.ContainsRune(path, '\x00')
}

// atomicWriteFile writes data to a file atomically by writing to a temp file first,
// then renaming. This ensures the file is never left in a partial state.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	// Get the directory of the target file
	dir := filepath.Dir(path)

	// Create a temp file in the same directory (ensures same filesystem for atomic rename)
	tmpFile, err := os.CreateTemp(dir, ".ez_tmp_*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on any error
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// Set permissions before writing (for security)
	if err := tmpFile.Chmod(perm); err != nil {
		_ = tmpFile.Close() // Error ignored; already returning an error
		return fmt.Errorf("set permissions: %w", err)
	}

	// Write data
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close() // Error ignored; already returning an error
		return fmt.Errorf("write data: %w", err)
	}

	// Sync to ensure data is on disk before rename
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close() // Error ignored; already returning an error
		return fmt.Errorf("sync: %w", err)
	}

	// Close before rename
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	success = true
	return nil
}

// IOBuiltins contains the io module functions for file system operations
var IOBuiltins = map[string]*object.Builtin{
	// ============================================================================
	// File Reading
	// ============================================================================

	// read_file reads the entire contents of a file as a string.
	// Takes file path. Returns (content, Error) tuple.
	"io.read_file": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.read_file() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.read_file() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.read_file()"); err != nil {
				return err
			}

			// Check if path is a directory
			info, statErr := os.Stat(path.Value)
			if statErr == nil && info.IsDir() {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7042", "io.read_file(): cannot read directory as file"),
				}}
			}

			content, err := os.ReadFile(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "read")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: string(content)},
				object.NIL,
			}}
		},
	},

	// read_bytes reads the entire contents of a file as a byte array.
	// Takes file path. Returns ([byte], Error) tuple.
	"io.read_bytes": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.read_bytes() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.read_bytes() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.read_bytes()"); err != nil {
				return err
			}

			// Check if path is a directory
			info, statErr := os.Stat(path.Value)
			if statErr == nil && info.IsDir() {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7042", "io.read_bytes(): cannot read directory as file"),
				}}
			}

			content, err := os.ReadFile(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "read")
			}

			// Convert to byte array
			elements := make([]object.Object, len(content))
			for i, b := range content {
				elements[i] = &object.Byte{Value: b}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Array{Elements: elements, ElementType: "byte"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// File Writing
	// ============================================================================

	// write_file writes content to a file atomically, creating or overwriting.
	// Takes path, content, and optional permissions (default 0644). Returns (bool, Error) tuple.
	"io.write_file": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: "io.write_file() takes 2-3 arguments (path, content, [perms])"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.write_file() requires a string path as first argument"}
			}
			content, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.write_file() requires a string content as second argument"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.write_file()"); err != nil {
				return err
			}

			// Default permissions
			perms := os.FileMode(0644)
			if len(args) == 3 {
				permVal, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "io.write_file() permissions must be an integer"}
				}
				perms = os.FileMode(permVal.Value.Int64())
			}

			err := atomicWriteFile(path.Value, []byte(content.Value), perms)
			if err != nil {
				return CreateStdlibErrorResult(err, "write")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// write_bytes writes a byte array to a file atomically, creating or overwriting.
	// Takes path, byte array, and optional permissions. Returns (bool, Error) tuple.
	"io.write_bytes": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: "io.write_bytes() takes 2-3 arguments (path, data, [perms])"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.write_bytes() requires a string path as first argument"}
			}
			data, ok := args[1].(*object.Array)
			if !ok || data.ElementType != "byte" {
				return &object.Error{Code: "E7002", Message: "io.write_bytes() requires a byte array as second argument"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.write_bytes()"); err != nil {
				return err
			}

			// Default permissions
			perms := os.FileMode(0644)
			if len(args) == 3 {
				permVal, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "io.write_bytes() permissions must be an integer"}
				}
				perms = os.FileMode(permVal.Value.Int64())
			}

			// Convert byte array to []byte
			bytes := make([]byte, len(data.Elements))
			for i, elem := range data.Elements {
				b, ok := elem.(*object.Byte)
				if !ok {
					return &object.Error{Code: "E7010", Message: fmt.Sprintf("io.write_bytes() byte array element %d is not a byte", i)}
				}
				bytes[i] = b.Value
			}

			err := atomicWriteFile(path.Value, bytes, perms)
			if err != nil {
				return CreateStdlibErrorResult(err, "write bytes")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// append_file appends content to a file, creating it if needed.
	// Takes path, content, and optional permissions. Returns (bool, Error) tuple.
	"io.append_file": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: "io.append_file() takes 2-3 arguments (path, content, [perms])"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.append_file() requires a string path as first argument"}
			}
			content, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.append_file() requires a string content as second argument"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.append_file()"); err != nil {
				return err
			}

			// Default permissions
			perms := os.FileMode(0644)
			if len(args) == 3 {
				permVal, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "io.append_file() permissions must be an integer"}
				}
				perms = os.FileMode(permVal.Value.Int64())
			}

			f, err := os.OpenFile(path.Value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perms)
			if err != nil {
				return CreateStdlibErrorResult(err, "open for append")
			}

			_, err = f.WriteString(content.Value)
			if err != nil {
				_ = f.Close()
				return CreateStdlibErrorResult(err, "append")
			}

			if err := f.Close(); err != nil {
				return CreateStdlibErrorResult(err, "close after append")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// Convenience Functions
	// ============================================================================

	// read_lines reads a file and returns its content as an array of lines.
	// Takes file path. Returns ([string], Error) tuple.
	"io.read_lines": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.read_lines() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.read_lines() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.read_lines()"); err != nil {
				return err
			}

			// Check if path is a directory
			info, err := os.Stat(path.Value)
			if err == nil && info.IsDir() {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7042", "io.read_lines(): cannot read directory as file"),
				}}
			}

			content, err := os.ReadFile(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "read")
			}

			// Split by newlines, handling both \n and \r\n
			text := strings.ReplaceAll(string(content), "\r\n", "\n")
			rawLines := strings.Split(text, "\n")

			// Remove trailing empty line if file ends with newline
			if len(rawLines) > 0 && rawLines[len(rawLines)-1] == "" {
				rawLines = rawLines[:len(rawLines)-1]
			}

			// Convert to EZ array
			lines := make([]object.Object, len(rawLines))
			for i, line := range rawLines {
				lines[i] = &object.String{Value: line}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Array{Elements: lines, ElementType: "string"},
				object.NIL,
			}}
		},
	},

	// append_line appends a line to a file with trailing newline.
	// Takes path, line, and optional permissions. Returns (bool, Error) tuple.
	"io.append_line": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: "io.append_line() takes 2-3 arguments (path, line, [perms])"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.append_line() requires a string path as first argument"}
			}
			line, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.append_line() requires a string as second argument"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.append_line()"); err != nil {
				return err
			}

			// Default permissions
			perms := os.FileMode(0644)
			if len(args) == 3 {
				permVal, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "io.append_line() permissions must be an integer"}
				}
				perms = os.FileMode(permVal.Value.Int64())
			}

			// Check if file exists and doesn't end with newline
			needsLeadingNewline := false
			if info, statErr := os.Stat(path.Value); statErr == nil && info.Size() > 0 {
				// File exists and has content, check if it ends with newline
				rf, readErr := os.Open(path.Value)
				if readErr == nil {
					// Seek to last byte
					_, _ = rf.Seek(-1, 2) // 2 = io.SeekEnd
					lastByte := make([]byte, 1)
					if n, _ := rf.Read(lastByte); n == 1 && lastByte[0] != '\n' {
						needsLeadingNewline = true
					}
					_ = rf.Close()
				}
			}

			f, err := os.OpenFile(path.Value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perms)
			if err != nil {
				return CreateStdlibErrorResult(err, "open for append")
			}

			// Write line with newline, prepending newline if file didn't end with one
			content := line.Value + "\n"
			if needsLeadingNewline {
				content = "\n" + content
			}
			_, err = f.WriteString(content)
			if err != nil {
				_ = f.Close()
				return CreateStdlibErrorResult(err, "append line")
			}

			if err := f.Close(); err != nil {
				return CreateStdlibErrorResult(err, "close after append")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// expand_path expands ~ to home directory and cleans the path.
	// Takes path string. Returns expanded path string.
	"io.expand_path": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.expand_path() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.expand_path() requires a string path"}
			}

			result := path.Value

			// Expand ~ to home directory
			if strings.HasPrefix(result, "~") {
				home, err := os.UserHomeDir()
				if err != nil {
					return &object.ReturnValue{Values: []object.Object{
						object.NIL,
						CreateStdlibError("E7029", "io.expand_path(): failed to get home directory"),
					}}
				}
				result = home + result[1:]
			}

			// Clean the path
			result = filepath.Clean(result)

			return &object.String{Value: result}
		},
	},

	// ============================================================================
	// Path Existence and Type Checks
	// ============================================================================

	// exists checks if a path exists (file or directory).
	// Takes path string. Returns bool.
	"io.exists": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.exists() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.exists() requires a string path"}
			}

			// Invalid paths don't exist
			if !validatePathBool(path.Value) {
				return object.FALSE
			}

			_, err := os.Stat(path.Value)
			if err == nil {
				return object.TRUE
			}
			if os.IsNotExist(err) {
				return object.FALSE
			}
			// For other errors (permission denied, etc.), return false
			return object.FALSE
		},
	},

	// is_file checks if a path is a regular file.
	// Takes path string. Returns bool.
	"io.is_file": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.is_file() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.is_file() requires a string path"}
			}

			// Invalid paths are not files
			if !validatePathBool(path.Value) {
				return object.FALSE
			}

			info, err := os.Stat(path.Value)
			if err != nil {
				return object.FALSE
			}
			if info.Mode().IsRegular() {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// is_dir checks if a path is a directory.
	// Takes path string. Returns bool.
	"io.is_dir": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.is_dir() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.is_dir() requires a string path"}
			}

			// Invalid paths are not directories
			if !validatePathBool(path.Value) {
				return object.FALSE
			}

			info, err := os.Stat(path.Value)
			if err != nil {
				return object.FALSE
			}
			if info.IsDir() {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// ============================================================================
	// File Operations
	// ============================================================================

	// remove removes a file (not directories).
	// Takes file path. Returns (bool, Error) tuple.
	"io.remove": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.remove() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.remove() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.remove()"); err != nil {
				return err
			}

			// Check if it's a directory - if so, don't allow removal with this function
			info, err := os.Stat(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "stat")
			}
			if info.IsDir() {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					CreateStdlibError("E7018", "io.remove() cannot remove directories, use io.remove_dir() instead"),
				}}
			}

			err = os.Remove(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "remove")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// remove_dir removes an empty directory.
	// Takes directory path. Returns (bool, Error) tuple.
	"io.remove_dir": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.remove_dir() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.remove_dir() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.remove_dir()"); err != nil {
				return err
			}

			// Check if it's actually a directory
			info, err := os.Stat(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "stat")
			}
			if !info.IsDir() {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					CreateStdlibError("E7019", "io.remove_dir() can only remove directories, use io.remove() for files"),
				}}
			}

			err = os.Remove(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "remove directory")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// remove_all removes a file or directory recursively.
	// Takes path. Returns (bool, Error) tuple. Use with caution.
	"io.remove_all": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.remove_all() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.remove_all() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.remove_all()"); err != nil {
				return err
			}

			// Safety check: don't allow removing root or home directory
			absPath, err := filepath.Abs(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "resolve path")
			}

			// Check for dangerous paths
			if absPath == "/" || absPath == filepath.Clean(os.Getenv("HOME")) {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					CreateStdlibError("E7020", "io.remove_all() cannot remove root or home directory for safety"),
				}}
			}

			err = os.RemoveAll(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "remove all")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// rename renames or moves a file or directory.
	// Takes old_path and new_path. Returns (bool, Error) tuple.
	"io.rename": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "io.rename() takes exactly 2 arguments (old_path, new_path)"}
			}
			oldPath, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.rename() requires a string as first argument (old_path)"}
			}
			newPath, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.rename() requires a string as second argument (new_path)"}
			}

			// Validate paths
			if err := validatePath(oldPath.Value, "io.rename()"); err != nil {
				return err
			}
			if err := validatePath(newPath.Value, "io.rename()"); err != nil {
				return err
			}

			err := os.Rename(oldPath.Value, newPath.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "rename")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// copy copies a file from source to destination.
	// Takes source, destination, and optional permissions. Returns (bool, Error) tuple.
	"io.copy": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: "io.copy() takes 2-3 arguments (source, destination, [perms])"}
			}
			srcPath, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.copy() requires a string as first argument (source)"}
			}
			dstPath, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.copy() requires a string as second argument (destination)"}
			}

			// Validate paths
			if err := validatePath(srcPath.Value, "io.copy()"); err != nil {
				return err
			}
			if err := validatePath(dstPath.Value, "io.copy()"); err != nil {
				return err
			}

			// Check source exists and is a file
			srcInfo, err := os.Stat(srcPath.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "stat source")
			}
			if srcInfo.IsDir() {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					CreateStdlibError("E7021", "io.copy() cannot copy directories, only files"),
				}}
			}

			// Determine permissions: explicit or preserve from source
			var perms os.FileMode
			if len(args) == 3 {
				permVal, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "io.copy() permissions must be an integer"}
				}
				perms = os.FileMode(permVal.Value.Int64())
			} else {
				perms = srcInfo.Mode()
			}

			// Open source file
			src, err := os.Open(srcPath.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "open source")
			}
			defer src.Close() // Read-only, error on close is not critical

			// Create destination file with proper permissions
			dst, err := os.OpenFile(dstPath.Value, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perms)
			if err != nil {
				return CreateStdlibErrorResult(err, "create destination")
			}

			// Copy contents
			_, err = io.Copy(dst, src)
			if err != nil {
				_ = dst.Close() // Explicitly ignore close error; copy error takes precedence
				return CreateStdlibErrorResult(err, "copy contents")
			}

			// Close destination and check for write errors
			if err := dst.Close(); err != nil {
				return CreateStdlibErrorResult(err, "close destination")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// Directory Operations
	// ============================================================================

	// mkdir creates a directory (parent must exist).
	// Takes path and optional permissions (default 0755). Returns (bool, Error) tuple.
	"io.mkdir": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Code: "E7001", Message: "io.mkdir() takes 1-2 arguments (path, [perms])"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.mkdir() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.mkdir()"); err != nil {
				return err
			}

			// Default permissions
			perms := os.FileMode(0755)
			if len(args) == 2 {
				permVal, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "io.mkdir() permissions must be an integer"}
				}
				perms = os.FileMode(permVal.Value.Int64())
			}

			err := os.Mkdir(path.Value, perms)
			if err != nil {
				return CreateStdlibErrorResult(err, "mkdir")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// mkdir_all creates a directory and all parent directories as needed.
	// Takes path and optional permissions (default 0755). Returns (bool, Error) tuple.
	"io.mkdir_all": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Code: "E7001", Message: "io.mkdir_all() takes 1-2 arguments (path, [perms])"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.mkdir_all() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.mkdir_all()"); err != nil {
				return err
			}

			// Default permissions
			perms := os.FileMode(0755)
			if len(args) == 2 {
				permVal, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "io.mkdir_all() permissions must be an integer"}
				}
				perms = os.FileMode(permVal.Value.Int64())
			}

			err := os.MkdirAll(path.Value, perms)
			if err != nil {
				return CreateStdlibErrorResult(err, "mkdir_all")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// read_dir lists the contents of a directory.
	// Takes directory path. Returns ([string], Error) tuple of filenames.
	"io.read_dir": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.read_dir() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.read_dir() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.read_dir()"); err != nil {
				return err
			}

			entries, err := os.ReadDir(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "read directory")
			}

			elements := make([]object.Object, len(entries))
			for i, entry := range entries {
				elements[i] = &object.String{Value: entry.Name()}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Array{Elements: elements, Mutable: false, ElementType: "string"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// File Metadata
	// ============================================================================

	// file_size returns the size of a file in bytes.
	// Takes file path. Returns (int, Error) tuple.
	"io.file_size": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.file_size() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.file_size() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.file_size()"); err != nil {
				return err
			}

			info, err := os.Stat(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "stat")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(info.Size())},
				object.NIL,
			}}
		},
	},

	// file_mod_time returns the modification time as a Unix timestamp.
	// Takes file path. Returns (int, Error) tuple.
	"io.file_mod_time": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.file_mod_time() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.file_mod_time() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.file_mod_time()"); err != nil {
				return err
			}

			info, err := os.Stat(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "stat")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(info.ModTime().Unix())},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// Path Utilities
	// ============================================================================

	// path_join joins path components using OS-specific separator.
	// Takes one or more path strings. Returns joined path string.
	"io.path_join": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.Error{Code: "E7001", Message: "io.path_join() takes at least 1 argument"}
			}

			parts := make([]string, len(args))
			for i, arg := range args {
				str, ok := arg.(*object.String)
				if !ok {
					return &object.Error{Code: "E7003", Message: fmt.Sprintf("io.path_join() argument %d must be a string", i+1)}
				}
				// Check for null bytes in path components
				if strings.ContainsRune(str.Value, '\x00') {
					return &object.Error{Code: "E7041", Message: "io.path_join(): path component contains null byte"}
				}
				parts[i] = str.Value
			}

			return &object.String{Value: filepath.Join(parts...)}
		},
	},

	// path_base returns the last element of a path (filename or directory name).
	// Takes path string. Returns base name string.
	"io.path_base": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.path_base() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.path_base() requires a string path"}
			}

			// Check for null bytes
			if strings.ContainsRune(path.Value, '\x00') {
				return &object.Error{Code: "E7041", Message: "io.path_base(): path contains null byte"}
			}

			return &object.String{Value: filepath.Base(path.Value)}
		},
	},

	// path_dir returns the directory portion of a path.
	// Takes path string. Returns directory path string.
	"io.path_dir": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.path_dir() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.path_dir() requires a string path"}
			}

			// Check for null bytes
			if strings.ContainsRune(path.Value, '\x00') {
				return &object.Error{Code: "E7041", Message: "io.path_dir(): path contains null byte"}
			}

			return &object.String{Value: filepath.Dir(path.Value)}
		},
	},

	// path_ext returns the file extension including the dot.
	// Takes path string. Returns extension string (e.g., ".txt").
	"io.path_ext": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.path_ext() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.path_ext() requires a string path"}
			}

			// Check for null bytes
			if strings.ContainsRune(path.Value, '\x00') {
				return &object.Error{Code: "E7041", Message: "io.path_ext(): path contains null byte"}
			}

			return &object.String{Value: filepath.Ext(path.Value)}
		},
	},

	// path_abs returns the absolute path.
	// Takes path string. Returns (string, Error) tuple.
	"io.path_abs": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.path_abs() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.path_abs() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.path_abs()"); err != nil {
				return err
			}

			absPath, err := filepath.Abs(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "resolve absolute path")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: absPath},
				object.NIL,
			}}
		},
	},

	// path_clean cleans a path by removing redundant separators and dots.
	// Takes path string. Returns cleaned path string.
	"io.path_clean": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.path_clean() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.path_clean() requires a string path"}
			}

			// Check for null bytes
			if strings.ContainsRune(path.Value, '\x00') {
				return &object.Error{Code: "E7041", Message: "io.path_clean(): path contains null byte"}
			}

			return &object.String{Value: filepath.Clean(path.Value)}
		},
	},

	// path_separator returns the OS-specific path separator.
	// Takes no arguments. Returns "/" on Unix or "\\" on Windows.
	"io.path_separator": {
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: string(filepath.Separator)}
		},
	},

	// ============================================================================
	// File Handle Constants
	// These constants are used with io.open() to specify file access modes.
	// ============================================================================

	// READ_ONLY opens a file for reading only.
	"io.READ_ONLY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(os.O_RDONLY))}
		},
		IsConstant: true,
	},

	// WRITE_ONLY opens a file for writing only.
	"io.WRITE_ONLY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(os.O_WRONLY))}
		},
		IsConstant: true,
	},

	// READ_WRITE opens a file for reading and writing.
	"io.READ_WRITE": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(os.O_RDWR))}
		},
		IsConstant: true,
	},

	// APPEND opens a file for appending (writes go to end of file).
	"io.APPEND": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(os.O_APPEND))}
		},
		IsConstant: true,
	},

	// CREATE creates the file if it does not exist.
	"io.CREATE": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(os.O_CREATE))}
		},
		IsConstant: true,
	},

	// TRUNCATE truncates the file to zero length when opening.
	"io.TRUNCATE": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(os.O_TRUNC))}
		},
		IsConstant: true,
	},

	// EXCLUSIVE fails if file already exists when used with CREATE.
	"io.EXCLUSIVE": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(os.O_EXCL))}
		},
		IsConstant: true,
	},

	// SEEK_START seeks relative to the start of the file.
	"io.SEEK_START": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(io.SeekStart))}
		},
		IsConstant: true,
	},

	// SEEK_CURRENT seeks relative to the current file position.
	"io.SEEK_CURRENT": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(io.SeekCurrent))}
		},
		IsConstant: true,
	},

	// SEEK_END seeks relative to the end of the file.
	"io.SEEK_END": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(io.SeekEnd))}
		},
		IsConstant: true,
	},

	// ============================================================================
	// File Handle Operations
	// ============================================================================

	// open opens a file and returns a file handle.
	// Takes path, optional mode flags, and optional permissions. Returns (FileHandle, Error) tuple.
	"io.open": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: "io.open() takes 1-3 arguments (path, [mode], [perms])"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.open() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.open()"); err != nil {
				return err
			}

			// Default mode is read-only
			mode := os.O_RDONLY
			if len(args) >= 2 {
				modeVal, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "io.open() mode must be an integer"}
				}
				mode = int(modeVal.Value.Int64())
			}

			// Default permissions
			perms := os.FileMode(0644)
			if len(args) == 3 {
				permVal, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "io.open() permissions must be an integer"}
				}
				perms = os.FileMode(permVal.Value.Int64())
			}

			file, err := os.OpenFile(path.Value, mode, perms)
			if err != nil {
				return CreateStdlibErrorResult(err, "open")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.FileHandle{
					File:     file,
					Path:     path.Value,
					Mode:     mode,
					IsClosed: false,
				},
				object.NIL,
			}}
		},
	},

	// read reads n bytes from a file handle.
	// Takes handle and byte count. Returns ([byte], Error) tuple.
	"io.read": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "io.read() takes exactly 2 arguments (handle, n)"}
			}
			handle, ok := args[0].(*object.FileHandle)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.read() requires a file handle as first argument"}
			}
			if handle.IsClosed {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7050", "io.read(): file handle is closed"),
				}}
			}
			n, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "io.read() requires an integer as second argument"}
			}
			if n.Value.Sign() < 0 {
				return &object.Error{Code: "E7011", Message: "io.read() byte count cannot be negative"}
			}

			buf := make([]byte, n.Value.Int64())
			bytesRead, err := handle.File.Read(buf)

			// Handle EOF - not an error, just return what we got
			if err != nil && err != io.EOF {
				return CreateStdlibErrorResult(err, "read")
			}

			// Convert to byte array
			elements := make([]object.Object, bytesRead)
			for i := 0; i < bytesRead; i++ {
				elements[i] = &object.Byte{Value: buf[i]}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Array{Elements: elements, ElementType: "byte"},
				object.NIL,
			}}
		},
	},

	// read_all reads all remaining bytes from a file handle.
	// Takes handle. Returns ([byte], Error) tuple.
	"io.read_all": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.read_all() takes exactly 1 argument (handle)"}
			}
			handle, ok := args[0].(*object.FileHandle)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.read_all() requires a file handle"}
			}
			if handle.IsClosed {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7050", "io.read_all(): file handle is closed"),
				}}
			}

			content, err := io.ReadAll(handle.File)
			if err != nil {
				return CreateStdlibErrorResult(err, "read all")
			}

			// Convert to byte array
			elements := make([]object.Object, len(content))
			for i, b := range content {
				elements[i] = &object.Byte{Value: b}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Array{Elements: elements, ElementType: "byte"},
				object.NIL,
			}}
		},
	},

	// read_string reads n bytes from a file handle as a string.
	// Takes handle and byte count. Returns (string, Error) tuple.
	"io.read_string": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "io.read_string() takes exactly 2 arguments (handle, n)"}
			}
			handle, ok := args[0].(*object.FileHandle)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.read_string() requires a file handle as first argument"}
			}
			if handle.IsClosed {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7050", "io.read_string(): file handle is closed"),
				}}
			}
			n, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "io.read_string() requires an integer as second argument"}
			}
			if n.Value.Sign() < 0 {
				return &object.Error{Code: "E7011", Message: "io.read_string() byte count cannot be negative"}
			}

			buf := make([]byte, n.Value.Int64())
			bytesRead, err := handle.File.Read(buf)

			// Handle EOF - not an error, just return what we got
			if err != nil && err != io.EOF {
				return CreateStdlibErrorResult(err, "read string")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: string(buf[:bytesRead])},
				object.NIL,
			}}
		},
	},

	// write writes data (string or byte array) to a file handle.
	// Takes handle and data. Returns (bytes_written, Error) tuple.
	"io.write": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "io.write() takes exactly 2 arguments (handle, data)"}
			}
			handle, ok := args[0].(*object.FileHandle)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.write() requires a file handle as first argument"}
			}
			if handle.IsClosed {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7050", "io.write(): file handle is closed"),
				}}
			}

			var data []byte
			switch v := args[1].(type) {
			case *object.String:
				data = []byte(v.Value)
			case *object.Array:
				if v.ElementType != "byte" {
					return &object.Error{Code: "E7002", Message: "io.write() array must be a byte array"}
				}
				data = make([]byte, len(v.Elements))
				for i, elem := range v.Elements {
					b, ok := elem.(*object.Byte)
					if !ok {
						return &object.Error{Code: "E7010", Message: fmt.Sprintf("io.write() byte array element %d is not a byte", i)}
					}
					data[i] = b.Value
				}
			default:
				return &object.Error{Code: "E7003", Message: "io.write() data must be a string or byte array"}
			}

			bytesWritten, err := handle.File.Write(data)
			if err != nil {
				return CreateStdlibErrorResult(err, "write")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(int64(bytesWritten))},
				object.NIL,
			}}
		},
	},

	// seek moves the file position to a new location.
	// Takes handle, offset, and whence (SEEK_START/CURRENT/END). Returns (position, Error) tuple.
	"io.seek": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Code: "E7001", Message: "io.seek() takes exactly 3 arguments (handle, offset, whence)"}
			}
			handle, ok := args[0].(*object.FileHandle)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.seek() requires a file handle as first argument"}
			}
			if handle.IsClosed {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7050", "io.seek(): file handle is closed"),
				}}
			}
			offset, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "io.seek() offset must be an integer"}
			}
			whence, ok := args[2].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "io.seek() whence must be an integer"}
			}

			newPos, err := handle.File.Seek(offset.Value.Int64(), int(whence.Value.Int64()))
			if err != nil {
				return CreateStdlibErrorResult(err, "seek")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(newPos)},
				object.NIL,
			}}
		},
	},

	// tell returns the current position in the file.
	// Takes handle. Returns (position, Error) tuple.
	"io.tell": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.tell() takes exactly 1 argument (handle)"}
			}
			handle, ok := args[0].(*object.FileHandle)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.tell() requires a file handle"}
			}
			if handle.IsClosed {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7050", "io.tell(): file handle is closed"),
				}}
			}

			// Seek with offset 0 from current position to get current position
			pos, err := handle.File.Seek(0, io.SeekCurrent)
			if err != nil {
				return CreateStdlibErrorResult(err, "tell")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(pos)},
				object.NIL,
			}}
		},
	},

	// flush flushes any buffered data to the file.
	// Takes handle. Returns (bool, Error) tuple.
	"io.flush": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.flush() takes exactly 1 argument (handle)"}
			}
			handle, ok := args[0].(*object.FileHandle)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.flush() requires a file handle"}
			}
			if handle.IsClosed {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					CreateStdlibError("E7050", "io.flush(): file handle is closed"),
				}}
			}

			err := handle.File.Sync()
			if err != nil {
				return CreateStdlibErrorResult(err, "flush")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// close closes a file handle.
	// Takes handle. Returns (bool, Error) tuple.
	"io.close": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.close() takes exactly 1 argument (handle)"}
			}
			handle, ok := args[0].(*object.FileHandle)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.close() requires a file handle"}
			}
			if handle.IsClosed {
				// Already closed - not an error, just return success
				return &object.ReturnValue{Values: []object.Object{
					object.TRUE,
					object.NIL,
				}}
			}

			err := handle.File.Close()
			handle.IsClosed = true

			if err != nil {
				return CreateStdlibErrorResult(err, "close")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// Filesystem Utilities
	// ============================================================================

	// glob finds files matching a glob pattern.
	// Takes pattern string (e.g., "*.txt"). Returns ([string], Error) tuple.
	"io.glob": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.glob() takes exactly 1 argument (pattern)"}
			}
			pattern, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.glob() requires a string pattern"}
			}

			matches, err := filepath.Glob(pattern.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E7043", fmt.Sprintf("io.glob() invalid pattern: %s", err.Error())),
				}}
			}

			// Convert matches to EZ array
			elements := make([]object.Object, len(matches))
			for i, m := range matches {
				elements[i] = &object.String{Value: m}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Array{Elements: elements, Mutable: true, ElementType: "string"},
				object.NIL,
			}}
		},
	},

	// walk recursively walks a directory tree, returning all file paths.
	// Takes directory path. Returns ([string], Error) tuple.
	"io.walk": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.walk() takes exactly 1 argument (directory)"}
			}
			dir, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.walk() requires a string directory path"}
			}

			// Validate path
			if err := validatePath(dir.Value, "io.walk()"); err != nil {
				return err
			}

			var files []string
			err := filepath.Walk(dir.Value, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// Only include files, not directories
				if !info.IsDir() {
					files = append(files, path)
				}
				return nil
			})

			if err != nil {
				return CreateStdlibErrorResult(err, "walk")
			}

			// Convert to EZ array
			elements := make([]object.Object, len(files))
			for i, f := range files {
				elements[i] = &object.String{Value: f}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Array{Elements: elements, Mutable: true, ElementType: "string"},
				object.NIL,
			}}
		},
	},

	// is_symlink checks if a path is a symbolic link.
	// Takes path string. Returns bool.
	"io.is_symlink": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.is_symlink() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.is_symlink() requires a string path"}
			}

			// Use Lstat to not follow symlinks
			info, err := os.Lstat(path.Value)
			if err != nil {
				// Non-existent paths return false, not an error
				return object.FALSE
			}

			if info.Mode()&os.ModeSymlink != 0 {
				return object.TRUE
			}
			return object.FALSE
		},
	},
}

// CreateStdlibErrorResult wraps a Go error into an EZ (nil, Error) return tuple
func CreateStdlibErrorResult(err error, operation string) *object.ReturnValue {
	var code string
	var message string

	// Determine error code based on error type
	if os.IsNotExist(err) {
		code = "E7016"
		message = fmt.Sprintf("file or directory not found: %s", err.Error())
	} else if os.IsPermission(err) {
		code = "E7017"
		message = fmt.Sprintf("permission denied: %s", err.Error())
	} else if os.IsExist(err) {
		code = "E7022"
		message = fmt.Sprintf("file or directory already exists: %s", err.Error())
	} else if strings.Contains(err.Error(), "directory not empty") {
		code = "E7023"
		message = fmt.Sprintf("directory not empty: %s", err.Error())
	} else {
		code = "E7099"
		message = fmt.Sprintf("io error during %s: %s", operation, err.Error())
	}

	return &object.ReturnValue{Values: []object.Object{
		object.NIL,
		CreateStdlibError(code, message),
	}}
}
