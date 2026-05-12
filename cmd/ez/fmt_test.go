package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"testing"
)

func TestFormatEZSourceTrailingWhitespace(t *testing.T) {
	in := "do main() {  \n    println(\"hi\")\t \n}\n"
	want := "do main() {\n    println(\"hi\")\n}\n"
	got := string(formatEZSource([]byte(in)))
	if got != want {
		t.Fatalf("trailing whitespace not trimmed\ngot:  %q\nwant: %q", got, want)
	}
}

func TestFormatEZSourceTabsToSpaces(t *testing.T) {
	in := "do main() {\n\tprintln(\"hi\")\n\t\treturn 0\n}\n"
	want := "do main() {\n    println(\"hi\")\n        return 0\n}\n"
	got := string(formatEZSource([]byte(in)))
	if got != want {
		t.Fatalf("leading tabs not expanded to 4 spaces\ngot:  %q\nwant: %q", got, want)
	}
}

func TestFormatEZSourceCollapseBlankLines(t *testing.T) {
	in := "do main() {\n    println(\"a\")\n\n\n\n\n    println(\"b\")\n}\n"
	want := "do main() {\n    println(\"a\")\n\n\n    println(\"b\")\n}\n"
	got := string(formatEZSource([]byte(in)))
	if got != want {
		t.Fatalf("blank-line run not collapsed to 2\ngot:  %q\nwant: %q", got, want)
	}
}

func TestFormatEZSourceEnsuresFinalNewline(t *testing.T) {
	for _, in := range []string{
		"println(\"hi\")",
		"println(\"hi\")\n",
		"println(\"hi\")\n\n\n",
	} {
		want := "println(\"hi\")\n"
		got := string(formatEZSource([]byte(in)))
		if got != want {
			t.Fatalf("EOF not normalized for %q\ngot:  %q\nwant: %q", in, got, want)
		}
	}
}

func TestFormatEZSourceIdempotent(t *testing.T) {
	in := []byte("do main() {\n\tif x {\n\t\tprintln(\"a\")  \n\n\n\n\t}\n}\n")
	once := formatEZSource(in)
	twice := formatEZSource(once)
	if string(once) != string(twice) {
		t.Fatalf("format is not idempotent\nonce:  %q\ntwice: %q", once, twice)
	}
}

func TestFormatEZSourceCleanInputUnchanged(t *testing.T) {
	in := "do main() {\n    println(\"hi\")\n}\n"
	got := string(formatEZSource([]byte(in)))
	if got != in {
		t.Fatalf("clean input rewritten\ngot:  %q\nwant: %q", got, in)
	}
}

func TestFormatEZSourceMixedTabsAndSpaces(t *testing.T) {
	in := "if x {\n\t    println(\"mixed\")\n}\n"
	want := "if x {\n        println(\"mixed\")\n}\n"
	got := string(formatEZSource([]byte(in)))
	if got != want {
		t.Fatalf("mixed tab+space indent not normalized\ngot:  %q\nwant: %q", got, want)
	}
}
