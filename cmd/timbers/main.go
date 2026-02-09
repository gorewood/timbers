// Package main provides the entry point for the timbers CLI.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/output"
)

// Build info set via ldflags at build time by goreleaser.
// Example: go build -ldflags "-X main.version=1.0.0 -X main.commit=abc123 -X main.date=2024-01-01"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// isJSONMode reads the --json persistent flag from the command hierarchy.
// This replaces the former global jsonFlag variable, making commands
// independently testable without shared mutable state.
func isJSONMode(cmd *cobra.Command) bool {
	flag := cmd.Flags().Lookup("json")
	if flag == nil {
		// Walk up to root to find the persistent flag
		flag = cmd.Root().PersistentFlags().Lookup("json")
	}
	return flag != nil && flag.Value.String() == "true"
}

// buildVersion returns the full version string including commit and date.
func buildVersion() string {
	if commit == "none" && date == "unknown" {
		return version
	}
	shortCommit := commit
	if len(commit) > 7 {
		shortCommit = commit[:7]
	}
	return fmt.Sprintf("%s (%s, %s)", version, shortCommit, date)
}

func main() {
	code := run()
	os.Exit(code)
}

func run() int {
	cmd := newRootCmd()
	err := fang.Execute(context.Background(), cmd, fang.WithVersion(buildVersion()))
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
		Version:       buildVersion(),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// If --json flag is set but no subcommand, output JSON error
			if isJSONMode(cmd) {
				printer := output.NewPrinter(cmd.OutOrStdout(), true, false)
				err := output.NewUserError("no command specified. Run 'timbers --help' for usage")
				printer.Error(err)
				return err
			}
			// Otherwise show help
			return cmd.Help()
		},
	}

	// Add persistent --json flag (available to all subcommands)
	cmd.PersistentFlags().Bool("json", false, "Output in JSON format")

	// Configure lipgloss for TTY detection
	lipgloss.SetHasDarkBackground(true)

	// Define command groups and add commands
	addCommandGroups(cmd)
	addCommands(cmd)

	return cmd
}

// addCommandGroups defines the command groups for help output.
func addCommandGroups(cmd *cobra.Command) {
	cmd.AddGroup(&cobra.Group{ID: "core", Title: "Core Commands:"})
	cmd.AddGroup(&cobra.Group{ID: "query", Title: "Query Commands:"})
	cmd.AddGroup(&cobra.Group{ID: "sync", Title: "Sync Commands:"})
	cmd.AddGroup(&cobra.Group{ID: "agent", Title: "Agent Commands:"})
	cmd.AddGroup(&cobra.Group{ID: "admin", Title: "Admin Commands:"})
}

// addCommands adds all subcommands with their group assignments.
func addCommands(cmd *cobra.Command) {
	// Core commands: log, pending, status, amend
	addGroupedCommand(cmd, newLogCmd(), "core")
	addGroupedCommand(cmd, newAmendCmd(), "core")
	addGroupedCommand(cmd, newPendingCmd(), "core")
	addGroupedCommand(cmd, newStatusCmd(), "core")

	// Query commands: show, query, export
	addGroupedCommand(cmd, newShowCmd(), "query")
	addGroupedCommand(cmd, newQueryCmd(), "query")
	addGroupedCommand(cmd, newExportCmd(), "query")

	// Sync commands: notes
	addGroupedCommand(cmd, newNotesCmd(), "sync")

	// Agent commands: prime, prompt, generate, catchup
	addGroupedCommand(cmd, newPrimeCmd(), "agent")
	addGroupedCommand(cmd, newPromptCmd(), "agent")
	addGroupedCommand(cmd, newGenerateCmd(), "agent")
	addGroupedCommand(cmd, newCatchupCmd(), "agent")

	// Admin commands: init, uninstall, doctor, hooks, setup, onboard
	addGroupedCommand(cmd, newInitCmd(), "admin")
	addGroupedCommand(cmd, newUninstallCmd(), "admin")
	addGroupedCommand(cmd, newDoctorCmd(), "admin")
	addGroupedCommand(cmd, newHooksCmd(), "admin")
	addGroupedCommand(cmd, newSetupCmd(), "admin")
	addGroupedCommand(cmd, newOnboardCmd(), "admin")

	// Hidden internal commands
	cmd.AddCommand(newHookCmd())
}

// addGroupedCommand adds a subcommand with a group assignment.
func addGroupedCommand(parent *cobra.Command, child *cobra.Command, groupID string) {
	child.GroupID = groupID
	parent.AddCommand(child)
}
