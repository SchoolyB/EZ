// watch.go — File watcher for live development ("gray watch"). Monitors
// a source file or directory for changes and automatically re-runs the
// program with debounced filesystem event handling.
//
// Author:  Marshall A Burns (@SchoolyB)
// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/grayscale-lang/grayscale/internal/grayc"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch [file.gray | directory]",
	Short: "Watch files and re-run on changes",
	Long:  `Watch a file or directory for changes and automatically re-run.`,
	Args:  cobra.ExactArgs(1),
	Run:   runWatch,
}

func runWatch(cmd *cobra.Command, args []string) {
	target := args[0]

	var compilerArgs []string
	quiet, _ := cmd.Flags().GetString("quiet")
	if quiet == "all" {
		compilerArgs = append(compilerArgs, "--quiet")
	} else if quiet != "" {
		compilerArgs = append(compilerArgs, "--quiet", quiet)
	}
	if noColor, _ := cmd.Flags().GetBool("no-color"); noColor {
		compilerArgs = append(compilerArgs, "--no-color")
	}

	// Resolve to absolute path
	absTarget, err := filepath.Abs(target)
	if err != nil {
		fmt.Printf("Error resolving path: %v\n", err)
		os.Exit(1)
	}

	// Check if target exists
	info, err := os.Stat(absTarget)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if info.IsDir() {
		watchDirectory(absTarget, compilerArgs)
	} else {
		if !strings.HasSuffix(absTarget, ".gray") {
			fmt.Println("Error: file must have .gray extension")
			os.Exit(1)
		}
		watchFile(absTarget, compilerArgs)
	}
}

// watchFile watches a single file and its imports for changes
func watchFile(file string, compilerArgs []string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("Error creating watcher: %v\n", err)
		os.Exit(1)
	}
	defer watcher.Close()

	// Get the list of files to watch (main file + imports via text scan)
	filesToWatch := collectFilesToWatch(file)

	for _, f := range filesToWatch {
		if err := watcher.Add(f); err != nil {
			fmt.Printf("Error watching %s: %v\n", f, err)
		}
	}

	importCount := len(filesToWatch) - 1
	if importCount > 0 {
		fmt.Printf("Watching %s (+ %d imports)\n", shortPath(file), importCount)
	} else {
		fmt.Printf("Watching %s\n", shortPath(file))
	}
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Run initially
	executeFile(file, compilerArgs)

	var mu sync.Mutex
	var timer *time.Timer
	debounceInterval := 100 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				mu.Lock()
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounceInterval, func() {
					newFilesToWatch := collectFilesToWatch(file)
					for _, f := range newFilesToWatch {
						watcher.Add(f)
					}
					executeFile(file, compilerArgs)
				})
				mu.Unlock()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Watcher error: %v\n", err)
		}
	}
}

// watchDirectory watches all .gray files in a directory
func watchDirectory(dirPath string, compilerArgs []string) {
	mainFile, err := findMainFile(dirPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("Error creating watcher: %v\n", err)
		os.Exit(1)
	}
	defer watcher.Close()

	filesToWatch := collectGrayFilesInDir(dirPath)

	for _, f := range filesToWatch {
		if err := watcher.Add(f); err != nil {
			fmt.Printf("Error watching %s: %v\n", f, err)
		}
	}

	fmt.Printf("Watching %s (%d files, entry: %s)\n", shortPath(dirPath), len(filesToWatch), shortPath(mainFile))
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	executeFile(mainFile, compilerArgs)

	var mu sync.Mutex
	var timer *time.Timer
	debounceInterval := 100 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if !strings.HasSuffix(event.Name, ".gray") {
					continue
				}
				mu.Lock()
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounceInterval, func() {
					newFilesToWatch := collectGrayFilesInDir(dirPath)
					for _, f := range newFilesToWatch {
						watcher.Add(f)
					}
					executeFile(mainFile, compilerArgs)
				})
				mu.Unlock()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Watcher error: %v\n", err)
		}
	}
}

// collectFilesToWatch returns the main file plus imported file paths via text scan.
// No parser needed — just scans for import lines.
func collectFilesToWatch(mainFile string) []string {
	files := []string{mainFile}
	dir := filepath.Dir(mainFile)
	seen := map[string]struct{}{mainFile: {}}

	imports := scanImports(mainFile)
	for _, imp := range imports {
		// Skip stdlib imports (@std, @math, etc.)
		if strings.HasPrefix(imp, "@") {
			continue
		}
		// Resolve relative to the main file's directory
		resolved := filepath.Join(dir, imp)
		if !strings.HasSuffix(resolved, ".gray") {
			// Could be a directory module
			if info, err := os.Stat(resolved); err == nil && info.IsDir() {
				dirFiles := collectGrayFilesInDir(resolved)
				for _, f := range dirFiles {
					if _, ok := seen[f]; !ok {
						files = append(files, f)
						seen[f] = struct{}{}
					}
				}
				continue
			}
			resolved += ".gray"
		}
		if _, ok := seen[resolved]; !ok {
			if _, err := os.Stat(resolved); err == nil {
				files = append(files, resolved)
				seen[resolved] = struct{}{}
			}
		}
	}
	return files
}

// scanImports does a simple text scan for import statements in a Grayscale file.
// Returns the import paths (e.g., "@std", "./mymodule", "utils").
func scanImports(filePath string) []string {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "import ") {
			continue
		}
		// import @math, ./mymodule
		rest := strings.TrimPrefix(line, "import ")
		// Strip "using ..." suffix
		if idx := strings.Index(rest, " using "); idx >= 0 {
			rest = rest[:idx]
		}
		for _, part := range strings.Split(rest, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				imports = append(imports, part)
			}
		}
	}
	return imports
}

// collectGrayFilesInDir recursively collects all .gray files in a directory
func collectGrayFilesInDir(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".gray") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

// findMainFile finds the .gray file containing do main() in a directory
func findMainFile(dir string) (string, error) {
	grayFiles := collectGrayFilesInDir(dir)
	if len(grayFiles) == 0 {
		return "", fmt.Errorf("no .gray files found in %s", dir)
	}

	var mainFiles []string
	for _, filePath := range grayFiles {
		if hasMainFunction(filePath) {
			mainFiles = append(mainFiles, filePath)
		}
	}

	if len(mainFiles) == 0 {
		return "", fmt.Errorf("no main() function found in %s\n  = help: create a file with a main() function", dir)
	}
	if len(mainFiles) > 1 {
		var fileList strings.Builder
		for _, f := range mainFiles {
			fileList.WriteString(fmt.Sprintf("\n    - %s", shortPath(f)))
		}
		return "", fmt.Errorf("multiple main() functions found in %s:%s\n  = help: only one file should contain main()", dir, fileList.String())
	}
	return mainFiles[0], nil
}

// hasMainFunction checks if a file contains "do main()" via text scan
func hasMainFunction(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "do main(") {
			return true
		}
	}
	return false
}

// executeFile runs the given file via grayc and prints output
func executeFile(filename string, compilerArgs []string) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] Running %s...\n", timestamp, shortPath(filename))

	_, err := grayc.Run(filename, compilerArgs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	printSeparator()
}

func printSeparator() {
	fmt.Println("────────────────────────────")
}

func shortPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Base(path)
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return filepath.Base(path)
	}
	if len(rel) < len(path) {
		return rel
	}
	return filepath.Base(path)
}
