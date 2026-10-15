// fmt.go — Source formatter for .gray files ("gray fmt"). Normalizes
// indentation, trailing whitespace, blank-line runs, and EOF newlines,
// with a --check mode for CI gating.
//
// Author:  Marshall A Burns (@SchoolyB)
// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.
//
// Contributors:
//  - @mvanhorn

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/grayscale-lang/grayscale/internal/driver"
)

const (
	tabWidth                 = 4
	maxConsecutiveBlankLines = 2
)

// formatGraySource applies text-level normalizations to Grayscale source bytes:
//   - trailing whitespace stripped from each line
//   - leading tabs expanded to 4 spaces each
//   - runs of more than 2 consecutive blank lines collapsed to 2
//   - exactly one trailing newline at EOF
func formatGraySource(src []byte) []byte {
	lines := strings.Split(string(src), "\n")
	startsInRaw, _ := multiLineLiteralLineStarts(string(src), len(lines))

	// Expand leading tabs and strip trailing whitespace on each line. A line
	// that starts inside a raw string keeps its leading whitespace, and one
	// that ends inside a raw string keeps its trailing whitespace, since both
	// belong to the literal's value.
	for i, line := range lines {
		endsInRaw := i+1 < len(startsInRaw) && startsInRaw[i+1]
		j := 0
		var prefix strings.Builder
		if !startsInRaw[i] {
			// Expand leading tabs: count leading tab/space mix, replace tabs with 4 spaces.
			for j < len(line) && (line[j] == '\t' || line[j] == ' ') {
				if line[j] == '\t' {
					prefix.WriteString(strings.Repeat(" ", tabWidth))
				} else {
					prefix.WriteByte(' ')
				}
				j++
			}
		}
		rest := line[j:]
		if !endsInRaw {
			rest = strings.TrimRight(rest, " \t")
		}
		lines[i] = prefix.String() + rest
	}

	// Collapse runs of more than 2 consecutive blank lines.
	var out []string
	blank := 0
	for i, line := range lines {
		if line == "" && !startsInRaw[i] {
			blank++
			if blank <= maxConsecutiveBlankLines {
				out = append(out, line)
			}
		} else {
			blank = 0
			out = append(out, line)
		}
	}

	// Strip trailing blank lines, then add exactly one trailing newline.
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return []byte(strings.Join(out, "\n") + "\n")
}

// multiLineLiteralLineStarts reports, for each line of src, whether the line
// begins inside a multi-line raw string literal and whether it begins inside
// a block comment. Backticks inside comments, strings, and char literals do
// not open a raw string.
func multiLineLiteralLineStarts(src string, lineCount int) (inRawString, inBlockComment []bool) {
	const (
		inCode = iota
		inLineComment
		inBlockCommentState
		inString
		inChar
		inRaw
	)
	starts := make([]bool, 0, lineCount)
	starts = append(starts, false)
	comments := make([]bool, 0, lineCount)
	comments = append(comments, false)
	state := inCode
	interpolationDepth := 0
	var nestedQuote byte // a string or char literal inside an interpolation
	for i := 0; i < len(src); i++ {
		c := src[i]
		if c == '\n' {
			if state == inLineComment {
				state = inCode
			}
			starts = append(starts, state == inRaw)
			comments = append(comments, state == inBlockCommentState)
			continue
		}
		switch state {
		case inCode:
			switch {
			case c == '/' && i+1 < len(src) && src[i+1] == '/':
				state = inLineComment
				i++
			case c == '/' && i+1 < len(src) && src[i+1] == '*':
				state = inBlockCommentState
				i++
			case c == '"':
				state = inString
				interpolationDepth = 0
				nestedQuote = 0
			case c == '\'':
				state = inChar
			case c == '`':
				state = inRaw
			}
		case inBlockCommentState:
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				state = inCode
				i++
			}
		case inChar:
			if c == '\\' && i+1 < len(src) && src[i+1] != '\n' {
				i++
			} else if c == '\'' {
				state = inCode
			}
		case inRaw:
			if c == '`' {
				state = inCode
			}
		case inString:
			switch {
			case c == '\\' && i+1 < len(src) && src[i+1] != '\n':
				i++
			case nestedQuote != 0:
				if c == nestedQuote {
					nestedQuote = 0
				}
			case c == '$' && i+1 < len(src) && src[i+1] == '{':
				interpolationDepth++
				i++
			case interpolationDepth > 0 && (c == '"' || c == '\''):
				nestedQuote = c
			case interpolationDepth > 0 && c == '{':
				interpolationDepth++
			case interpolationDepth > 0 && c == '}':
				interpolationDepth--
			case interpolationDepth == 0 && c == '"':
				state = inCode
			}
		}
	}
	for len(starts) < lineCount {
		starts = append(starts, false)
		comments = append(comments, false)
	}
	return starts, comments
}

// runFmt is the entry point invoked by the Cobra fmtCmd. It returns the
// exit code the caller should propagate (0 success, 1 on error).
func runFmt(args []string, checkMode bool) int {
	files, ok := expandGraySourceArgs("fmt", args)
	exit := 0
	if !ok {
		exit = 1
	}
	if len(files) == 0 {
		fmt.Println("gray fmt: no source files found")
		return exit
	}

	changed := 0

	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gray fmt: %v\n", err)
			exit = 1
			continue
		}
		orig, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gray fmt: %v\n", err)
			exit = 1
			continue
		}

		if checkMode {
			// Copy to temp, format it, compare
			tmp, err := os.CreateTemp("", "gray-fmt-check-*"+sourceExtensions[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "gray fmt: %v\n", err)
				exit = 1
				continue
			}
			tmpName := tmp.Name()
			tmp.Write(orig)
			tmp.Close()

			code, fmtErr := driver.Fmt(tmpName)
			if fmtErr != nil || code != 0 {
				os.Remove(tmpName)
				fmt.Fprintf(os.Stderr, "gray fmt: failed to format '%s'\n", path)
				exit = 1
				continue
			}

			formatted, err := os.ReadFile(tmpName)
			os.Remove(tmpName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gray fmt: %v\n", err)
				exit = 1
				continue
			}
			formatted = formatGraySource(formatted)
			if string(orig) != string(formatted) {
				fmt.Printf("would format: %s\n", path)
				if exit == 0 {
					exit = 1
				}
			}
			continue
		}

		// Normal mode: format in place
		code, err := driver.Fmt(path)
		if err != nil || code != 0 {
			fmt.Fprintf(os.Stderr, "gray fmt: failed to format '%s'\n", path)
			exit = 1
			continue
		}
		after, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gray fmt: %v\n", err)
			exit = 1
			continue
		}
		after = formatGraySource(after)
		if err := os.WriteFile(path, after, info.Mode()); err != nil {
			fmt.Fprintf(os.Stderr, "gray fmt: %v\n", err)
			exit = 1
			continue
		}
		if string(orig) != string(after) {
			fmt.Printf("formatted: %s\n", path)
			changed++
		}
	}

	if !checkMode && changed == 0 && exit == 0 {
		fmt.Printf("gray fmt: %d file(s) checked, already formatted\n", len(files))
	}
	return exit
}
