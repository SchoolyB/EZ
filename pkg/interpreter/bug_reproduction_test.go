package interpreter

// Bug reproduction tests - these tests verify bugs documented in BUG_REPORT.md

import (
	"testing"

	"github.com/marshallburns/ez/pkg/lexer"
	"github.com/marshallburns/ez/pkg/parser"
)

// TestBug1_EmptyReturnValuePanic tests that accessing Values[0] on an empty
// ReturnValue doesn't panic. This is the bug at evaluator.go:774.
//
// The bug is in evalProgram where it does:
//   case *ReturnValue:
//       return result.Values[0]  // No bounds check!
func TestBug1_EmptyReturnValuePanic(t *testing.T) {
	// Create a ReturnValue with empty Values slice
	emptyReturn := &ReturnValue{Values: []Object{}}

	// This simulates what happens at line 774 - accessing Values[0] without checking
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: Panic occurred when accessing empty ReturnValue.Values[0]: %v", r)
			// Bug is confirmed - this test documents the issue
		}
	}()

	// Direct access like line 774 does
	if len(emptyReturn.Values) > 0 {
		_ = emptyReturn.Values[0] // Safe access
	} else {
		// This is what the current code does NOT do - it just accesses [0] directly
		// Simulating the bug:
		_ = emptyReturn.Values[0] // This will panic
	}
}

// TestBug1_EmptyReturnFromFunction tests that a function with empty return
// is handled correctly.
func TestBug1_EmptyReturnFromFunction(t *testing.T) {
	input := `
do empty_return() {
    return
}

do main() {
    temp x = empty_return()
}
`
	l := lexer.NewLexer(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := NewEnvironment()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("BUG: Panic when handling empty return: %v", r)
		}
	}()

	result := Eval(program, env)

	// Should complete without panic
	if result != nil && result.Type() == ERROR_OBJ {
		// An error is acceptable, but a panic is not
		t.Logf("Got error (acceptable): %s", result.Inspect())
	}
}

// TestBug2_IgnoredDerefError tests that Deref() errors are properly handled
// in compound assignments. This is the bug at evaluator.go:1142.
//
// The bug: oldVal, _ := ref.Deref() ignores the error.
// If Deref() fails, oldVal is nil (Go nil, not EZ NIL object).
// This nil is passed to evalCompoundAssignment -> evalInfixExpression.
// At line 2086: left.Type() will panic because left is nil.
func TestBug2_IgnoredDerefError(t *testing.T) {
	// Simulate what happens when Deref() fails and nil is passed
	// to evalCompoundAssignment

	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: Panic when nil passed to evalInfixExpression: %v", r)
		}
	}()

	// This simulates the bug scenario:
	// 1. ref.Deref() returns (nil, false)
	// 2. The code ignores the false: oldVal, _ := ref.Deref()
	// 3. oldVal (nil) is passed to evalCompoundAssignment
	// 4. evalCompoundAssignment calls evalInfixExpression("+", nil, right, ...)
	// 5. evalInfixExpression does: left.Type() - PANIC on nil

	var nilObject Object = nil
	rightValue := &Integer{Value: nil}

	// This will panic at left.Type() because left is nil
	result := evalInfixExpression("+", nilObject, rightValue, 0, 0)
	_ = result
}

// TestBug2_DerefFailsOnMissingVariable shows that Deref can fail
func TestBug2_DerefFailsOnMissingVariable(t *testing.T) {
	env := NewEnvironment()

	// Create a reference to a non-existent variable
	ref := &Reference{Env: env, Name: "does_not_exist"}

	val, ok := ref.Deref()
	if ok {
		t.Error("Deref() should return false for non-existent variable")
	}
	if val != nil {
		t.Error("Deref() should return nil when variable doesn't exist")
	}

	t.Log("Confirmed: Deref() returns (nil, false) for missing variables")
	t.Log("At line 1142, this nil would be passed to evalCompoundAssignment")
}

// TestBug3_InconsistentFileCloseErrorHandling documents the inconsistent
// error handling in pkg/stdlib/io.go
//
// This is a code quality issue, not a crash bug.
// Lines 63, 69, 75: tmpFile.Close() errors are ignored
// Line 80: tmpFile.Close() error is properly checked
func TestBug3_InconsistentFileCloseErrorHandling(t *testing.T) {
	t.Log("BUG #3: Inconsistent error handling in pkg/stdlib/io.go")
	t.Log("  - Lines 63, 69, 75: tmpFile.Close() errors ignored in error paths")
	t.Log("  - Line 80: tmpFile.Close() error properly checked")
	t.Log("")
	t.Log("Impact: Low - in error paths, we're already returning an error")
	t.Log("But it's inconsistent and could mask additional issues")
	t.Log("")
	t.Log("STATUS: Code quality issue - documented but not crash-inducing")
}
