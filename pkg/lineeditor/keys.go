package lineeditor

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

// Key represents a parsed keyboard input
type Key int

const (
	KeyUnknown Key = iota
	KeyChar        // Regular character
	KeyEnter
	KeyBackspace
	KeyDelete
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyTab
	KeyCtrlC
	KeyCtrlD
	KeyCtrlL
)

// KeyEvent represents a keyboard event with the key type and optional rune
type KeyEvent struct {
	Key  Key
	Char rune
}

// ParseKey parses raw input bytes into a KeyEvent
// Returns the KeyEvent and number of bytes consumed
func ParseKey(buf []byte, n int) (KeyEvent, int) {
	if n == 0 {
		return KeyEvent{Key: KeyUnknown}, 0
	}

	// Single byte characters
	if n == 1 {
		return parseSingleByte(buf[0]), 1
	}

	// Check for escape sequences
	if buf[0] == 0x1b {
		return parseEscapeSequence(buf, n)
	}

	// Multi-byte UTF-8 character
	r, size := decodeUTF8(buf, n)
	if size > 0 {
		return KeyEvent{Key: KeyChar, Char: r}, size
	}

	return KeyEvent{Key: KeyUnknown}, 1
}

// parseSingleByte handles single byte input
func parseSingleByte(b byte) KeyEvent {
	switch b {
	case 0x03: // Ctrl+C
		return KeyEvent{Key: KeyCtrlC}
	case 0x04: // Ctrl+D
		return KeyEvent{Key: KeyCtrlD}
	case 0x09: // Tab
		return KeyEvent{Key: KeyTab}
	case 0x0c: // Ctrl+L
		return KeyEvent{Key: KeyCtrlL}
	case 0x0d, 0x0a: // Enter (CR or LF)
		return KeyEvent{Key: KeyEnter}
	case 0x7f, 0x08: // Backspace (DEL or BS)
		return KeyEvent{Key: KeyBackspace}
	case 0x1b: // Escape - but alone, treat as unknown
		return KeyEvent{Key: KeyUnknown}
	default:
		if b >= 32 && b < 127 {
			return KeyEvent{Key: KeyChar, Char: rune(b)}
		}
		return KeyEvent{Key: KeyUnknown}
	}
}

// parseEscapeSequence handles ANSI escape sequences
func parseEscapeSequence(buf []byte, n int) (KeyEvent, int) {
	if n < 2 {
		return KeyEvent{Key: KeyUnknown}, 1
	}

	// CSI sequences: ESC [
	if buf[1] == '[' {
		if n < 3 {
			return KeyEvent{Key: KeyUnknown}, 2
		}

		switch buf[2] {
		case 'A': // Up arrow
			return KeyEvent{Key: KeyUp}, 3
		case 'B': // Down arrow
			return KeyEvent{Key: KeyDown}, 3
		case 'C': // Right arrow
			return KeyEvent{Key: KeyRight}, 3
		case 'D': // Left arrow
			return KeyEvent{Key: KeyLeft}, 3
		case 'H': // Home
			return KeyEvent{Key: KeyHome}, 3
		case 'F': // End
			return KeyEvent{Key: KeyEnd}, 3
		case '1': // Could be Home (ESC[1~) or other
			if n >= 4 && buf[3] == '~' {
				return KeyEvent{Key: KeyHome}, 4
			}
		case '3': // Delete (ESC[3~)
			if n >= 4 && buf[3] == '~' {
				return KeyEvent{Key: KeyDelete}, 4
			}
		case '4': // End (ESC[4~)
			if n >= 4 && buf[3] == '~' {
				return KeyEvent{Key: KeyEnd}, 4
			}
		}
	}

	// SS3 sequences: ESC O (alternate arrow keys on some terminals)
	if buf[1] == 'O' && n >= 3 {
		switch buf[2] {
		case 'A':
			return KeyEvent{Key: KeyUp}, 3
		case 'B':
			return KeyEvent{Key: KeyDown}, 3
		case 'C':
			return KeyEvent{Key: KeyRight}, 3
		case 'D':
			return KeyEvent{Key: KeyLeft}, 3
		case 'H':
			return KeyEvent{Key: KeyHome}, 3
		case 'F':
			return KeyEvent{Key: KeyEnd}, 3
		}
	}

	return KeyEvent{Key: KeyUnknown}, 1
}

// decodeUTF8 decodes a UTF-8 character from the buffer
func decodeUTF8(buf []byte, n int) (rune, int) {
	if n == 0 {
		return 0, 0
	}

	b := buf[0]

	// Single byte (ASCII)
	if b < 0x80 {
		return rune(b), 1
	}

	// Multi-byte sequence
	var size int
	var min rune
	var r rune

	switch {
	case b&0xe0 == 0xc0: // 2-byte sequence
		size = 2
		min = 0x80
		r = rune(b & 0x1f)
	case b&0xf0 == 0xe0: // 3-byte sequence
		size = 3
		min = 0x800
		r = rune(b & 0x0f)
	case b&0xf8 == 0xf0: // 4-byte sequence
		size = 4
		min = 0x10000
		r = rune(b & 0x07)
	default:
		return 0xFFFD, 1 // Invalid, return replacement char
	}

	if n < size {
		return 0xFFFD, 1 // Incomplete sequence
	}

	for i := 1; i < size; i++ {
		if buf[i]&0xc0 != 0x80 {
			return 0xFFFD, 1 // Invalid continuation byte
		}
		r = r<<6 | rune(buf[i]&0x3f)
	}

	if r < min {
		return 0xFFFD, 1 // Overlong encoding
	}

	return r, size
}
