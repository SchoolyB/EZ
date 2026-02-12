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

	"github.com/marshallburns/ez/pkg/errors"
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

	// get_env retrieves an environment variable by name.
	// Takes variable name. Returns (string, Error) tuple.
	"os.get_env": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument (name)", errors.Ident("os.get_env()"))}
			}
			name, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("os.get_env()"), errors.TypeExpected("string"))}
			}

			value, exists := os.LookupEnv(name.Value)
			if !exists {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					CreateStdlibError("E7035", fmt.Sprintf("environment variable '%s' is not set", name.Value)),
				}}
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: value},
				object.NIL,
			}}
		},
	},

	// set_env sets an environment variable (process-scoped only).
	// Takes name and value. Returns (bool, Error) tuple.
	"os.set_env": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 2 arguments (name, value)", errors.Ident("os.set_env()"))}
			}
			name, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s name as first argument", errors.Ident("os.set_env()"), errors.TypeExpected("string"))}
			}
			value, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s value as second argument", errors.Ident("os.set_env()"), errors.TypeExpected("string"))}
			}

			err := os.Setenv(name.Value, value.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					CreateStdlibError("E7024", fmt.Sprintf("failed to set environment variable: %s", err.Error())),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// unset_env removes an environment variable.
	// Takes variable name. Returns (bool, Error) tuple.
	"os.unset_env": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument (name)", errors.Ident("os.unset_env()"))}
			}
			name, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("os.unset_env()"), errors.TypeExpected("string"))}
			}

			err := os.Unsetenv(name.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					CreateStdlibError("E7024", fmt.Sprintf("failed to unset environment variable: %s", err.Error())),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// env returns all environment variables as a map.
	// Takes no arguments. Returns map[string:string].
	"os.env": {
		Fn: func(args ...object.Object) object.Object {
			envMap := object.NewMap()
			envMap.KeyType = "string"
			envMap.ValueType = "string"
			for _, entry := range os.Environ() {
				if key, value, found := strings.Cut(entry, "="); found {
					envMap.Set(&object.String{Value: key}, &object.String{Value: value})
				}
			}
			envMap.Mutable = false
			return envMap
		},
	},

	// args returns command-line arguments as an array.
	// Takes no arguments. Returns [string] (first element is program name).
	"os.args": {
		Fn: func(args ...object.Object) object.Object {
			// Use CommandLineArgs if set, otherwise fall back to os.Args
			elements := make([]object.Object, len(CommandLineArgs))
			for i, arg := range CommandLineArgs {
				elements[i] = &object.String{Value: arg}
			}
			return &object.Array{Elements: elements, Mutable: false, ElementType: "string"}
		},
	},

	// ============================================================================
	// Process / System
	// ============================================================================

	// exit terminates the program with a status code.
	// Takes optional int exit code (default 0). Does not return.
	"os.exit": {
		Fn: func(args ...object.Object) object.Object {
			code := 0
			if len(args) > 0 {
				if codeObj, ok := args[0].(*object.Integer); ok {
					code = int(codeObj.Value.Int64())
				} else {
					return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires an %s exit code", errors.Ident("os.exit()"), errors.TypeExpected("integer"))}
				}
			}
			os.Exit(code)
			select {} // Unreachable; satisfies compiler requirement
		},
	},

	// cwd returns the current working directory.
	// Takes no arguments. Returns (string, Error) tuple.
	"os.cwd": {
		Fn: func(args ...object.Object) object.Object {
			cwd, err := os.Getwd()
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					CreateStdlibError("E7025", fmt.Sprintf("failed to get current directory: %s", err.Error())),
				}}
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: cwd},
				object.NIL,
			}}
		},
	},

	// chdir changes the current working directory.
	// Takes path string. Returns (bool, Error) tuple.
	"os.chdir": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument (path)", errors.Ident("os.chdir()"))}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s path", errors.Ident("os.chdir()"), errors.TypeExpected("string"))}
			}

			err := os.Chdir(path.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					CreateStdlibError("E7026", fmt.Sprintf("failed to change directory: %s", err.Error())),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},

	// hostname returns the system's hostname.
	// Takes no arguments. Returns (string, Error) tuple.
	"os.hostname": {
		Fn: func(args ...object.Object) object.Object {
			hostname, err := os.Hostname()
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					CreateStdlibError("E7027", fmt.Sprintf("failed to get hostname: %s", err.Error())),
				}}
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: hostname},
				object.NIL,
			}}
		},
	},

	// username returns the current user's name.
	// Takes no arguments. Returns (string, Error) tuple.
	"os.username": {
		Fn: func(args ...object.Object) object.Object {
			currentUser, err := user.Current()
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					CreateStdlibError("E7028", fmt.Sprintf("failed to get username: %s", err.Error())),
				}}
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: currentUser.Username},
				object.NIL,
			}}
		},
	},

	// home_dir returns the current user's home directory.
	// Takes no arguments. Returns (string, Error) tuple.
	"os.home_dir": {
		Fn: func(args ...object.Object) object.Object {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					CreateStdlibError("E7029", fmt.Sprintf("failed to get home directory: %s", err.Error())),
				}}
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: homeDir},
				object.NIL,
			}}
		},
	},

	// temp_dir returns the system's temporary directory.
	// Takes no arguments. Returns path string.
	"os.temp_dir": {
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: os.TempDir()}
		},
	},

	// pid returns the current process ID.
	// Takes no arguments. Returns int.
	"os.pid": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(os.Getpid()))}
		},
	},

	// ppid returns the parent process ID.
	// Takes no arguments. Returns int.
	"os.ppid": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(os.Getppid()))}
		},
	},

	// ============================================================================
	// Platform Detection
	// OS constants: MAC_OS=0, LINUX=1, WINDOWS=2.
	// ============================================================================

	// MAC_OS is the constant for macOS (value 0).
	"os.MAC_OS": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(0)}
		},
		IsConstant: true,
	},

	// LINUX is the constant for Linux (value 1).
	"os.LINUX": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(1)}
		},
		IsConstant: true,
	},

	// WINDOWS is the constant for Windows (value 2).
	"os.WINDOWS": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(2)}
		},
		IsConstant: true,
	},

	// CURRENT_OS returns the current OS as a constant.
	// Compare with MAC_OS, LINUX, or WINDOWS.
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

	// platform returns the OS name string (e.g., "darwin", "linux", "windows").
	// Takes no arguments. Returns string.
	"os.platform": {
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: runtime.GOOS}
		},
	},

	// arch returns the CPU architecture (e.g., "amd64", "arm64").
	// Takes no arguments. Returns string.
	"os.arch": {
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: runtime.GOARCH}
		},
	},

	// is_windows returns true if running on Windows.
	// Takes no arguments. Returns bool.
	"os.is_windows": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS == "windows" {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// is_linux returns true if running on Linux.
	// Takes no arguments. Returns bool.
	"os.is_linux": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS == "linux" {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// is_macos returns true if running on macOS.
	// Takes no arguments. Returns bool.
	"os.is_macos": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS == "darwin" {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// num_cpu returns the number of available CPUs.
	// Takes no arguments. Returns int.
	"os.num_cpu": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(int64(runtime.NumCPU()))}
		},
	},

	// ============================================================================
	// Platform Constants
	// ============================================================================

	// line_separator is the platform-specific line ending ("\r\n" or "\n").
	"os.line_separator": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS == "windows" {
				return &object.String{Value: "\r\n"}
			}
			return &object.String{Value: "\n"}
		},
		IsConstant: true,
	},

	// dev_null is the platform-specific null device ("NUL" or "/dev/null").
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

	// exec runs a shell command and returns the exit code.
	// Takes command string. Returns (int, Error) tuple.
	"os.exec": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument (command)", errors.Ident("os.exec()"))}
			}
			cmdStr, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s command", errors.Ident("os.exec()"), errors.TypeExpected("string"))}
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
						CreateStdlibError("E7031", fmt.Sprintf("command exited with code %d", exitCode)),
					}}
				} else {
					// Command failed to start entirely
					return &object.ReturnValue{Values: []object.Object{
						&object.Integer{Value: big.NewInt(-1)},
						CreateStdlibError("E7030", fmt.Sprintf("command failed to execute: %s", err.Error())),
					}}
				}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(exitCode)},
				object.NIL,
			}}
		},
	},

	// exec_output runs a shell command and returns its output.
	// Takes command string. Returns (string, Error) tuple.
	"os.exec_output": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument (command)", errors.Ident("os.exec_output()"))}
			}
			cmdStr, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s command", errors.Ident("os.exec_output()"), errors.TypeExpected("string"))}
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
						CreateStdlibError("E7031", fmt.Sprintf("command exited with error: %s", err.Error())),
					}}
				}
				// Command failed to start entirely
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					CreateStdlibError("E7030", fmt.Sprintf("command failed to execute: %s", err.Error())),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: outputStr},
				object.NIL,
			}}
		},
	},
}
