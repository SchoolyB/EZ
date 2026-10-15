// completion_test.go — Tests for shell tab-completion: the completion
// subcommand, source-file filtering, flag values, and gray man names.
//
// Author:  Marshall A Burns (@SchoolyB)
// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompletionSubcommandGeneratesEachShell(t *testing.T) {
	rootCmd.InitDefaultCompletionCmd()
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			var output bytes.Buffer
			var generateErr error
			switch shell {
			case "bash":
				generateErr = rootCmd.GenBashCompletionV2(&output, true)
			case "zsh":
				generateErr = rootCmd.GenZshCompletion(&output)
			case "fish":
				generateErr = rootCmd.GenFishCompletion(&output, true)
			case "powershell":
				generateErr = rootCmd.GenPowerShellCompletionWithDesc(&output)
			}
			if generateErr != nil {
				t.Fatalf("generating %s completion: %v", shell, generateErr)
			}
			if !strings.Contains(output.String(), "gray") {
				t.Errorf("%s completion script does not mention gray", shell)
			}
		})
	}

	command, _, err := rootCmd.Find([]string{"completion", "bash"})
	if err != nil || command.Name() != "bash" {
		t.Fatalf("rootCmd has no completion bash subcommand: %v", err)
	}
}

func TestIsCompletionCommand(t *testing.T) {
	rootCmd.InitDefaultCompletionCmd()
	// Cobra adds the hidden request command only while executing, so stand one in.
	request := &cobra.Command{Use: cobra.ShellCompRequestCmd}
	rootCmd.AddCommand(request)
	defer rootCmd.RemoveCommand(request)

	for _, path := range [][]string{{"completion"}, {"completion", "bash"}, {cobra.ShellCompRequestCmd}} {
		command, _, err := rootCmd.Find(path)
		if err != nil || !isCompletionCommand(command) {
			t.Errorf("isCompletionCommand(%v) = false, want true (err %v)", path, err)
		}
	}
	for _, path := range [][]string{{"build"}, {"man"}, {"cross", "build"}} {
		command, _, err := rootCmd.Find(path)
		if err != nil || isCompletionCommand(command) {
			t.Errorf("isCompletionCommand(%v) = true, want false (err %v)", path, err)
		}
	}
}

func TestSourceCommandsFilterToSourceExtensions(t *testing.T) {
	for _, name := range []string{"build", "check", "watch", "fmt", "doc", "test"} {
		command, _, err := rootCmd.Find([]string{name})
		if err != nil {
			t.Fatalf("finding %s: %v", name, err)
		}
		extensions, directive := command.ValidArgsFunction(command, nil, "")
		if directive != cobra.ShellCompDirectiveFilterFileExt || !reflect.DeepEqual(extensions, []string{"gray", "grayscale"}) {
			t.Errorf("gray %s completes %v / %v, want gray and grayscale extension filter", name, extensions, directive)
		}
	}
	command, _, _ := rootCmd.Find([]string{"cross", "build"})
	if command.ValidArgsFunction == nil {
		t.Error("gray cross build has no source file completion")
	}
}

func TestCompleteRootListsSourceFilesAndDirectories(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"main.gray", "lib.grayscale", "notes.txt", ".hidden.gray", "other.go"} {
		if err := os.WriteFile(filepath.Join(directory, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(directory, "src"), 0o755); err != nil {
		t.Fatal(err)
	}

	completions, directive := completeRoot(rootCmd, nil, directory+string(filepath.Separator))
	sort.Strings(completions)
	want := []string{
		directory + "/lib.grayscale",
		directory + "/main.gray",
		directory + "/src/",
	}
	if !reflect.DeepEqual(completions, want) {
		t.Errorf("completeRoot = %v, want %v", completions, want)
	}
	if directive&cobra.ShellCompDirectiveNoFileComp == 0 || directive&cobra.ShellCompDirectiveNoSpace == 0 {
		t.Errorf("completeRoot directive = %v, want NoFileComp and NoSpace", directive)
	}

	completions, _ = completeRoot(rootCmd, nil, filepath.Join(directory, "ma"))
	if !reflect.DeepEqual(completions, []string{directory + "/main.gray"}) {
		t.Errorf("completeRoot with prefix = %v, want only main.gray", completions)
	}

	completions, _ = completeRoot(rootCmd, []string{"main.gray"}, "")
	if len(completions) != 0 {
		t.Errorf("completeRoot after the file argument = %v, want none", completions)
	}
}

func TestFlagValueCompletion(t *testing.T) {
	cases := []struct {
		path []string
		flag string
		want []string
	}{
		{[]string{"new"}, "template", []string{"basic", "cli", "lib", "multi", "server", "client"}},
		{[]string{"new"}, "server-type", []string{"minimal", "normal"}},
		{[]string{"cross", "build"}, "target", []string{"linux-amd64", "linux-arm64", "mac-amd64", "mac-arm64", "windows-amd64"}},
	}
	for _, c := range cases {
		command, _, err := rootCmd.Find(c.path)
		if err != nil {
			t.Fatal(err)
		}
		complete, ok := command.GetFlagCompletionFunc(c.flag)
		if !ok {
			t.Errorf("%v --%s has no completion", c.path, c.flag)
			continue
		}
		got, directive := complete(command, nil, "")
		if !reflect.DeepEqual(got, c.want) || directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("%v --%s completes %v / %v, want %v", c.path, c.flag, got, directive, c.want)
		}
	}
}

func TestCompleteManNames(t *testing.T) {
	completions, directive := completeManNames(manCmd, nil, "os.ex")
	if !reflect.DeepEqual(completions, []string{"os.exec"}) || directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("man completion for os.ex = %v / %v, want [os.exec]", completions, directive)
	}

	completions, _ = completeManNames(manCmd, nil, "")
	for _, want := range []string{"builtins", "lang", "os", "os.exec", "println", "i64"} {
		found := false
		for _, name := range completions {
			if name == want {
				found = true
			}
		}
		if !found {
			t.Errorf("man completion is missing %q", want)
		}
	}

	completions, _ = completeManNames(manCmd, []string{"os"}, "")
	if len(completions) != 0 {
		t.Errorf("man completion after the first argument = %v, want none", completions)
	}
}

func TestCompleteRootListsSymlinkedDirectories(t *testing.T) {
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(directory, "sub"), filepath.Join(directory, "linkdir")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}

	completions, _ := completeRoot(rootCmd, nil, filepath.Join(directory, "link"))
	if !reflect.DeepEqual(completions, []string{directory + "/linkdir/"}) {
		t.Errorf("completeRoot = %v, want the symlinked directory with a trailing separator", completions)
	}
}
