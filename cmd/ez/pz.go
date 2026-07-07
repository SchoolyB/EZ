package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed templates
var templatesFS embed.FS

var errPzCancelled = fmt.Errorf("pz cancelled")

// quickRefBlock is prepended to the entry file of a scaffolded project
// when `ez pz -c` is used. Keep it short — a dense EZ 3.0 cheat sheet,
// not a tutorial.
const quickRefBlock = `// EZ 3.0 Quick Reference
// ---------------------
// Variables:        mut x int = 5          mut x = 5          (inferred)
// Constants:        const PI float = 3.14
// Strings:          "Hello, ${name}!"      (interpolation; + is not supported)
// Functions:        do greet(name string) -> string { return "Hi ${name}" }
// Multi-return:     do swap(a int, b int) -> (x int, y int) { return b, a }
// If/else:          if x > 0 { } or x == 0 { } otherwise { }
// Loops:            for i in range(0, 10) { }    for_each item in items { }
// Structs:          const Point struct { x int, y int }
//                   mut p = Point{ x: 1, y: 2 }
// Imports:          import @os                   import "./sibling"
//
`

var pzCmd = &cobra.Command{
	Use:   "pz [project-name]",
	Short: "Scaffold a new EZ project",
	Long: `Scaffold a new EZ project with templates.

Examples:
  ez pz                      Interactive mode
  ez pz myproject            Create with basic template
  ez pz myproject -t cli     Create with cli template
  ez pz myproject -t lib -c  Create library with comments

Templates:
  basic  - Single file hello world (default)
  cli    - CLI application with arg handling
  lib    - Reusable library module
  multi  - Multi-module project
  server - HTTP server (use -s for minimal/normal)
  client - HTTP client (use -s for minimal/normal)`,
	Args: cobra.MaximumNArgs(1),
	Run:  runPz,
}

func runPz(cmd *cobra.Command, args []string) {
	template, _ := cmd.Flags().GetString("template")
	comments, _ := cmd.Flags().GetBool("comments")
	force, _ := cmd.Flags().GetBool("force")
	serverType, _ := cmd.Flags().GetString("server-type")

	var name string

	if len(args) == 0 {
		name = promptForInput("Project name: ", "")
		if name == "" {
			fmt.Fprintln(os.Stderr, "error: project name is required")
			os.Exit(1)
		}

		if !cmd.Flags().Changed("template") {
			template = promptForInput("Template (basic/cli/lib/multi/server/client) [basic]: ", "basic")
			if template == "" {
				template = "basic"
			}
		}

		if (template == "server" || template == "client") && !cmd.Flags().Changed("server-type") {
			serverType = promptForInput("Type (minimal/normal) [normal]: ", "normal")
			if serverType == "" {
				serverType = "normal"
			}
		}

		if !cmd.Flags().Changed("comments") {
			commentsStr := promptForInput("Include comments? (y/N): ", "n")
			comments = strings.ToLower(commentsStr) == "y" || strings.ToLower(commentsStr) == "yes"
		}
	} else {
		name = args[0]
	}

	if !isValidProjectName(name) {
		fmt.Fprintln(os.Stderr, "error: project name must be a plain directory name with no path separators, '..' sequences, or absolute paths")
		os.Exit(1)
	}

	validTemplates := map[string]bool{"basic": true, "cli": true, "lib": true, "multi": true, "server": true, "client": true}
	if !validTemplates[template] {
		fmt.Printf("Invalid template '%s'. Choose from: basic, cli, lib, multi, server, client\n", template)
		return
	}

	if template == "server" || template == "client" {
		validSubTypes := map[string]bool{"minimal": true, "normal": true}
		if !validSubTypes[serverType] {
			fmt.Printf("Invalid type '%s'. Choose from: minimal, normal\n", serverType)
			return
		}
	}

	if err := createProject(name, template, comments, force, serverType); err != nil {
		if err != errPzCancelled {
			fmt.Printf("Error: %v\n", err)
		}
		return
	}

	switch template {
	case "server":
		fmt.Printf("\nDone! Run your server:\n")
		fmt.Printf("  cd %s && ez main.ez\n", name)
		fmt.Printf("  # Then visit http://localhost:8080\n")
	case "client":
		fmt.Printf("\nDone! Run your client:\n")
		fmt.Printf("  cd %s && ez main.ez\n", name)
	default:
		fmt.Printf("\nDone! Run your project:\n")
		fmt.Printf("  cd %s && ez main.ez\n", name)
	}
}

// isValidProjectName returns true only when name is a plain single-element
// directory name. Names containing path separators, parent references (..),
// absolute paths, or home-directory shortcuts (~) are rejected so that
// createProject cannot write or delete files outside the current directory.
func isValidProjectName(name string) bool {
	if name == "" {
		return false
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, "~") {
		return false
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return false
	}
	cleaned := filepath.Clean(name)
	return cleaned == name && cleaned != "." && cleaned != ".."
}

func promptForInput(prompt, defaultVal string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func createProject(name, template string, comments, force bool, serverType string) error {
	if _, err := os.Stat(name); err == nil {
		if !force {
			return fmt.Errorf("directory '%s' already exists (use --force to overwrite)", name)
		}
		fmt.Printf("Directory '%s' already exists and will be permanently deleted.\n", name)
		fmt.Printf("Delete '%s/' and continue? (y/N): ", name)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return errPzCancelled
		}
		os.RemoveAll(name)
	}

	fmt.Printf("\nCreating project '%s' with template '%s'...\n", name, template)

	if err := os.MkdirAll(name, 0755); err != nil {
		return err
	}
	fmt.Printf("  created %s/\n", name)

	// Resolve which template subtree to walk, and which file (if any)
	// receives a name-based rename at scaffold time.
	src, entry, rename := resolveTemplate(template, serverType, name)

	return extractTemplate(src, name, entry, rename, comments)
}

// resolveTemplate maps a (template, serverType) pair to the embedded
// template subdirectory, the path of the entry file within that subtree
// (used by the -c flag and for name substitution), and an optional
// {oldPath: newPath} rename applied during extraction.
func resolveTemplate(template, serverType, projectName string) (src, entry string, rename map[string]string) {
	switch template {
	case "basic":
		return "templates/basic", "main.ez", nil
	case "cli":
		return "templates/cli", "main.ez", nil
	case "lib":
		return "templates/lib", "main.ez", nil
	case "multi":
		return "templates/multi", "main.ez", nil
	case "server":
		if serverType == "minimal" {
			return "templates/server_minimal", "main.ez", nil
		}
		return "templates/server_normal", "main.ez", nil
	case "client":
		if serverType == "minimal" {
			return "templates/client_minimal", "main.ez", nil
		}
		return "templates/client_normal", "main.ez", nil
	}
	return "", "", nil
}

// extractTemplate walks the embedded template subtree rooted at src and
// materialises it into destRoot, applying any rename and prepending the
// quick-reference comment block to entryFile when comments is set.
func extractTemplate(src, destRoot, entryFile string, rename map[string]string, comments bool) error {
	return fs.WalkDir(templatesFS, src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == src {
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if newName, ok := rename[rel]; ok {
			rel = newName
		}
		dest := filepath.Join(destRoot, rel)

		if d.IsDir() {
			if err := os.MkdirAll(dest, 0755); err != nil {
				return err
			}
			fmt.Printf("  created %s/\n", dest)
			return nil
		}

		data, err := templatesFS.ReadFile(path)
		if err != nil {
			return err
		}

		if comments && rel == entryFile {
			data = append([]byte(quickRefBlock), data...)
		}

		if err := os.WriteFile(dest, data, 0644); err != nil {
			return err
		}
		fmt.Printf("  created %s\n", dest)
		return nil
	})
}
