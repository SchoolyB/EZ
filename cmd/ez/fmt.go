package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/marshallburns/ez/internal/ezc"
)

// collectFmtFiles expands the user-supplied args into a deduplicated list
// of .ez file paths. Supported forms:
//
//   - "file.ez"           — single file
//   - "dir"               — all .ez files directly inside dir (non-recursive)
//   - "dir/subdir"        — all .ez files directly inside dir/subdir
//   - "./..." or "dir/..." — recursive walk for .ez files
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
		if strings.HasSuffix(arg, "/...") || arg == "..." {
			baseDir := strings.TrimSuffix(arg, "/...")
			if baseDir == "" || baseDir == "." || baseDir == "..." {
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
			continue
		}

		info, err := os.Stat(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ez fmt: %v\n", err)
			continue
		}

		if info.IsDir() {
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
		} else if strings.HasSuffix(arg, ".ez") {
			add(arg)
		} else {
			fmt.Fprintf(os.Stderr, "ez fmt: '%s' is not a .ez file or directory\n", arg)
		}
	}
	return files
}

// runFmt is the entry point invoked by the Cobra fmtCmd. It returns the
// exit code the caller should propagate (0 success, 1 on error).
func runFmt(args []string, checkMode bool) int {
	files := collectFmtFiles(args)
	if len(files) == 0 {
		fmt.Println("ez fmt: no .ez files found")
		return 0
	}

	exit := 0
	changed := 0

	for _, path := range files {
		orig, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ez fmt: %v\n", err)
			exit = 1
			continue
		}

		if checkMode {
			// Copy to temp, format it, compare
			tmp, err := os.CreateTemp("", "ez-fmt-check-*.ez")
			if err != nil {
				fmt.Fprintf(os.Stderr, "ez fmt: %v\n", err)
				exit = 1
				continue
			}
			tmpName := tmp.Name()
			tmp.Write(orig)
			tmp.Close()

			ezc.Fmt(tmpName)

			formatted, err := os.ReadFile(tmpName)
			os.Remove(tmpName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ez fmt: %v\n", err)
				exit = 1
				continue
			}
			if string(orig) != string(formatted) {
				fmt.Printf("would format: %s\n", path)
				if exit == 0 {
					exit = 1
				}
			}
			continue
		}

		// Normal mode: format in place
		code, err := ezc.Fmt(path)
		if err != nil || code != 0 {
			fmt.Fprintf(os.Stderr, "ez fmt: failed to format '%s'\n", path)
			exit = 1
			continue
		}
		after, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ez fmt: %v\n", err)
			exit = 1
			continue
		}
		if string(orig) != string(after) {
			fmt.Printf("formatted: %s\n", path)
			changed++
		}
	}

	if !checkMode && changed == 0 {
		fmt.Printf("ez fmt: %d file(s) checked, already formatted\n", len(files))
	}
	return exit
}
