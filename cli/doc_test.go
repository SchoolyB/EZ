// doc_test.go — Tests for #doc collection in "gray doc".

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectDocsIgnoresRawStringsAndBlockComments(t *testing.T) {
	src := "/*\n#doc(\"in a comment\")\ndo commented() { }\n*/\n\n" +
		"#doc(\"real\")\ndo real() { }\n\n" +
		"do main() {\n    mut t string = `\n#doc(\"not real\")\ndo fake() { }\n`\n    println(t)\n}\n"
	path := filepath.Join(t.TempDir(), "f.gray")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	entries := collectDocsFromFile(path)
	if len(entries) != 1 || entries[0].Name != "real" {
		t.Errorf("got %+v, want only the function real", entries)
	}
}
