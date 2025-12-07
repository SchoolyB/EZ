//go:build darwin

package lineeditor

import "syscall"

// Darwin/macOS ioctl constants for terminal control
const (
	ioctlReadTermios  = syscall.TIOCGETA
	ioctlWriteTermios = syscall.TIOCSETA
)
