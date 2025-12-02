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

	// ============================================================================
	// File Writing
	// ============================================================================

	// Writes content to a file, creating it if it doesn't exist or overwriting if it does
	// Returns (success, error) tuple
	"io.write_file": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "io.write_file() takes exactly 2 arguments (path, content)"}
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

			err := os.WriteFile(path.Value, []byte(content.Value), 0644)
			if err != nil {
				return createIOErrorResult(err, "write")
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Appends content to a file, creating it if it doesn't exist
	// Returns (success, error) tuple
	"io.append_file": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "io.append_file() takes exactly 2 arguments (path, content)"}
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

			f, err := os.OpenFile(path.Value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return createIOErrorResult(err, "open for append")
			}

			_, err = f.WriteString(content.Value)
			if err != nil {
				f.Close()
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
	// Returns (success, error) tuple
	"io.copy": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "io.copy() takes exactly 2 arguments (source, destination)"}
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

			// Open source file
			src, err := os.Open(srcPath.Value)
			if err != nil {
				return createIOErrorResult(err, "open source")
			}
			defer src.Close() // Read-only, error on close is not critical

			// Create destination file
			dst, err := os.Create(dstPath.Value)
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

			// Preserve file mode
			err = os.Chmod(dstPath.Value, srcInfo.Mode())
			if err != nil {
				// Non-fatal on Windows, just continue
				_ = err
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
	// Returns (success, error) tuple
	"io.mkdir": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.mkdir() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.mkdir() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.mkdir()"); err != nil {
				return err
			}

			err := os.Mkdir(path.Value, 0755)
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
	// Returns (success, error) tuple
	"io.mkdir_all": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "io.mkdir_all() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "io.mkdir_all() requires a string path"}
			}

			// Validate path
			if err := validatePath(path.Value, "io.mkdir_all()"); err != nil {
				return err
			}

			err := os.MkdirAll(path.Value, 0755)
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
