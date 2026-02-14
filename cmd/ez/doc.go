package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/lexer"
	"github.com/marshallburns/ez/pkg/parser"
)

// DocEntry represents a documented item
type DocEntry struct {
	Name        string
	Kind        string // "function", "struct", "enum"
	Signature   string
	Description string
	File        string
}

// generateDocs is the entry point for the ez doc command
func generateDocs(args []string) {
	var entries []DocEntry

	for _, arg := range args {
		if strings.HasSuffix(arg, "/...") {
			// Recursive mode
			baseDir := strings.TrimSuffix(arg, "/...")
			if baseDir == "." || baseDir == "" {
				baseDir = "."
			}
			entries = append(entries, collectDocsRecursive(baseDir)...)
		} else if strings.HasSuffix(arg, ".ez") {
			// Single file
			entries = append(entries, collectDocsFromFile(arg)...)
		} else {
			// Directory (non-recursive)
			entries = append(entries, collectDocsFromDir(arg)...)
		}
	}

	if len(entries) == 0 {
		fmt.Println("No documented items found.")
		return
	}

	// Generate markdown
	output := generateMarkdown(entries)

	// Write to DOCS.md
	err := os.WriteFile("DOCS.md", []byte(output), 0644)
	if err != nil {
		fmt.Printf("Error writing DOCS.md: %v\n", err)
		return
	}

	fmt.Printf("Generated DOCS.md with %d documented item(s)\n", len(entries))
}

// collectDocsFromFile extracts documented items from a single .ez file
func collectDocsFromFile(filename string) []DocEntry {
	var entries []DocEntry

	absPath, err := filepath.Abs(filename)
	if err != nil {
		fmt.Printf("Error resolving path %s: %v\n", filename, err)
		return entries
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		fmt.Printf("Error reading file %s: %v\n", filename, err)
		return entries
	}

	source := string(data)
	l := lexer.NewLexer(source)
	p := parser.NewWithSource(l, source, absPath)
	program := p.ParseProgram()

	// Skip files with parse errors
	if p.EZErrors().HasErrors() {
		return entries
	}

	// Extract documented items
	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *ast.FunctionDeclaration:
			if doc := getDocAttribute(s.Attributes); doc != nil {
				entries = append(entries, DocEntry{
					Name:        s.Name.Value,
					Kind:        "function",
					Signature:   formatFunctionSignature(s),
					Description: getDocDescription(doc),
					File:        filename,
				})
			}
		case *ast.StructDeclaration:
			if doc := getDocAttribute(s.Attributes); doc != nil {
				entries = append(entries, DocEntry{
					Name:        s.Name.Value,
					Kind:        "struct",
					Signature:   formatStructSignature(s),
					Description: getDocDescription(doc),
					File:        filename,
				})
			}
		case *ast.EnumDeclaration:
			if s.DocAttribute != nil {
				entries = append(entries, DocEntry{
					Name:        s.Name.Value,
					Kind:        "enum",
					Signature:   formatEnumSignature(s),
					Description: getDocDescription(s.DocAttribute),
					File:        filename,
				})
			}
		}
	}

	return entries
}

// collectDocsFromDir collects docs from all .ez files in a directory (non-recursive)
func collectDocsFromDir(dir string) []DocEntry {
	var entries []DocEntry

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Printf("Error resolving path %s: %v\n", dir, err)
		return entries
	}

	files, err := os.ReadDir(absDir)
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", dir, err)
		return entries
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".ez") {
			entries = append(entries, collectDocsFromFile(filepath.Join(absDir, file.Name()))...)
		}
	}

	return entries
}

// collectDocsRecursive collects docs from all .ez files recursively
func collectDocsRecursive(dir string) []DocEntry {
	var entries []DocEntry

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Printf("Error resolving path %s: %v\n", dir, err)
		return entries
	}

	filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ez") {
			entries = append(entries, collectDocsFromFile(path)...)
		}
		return nil
	})

	return entries
}

// getDocAttribute finds the #doc attribute in a list of attributes
func getDocAttribute(attrs []*ast.Attribute) *ast.Attribute {
	for _, attr := range attrs {
		if attr.Name == "doc" {
			return attr
		}
	}
	return nil
}

// getDocDescription extracts the description string from a #doc attribute
func getDocDescription(doc *ast.Attribute) string {
	if len(doc.Args) > 0 {
		return doc.Args[0]
	}
	return ""
}

// formatFunctionSignature formats a function declaration as a signature string
func formatFunctionSignature(f *ast.FunctionDeclaration) string {
	var params []string
	for _, p := range f.Parameters {
		if p.Mutable {
			params = append(params, fmt.Sprintf("&%s %s", p.Name.Value, p.TypeName))
		} else {
			params = append(params, fmt.Sprintf("%s %s", p.Name.Value, p.TypeName))
		}
	}

	sig := fmt.Sprintf("do %s(%s)", f.Name.Value, strings.Join(params, ", "))

	if len(f.ReturnTypes) > 0 {
		if len(f.ReturnTypes) == 1 {
			sig += " -> " + f.ReturnTypes[0]
		} else {
			sig += " -> (" + strings.Join(f.ReturnTypes, ", ") + ")"
		}
	}

	return sig
}

// formatStructSignature formats a struct declaration
func formatStructSignature(s *ast.StructDeclaration) string {
	var fields []string
	for _, f := range s.Fields {
		fields = append(fields, fmt.Sprintf("    %s %s", f.Name.Value, f.TypeName))
	}
	return fmt.Sprintf("const %s struct {\n%s\n}", s.Name.Value, strings.Join(fields, "\n"))
}

// formatEnumSignature formats an enum declaration
func formatEnumSignature(e *ast.EnumDeclaration) string {
	var values []string
	for _, v := range e.Values {
		if v.Value != nil {
			values = append(values, fmt.Sprintf("    %s = %s", v.Name.Value, v.Value.TokenLiteral()))
		} else {
			values = append(values, "    "+v.Name.Value)
		}
	}
	return fmt.Sprintf("const %s enum {\n%s\n}", e.Name.Value, strings.Join(values, "\n"))
}

// generateMarkdown creates the markdown output from doc entries
func generateMarkdown(entries []DocEntry) string {
	var buf strings.Builder

	buf.WriteString("# Documentation\n\n")
	buf.WriteString("*Generated by `ez doc`*\n\n")

	// Group by kind
	var functions, structs, enums []DocEntry
	for _, e := range entries {
		switch e.Kind {
		case "function":
			functions = append(functions, e)
		case "struct":
			structs = append(structs, e)
		case "enum":
			enums = append(enums, e)
		}
	}

	// Sort each group alphabetically
	sort.Slice(functions, func(i, j int) bool { return functions[i].Name < functions[j].Name })
	sort.Slice(structs, func(i, j int) bool { return structs[i].Name < structs[j].Name })
	sort.Slice(enums, func(i, j int) bool { return enums[i].Name < enums[j].Name })

	// Functions section
	if len(functions) > 0 {
		buf.WriteString("## Functions\n\n")
		for _, f := range functions {
			buf.WriteString(fmt.Sprintf("### %s\n\n", f.Name))
			buf.WriteString(fmt.Sprintf("```ez\n%s\n```\n\n", f.Signature))
			if f.Description != "" {
				buf.WriteString(f.Description + "\n\n")
			}
		}
	}

	// Structs section
	if len(structs) > 0 {
		buf.WriteString("## Structs\n\n")
		for _, s := range structs {
			buf.WriteString(fmt.Sprintf("### %s\n\n", s.Name))
			buf.WriteString(fmt.Sprintf("```ez\n%s\n```\n\n", s.Signature))
			if s.Description != "" {
				buf.WriteString(s.Description + "\n\n")
			}
		}
	}

	// Enums section
	if len(enums) > 0 {
		buf.WriteString("## Enums\n\n")
		for _, e := range enums {
			buf.WriteString(fmt.Sprintf("### %s\n\n", e.Name))
			buf.WriteString(fmt.Sprintf("```ez\n%s\n```\n\n", e.Signature))
			if e.Description != "" {
				buf.WriteString(e.Description + "\n\n")
			}
		}
	}

	return buf.String()
}
