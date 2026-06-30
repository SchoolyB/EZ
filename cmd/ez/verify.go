package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/marshallburns/ez/internal/ezc"
	"github.com/spf13/cobra"
)

//go:embed tests.ez
var verifyTestSrc []byte

// runVerify writes the embedded tests.ez to a temp file, compiles and runs it
// with ezc, then reports pass/fail. Returns the exit code (0 = pass).
func runVerify() int {
	tmp, err := os.CreateTemp("", "ez-verify-*.ez")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not create temp file: %v\n", err)
		return 1
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(verifyTestSrc); err != nil {
		tmp.Close()
		fmt.Fprintf(os.Stderr, "error: could not write temp file: %v\n", err)
		return 1
	}
	tmp.Close()

	fmt.Println("Running EZ language verification test...")
	code, err := ezc.Run(tmp.Name(), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	if code == 0 {
		fmt.Println("Verification passed.")
	} else {
		fmt.Fprintln(os.Stderr, "Verification FAILED.")
	}
	return code
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Run the built-in language verification test suite",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(runVerify())
	},
}
