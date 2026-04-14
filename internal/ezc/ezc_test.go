package ezc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStatFile(t *testing.T) {
	dir := t.TempDir()

	regular := filepath.Join(dir, "a_regular_file")
	if err := os.WriteFile(regular, []byte("x"), 0o644); err != nil {
		t.Fatalf("write regular: %v", err)
	}

	subdir := filepath.Join(dir, "a_directory")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	symlink := filepath.Join(dir, "a_symlink_to_file")
	if err := os.Symlink(regular, symlink); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	cases := []struct {
		name string
		path string
		want bool
	}{
		{"regular file", regular, true},
		{"directory", subdir, false},
		{"nonexistent", filepath.Join(dir, "nope"), false},
		{"empty path", "", false},
		// os.Stat follows symlinks, so a symlink pointing at a regular
		// file stats as a regular file. Documenting current behavior.
		{"symlink to regular", symlink, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := statFile(c.path); got != c.want {
				t.Errorf("statFile(%q) = %v, want %v", c.path, got, c.want)
			}
		})
	}
}

func TestFindCompilerPathOverride(t *testing.T) {
	dir := t.TempDir()

	regular := filepath.Join(dir, "ezc")
	if err := os.WriteFile(regular, []byte("x"), 0o755); err != nil {
		t.Fatalf("write regular: %v", err)
	}
	subdir := filepath.Join(dir, "ezc_as_directory")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	t.Run("regular file is returned", func(t *testing.T) {
		t.Setenv("EZ_COMPILER_PATH", regular)
		got, err := Find()
		if err != nil {
			t.Fatalf("Find: %v", err)
		}
		if got != regular {
			t.Errorf("Find() = %q, want %q", got, regular)
		}
	})

	t.Run("directory is not returned", func(t *testing.T) {
		// Regression for #1461 follow-up: EZ_COMPILER_PATH pointing at a
		// directory must not be returned verbatim. Either another
		// fallback path succeeds (in which case Find returns that),
		// or Find returns the not-found error — both outcomes mean
		// path 1 rejected the directory. We only care that the
		// returned path is NOT the directory itself.
		t.Setenv("EZ_COMPILER_PATH", subdir)
		got, _ := Find()
		if got == subdir {
			t.Errorf("Find() returned EZ_COMPILER_PATH directory %q; expected fall-through", subdir)
		}
	})

	t.Run("nonexistent path is not returned", func(t *testing.T) {
		nonexistent := filepath.Join(dir, "does_not_exist")
		t.Setenv("EZ_COMPILER_PATH", nonexistent)
		got, _ := Find()
		if got == nonexistent {
			t.Errorf("Find() returned nonexistent EZ_COMPILER_PATH %q; expected fall-through", nonexistent)
		}
	})
}
