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

func TestCollectDocsJoinsWrappedFunctionSignature(t *testing.T) {
	src := "#doc(\"wide\")\nprivate do wide(\n    a i64,\n    b i64\n) -> i64 {\n    return a + b\n}\n"
	path := filepath.Join(t.TempDir(), "f.gray")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	entries := collectDocsFromFile(path)
	if len(entries) != 1 || entries[0].Signature != "private do wide(a i64, b i64) -> i64" || entries[0].Line != 2 {
		t.Errorf("got %+v", entries)
	}
}

func TestCollectDocsKeepsMultiLineVariableInitializer(t *testing.T) {
	src := "#doc(\"names\")\nconst NAMES [string, 2] = {\n    \"a\",\n    \"b\"\n}\n"
	path := filepath.Join(t.TempDir(), "f.gray")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	entries := collectDocsFromFile(path)
	want := "const NAMES [string, 2] = {\n    \"a\",\n    \"b\"\n}"
	if len(entries) != 1 || entries[0].Signature != want || entries[0].Line != 2 {
		t.Errorf("got %+v", entries)
	}
}

func TestExtractDocStringHandlesEscapedQuote(t *testing.T) {
	got := extractDocString(`#doc("Has \"quotes\" inside")`)
	if got != `Has "quotes" inside` {
		t.Errorf("got %q", got)
	}
}
