package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"io"
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
			createIOError("E7040", fmt.Sprintf("%s: path cannot be empty", funcName)),
		}}
	}
	if strings.ContainsRune(path, '\x00') {
		return &object.ReturnValue{Values: []object.Object{
			object.NIL,
			createIOError("E7041", fmt.Sprintf("%s: path contains null byte", funcName)),
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
		tmpFile.Close()
		return fmt.Errorf("set permissions: %w", err)
	}

	// Write data
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write data: %w", err)
	}

	// Sync to ensure data is on disk before rename
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
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

	// Reads the entire contents of a file as a string
	// Returns (content, error) tuple - error is nil on success
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
					createIOError("E7042", "io.read_file(): cannot read directory as file"),
				}}
			}

			content, err := os.ReadFile(path.Value)
			if err != nil {
				return createIOErrorResult(err, "read")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: string(content)},
				object.NIL,
			}}
		},
	},

	// Reads the entire contents of a file as a byte array
	// Returns (bytes, error) tuple - error is nil on success
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
					createIOError("E7042", "io.read_bytes(): cannot read directory as file"),
				}}
			}

			content, err := os.ReadFile(path.Value)
			if err != nil {
				return createIOErrorResult(err, "read")
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

	// Writes content to a file atomically, creating it if it doesn't exist or overwriting if it does
	// Uses temp file + rename to ensure file is never left in a partial state
	// Takes 2-3 arguments: path, content, and optional permissions (int, default 0644)
	// Returns (success, error) tuple
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
				perms = os.FileMode(permVal.Value)
			}

			err := atomicWriteFile(path.Value, []byte(content.Value), perms)
			if err != nil {
				return createIOErrorResult(err, "write")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Writes bytes to a file atomically, creating it if it doesn't exist or overwriting if it does
	// Uses temp file + rename to ensure file is never left in a partial state
	// Takes 2-3 arguments: path, data (byte array), and optional permissions (int)
	// Returns (success, error) tuple
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
				perms = os.FileMode(permVal.Value)
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
				return createIOErrorResult(err, "write bytes")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Appends content to a file, creating it if it doesn't exist
	// Takes 2-3 arguments: path, content, and optional permissions (int, default 0644)
	// Returns (success, error) tuple
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
				perms = os.FileMode(permVal.Value)
			}

			f, err := os.OpenFile(path.Value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perms)
			if err != nil {
				return createIOErrorResult(err, "open for append")
			}

			_, err = f.WriteString(content.Value)
			if err != nil {
				_ = f.Close()
				return createIOErrorResult(err, "append")
			}

			if err := f.Close(); err != nil {
				return createIOErrorResult(err, "close after append")
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

	// Reads a file and returns its content as an array of lines
	// Returns (lines, error) tuple where lines is an array of strings
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
					createIOError("E7042", "io.read_lines(): cannot read directory as file"),
				}}
			}

			content, err := os.ReadFile(path.Value)
			if err != nil {
				return createIOErrorResult(err, "read")
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

	// Appends a line to a file, adding a newline at the end
	// Takes 2-3 arguments: path, line, and optional permissions (int, default 0644)
	// Returns (success, error) tuple
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
				perms = os.FileMode(permVal.Value)
			}

			f, err := os.OpenFile(path.Value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perms)
			if err != nil {
				return createIOErrorResult(err, "open for append")
			}

			// Write line with newline
			_, err = f.WriteString(line.Value + "\n")
			if err != nil {
				_ = f.Close()
				return createIOErrorResult(err, "append line")
			}

			if err := f.Close(); err != nil {
				return createIOErrorResult(err, "close after append")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Expands ~ to home directory and cleans the path
	// Returns the expanded path string
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
						createIOError("E7029", "io.expand_path(): failed to get home directory"),
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

	// Checks if a path exists (file or directory)
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

	// Checks if a path is a regular file
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

	// Checks if a path is a directory
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

	// Removes a file
	// Returns (success, error) tuple
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
				return createIOErrorResult(err, "stat")
			}
			if info.IsDir() {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					createIOError("E7018", "io.remove() cannot remove directories, use io.remove_dir() instead"),
				}}
			}

			err = os.Remove(path.Value)
			if err != nil {
				return createIOErrorResult(err, "remove")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Removes an empty directory
	// Returns (success, error) tuple
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
				return createIOErrorResult(err, "stat")
			}
			if !info.IsDir() {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					createIOError("E7019", "io.remove_dir() can only remove directories, use io.remove() for files"),
				}}
			}

			err = os.Remove(path.Value)
			if err != nil {
				return createIOErrorResult(err, "remove directory")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Removes a file or directory recursively (DANGEROUS - use with caution)
	// Returns (success, error) tuple
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
				return createIOErrorResult(err, "resolve path")
			}

			// Check for dangerous paths
			if absPath == "/" || absPath == filepath.Clean(os.Getenv("HOME")) {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					createIOError("E7020", "io.remove_all() cannot remove root or home directory for safety"),
				}}
			}

			err = os.RemoveAll(path.Value)
			if err != nil {
				return createIOErrorResult(err, "remove all")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Renames or moves a file or directory
	// Returns (success, error) tuple
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
				return createIOErrorResult(err, "rename")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Copies a file from source to destination
	// Takes 2-3 arguments: source, destination, and optional permissions (int)
	// If permissions not specified, preserves source file's permissions
	// Returns (success, error) tuple
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
				return createIOErrorResult(err, "stat source")
			}
			if srcInfo.IsDir() {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					createIOError("E7021", "io.copy() cannot copy directories, only files"),
				}}
			}

			// Determine permissions: explicit or preserve from source
			var perms os.FileMode
			if len(args) == 3 {
				permVal, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "io.copy() permissions must be an integer"}
				}
				perms = os.FileMode(permVal.Value)
			} else {
				perms = srcInfo.Mode()
			}

			// Open source file
			src, err := os.Open(srcPath.Value)
			if err != nil {
				return createIOErrorResult(err, "open source")
			}
			defer src.Close() // Read-only, error on close is not critical

			// Create destination file with proper permissions
			dst, err := os.OpenFile(dstPath.Value, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perms)
			if err != nil {
				return createIOErrorResult(err, "create destination")
			}

			// Copy contents
			_, err = io.Copy(dst, src)
			if err != nil {
				dst.Close()
				return createIOErrorResult(err, "copy contents")
			}

			// Close destination and check for write errors
			if err := dst.Close(); err != nil {
				return createIOErrorResult(err, "close destination")
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

	// Creates a directory (parent must exist)
	// Takes 1-2 arguments: path, and optional permissions (int, default 0755)
	// Returns (success, error) tuple
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
				perms = os.FileMode(permVal.Value)
			}

			err := os.Mkdir(path.Value, perms)
			if err != nil {
				return createIOErrorResult(err, "mkdir")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Creates a directory and all parent directories as needed
	// Takes 1-2 arguments: path, and optional permissions (int, default 0755)
	// Returns (success, error) tuple
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
				perms = os.FileMode(permVal.Value)
			}

			err := os.MkdirAll(path.Value, perms)
			if err != nil {
				return createIOErrorResult(err, "mkdir_all")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Lists the contents of a directory
	// Returns (entries, error) tuple where entries is an array of filenames
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
				return createIOErrorResult(err, "read directory")
			}

			elements := make([]object.Object, len(entries))
			for i, entry := range entries {
				elements[i] = &object.String{Value: entry.Name()}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Array{Elements: elements, Mutable: false},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// File Metadata
	// ============================================================================

	// Returns the size of a file in bytes
	// Returns (size, error) tuple
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
				return createIOErrorResult(err, "stat")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: info.Size()},
				object.NIL,
			}}
		},
	},

	// Returns the modification time of a file as a Unix timestamp
	// Returns (timestamp, error) tuple
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
				return createIOErrorResult(err, "stat")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: info.ModTime().Unix()},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// Path Utilities
	// ============================================================================

	// Joins path components using the OS-specific separator
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

	// Returns the last element of a path (filename or directory name)
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

	// Returns the directory portion of a path
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

	// Returns the file extension (including the dot)
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

	// Returns the absolute path
	// Returns (abs_path, error) tuple
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
				return createIOErrorResult(err, "resolve absolute path")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: absPath},
				object.NIL,
			}}
		},
	},

	// Cleans a path (removes redundant separators, . and ..)
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

	// Returns the OS-specific path separator
	"io.path_separator": {
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: string(filepath.Separator)}
		},
	},

	// ============================================================================
	// File Handle Constants
	// ============================================================================

	// File open mode constants
	"io.READ_ONLY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: int64(os.O_RDONLY)}
		},
	},
	"io.WRITE_ONLY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: int64(os.O_WRONLY)}
		},
	},
	"io.READ_WRITE": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: int64(os.O_RDWR)}
		},
	},
	"io.APPEND": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: int64(os.O_APPEND)}
		},
	},
	"io.CREATE": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: int64(os.O_CREATE)}
		},
	},
	"io.TRUNCATE": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: int64(os.O_TRUNC)}
		},
	},
	"io.EXCLUSIVE": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: int64(os.O_EXCL)}
		},
	},

	// Seek whence constants
	"io.SEEK_START": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: int64(io.SeekStart)}
		},
	},
	"io.SEEK_CURRENT": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: int64(io.SeekCurrent)}
		},
	},
	"io.SEEK_END": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: int64(io.SeekEnd)}
		},
	},

	// ============================================================================
	// File Handle Operations
	// ============================================================================

	// Opens a file and returns a file handle
	// Takes path and mode flags (combine with | operator)
	// Returns (handle, error) tuple
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
				mode = int(modeVal.Value)
			}

			// Default permissions
			perms := os.FileMode(0644)
			if len(args) == 3 {
				permVal, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "io.open() permissions must be an integer"}
				}
				perms = os.FileMode(permVal.Value)
			}

			file, err := os.OpenFile(path.Value, mode, perms)
			if err != nil {
				return createIOErrorResult(err, "open")
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

	// Reads n bytes from a file handle
	// Returns (bytes, error) tuple
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
					createIOError("E7050", "io.read(): file handle is closed"),
				}}
			}
			n, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "io.read() requires an integer as second argument"}
			}
			if n.Value < 0 {
				return &object.Error{Code: "E7011", Message: "io.read() byte count cannot be negative"}
			}

			buf := make([]byte, n.Value)
			bytesRead, err := handle.File.Read(buf)

			// Handle EOF - not an error, just return what we got
			if err != nil && err != io.EOF {
				return createIOErrorResult(err, "read")
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

	// Reads all remaining bytes from a file handle
	// Returns (bytes, error) tuple
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
					createIOError("E7050", "io.read_all(): file handle is closed"),
				}}
			}

			content, err := io.ReadAll(handle.File)
			if err != nil {
				return createIOErrorResult(err, "read all")
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

	// Reads n bytes from a file handle as a string
	// Returns (string, error) tuple
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
					createIOError("E7050", "io.read_string(): file handle is closed"),
				}}
			}
			n, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "io.read_string() requires an integer as second argument"}
			}
			if n.Value < 0 {
				return &object.Error{Code: "E7011", Message: "io.read_string() byte count cannot be negative"}
			}

			buf := make([]byte, n.Value)
			bytesRead, err := handle.File.Read(buf)

			// Handle EOF - not an error, just return what we got
			if err != nil && err != io.EOF {
				return createIOErrorResult(err, "read string")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: string(buf[:bytesRead])},
				object.NIL,
			}}
		},
	},

	// Writes data to a file handle
	// Data can be a string or byte array
	// Returns (bytes_written, error) tuple
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
					createIOError("E7050", "io.write(): file handle is closed"),
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
				return createIOErrorResult(err, "write")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: int64(bytesWritten)},
				object.NIL,
			}}
		},
	},

	// Seeks to a position in the file
	// Returns (new_position, error) tuple
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
					createIOError("E7050", "io.seek(): file handle is closed"),
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

			newPos, err := handle.File.Seek(offset.Value, int(whence.Value))
			if err != nil {
				return createIOErrorResult(err, "seek")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: newPos},
				object.NIL,
			}}
		},
	},

	// Returns the current position in the file
	// Returns (position, error) tuple
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
					createIOError("E7050", "io.tell(): file handle is closed"),
				}}
			}

			// Seek with offset 0 from current position to get current position
			pos, err := handle.File.Seek(0, io.SeekCurrent)
			if err != nil {
				return createIOErrorResult(err, "tell")
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: pos},
				object.NIL,
			}}
		},
	},

	// Flushes any buffered data to the file
	// Returns (success, error) tuple
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
					createIOError("E7050", "io.flush(): file handle is closed"),
				}}
			}

			err := handle.File.Sync()
			if err != nil {
				return createIOErrorResult(err, "flush")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Closes a file handle
	// Returns (success, error) tuple
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
				return createIOErrorResult(err, "close")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},
}

// createIOError creates an Error struct for IO operations
func createIOError(code, message string) *object.Struct {
	return &object.Struct{
		TypeName: "Error",
		Fields: map[string]object.Object{
			"message": &object.String{Value: message},
			"code":    &object.String{Value: code},
		},
	}
}

// createIOErrorResult wraps a Go error into an EZ (nil, Error) return tuple
func createIOErrorResult(err error, operation string) *object.ReturnValue {
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
		createIOError(code, message),
	}}
}
