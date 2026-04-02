package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/marshallburns/ez/internal/ezc"
	"github.com/spf13/cobra"
)

func filterEzFiles(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return []cobra.Completion{"ez"}, cobra.ShellCompDirectiveFilterFileExt
}

var runCmd = &cobra.Command{
	Use:               "run [file.ez]",
	Short:             "Compile and run an EZ source file",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: filterEzFiles,
	Run: func(cmd *cobra.Command, args []string) {
		var extraArgs []string
		if len(args) > 1 {
			extraArgs = args[1:]
		}
		code, err := ezc.Run(args[0], extraArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(code)
	},
}

var checkCmd = &cobra.Command{
	Use:               "check [file.ez | directory]",
	Short:             "Type-check a file or project without compiling",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: filterEzFiles,
	Run: func(cmd *cobra.Command, args []string) {
		code, err := ezc.Check(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(code)
	},
}

var buildCmd = &cobra.Command{
	Use:               "build [file.ez]",
	Short:             "Compile an EZ source file to a native binary",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: filterEzFiles,
	Run: func(cmd *cobra.Command, args []string) {
		output, _ := cmd.Flags().GetString("output")
		verbose, _ := cmd.Flags().GetBool("verbose")
		emitC, _ := cmd.Flags().GetBool("emit-c")

		opts := ezc.BuildOpts{
			Output:  output,
			Verbose: verbose,
			EmitC:   emitC,
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
		// Version string from ldflags contains the commit hash (e.g., v2.0.0-425-gabcdef1)
		commit := "unknown"
		// Version format: v2.0.0-NNN-gabcdef1 or v2.0.0-NNN-gabcdef1-dirty
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
		)

		failed := false
		passed := 0
		total := 0

		steps := []struct {
			name string
			cmd  string
			args []string
			dir  string
		}{
			{"Go tooling tests", "go", []string{"test", "./pkg/errors/...", "./pkg/lineeditor/..."}, root},
			{"Compiler unit tests", "make", []string{"test-unit"}, filepath.Join(root, "ezc")},
			{"Compiler e2e tests", "make", []string{"test-e2e"}, filepath.Join(root, "ezc")},
			{"Compiler integration tests", "bash", []string{filepath.Join(root, "scripts", "run_integration.sh")}, root},
			{"CLI integration tests", "bash", []string{filepath.Join(root, "scripts", "run_tests.sh")}, root},
		}

		for _, step := range steps {
			total++
			fmt.Printf("\n%s=== %s ===%s\n", bold, step.name, reset)
			c := exec.Command(step.cmd, step.args...)
			c.Dir = step.dir
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "  %s%sFAIL%s  %s\n", bold, red, reset, step.name)
				failed = true
			} else {
				fmt.Printf("  %s%sPASS%s  %s\n", bold, green, reset, step.name)
				passed++
			}
		}

		fmt.Println()
		if failed {
			fmt.Printf("%s%sSOME TEST SUITES FAILED%s (%d/%d passed)\n", bold, red, reset, passed, total)
			os.Exit(1)
		}
		fmt.Printf("%s%sALL TEST SUITES PASSED%s (%d/%d)\n", bold, green, reset, passed, total)
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
			return cmd.Help()
		}
		if !strings.HasSuffix(args[0], ".ez") {
			return cmd.Help()
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
	rootCmd.AddCommand(runCmd, replCmd, updateCmd, checkCmd, buildCmd, testCmd, reportCmd, versionCmd, docCmd, pzCmd, watchCmd)
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		CheckForUpdateAsync()
		ensureCompletionInstalled()
	}
	updateCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")

	buildCmd.Flags().StringP("output", "o", "", "Output binary name")
	buildCmd.Flags().BoolP("verbose", "v", false, "Show compilation commands")
	buildCmd.Flags().Bool("emit-c", false, "Emit generated C source only")

	pzCmd.Flags().StringP("template", "t", "basic", "Template: basic, cli, lib, multi, server, client")
	pzCmd.Flags().BoolP("comments", "c", false, "Include helpful syntax comments")
	pzCmd.Flags().BoolP("force", "f", false, "Overwrite existing directory")
	pzCmd.Flags().StringP("server-type", "s", "normal", "Server template type: minimal, normal")
}
