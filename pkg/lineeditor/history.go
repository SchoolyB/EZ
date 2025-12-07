package lineeditor

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

// History manages command history for the line editor
type History struct {
	entries []string
	index   int // Current position in history (-1 means not browsing)
	maxSize int
	current string // Stores current input when browsing history
}

// NewHistory creates a new History with the given maximum size
func NewHistory(maxSize int) *History {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &History{
		entries: make([]string, 0, maxSize),
		index:   -1,
		maxSize: maxSize,
	}
}

// Add adds a new entry to the history
// Empty strings and duplicates of the last entry are ignored
func (h *History) Add(entry string) {
	if entry == "" {
		return
	}

	// Don't add duplicates of the last entry
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == entry {
		return
	}

	h.entries = append(h.entries, entry)

	// Trim if exceeds max size
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[1:]
	}

	h.Reset()
}

// Previous returns the previous history entry
// Returns the current entry and true if available, or empty string and false if at start
func (h *History) Previous(currentInput string) (string, bool) {
	if len(h.entries) == 0 {
		return "", false
	}

	// Save current input when starting to browse
	if h.index == -1 {
		h.current = currentInput
		h.index = len(h.entries)
	}

	if h.index > 0 {
		h.index--
		return h.entries[h.index], true
	}

	return h.entries[0], true
}

// Next returns the next history entry (more recent)
// Returns the entry and true if available, or the saved current input if at end
func (h *History) Next() (string, bool) {
	if h.index == -1 {
		return "", false
	}

	h.index++

	if h.index >= len(h.entries) {
		// Restore the original input
		h.index = -1
		return h.current, true
	}

	return h.entries[h.index], true
}

// Reset resets the browsing position
func (h *History) Reset() {
	h.index = -1
	h.current = ""
}

// Len returns the number of entries in history
func (h *History) Len() int {
	return len(h.entries)
}

// Entries returns a copy of all history entries
func (h *History) Entries() []string {
	result := make([]string, len(h.entries))
	copy(result, h.entries)
	return result
}

// Clear removes all history entries
func (h *History) Clear() {
	h.entries = h.entries[:0]
	h.Reset()
}
