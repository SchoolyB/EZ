package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/marshallburns/ez/pkg/errors"
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
	githubAPIURL         = "https://api.github.com/repos/SchoolyB/EZ/releases/latest"
	githubReleasesURL    = "https://api.github.com/repos/SchoolyB/EZ/releases"
	updateCheckDir       = ".ez"
	updateCheckFile      = "update_check"
	checkTimeout         = 2 * time.Second
	maxChangelogVersions = 10 // Maximum number of versions to show in changelog
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

// isNewerVersion returns true if remote version is higher than local
func isNewerVersion(local, remote string) bool {
	localMajor, localMinor, localPatch := parseVersion(local)
	remoteMajor, remoteMinor, remotePatch := parseVersion(remote)

	if remoteMajor > localMajor {
		return true
	}
	if remoteMajor == localMajor && remoteMinor > localMinor {
		return true
	}
	if remoteMajor == localMajor && remoteMinor == localMinor && remotePatch > localPatch {
		return true
	}
	return false
}

// isVersionInRange returns true if version is between fromVersion (exclusive) and toVersion (inclusive)
// i.e., fromVersion < version <= toVersion
func isVersionInRange(version, fromVersion, toVersion string) bool {
	// version must be greater than fromVersion
	// isNewerVersion(local, remote) returns true if remote > local
	if !isNewerVersion(fromVersion, version) {
		return false
	}
	// version must be less than or equal to toVersion
	// i.e., version is NOT greater than toVersion
	// NOT(version > toVersion) = NOT(isNewerVersion(toVersion, version))
	return !isNewerVersion(toVersion, version)
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

// fetchAllReleases fetches all releases from GitHub
func fetchAllReleases(ctx context.Context) ([]GitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", githubReleasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "EZ-Language-Updater")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return releases, nil
}

// getReleasesInRange filters releases to those between currentVersion (exclusive) and latestVersion (inclusive)
func getReleasesInRange(releases []GitHubRelease, currentVersion, latestVersion string) []GitHubRelease {
	var result []GitHubRelease
	for _, release := range releases {
		if isVersionInRange(release.TagName, currentVersion, latestVersion) {
			result = append(result, release)
		}
	}
	return result
}

// ParsedChangelog holds the parsed features and bug fixes from a release body
type ParsedChangelog struct {
	Features []string
	BugFixes []string
}

// parseReleaseBody extracts Features and Bug Fixes sections from a release body
func parseReleaseBody(body string) ParsedChangelog {
	result := ParsedChangelog{}
	lines := strings.Split(body, "\n")

	var currentSection string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for section headers
		if strings.HasPrefix(trimmed, "### Features") {
			currentSection = "features"
			continue
		}
		if strings.HasPrefix(trimmed, "### Bug Fixes") {
			currentSection = "bugfixes"
			continue
		}
		// Any other ### header ends the current section
		if strings.HasPrefix(trimmed, "###") {
			currentSection = ""
			continue
		}

		// Parse list items
		if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") {
			item := strings.TrimPrefix(trimmed, "* ")
			item = strings.TrimPrefix(item, "- ")

			// Clean up the item - formats:
			// **scope:** description ([#issue](url)) ([hash](url))
			// **scope:** description ([hash](url))
			// scope: description (hash)

			// Remove markdown bold markers: **text:** -> text:
			item = strings.ReplaceAll(item, "**", "")

			// Decode HTML entities
			item = strings.ReplaceAll(item, "&lt;", "<")
			item = strings.ReplaceAll(item, "&gt;", ">")
			item = strings.ReplaceAll(item, "&amp;", "&")

			// Remove ", closes [#xxx](url)" suffixes
			if idx := strings.Index(item, ", closes [#"); idx != -1 {
				item = item[:idx]
			}

			// Remove issue references like (#123) or #123
			for {
				if idx := strings.Index(item, " (#"); idx != -1 {
					if endIdx := strings.Index(item[idx:], ")"); endIdx != -1 {
						item = item[:idx] + item[idx+endIdx+1:]
						continue
					}
				}
				if idx := strings.Index(item, " #"); idx != -1 {
					// Check if followed by digits
					endIdx := idx + 2
					for endIdx < len(item) && item[endIdx] >= '0' && item[endIdx] <= '9' {
						endIdx++
					}
					if endIdx > idx+2 {
						item = item[:idx] + item[endIdx:]
						continue
					}
				}
				break
			}

			// Extract commit hash - it's the last ([hash](url)) or (hash)
			var commitHash string

			// Try to find markdown link hash: ([hash](url))
			// Look for pattern starting from end
			if lastParen := strings.LastIndex(item, ")"); lastParen != -1 {
				// Find the matching opening
				depth := 1
				start := lastParen - 1
				for start >= 0 && depth > 0 {
					if item[start] == ')' {
						depth++
					} else if item[start] == '(' {
						depth--
					}
					if depth > 0 {
						start--
					}
				}
				if start >= 0 && depth == 0 {
					parenContent := item[start+1 : lastParen]
					// Check if it's a markdown link [hash](url)
					if strings.HasPrefix(parenContent, "[") && strings.Contains(parenContent, "](") {
						// Extract just the hash
						if hashEnd := strings.Index(parenContent, "]"); hashEnd != -1 {
							commitHash = parenContent[1:hashEnd]
						}
						// Remove this from item
						item = strings.TrimSpace(item[:start])
					} else if !strings.HasPrefix(parenContent, "#") && !strings.Contains(parenContent, "://") {
						// It's a simple (hash) without markdown
						commitHash = parenContent
						item = strings.TrimSpace(item[:start])
					}
				}
			}

			// Remove issue link: ([#123](url)) if still present
			if idx := strings.Index(item, " ([#"); idx != -1 {
				if endIdx := strings.Index(item[idx:], "])"); endIdx != -1 {
					// Find the actual end including the outer )
					fullEnd := idx + endIdx + 2
					if fullEnd < len(item) && item[fullEnd] == ')' {
						fullEnd++
					}
					item = item[:idx] + item[fullEnd:]
				}
			}

			// Clean up markdown links in text: [@module](url) -> @module
			for strings.Contains(item, "](") {
				start := strings.Index(item, "[")
				if start == -1 {
					break
				}
				end := strings.Index(item[start:], ")")
				if end == -1 {
					break
				}
				end += start
				linkText := item[start+1 : strings.Index(item[start:], "]")+start]
				item = item[:start] + linkText + item[end+1:]
			}

			// Add commit hash back if we found one
			if commitHash != "" {
				item = strings.TrimSpace(item) + " (" + commitHash + ")"
			}

			item = strings.TrimSpace(item)

			if currentSection == "features" && item != "" {
				result.Features = append(result.Features, item)
			} else if currentSection == "bugfixes" && item != "" {
				result.BugFixes = append(result.BugFixes, item)
			}
		}
	}

	return result
}

// formatChangelog formats the changelog for display
func formatChangelog(releases []GitHubRelease, currentVersion, latestVersion string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\nUpdating from %s -> %s\n", currentVersion, latestVersion))
	sb.WriteString("\nWhat's new since your version:\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")

	// Limit to maxChangelogVersions
	displayReleases := releases
	truncated := false
	if len(releases) > maxChangelogVersions {
		displayReleases = releases[:maxChangelogVersions]
		truncated = true
	}

	for _, release := range displayReleases {
		parsed := parseReleaseBody(release.Body)

		// Skip releases with no features or bug fixes
		if len(parsed.Features) == 0 && len(parsed.BugFixes) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("\n%s%s%s\n", errors.Bold, release.TagName, errors.Reset))

		if len(parsed.Features) > 0 {
			sb.WriteString("  Features:\n")
			for _, f := range parsed.Features {
				sb.WriteString(fmt.Sprintf("    - %s\n", f))
			}
		}

		if len(parsed.BugFixes) > 0 {
			sb.WriteString("  Bug Fixes:\n")
			for _, b := range parsed.BugFixes {
				sb.WriteString(fmt.Sprintf("    - %s\n", b))
			}
		}
	}

	if truncated {
		remaining := len(releases) - maxChangelogVersions
		sb.WriteString(fmt.Sprintf("\n... and %d more version(s)\n", remaining))
	}

	sb.WriteString("\n" + strings.Repeat("-", 40))

	return sb.String()
}

// CheckForUpdateAsync checks for updates and prints notice if available
// This is called at startup. It:
// 1. If cache is fresh, use cached state to print notice
// 2. If cache is stale/empty, do a synchronous check and print notice
func CheckForUpdateAsync() {
	state, _ := readUpdateState()

	// If we have cached state and it's fresh (checked today), use it
	if state != nil && state.LatestVersion != "" && !shouldCheckForUpdate() {
		if isNewerVersion(Version, state.LatestVersion) {
			fmt.Printf("Note: EZ %s available (you have %s). Run `ez update` to upgrade.\n\n",
				state.LatestVersion, Version)
		}
		return
	}

	// Cache is stale or empty - do a synchronous check
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	release, err := fetchLatestRelease(ctx)
	if err != nil {
		// Network error - fall back to cached state if available
		if state != nil && state.LatestVersion != "" {
			if isNewerVersion(Version, state.LatestVersion) {
				fmt.Printf("Note: EZ %s available (you have %s). Run `ez update` to upgrade.\n\n",
					state.LatestVersion, Version)
			}
		}
		return
	}

	// Update cache
	newState := &UpdateState{
		LastCheck:     time.Now().Format("2006-01-02"),
		LatestVersion: release.TagName,
	}
	writeUpdateState(newState)

	// Print notification if newer version available
	if isNewerVersion(Version, release.TagName) {
		fmt.Printf("Note: EZ %s available (you have %s). Run `ez update` to upgrade.\n\n",
			release.TagName, Version)
	}
}

// runUpdate runs the interactive update command
// If args contains "--confirm" followed by a URL, it skips confirmation and installs directly
// (used when re-executing with sudo)
func runUpdate(confirm bool, url string) {
	// Check for --confirm flag (used by sudo re-exec)
	if confirm {
		downloadURL := url
		fmt.Printf("Installing update...\n")
		if err := downloadAndInstall(downloadURL); err != nil {
			fmt.Printf("Error during update: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(asciiBanner)
		fmt.Println("\nSuccessfully updated!")
		fmt.Println("Restart your terminal or run `ez version` to verify.")
		return
	}

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

	// Fetch all releases and show changelog for versions between current and latest
	allReleases, err := fetchAllReleases(ctx)
	if err != nil {
		// Fall back to old behavior if we can't fetch all releases
		fmt.Println("\nWhat's new:")
		fmt.Println(strings.Repeat("-", 40))
		changelog := release.Body
		lines := strings.Split(changelog, "\n")
		if len(lines) > 20 {
			changelog = strings.Join(lines[:20], "\n") + "\n... (truncated)"
		}
		fmt.Println(changelog)
		fmt.Println(strings.Repeat("-", 40))
	} else {
		releasesInRange := getReleasesInRange(allReleases, Version, release.TagName)
		fmt.Print(formatChangelog(releasesInRange, Version, release.TagName))
	}

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

	fmt.Print(asciiBanner)
	fmt.Printf("\n%s%s%s\n", errors.Bold, release.TagName, errors.Reset)
	fmt.Println("\nSuccessfully updated!")
	fmt.Println("Restart your terminal or run `ez version` to verify.")
}

// getAssetName returns the expected archive name for this OS/arch
func getAssetName() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	name := fmt.Sprintf("ez-%s-%s", osName, arch)
	if runtime.GOOS == "windows" {
		name += ".zip"
	} else {
		name += ".tar.gz"
	}
	return name
}

// checkWritePermission checks if we can write to the executable's directory
func checkWritePermission(path string) bool {
	dir := filepath.Dir(path)
	testFile := filepath.Join(dir, ".ez-update-test")
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(testFile)
	return true
}

// downloadAndInstall downloads the archive and extracts/installs the binary
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

	// Check if we have write permission
	if !checkWritePermission(execPath) {
		// Need elevated permissions - re-exec with sudo
		if runtime.GOOS == "windows" {
			return fmt.Errorf("permission denied. Please run as Administrator")
		}
		fmt.Println("Root permissions required. Running with sudo...")
		// Re-run the update command with sudo
		cmd := exec.Command("sudo", os.Args[0], "update", "--confirm", url)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return doInstall(url, execPath)
}

// doInstall performs the actual download and installation
func doInstall(url, execPath string) error {
	// Download archive to temp file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp directory for extraction
	tmpDir, err := os.MkdirTemp("", "ez-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download archive
	archivePath := filepath.Join(tmpDir, "archive")
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}

	_, err = io.Copy(archiveFile, resp.Body)
	archiveFile.Close()
	if err != nil {
		return fmt.Errorf("failed to download archive: %w", err)
	}

	// Extract binary from archive
	var binaryPath string
	if runtime.GOOS == "windows" {
		binaryPath, err = extractZip(archivePath, tmpDir)
	} else {
		binaryPath, err = extractTarGz(archivePath, tmpDir)
	}
	if err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Make executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Backup current executable
	backupPath := execPath + ".backup"
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Copy new binary into place (can't rename across filesystems)
	if err := copyFile(binaryPath, execPath); err != nil {
		// Try to restore backup
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Make the new binary executable
	if err := os.Chmod(execPath, 0755); err != nil {
		// Try to restore backup
		os.Remove(execPath)
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Remove backup
	os.Remove(backupPath)

	return nil
}

// sanitizeArchivePath validates that the extracted file path is within the destination directory.
// This prevents path traversal attacks (CWE-22) from malicious archives.
func sanitizeArchivePath(destDir, filename string) (string, error) {
	// Clean the filename to remove any path traversal sequences
	cleanName := filepath.Clean(filepath.Base(filename))

	// Reject any filename that is empty, a dot, or contains path separators after cleaning
	if cleanName == "" || cleanName == "." || cleanName == ".." {
		return "", fmt.Errorf("invalid filename in archive: %s", filename)
	}

	// Construct the destination path
	destPath := filepath.Join(destDir, cleanName)

	// Resolve to absolute path and verify it's within destDir
	absDestPath, err := filepath.Abs(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve destination directory: %w", err)
	}

	// Ensure the destination path starts with the destination directory
	// Add separator to prevent matching partial directory names (e.g., /tmp/ez vs /tmp/ez-malicious)
	if !strings.HasPrefix(absDestPath, absDestDir+string(filepath.Separator)) && absDestPath != absDestDir {
		return "", fmt.Errorf("path traversal detected: %s", filename)
	}

	return destPath, nil
}

// extractTarGz extracts the ez binary from a .tar.gz archive
func extractTarGz(archivePath, destDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Look for the ez binary (might be "ez" or in a subdirectory)
		name := filepath.Base(header.Name)
		if name == "ez" || name == "ez.exe" {
			// Validate the destination path to prevent path traversal attacks
			destPath, err := sanitizeArchivePath(destDir, name)
			if err != nil {
				return "", err
			}

			outFile, err := os.Create(destPath)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return "", err
			}
			outFile.Close()
			return destPath, nil
		}
	}

	return "", fmt.Errorf("ez binary not found in archive")
}

// extractZip extracts the ez binary from a .zip archive
func extractZip(archivePath, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == "ez" || name == "ez.exe" {
			// Validate the destination path to prevent path traversal attacks
			destPath, err := sanitizeArchivePath(destDir, name)
			if err != nil {
				return "", err
			}

			rc, err := f.Open()
			if err != nil {
				return "", err
			}

			outFile, err := os.Create(destPath)
			if err != nil {
				rc.Close()
				return "", err
			}

			_, err = io.Copy(outFile, rc)
			outFile.Close()
			rc.Close()
			if err != nil {
				return "", err
			}
			return destPath, nil
		}
	}

	return "", fmt.Errorf("ez binary not found in archive")
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
