package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// UpdateState stores the last update check info
type UpdateState struct {
	LastCheck     string `json:"last_check"`
	LatestVersion string `json:"latest_version"`
}

// GitHubRelease represents the GitHub API response for releases
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

const (
	githubAPIURL    = "https://api.github.com/repos/SchoolyB/EZ/releases/latest"
	updateCheckDir  = ".ez"
	updateCheckFile = "update_check"
	checkTimeout    = 2 * time.Second
)

// getUpdateStatePath returns the path to the update state file
func getUpdateStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, updateCheckDir, updateCheckFile), nil
}

// readUpdateState reads the update state from disk
func readUpdateState() (*UpdateState, error) {
	path, err := getUpdateStatePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &UpdateState{}, nil
		}
		return nil, err
	}

	var state UpdateState
	if err := json.Unmarshal(data, &state); err != nil {
		return &UpdateState{}, nil
	}
	return &state, nil
}

// writeUpdateState writes the update state to disk
func writeUpdateState(state *UpdateState) error {
	path, err := getUpdateStatePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// shouldCheckForUpdate returns true if we should check for updates (once per day)
func shouldCheckForUpdate() bool {
	state, err := readUpdateState()
	if err != nil {
		return true
	}

	if state.LastCheck == "" {
		return true
	}

	today := time.Now().Format("2006-01-02")
	return state.LastCheck != today
}

// parseVersion extracts major, minor, patch from version string like "v0.16.10" or "0.16.10"
func parseVersion(v string) (major, minor, patch int) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")

	if len(parts) >= 1 {
		fmt.Sscanf(parts[0], "%d", &major)
	}
	if len(parts) >= 2 {
		fmt.Sscanf(parts[1], "%d", &minor)
	}
	if len(parts) >= 3 {
		fmt.Sscanf(parts[2], "%d", &patch)
	}
	return
}

// isNewerVersion returns true if remote version has higher major or minor than local
func isNewerVersion(local, remote string) bool {
	localMajor, localMinor, _ := parseVersion(local)
	remoteMajor, remoteMinor, _ := parseVersion(remote)

	if remoteMajor > localMajor {
		return true
	}
	if remoteMajor == localMajor && remoteMinor > localMinor {
		return true
	}
	return false
}

// fetchLatestRelease fetches the latest release info from GitHub
func fetchLatestRelease(ctx context.Context) (*GitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", githubAPIURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "EZ-Language-Updater")

	client := &http.Client{Timeout: checkTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// CheckForUpdateAsync checks for updates and prints notice if available
// This is called at startup. It:
// 1. Checks cached state and prints notice immediately if update available
// 2. Spawns background goroutine to refresh the cache (once per day)
func CheckForUpdateAsync() {
	// First, check cached state and print notice immediately (synchronous)
	state, _ := readUpdateState()
	if state != nil && state.LatestVersion != "" {
		if isNewerVersion(Version, state.LatestVersion) {
			fmt.Printf("Note: EZ %s available (you have %s). Run `ez update` to upgrade.\n\n",
				state.LatestVersion, Version)
		}
	}

	// Then, refresh cache in background if needed (async, no printing)
	if !shouldCheckForUpdate() {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
		defer cancel()

		release, err := fetchLatestRelease(ctx)
		if err != nil {
			return // Silently fail - don't bother user with network errors
		}

		// Update state (will be used on next run)
		newState := &UpdateState{
			LastCheck:     time.Now().Format("2006-01-02"),
			LatestVersion: release.TagName,
		}
		writeUpdateState(newState)
	}()
}

// runUpdate runs the interactive update command
func runUpdate() {
	fmt.Printf("Current version: %s\n", Version)
	fmt.Println("Checking for updates...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	release, err := fetchLatestRelease(ctx)
	if err != nil {
		fmt.Printf("Error checking for updates: %v\n", err)
		return
	}

	// Update state file so future runs don't re-check today
	state := &UpdateState{
		LastCheck:     time.Now().Format("2006-01-02"),
		LatestVersion: release.TagName,
	}
	writeUpdateState(state)

	fmt.Printf("Latest version:  %s\n", release.TagName)

	if !isNewerVersion(Version, release.TagName) {
		fmt.Println("\nYou're already on the latest version!")
		return
	}

	// Show changelog
	fmt.Println("\nWhat's new:")
	fmt.Println(strings.Repeat("-", 40))
	// Truncate changelog if too long
	changelog := release.Body
	lines := strings.Split(changelog, "\n")
	if len(lines) > 20 {
		changelog = strings.Join(lines[:20], "\n") + "\n... (truncated)"
	}
	fmt.Println(changelog)
	fmt.Println(strings.Repeat("-", 40))

	// Prompt for confirmation
	fmt.Printf("\nUpgrade to %s? (y/N): ", release.TagName)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println("Update cancelled.")
		return
	}

	// Find the right asset for this OS/arch
	assetName := getAssetName()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		fmt.Printf("Error: No binary available for %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Println("You may need to build from source: go install github.com/marshallburns/ez/cmd/ez@latest")
		return
	}

	// Download and install
	fmt.Printf("Downloading %s...\n", assetName)
	if err := downloadAndInstall(downloadURL); err != nil {
		fmt.Printf("Error during update: %v\n", err)
		return
	}

	fmt.Printf("\nSuccessfully updated to %s!\n", release.TagName)
	fmt.Println("Restart your terminal or run `ez version` to verify.")
}

// getAssetName returns the expected binary name for this OS/arch
func getAssetName() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Map to common naming conventions
	switch os {
	case "darwin":
		os = "darwin"
	case "linux":
		os = "linux"
	case "windows":
		os = "windows"
	}

	switch arch {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	}

	name := fmt.Sprintf("ez-%s-%s", os, arch)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// downloadAndInstall downloads the new binary and replaces the current one
func downloadAndInstall(url string) error {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Download to temp file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp file in same directory as executable (for atomic rename)
	dir := filepath.Dir(execPath)
	tmpFile, err := os.CreateTemp(dir, "ez-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Download to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write update: %w", err)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Backup current executable
	backupPath := execPath + ".backup"
	if err := os.Rename(execPath, backupPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Move new binary into place
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Try to restore backup
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Remove backup
	os.Remove(backupPath)

	return nil
}
