package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// ensureCompletionInstalled auto-installs shell completions on first run.
// It checks a flag file at ~/.ez/completion_installed and skips if present.
// Any error is silently ignored — this must never block the user.
func ensureCompletionInstalled() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	flagFile := filepath.Join(home, ".ez", "completion_installed")
	if _, err := os.Stat(flagFile); err == nil {
		return
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		return
	}
	shellName := filepath.Base(shell)

	var buf bytes.Buffer
	var scriptPath string
	var rcFile string
	var rcSnippet string

	switch shellName {
	case "zsh":
		if err := rootCmd.GenZshCompletion(&buf); err != nil {
			return
		}
		dir := filepath.Join(home, ".zfunc")
		scriptPath = filepath.Join(dir, "_ez")
		rcFile = filepath.Join(home, ".zshrc")
		rcSnippet = "\n# EZ shell completions\nfpath+=~/.zfunc\nautoload -U compinit && compinit\n"

	case "bash":
		if err := rootCmd.GenBashCompletionV2(&buf, true); err != nil {
			return
		}
		dir := filepath.Join(home, ".bash_completion.d")
		scriptPath = filepath.Join(dir, "ez")
		rcFile = filepath.Join(home, ".bashrc")
		rcSnippet = "\n# EZ shell completions\n. ~/.bash_completion.d/ez\n"

	case "fish":
		if err := rootCmd.GenFishCompletion(&buf, true); err != nil {
			return
		}
		dir := filepath.Join(home, ".config", "fish", "completions")
		scriptPath = filepath.Join(dir, "ez.fish")
		// fish auto-loads from completions dir, no rc edit needed

	default:
		return
	}

	// Write the completion script
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
		return
	}
	if err := os.WriteFile(scriptPath, buf.Bytes(), 0644); err != nil {
		return
	}

	// Append source/fpath line to shell rc if needed
	if rcFile != "" {
		rcContents, err := os.ReadFile(rcFile)
		if err != nil && !os.IsNotExist(err) {
			return
		}
		if !strings.Contains(string(rcContents), rcSnippet) {
			// Check for the key line to avoid duplicating
			keyLine := ""
			switch shellName {
			case "zsh":
				keyLine = "fpath+=~/.zfunc"
			case "bash":
				keyLine = ". ~/.bash_completion.d/ez"
			}
			if keyLine == "" || !strings.Contains(string(rcContents), keyLine) {
				f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return
				}
				f.WriteString(rcSnippet)
				f.Close()
			}
		}
	}

	// Write flag file
	os.MkdirAll(filepath.Join(home, ".ez"), 0755)
	os.WriteFile(flagFile, []byte(shellName), 0644)

	rootCmd.Printf("Shell completions installed for %s. Restart your terminal to enable tab completion.\n", shellName)
}
