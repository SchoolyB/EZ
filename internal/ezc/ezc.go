// Package ezc provides a Go wrapper for invoking the ezc compiler binary.
package ezc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Find locates the ezc binary using priority-ordered lookup:
// 1. EZC_PATH environment variable (explicit override, always wins)
// 2. Embedded runtime extracted to ~/.ez/runtime/<hash>/ (release builds)
// 3. Same directory as the running ez binary (legacy side-by-side install)
// 4. PATH lookup (legacy)
// 5. Known install locations (legacy)
//
// Release builds should always hit path 2; the legacy search paths are
// retained so a dev `go build ./cmd/ez` without running `make build`
// first (which leaves the embedded assets as empty stubs) still works.
func Find() (string, error) {
	// 1. Explicit override
	if p := os.Getenv("EZC_PATH"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// 2. Embedded runtime (release builds)
	if p, err := extractEmbedded(); err == nil {
		return p, nil
	}

	// 3. Same directory as ez binary
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, "ezc")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// 4. PATH lookup
	if p, err := exec.LookPath("ezc"); err == nil {
		return p, nil
	}

	// 5. Known install locations
	for _, p := range []string{"/usr/local/bin/ezc", "/usr/bin/ezc"} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("ezc compiler not found. Install it or set EZC_PATH")
}

// BuildOpts configures a build invocation.
type BuildOpts struct {
	Output     string
	OptLevel   string // "O0", "O1", "O2", "O3"
	Debug      bool
	Verbose    bool
	EmitC      bool
	NoColor    bool
	Quiet      bool   // Suppress all warnings
	QuietCodes string // Suppress specific warning codes (comma-separated)
}

// Run compiles and executes an EZ source file via ezc run.
// stdout/stderr are streamed directly to the caller's terminal.
// Returns the exit code from the compiled program.
func Run(file string, extraArgs []string) (int, error) {
	ezcPath, err := Find()
	if err != nil {
		return 1, err
	}

	args := []string{"run", file}
	args = append(args, extraArgs...)

	return execute(ezcPath, args)
}

// Build compiles an EZ source file to a native binary via ezc build.
func Build(file string, opts BuildOpts) (int, error) {
	ezcPath, err := Find()
	if err != nil {
		return 1, err
	}

	args := []string{"build", file}
	if opts.Output != "" {
		args = append(args, "-o", opts.Output)
	}
	if opts.OptLevel != "" {
		args = append(args, "-"+opts.OptLevel)
	}
	if opts.Debug {
		args = append(args, "-g")
	}
	if opts.Verbose {
		args = append(args, "-v")
	}
	if opts.EmitC {
		args = append(args, "-c")
	}
	if opts.NoColor {
		args = append(args, "--no-color")
	}
	if opts.Quiet {
		args = append(args, "--quiet")
	} else if opts.QuietCodes != "" {
		args = append(args, "--quiet", opts.QuietCodes)
	}

	return execute(ezcPath, args)
}

// Check type-checks an EZ source file without compiling.
func Check(file string, extraArgs []string) (int, error) {
	ezcPath, err := Find()
	if err != nil {
		return 1, err
	}

	args := []string{"check", file}
	args = append(args, extraArgs...)
	return execute(ezcPath, args)
}

// Version returns the ezc compiler version string.
func Version() (string, error) {
	ezcPath, err := Find()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(ezcPath, "version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get ezc version: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// execute runs the ezc binary with the given args, streaming I/O.
func execute(ezcPath string, args []string) (int, error) {
	cmd := exec.Command(ezcPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}
