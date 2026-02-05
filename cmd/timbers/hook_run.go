// Package main provides the entry point for the timbers CLI.
package main

import (
	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
	"github.com/spf13/cobra"
)

// newHookCmd creates the hidden hook parent command for internal hook execution.
func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "hook",
		Short:  "Internal hook runner",
		Long:   `Internal command for running hook logic. Called by git hooks.`,
		Hidden: true,
	}

	cmd.AddCommand(newHookRunCmd())
	return cmd
}

// newHookRunCmd creates the hook run subcommand.
func newHookRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <hook-name>",
		Short: "Execute hook logic",
		Long:  `Execute the logic for the specified hook. Called by installed git hooks.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runHookRun,
	}
}

// runHookRun executes the hook run command.
func runHookRun(cmd *cobra.Command, args []string) error {
	hookName := args[0]

	switch hookName {
	case "pre-commit":
		return runPreCommitHook(cmd)
	default:
		// Unknown hook - silently succeed to not block operations
		return nil
	}
}

// runPreCommitHook executes the pre-commit hook logic.
// It checks for pending commits and warns if any exist.
// This is non-blocking - it never returns an error to allow the commit.
func runPreCommitHook(cmd *cobra.Command) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), false, output.IsTTY(cmd.OutOrStdout()))

	// Check if we're in a git repo
	if !git.IsRepo() {
		// Not in a repo - silently succeed
		return nil
	}

	// Create storage and get pending count
	storage := ledger.NewStorage(nil)
	commits, _, err := storage.GetPendingCommits()
	if err != nil {
		// Error getting pending - silently succeed to not block commits
		return nil //nolint:nilerr // intentional: hook must not block git operations
	}

	pendingCount := len(commits)
	if pendingCount == 0 {
		// No pending commits - nothing to warn about
		return nil
	}

	// Warn about pending commits (non-blocking)
	printer.Println()
	printer.Print("[timbers] Warning: %d undocumented commit(s)\n", pendingCount)
	printer.Print("[timbers] Run 'timbers pending' to see details\n")
	printer.Print("[timbers] Run 'timbers log' to document your work\n")
	printer.Println()

	// Always succeed - this is a warning, not a blocker
	return nil
}
