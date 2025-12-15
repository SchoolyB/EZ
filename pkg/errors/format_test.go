package errors

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"strings"
	"testing"
)

// TestFormatError verifies error formatting
func TestFormatError(t *testing.T) {
	t.Run("formats error with all fields", func(t *testing.T) {
		err := &EZError{
			ErrorCode:  E1001,
			Message:    "illegal character '@'",
			File:       "main.ez",
			Line:       10,
			Column:     5,
			EndColumn:  6,
			SourceLine: "temp @x = 5",
			Help:       "remove the @ symbol",
			Severity:   SeverityError,
		}

		output := FormatError(err)

		// Check key components are present
		if !strings.Contains(output, "error[E1001]") {
			t.Error("output should contain error code")
		}
		if !strings.Contains(output, "illegal character '@'") {
			t.Error("output should contain message")
		}
		if !strings.Contains(output, "main.ez:10:5") {
			t.Error("output should contain file location")
		}
		if !strings.Contains(output, "temp @x = 5") {
			t.Error("output should contain source line")
		}
		if !strings.Contains(output, "help") {
			t.Error("output should contain help section")
		}
	})

	t.Run("formats warning correctly", func(t *testing.T) {
		warn := &EZError{
			ErrorCode:  W1001,
			Message:    "unused variable 'x'",
			File:       "main.ez",
			Line:       5,
			Column:     10,
			EndColumn:  11,
			SourceLine: "temp x int = 42",
			Severity:   SeverityWarning,
		}

		output := FormatError(warn)

		if !strings.Contains(output, "warning[W1001]") {
			t.Error("output should contain warning code")
		}
	})

	t.Run("handles missing source line", func(t *testing.T) {
		err := &EZError{
			ErrorCode: E1001,
			Message:   "test error",
			File:      "test.ez",
			Line:      1,
			Column:    1,
			Severity:  SeverityError,
		}

		// Should not panic
		output := FormatError(err)
		if output == "" {
			t.Error("output should not be empty")
		}
	})

	t.Run("handles zero column", func(t *testing.T) {
		err := &EZError{
			ErrorCode:  E1001,
			Message:    "test",
			File:       "test.ez",
			Line:       1,
			Column:     0, // Edge case
			EndColumn:  1,
			SourceLine: "code",
			Severity:   SeverityError,
		}

		// Should not panic
		output := FormatError(err)
		if output == "" {
			t.Error("output should not be empty")
		}
	})
}

// TestFormatErrorList verifies error list formatting
func TestFormatErrorList(t *testing.T) {
	t.Run("formats multiple errors and warnings", func(t *testing.T) {
		el := NewErrorList()
		el.AddError(&EZError{
			ErrorCode:  E1001,
			Message:    "error 1",
			File:       "test.ez",
			Line:       1,
			Column:     1,
			EndColumn:  2,
			SourceLine: "line 1",
		})
		el.AddError(&EZError{
			ErrorCode:  E2001,
			Message:    "error 2",
			File:       "test.ez",
			Line:       2,
			Column:     1,
			EndColumn:  2,
			SourceLine: "line 2",
		})
		el.AddWarning(&EZError{
			ErrorCode:  W1001,
			Message:    "warning 1",
			File:       "test.ez",
			Line:       3,
			Column:     1,
			EndColumn:  2,
			SourceLine: "line 3",
		})

		output := FormatErrorList(el)

		// Check all items present
		if !strings.Contains(output, "E1001") {
			t.Error("output should contain first error")
		}
		if !strings.Contains(output, "E2001") {
			t.Error("output should contain second error")
		}
		if !strings.Contains(output, "W1001") {
			t.Error("output should contain warning")
		}

		// Check summary
		if !strings.Contains(output, "2 errors") {
			t.Error("output should contain error count")
		}
		if !strings.Contains(output, "1 warning") {
			t.Error("output should contain warning count")
		}
	})

	t.Run("formats single error correctly", func(t *testing.T) {
		el := NewErrorList()
		el.AddError(&EZError{
			ErrorCode:  E1001,
			Message:    "single error",
			File:       "test.ez",
			Line:       1,
			Column:     1,
			EndColumn:  2,
			SourceLine: "code",
		})

		output := FormatErrorList(el)

		// Should say "1 error" not "1 errors"
		if !strings.Contains(output, "1 error") {
			t.Error("output should say '1 error'")
		}
		if strings.Contains(output, "1 errors") {
			t.Error("output should not say '1 errors'")
		}
	})

	t.Run("empty list produces no summary", func(t *testing.T) {
		el := NewErrorList()
		output := FormatErrorList(el)

		if strings.Contains(output, "summary") {
			t.Error("empty list should not have summary")
		}
	})
}

// TestFormatSimple verifies simple formatting
func TestFormatSimple(t *testing.T) {
	err := &EZError{
		ErrorCode: E1001,
		Message:   "illegal character",
		File:      "main.ez",
		Line:      10,
		Column:    5,
		Severity:  SeverityError,
	}

	output := FormatSimple(err)
	expected := "error[E1001]: illegal character at main.ez:10:5"

	if output != expected {
		t.Errorf("FormatSimple() = %q, want %q", output, expected)
	}
}

// TestColorConstants verifies ANSI codes are defined
func TestColorConstants(t *testing.T) {
	colors := []struct {
		name  string
		value string
	}{
		{"Reset", Reset},
		{"Bold", Bold},
		{"Red", Red},
		{"Green", Green},
		{"Yellow", Yellow},
		{"Blue", Blue},
		{"Cyan", Cyan},
		{"BoldRed", BoldRed},
		{"BoldYellow", BoldYellow},
		{"BoldBlue", BoldBlue},
		{"BoldCyan", BoldCyan},
		{"BoldGreen", BoldGreen},
	}

	for _, c := range colors {
		t.Run(c.name, func(t *testing.T) {
			if c.value == "" {
				t.Errorf("%s should not be empty", c.name)
			}
			if !strings.HasPrefix(c.value, "\033[") {
				t.Errorf("%s should be an ANSI escape code", c.name)
			}
		})
	}
}
