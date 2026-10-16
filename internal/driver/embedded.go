// embedded.go — Embeds the grayc binary, libgrayrt.a, and runtime/stdlib sources
// into the gray CLI binary and extracts them on first use to ~/.gray/runtime/.
//
// Author:  Marshall A Burns (@SchoolyB)
// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

package driver

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

//go:embed runtime/grayc
var embeddedGrayc []byte

//go:embed runtime/libgrayrt.a
var embeddedLibgrayrt []byte

//go:embed all:runtime/src
var embeddedSrc embed.FS

// ErrNoEmbed indicates the embedded runtime assets are empty stubs (a
// `go build ./cli` without a prior `make build` stage), so callers
// should fall back to a path search.
var ErrNoEmbed = errors.New("embedded grayc runtime is empty (dev build)")

var (
	extractOnce        sync.Once
	extractedGraycPath string
	extractErr         error
)

// embedTag is the 16-hex-char content hash of every embedded input: grayc,
// libgrayrt.a, and the src/ tree (file list and contents). `make embed-tag`
// computes it and the build injects it via -ldflags, so a fresh process never
// hashes the assets and a changed source file always names a new extraction
// directory. Empty outside `make build` (e.g. `go test`), where
// computeEmbedTag produces the same value at runtime.
var embedTag string

// computeEmbedTag mirrors the Makefile's embed-tag target: sha256 of grayc,
// libgrayrt.a, the byte-sorted "./"-prefixed src/ file list (one per line),
// then each file's contents in that order.
func computeEmbedTag() (string, error) {
	var names []string
	err := fs.WalkDir(embeddedSrc, "runtime/src", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			names = append(names, "./"+path[len("runtime/src/"):])
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(names)

	hash := sha256.New()
	hash.Write(embeddedGrayc)
	hash.Write(embeddedLibgrayrt)
	for _, name := range names {
		hash.Write([]byte(name + "\n"))
	}
	for _, name := range names {
		data, err := embeddedSrc.ReadFile("runtime/src/" + name[2:])
		if err != nil {
			return "", err
		}
		hash.Write(data)
	}
	return hex.EncodeToString(hash.Sum(nil))[:16], nil
}

// extractEmbedded materialises the embedded grayc binary, libgrayrt.a, and
// runtime/stdlib source files into a per-content-hash subdirectory of
// ~/.gray/runtime so multiple installs don't collide and a version bump
// forces a clean extraction. Safe to call repeatedly; the work runs
// once per process via sync.Once.
func extractEmbedded() (string, error) {
	extractOnce.Do(func() {
		// libgrayrt.a may legitimately be an empty stub: the Windows
		// bootstrap ships a compiler without the runtime archive, which is
		// enough for check, fmt, doc, and --emit-c. Only a missing compiler
		// means "this is a dev build, fall back to a path search".
		if len(embeddedGrayc) == 0 {
			extractErr = ErrNoEmbed
			return
		}

		// Content-address the embedded inputs so a new gray binary lands in a
		// fresh directory instead of reusing a stale extraction from an
		// older install.
		tag := embedTag
		if tag == "" {
			var err error
			if tag, err = computeEmbedTag(); err != nil {
				extractErr = fmt.Errorf("cannot hash embedded runtime: %w", err)
				return
			}
		}

		home, err := os.UserHomeDir()
		if err != nil {
			extractErr = fmt.Errorf("cannot resolve home dir: %w", err)
			return
		}
		dir := filepath.Join(home, ".gray", "runtime", tag)

		// Fast path: a prior process already extracted everything into this
		// content-addressed directory and left the marker. Skip the tree
		// walk that would otherwise stat every embedded source file.
		marker := filepath.Join(dir, ".extracted")
		if _, err := os.Stat(marker); err == nil {
			extractedGraycPath = filepath.Join(dir, graycBinaryName())
			return
		}

		if err := os.MkdirAll(dir, 0o755); err != nil {
			extractErr = fmt.Errorf("cannot create runtime dir %s: %w", dir, err)
			return
		}

		graycDest := filepath.Join(dir, graycBinaryName())
		libDest := filepath.Join(dir, "libgrayrt.a")

		if err := writeIfMissing(graycDest, embeddedGrayc, 0o755); err != nil {
			extractErr = err
			return
		}
		if len(embeddedLibgrayrt) > 0 {
			if err := writeIfMissing(libDest, embeddedLibgrayrt, 0o644); err != nil {
				extractErr = err
				return
			}
		}

		// Extract runtime and stdlib source files so grayc can find its
		// headers via the "development layout" search (src/ relative to
		// binary).
		if err := extractSourceTree(dir); err != nil {
			extractErr = err
			return
		}

		// Drop the marker last so a crash mid-extraction is retried rather
		// than trusted by the fast path above.
		if err := os.WriteFile(marker, nil, 0o644); err != nil {
			extractErr = fmt.Errorf("cannot finalize runtime dir %s: %w", dir, err)
			return
		}

		extractedGraycPath = graycDest
	})
	return extractedGraycPath, extractErr
}

// extractSourceTree walks the embedded src/ tree and writes each file
// into dir/src/{runtime,stdlib}/. Skips files that already exist with
// the correct size (same logic as writeIfMissing).
func extractSourceTree(dir string) error {
	return fs.WalkDir(embeddedSrc, "runtime/src", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// path is like "runtime/src/runtime/runtime.h" or "runtime/src/stdlib/fmt.c"
		// Strip the leading "runtime/" prefix to get "src/runtime/..." on disk.
		rel := path[len("runtime/"):]
		dest := filepath.Join(dir, rel)

		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}

		data, err := embeddedSrc.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}
		return writeIfMissing(dest, data, 0o644)
	})
}

// writeIfMissing writes data to path with the given mode unless a file of
// the same size already exists there. The content-hash tag in the parent
// directory already guarantees uniqueness per build, so a size check is
// sufficient to detect a successful prior extraction.
func writeIfMissing(path string, data []byte, mode fs.FileMode) error {
	if st, err := os.Stat(path); err == nil && st.Size() == int64(len(data)) {
		return nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("install %s: %w", path, err)
	}
	return nil
}
