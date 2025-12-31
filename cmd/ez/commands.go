package main

import (
	"strings"

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
	Use:   "update",
	Short: "Check for updates and upgrade EZ",
	Run: func(cmd *cobra.Command, args []string) {
		runUpdate(args)
	},
}

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Help about EZ",
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd.Help()
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

var rootCmd = &cobra.Command{
	Use:     "ez [file.ez]",
	Short:   "EZ Language Interpreter",
	Long:    "A simple, interpreted, statically-typed programming language designed for clarity and ease of use.",
	Args:    cobra.MaximumNArgs(1),
	Version: getVersionString(),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 && strings.HasSuffix(args[0], ".ez") {
			runFile(args[0])
			return nil
		}
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(replCmd, updateCmd, checkCmd, lexCmd, parseCmd, helpCmd, versionCmd)
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		CheckForUpdateAsync()
	}
	updateCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")
}
