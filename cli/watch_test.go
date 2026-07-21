// watch_test.go — Tests for the file watcher covering main-function
// detection, import scanning, directory file collection, main-file
// discovery, and path shortening utilities.
//
// Author:  Marshall A Burns (@SchoolyB)
// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestHasMainFunction(t *testing.T) {
	dir := t.TempDir()

	withMain := filepath.Join(dir, "with_main.gray")
	os.WriteFile(withMain, []byte("do main() {\n    println(\"hi\")\n}\n"), 0o644)

	withMainArgs := filepath.Join(dir, "with_main_args.gray")
	os.WriteFile(withMainArgs, []byte("do main(args [string]) {\n}\n"), 0o644)

	noMain := filepath.Join(dir, "no_main.gray")
	os.WriteFile(noMain, []byte("do helper() {\n}\n"), 0o644)

	empty := filepath.Join(dir, "empty.gray")
	os.WriteFile(empty, []byte(""), 0o644)

	cases := []struct {
		name string
		path string
		want bool
	}{
		{"has do main()", withMain, true},
		{"has do main(args)", withMainArgs, true},
		{"no main", noMain, false},
		{"empty file", empty, false},
		{"nonexistent", filepath.Join(dir, "nope.gray"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hasMainFunction(c.path); got != c.want {
				t.Errorf("hasMainFunction(%q) = %v, want %v", c.path, got, c.want)
			}
		})
	}
}

func TestScanImports(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(dir, "main.gray")
	os.WriteFile(src, []byte(
		"import @math\n"+
			"import @strings, @io\n"+
			"import ./utils\n"+
			"import ./helpers using sqrt\n"+
			"do main() {}\n",
	), 0o644)

	got := scanImports(src)
	want := []string{"@math", "@strings", "@io", "./utils", "./helpers"}

	if len(got) != len(want) {
		t.Fatalf("scanImports got %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("import[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestScanImports_NonexistentFile(t *testing.T) {
	got := scanImports("/no/such/file.gray")
	if got != nil {
		t.Errorf("expected nil for missing file, got %v", got)
	}
}

func TestCollectGrayFilesInDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0o755)

	os.WriteFile(filepath.Join(dir, "a.gray"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "b.gray"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(sub, "c.gray"), []byte(""), 0o644)

	files := collectGrayFilesInDir(dir)
	var names []string
	for _, f := range files {
		names = append(names, filepath.Base(f))
	}
	sort.Strings(names)

	want := []string{"a.gray", "b.gray", "c.gray"}
	if len(names) != len(want) {
		t.Fatalf("collectGrayFilesInDir got %v, want %v", names, want)
	}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("[%d] got %q, want %q", i, names[i], w)
		}
	}
}

func TestFindMainFile(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		_, err := findMainFile(dir)
		if err == nil {
			t.Fatal("expected error for empty directory")
		}
	})

	t.Run("no main function", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "lib.gray"), []byte("do helper() {}\n"), 0o644)
		_, err := findMainFile(dir)
		if err == nil {
			t.Fatal("expected error when no main() present")
		}
	})

	t.Run("single main file", func(t *testing.T) {
		dir := t.TempDir()
		main := filepath.Join(dir, "main.gray")
		os.WriteFile(main, []byte("do main() {}\n"), 0o644)
		os.WriteFile(filepath.Join(dir, "lib.gray"), []byte("do helper() {}\n"), 0o644)

		got, err := findMainFile(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != main {
			t.Errorf("got %q, want %q", got, main)
		}
	})

	t.Run("multiple main files", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "a.gray"), []byte("do main() {}\n"), 0o644)
		os.WriteFile(filepath.Join(dir, "b.gray"), []byte("do main() {}\n"), 0o644)

		_, err := findMainFile(dir)
		if err == nil {
			t.Fatal("expected error for multiple main() files")
		}
	})
}

func TestCollectFilesToWatch_StdlibSkipped(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.gray")
	os.WriteFile(main, []byte("import @math\nimport @strings\ndo main() {}\n"), 0o644)

	files := collectFilesToWatch(main)
	// Only main.gray — stdlib imports must not be resolved to disk paths
	if len(files) != 1 {
		t.Errorf("expected 1 file (no stdlib resolved), got %v", files)
	}
	if files[0] != main {
		t.Errorf("got %q, want %q", files[0], main)
	}
}

func TestCollectFilesToWatch_LocalImport(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.gray")
	util := filepath.Join(dir, "util.gray")
	os.WriteFile(main, []byte("import ./util\ndo main() {}\n"), 0o644)
	os.WriteFile(util, []byte("do helper() {}\n"), 0o644)

	files := collectFilesToWatch(main)
	var names []string
	for _, f := range files {
		names = append(names, filepath.Base(f))
	}
	sort.Strings(names)

	if len(names) != 2 {
		t.Fatalf("expected 2 files, got %v", names)
	}
	if names[0] != "main.gray" || names[1] != "util.gray" {
		t.Errorf("got %v, want [main.gray util.gray]", names)
	}
}

func TestShortPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("getwd failed")
	}

	abs := filepath.Join(cwd, "some", "file.gray")
	got := shortPath(abs)
	// Should return relative path since it's shorter than abs or base
	if got == abs {
		t.Errorf("shortPath should shorten %q, returned full path", abs)
	}

	// Path outside cwd: should fall back to base name
	outside := "/completely/different/path/program.gray"
	if got := shortPath(outside); got == "" {
		t.Error("shortPath returned empty string for outside-cwd path")
	}
}
