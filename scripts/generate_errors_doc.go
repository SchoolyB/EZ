// Package main generates ERRORS.md from pkg/errors/codes.go
//
// Usage: go run scripts/generate_errors_doc.go
package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

const codesGoPath = "pkg/errors/codes.go"
const outputPath = "ERRORS.md"

type ErrorCode struct {
	Code    string
	Slug    string
	Message string
}

func main() {
	data, err := os.ReadFile(codesGoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", codesGoPath, err)
		os.Exit(1)
	}

	codes := parseCodesGo(string(data))
	markdown := generateMarkdown(codes)

	if err := os.WriteFile(outputPath, []byte(markdown), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outputPath, err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s with %d error codes\n", outputPath, len(codes))
}

func parseCodesGo(data string) []ErrorCode {
	var codes []ErrorCode

	re := regexp.MustCompile(`(E\d+|W\d+)\s*=\s*ErrorCode\{"([^"]+)",\s*"([^"]+)",\s*"([^"]+)"\}`)
	matches := re.FindAllStringSubmatch(data, -1)

	for _, match := range matches {
		if len(match) == 5 {
			codes = append(codes, ErrorCode{
				Code:    match[2],
				Slug:    match[3],
				Message: match[4],
			})
		}
	}

	sort.Slice(codes, func(i, j int) bool {
		return codes[i].Code < codes[j].Code
	})

	return codes
}

func generateMarkdown(codes []ErrorCode) string {
	var sb strings.Builder

	sb.WriteString("# EZ Error Reference\n\n")
	sb.WriteString("This document is auto-generated from `pkg/errors/codes.go`.\n\n")
	sb.WriteString("Run `go run scripts/generate_errors_doc.go` to regenerate.\n\n")
	sb.WriteString("---\n\n")

	categories := []struct {
		prefix string
		name   string
		desc   string
	}{
		{"E1", "Lexer Errors (E1xxx)", "Tokenization and lexical analysis errors"},
		{"E2", "Parse Errors (E2xxx)", "Syntax and parsing errors"},
		{"E3", "Type Errors (E3xxx)", "Type system errors"},
		{"E4", "Reference Errors (E4xxx)", "Undefined variables, functions, modules"},
		{"E5", "Runtime Errors (E5xxx)", "General runtime errors"},
		{"E6", "Import Errors (E6xxx)", "Module import errors"},
		{"E7", "Stdlib Errors (E7xxx)", "Standard library validation errors"},
		{"E8", "Math Errors (E8xxx)", "Mathematical operation errors"},
		{"E9", "Array Errors (E9xxx)", "Array-specific errors"},
		{"E10", "String Errors (E10xxx)", "String-specific errors"},
		{"E11", "Time Errors (E11xxx)", "Time-specific errors"},
		{"E12", "Map Errors (E12xxx)", "Map-specific errors"},
		{"E13", "JSON Errors (E13xxx)", "JSON module errors"},
		{"E14", "HTTP Errors (E14xxx)", "HTTP request errors"},
		{"E15", "Crypto Errors (E15xxx)", "Cryptography-specific errors"},
		{"E16", "Encoding Errors (E16xxx)", "Encoding/decoding errors"},
		{"E17", "DB Errors (E17xxx)", "Database operation errors"},
		{"E18", "Server Errors (E18xxx)", "HTTP server-specific errors"},
		{"W1", "Code Style Warnings (W1xxx)", "Unused declarations"},
		{"W2", "Potential Bug Warnings (W2xxx)", "Possible bugs"},
		{"W3", "Code Quality Warnings (W3xxx)", "Code quality issues"},
		{"W4", "Module Warnings (W4xxx)", "Module-related warnings"},
	}

	for _, cat := range categories {
		catCodes := filterByPrefix(codes, cat.prefix)
		if len(catCodes) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %s\n\n", cat.name))
		sb.WriteString(fmt.Sprintf("%s\n\n", cat.desc))
		sb.WriteString("| Code | Type | Message |\n")
		sb.WriteString("|------|------|---------|\n")

		for _, c := range catCodes {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", c.Code, c.Slug, c.Message))
		}

		sb.WriteString("\n")
	}

	// Summary
	sb.WriteString("---\n\n")
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("**Total:** %d error/warning codes\n", len(codes)))

	return sb.String()
}

func filterByPrefix(codes []ErrorCode, prefix string) []ErrorCode {
	var result []ErrorCode
	for _, c := range codes {
		if matchesPrefix(c.Code, prefix) {
			result = append(result, c)
		}
	}
	return result
}

func matchesPrefix(code, prefix string) bool {
	// Double-digit prefixes (E10-E18) produce 6-character codes (E1xxxx)
	// Single-digit prefixes (E1-E9, W1-W4) produce 5-character codes (Exxx, Wxxx)

	if len(prefix) == 3 && prefix[0] == 'E' && prefix[1] >= '1' && prefix[2] >= '0' {
		// E10xxx - E18xxx codes must be 6 characters AND start with the prefix
		return len(code) == 6 && strings.HasPrefix(code, prefix)
	}

	// Single-digit prefixes
	if !strings.HasPrefix(code, prefix) {
		return false
	}

	// E1-E9 codes must be exactly 5 characters
	if len(prefix) == 2 && prefix[0] == 'E' && prefix[1] >= '1' && prefix[1] <= '9' {
		return len(code) == 5
	}

	// W1-W4 codes must be exactly 5 characters
	if len(prefix) == 2 && prefix[0] == 'W' && prefix[1] >= '1' && prefix[1] <= '4' {
		return len(code) == 5
	}

	return true
}
