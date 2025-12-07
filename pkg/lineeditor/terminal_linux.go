//go:build linux

package lineeditor

import "syscall"

// Linux ioctl constants for terminal control
const (
	ioctlReadTermios  = syscall.TCGETS
	ioctlWriteTermios = syscall.TCSETS
)
