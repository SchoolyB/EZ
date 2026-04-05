package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

)

// Version information - injected at build time via ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
)

const asciiBanner = `
 _____ ____
| ____|__  |
|  _|   / /
| |___ / /_
|_____/____|
Programming made EZ
`

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getVersionString() string {
	buf := bytes.Buffer{}
	buf.WriteString(asciiBanner)
	fmt.Fprintf(&buf, "\n\033[1mEZ %s\033[0m\n", Version)
	fmt.Fprintf(&buf, "Built: %s\n", BuildTime)

	// Always fetch fresh version info when user explicitly runs 'ez version'
	var latestVersion string

	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	release, err := fetchLatestRelease(ctx)
	if err != nil {
		// Network error - fall back to cached state if available
		state, _ := readUpdateState()
		if state != nil && state.LatestVersion != "" {
			latestVersion = state.LatestVersion
		}
	} else {
		// Update cache with fresh data
		newState := &UpdateState{
			LastCheck:     time.Now().Format("2006-01-02"),
			LatestVersion: release.TagName,
		}
		writeUpdateState(newState)
		latestVersion = release.TagName
	}

	// Always display latest version if we have it
	if latestVersion != "" {
		fmt.Fprintf(&buf, "Latest: %s\n", latestVersion)
		if isNewerVersion(Version, latestVersion) {
			fmt.Fprintf(&buf, "\nUpdate available! Run `ez update` to upgrade.\n")
		}
	}

	fmt.Fprintf(&buf, "Copyright (c) 2025-Present Marshall A Burns")
	return buf.String()
}
