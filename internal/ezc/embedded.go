// Embedded runtime assets — staged into runtime/ by `make build`.
//
// At build time the root Makefile compiles ezc and libezrt.a, copies them
// into internal/ezc/runtime/, and copies the runtime/stdlib source files
// into internal/ezc/runtime/src/{runtime,stdlib}/. Then `go build` bakes
// everything into the final `ez` binary. Dev builds without `make build`
// still compile because committed stub files keep the go:embed directives
// satisfied; the extractor detects the stubs by their zero length and
// returns ErrNoEmbed so Find() falls back to the path search.
package ezc

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

//go:embed runtime/ezc
var embeddedEzc []byte

//go:embed runtime/libezrt.a
var embeddedLibezrt []byte

//go:embed all:runtime/src
var embeddedSrc embed.FS

// ErrNoEmbed indicates the embedded runtime assets are empty stubs (a
// `go build ./cmd/ez` without a prior `make build` stage), so callers
// should fall back to a path search.
var ErrNoEmbed = errors.New("embedded ezc runtime is empty (dev build)")

var (
	extractOnce      sync.Once
	extractedEzcPath string
	extractErr       error
)

// extractEmbedded materialises the embedded ezc binary, libezrt.a, and
// runtime/stdlib source files into a per-content-hash subdirectory of
// ~/.ez/runtime so multiple installs don't collide and a version bump
// forces a clean extraction. Safe to call repeatedly; the work runs
// once per process via sync.Once.
func extractEmbedded() (string, error) {
	extractOnce.Do(func() {
		if len(embeddedEzc) == 0 || len(embeddedLibezrt) == 0 {
			extractErr = ErrNoEmbed
			return
		}

		// Content-address the pair so a new ez binary (built against new
		// compiler artifacts) lands in a fresh directory instead of
		// reusing a stale extraction from an older install.
		h := sha256.New()
		h.Write(embeddedEzc)
		h.Write(embeddedLibezrt)
		tag := hex.EncodeToString(h.Sum(nil))[:16]

		home, err := os.UserHomeDir()
		if err != nil {
			extractErr = fmt.Errorf("cannot resolve home dir: %w", err)
			return
		}
		dir := filepath.Join(home, ".ez", "runtime", tag)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			extractErr = fmt.Errorf("cannot create runtime dir %s: %w", dir, err)
			return
		}

		ezcName := "ezc"
		if runtime.GOOS == "windows" {
			ezcName = "ezc.exe"
		}
		ezcDest := filepath.Join(dir, ezcName)
		libDest := filepath.Join(dir, "libezrt.a")

		if err := writeIfMissing(ezcDest, embeddedEzc, 0o755); err != nil {
			extractErr = err
			return
		}
		if err := writeIfMissing(libDest, embeddedLibezrt, 0o644); err != nil {
			extractErr = err
			return
		}

		// Extract runtime and stdlib source files so ezc can find its
		// headers via the "development layout" search (src/ relative to
		// binary).
		if err := extractSourceTree(dir); err != nil {
			extractErr = err
			return
		}

		extractedEzcPath = ezcDest
	})
	return extractedEzcPath, extractErr
}

// extractSourceTree walks the embedded src/ tree and writes each file
// into dir/src/{runtime,stdlib}/. Skips files that already exist with
// the correct size (same logic as writeIfMissing).
func extractSourceTree(dir string) error {
	return fs.WalkDir(embeddedSrc, "runtime/src", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// path is like "runtime/src/runtime/ez_runtime.h" or "runtime/src/stdlib/ez_fmt.c"
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
