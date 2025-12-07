package lineeditor

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"testing"
)

// ============================================================================
// Key Parsing Tests
// ============================================================================

func TestParseSingleByteKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    byte
		wantKey  Key
		wantChar rune
	}{
		{"enter CR", 0x0d, KeyEnter, 0},
		{"enter LF", 0x0a, KeyEnter, 0},
		{"backspace DEL", 0x7f, KeyBackspace, 0},
		{"backspace BS", 0x08, KeyBackspace, 0},
		{"tab", 0x09, KeyTab, 0},
		{"ctrl-c", 0x03, KeyCtrlC, 0},
		{"ctrl-d", 0x04, KeyCtrlD, 0},
		{"ctrl-l", 0x0c, KeyCtrlL, 0},
		{"letter a", 'a', KeyChar, 'a'},
		{"letter Z", 'Z', KeyChar, 'Z'},
		{"digit 5", '5', KeyChar, '5'},
		{"space", ' ', KeyChar, ' '},
		{"symbol @", '@', KeyChar, '@'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, consumed := ParseKey([]byte{tt.input}, 1)
			if event.Key != tt.wantKey {
				t.Errorf("ParseKey(%x) key = %v, want %v", tt.input, event.Key, tt.wantKey)
			}
			if tt.wantChar != 0 && event.Char != tt.wantChar {
				t.Errorf("ParseKey(%x) char = %v, want %v", tt.input, event.Char, tt.wantChar)
			}
			if consumed != 1 {
				t.Errorf("ParseKey(%x) consumed = %d, want 1", tt.input, consumed)
			}
		})
	}
}

func TestParseEscapeSequences(t *testing.T) {
	tests := []struct {
		name         string
		input        []byte
		wantKey      Key
		wantConsumed int
	}{
		{"up arrow", []byte{0x1b, '[', 'A'}, KeyUp, 3},
		{"down arrow", []byte{0x1b, '[', 'B'}, KeyDown, 3},
		{"right arrow", []byte{0x1b, '[', 'C'}, KeyRight, 3},
		{"left arrow", []byte{0x1b, '[', 'D'}, KeyLeft, 3},
		{"home CSI H", []byte{0x1b, '[', 'H'}, KeyHome, 3},
		{"end CSI F", []byte{0x1b, '[', 'F'}, KeyEnd, 3},
		{"delete", []byte{0x1b, '[', '3', '~'}, KeyDelete, 4},
		{"home tilde", []byte{0x1b, '[', '1', '~'}, KeyHome, 4},
		{"end tilde", []byte{0x1b, '[', '4', '~'}, KeyEnd, 4},
		// SS3 sequences
		{"up SS3", []byte{0x1b, 'O', 'A'}, KeyUp, 3},
		{"down SS3", []byte{0x1b, 'O', 'B'}, KeyDown, 3},
		{"right SS3", []byte{0x1b, 'O', 'C'}, KeyRight, 3},
		{"left SS3", []byte{0x1b, 'O', 'D'}, KeyLeft, 3},
		{"home SS3", []byte{0x1b, 'O', 'H'}, KeyHome, 3},
		{"end SS3", []byte{0x1b, 'O', 'F'}, KeyEnd, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, consumed := ParseKey(tt.input, len(tt.input))
			if event.Key != tt.wantKey {
				t.Errorf("ParseKey(%v) key = %v, want %v", tt.input, event.Key, tt.wantKey)
			}
			if consumed != tt.wantConsumed {
				t.Errorf("ParseKey(%v) consumed = %d, want %d", tt.input, consumed, tt.wantConsumed)
			}
		})
	}
}

func TestParseUTF8(t *testing.T) {
	tests := []struct {
		name         string
		input        []byte
		wantChar     rune
		wantConsumed int
	}{
		{"2-byte é", []byte{0xc3, 0xa9}, 'é', 2},
		{"2-byte ñ", []byte{0xc3, 0xb1}, 'ñ', 2},
		{"3-byte euro", []byte{0xe2, 0x82, 0xac}, '€', 3},
		{"3-byte chinese", []byte{0xe4, 0xb8, 0xad}, '中', 3},
		{"4-byte emoji", []byte{0xf0, 0x9f, 0x98, 0x80}, '😀', 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, consumed := ParseKey(tt.input, len(tt.input))
			if event.Key != KeyChar {
				t.Errorf("ParseKey(%v) key = %v, want KeyChar", tt.input, event.Key)
			}
			if event.Char != tt.wantChar {
				t.Errorf("ParseKey(%v) char = %c (%U), want %c (%U)", tt.input, event.Char, event.Char, tt.wantChar, tt.wantChar)
			}
			if consumed != tt.wantConsumed {
				t.Errorf("ParseKey(%v) consumed = %d, want %d", tt.input, consumed, tt.wantConsumed)
			}
		})
	}
}

func TestParseEmptyInput(t *testing.T) {
	event, consumed := ParseKey([]byte{}, 0)
	if event.Key != KeyUnknown {
		t.Errorf("ParseKey(empty) key = %v, want KeyUnknown", event.Key)
	}
	if consumed != 0 {
		t.Errorf("ParseKey(empty) consumed = %d, want 0", consumed)
	}
}

// ============================================================================
// History Tests
// ============================================================================

func TestNewHistory(t *testing.T) {
	h := NewHistory(50)
	if h.Len() != 0 {
		t.Errorf("NewHistory().Len() = %d, want 0", h.Len())
	}
}

func TestNewHistoryDefaultSize(t *testing.T) {
	h := NewHistory(0)
	if h.maxSize != 100 {
		t.Errorf("NewHistory(0).maxSize = %d, want 100", h.maxSize)
	}

	h = NewHistory(-5)
	if h.maxSize != 100 {
		t.Errorf("NewHistory(-5).maxSize = %d, want 100", h.maxSize)
	}
}

func TestHistoryAdd(t *testing.T) {
	h := NewHistory(10)

	h.Add("first")
	h.Add("second")
	h.Add("third")

	if h.Len() != 3 {
		t.Errorf("h.Len() = %d, want 3", h.Len())
	}

	entries := h.Entries()
	expected := []string{"first", "second", "third"}
	for i, e := range expected {
		if entries[i] != e {
			t.Errorf("entries[%d] = %q, want %q", i, entries[i], e)
		}
	}
}

func TestHistoryAddEmpty(t *testing.T) {
	h := NewHistory(10)
	h.Add("")
	if h.Len() != 0 {
		t.Errorf("Adding empty string should not increase history, got len %d", h.Len())
	}
}

func TestHistoryAddDuplicate(t *testing.T) {
	h := NewHistory(10)
	h.Add("command")
	h.Add("command")
	if h.Len() != 1 {
		t.Errorf("Adding duplicate should not increase history, got len %d", h.Len())
	}
}

func TestHistoryMaxSize(t *testing.T) {
	h := NewHistory(3)
	h.Add("one")
	h.Add("two")
	h.Add("three")
	h.Add("four")

	if h.Len() != 3 {
		t.Errorf("h.Len() = %d, want 3", h.Len())
	}

	entries := h.Entries()
	if entries[0] != "two" {
		t.Errorf("entries[0] = %q, want %q", entries[0], "two")
	}
}

func TestHistoryPrevious(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")
	h.Add("third")

	// Navigate backwards
	entry, ok := h.Previous("current")
	if !ok || entry != "third" {
		t.Errorf("Previous() = %q, %v; want %q, true", entry, ok, "third")
	}

	entry, ok = h.Previous("")
	if !ok || entry != "second" {
		t.Errorf("Previous() = %q, %v; want %q, true", entry, ok, "second")
	}

	entry, ok = h.Previous("")
	if !ok || entry != "first" {
		t.Errorf("Previous() = %q, %v; want %q, true", entry, ok, "first")
	}

	// At beginning, should stay at first
	entry, ok = h.Previous("")
	if !ok || entry != "first" {
		t.Errorf("Previous() at start = %q, %v; want %q, true", entry, ok, "first")
	}
}

func TestHistoryNext(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")

	// Go back first
	h.Previous("typing")
	h.Previous("")

	// Navigate forward
	entry, ok := h.Next()
	if !ok || entry != "second" {
		t.Errorf("Next() = %q, %v; want %q, true", entry, ok, "second")
	}

	// Should return to original input
	entry, ok = h.Next()
	if !ok || entry != "typing" {
		t.Errorf("Next() = %q, %v; want %q, true", entry, ok, "typing")
	}

	// No more next
	entry, ok = h.Next()
	if ok {
		t.Errorf("Next() at end should return false, got %q, %v", entry, ok)
	}
}

func TestHistoryPreviousEmpty(t *testing.T) {
	h := NewHistory(10)
	entry, ok := h.Previous("current")
	if ok {
		t.Errorf("Previous() on empty history should return false, got %q, %v", entry, ok)
	}
}

func TestHistoryReset(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")

	h.Previous("current")
	h.Reset()

	// After reset, should start from end again
	entry, ok := h.Previous("new")
	if !ok || entry != "second" {
		t.Errorf("After Reset(), Previous() = %q, %v; want %q, true", entry, ok, "second")
	}
}

func TestHistoryClear(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")
	h.Clear()

	if h.Len() != 0 {
		t.Errorf("After Clear(), Len() = %d, want 0", h.Len())
	}
}

// ============================================================================
// Editor Tests
// ============================================================================

func TestNewEditor(t *testing.T) {
	e := New(100)
	if e == nil {
		t.Fatal("New() returned nil")
	}
	if e.history == nil {
		t.Error("New() editor has nil history")
	}
	if e.terminal == nil {
		t.Error("New() editor has nil terminal")
	}
}

func TestEditorHistory(t *testing.T) {
	e := New(50)
	h := e.History()
	if h == nil {
		t.Fatal("History() returned nil")
	}
	if h.maxSize != 50 {
		t.Errorf("History maxSize = %d, want 50", h.maxSize)
	}
}

func TestEditorSetPrompt(t *testing.T) {
	e := New(10)
	e.SetPrompt(">> ")
	if e.prompt != ">> " {
		t.Errorf("SetPrompt() prompt = %q, want %q", e.prompt, ">> ")
	}
}

func TestEditorErrors(t *testing.T) {
	if ErrInterrupted.Error() != "interrupted" {
		t.Errorf("ErrInterrupted.Error() = %q, want %q", ErrInterrupted.Error(), "interrupted")
	}
	if ErrEOF.Error() != "EOF" {
		t.Errorf("ErrEOF.Error() = %q, want %q", ErrEOF.Error(), "EOF")
	}
}

// ============================================================================
// Terminal Tests
// ============================================================================

func TestNewTerminal(t *testing.T) {
	term := NewTerminal()
	if term == nil {
		t.Fatal("NewTerminal() returned nil")
	}
	if term.isRaw {
		t.Error("New terminal should not be in raw mode")
	}
}

// ============================================================================
// Integration-style buffer tests
// ============================================================================

// TestBufferOperations tests internal buffer manipulation logic
func TestBufferOperations(t *testing.T) {
	e := New(10)

	// Test insertChar logic by manually manipulating buffer
	e.buffer = []rune("hello")
	e.cursor = 5

	// Simulate inserting at end
	ch := '!'
	e.buffer = append(e.buffer, 0)
	copy(e.buffer[e.cursor+1:], e.buffer[e.cursor:])
	e.buffer[e.cursor] = ch
	e.cursor++

	if string(e.buffer) != "hello!" {
		t.Errorf("Insert at end: buffer = %q, want %q", string(e.buffer), "hello!")
	}
	if e.cursor != 6 {
		t.Errorf("Insert at end: cursor = %d, want 6", e.cursor)
	}
}

func TestBufferInsertMiddle(t *testing.T) {
	e := New(10)
	e.buffer = []rune("helo")
	e.cursor = 2 // After 'e'

	// Insert 'l' at cursor
	ch := 'l'
	e.buffer = append(e.buffer, 0)
	copy(e.buffer[e.cursor+1:], e.buffer[e.cursor:])
	e.buffer[e.cursor] = ch
	e.cursor++

	if string(e.buffer) != "hello" {
		t.Errorf("Insert middle: buffer = %q, want %q", string(e.buffer), "hello")
	}
	if e.cursor != 3 {
		t.Errorf("Insert middle: cursor = %d, want 3", e.cursor)
	}
}

func TestBufferBackspace(t *testing.T) {
	e := New(10)
	e.buffer = []rune("hello")
	e.cursor = 5

	// Simulate backspace
	if e.cursor > 0 {
		copy(e.buffer[e.cursor-1:], e.buffer[e.cursor:])
		e.buffer = e.buffer[:len(e.buffer)-1]
		e.cursor--
	}

	if string(e.buffer) != "hell" {
		t.Errorf("Backspace: buffer = %q, want %q", string(e.buffer), "hell")
	}
	if e.cursor != 4 {
		t.Errorf("Backspace: cursor = %d, want 4", e.cursor)
	}
}

func TestBufferBackspaceMiddle(t *testing.T) {
	e := New(10)
	e.buffer = []rune("hello")
	e.cursor = 3 // After 'l'

	// Simulate backspace
	if e.cursor > 0 {
		copy(e.buffer[e.cursor-1:], e.buffer[e.cursor:])
		e.buffer = e.buffer[:len(e.buffer)-1]
		e.cursor--
	}

	if string(e.buffer) != "helo" {
		t.Errorf("Backspace middle: buffer = %q, want %q", string(e.buffer), "helo")
	}
	if e.cursor != 2 {
		t.Errorf("Backspace middle: cursor = %d, want 2", e.cursor)
	}
}

func TestBufferDelete(t *testing.T) {
	e := New(10)
	e.buffer = []rune("hello")
	e.cursor = 0

	// Simulate delete
	if e.cursor < len(e.buffer) {
		copy(e.buffer[e.cursor:], e.buffer[e.cursor+1:])
		e.buffer = e.buffer[:len(e.buffer)-1]
	}

	if string(e.buffer) != "ello" {
		t.Errorf("Delete: buffer = %q, want %q", string(e.buffer), "ello")
	}
	if e.cursor != 0 {
		t.Errorf("Delete: cursor = %d, want 0", e.cursor)
	}
}

func TestBufferCursorMovement(t *testing.T) {
	e := New(10)
	e.buffer = []rune("hello")
	e.cursor = 2

	// Move left
	if e.cursor > 0 {
		e.cursor--
	}
	if e.cursor != 1 {
		t.Errorf("Move left: cursor = %d, want 1", e.cursor)
	}

	// Move right
	if e.cursor < len(e.buffer) {
		e.cursor++
	}
	if e.cursor != 2 {
		t.Errorf("Move right: cursor = %d, want 2", e.cursor)
	}

	// Move home
	e.cursor = 0
	if e.cursor != 0 {
		t.Errorf("Move home: cursor = %d, want 0", e.cursor)
	}

	// Move end
	e.cursor = len(e.buffer)
	if e.cursor != 5 {
		t.Errorf("Move end: cursor = %d, want 5", e.cursor)
	}
}

func TestBufferReplaceLine(t *testing.T) {
	e := New(10)
	e.buffer = []rune("hello")
	e.cursor = 3

	// Simulate replaceLine
	newLine := "world"
	e.buffer = []rune(newLine)
	e.cursor = len(e.buffer)

	if string(e.buffer) != "world" {
		t.Errorf("ReplaceLine: buffer = %q, want %q", string(e.buffer), "world")
	}
	if e.cursor != 5 {
		t.Errorf("ReplaceLine: cursor = %d, want 5", e.cursor)
	}
}
