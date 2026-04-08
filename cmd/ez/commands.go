package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/marshallburns/ez/internal/ezc"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check [file.ez | directory]",
	Short: "Type-check a file or project without compiling",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Allow directories for project-wide check
		info, statErr := os.Stat(args[0])
		if statErr == nil && !info.IsDir() && !strings.HasSuffix(args[0], ".ez") {
			fmt.Fprintf(os.Stderr, "error: '%s' is not a valid EZ source file — expected a .ez file\n", args[0])
			os.Exit(1)
		}
		var extraArgs []string
		quiet, _ := cmd.Flags().GetString("quiet")
		if quiet == "all" {
			extraArgs = append(extraArgs, "--quiet")
		} else if quiet != "" {
			extraArgs = append(extraArgs, "--quiet", quiet)
		}
		code, err := ezc.Check(args[0], extraArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(code)
	},
}

var buildCmd = &cobra.Command{
	Use:   "build [file.ez]",
	Short: "Compile an EZ source file to a native binary",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !strings.HasSuffix(args[0], ".ez") {
			fmt.Fprintf(os.Stderr, "error: '%s' is not a valid EZ source file — expected a .ez file\n", args[0])
			os.Exit(1)
		}
		output, _ := cmd.Flags().GetString("output")
		verbose, _ := cmd.Flags().GetBool("verbose")
		emitC, _ := cmd.Flags().GetBool("emit-c")
		quiet, _ := cmd.Flags().GetString("quiet")

		opts := ezc.BuildOpts{
			Output:  output,
			Verbose: verbose,
			EmitC:   emitC,
		}
		if quiet == "all" {
			opts.Quiet = true
		} else if quiet != "" {
			opts.QuietCodes = quiet
		}
		code, err := ezc.Build(args[0], opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(code)
	},
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Start interactive REPL mode",
	Run: func(cmd *cobra.Command, args []string) {
		startREPL()
	},
}

var updateCmd = &cobra.Command{
	Use:   "update [url]",
	Short: "Check for updates and upgrade EZ",
	Long:  "Check for updates and upgrade EZ. Optionally specify a custom URL for the update source.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		confirm, _ := cmd.Flags().GetBool("confirm")
		var url string
		if len(args) > 0 {
			url = args[0]
		}
		runUpdate(confirm, url)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd.Printf("%s\n", getVersionString())
	},
}

var docCmd = &cobra.Command{
	Use:   "doc <path>",
	Short: "Generate documentation from #doc attributes",
	Long: `Generate markdown documentation from #doc attributes in EZ source files.

Examples:
  ez doc .              Generate docs for current directory (no recursion)
  ez doc ./...          Generate docs recursively from current directory
  ez doc ./src          Generate docs for src directory only
  ez doc ./src/...      Generate docs recursively from src directory
  ez doc file.ez        Generate docs for a single file
  ez doc a.ez b.ez      Generate docs for multiple files

Output is written to DOCS.md in the current working directory.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		generateDocs(args)
	},
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Print system info for bug reports",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("EZ Bug Report Info")
		fmt.Println("======================")

		// EZ version
		fmt.Printf("EZ Version:  %s\n", Version)

		// Commit
		// Version string from ldflags contains the commit hash (e.g., v3.0.0-425-gabcdef1)
		commit := "unknown"
		// Version format: v3.0.0-NNN-gabcdef1 or v3.0.0-NNN-gabcdef1-dirty
		cleanVer := strings.TrimSuffix(Version, "-dirty")
		parts := strings.Split(cleanVer, "-")
		if len(parts) >= 3 {
			hash := parts[len(parts)-1]
			if len(hash) > 1 && hash[0] == 'g' {
				commit = hash[1:]
			}
		}
		fmt.Printf("Commit:      %s\n", commit)

		// OS and architecture
		fmt.Printf("OS:          %s/%s\n", runtime.GOOS, runtime.GOARCH)

		// RAM
		memCmd := exec.Command("sysctl", "-n", "hw.memsize")
		if runtime.GOOS == "linux" {
			memCmd = exec.Command("sh", "-c", "grep MemTotal /proc/meminfo | awk '{print $2 * 1024}'")
		}
		if out, err := memCmd.Output(); err == nil {
			s := strings.TrimSpace(string(out))
			var bytes uint64
			fmt.Sscanf(s, "%d", &bytes)
			gb := float64(bytes) / (1024 * 1024 * 1024)
			fmt.Printf("RAM:         %.0f GB\n", gb)
		} else {
			fmt.Printf("RAM:         unknown\n")
		}
	},
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run the full EZ test suite",
	Long: `Run all tests: Go tooling tests, compiler unit tests, compiler e2e tests,
compiler integration tests, and CLI integration tests.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Find project root relative to the ez binary
		exePath, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot locate ez binary: %v\n", err)
			os.Exit(1)
		}
		root := filepath.Dir(exePath)

		// Verify we're in the project root by checking for scripts/
		if _, err := os.Stat(filepath.Join(root, "scripts")); os.IsNotExist(err) {
			// Fall back to current working directory
			root, _ = os.Getwd()
			if _, err := os.Stat(filepath.Join(root, "scripts")); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "error: cannot find project root — run from the EZ project directory\n")
				os.Exit(1)
			}
		}

		const (
			green = "\033[0;32m"
			red   = "\033[0;31m"
			bold  = "\033[1m"
			reset = "\033[0m"
			dim   = "\033[2m"
		)

		type suiteResult struct {
			name   string
			ok     bool
			passed int
			failed int
		}

		// Strip ANSI escape codes before parsing test counts
		ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*m`)

		// Regex patterns for parsing test counts from output
		// C unit/e2e tests: "N passed, N failed (N total)"
		cTestRe := regexp.MustCompile(`(\d+)\s+passed.*?(\d+)\s+failed`)
		// Integration tests (run_tests.sh): "Passed:  N" and "Failed:  N"
		intPassRe := regexp.MustCompile(`Passed:\s+(\d+)`)
		intFailRe := regexp.MustCompile(`Failed:\s+(\d+)`)
		// Go tests: "--- PASS:" and "--- FAIL:" lines
		goPassRe := regexp.MustCompile(`--- PASS:`)
		goFailRe := regexp.MustCompile(`--- FAIL:`)

		steps := []struct {
			name  string
			cmd   string
			args  []string
			dir   string
			parse string // "go", "c", or "integration"
		}{
			{"Go tooling tests", "go", []string{"test", "-v", "./pkg/lineeditor/..."}, root, "go"},
			{"Compiler unit tests", "make", []string{"test-unit"}, filepath.Join(root, "ezc"), "c"},
			{"Compiler e2e tests", "make", []string{"test-e2e"}, filepath.Join(root, "ezc"), "c"},
			{"Integration tests", "bash", []string{filepath.Join(root, "scripts", "run_tests.sh")}, root, "integration"},
		}

		results := make([]suiteResult, 0, len(steps))

		for _, step := range steps {
			fmt.Printf("\n%s=== %s ===%s\n", bold, step.name, reset)

			var buf bytes.Buffer
			tee := io.MultiWriter(os.Stdout, &buf)

			c := exec.Command(step.cmd, step.args...)
			c.Dir = step.dir
			c.Stdout = tee
			c.Stderr = tee

			runErr := c.Run()
			ok := runErr == nil
			output := ansiRe.ReplaceAllString(buf.String(), "")

			sr := suiteResult{name: step.name, ok: ok}

			// Parse test counts from captured output
			switch step.parse {
			case "go":
				sr.passed = len(goPassRe.FindAllString(output, -1))
				sr.failed = len(goFailRe.FindAllString(output, -1))
			case "c":
				// May have multiple summary lines (one per test binary); sum them
				for _, m := range cTestRe.FindAllStringSubmatch(output, -1) {
					p, _ := strconv.Atoi(m[1])
					f, _ := strconv.Atoi(m[2])
					sr.passed += p
					sr.failed += f
				}
			case "integration":
				if m := intPassRe.FindStringSubmatch(output); len(m) > 1 {
					sr.passed, _ = strconv.Atoi(m[1])
				}
				if m := intFailRe.FindStringSubmatch(output); len(m) > 1 {
					sr.failed, _ = strconv.Atoi(m[1])
				}
			}

			results = append(results, sr)
		}

		// Print unified summary
		fmt.Printf("\n%s══════════════════════════════════════%s\n", dim, reset)
		fmt.Printf("  %sEZ Test Results%s\n", bold, reset)
		fmt.Printf("%s══════════════════════════════════════%s\n\n", dim, reset)

		totalPassed := 0
		totalFailed := 0
		anyFailed := false

		for _, sr := range results {
			totalPassed += sr.passed
			totalFailed += sr.failed
			label := fmt.Sprintf("%s%sPASS%s", bold, green, reset)
			if !sr.ok {
				label = fmt.Sprintf("%s%sFAIL%s", bold, red, reset)
				anyFailed = true
			}
			fmt.Printf("  %s  %-24s (%d passed, %d failed)\n", label, sr.name, sr.passed, sr.failed)
		}

		fmt.Printf("\n  %sTotal: %d passed, %d failed%s\n", bold, totalPassed, totalFailed, reset)
		fmt.Printf("%s══════════════════════════════════════%s\n", dim, reset)

		if anyFailed {
			os.Exit(1)
		}
	},
}

var rootCmd = &cobra.Command{
	Use:     "ez [file.ez]",
	Short:   "EZ Programming Language",
	Long:    "A statically-typed programming language that compiles to native binaries.",
	Args:    cobra.ArbitraryArgs,
	Version: getVersionString(),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			fmt.Print(asciiBanner)
			return cmd.Help()
		}
		if !strings.HasSuffix(args[0], ".ez") {
			fmt.Fprintf(os.Stderr, "error: unknown command or invalid file '%s' — expected a .ez file\n", args[0])
			fmt.Fprintf(os.Stderr, "  usage: ez <file.ez>\n")
			fmt.Fprintf(os.Stderr, "  help:  ez --help\n")
			os.Exit(1)
		}

		// Pass extra args through to the compiled program
		var extraArgs []string
		if cmd.ArgsLenAtDash() > 0 {
			extraArgs = args[cmd.ArgsLenAtDash():]
		} else if len(args) > 1 {
			extraArgs = args[1:]
		}

		code, err := ezc.Run(args[0], extraArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(code)
		return nil
	},
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(replCmd, updateCmd, checkCmd, buildCmd, testCmd, reportCmd, versionCmd, docCmd, pzCmd, watchCmd)
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		CheckForUpdateAsync()
	}
	updateCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")

	buildCmd.Flags().StringP("output", "o", "", "Output binary name")
	buildCmd.Flags().BoolP("verbose", "v", false, "Show compilation commands")
	buildCmd.Flags().Bool("emit-c", false, "Emit generated C source only")
	buildCmd.Flags().StringP("quiet", "q", "", "Suppress warnings (use 'all' or comma-separated codes like W1001,W1002)")
	checkCmd.Flags().StringP("quiet", "q", "", "Suppress warnings (use 'all' or comma-separated codes like W1001,W1002)")

	pzCmd.Flags().StringP("template", "t", "basic", "Template: basic, cli, lib, multi, server, client")
	pzCmd.Flags().BoolP("comments", "c", false, "Include helpful syntax comments")
	pzCmd.Flags().BoolP("force", "f", false, "Overwrite existing directory")
	pzCmd.Flags().StringP("server-type", "s", "normal", "Server template type: minimal, normal")
}
