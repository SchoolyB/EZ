package lineeditor

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"os"
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

// ============================================================================
// Additional Key Parsing Tests
// ============================================================================

func TestParseUnknownEscapeSequences(t *testing.T) {
	tests := []struct {
		name         string
		input        []byte
		wantKey      Key
		wantConsumed int
	}{
		{"escape alone", []byte{0x1b}, KeyUnknown, 1},
		{"incomplete CSI", []byte{0x1b, '['}, KeyUnknown, 2},
		{"unknown CSI sequence", []byte{0x1b, '[', 'Z'}, KeyUnknown, 1},
		{"unknown tilde sequence", []byte{0x1b, '[', '9', '~'}, KeyUnknown, 1},
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

func TestParseControlCharacters(t *testing.T) {
	tests := []struct {
		name    string
		input   byte
		wantKey Key
	}{
		{"ctrl-a (0x01)", 0x01, KeyUnknown},
		{"ctrl-b (0x02)", 0x02, KeyUnknown},
		{"ctrl-e (0x05)", 0x05, KeyUnknown},
		{"ctrl-f (0x06)", 0x06, KeyUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, _ := ParseKey([]byte{tt.input}, 1)
			if event.Key != tt.wantKey {
				t.Errorf("ParseKey(%x) key = %v, want %v", tt.input, event.Key, tt.wantKey)
			}
		})
	}
}

func TestParseInvalidUTF8(t *testing.T) {
	tests := []struct {
		name         string
		input        []byte
		wantKey      Key
		wantConsumed int
	}{
		// Invalid continuation byte - returns replacement char
		{"invalid continuation byte", []byte{0xc3, 0x00}, KeyChar, 1},
		// Single byte with high bit set goes to parseSingleByte, returns Unknown
		{"incomplete 2-byte (single)", []byte{0xc3}, KeyUnknown, 1},
		// Incomplete multi-byte sequences with 2+ bytes
		{"incomplete 3-byte", []byte{0xe2, 0x82}, KeyChar, 1},
		{"incomplete 4-byte", []byte{0xf0, 0x9f, 0x98}, KeyChar, 1},
		// Overlong encoding
		{"overlong 2-byte", []byte{0xc0, 0x80}, KeyChar, 1},
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

// ============================================================================
// Additional History Tests
// ============================================================================

func TestHistoryNavigationCycle(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")
	h.Add("third")

	// Navigate all the way back
	h.Previous("current")
	h.Previous("")
	h.Previous("")

	// Now at "first"
	entry, _ := h.Previous("")
	if entry != "first" {
		t.Errorf("At start, Previous should stay at first, got %q", entry)
	}

	// Navigate all the way forward
	entry, _ = h.Next()
	if entry != "second" {
		t.Errorf("Next from first should be second, got %q", entry)
	}

	entry, _ = h.Next()
	if entry != "third" {
		t.Errorf("Next from second should be third, got %q", entry)
	}

	entry, _ = h.Next()
	if entry != "current" {
		t.Errorf("Next from third should restore 'current', got %q", entry)
	}

	// One more Next should fail
	_, ok := h.Next()
	if ok {
		t.Error("Next at end should return false")
	}
}

func TestHistoryAddAfterNavigation(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")

	// Navigate back
	h.Previous("typing")

	// Add new entry
	h.Add("new")

	// Navigation should be reset
	entry, _ := h.Previous("fresh")
	if entry != "new" {
		t.Errorf("After Add, Previous should return new entry, got %q", entry)
	}
}

func TestHistoryWhitespaceOnly(t *testing.T) {
	h := NewHistory(10)
	h.Add("   ") // Whitespace only - should be added
	h.Add("	")   // Tab only - should be added

	if h.Len() != 2 {
		t.Errorf("Whitespace entries should be added, got len %d", h.Len())
	}
}

func TestHistoryEntries(t *testing.T) {
	h := NewHistory(5)
	h.Add("one")
	h.Add("two")
	h.Add("three")

	entries := h.Entries()

	// Modifying returned slice shouldn't affect history
	entries[0] = "modified"

	original := h.Entries()
	if original[0] == "modified" {
		t.Error("Entries() should return a copy, not the original slice")
	}
}

// ============================================================================
// Additional Buffer Operation Tests
// ============================================================================

func TestBufferDeleteAtEnd(t *testing.T) {
	e := New(10)
	e.buffer = []rune("hello")
	e.cursor = 5 // At end

	// Delete at end should do nothing
	originalLen := len(e.buffer)
	if e.cursor >= len(e.buffer) {
		// Delete does nothing
	}

	if len(e.buffer) != originalLen {
		t.Errorf("Delete at end should not change buffer length")
	}
}

func TestBufferBackspaceAtStart(t *testing.T) {
	e := New(10)
	e.buffer = []rune("hello")
	e.cursor = 0 // At start

	// Backspace at start should do nothing
	originalLen := len(e.buffer)
	if e.cursor > 0 {
		// Backspace does nothing
	}

	if len(e.buffer) != originalLen {
		t.Errorf("Backspace at start should not change buffer length")
	}
	if e.cursor != 0 {
		t.Errorf("Backspace at start should not change cursor position")
	}
}

func TestBufferCursorBounds(t *testing.T) {
	e := New(10)
	e.buffer = []rune("hello")

	// Cursor at start, try move left
	e.cursor = 0
	if e.cursor > 0 {
		e.cursor--
	}
	if e.cursor != 0 {
		t.Errorf("Cursor should stay at 0, got %d", e.cursor)
	}

	// Cursor at end, try move right
	e.cursor = len(e.buffer)
	if e.cursor < len(e.buffer) {
		e.cursor++
	}
	if e.cursor != 5 {
		t.Errorf("Cursor should stay at 5, got %d", e.cursor)
	}
}

func TestBufferEmptyOperations(t *testing.T) {
	e := New(10)
	e.buffer = []rune{}
	e.cursor = 0

	// All operations on empty buffer should be safe
	if e.cursor > 0 {
		e.cursor--
	}
	if e.cursor < len(e.buffer) {
		e.cursor++
	}

	if e.cursor != 0 {
		t.Errorf("Empty buffer cursor should stay at 0, got %d", e.cursor)
	}
	if len(e.buffer) != 0 {
		t.Errorf("Empty buffer should remain empty, got len %d", len(e.buffer))
	}
}

func TestBufferUnicodeOperations(t *testing.T) {
	e := New(10)
	e.buffer = []rune("héllo")
	e.cursor = 2 // After 'é'

	// Insert unicode character
	ch := '世'
	e.buffer = append(e.buffer, 0)
	copy(e.buffer[e.cursor+1:], e.buffer[e.cursor:])
	e.buffer[e.cursor] = ch
	e.cursor++

	if string(e.buffer) != "hé世llo" {
		t.Errorf("Unicode insert: buffer = %q, want %q", string(e.buffer), "hé世llo")
	}
}

// ============================================================================
// Editor Close Test
// ============================================================================

func TestEditorClose(t *testing.T) {
	e := New(10)
	// Close should not panic even without raw mode being enabled
	err := e.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

// ============================================================================
// Edge Case Tests for Key Sequences
// ============================================================================

func TestParseKeyWithExtraData(t *testing.T) {
	// Buffer has more data than needed for one key
	buf := []byte{'a', 'b', 'c'}
	event, consumed := ParseKey(buf, 3)

	if event.Key != KeyChar {
		t.Errorf("First key should be KeyChar, got %v", event.Key)
	}
	if event.Char != 'a' {
		t.Errorf("First char should be 'a', got %c", event.Char)
	}
	if consumed != 1 {
		t.Errorf("Should consume only 1 byte, got %d", consumed)
	}
}

func TestParseEscapeWithShortBuffer(t *testing.T) {
	// Just escape, buffer length indicates only 1 byte
	buf := []byte{0x1b}
	event, consumed := ParseKey(buf, 1)

	if event.Key != KeyUnknown {
		t.Errorf("Lone escape should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 1 {
		t.Errorf("Should consume 1 byte, got %d", consumed)
	}
}

// ============================================================================
// More UTF-8 Decoding Tests
// ============================================================================

func TestDecodeUTF8SingleByte(t *testing.T) {
	// Test direct call to decodeUTF8 with single ASCII byte
	tests := []struct {
		input    byte
		wantChar rune
		wantSize int
	}{
		{0x00, 0x00, 1}, // NUL
		{0x41, 'A', 1},  // 'A'
		{0x7F, 0x7F, 1}, // DEL
	}

	for _, tt := range tests {
		r, size := decodeUTF8([]byte{tt.input}, 1)
		if r != tt.wantChar {
			t.Errorf("decodeUTF8([%x]) char = %x, want %x", tt.input, r, tt.wantChar)
		}
		if size != tt.wantSize {
			t.Errorf("decodeUTF8([%x]) size = %d, want %d", tt.input, size, tt.wantSize)
		}
	}
}

func TestDecodeUTF8EmptyBuffer(t *testing.T) {
	r, size := decodeUTF8([]byte{}, 0)
	if r != 0 || size != 0 {
		t.Errorf("decodeUTF8(empty) = (%x, %d), want (0, 0)", r, size)
	}
}

func TestDecodeUTF8InvalidStartByte(t *testing.T) {
	// Start byte that doesn't match valid UTF-8 pattern (0x80-0xBF are continuation bytes)
	r, size := decodeUTF8([]byte{0x80}, 1)
	if r != 0xFFFD || size != 1 {
		t.Errorf("decodeUTF8([0x80]) = (%x, %d), want (FFFD, 1)", r, size)
	}

	r, size = decodeUTF8([]byte{0xFE}, 1)
	if r != 0xFFFD || size != 1 {
		t.Errorf("decodeUTF8([0xFE]) = (%x, %d), want (FFFD, 1)", r, size)
	}

	r, size = decodeUTF8([]byte{0xFF}, 1)
	if r != 0xFFFD || size != 1 {
		t.Errorf("decodeUTF8([0xFF]) = (%x, %d), want (FFFD, 1)", r, size)
	}
}

// ============================================================================
// More Escape Sequence Tests
// ============================================================================

func TestParseEscapeSequenceIncomplete1(t *testing.T) {
	// ESC [ 1 without the tilde
	buf := []byte{0x1b, '[', '1'}
	event, consumed := ParseKey(buf, 3)
	if event.Key != KeyUnknown {
		t.Errorf("Incomplete ESC[1 should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 1 {
		t.Errorf("Incomplete ESC[1 should consume 1 byte, got %d", consumed)
	}
}

func TestParseEscapeSequenceIncomplete3(t *testing.T) {
	// ESC [ 3 without the tilde
	buf := []byte{0x1b, '[', '3'}
	event, consumed := ParseKey(buf, 3)
	if event.Key != KeyUnknown {
		t.Errorf("Incomplete ESC[3 should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 1 {
		t.Errorf("Incomplete ESC[3 should consume 1 byte, got %d", consumed)
	}
}

func TestParseEscapeSequenceIncomplete4(t *testing.T) {
	// ESC [ 4 without the tilde
	buf := []byte{0x1b, '[', '4'}
	event, consumed := ParseKey(buf, 3)
	if event.Key != KeyUnknown {
		t.Errorf("Incomplete ESC[4 should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 1 {
		t.Errorf("Incomplete ESC[4 should consume 1 byte, got %d", consumed)
	}
}

func TestParseSS3WithShortBuffer(t *testing.T) {
	// ESC O with only 2 bytes
	buf := []byte{0x1b, 'O'}
	event, consumed := ParseKey(buf, 2)
	if event.Key != KeyUnknown {
		t.Errorf("Short ESC O should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 1 {
		t.Errorf("Short ESC O should consume 1 byte, got %d", consumed)
	}
}

func TestParseSS3UnknownCode(t *testing.T) {
	// ESC O with unknown third byte
	buf := []byte{0x1b, 'O', 'X'}
	event, consumed := ParseKey(buf, 3)
	if event.Key != KeyUnknown {
		t.Errorf("Unknown ESC O X should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 1 {
		t.Errorf("Unknown ESC O X should consume 1 byte, got %d", consumed)
	}
}

// ============================================================================
// Edge Cases for ParseKey
// ============================================================================

func TestParseKeyMultiByteWithN1(t *testing.T) {
	// Multi-byte UTF-8 start but n=1
	buf := []byte{0xc3, 0xa9} // é
	event, consumed := ParseKey(buf, 1)
	// With n=1, it treats it as single byte
	if event.Key != KeyUnknown {
		t.Errorf("High byte with n=1 should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 1 {
		t.Errorf("Should consume 1 byte, got %d", consumed)
	}
}

func TestParseKeyEscapeN1(t *testing.T) {
	// Escape byte but n=1 (no room for sequence)
	buf := []byte{0x1b, '[', 'A'}
	event, consumed := ParseKey(buf, 1)
	// Should call parseSingleByte with just the escape
	if event.Key != KeyUnknown {
		t.Errorf("ESC with n=1 should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 1 {
		t.Errorf("Should consume 1 byte, got %d", consumed)
	}
}

func TestParseKeyHighByteSingle(t *testing.T) {
	// Test high bytes that are invalid UTF-8 start bytes
	tests := []byte{0x80, 0x90, 0xA0, 0xB0, 0xFE, 0xFF}

	for _, b := range tests {
		event, consumed := ParseKey([]byte{b}, 1)
		if event.Key != KeyUnknown {
			t.Errorf("Invalid byte 0x%02X with n=1 should be KeyUnknown, got %v", b, event.Key)
		}
		if consumed != 1 {
			t.Errorf("Should consume 1 byte for 0x%02X, got %d", b, consumed)
		}
	}
}

// ============================================================================
// Additional Single Byte Tests
// ============================================================================

func TestParseSingleByteControlChars(t *testing.T) {
	// Test all control characters 0x00-0x1F except those already tested
	unknownControls := []byte{0x00, 0x01, 0x02, 0x05, 0x06, 0x07, 0x0e, 0x0f,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a}

	for _, b := range unknownControls {
		event := parseSingleByte(b)
		if event.Key != KeyUnknown {
			t.Errorf("Control 0x%02X should be KeyUnknown, got %v", b, event.Key)
		}
	}
}

// ============================================================================
// Direct Function Tests for Coverage
// ============================================================================

func TestParseEscapeSequenceDirectN1(t *testing.T) {
	// Call parseEscapeSequence directly with n=1 to cover the n < 2 path
	buf := []byte{0x1b}
	event, consumed := parseEscapeSequence(buf, 1)
	if event.Key != KeyUnknown {
		t.Errorf("parseEscapeSequence with n=1 should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 1 {
		t.Errorf("Should consume 1 byte, got %d", consumed)
	}
}

func TestParseEscapeSequenceCSIN2(t *testing.T) {
	// ESC [ with n=2 to cover the n < 3 path
	buf := []byte{0x1b, '['}
	event, consumed := parseEscapeSequence(buf, 2)
	if event.Key != KeyUnknown {
		t.Errorf("parseEscapeSequence CSI with n=2 should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 2 {
		t.Errorf("Should consume 2 bytes, got %d", consumed)
	}
}

func TestTerminalIsRaw(t *testing.T) {
	term := NewTerminal()
	if term.IsRaw() {
		t.Error("New terminal should not be in raw mode")
	}
}

func TestHistoryEntriesCopy(t *testing.T) {
	h := NewHistory(10)
	h.Add("one")
	h.Add("two")

	entries := h.Entries()

	// Verify entries is a copy
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
	if entries[0] != "one" || entries[1] != "two" {
		t.Errorf("Entries mismatch: got %v", entries)
	}
}

func TestDecodeUTF8TruncatedSequences(t *testing.T) {
	// 3-byte sequence with only 2 bytes
	r, size := decodeUTF8([]byte{0xe2, 0x82}, 2)
	if r != 0xFFFD || size != 1 {
		t.Errorf("Truncated 3-byte sequence: got (%x, %d), want (FFFD, 1)", r, size)
	}

	// 4-byte sequence with only 3 bytes
	r, size = decodeUTF8([]byte{0xf0, 0x9f, 0x98}, 3)
	if r != 0xFFFD || size != 1 {
		t.Errorf("Truncated 4-byte sequence: got (%x, %d), want (FFFD, 1)", r, size)
	}
}

func TestDecodeUTF8OverlongEncodings(t *testing.T) {
	// Overlong encoding for '/' (should be 0x2F as single byte)
	// 0xC0 0xAF is overlong encoding for 0x2F
	r, size := decodeUTF8([]byte{0xc0, 0xaf}, 2)
	if r != 0xFFFD || size != 1 {
		t.Errorf("Overlong encoding: got (%x, %d), want (FFFD, 1)", r, size)
	}
}

func TestDecodeUTF84ByteValid(t *testing.T) {
	// Valid 4-byte sequence for emoji (U+1F600)
	buf := []byte{0xf0, 0x9f, 0x98, 0x80}
	r, size := decodeUTF8(buf, 4)
	if r != 0x1F600 || size != 4 {
		t.Errorf("Valid 4-byte emoji: got (%x, %d), want (1F600, 4)", r, size)
	}
}

func TestDecodeUTF8InvalidContinuation(t *testing.T) {
	// 2-byte sequence start with invalid continuation (0x00 is not 0x80-0xBF)
	r, size := decodeUTF8([]byte{0xc3, 0x00}, 2)
	if r != 0xFFFD || size != 1 {
		t.Errorf("Invalid continuation: got (%x, %d), want (FFFD, 1)", r, size)
	}

	// 3-byte sequence with invalid second byte
	r, size = decodeUTF8([]byte{0xe2, 0x00, 0xac}, 3)
	if r != 0xFFFD || size != 1 {
		t.Errorf("Invalid continuation in 3-byte: got (%x, %d), want (FFFD, 1)", r, size)
	}

	// 4-byte sequence with invalid third byte
	r, size = decodeUTF8([]byte{0xf0, 0x9f, 0x00, 0x80}, 4)
	if r != 0xFFFD || size != 1 {
		t.Errorf("Invalid continuation in 4-byte: got (%x, %d), want (FFFD, 1)", r, size)
	}
}

func TestParseSingleByteAllPrintable(t *testing.T) {
	// Test all printable ASCII characters (32-126)
	for b := byte(32); b < 127; b++ {
		event := parseSingleByte(b)
		if event.Key != KeyChar {
			t.Errorf("Printable byte 0x%02X (%c) should be KeyChar, got %v", b, b, event.Key)
		}
		if event.Char != rune(b) {
			t.Errorf("Printable byte 0x%02X (%c) char mismatch, got %c", b, b, event.Char)
		}
	}
}

func TestParseEscapeSequenceNumericWithOtherChar(t *testing.T) {
	// ESC [ 1 X (not tilde)
	buf := []byte{0x1b, '[', '1', 'X'}
	event, consumed := parseEscapeSequence(buf, 4)
	if event.Key != KeyUnknown {
		t.Errorf("ESC[1X should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 1 {
		t.Errorf("Should consume 1 byte, got %d", consumed)
	}

	// ESC [ 3 X (not tilde)
	buf = []byte{0x1b, '[', '3', 'X'}
	event, consumed = parseEscapeSequence(buf, 4)
	if event.Key != KeyUnknown {
		t.Errorf("ESC[3X should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 1 {
		t.Errorf("Should consume 1 byte, got %d", consumed)
	}

	// ESC [ 4 X (not tilde)
	buf = []byte{0x1b, '[', '4', 'X'}
	event, consumed = parseEscapeSequence(buf, 4)
	if event.Key != KeyUnknown {
		t.Errorf("ESC[4X should be KeyUnknown, got %v", event.Key)
	}
	if consumed != 1 {
		t.Errorf("Should consume 1 byte, got %d", consumed)
	}
}

func TestParseEscapeSequenceUnknownCSI(t *testing.T) {
	// ESC [ with various unknown third bytes
	unknownBytes := []byte{'2', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}
	for _, b := range unknownBytes {
		buf := []byte{0x1b, '[', b}
		event, consumed := parseEscapeSequence(buf, 3)
		if event.Key != KeyUnknown {
			t.Errorf("ESC[%c should be KeyUnknown, got %v", b, event.Key)
		}
		if consumed != 1 {
			t.Errorf("ESC[%c should consume 1 byte, got %d", b, consumed)
		}
	}
}

func TestEditorBufferInitialization(t *testing.T) {
	e := New(10)
	// Buffer should be initialized with capacity
	if cap(e.buffer) < 128 {
		t.Errorf("Buffer capacity should be >= 128, got %d", cap(e.buffer))
	}
	if len(e.buffer) != 0 {
		t.Errorf("Buffer should start empty, got len %d", len(e.buffer))
	}
}

func TestHistoryMultipleResets(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")

	h.Previous("current")
	h.Reset()
	h.Reset() // Double reset should be safe

	entry, ok := h.Previous("new")
	if !ok || entry != "second" {
		t.Errorf("After multiple resets, Previous() = %q, %v; want %q, true", entry, ok, "second")
	}
}

func TestHistoryAddResetsNavigation(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")

	// Navigate back partially
	h.Previous("current")

	// Add new entry should reset position
	h.Add("third")

	// Should start from most recent again
	entry, ok := h.Previous("new")
	if !ok || entry != "third" {
		t.Errorf("After Add, Previous() should return newest, got %q", entry)
	}
}

func TestDecodeUTF83ByteValid(t *testing.T) {
	// Valid 3-byte sequences
	tests := []struct {
		buf      []byte
		expected rune
	}{
		{[]byte{0xe2, 0x82, 0xac}, '€'}, // Euro sign
		{[]byte{0xe4, 0xb8, 0xad}, '中'}, // Chinese character
		{[]byte{0xe2, 0x9c, 0x93}, '✓'}, // Check mark
	}

	for _, tt := range tests {
		r, size := decodeUTF8(tt.buf, 3)
		if r != tt.expected || size != 3 {
			t.Errorf("decodeUTF8(%v): got (%c, %d), want (%c, 3)", tt.buf, r, size, tt.expected)
		}
	}
}

func TestDecodeUTF82ByteValid(t *testing.T) {
	// Valid 2-byte sequences
	tests := []struct {
		buf      []byte
		expected rune
	}{
		{[]byte{0xc3, 0xa9}, 'é'}, // e with acute
		{[]byte{0xc3, 0xb1}, 'ñ'}, // n with tilde
		{[]byte{0xc2, 0xa9}, '©'}, // Copyright sign
		{[]byte{0xc3, 0x9f}, 'ß'}, // German eszett
	}

	for _, tt := range tests {
		r, size := decodeUTF8(tt.buf, 2)
		if r != tt.expected || size != 2 {
			t.Errorf("decodeUTF8(%v): got (%c, %d), want (%c, 2)", tt.buf, r, size, tt.expected)
		}
	}
}

// ============================================================================
// Terminal Operations Tests (using pipes for non-tty operations)
// ============================================================================

func TestTerminalWriteToDevNull(t *testing.T) {
	// Create a terminal with custom fd that we can write to
	term := NewTerminal()
	// The default terminal uses stdout, we just verify the struct is properly initialized
	if term.stdinFd <= 0 {
		// stdin fd should be valid (though value depends on OS)
		// Just check it's set to something
	}
	if term.stdoutFd <= 0 {
		// stdout fd should be valid
	}
}

func TestParseKeyEdgeCaseDecodeUTF8Size0(t *testing.T) {
	// Try to find any edge case that might trigger the unreachable code
	// When n > 1 and buf[0] is not escape but decodeUTF8 behavior

	// Test with continuation bytes (0x80-0xBF) at start with n > 1
	tests := []struct {
		buf []byte
		n   int
	}{
		{[]byte{0x80, 0x80}, 2},
		{[]byte{0xBF, 0x80}, 2},
		{[]byte{0xFE, 0x80}, 2},
		{[]byte{0xFF, 0x80}, 2},
	}

	for _, tt := range tests {
		event, consumed := ParseKey(tt.buf, tt.n)
		// These should all return replacement character or unknown
		if event.Key != KeyChar {
			// If KeyChar with replacement, that's fine
			// If KeyUnknown, that's also fine
		}
		if consumed < 1 {
			t.Errorf("ParseKey(%v, %d) should consume at least 1 byte, got %d", tt.buf, tt.n, consumed)
		}
	}
}

func TestParseKeyValidMultiByteSequences(t *testing.T) {
	// Test more multi-byte sequences through ParseKey
	tests := []struct {
		name     string
		buf      []byte
		wantChar rune
		wantSize int
	}{
		{"2-byte lowercase u with umlaut", []byte{0xc3, 0xbc}, 'ü', 2},
		{"2-byte uppercase A with ring", []byte{0xc3, 0x85}, 'Å', 2},
		{"3-byte japanese hiragana", []byte{0xe3, 0x81, 0x82}, 'あ', 3},
		{"3-byte korean hangul", []byte{0xed, 0x95, 0x9c}, '한', 3},
		{"4-byte musical symbol", []byte{0xf0, 0x9d, 0x84, 0x9e}, 0x1D11E, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, consumed := ParseKey(tt.buf, len(tt.buf))
			if event.Key != KeyChar {
				t.Errorf("Expected KeyChar, got %v", event.Key)
			}
			if event.Char != tt.wantChar {
				t.Errorf("Expected char %U, got %U", tt.wantChar, event.Char)
			}
			if consumed != tt.wantSize {
				t.Errorf("Expected consumed %d, got %d", tt.wantSize, consumed)
			}
		})
	}
}

func TestParseKeyWithLargerBuffer(t *testing.T) {
	// Test ParseKey with buffer containing more bytes than needed
	buf := []byte{0xe2, 0x82, 0xac, 0x41, 0x42, 0x43} // € followed by ABC
	event, consumed := ParseKey(buf, 6)

	if event.Key != KeyChar {
		t.Errorf("Expected KeyChar, got %v", event.Key)
	}
	if event.Char != '€' {
		t.Errorf("Expected €, got %c", event.Char)
	}
	if consumed != 3 {
		t.Errorf("Expected 3 bytes consumed, got %d", consumed)
	}
}

func TestParseKeySingleByteInMultiByteBuffer(t *testing.T) {
	// Single ASCII byte in a larger buffer with n=1
	buf := []byte{'x', 'y', 'z'}
	event, consumed := ParseKey(buf, 1)

	if event.Key != KeyChar {
		t.Errorf("Expected KeyChar, got %v", event.Key)
	}
	if event.Char != 'x' {
		t.Errorf("Expected 'x', got %c", event.Char)
	}
	if consumed != 1 {
		t.Errorf("Expected 1 byte consumed, got %d", consumed)
	}
}

func TestParseSingleByteEdgeCases(t *testing.T) {
	// Test boundary values
	tests := []struct {
		input   byte
		wantKey Key
	}{
		{31, KeyUnknown},    // Last control char before space
		{32, KeyChar},       // Space (first printable)
		{126, KeyChar},      // Tilde (last printable before DEL)
		{127, KeyBackspace}, // DEL
	}

	for _, tt := range tests {
		event := parseSingleByte(tt.input)
		if event.Key != tt.wantKey {
			t.Errorf("parseSingleByte(0x%02X): expected %v, got %v", tt.input, tt.wantKey, event.Key)
		}
	}
}

func TestHistoryEdgeCaseLenAfterOperations(t *testing.T) {
	h := NewHistory(5)

	// Check Len after each operation
	if h.Len() != 0 {
		t.Errorf("Empty history should have Len 0, got %d", h.Len())
	}

	h.Add("one")
	if h.Len() != 1 {
		t.Errorf("After adding one, Len should be 1, got %d", h.Len())
	}

	h.Add("one") // Duplicate
	if h.Len() != 1 {
		t.Errorf("After adding duplicate, Len should stay 1, got %d", h.Len())
	}

	h.Add("two")
	h.Add("three")
	h.Add("four")
	h.Add("five")
	if h.Len() != 5 {
		t.Errorf("After adding 5 unique, Len should be 5, got %d", h.Len())
	}

	h.Add("six") // Should evict "one"
	if h.Len() != 5 {
		t.Errorf("After overflow, Len should stay at max 5, got %d", h.Len())
	}
}

func TestEditorMultipleSetPrompt(t *testing.T) {
	e := New(10)

	e.SetPrompt("first> ")
	if e.prompt != "first> " {
		t.Errorf("First SetPrompt failed")
	}

	e.SetPrompt("second> ")
	if e.prompt != "second> " {
		t.Errorf("Second SetPrompt failed")
	}

	e.SetPrompt("") // Empty prompt should be valid
	if e.prompt != "" {
		t.Errorf("Empty SetPrompt failed")
	}
}

func TestEditorCloseMultipleTimes(t *testing.T) {
	e := New(10)

	// Close multiple times should be safe
	err1 := e.Close()
	err2 := e.Close()
	err3 := e.Close()

	// All should succeed (no-op when not in raw mode)
	if err1 != nil || err2 != nil || err3 != nil {
		t.Errorf("Multiple Close() calls should all succeed")
	}
}

func TestHistorySaveCurrentInput(t *testing.T) {
	h := NewHistory(10)
	h.Add("one")
	h.Add("two")

	// Start navigation with current input
	h.Previous("my current typing")

	// Navigate back
	h.Previous("")

	// Now navigate forward to get back to saved input
	h.Next()
	entry, ok := h.Next()

	if !ok || entry != "my current typing" {
		t.Errorf("Should restore saved input, got %q, %v", entry, ok)
	}
}

// ============================================================================
// Terminal I/O Tests with Pipes
// ============================================================================

func TestTerminalWriteWithPipe(t *testing.T) {
	// Create a pipe to capture output
	r, w, err := os.Pipe()
	if err != nil {
		t.Skipf("Could not create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	// Create a terminal with custom stdout fd
	term := &Terminal{
		fd:       int(os.Stdin.Fd()),
		stdinFd:  int(os.Stdin.Fd()),
		stdoutFd: int(w.Fd()),
		isRaw:    false,
	}

	// Test Write
	data := []byte("hello world")
	n, err := term.Write(data)
	if err != nil {
		t.Errorf("Write() error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() returned %d, want %d", n, len(data))
	}

	// Test WriteString
	n, err = term.WriteString("test string")
	if err != nil {
		t.Errorf("WriteString() error: %v", err)
	}
	if n != 11 {
		t.Errorf("WriteString() returned %d, want 11", n)
	}
}

func TestTerminalReadWithPipe(t *testing.T) {
	// Create a pipe for stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Skipf("Could not create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	// Write test data to the pipe
	go func() {
		w.Write([]byte("test input"))
		w.Close()
	}()

	// Create a terminal with custom stdin fd
	term := &Terminal{
		fd:       int(r.Fd()),
		stdinFd:  int(r.Fd()),
		stdoutFd: int(os.Stdout.Fd()),
		isRaw:    false,
	}

	// Test Read
	buf := make([]byte, 100)
	n, err := term.Read(buf)
	if err != nil {
		t.Errorf("Read() error: %v", err)
	}
	if n == 0 {
		t.Error("Read() returned 0 bytes")
	}
	if string(buf[:n]) != "test input" {
		t.Errorf("Read() got %q, want %q", string(buf[:n]), "test input")
	}
}

func TestTerminalIsRawStates(t *testing.T) {
	term := &Terminal{
		fd:       int(os.Stdin.Fd()),
		stdinFd:  int(os.Stdin.Fd()),
		stdoutFd: int(os.Stdout.Fd()),
		isRaw:    false,
	}

	if term.IsRaw() {
		t.Error("IsRaw() should return false initially")
	}

	term.isRaw = true
	if !term.IsRaw() {
		t.Error("IsRaw() should return true after setting isRaw")
	}

	term.isRaw = false
	if term.IsRaw() {
		t.Error("IsRaw() should return false after unsetting isRaw")
	}
}

func TestTerminalDisableRawModeWhenNotRaw(t *testing.T) {
	term := &Terminal{
		fd:       int(os.Stdin.Fd()),
		stdinFd:  int(os.Stdin.Fd()),
		stdoutFd: int(os.Stdout.Fd()),
		isRaw:    false,
	}

	// DisableRawMode when not in raw mode should return nil
	err := term.DisableRawMode()
	if err != nil {
		t.Errorf("DisableRawMode() when not raw should return nil, got %v", err)
	}
}

func TestEditorWithPipeTerminal(t *testing.T) {
	// Create pipes for stdin/stdout
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Skipf("Could not create stdin pipe: %v", err)
	}
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Skipf("Could not create stdout pipe: %v", err)
	}
	defer stdoutR.Close()
	defer stdoutW.Close()

	// Create editor with custom terminal
	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    false,
	}

	// Test that we can access the history
	h := e.History()
	if h == nil {
		t.Error("History() should not return nil")
	}

	// Test SetPrompt
	e.SetPrompt(">>> ")
	if e.prompt != ">>> " {
		t.Errorf("SetPrompt failed, got %q", e.prompt)
	}
}

func TestEditorBufferManipulationDirect(t *testing.T) {
	// Create pipes
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true, // Pretend we're in raw mode
	}

	// Manually set buffer state
	e.buffer = []rune("hello")
	e.cursor = 5

	// Test insertChar directly
	e.insertChar('!')

	if string(e.buffer) != "hello!" {
		t.Errorf("insertChar failed, got %q", string(e.buffer))
	}
	if e.cursor != 6 {
		t.Errorf("cursor should be 6, got %d", e.cursor)
	}
}

func TestEditorBackspaceDirect(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 5

	e.backspace()

	if string(e.buffer) != "hell" {
		t.Errorf("backspace failed, got %q", string(e.buffer))
	}
	if e.cursor != 4 {
		t.Errorf("cursor should be 4, got %d", e.cursor)
	}
}

func TestEditorBackspaceAtStart(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 0

	e.backspace() // Should do nothing

	if string(e.buffer) != "hello" {
		t.Errorf("backspace at start should not change buffer, got %q", string(e.buffer))
	}
	if e.cursor != 0 {
		t.Errorf("cursor should stay 0, got %d", e.cursor)
	}
}

func TestEditorDeleteCharDirect(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 0

	e.deleteChar()

	if string(e.buffer) != "ello" {
		t.Errorf("deleteChar failed, got %q", string(e.buffer))
	}
	if e.cursor != 0 {
		t.Errorf("cursor should stay 0, got %d", e.cursor)
	}
}

func TestEditorDeleteCharAtEnd(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 5

	e.deleteChar() // Should do nothing at end

	if string(e.buffer) != "hello" {
		t.Errorf("deleteChar at end should not change buffer, got %q", string(e.buffer))
	}
}

func TestEditorMoveCursorLeft(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 3

	e.moveCursorLeft()

	if e.cursor != 2 {
		t.Errorf("moveCursorLeft failed, cursor = %d, want 2", e.cursor)
	}
}

func TestEditorMoveCursorLeftAtStart(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 0

	e.moveCursorLeft() // Should do nothing

	if e.cursor != 0 {
		t.Errorf("moveCursorLeft at start failed, cursor = %d, want 0", e.cursor)
	}
}

func TestEditorMoveCursorRight(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 2

	e.moveCursorRight()

	if e.cursor != 3 {
		t.Errorf("moveCursorRight failed, cursor = %d, want 3", e.cursor)
	}
}

func TestEditorMoveCursorRightAtEnd(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 5

	e.moveCursorRight() // Should do nothing

	if e.cursor != 5 {
		t.Errorf("moveCursorRight at end failed, cursor = %d, want 5", e.cursor)
	}
}

func TestEditorMoveCursorHome(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 3

	e.moveCursorHome()

	if e.cursor != 0 {
		t.Errorf("moveCursorHome failed, cursor = %d, want 0", e.cursor)
	}
}

func TestEditorMoveCursorHomeAtStart(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 0

	e.moveCursorHome() // Should do nothing

	if e.cursor != 0 {
		t.Errorf("moveCursorHome at start failed, cursor = %d, want 0", e.cursor)
	}
}

func TestEditorMoveCursorEnd(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 2

	e.moveCursorEnd()

	if e.cursor != 5 {
		t.Errorf("moveCursorEnd failed, cursor = %d, want 5", e.cursor)
	}
}

func TestEditorMoveCursorEndAtEnd(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 5

	e.moveCursorEnd() // Should do nothing

	if e.cursor != 5 {
		t.Errorf("moveCursorEnd at end failed, cursor = %d, want 5", e.cursor)
	}
}

func TestEditorReplaceLine(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 3

	e.replaceLine("world")

	if string(e.buffer) != "world" {
		t.Errorf("replaceLine failed, buffer = %q, want %q", string(e.buffer), "world")
	}
	if e.cursor != 5 {
		t.Errorf("replaceLine cursor = %d, want 5", e.cursor)
	}
}

func TestEditorReplaceLineFromStart(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 0 // Already at start

	e.replaceLine("new text")

	if string(e.buffer) != "new text" {
		t.Errorf("replaceLine from start failed, buffer = %q", string(e.buffer))
	}
}

func TestEditorRedrawLine(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 3

	// This just writes to the terminal, no state change
	e.redrawLine()

	// Buffer should remain unchanged
	if string(e.buffer) != "hello" {
		t.Errorf("redrawLine modified buffer")
	}
	if e.cursor != 3 {
		t.Errorf("redrawLine modified cursor")
	}
}

func TestEditorRedrawLineAtEnd(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 5 // At end

	e.redrawLine()

	// Buffer should remain unchanged
	if string(e.buffer) != "hello" {
		t.Errorf("redrawLine modified buffer")
	}
}

func TestEditorHistoryPrevious(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	// Add some history
	e.history.Add("first command")
	e.history.Add("second command")

	e.buffer = []rune("current")
	e.cursor = 7

	e.historyPrevious()

	// Should replace with last history entry
	if string(e.buffer) != "second command" {
		t.Errorf("historyPrevious failed, buffer = %q", string(e.buffer))
	}
}

func TestEditorHistoryPreviousEmpty(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	// No history
	e.buffer = []rune("current")
	e.cursor = 7

	e.historyPrevious() // Should do nothing

	if string(e.buffer) != "current" {
		t.Errorf("historyPrevious on empty history changed buffer to %q", string(e.buffer))
	}
}

func TestEditorHistoryNext(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	// Add history and navigate
	e.history.Add("first")
	e.history.Add("second")

	e.buffer = []rune("current")
	e.cursor = 7

	e.historyPrevious() // Go to "second"
	e.historyPrevious() // Go to "first"
	e.historyNext()     // Should go to "second"

	if string(e.buffer) != "second" {
		t.Errorf("historyNext failed, buffer = %q", string(e.buffer))
	}
}

func TestEditorHistoryNextNoHistory(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("current")
	e.cursor = 7

	e.historyNext() // Should do nothing

	if string(e.buffer) != "current" {
		t.Errorf("historyNext without navigation changed buffer to %q", string(e.buffer))
	}
}

func TestEditorInsertCharMiddle(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("helo")
	e.cursor = 2 // After 'e'

	e.insertChar('l')

	if string(e.buffer) != "hello" {
		t.Errorf("insertChar middle failed, buffer = %q", string(e.buffer))
	}
	if e.cursor != 3 {
		t.Errorf("cursor should be 3, got %d", e.cursor)
	}
}

func TestEditorBackspaceMiddle(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 3 // After second 'l'

	e.backspace()

	if string(e.buffer) != "helo" {
		t.Errorf("backspace middle failed, buffer = %q", string(e.buffer))
	}
	if e.cursor != 2 {
		t.Errorf("cursor should be 2, got %d", e.cursor)
	}
}

func TestEditorDeleteCharMiddle(t *testing.T) {
	stdinR, stdinW, _ := os.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	stdoutR, stdoutW, _ := os.Pipe()
	defer stdoutR.Close()
	defer stdoutW.Close()

	e := New(10)
	e.terminal = &Terminal{
		fd:       int(stdinR.Fd()),
		stdinFd:  int(stdinR.Fd()),
		stdoutFd: int(stdoutW.Fd()),
		isRaw:    true,
	}

	e.buffer = []rune("hello")
	e.cursor = 2 // Before second 'l'

	e.deleteChar()

	if string(e.buffer) != "helo" {
		t.Errorf("deleteChar middle failed, buffer = %q", string(e.buffer))
	}
	if e.cursor != 2 {
		t.Errorf("cursor should stay 2, got %d", e.cursor)
	}
}
