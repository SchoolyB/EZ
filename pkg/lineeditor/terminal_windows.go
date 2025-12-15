//go:build windows

package lineeditor

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"os"
)

// Terminal handles terminal operations (limited support on Windows)
type Terminal struct {
	isRaw bool
}

// NewTerminal creates a new Terminal
func NewTerminal() *Terminal {
	return &Terminal{}
}

// EnableRawMode attempts to put the terminal into raw mode
// On Windows, this currently returns an error to trigger fallback behavior
func (t *Terminal) EnableRawMode() error {
	// Windows console raw mode requires different APIs
	// Return error to trigger fallback to simple line reading
	return errNotSupported
}

// DisableRawMode restores the terminal to its original mode
func (t *Terminal) DisableRawMode() error {
	t.isRaw = false
	return nil
}

// IsRaw returns whether the terminal is in raw mode
func (t *Terminal) IsRaw() bool {
	return t.isRaw
}

// Read reads up to len(buf) bytes from stdin
func (t *Terminal) Read(buf []byte) (int, error) {
	return os.Stdin.Read(buf)
}

// Write writes bytes to stdout
func (t *Terminal) Write(buf []byte) (int, error) {
	return os.Stdout.Write(buf)
}

// WriteString writes a string to stdout
func (t *Terminal) WriteString(s string) (int, error) {
	return t.Write([]byte(s))
}

// GetSize returns the terminal width and height
func (t *Terminal) GetSize() (width, height int, err error) {
	return 80, 24, nil // Default size on Windows
}

// errNotSupported indicates raw mode is not supported
type notSupportedError struct{}

func (notSupportedError) Error() string { return "raw mode not supported on Windows" }

var errNotSupported = notSupportedError{}
