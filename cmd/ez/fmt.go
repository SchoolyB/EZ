package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// defaultFmtIndentSpaces is the width a leading tab is expanded to when
// normalizing indentation in formatEZSource.
const defaultFmtIndentSpaces = 4

// formatEZSource applies conservative formatting rules and returns the
// rewritten source. Rules are intentionally limited so `ez fmt` cannot
// accidentally alter the meaning of legal .ez code:
//
//  1. Strip trailing whitespace from every line.
//  2. Convert leading tabs in each line to defaultFmtIndentSpaces spaces
//     (only inside the indent prefix; tabs inside string literals or
//     after non-whitespace are left alone).
//  3. Collapse three-or-more consecutive blank lines down to exactly two.
//  4. Ensure the output ends with exactly one trailing newline.
//
// The function is byte-stable: feeding the output back through it
// produces the same bytes, so `ez fmt --check` is a reliable CI gate.
func formatEZSource(src []byte) []byte {
	// Split preserving the trailing-newline situation; a trailing empty
	// element after the final \n signals "file ended with a newline".
	lines := strings.Split(string(src), "\n")
	endsWithNewline := len(lines) > 0 && lines[len(lines)-1] == ""
	if endsWithNewline {
		lines = lines[:len(lines)-1]
	}

	for i, line := range lines {
		// Rule 2: leading-tab indent normalization.
		var indent strings.Builder
		j := 0
		for j < len(line) {
			c := line[j]
			if c == '\t' {
				for k := 0; k < defaultFmtIndentSpaces; k++ {
					indent.WriteByte(' ')
				}
				j++
			} else if c == ' ' {
				indent.WriteByte(' ')
				j++
			} else {
				break
			}
		}
		rest := line[j:]
		// Rule 1: trim trailing whitespace.
		rest = strings.TrimRight(rest, " \t")
		// An all-whitespace line collapses to "" so blank-line detection
		// in rule 3 catches it.
		if rest == "" {
			lines[i] = ""
		} else {
			lines[i] = indent.String() + rest
		}
	}

	// Rule 3: collapse 3+ consecutive blank lines to exactly 2.
	var out []string
	blank := 0
	for _, line := range lines {
		if line == "" {
			blank++
			if blank <= 2 {
				out = append(out, line)
			}
		} else {
			blank = 0
			out = append(out, line)
		}
	}

	// Rule 4: trim trailing blank lines, then append exactly one newline.
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return []byte(strings.Join(out, "\n") + "\n")
}

// collectFmtFiles expands the user-supplied args into a deduplicated list
// of .ez file paths, mirroring the arg shape used by `ez doc`:
//
//   - "./..." or "<dir>/..." -> recursive walk for .ez files
//   - "foo.ez"               -> single file
//   - "<dir>"                -> non-recursive directory scan
//
// Errors on individual entries are printed to stderr and the bad entry
// is skipped; the function never aborts the whole run on a single
// missing arg, matching how `generateDocs` in doc.go behaves.
func collectFmtFiles(args []string) []string {
	seen := make(map[string]struct{})
	var files []string
	add := func(p string) {
		ap, err := filepath.Abs(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ez fmt: cannot resolve %s: %v\n", p, err)
			return
		}
		if _, ok := seen[ap]; ok {
			return
		}
		seen[ap] = struct{}{}
		files = append(files, ap)
	}

	for _, arg := range args {
		switch {
		case strings.HasSuffix(arg, "/..."):
			baseDir := strings.TrimSuffix(arg, "/...")
			if baseDir == "" || baseDir == "." {
				baseDir = "."
			}
			filepath.Walk(baseDir, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() && strings.HasSuffix(p, ".ez") {
					add(p)
				}
				return nil
			})
		case strings.HasSuffix(arg, ".ez"):
			add(arg)
		default:
			entries, err := os.ReadDir(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ez fmt: %v\n", err)
				continue
			}
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".ez") {
					add(filepath.Join(arg, e.Name()))
				}
			}
		}
	}
	return files
}

// runFmt is the entry point invoked by the Cobra fmtCmd. It returns the
// exit code the caller should propagate (0 success, 1 if --check found
// unformatted files, 2 on file I/O error).
func runFmt(args []string, checkMode bool) int {
	files := collectFmtFiles(args)
	if len(files) == 0 {
		fmt.Println("ez fmt: no .ez files found")
		return 0
	}

	exit := 0
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ez fmt: %v\n", err)
			exit = 2
			continue
		}
		out := formatEZSource(src)
		if bytes.Equal(src, out) {
			continue
		}
		if checkMode {
			fmt.Printf("would format: %s\n", path)
			if exit == 0 {
				exit = 1
			}
			continue
		}
		if err := os.WriteFile(path, out, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "ez fmt: writing %s: %v\n", path, err)
			exit = 2
			continue
		}
		fmt.Printf("formatted: %s\n", path)
	}
	return exit
}
