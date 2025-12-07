package lineeditor

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"os"
	"strings"
)

// Editor provides interactive line editing capabilities
type Editor struct {
	terminal *Terminal
	history  *History
	prompt   string

	// Current line state
	buffer []rune
	cursor int // Cursor position in buffer
}

// New creates a new line editor with the given history size
func New(historySize int) *Editor {
	return &Editor{
		terminal: NewTerminal(),
		history:  NewHistory(historySize),
		buffer:   make([]rune, 0, 128),
	}
}

// SetPrompt sets the prompt string
func (e *Editor) SetPrompt(prompt string) {
	e.prompt = prompt
}

// ReadLine reads a line of input with full editing support
// Returns the line and any error encountered
func (e *Editor) ReadLine(prompt string) (string, error) {
	e.prompt = prompt
	e.buffer = e.buffer[:0]
	e.cursor = 0

	// Enable raw mode
	if err := e.terminal.EnableRawMode(); err != nil {
		// Fallback to simple input if raw mode fails
		return e.fallbackReadLine()
	}
	defer e.terminal.DisableRawMode()

	// Display prompt
	e.terminal.WriteString(prompt)

	buf := make([]byte, 32)
	for {
		n, err := e.terminal.Read(buf)
		if err != nil {
			return "", err
		}

		event, _ := ParseKey(buf, n)

		switch event.Key {
		case KeyEnter:
			e.terminal.WriteString("\r\n")
			line := string(e.buffer)
			e.history.Add(line)
			return line, nil

		case KeyCtrlC:
			e.terminal.WriteString("^C\r\n")
			return "", ErrInterrupted

		case KeyCtrlD:
			if len(e.buffer) == 0 {
				e.terminal.WriteString("\r\n")
				return "", ErrEOF
			}
			// Delete character at cursor if buffer not empty
			e.deleteChar()

		case KeyCtrlL:
			// Clear screen
			e.terminal.WriteString("\033[H\033[2J")
			e.terminal.WriteString(prompt)
			e.redrawLine()

		case KeyBackspace:
			e.backspace()

		case KeyDelete:
			e.deleteChar()

		case KeyLeft:
			e.moveCursorLeft()

		case KeyRight:
			e.moveCursorRight()

		case KeyUp:
			e.historyPrevious()

		case KeyDown:
			e.historyNext()

		case KeyHome:
			e.moveCursorHome()

		case KeyEnd:
			e.moveCursorEnd()

		case KeyTab:
			// Tab completion placeholder - insert spaces for now
			e.insertChar(' ')
			e.insertChar(' ')

		case KeyChar:
			e.insertChar(event.Char)
		}
	}
}

// insertChar inserts a character at the cursor position
func (e *Editor) insertChar(ch rune) {
	// Insert character at cursor position
	e.buffer = append(e.buffer, 0)
	copy(e.buffer[e.cursor+1:], e.buffer[e.cursor:])
	e.buffer[e.cursor] = ch
	e.cursor++

	// Redraw from cursor to end of line
	e.terminal.WriteString(string(e.buffer[e.cursor-1:]))
	// Move cursor back to correct position
	if e.cursor < len(e.buffer) {
		e.terminal.WriteString(fmt.Sprintf("\033[%dD", len(e.buffer)-e.cursor))
	}
}

// backspace deletes the character before the cursor
func (e *Editor) backspace() {
	if e.cursor == 0 {
		return
	}

	// Remove character before cursor
	copy(e.buffer[e.cursor-1:], e.buffer[e.cursor:])
	e.buffer = e.buffer[:len(e.buffer)-1]
	e.cursor--

	// Move cursor back, redraw rest of line, clear trailing char
	e.terminal.WriteString("\033[D")
	e.terminal.WriteString(string(e.buffer[e.cursor:]))
	e.terminal.WriteString(" \033[D")
	// Move cursor back to correct position
	if e.cursor < len(e.buffer) {
		e.terminal.WriteString(fmt.Sprintf("\033[%dD", len(e.buffer)-e.cursor))
	}
}

// deleteChar deletes the character at the cursor position
func (e *Editor) deleteChar() {
	if e.cursor >= len(e.buffer) {
		return
	}

	// Remove character at cursor
	copy(e.buffer[e.cursor:], e.buffer[e.cursor+1:])
	e.buffer = e.buffer[:len(e.buffer)-1]

	// Redraw rest of line, clear trailing char
	e.terminal.WriteString(string(e.buffer[e.cursor:]))
	e.terminal.WriteString(" \033[D")
	// Move cursor back to correct position
	if e.cursor < len(e.buffer) {
		e.terminal.WriteString(fmt.Sprintf("\033[%dD", len(e.buffer)-e.cursor))
	}
}

// moveCursorLeft moves the cursor one position to the left
func (e *Editor) moveCursorLeft() {
	if e.cursor > 0 {
		e.cursor--
		e.terminal.WriteString("\033[D")
	}
}

// moveCursorRight moves the cursor one position to the right
func (e *Editor) moveCursorRight() {
	if e.cursor < len(e.buffer) {
		e.cursor++
		e.terminal.WriteString("\033[C")
	}
}

// moveCursorHome moves the cursor to the beginning of the line
func (e *Editor) moveCursorHome() {
	if e.cursor > 0 {
		e.terminal.WriteString(fmt.Sprintf("\033[%dD", e.cursor))
		e.cursor = 0
	}
}

// moveCursorEnd moves the cursor to the end of the line
func (e *Editor) moveCursorEnd() {
	if e.cursor < len(e.buffer) {
		e.terminal.WriteString(fmt.Sprintf("\033[%dC", len(e.buffer)-e.cursor))
		e.cursor = len(e.buffer)
	}
}

// historyPrevious navigates to the previous history entry
func (e *Editor) historyPrevious() {
	entry, ok := e.history.Previous(string(e.buffer))
	if ok {
		e.replaceLine(entry)
	}
}

// historyNext navigates to the next history entry
func (e *Editor) historyNext() {
	entry, ok := e.history.Next()
	if ok {
		e.replaceLine(entry)
	}
}

// replaceLine replaces the current buffer with a new string
func (e *Editor) replaceLine(s string) {
	// Move to start of input
	if e.cursor > 0 {
		e.terminal.WriteString(fmt.Sprintf("\033[%dD", e.cursor))
	}

	// Clear to end of line
	e.terminal.WriteString("\033[K")

	// Set new buffer and cursor
	e.buffer = []rune(s)
	e.cursor = len(e.buffer)

	// Write new content
	e.terminal.WriteString(s)
}

// redrawLine redraws the current line
func (e *Editor) redrawLine() {
	e.terminal.WriteString(string(e.buffer))
	// Move cursor to correct position
	if e.cursor < len(e.buffer) {
		e.terminal.WriteString(fmt.Sprintf("\033[%dD", len(e.buffer)-e.cursor))
	}
}

// fallbackReadLine provides simple line reading when raw mode is unavailable
func (e *Editor) fallbackReadLine() (string, error) {
	fmt.Print(e.prompt)
	var line strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return line.String(), err
		}
		if n == 0 {
			continue
		}
		if buf[0] == '\n' || buf[0] == '\r' {
			return line.String(), nil
		}
		line.WriteByte(buf[0])
	}
}

// History returns the editor's history for external access
func (e *Editor) History() *History {
	return e.history
}

// Close cleans up the editor resources
func (e *Editor) Close() error {
	return e.terminal.DisableRawMode()
}

// Sentinel errors
type editorError string

func (e editorError) Error() string { return string(e) }

const (
	ErrInterrupted editorError = "interrupted"
	ErrEOF         editorError = "EOF"
)
