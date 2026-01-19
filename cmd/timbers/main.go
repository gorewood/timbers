// Package main provides the entry point for the timbers CLI.
package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss"
	"github.com/rbergman/timbers/internal/output"
	"github.com/spf13/cobra"
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
	err := fang.Execute(context.Background(), cmd, fang.WithVersion(version))
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

	// Add persistent --json flag (available to all subcommands)
	cmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output in JSON format")

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
	// Core commands: log, pending, status
	addGroupedCommand(cmd, newLogCmd(), "core")
	addGroupedCommand(cmd, newPendingCmd(), "core")
	addGroupedCommand(cmd, newStatusCmd(), "core")

	// Query commands: show, query, export
	addGroupedCommand(cmd, newShowCmd(), "query")
	addGroupedCommand(cmd, newQueryCmd(), "query")
	addGroupedCommand(cmd, newExportCmd(), "query")

	// Sync commands: notes
	addGroupedCommand(cmd, newNotesCmd(), "sync")

	// Agent commands: prime, skill, prompt, generate, catchup
	addGroupedCommand(cmd, newPrimeCmd(), "agent")
	addGroupedCommand(cmd, newSkillCmd(), "agent")
	addGroupedCommand(cmd, newPromptCmd(), "agent")
	addGroupedCommand(cmd, newGenerateCmd(), "agent")
	addGroupedCommand(cmd, newCatchupCmd(), "agent")

	// Admin commands: uninstall
	addGroupedCommand(cmd, newUninstallCmd(), "admin")
}

// addGroupedCommand adds a subcommand with a group assignment.
func addGroupedCommand(parent *cobra.Command, child *cobra.Command, groupID string) {
	child.GroupID = groupID
	parent.AddCommand(child)
}
