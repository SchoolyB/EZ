package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"os"
	"os/user"
	"runtime"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// ============================================================================
// Environment Variables Tests
// ============================================================================

func TestOSGetEnv(t *testing.T) {
	fn := OSBuiltins["os.get_env"]

	// Set a test environment variable
	os.Setenv("EZ_TEST_VAR", "test_value")
	defer os.Unsetenv("EZ_TEST_VAR")

	// Test getting existing variable
	result := fn.Fn(&object.String{Value: "EZ_TEST_VAR"})
	strResult, ok := result.(*object.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}
	if strResult.Value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", strResult.Value)
	}

	// Test getting non-existent variable
	result = fn.Fn(&object.String{Value: "EZ_NONEXISTENT_VAR_12345"})
	if result != object.NIL {
		t.Errorf("Expected NIL for non-existent var, got %T", result)
	}

	// Test wrong argument count
	result = fn.Fn()
	errResult, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("Expected Error, got %T", result)
	}
	if errResult.Code != "E7001" {
		t.Errorf("Expected error code E7001, got %s", errResult.Code)
	}

	// Test wrong argument type
	result = fn.Fn(&object.Integer{Value: 123})
	errResult, ok = result.(*object.Error)
	if !ok {
		t.Fatalf("Expected Error, got %T", result)
	}
	if errResult.Code != "E7003" {
		t.Errorf("Expected error code E7003, got %s", errResult.Code)
	}
}

func TestOSSetEnv(t *testing.T) {
	fn := OSBuiltins["os.set_env"]

	// Test setting a new variable
	result := fn.Fn(&object.String{Value: "EZ_SET_TEST"}, &object.String{Value: "new_value"})
	defer os.Unsetenv("EZ_SET_TEST")

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("Expected ReturnValue, got %T", result)
	}
	if len(retVal.Values) != 2 {
		t.Fatalf("Expected 2 return values, got %d", len(retVal.Values))
	}
	if retVal.Values[0] != object.TRUE {
		t.Errorf("Expected TRUE, got %v", retVal.Values[0])
	}

	// Verify it was actually set
	if os.Getenv("EZ_SET_TEST") != "new_value" {
		t.Errorf("Environment variable was not set correctly")
	}

	// Test wrong argument count
	result = fn.Fn(&object.String{Value: "test"})
	errResult, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("Expected Error, got %T", result)
	}
	if errResult.Code != "E7001" {
		t.Errorf("Expected error code E7001, got %s", errResult.Code)
	}
}

func TestOSUnsetEnv(t *testing.T) {
	fn := OSBuiltins["os.unset_env"]

	// Set then unset a variable
	os.Setenv("EZ_UNSET_TEST", "to_be_removed")

	result := fn.Fn(&object.String{Value: "EZ_UNSET_TEST"})
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("Expected ReturnValue, got %T", result)
	}
	if retVal.Values[0] != object.TRUE {
		t.Errorf("Expected TRUE, got %v", retVal.Values[0])
	}

	// Verify it was unset
	if _, exists := os.LookupEnv("EZ_UNSET_TEST"); exists {
		t.Errorf("Environment variable should have been unset")
	}
}

func TestOSEnv(t *testing.T) {
	fn := OSBuiltins["os.env"]

	// Set a known test variable
	os.Setenv("EZ_ENV_TEST", "test123")
	defer os.Unsetenv("EZ_ENV_TEST")

	result := fn.Fn()
	mapResult, ok := result.(*object.Map)
	if !ok {
		t.Fatalf("Expected Map, got %T", result)
	}

	// Check that it contains our test variable
	key := &object.String{Value: "EZ_ENV_TEST"}
	val, _ := mapResult.Get(key)
	strVal, ok := val.(*object.String)
	if !ok {
		t.Fatalf("Expected String value for EZ_ENV_TEST, got %T", val)
	}
	if strVal.Value != "test123" {
		t.Errorf("Expected 'test123', got '%s'", strVal.Value)
	}

	// Check that the map is immutable
	if mapResult.Mutable {
		t.Errorf("Environment map should be immutable")
	}
}

func TestOSArgs(t *testing.T) {
	fn := OSBuiltins["os.args"]

	// Set CommandLineArgs for testing
	originalArgs := CommandLineArgs
	CommandLineArgs = []string{"program", "arg1", "arg2"}
	defer func() { CommandLineArgs = originalArgs }()

	result := fn.Fn()
	arrResult, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("Expected Array, got %T", result)
	}

	if len(arrResult.Elements) != 3 {
		t.Fatalf("Expected 3 elements, got %d", len(arrResult.Elements))
	}

	expectedArgs := []string{"program", "arg1", "arg2"}
	for i, expected := range expectedArgs {
		strElem, ok := arrResult.Elements[i].(*object.String)
		if !ok {
			t.Fatalf("Expected String at index %d, got %T", i, arrResult.Elements[i])
		}
		if strElem.Value != expected {
			t.Errorf("At index %d: expected '%s', got '%s'", i, expected, strElem.Value)
		}
	}

	// Check that the array is immutable
	if arrResult.Mutable {
		t.Errorf("Args array should be immutable")
	}
}

// ============================================================================
// Process / System Tests
// ============================================================================

func TestOSCwd(t *testing.T) {
	fn := OSBuiltins["os.cwd"]

	result := fn.Fn()
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("Expected ReturnValue, got %T", result)
	}

	if len(retVal.Values) != 2 {
		t.Fatalf("Expected 2 return values, got %d", len(retVal.Values))
	}

	strResult, ok := retVal.Values[0].(*object.String)
	if !ok {
		t.Fatalf("Expected String, got %T", retVal.Values[0])
	}

	// Get actual cwd
	expectedCwd, _ := os.Getwd()
	if strResult.Value != expectedCwd {
		t.Errorf("Expected '%s', got '%s'", expectedCwd, strResult.Value)
	}

	// Error should be nil
	if retVal.Values[1] != object.NIL {
		t.Errorf("Expected NIL error, got %v", retVal.Values[1])
	}
}

func TestOSChdir(t *testing.T) {
	fn := OSBuiltins["os.chdir"]

	// Save current directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Change to temp dir
	tempDir := os.TempDir()
	result := fn.Fn(&object.String{Value: tempDir})

	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("Expected ReturnValue, got %T", result)
	}

	if retVal.Values[0] != object.TRUE {
		t.Errorf("Expected TRUE, got %v", retVal.Values[0])
	}

	// Test changing to non-existent directory
	result = fn.Fn(&object.String{Value: "/nonexistent/path/12345"})
	retVal, ok = result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("Expected ReturnValue, got %T", result)
	}

	if retVal.Values[0] != object.FALSE {
		t.Errorf("Expected FALSE for invalid path, got %v", retVal.Values[0])
	}

	// Test wrong argument count
	result = fn.Fn()
	errResult, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("Expected Error, got %T", result)
	}
	if errResult.Code != "E7001" {
		t.Errorf("Expected error code E7001, got %s", errResult.Code)
	}
}

func TestOSHostname(t *testing.T) {
	fn := OSBuiltins["os.hostname"]

	result := fn.Fn()
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("Expected ReturnValue, got %T", result)
	}

	strResult, ok := retVal.Values[0].(*object.String)
	if !ok {
		t.Fatalf("Expected String, got %T", retVal.Values[0])
	}

	expectedHostname, _ := os.Hostname()
	if strResult.Value != expectedHostname {
		t.Errorf("Expected '%s', got '%s'", expectedHostname, strResult.Value)
	}
}

func TestOSUsername(t *testing.T) {
	fn := OSBuiltins["os.username"]

	result := fn.Fn()
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("Expected ReturnValue, got %T", result)
	}

	strResult, ok := retVal.Values[0].(*object.String)
	if !ok {
		t.Fatalf("Expected String, got %T", retVal.Values[0])
	}

	currentUser, _ := user.Current()
	if strResult.Value != currentUser.Username {
		t.Errorf("Expected '%s', got '%s'", currentUser.Username, strResult.Value)
	}
}

func TestOSHomeDir(t *testing.T) {
	fn := OSBuiltins["os.home_dir"]

	result := fn.Fn()
	retVal, ok := result.(*object.ReturnValue)
	if !ok {
		t.Fatalf("Expected ReturnValue, got %T", result)
	}

	strResult, ok := retVal.Values[0].(*object.String)
	if !ok {
		t.Fatalf("Expected String, got %T", retVal.Values[0])
	}

	expectedHome, _ := os.UserHomeDir()
	if strResult.Value != expectedHome {
		t.Errorf("Expected '%s', got '%s'", expectedHome, strResult.Value)
	}
}

func TestOSTempDir(t *testing.T) {
	fn := OSBuiltins["os.temp_dir"]

	result := fn.Fn()
	strResult, ok := result.(*object.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	if strResult.Value != os.TempDir() {
		t.Errorf("Expected '%s', got '%s'", os.TempDir(), strResult.Value)
	}
}

func TestOSPid(t *testing.T) {
	fn := OSBuiltins["os.pid"]

	result := fn.Fn()
	intResult, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("Expected Integer, got %T", result)
	}

	if intResult.Value != int64(os.Getpid()) {
		t.Errorf("Expected %d, got %d", os.Getpid(), intResult.Value)
	}
}

func TestOSPpid(t *testing.T) {
	fn := OSBuiltins["os.ppid"]

	result := fn.Fn()
	intResult, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("Expected Integer, got %T", result)
	}

	if intResult.Value != int64(os.Getppid()) {
		t.Errorf("Expected %d, got %d", os.Getppid(), intResult.Value)
	}
}

// ============================================================================
// Platform Detection Tests
// ============================================================================

func TestOSConstants(t *testing.T) {
	macOS := OSBuiltins["os.MAC_OS"]
	linux := OSBuiltins["os.LINUX"]
	windows := OSBuiltins["os.WINDOWS"]
	currentOS := OSBuiltins["os.CURRENT_OS"]

	// Check constants have expected values
	macResult := macOS.Fn().(*object.Integer)
	linuxResult := linux.Fn().(*object.Integer)
	windowsResult := windows.Fn().(*object.Integer)

	if macResult.Value != 0 {
		t.Errorf("Expected MAC_OS = 0, got %d", macResult.Value)
	}
	if linuxResult.Value != 1 {
		t.Errorf("Expected LINUX = 1, got %d", linuxResult.Value)
	}
	if windowsResult.Value != 2 {
		t.Errorf("Expected WINDOWS = 2, got %d", windowsResult.Value)
	}

	// Check CURRENT_OS matches expected constant for this platform
	currentResult := currentOS.Fn().(*object.Integer)
	switch runtime.GOOS {
	case "darwin":
		if currentResult.Value != 0 {
			t.Errorf("Expected CURRENT_OS = MAC_OS (0) on darwin, got %d", currentResult.Value)
		}
	case "linux":
		if currentResult.Value != 1 {
			t.Errorf("Expected CURRENT_OS = LINUX (1) on linux, got %d", currentResult.Value)
		}
	case "windows":
		if currentResult.Value != 2 {
			t.Errorf("Expected CURRENT_OS = WINDOWS (2) on windows, got %d", currentResult.Value)
		}
	}
}

func TestOSPlatform(t *testing.T) {
	fn := OSBuiltins["os.platform"]

	result := fn.Fn()
	strResult, ok := result.(*object.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	if strResult.Value != runtime.GOOS {
		t.Errorf("Expected '%s', got '%s'", runtime.GOOS, strResult.Value)
	}
}

func TestOSArch(t *testing.T) {
	fn := OSBuiltins["os.arch"]

	result := fn.Fn()
	strResult, ok := result.(*object.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	if strResult.Value != runtime.GOARCH {
		t.Errorf("Expected '%s', got '%s'", runtime.GOARCH, strResult.Value)
	}
}

func TestOSIsWindows(t *testing.T) {
	fn := OSBuiltins["os.is_windows"]

	result := fn.Fn()
	boolResult, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("Expected Boolean, got %T", result)
	}

	expected := runtime.GOOS == "windows"
	if boolResult.Value != expected {
		t.Errorf("Expected %v, got %v", expected, boolResult.Value)
	}
}

func TestOSIsLinux(t *testing.T) {
	fn := OSBuiltins["os.is_linux"]

	result := fn.Fn()
	boolResult, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("Expected Boolean, got %T", result)
	}

	expected := runtime.GOOS == "linux"
	if boolResult.Value != expected {
		t.Errorf("Expected %v, got %v", expected, boolResult.Value)
	}
}

func TestOSIsMacos(t *testing.T) {
	fn := OSBuiltins["os.is_macos"]

	result := fn.Fn()
	boolResult, ok := result.(*object.Boolean)
	if !ok {
		t.Fatalf("Expected Boolean, got %T", result)
	}

	expected := runtime.GOOS == "darwin"
	if boolResult.Value != expected {
		t.Errorf("Expected %v, got %v", expected, boolResult.Value)
	}
}

func TestOSNumCpu(t *testing.T) {
	fn := OSBuiltins["os.num_cpu"]

	result := fn.Fn()
	intResult, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("Expected Integer, got %T", result)
	}

	if intResult.Value != int64(runtime.NumCPU()) {
		t.Errorf("Expected %d, got %d", runtime.NumCPU(), intResult.Value)
	}
}

// ============================================================================
// Platform Constants Tests
// ============================================================================

func TestOSLineSeparator(t *testing.T) {
	fn := OSBuiltins["os.line_separator"]

	result := fn.Fn()
	strResult, ok := result.(*object.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	var expected string
	if runtime.GOOS == "windows" {
		expected = "\r\n"
	} else {
		expected = "\n"
	}

	if strResult.Value != expected {
		t.Errorf("Expected '%q', got '%q'", expected, strResult.Value)
	}
}

func TestOSDevNull(t *testing.T) {
	fn := OSBuiltins["os.dev_null"]

	result := fn.Fn()
	strResult, ok := result.(*object.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	var expected string
	if runtime.GOOS == "windows" {
		expected = "NUL"
	} else {
		expected = "/dev/null"
	}

	if strResult.Value != expected {
		t.Errorf("Expected '%s', got '%s'", expected, strResult.Value)
	}
}

// ============================================================================
// Exit function test (cannot test os.Exit directly, just verify function exists)
// ============================================================================

func TestOSExitExists(t *testing.T) {
	fn := OSBuiltins["os.exit"]
	if fn == nil {
		t.Error("os.exit function should exist")
	}

	// Test wrong argument type (we can test error handling but not actual exit)
	result := fn.Fn(&object.String{Value: "not a number"})
	errResult, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("Expected Error for wrong type, got %T", result)
	}
	if errResult.Code != "E7003" {
		t.Errorf("Expected error code E7003, got %s", errResult.Code)
	}
}

// ============================================================================
// Helper function tests
// ============================================================================

func TestCreateOSError(t *testing.T) {
	err := createOSError("E9999", "test error message")

	if err.TypeName != "Error" {
		t.Errorf("Expected TypeName 'Error', got '%s'", err.TypeName)
	}

	msgField, ok := err.Fields["message"].(*object.String)
	if !ok {
		t.Fatalf("Expected message field to be String, got %T", err.Fields["message"])
	}
	if msgField.Value != "test error message" {
		t.Errorf("Expected message 'test error message', got '%s'", msgField.Value)
	}

	codeField, ok := err.Fields["code"].(*object.String)
	if !ok {
		t.Fatalf("Expected code field to be String, got %T", err.Fields["code"])
	}
	if codeField.Value != "E9999" {
		t.Errorf("Expected code 'E9999', got '%s'", codeField.Value)
	}
}

// ============================================================================
// Command Execution Tests
// ============================================================================

func TestOSExec(t *testing.T) {
	execFn := OSBuiltins["os.exec"].Fn

	t.Run("successful command", func(t *testing.T) {
		result := execFn(&object.String{Value: "echo hello"})
		rv, ok := result.(*object.ReturnValue)
		if !ok {
			t.Fatalf("expected ReturnValue, got %T", result)
		}
		if len(rv.Values) != 2 {
			t.Fatalf("expected 2 return values, got %d", len(rv.Values))
		}

		exitCode, ok := rv.Values[0].(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer exit code, got %T", rv.Values[0])
		}
		if exitCode.Value != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode.Value)
		}

		if rv.Values[1] != object.NIL {
			t.Errorf("expected nil error, got %T", rv.Values[1])
		}
	})

	t.Run("command with non-zero exit", func(t *testing.T) {
		result := execFn(&object.String{Value: "exit 42"})
		rv := result.(*object.ReturnValue)

		exitCode := rv.Values[0].(*object.Integer)
		if exitCode.Value != 42 {
			t.Errorf("expected exit code 42, got %d", exitCode.Value)
		}

		// Error should still be nil for non-zero exit
		if rv.Values[1] != object.NIL {
			t.Errorf("expected nil error for non-zero exit, got %T", rv.Values[1])
		}
	})

	t.Run("wrong argument type", func(t *testing.T) {
		result := execFn(&object.Integer{Value: 123})
		if _, ok := result.(*object.Error); !ok {
			t.Errorf("expected Error for wrong type, got %T", result)
		}
	})
}

func TestOSExecOutput(t *testing.T) {
	execOutputFn := OSBuiltins["os.exec_output"].Fn

	t.Run("capture output", func(t *testing.T) {
		result := execOutputFn(&object.String{Value: "echo hello"})
		rv, ok := result.(*object.ReturnValue)
		if !ok {
			t.Fatalf("expected ReturnValue, got %T", result)
		}

		output, ok := rv.Values[0].(*object.String)
		if !ok {
			t.Fatalf("expected String output, got %T", rv.Values[0])
		}
		if output.Value != "hello" {
			t.Errorf("expected 'hello', got '%s'", output.Value)
		}

		if rv.Values[1] != object.NIL {
			t.Errorf("expected nil error, got %T", rv.Values[1])
		}
	})

	t.Run("output is trimmed", func(t *testing.T) {
		result := execOutputFn(&object.String{Value: "printf 'test\\n\\n'"})
		rv := result.(*object.ReturnValue)
		output := rv.Values[0].(*object.String)

		if output.Value != "test" {
			t.Errorf("expected 'test' (trimmed), got '%s'", output.Value)
		}
	})

	t.Run("command with non-zero exit returns output and error", func(t *testing.T) {
		// This command outputs something then exits with error
		result := execOutputFn(&object.String{Value: "echo 'error output' && exit 1"})
		rv := result.(*object.ReturnValue)

		output := rv.Values[0].(*object.String)
		if output.Value != "error output" {
			t.Errorf("expected 'error output', got '%s'", output.Value)
		}

		// Should have error for non-zero exit
		if rv.Values[1] == object.NIL {
			t.Error("expected error for non-zero exit")
		}
	})

	t.Run("invalid command returns error", func(t *testing.T) {
		result := execOutputFn(&object.String{Value: "nonexistent_command_xyz123"})
		rv := result.(*object.ReturnValue)

		// Should have error
		if rv.Values[1] == object.NIL {
			t.Error("expected error for invalid command")
		}
	})
}
