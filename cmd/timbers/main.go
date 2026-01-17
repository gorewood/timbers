// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/steveyegge/timbers/internal/output"
)

// version is set via ldflags at build time.
// Example: go build -ldflags "-X main.version=1.0.0"
var version = "dev"

// jsonFlag is the global --json flag value.
var jsonFlag bool

func main() {
	code := run()
	os.Exit(code)
}

func run() int {
	cmd := newRootCmd()
	err := cmd.Execute()
	return output.GetExitCode(err)
}

// newRootCmd creates the root command for the timbers CLI.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timbers",
		Short: "A Git-native development ledger",
		Long: `Timbers - A Git-native development ledger that captures what/why/how as structured records.

Timbers turns Git history into a durable development ledger by:
  - Harvesting objective facts from Git (commits, diffstat, changed files)
  - Pairing them with agent/human-authored rationale (what/why/how)
  - Storing as portable Git notes that sync to remotes
  - Exporting structured data for downstream narrative generation

All commands support --json for structured output.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// If --json flag is set but no subcommand, output JSON error
			if jsonFlag {
				printer := output.NewPrinter(cmd.OutOrStdout(), true, false)
				err := output.NewUserError("no command specified. Run 'timbers --help' for usage")
				printer.Error(err)
				return err
			}
			// Otherwise show help
			return cmd.Help()
		},
	}

	// Set custom version template
	cmd.SetVersionTemplate(fmt.Sprintf("timbers version %s\n", version))

	// Add persistent --json flag (available to all subcommands)
	cmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output in JSON format")

	// Configure lipgloss for TTY detection
	lipgloss.SetHasDarkBackground(true)

	// Add subcommands
	cmd.AddCommand(newStatusCmd())

	return cmd
}
