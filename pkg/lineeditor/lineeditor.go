package lineeditor

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ANSI escape sequence helpers to avoid fmt.Sprintf overhead
func cursorLeft(n int) string  { return "\033[" + strconv.Itoa(n) + "D" }
func cursorRight(n int) string { return "\033[" + strconv.Itoa(n) + "C" }
func cursorUp(n int) string    { return "\033[" + strconv.Itoa(n) + "A" }
func cursorCol(n int) string   { return "\033[" + strconv.Itoa(n) + "G" }

// Editor provides interactive line editing capabilities
type Editor struct {
	terminal *Terminal
	history  *History
	prompt   string

	// Current line state (single-line mode)
	buffer []rune
	cursor int // Cursor position in buffer

	// Multi-line state
	lines          [][]rune // All lines in multi-line mode
	currentLine    int      // Current line index (0-based)
	multiLineMode  bool     // Whether we're in multi-line editing mode
	continuePrompt string   // Prompt for continuation lines
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
		e.terminal.WriteString(cursorLeft(len(e.buffer) - e.cursor))
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
		e.terminal.WriteString(cursorLeft(len(e.buffer) - e.cursor))
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
		e.terminal.WriteString(cursorLeft(len(e.buffer) - e.cursor))
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
		e.terminal.WriteString(cursorLeft(e.cursor))
		e.cursor = 0
	}
}

// moveCursorEnd moves the cursor to the end of the line
func (e *Editor) moveCursorEnd() {
	if e.cursor < len(e.buffer) {
		e.terminal.WriteString(cursorRight(len(e.buffer) - e.cursor))
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
		e.terminal.WriteString(cursorLeft(e.cursor))
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
		e.terminal.WriteString(cursorLeft(len(e.buffer) - e.cursor))
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

// ReadMultiLine reads multi-line input with full editing support including
// navigation between lines. It starts with an initial line and continues
// until braces are balanced.
func (e *Editor) ReadMultiLine(prompt, continuePrompt, initial string) (string, error) {
	e.prompt = prompt
	e.continuePrompt = continuePrompt
	e.multiLineMode = true
	e.lines = [][]rune{[]rune(initial), {}}
	e.currentLine = 1
	e.cursor = 0

	if err := e.terminal.EnableRawMode(); err != nil {
		return e.fallbackReadMultiLine(initial)
	}
	defer e.terminal.DisableRawMode()

	e.terminal.WriteString(continuePrompt)

	buf := make([]byte, 32)
	for {
		n, err := e.terminal.Read(buf)
		if err != nil {
			e.multiLineMode = false
			return "", err
		}

		event, _ := ParseKey(buf, n)

		switch event.Key {
		case KeyEnter:
			fullText := e.getMultiLineText()
			openBraces := strings.Count(fullText, "{")
			closeBraces := strings.Count(fullText, "}")

			if openBraces == closeBraces {
				e.terminal.WriteString("\r\n")
				e.history.Add(fullText)
				e.multiLineMode = false
				return fullText, nil
			}
			e.insertNewLine()

		case KeyCtrlC:
			e.terminal.WriteString("^C\r\n")
			e.multiLineMode = false
			return "", ErrInterrupted

		case KeyCtrlD:
			if len(e.lines[e.currentLine]) == 0 && len(e.lines) == 1 {
				e.terminal.WriteString("\r\n")
				e.multiLineMode = false
				return "", ErrEOF
			}
			e.deleteCharMultiLine()

		case KeyCtrlL:
			e.terminal.WriteString("\033[H\033[2J")
			e.redrawMultiLine()

		case KeyBackspace:
			e.backspaceMultiLine()

		case KeyDelete:
			e.deleteCharMultiLine()

		case KeyLeft:
			e.moveCursorLeftMultiLine()

		case KeyRight:
			e.moveCursorRightMultiLine()

		case KeyUp:
			e.moveUpMultiLine()

		case KeyDown:
			e.moveDownMultiLine()

		case KeyHome:
			e.moveCursorHomeMultiLine()

		case KeyEnd:
			e.moveCursorEndMultiLine()

		case KeyTab:
			e.insertCharMultiLine(' ')
			e.insertCharMultiLine(' ')

		case KeyChar:
			e.insertCharMultiLine(event.Char)
		}
	}
}

// getMultiLineText returns all lines joined with newlines
func (e *Editor) getMultiLineText() string {
	var parts []string
	for _, line := range e.lines {
		parts = append(parts, string(line))
	}
	return strings.Join(parts, "\n")
}

func (e *Editor) insertNewLine() {
	currentLine := e.lines[e.currentLine]
	beforeCursor := make([]rune, e.cursor)
	copy(beforeCursor, currentLine[:e.cursor])
	afterCursor := make([]rune, len(currentLine)-e.cursor)
	copy(afterCursor, currentLine[e.cursor:])

	e.lines[e.currentLine] = beforeCursor

	newLines := make([][]rune, len(e.lines)+1)
	copy(newLines, e.lines[:e.currentLine+1])
	newLines[e.currentLine+1] = afterCursor
	copy(newLines[e.currentLine+2:], e.lines[e.currentLine+1:])
	e.lines = newLines

	e.currentLine++
	e.cursor = 0

	e.terminal.WriteString("\r\n")
	e.redrawFromCurrentLine()
}

func (e *Editor) redrawMultiLine() {
	if e.currentLine > 0 {
		e.terminal.WriteString(cursorUp(e.currentLine))
	}
	e.terminal.WriteString("\r")

	for i, line := range e.lines {
		e.terminal.WriteString("\033[K")
		if i == 0 {
			e.terminal.WriteString(e.prompt)
		} else {
			e.terminal.WriteString(e.continuePrompt)
		}
		e.terminal.WriteString(string(line))

		if i < len(e.lines)-1 {
			e.terminal.WriteString("\r\n")
		}
	}

	e.positionCursor()
}

func (e *Editor) redrawFromCurrentLine() {
	startLine := e.currentLine

	for i := startLine; i < len(e.lines); i++ {
		e.terminal.WriteString("\033[K")
		if i == 0 {
			e.terminal.WriteString(e.prompt)
		} else {
			e.terminal.WriteString(e.continuePrompt)
		}
		e.terminal.WriteString(string(e.lines[i]))

		if i < len(e.lines)-1 {
			e.terminal.WriteString("\r\n")
		}
	}

	e.positionCursor()
}

func (e *Editor) positionCursor() {
	linesToMove := len(e.lines) - 1 - e.currentLine
	if linesToMove > 0 {
		e.terminal.WriteString(cursorUp(linesToMove))
	}

	e.terminal.WriteString("\r")
	promptLen := len(e.prompt)
	if e.currentLine > 0 {
		promptLen = len(e.continuePrompt)
	}
	if promptLen+e.cursor > 0 {
		e.terminal.WriteString(cursorRight(promptLen + e.cursor))
	}
}

func (e *Editor) moveUpMultiLine() {
	if e.currentLine == 0 {
		return
	}

	e.currentLine--
	e.cursor = len(e.lines[e.currentLine])

	e.terminal.WriteString("\033[A")
	e.terminal.WriteString("\r\033[K")
	if e.currentLine == 0 {
		e.terminal.WriteString(e.prompt)
	} else {
		e.terminal.WriteString(e.continuePrompt)
	}
	e.terminal.WriteString(string(e.lines[e.currentLine]))
}

func (e *Editor) moveDownMultiLine() {
	if e.currentLine >= len(e.lines)-1 {
		return
	}

	e.currentLine++
	e.cursor = len(e.lines[e.currentLine])

	promptLen := len(e.continuePrompt)
	if e.currentLine == 0 {
		promptLen = len(e.prompt)
	}

	e.terminal.WriteString("\033[B" + cursorCol(promptLen+e.cursor+1))
}

func (e *Editor) insertCharMultiLine(ch rune) {
	line := e.lines[e.currentLine]
	newLine := make([]rune, len(line)+1)
	copy(newLine, line[:e.cursor])
	newLine[e.cursor] = ch
	copy(newLine[e.cursor+1:], line[e.cursor:])
	e.lines[e.currentLine] = newLine
	e.cursor++

	e.terminal.WriteString(string(e.lines[e.currentLine][e.cursor-1:]))
	if e.cursor < len(e.lines[e.currentLine]) {
		e.terminal.WriteString(cursorLeft(len(e.lines[e.currentLine]) - e.cursor))
	}
}

func (e *Editor) backspaceMultiLine() {
	if e.cursor == 0 {
		return
	}

	line := e.lines[e.currentLine]
	newLine := make([]rune, len(line)-1)
	copy(newLine, line[:e.cursor-1])
	copy(newLine[e.cursor-1:], line[e.cursor:])
	e.lines[e.currentLine] = newLine
	e.cursor--

	e.terminal.WriteString("\033[D")
	e.terminal.WriteString(string(e.lines[e.currentLine][e.cursor:]))
	e.terminal.WriteString(" \033[D")
	if e.cursor < len(e.lines[e.currentLine]) {
		e.terminal.WriteString(cursorLeft(len(e.lines[e.currentLine]) - e.cursor))
	}
}

func (e *Editor) deleteCharMultiLine() {
	line := e.lines[e.currentLine]
	if e.cursor >= len(line) {
		return
	}

	newLine := make([]rune, len(line)-1)
	copy(newLine, line[:e.cursor])
	copy(newLine[e.cursor:], line[e.cursor+1:])
	e.lines[e.currentLine] = newLine

	e.terminal.WriteString(string(e.lines[e.currentLine][e.cursor:]))
	e.terminal.WriteString(" \033[D")
	if e.cursor < len(e.lines[e.currentLine]) {
		e.terminal.WriteString(cursorLeft(len(e.lines[e.currentLine]) - e.cursor))
	}
}

func (e *Editor) moveCursorLeftMultiLine() {
	if e.cursor > 0 {
		e.cursor--
		e.terminal.WriteString("\033[D")
	}
}

func (e *Editor) moveCursorRightMultiLine() {
	if e.cursor < len(e.lines[e.currentLine]) {
		e.cursor++
		e.terminal.WriteString("\033[C")
	}
}

func (e *Editor) moveCursorHomeMultiLine() {
	if e.cursor > 0 {
		e.terminal.WriteString(cursorLeft(e.cursor))
		e.cursor = 0
	}
}

func (e *Editor) moveCursorEndMultiLine() {
	lineLen := len(e.lines[e.currentLine])
	if e.cursor < lineLen {
		e.terminal.WriteString(cursorRight(lineLen - e.cursor))
		e.cursor = lineLen
	}
}

func (e *Editor) fallbackReadMultiLine(initial string) (string, error) {
	var lines []string
	lines = append(lines, initial)

	for {
		fmt.Print(e.continuePrompt)
		var line strings.Builder
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return strings.Join(lines, "\n"), err
			}
			if n == 0 {
				continue
			}
			if buf[0] == '\n' || buf[0] == '\r' {
				break
			}
			line.WriteByte(buf[0])
		}

		lines = append(lines, line.String())

		fullText := strings.Join(lines, "\n")
		openBraces := strings.Count(fullText, "{")
		closeBraces := strings.Count(fullText, "}")

		if openBraces == closeBraces {
			e.multiLineMode = false
			return fullText, nil
		}
	}
}

// Sentinel errors
type editorError string

func (e editorError) Error() string { return string(e) }

const (
	ErrInterrupted editorError = "interrupted"
	ErrEOF         editorError = "EOF"
)
