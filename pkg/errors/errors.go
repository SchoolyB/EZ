package errors

import (
	"fmt"
	"strings"
)

// Severity levels
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

// EZError represents a comprehensive error with context
type EZError struct {
	ErrorCode  ErrorCode // The error code (E1001, etc.)
	Message    string    // Human-readable message
	File       string    // File path
	Line       int       // Line number (1-indexed)
	Column     int       // Column number (1-indexed)
	EndColumn  int       // End column for underlining
	SourceLine string    // The actual source code line
	Help       string    // Optional help text
	Severity   string    // "error" or "warning"
}

// Error implements the error interface
func (e *EZError) Error() string {
	return fmt.Sprintf("%s[%s]: %s", e.Severity, e.ErrorCode.Code, e.Message)
}

// EZErrorList holds multiple errors/warnings
type EZErrorList struct {
	Errors   []*EZError
	Warnings []*EZError
}

// NewErrorList creates a new error list
func NewErrorList() *EZErrorList {
	return &EZErrorList{
		Errors:   []*EZError{},
		Warnings: []*EZError{},
	}
}

// AddError adds an error to the list
func (el *EZErrorList) AddError(err *EZError) {
	err.Severity = SeverityError
	el.Errors = append(el.Errors, err)
}

// AddWarning adds a warning to the list
func (el *EZErrorList) AddWarning(warn *EZError) {
	warn.Severity = SeverityWarning
	el.Warnings = append(el.Warnings, warn)
}

// HasErrors returns true if there are any errors
func (el *EZErrorList) HasErrors() bool {
	return len(el.Errors) > 0
}

// HasWarnings returns true if there are any warnings
func (el *EZErrorList) HasWarnings() bool {
	return len(el.Warnings) > 0
}

// Count returns total number of errors and warnings
func (el *EZErrorList) Count() (errors int, warnings int) {
	return len(el.Errors), len(el.Warnings)
}

// NewError creates a new EZError
func NewError(code ErrorCode, message string, file string, line, column int) *EZError {
	return &EZError{
		ErrorCode: code,
		Message:   message,
		File:      file,
		Line:      line,
		Column:    column,
		EndColumn: column + 1,
		Severity:  SeverityError,
	}
}

// NewErrorWithSource creates an error with source context
func NewErrorWithSource(code ErrorCode, message string, file string, line, column int, sourceLine string) *EZError {
	err := NewError(code, message, file, line, column)
	err.SourceLine = sourceLine
	return err
}

// NewErrorWithHelp creates an error with help text
func NewErrorWithHelp(code ErrorCode, message string, file string, line, column int, sourceLine, help string) *EZError {
	err := NewErrorWithSource(code, message, file, line, column, sourceLine)
	err.Help = help
	return err
}

// SetSpan sets the column span for underlining
func (e *EZError) SetSpan(start, end int) *EZError {
	e.Column = start
	e.EndColumn = end
	return e
}

// WithHelp adds help text to an error
func (e *EZError) WithHelp(help string) *EZError {
	e.Help = help
	return e
}

// WithSource adds source line to an error
func (e *EZError) WithSource(source string) *EZError {
	e.SourceLine = source
	return e
}

// GetSourceLines retrieves source lines from file content
// Returns the specified line and optionally surrounding context
func GetSourceLine(source string, lineNum int) string {
	lines := strings.Split(source, "\n")
	if lineNum < 1 || lineNum > len(lines) {
		return ""
	}
	return lines[lineNum-1]
}
