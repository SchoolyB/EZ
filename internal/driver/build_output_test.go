package driver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildRefusesOutputEqualToInput(t *testing.T) {
	source := filepath.Join(t.TempDir(), "victim.gray")
	if err := os.WriteFile(source, []byte("do main() { }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, err := Build(source, BuildOpts{Output: source})
	if err == nil || code == 0 {
		t.Fatalf("Build accepted an output path equal to the input: code=%d err=%v", code, err)
	}
	contents, readErr := os.ReadFile(source)
	if readErr != nil || string(contents) != "do main() { }\n" {
		t.Errorf("source was modified: %q (%v)", contents, readErr)
	}
}

func TestBuildRefusesUnusableOutputPath(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "one.gray")
	if err := os.WriteFile(source, []byte("do main() { }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{directory, filepath.Join(directory, "nodir", "app")} {
		if code, err := Build(source, BuildOpts{Output: output}); err == nil || code == 0 {
			t.Errorf("Build accepted unusable output %q: code=%d err=%v", output, code, err)
		}
	}
}
