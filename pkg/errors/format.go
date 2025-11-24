package errors

import (
	"fmt"
	"strings"
)

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Yellow    = "\033[33m"
	Blue      = "\033[34m"
	Magenta   = "\033[35m"
	Cyan      = "\033[36m"
	White     = "\033[37m"
	BoldRed   = "\033[1;31m"
	BoldYellow = "\033[1;33m"
	BoldBlue  = "\033[1;34m"
	BoldCyan  = "\033[1;36m"
	BoldGreen = "\033[1;32m"
)

// FormatError formats a single error with colors and context
func FormatError(err *EZError) string {
	var sb strings.Builder

	// Line 1: error[E1001]: message
	if err.Severity == SeverityError {
		sb.WriteString(fmt.Sprintf("%serror[%s]%s: %s\n",
			BoldRed, err.ErrorCode.Code, Reset, err.Message))
	} else {
		sb.WriteString(fmt.Sprintf("%swarning[%s]%s: %s\n",
			BoldYellow, err.ErrorCode.Code, Reset, err.Message))
	}

	// Line 2: --> file:line:column
	sb.WriteString(fmt.Sprintf("  %s-->%s %s:%d:%d\n",
		BoldCyan, Reset, err.File, err.Line, err.Column))

	// Line 3: empty pipe
	sb.WriteString(fmt.Sprintf("   %s|%s\n", Blue, Reset))

	// Line 4: line number and source
	if err.SourceLine != "" {
		lineNumStr := fmt.Sprintf("%d", err.Line)

		sb.WriteString(fmt.Sprintf("%s%s |%s %s\n",
			BoldBlue, lineNumStr, Reset, err.SourceLine))

		// Line 5: pointer/underline
		pointerPadding := strings.Repeat(" ", err.Column-1)
		pointerLen := err.EndColumn - err.Column
		if pointerLen < 1 {
			pointerLen = 1
		}
		pointer := strings.Repeat("^", pointerLen)

		// Show the error code description under the pointer
		detail := err.ErrorCode.Description
		if err.Severity == SeverityError {
			sb.WriteString(fmt.Sprintf("   %s|%s %s%s%s %s%s\n",
				Blue, Reset, pointerPadding, BoldRed, pointer, detail, Reset))
		} else {
			sb.WriteString(fmt.Sprintf("   %s|%s %s%s%s %s%s\n",
				Blue, Reset, pointerPadding, BoldYellow, pointer, detail, Reset))
		}

		// Line 6: empty pipe (before help)
		sb.WriteString(fmt.Sprintf("   %s|%s\n", Blue, Reset))

		// Line 7: help text (if present)
		if err.Help != "" {
			sb.WriteString(fmt.Sprintf("   %s=%s %shelp%s: %s\n",
				Blue, Reset, BoldGreen, Reset, err.Help))
		}
	}

	return sb.String()
}

// FormatErrorList formats multiple errors and warnings
func FormatErrorList(el *EZErrorList) string {
	var sb strings.Builder

	// Print all errors first
	for _, err := range el.Errors {
		sb.WriteString(FormatError(err))
		sb.WriteString("\n")
	}

	// Then print warnings
	for _, warn := range el.Warnings {
		sb.WriteString(FormatError(warn))
		sb.WriteString("\n")
	}

	// Summary line
	errorCount, warningCount := el.Count()
	if errorCount > 0 || warningCount > 0 {
		var parts []string
		if errorCount > 0 {
			errWord := "error"
			if errorCount > 1 {
				errWord = "errors"
			}
			parts = append(parts, fmt.Sprintf("%s%d %s%s", BoldRed, errorCount, errWord, Reset))
		}
		if warningCount > 0 {
			warnWord := "warning"
			if warningCount > 1 {
				warnWord = "warnings"
			}
			parts = append(parts, fmt.Sprintf("%s%d %s%s", BoldYellow, warningCount, warnWord, Reset))
		}
		sb.WriteString(fmt.Sprintf("%s: generated %s\n", Bold+"summary"+Reset, strings.Join(parts, ", ")))
	}

	return sb.String()
}

// FormatSimple formats an error without colors (for logging, etc.)
func FormatSimple(err *EZError) string {
	return fmt.Sprintf("%s[%s]: %s at %s:%d:%d",
		err.Severity, err.ErrorCode.Code, err.Message, err.File, err.Line, err.Column)
}
