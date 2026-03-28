// Package main provides the entry point for the timbers CLI.
package main

import (
	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
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
	case "post-commit":
		return runPostCommitHook(cmd)
	case "claude-stop":
		return runClaudeStop(cmd)
	default:
		// Unknown hook - silently succeed to not block operations
		return nil
	}
}

// runPreCommitHook executes the pre-commit hook logic.
// Blocks the commit when undocumented commits exist, forcing the user/agent
// to run 'timbers log' before committing again. This prevents stacking
// undocumented commits — each commit must be logged before the next.
//
// Errors during the check silently allow the commit (hooks must never break
// git operations due to timbers infrastructure failures).
func runPreCommitHook(cmd *cobra.Command) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), false, useColor(cmd))

	if !git.IsRepo() {
		return nil
	}

	// During rebase/merge/cherry-pick, the pre-commit hook fires for each
	// replayed commit. Don't block — the work is already documented (or will
	// be after the operation completes and the anchor self-heals).
	if git.IsInteractiveGitOp() {
		return nil
	}

	storage, storageErr := ledger.NewDefaultStorage()
	if storageErr != nil {
		return nil //nolint:nilerr // hook must not block on infrastructure failure
	}

	pending, err := storage.HasPendingCommits()
	if err != nil {
		return nil //nolint:nilerr // hook must not block on infrastructure failure
	}

	if !pending {
		return nil
	}

	printer.Println()
	printer.Print("[timbers] Commit blocked: undocumented commit(s) exist\n")
	printer.Print("[timbers] Run 'timbers log \"what\" --why \"why\" --how \"how\"' first\n")
	printer.Print("[timbers] Or use --no-verify to bypass\n")
	printer.Println()

	return output.NewUserError("undocumented commits exist — run 'timbers log' first")
}

// runPostCommitHook executes the post-commit hook logic.
// It reminds users/agents to document the commit with timbers log.
// This is non-blocking - it never returns an error.
func runPostCommitHook(cmd *cobra.Command) error {
	if !git.IsRepo() {
		return nil
	}

	// Suppress per-commit reminders during rebase — they're noise for
	// replayed commits and confuse agents into thinking they need to log each one.
	if git.IsInteractiveGitOp() {
		return nil
	}

	printer := output.NewPrinter(cmd.OutOrStdout(), false, useColor(cmd))

	printer.Println(
		"[timbers] document this commit — " +
			"timbers log \"what\" --why \"why\" --how \"how\"",
	)

	// Always succeed - this is a nudge, not a blocker
	return nil
}
