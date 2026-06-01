package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
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
		isDir := statErr == nil && info.IsDir()
		if !isDir && !strings.HasSuffix(args[0], ".ez") {
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
		showTime, _ := cmd.Flags().GetBool("time")
		noColor, _ := cmd.Flags().GetBool("no-color")

		opts := ezc.BuildOpts{
			Output:  output,
			Verbose: verbose,
			EmitC:   emitC,
			Time:    showTime,
			NoColor: noColor,
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

var installCmd = &cobra.Command{
	Use:   "install <version>",
	Short: "Install a specific EZ version, replacing the current install",
	Long: "Install a specific EZ version by exact semver, replacing the current " +
		"install. Downgrades and pre-release versions are supported.\n\n" +
		"Examples:\n" +
		"  ez install 2.5.0\n" +
		"  ez install 3.0.0-beta.2",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runInstall(args[0])
	},
}

var updateCmd = &cobra.Command{
	Use:   "update [url]",
	Short: "Check for updates and upgrade EZ",
	Long: "Check for updates and upgrade EZ.\n\n" +
		"Without flags, installs the latest stable release and prints a note " +
		"if a newer pre-release is available.\n\n" +
		"With --pre, installs the latest pre-release (alpha, beta, or rc).",
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		confirm, _ := cmd.Flags().GetBool("confirm")
		pre, _ := cmd.Flags().GetBool("pre")
		var url string
		if len(args) > 0 {
			url = args[0]
		}
		runUpdate(confirm, url, pre)
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

Output is written to DOCS.md by default. Use -o/--output to write
to a different path (parent directories are created as needed).`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		output, _ := cmd.Flags().GetString("output")
		generateDocs(args, output)
	},
}

var fmtCmd = &cobra.Command{
	Use:   "fmt <path>",
	Short: "Format .ez source files",
	Long: `Normalize formatting of .ez source files (indentation, trailing
whitespace, end-of-file newline, blank-line runs).

Examples:
  ez fmt .              Format .ez files in current directory (no recursion)
  ez fmt ./...          Format recursively from current directory
  ez fmt file.ez        Format a single file
  ez fmt a.ez b.ez      Format multiple files
  ez fmt --check ./...  Exit non-zero if any file would change (CI gate)

By default files are rewritten in place. Use --check for a non-mutating CI gate.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		check, _ := cmd.Flags().GetBool("check")
		exit := runFmt(args, check)
		if exit != 0 {
			os.Exit(exit)
		}
		return nil
	},
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Print system info for bug reports",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("EZ Bug Report Info")
		fmt.Println("======================")

		vi := GetVersionInfo()
		channel := "stable"
		switch vi.Channel {
		case "pre-release":
			channel = "pre-release"
		case "dev":
			channel = "dev"
		}
		fmt.Printf("EZ Version:  %s  (%s)\n", vi.Display, channel)

		if vi.Commit == "" {
			fmt.Println("Commit:      (released build)")
		} else {
			fmt.Printf("Commit:      %s\n", vi.Commit)
		}

		fmt.Printf("Install:     %s\n", reportInstallPath())

		fmt.Printf("OS:          %s/%s  %s\n", runtime.GOOS, runtime.GOARCH, reportKernel())
		fmt.Printf("CPU:         %s\n", reportCPUModel())

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

		cc, ccVer, ccTriple := reportCCompiler()
		fmt.Printf("C compiler:  %s\n", cc)
		if ccVer != "" {
			fmt.Printf("             %s\n", ccVer)
		}
		if ccTriple != "" {
			fmt.Printf("             target: %s\n", ccTriple)
		}
	},
}

// reportInstallPath returns the path to the running ez binary with $HOME
// redacted to ~ so the output is safe to paste into a public bug report.
func reportInstallPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(exe, home) {
		return "~" + strings.TrimPrefix(exe, home)
	}
	return exe
}

// reportKernel returns `uname -sr` output (e.g. "Darwin 25.5.0").
func reportKernel() string {
	out, err := exec.Command("uname", "-sr").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// reportCPUModel returns the human-readable CPU brand string, or empty.
func reportCPUModel() string {
	switch runtime.GOOS {
	case "darwin":
		if out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	case "linux":
		if out, err := exec.Command("sh", "-c", "grep -m1 'model name' /proc/cpuinfo | cut -d: -f2-").Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return "unknown"
}

// reportCCompiler resolves the C compiler ezc will actually invoke
// (honouring $CC, else the first of clang/gcc/cc on PATH) and returns
// its resolved path, first line of --version output, and target triple.
func reportCCompiler() (path, version, triple string) {
	cc := os.Getenv("CC")
	if cc == "" {
		for _, candidate := range []string{"clang", "gcc", "cc"} {
			if p, err := exec.LookPath(candidate); err == nil {
				cc = p
				break
			}
		}
	}
	if cc == "" {
		return "not found", "", ""
	}
	if resolved, err := exec.LookPath(cc); err == nil {
		path = resolved
	} else {
		path = cc
	}
	if out, err := exec.Command(cc, "--version").Output(); err == nil {
		if line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]; line != "" {
			version = line
		}
	}
	if out, err := exec.Command(cc, "-dumpmachine").Output(); err == nil {
		triple = strings.TrimSpace(string(out))
	}
	return
}

func printManUsage() {
	fmt.Println("ez man — builtin and stdlib documentation")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ez man builtins          list all builtin functions")
	fmt.Println("  ez man <module>          list all functions in a stdlib module")
	fmt.Println("                           (e.g. ez man math)")
	fmt.Println("  ez man <name>            docs for a specific builtin or stdlib function")
	fmt.Println("                           (e.g. ez man println, ez man sqrt)")
	fmt.Println("  ez man <name()>          same — trailing () is ignored")
	fmt.Println()
	fmt.Println("Stdlib modules:")
	mods := make([]string, 0, len(stdlibModules))
	for m := range stdlibModules {
		mods = append(mods, m)
	}
	sort.Strings(mods)
	fmt.Printf("  %s\n", strings.Join(mods, "  "))
}

func printBuiltinsIndex() {
	fmt.Println("Builtin functions  (ez man <name> for details)")
	fmt.Println(strings.Repeat("─", 50))
	groups := []struct {
		label string
		names []string
	}{
		{"I/O        ", []string{"println", "print", "eprintln", "eprint", "input"}},
		{"Control    ", []string{"exit", "panic", "assert"}},
		{"Sleep      ", []string{"sleep_s", "sleep_ms", "sleep_ns"}},
		{"Type casts ", []string{"int", "uint", "float", "string", "char", "byte", "bool", "cast"}},
		{"Width casts", []string{"i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64", "f32", "f64", "i128", "i256", "u128", "u256"}},
		{"Memory     ", []string{"new", "ref", "addr", "copy"}},
		{"Introspect ", []string{"len", "type_of", "size_of"}},
		{"Misc       ", []string{"error", "range", "c_string", "to_char", "char_count", "here"}},
		{"Types      ", []string{"SourceLocation", "Error"}},
	}
	for _, g := range groups {
		fmt.Printf("  %s  %s\n", g.label, strings.Join(g.names, "  "))
	}
}

func printManEntry(module, name, kind, sig, fields, desc, example string) {
	label := name
	if kind == "type" {
		// types don't get ()
	} else if strings.Contains(sig, "(") {
		label = name + "()"
	}
	fmt.Printf("%s: %s\n", module, label)
	fmt.Println(strings.Repeat("─", 44))
	fmt.Printf("Module:  %s\n", module)
	if kind == "type" {
		fmt.Printf("Kind:    type\n")
		if fields != "" {
			fmt.Println("\nFields:")
			for _, f := range strings.Split(fields, "\n") {
				parts := strings.Fields(f)
				if len(parts) >= 2 {
					fmt.Printf("  %-12s %s\n", parts[0], parts[1])
				}
			}
		}
		fmt.Printf("\n%s\n", desc)
	} else {
		fmt.Printf("Signature:  %s\n\n", sig)
		fmt.Println(desc)
	}
	if example != "" {
		fmt.Println("\nExample:")
		for _, line := range strings.Split(example, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}
}

func printBuiltinEntry(name string, entry BuiltinManEntry) {
	printManEntry("builtin", name, entry.Kind, entry.Sig, entry.Fields, entry.Desc, entry.Example)
}

func printStdlibEntry(name string, entry StdlibManEntry) {
	printManEntry(entry.Module, name, entry.Kind, entry.Sig, entry.Fields, entry.Desc, entry.Example)
}

func printStdlibModuleIndex(module string) {
	groups, ok := stdlibModuleGroups[module]
	if !ok {
		fmt.Fprintf(os.Stderr, "ez: no documentation for module '%s'\n", module)
		fmt.Fprintf(os.Stderr, "    try: ez man\n")
		os.Exit(1)
	}
	fmt.Printf("module: %s  (ez man <name> for details)\n", module)
	fmt.Println(strings.Repeat("─", 50))
	for _, g := range groups {
		fmt.Printf("  %s  %s\n", g.Label, strings.Join(g.Names, "  "))
	}
}

var manCmd = &cobra.Command{
	Use:   "man [name]",
	Short: "Show documentation for a builtin or stdlib function",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printManUsage()
			return
		}

		name := strings.TrimSuffix(args[0], "()")

		if name == "builtins" || name == "builtin" {
			printBuiltinsIndex()
			return
		}

		// Module-level index (e.g. ez man math)
		if _, isMod := stdlibModules[name]; isMod {
			printStdlibModuleIndex(name)
			return
		}

		// Builtin lookup
		if entry, ok := builtinManDocs[name]; ok {
			printBuiltinEntry(name, entry)
			return
		}

		// Stdlib function/type lookup
		if entry, ok := stdlibManDocs[name]; ok {
			printStdlibEntry(name, entry)
			return
		}

		fmt.Fprintf(os.Stderr, "ez: no documentation for '%s'\n", name)
		fmt.Fprintf(os.Stderr, "    try: ez man builtins\n")
		os.Exit(1)
	},
}

var rootCmd = &cobra.Command{
	Use:   "ez [file.ez]",
	Short: "EZ Programming Language",
	Long:  "A statically-typed programming language that compiles to native binaries.",
	Args:  cobra.ArbitraryArgs,
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

		// Prepend compiler flags (before program args)
		var compilerArgs []string
		quiet, _ := cmd.Flags().GetString("quiet")
		if quiet == "all" {
			compilerArgs = append(compilerArgs, "--quiet")
		} else if quiet != "" {
			compilerArgs = append(compilerArgs, "--quiet", quiet)
		}
		if noColor, _ := cmd.Flags().GetBool("no-color"); noColor {
			compilerArgs = append(compilerArgs, "--no-color")
		}
		compilerArgs = append(compilerArgs, extraArgs...)

		code, err := ezc.Run(args[0], compilerArgs)
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
	rootCmd.AddCommand(updateCmd, installCmd, checkCmd, buildCmd, reportCmd, versionCmd, docCmd, fmtCmd, pzCmd, watchCmd, manCmd)
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		CheckForUpdateAsync()
	}
	// Custom help for root only: hide -h (users have `ez help`) and -v (use `ez version`)
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != rootCmd {
			defaultHelp(cmd, args)
			return
		}
		fmt.Fprintf(cmd.OutOrStdout(), `%s

Usage:
  ez [file.ez] [flags]
  ez [command]

Available Commands:
`, cmd.Long)
		for _, c := range cmd.Commands() {
			if c.IsAvailableCommand() || c.Name() == "help" {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-12s%s\n", c.Name(), c.Short)
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), `
Flags:
  -q, --quiet string   Suppress warnings ('all' or comma-separated codes like W1001,W1002)
      --no-color       Disable colored output

Use "ez [command] --help" for more information about a command.
`)
	})
	updateCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")
	updateCmd.Flags().Bool("pre", false, "Install the latest pre-release (alpha/beta/rc) instead of the latest stable")

	buildCmd.Flags().StringP("output", "o", "", "Output binary name")
	buildCmd.Flags().BoolP("verbose", "v", false, "Show compilation commands")
	buildCmd.Flags().MarkHidden("verbose")
	buildCmd.Flags().Bool("emit-c", false, "Emit generated C source only")
	buildCmd.Flags().Bool("time", false, "Show compilation timing")
	buildCmd.Flags().Bool("no-color", false, "Disable colored output")
	rootCmd.Flags().StringP("quiet", "q", "", "Suppress warnings (use 'all' or comma-separated codes like W1001,W1002)")
	rootCmd.Flags().Bool("no-color", false, "Disable colored output")
	buildCmd.Flags().StringP("quiet", "q", "", "Suppress warnings (use 'all' or comma-separated codes like W1001,W1002)")
	checkCmd.Flags().StringP("quiet", "q", "", "Suppress warnings (use 'all' or comma-separated codes like W1001,W1002)")
	watchCmd.Flags().StringP("quiet", "q", "", "Suppress warnings (use 'all' or comma-separated codes like W1001,W1002)")

	docCmd.Flags().StringP("output", "o", defaultDocOutputPath, "Path to write generated markdown")

	fmtCmd.Flags().Bool("check", false, "Exit non-zero if any file would change; don't modify files")

	pzCmd.Flags().StringP("template", "t", "basic", "Template: basic, cli, lib, multi, server, client")
	pzCmd.Flags().BoolP("comments", "c", false, "Include helpful syntax comments")
	pzCmd.Flags().BoolP("force", "f", false, "Overwrite existing directory")
	pzCmd.Flags().StringP("server-type", "s", "normal", "Server template type: minimal, normal")
}
