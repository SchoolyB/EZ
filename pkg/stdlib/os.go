package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"

	"github.com/marshallburns/ez/pkg/object"
)

// CommandLineArgs stores the command-line arguments passed to the EZ program
// This should be set by the main package before evaluation
var CommandLineArgs []string

// OSBuiltins contains the os module functions for system operations
var OSBuiltins = map[string]*object.Builtin{
	// ============================================================================
	// Environment Variables
	// ============================================================================

	// Gets an environment variable by name
	// Returns the value as a string, or nil if not set
	"os.get_env": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "os.get_env() takes exactly 1 argument (name)"}
			}
			name, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "os.get_env() requires a string argument"}
			}

			value, exists := os.LookupEnv(name.Value)
			if !exists {
				return object.NIL
			}
			return &object.String{Value: value}
		},
	},

	// Sets an environment variable (process-scoped only)
	// Returns true on success, (false, error) on failure
	"os.set_env": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "os.set_env() takes exactly 2 arguments (name, value)"}
			}
			name, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "os.set_env() requires a string name as first argument"}
			}
			value, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "os.set_env() requires a string value as second argument"}
			}

			err := os.Setenv(name.Value, value.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					createOSError("E7024", fmt.Sprintf("failed to set environment variable: %s", err.Error())),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Unsets an environment variable
	// Returns true on success, (false, error) on failure
	"os.unset_env": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "os.unset_env() takes exactly 1 argument (name)"}
			}
			name, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "os.unset_env() requires a string argument"}
			}

			err := os.Unsetenv(name.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					createOSError("E7024", fmt.Sprintf("failed to unset environment variable: %s", err.Error())),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Returns all environment variables as a map
	"os.env": {
		Fn: func(args ...object.Object) object.Object {
			envMap := object.NewMap()
			envMap.KeyType = "string"
			envMap.ValueType = "string"
			for _, entry := range os.Environ() {
				// Find the first = to split key and value
				for i := 0; i < len(entry); i++ {
					if entry[i] == '=' {
						key := entry[:i]
						value := entry[i+1:]
						envMap.Set(&object.String{Value: key}, &object.String{Value: value})
						break
					}
				}
			}
			envMap.Mutable = false
			return envMap
		},
	},

	// Returns command-line arguments as an array
	// First element is the program name/path
	"os.args": {
		Fn: func(args ...object.Object) object.Object {
			// Use CommandLineArgs if set, otherwise fall back to os.Args
			sourceArgs := CommandLineArgs
			if len(sourceArgs) == 0 {
				sourceArgs = os.Args
			}

			elements := make([]object.Object, len(sourceArgs))
			for i, arg := range sourceArgs {
				elements[i] = &object.String{Value: arg}
			}
			return &object.Array{Elements: elements, Mutable: false, ElementType: "string"}
		},
	},

	// ============================================================================
	// Process / System
	// ============================================================================

	// Exits the program with the given status code
	"os.exit": {
		Fn: func(args ...object.Object) object.Object {
			code := 0
			if len(args) > 0 {
				if codeObj, ok := args[0].(*object.Integer); ok {
					code = int(codeObj.Value.Int64())
				} else {
					return &object.Error{Code: "E7003", Message: "os.exit() requires an integer exit code"}
				}
			}
			os.Exit(code)
			select {} // Unreachable; satisfies compiler requirement
		},
	},

	// Returns the current working directory
	// Returns (path, error) tuple
	"os.cwd": {
		Fn: func(args ...object.Object) object.Object {
			cwd, err := os.Getwd()
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					createOSError("E7025", fmt.Sprintf("failed to get current directory: %s", err.Error())),
				}}
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: cwd},
				object.NIL,
			}}
		},
	},

	// Changes the current working directory
	// Returns (success, error) tuple
	"os.chdir": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "os.chdir() takes exactly 1 argument (path)"}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "os.chdir() requires a string path"}
			}

			err := os.Chdir(path.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					createOSError("E7026", fmt.Sprintf("failed to change directory: %s", err.Error())),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// Returns the hostname of the machine
	// Returns (hostname, error) tuple
	"os.hostname": {
		Fn: func(args ...object.Object) object.Object {
			hostname, err := os.Hostname()
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					createOSError("E7027", fmt.Sprintf("failed to get hostname: %s", err.Error())),
				}}
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: hostname},
				object.NIL,
			}}
		},
	},

	// Returns the current user's username
	// Returns (username, error) tuple
	"os.username": {
		Fn: func(args ...object.Object) object.Object {
			currentUser, err := user.Current()
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					createOSError("E7028", fmt.Sprintf("failed to get username: %s", err.Error())),
				}}
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: currentUser.Username},
				object.NIL,
			}}
		},
	},

	// Returns the current user's home directory
	// Returns (path, error) tuple
	"os.home_dir": {
		Fn: func(args ...object.Object) object.Object {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					createOSError("E7029", fmt.Sprintf("failed to get home directory: %s", err.Error())),
				}}
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: homeDir},
				object.NIL,
			}}
		},
	},

	// Returns the system's temporary directory
	"os.temp_dir": {
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: os.TempDir()}
		},
	},

	// Returns the process ID of the current process
	"os.pid": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(os.Getpid()))}
		},
	},

	// Returns the parent process ID
	"os.ppid": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(os.Getppid()))}
		},
	},

	// ============================================================================
	// Platform Detection
	// ============================================================================

	// OS Constants - use these for comparison with CURRENT_OS
	"os.MAC_OS": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(0)}
		},
		IsConstant: true,
	},
	"os.LINUX": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(1)}
		},
		IsConstant: true,
	},
	"os.WINDOWS": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(2)}
		},
		IsConstant: true,
	},

	// Returns the current OS as a constant (matches os.MAC_OS, os.LINUX, or os.WINDOWS)
	"os.CURRENT_OS": {
		Fn: func(args ...object.Object) object.Object {
			switch runtime.GOOS {
			case "darwin":
				return &object.Integer{Value: big.NewInt(0)} // MAC_OS
			case "linux":
				return &object.Integer{Value: big.NewInt(1)} // LINUX
			case "windows":
				return &object.Integer{Value: big.NewInt(2)} // WINDOWS
			default:
				return &object.Integer{Value: big.NewInt(-1)} // Unknown
			}
		},
		IsConstant: true,
	},

	// Returns the operating system name as a string
	// Possible values: "darwin", "linux", "windows", "freebsd", etc.
	"os.platform": {
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: runtime.GOOS}
		},
	},

	// Returns the CPU architecture
	// Possible values: "amd64", "arm64", "386", "arm", etc.
	"os.arch": {
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: runtime.GOARCH}
		},
	},

	// Returns true if running on Windows
	"os.is_windows": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS == "windows" {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// Returns true if running on Linux
	"os.is_linux": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS == "linux" {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// Returns true if running on macOS
	"os.is_macos": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS == "darwin" {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// Returns the number of CPUs available
	"os.num_cpu": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(runtime.NumCPU()))}
		},
	},

	// ============================================================================
	// Platform Constants
	// ============================================================================

	// Returns the line separator for the current platform
	// "\r\n" on Windows, "\n" on Unix-like systems
	"os.line_separator": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS == "windows" {
				return &object.String{Value: "\r\n"}
			}
			return &object.String{Value: "\n"}
		},
		IsConstant: true,
	},

	// Returns the null device path for the current platform
	// "NUL" on Windows, "/dev/null" on Unix-like systems
	"os.dev_null": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS == "windows" {
				return &object.String{Value: "NUL"}
			}
			return &object.String{Value: "/dev/null"}
		},
		IsConstant: true,
	},

	// ============================================================================
	// Command Execution
	// ============================================================================

	// Runs a shell command and returns (exit_code, error)
	// Commands run through the system shell (/bin/sh -c on Unix, cmd /c on Windows)
	// Note: User input should be sanitized before passing to this function
	"os.exec": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "os.exec() takes exactly 1 argument (command)"}
			}
			cmdStr, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "os.exec() requires a string command"}
			}

			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", "/c", cmdStr.Value)
			} else {
				cmd = exec.Command("/bin/sh", "-c", cmdStr.Value)
			}

			err := cmd.Run()

			var exitCode int64 = 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = int64(exitErr.ExitCode())
					// Command ran but returned non-zero exit code - return error for consistency with os.exec_output
					return &object.ReturnValue{Values: []object.Object{
						&object.Integer{Value: big.NewInt(exitCode)},
						createOSError("E7031", fmt.Sprintf("command exited with code %d", exitCode)),
					}}
				} else {
					// Command failed to start entirely
					return &object.ReturnValue{Values: []object.Object{
						&object.Integer{Value: big.NewInt(-1)},
						createOSError("E7030", fmt.Sprintf("command failed to execute: %s", err.Error())),
					}}
				}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(exitCode)},
				object.NIL,
			}}
		},
	},

	// Runs a shell command and returns (output, error)
	// Captures both stdout and stderr combined
	// Output has trailing whitespace trimmed
	// Note: User input should be sanitized before passing to this function
	"os.exec_output": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "os.exec_output() takes exactly 1 argument (command)"}
			}
			cmdStr, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "os.exec_output() requires a string command"}
			}

			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", "/c", cmdStr.Value)
			} else {
				cmd = exec.Command("/bin/sh", "-c", cmdStr.Value)
			}

			output, err := cmd.CombinedOutput()
			outputStr := strings.TrimRight(string(output), " \t\n\r")

			if err != nil {
				if _, ok := err.(*exec.ExitError); ok {
					// Command ran but returned non-zero exit code
					// Still return output with error
					return &object.ReturnValue{Values: []object.Object{
						&object.String{Value: outputStr},
						createOSError("E7031", fmt.Sprintf("command exited with error: %s", err.Error())),
					}}
				}
				// Command failed to start entirely
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					createOSError("E7030", fmt.Sprintf("command failed to execute: %s", err.Error())),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: outputStr},
				object.NIL,
			}}
		},
	},
}

// createOSError creates an Error struct for OS operations
func createOSError(code, message string) *object.Struct {
	return &object.Struct{
		TypeName: "Error",
		Fields: map[string]object.Object{
			"message": &object.String{Value: message},
			"code":    &object.String{Value: code},
		},
	}
}
