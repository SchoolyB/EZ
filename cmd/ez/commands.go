package main

import (
	"os"
	"slices"
	"strings"

	"github.com/marshallburns/ez/pkg/stdlib"
	"github.com/spf13/cobra"
)

func filterEzFiles(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return []cobra.Completion{"ez"}, cobra.ShellCompDirectiveFilterFileExt
}

var checkCmd = &cobra.Command{
	Use:               "check [file.ez | directory]",
	Aliases:           []string{"build"},
	Short:             "Check syntax and types for a file or directory",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: filterEzFiles,
	Run: func(cmd *cobra.Command, args []string) {
		arg := args[0]
		if strings.HasSuffix(arg, ".ez") {
			checkFile(arg)
		} else {
			checkProject(arg)
		}
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

var lexCmd = &cobra.Command{
	Use:               "lex [file]",
	Short:             "Tokenize a file",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: filterEzFiles,
	Hidden:            true,
	Run: func(cmd *cobra.Command, args []string) {
		lexFile(args[0])
	},
}

var parseCmd = &cobra.Command{
	Use:               "parse [file]",
	Short:             "Parse a file",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: filterEzFiles,
	Hidden:            true,
	Run: func(cmd *cobra.Command, args []string) {
		parse_file(args[0])
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

var rootCmd = &cobra.Command{
	Use:     "ez [file.ez]",
	Short:   "EZ Language Interpreter",
	Long:    "A simple, interpreted, statically-typed programming language designed for clarity and ease of use.",
	Args:    cobra.ArbitraryArgs,
	Version: getVersionString(),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		if !strings.HasSuffix(args[0], ".ez") {
			return cmd.Help()
		}

		programArgs := []string{os.Args[0], args[0]}
		if cmd.ArgsLenAtDash() > 0 {
			programArgs = slices.Concat(programArgs, args[cmd.ArgsLenAtDash():])
		} else if len(args) > 1 {
			programArgs = slices.Concat(programArgs, args[1:])
		}
		stdlib.CommandLineArgs = programArgs

		runFile(args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(replCmd, updateCmd, checkCmd, lexCmd, parseCmd, versionCmd, docCmd, pzCmd)
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		CheckForUpdateAsync()
	}
	updateCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")

	pzCmd.Flags().StringP("template", "t", "basic", "Template: basic, cli, lib, multi")
	pzCmd.Flags().BoolP("comments", "c", false, "Include helpful syntax comments")
	pzCmd.Flags().BoolP("force", "f", false, "Overwrite existing directory")
}
