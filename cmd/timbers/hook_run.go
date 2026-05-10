// Package main provides the entry point for the timbers CLI.
package main

import (
	"os"
	"path/filepath"

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

// hasActionablePending reports whether pre/post-commit hooks should take
// action — i.e., whether there is at least one undocumented commit that
// survives the default skip rules and .timbersignore filtering.
//
// Returns false for every infrastructure failure (not a git repo, mid-rebase,
// no .timbers/, storage error, pending-detection error) so that hooks never
// break git operations. The pre-commit hook turns this into a block; the
// post-commit hook turns this into a reminder. Both share the same definition
// of "actionable" so that pending, log, and the hooks always agree.
func hasActionablePending() bool {
	if !git.IsRepo() {
		return false
	}
	// During rebase/merge/cherry-pick, hooks fire for each replayed commit.
	// The work is already documented (or will be once the operation finishes
	// and the anchor self-heals) — don't block and don't nudge.
	if git.IsInteractiveGitOp() {
		return false
	}
	// Skip when .timbers/ is absent at the worktree root. This handles
	// infrastructure worktrees (e.g., beads backup branches) where git hooks
	// are shared but timbers isn't initialized.
	root, err := git.RepoRoot()
	if err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(root, ".timbers"))
	if err != nil || !info.IsDir() {
		return false
	}
	storage, err := ledger.NewDefaultStorage()
	if err != nil {
		return false
	}
	pending, err := storage.HasPendingCommits()
	if err != nil {
		return false
	}
	return pending
}

// runPreCommitHook executes the pre-commit hook logic.
// Blocks the commit when undocumented commits exist, forcing the user/agent
// to run 'timbers log' before committing again. This prevents stacking
// undocumented commits — each commit must be logged before the next.
//
// Errors during the check silently allow the commit (hooks must never break
// git operations due to timbers infrastructure failures).
func runPreCommitHook(cmd *cobra.Command) error {
	if !hasActionablePending() {
		return nil
	}

	printer := output.NewPrinter(cmd.OutOrStdout(), false, useColor(cmd))
	printer.Println()
	printer.Print("[timbers] Commit blocked: undocumented commit(s) exist\n")
	printer.Print("[timbers] Run 'timbers log \"what\" --why \"why\" --how \"how\"' first\n")
	printer.Print("[timbers] Or use --no-verify to bypass\n")
	printer.Println()

	return output.NewUserError("undocumented commits exist — run 'timbers log' first")
}

// runPostCommitHook executes the post-commit hook logic.
// It reminds users/agents to document the commit with timbers log, but only
// when there is at least one actionable pending commit. Infrastructure-only
// commits (.timbers/, .beads/, lockfiles, .timbersignore matches) are skipped
// so the reminder agrees with `timbers pending` and `timbers log`.
//
// This is non-blocking — it never returns an error.
func runPostCommitHook(cmd *cobra.Command) error {
	if !hasActionablePending() {
		return nil
	}

	printer := output.NewPrinter(cmd.OutOrStdout(), false, useColor(cmd))
	printer.Println(
		"[timbers] document this commit — " +
			"timbers log \"what\" --why \"why\" --how \"how\"",
	)
	return nil
}
