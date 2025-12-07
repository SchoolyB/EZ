//go:build darwin || linux

package lineeditor

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"os"
	"syscall"
	"unsafe"
)

// Terminal handles raw mode terminal operations
type Terminal struct {
	fd       int
	origios  syscall.Termios
	isRaw    bool
	stdinFd  int
	stdoutFd int
}

// NewTerminal creates a new Terminal for the given file descriptors
func NewTerminal() *Terminal {
	return &Terminal{
		fd:       int(os.Stdin.Fd()),
		stdinFd:  int(os.Stdin.Fd()),
		stdoutFd: int(os.Stdout.Fd()),
	}
}

// EnableRawMode puts the terminal into raw mode for character-by-character input
func (t *Terminal) EnableRawMode() error {
	if t.isRaw {
		return nil
	}

	// Get current terminal attributes
	if err := t.getTermios(&t.origios); err != nil {
		return err
	}

	raw := t.origios

	// Input modes: no break, no CR to NL, no parity check, no strip char,
	// no start/stop output control
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON

	// Output modes: disable post processing
	raw.Oflag &^= syscall.OPOST

	// Control modes: set 8 bit chars
	raw.Cflag |= syscall.CS8

	// Local modes: echo off, canonical off, no extended functions, no signal chars
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG

	// Control chars: set read to return after 1 byte, no timeout
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if err := t.setTermios(&raw); err != nil {
		return err
	}

	t.isRaw = true
	return nil
}

// DisableRawMode restores the terminal to its original mode
func (t *Terminal) DisableRawMode() error {
	if !t.isRaw {
		return nil
	}

	if err := t.setTermios(&t.origios); err != nil {
		return err
	}

	t.isRaw = false
	return nil
}

// IsRaw returns whether the terminal is in raw mode
func (t *Terminal) IsRaw() bool {
	return t.isRaw
}

// Read reads up to len(buf) bytes from stdin
func (t *Terminal) Read(buf []byte) (int, error) {
	return syscall.Read(t.stdinFd, buf)
}

// Write writes bytes to stdout
func (t *Terminal) Write(buf []byte) (int, error) {
	return syscall.Write(t.stdoutFd, buf)
}

// WriteString writes a string to stdout
func (t *Terminal) WriteString(s string) (int, error) {
	return t.Write([]byte(s))
}

// getTermios gets the terminal attributes
func (t *Terminal) getTermios(termios *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(t.fd),
		ioctlReadTermios,
		uintptr(unsafe.Pointer(termios)),
		0, 0, 0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// setTermios sets the terminal attributes
func (t *Terminal) setTermios(termios *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(t.fd),
		ioctlWriteTermios,
		uintptr(unsafe.Pointer(termios)),
		0, 0, 0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// GetSize returns the terminal width and height
func (t *Terminal) GetSize() (width, height int, err error) {
	var ws struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	_, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(t.stdoutFd),
		syscall.TIOCGWINSZ,
		uintptr(unsafe.Pointer(&ws)),
		0, 0, 0,
	)
	if errno != 0 {
		return 80, 24, errno // Return default size on error
	}

	return int(ws.Col), int(ws.Row), nil
}
