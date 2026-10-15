// completion.go — Shell tab-completion for the gray CLI: source files,
// flag values, and gray man names.
//
// Author:  Marshall A Burns (@SchoolyB)
// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// manIndexNames are the index words gray man accepts besides entry names.
var manIndexNames = []string{"builtins", "builtin", "lang", "language", "keywords", "types", "symbols", "attributes"}

// isCompletionCommand reports whether cmd is the completion generator, one of
// its shell subcommands, or the hidden command shells call on every Tab press.
func isCompletionCommand(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		switch current.Name() {
		case "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			return true
		}
	}
	return false
}

// completeSourceFiles completes source files and directories for a command
// that takes source paths.
func completeSourceFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	extensions := make([]string, len(sourceExtensions))
	for index, extension := range sourceExtensions {
		extensions[index] = strings.TrimPrefix(extension, ".")
	}
	return extensions, cobra.ShellCompDirectiveFilterFileExt
}

// completeRoot completes the root command's own argument: the subcommand
// names Cobra adds itself, plus the source files and directories matching
// toComplete. They are listed by hand because a file-extension directive would
// treat the subcommand names as extensions.
func completeRoot(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	directory, prefix := filepath.Split(toComplete)
	listed := directory
	if listed == "" {
		listed = "."
	}
	entries, err := os.ReadDir(listed)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var completions []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || (strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".")) {
			continue
		}
		if entry.IsDir() {
			completions = append(completions, directory+name+string(filepath.Separator))
		} else if hasSourceExtension(name) {
			completions = append(completions, directory+name)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

// completeFixedValues returns a flag-value completer for a closed set.
func completeFixedValues(values ...string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return values, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeCrossTargets completes the --target values of gray cross build.
func completeCrossTargets(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	targets := make([]string, 0, len(crossTargets))
	for target := range crossTargets {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets, cobra.ShellCompDirectiveNoFileComp
}

// completeManNames completes the first argument of gray man: the index words,
// module names, builtins, qualified stdlib entries, and language reference
// entries.
func completeManNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	seen := make(map[string]bool)
	add := func(name string) {
		// A name starting with '-' is read as a flag when typed, so it is not offered.
		if strings.HasPrefix(name, toComplete) && !strings.HasPrefix(name, "-") {
			seen[name] = true
		}
	}
	for _, name := range manIndexNames {
		add(name)
	}
	for module := range stdlibModules {
		add(module)
	}
	for name := range builtinManDocs {
		add(name)
	}
	for name := range stdlibManDocs {
		add(name)
	}
	for name := range langManDocs {
		add(langDisplayName(name))
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, cobra.ShellCompDirectiveNoFileComp
}
