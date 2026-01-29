// Package main is the entry point for the cfl CLI.
package main

import (
	"fmt"
	"os"

	"github.com/open-cli-collective/atlassian-go/exitcode"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func main() {
	cmd := root.NewCmdRoot()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(exitcode.GeneralError)
	}
}
