package errors

import (
	"strings"
	"testing"
)

// TestErrorCodeStructure verifies error codes have required fields
func TestErrorCodeStructure(t *testing.T) {
	// Sample of error codes from each category
	codes := []ErrorCode{
		E1001, E1004, E1010,
		E2001, E2005, E2020,
		E3001, E3008, E3015,
		E4001, E4009,
		E5001, E5008,
		E6001, E6002,
		E7001, E7002,
		E8001, E8006,
		E9001, E9004,
		E10001,
		E11001,
		E12001, E12003,
		W1001, W2001, W3001, W4001,
	}

	for _, code := range codes {
		t.Run(code.Code, func(t *testing.T) {
			if code.Code == "" {
				t.Error("ErrorCode.Code is empty")
			}
			if code.Name == "" {
				t.Error("ErrorCode.Name is empty")
			}
			if code.Description == "" {
				t.Error("ErrorCode.Description is empty")
			}
			// Verify code format (Exxxx or Wxxxx)
			if !strings.HasPrefix(code.Code, "E") && !strings.HasPrefix(code.Code, "W") {
				t.Errorf("ErrorCode.Code %q should start with E or W", code.Code)
			}
		})
	}
}

// TestLexerErrorCodes verifies all E1xxx codes exist
func TestLexerErrorCodes(t *testing.T) {
	lexerCodes := []struct {
		code     ErrorCode
		expected string
	}{
		{E1001, "E1001"},
		{E1002, "E1002"},
		{E1003, "E1003"},
		{E1004, "E1004"},
		{E1005, "E1005"},
		{E1006, "E1006"},
		{E1007, "E1007"},
		{E1008, "E1008"},
		{E1009, "E1009"},
		{E1010, "E1010"},
		{E1011, "E1011"},
		{E1012, "E1012"},
		{E1013, "E1013"},
		{E1014, "E1014"},
		{E1015, "E1015"},
		{E1016, "E1016"},
	}

	for _, tt := range lexerCodes {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.code.Code != tt.expected {
				t.Errorf("got %q, want %q", tt.code.Code, tt.expected)
			}
		})
	}
}

// TestParseErrorCodes verifies all E2xxx codes exist
func TestParseErrorCodes(t *testing.T) {
	parseCodes := []struct {
		code     ErrorCode
		expected string
	}{
		{E2001, "E2001"},
		{E2002, "E2002"},
		{E2003, "E2003"},
		{E2004, "E2004"},
		{E2005, "E2005"},
		{E2006, "E2006"},
		{E2007, "E2007"},
		{E2008, "E2008"},
		{E2009, "E2009"},
		{E2010, "E2010"},
		{E2011, "E2011"},
		{E2012, "E2012"},
		{E2013, "E2013"},
		{E2014, "E2014"},
		{E2015, "E2015"},
		{E2016, "E2016"},
		{E2017, "E2017"},
		{E2018, "E2018"},
		{E2019, "E2019"},
		{E2020, "E2020"},
		{E2021, "E2021"},
		{E2022, "E2022"},
		{E2023, "E2023"},
		{E2024, "E2024"},
		{E2025, "E2025"},
		{E2026, "E2026"},
		{E2027, "E2027"},
		{E2028, "E2028"},
		{E2029, "E2029"},
		{E2030, "E2030"},
		{E2031, "E2031"},
		{E2032, "E2032"},
	}

	for _, tt := range parseCodes {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.code.Code != tt.expected {
				t.Errorf("got %q, want %q", tt.code.Code, tt.expected)
			}
		})
	}
}

// TestNewError creates an error with correct fields
func TestNewError(t *testing.T) {
	err := NewError(E1001, "test message", "test.ez", 10, 5)

	if err.ErrorCode.Code != "E1001" {
		t.Errorf("ErrorCode.Code = %q, want %q", err.ErrorCode.Code, "E1001")
	}
	if err.Message != "test message" {
		t.Errorf("Message = %q, want %q", err.Message, "test message")
	}
	if err.File != "test.ez" {
		t.Errorf("File = %q, want %q", err.File, "test.ez")
	}
	if err.Line != 10 {
		t.Errorf("Line = %d, want %d", err.Line, 10)
	}
	if err.Column != 5 {
		t.Errorf("Column = %d, want %d", err.Column, 5)
	}
	if err.EndColumn != 6 {
		t.Errorf("EndColumn = %d, want %d", err.EndColumn, 6)
	}
	if err.Severity != SeverityError {
		t.Errorf("Severity = %q, want %q", err.Severity, SeverityError)
	}
}

// TestNewErrorWithSource creates error with source line
func TestNewErrorWithSource(t *testing.T) {
	err := NewErrorWithSource(E2001, "unexpected", "main.ez", 5, 3, "temp x int = 42")

	if err.SourceLine != "temp x int = 42" {
		t.Errorf("SourceLine = %q, want %q", err.SourceLine, "temp x int = 42")
	}
	if err.ErrorCode.Code != "E2001" {
		t.Errorf("ErrorCode.Code = %q, want %q", err.ErrorCode.Code, "E2001")
	}
}

// TestNewErrorWithHelp creates error with help text
func TestNewErrorWithHelp(t *testing.T) {
	err := NewErrorWithHelp(E3001, "type mismatch", "main.ez", 5, 3, "temp x int = \"hello\"", "expected int, got string")

	if err.Help != "expected int, got string" {
		t.Errorf("Help = %q, want %q", err.Help, "expected int, got string")
	}
	if err.SourceLine != "temp x int = \"hello\"" {
		t.Errorf("SourceLine = %q, want %q", err.SourceLine, "temp x int = \"hello\"")
	}
}

// TestEZErrorError tests the Error() interface implementation
func TestEZErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      *EZError
		expected string
	}{
		{
			name: "error severity",
			err: &EZError{
				ErrorCode: E1001,
				Message:   "illegal character",
				Severity:  SeverityError,
			},
			expected: "error[E1001]: illegal character",
		},
		{
			name: "warning severity",
			err: &EZError{
				ErrorCode: W1001,
				Message:   "unused variable",
				Severity:  SeverityWarning,
			},
			expected: "warning[W1001]: unused variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestSetSpan modifies column range
func TestSetSpan(t *testing.T) {
	err := NewError(E1001, "test", "test.ez", 1, 1)
	err.SetSpan(5, 10)

	if err.Column != 5 {
		t.Errorf("Column = %d, want %d", err.Column, 5)
	}
	if err.EndColumn != 10 {
		t.Errorf("EndColumn = %d, want %d", err.EndColumn, 10)
	}
}

// TestWithHelp adds help text
func TestWithHelp(t *testing.T) {
	err := NewError(E1001, "test", "test.ez", 1, 1)
	err.WithHelp("try this instead")

	if err.Help != "try this instead" {
		t.Errorf("Help = %q, want %q", err.Help, "try this instead")
	}
}

// TestWithSource adds source line
func TestWithSource(t *testing.T) {
	err := NewError(E1001, "test", "test.ez", 1, 1)
	err.WithSource("const x = 42")

	if err.SourceLine != "const x = 42" {
		t.Errorf("SourceLine = %q, want %q", err.SourceLine, "const x = 42")
	}
}

// TestMethodChaining verifies fluent interface works
func TestMethodChaining(t *testing.T) {
	err := NewError(E1001, "test", "test.ez", 1, 1).
		SetSpan(5, 15).
		WithSource("some code").
		WithHelp("some help")

	if err.Column != 5 {
		t.Errorf("Column = %d, want %d", err.Column, 5)
	}
	if err.EndColumn != 15 {
		t.Errorf("EndColumn = %d, want %d", err.EndColumn, 15)
	}
	if err.SourceLine != "some code" {
		t.Errorf("SourceLine = %q, want %q", err.SourceLine, "some code")
	}
	if err.Help != "some help" {
		t.Errorf("Help = %q, want %q", err.Help, "some help")
	}
}

// TestEZErrorList tests error list operations
func TestEZErrorList(t *testing.T) {
	t.Run("NewErrorList creates empty list", func(t *testing.T) {
		el := NewErrorList()
		if len(el.Errors) != 0 {
			t.Errorf("Errors length = %d, want 0", len(el.Errors))
		}
		if len(el.Warnings) != 0 {
			t.Errorf("Warnings length = %d, want 0", len(el.Warnings))
		}
	})

	t.Run("AddError adds to Errors", func(t *testing.T) {
		el := NewErrorList()
		err := NewError(E1001, "test", "test.ez", 1, 1)
		el.AddError(err)

		if len(el.Errors) != 1 {
			t.Errorf("Errors length = %d, want 1", len(el.Errors))
		}
		if el.Errors[0].Severity != SeverityError {
			t.Errorf("Severity = %q, want %q", el.Errors[0].Severity, SeverityError)
		}
	})

	t.Run("AddWarning adds to Warnings", func(t *testing.T) {
		el := NewErrorList()
		warn := NewError(W1001, "unused", "test.ez", 1, 1)
		el.AddWarning(warn)

		if len(el.Warnings) != 1 {
			t.Errorf("Warnings length = %d, want 1", len(el.Warnings))
		}
		if el.Warnings[0].Severity != SeverityWarning {
			t.Errorf("Severity = %q, want %q", el.Warnings[0].Severity, SeverityWarning)
		}
	})

	t.Run("HasErrors returns correct value", func(t *testing.T) {
		el := NewErrorList()
		if el.HasErrors() {
			t.Error("HasErrors() = true, want false")
		}

		el.AddError(NewError(E1001, "test", "test.ez", 1, 1))
		if !el.HasErrors() {
			t.Error("HasErrors() = false, want true")
		}
	})

	t.Run("HasWarnings returns correct value", func(t *testing.T) {
		el := NewErrorList()
		if el.HasWarnings() {
			t.Error("HasWarnings() = true, want false")
		}

		el.AddWarning(NewError(W1001, "test", "test.ez", 1, 1))
		if !el.HasWarnings() {
			t.Error("HasWarnings() = false, want true")
		}
	})

	t.Run("Count returns correct counts", func(t *testing.T) {
		el := NewErrorList()
		el.AddError(NewError(E1001, "err1", "test.ez", 1, 1))
		el.AddError(NewError(E1002, "err2", "test.ez", 2, 1))
		el.AddWarning(NewError(W1001, "warn1", "test.ez", 3, 1))

		errors, warnings := el.Count()
		if errors != 2 {
			t.Errorf("errors = %d, want 2", errors)
		}
		if warnings != 1 {
			t.Errorf("warnings = %d, want 1", warnings)
		}
	})
}

// TestGetSourceLine retrieves correct line from source
func TestGetSourceLine(t *testing.T) {
	source := "line 1\nline 2\nline 3\nline 4"

	tests := []struct {
		lineNum  int
		expected string
	}{
		{1, "line 1"},
		{2, "line 2"},
		{3, "line 3"},
		{4, "line 4"},
		{0, ""},  // Invalid line number
		{5, ""},  // Out of range
		{-1, ""}, // Negative
	}

	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.expected, " ", "_"), func(t *testing.T) {
			got := GetSourceLine(source, tt.lineNum)
			if got != tt.expected {
				t.Errorf("GetSourceLine(%d) = %q, want %q", tt.lineNum, got, tt.expected)
			}
		})
	}
}

// TestSeverityConstants verifies severity values
func TestSeverityConstants(t *testing.T) {
	if SeverityError != "error" {
		t.Errorf("SeverityError = %q, want %q", SeverityError, "error")
	}
	if SeverityWarning != "warning" {
		t.Errorf("SeverityWarning = %q, want %q", SeverityWarning, "warning")
	}
}
