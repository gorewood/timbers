// Package main provides the entry point for the timbers CLI.
package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// envSkipCrossAgentDebt, when set to a truthy value ("1", "true", "yes", "on"),
// short-circuits the pre/post-commit gate. Intended as an escape hatch for
// parallel-agent flows where first-parent traversal alone still over-fires
// (e.g., a merge commit on the first-parent line that the current agent
// considers "not theirs"). Mirrors existing bypass conventions and is
// cheaper than --no-verify because it doesn't disable other hooks.
const envSkipCrossAgentDebt = "TIMBERS_SKIP_CROSS_AGENT_DEBT"

// envTruthy reports whether the named env var is set to a recognized
// truthy value. Case-insensitive; whitespace-trimmed.
func envTruthy(name string) bool {
	val := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch val {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

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
	// Cross-agent escape hatch: when the user has explicitly opted out of
	// the gate (typically because multiple agents are working in parallel
	// and one of them is about to run timbers catchup), skip both the
	// block and the nudge. Cheaper than --no-verify because it doesn't
	// disable other hooks.
	if envTruthy(envSkipCrossAgentDebt) {
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
	// When the gate aborts, the user's `git commit` did NOT run — but
	// `git add` already moved their work into the index. Surface that
	// explicitly so the caller doesn't follow up with `timbers log`
	// thinking the gate ate their staging. (timbers log will now refuse
	// on a dirty tree anyway since v0.22.8; this hint short-circuits the
	// confusion.) Only emit when there's actually staged work — silent
	// otherwise.
	if git.HasStagedChanges() {
		printer.Print("[timbers] Your staged changes remain in the index — inspect with: git diff --cached\n")
	}
	printer.Print("[timbers] Document the prior commit(s) first: timbers log \"what\" --why \"why\" --how \"how\"\n")
	printer.Print("[timbers] Or TIMBERS_SKIP_CROSS_AGENT_DEBT=1 (parallel-agent flows), or --no-verify\n")
	printer.Println()

	return output.NewUserError("undocumented commits exist — run 'timbers log' first")
}

// runPostCommitHook executes the post-commit hook logic.
//
// Two independent surfaces fire from here:
//
//  1. Actionable-pending reminder — "[timbers] document this commit". Fires
//     when at least one in-session commit is undocumented. Identical to
//     pre-v0.23.0 behavior, just routed through the provenance-aware count
//     so foreign-author and stale commits don't trigger the nudge.
//
//  2. Stale-self auto-skip note — "[timbers] auto-skipped N stale commit(s)".
//     Fires when the cross-agent debt classifier silently dropped at least
//     one of the user's OWN commits on staleness (NOT email-mismatch). This
//     is the visibility safety net for the worst-case failure mode: a long
//     autonomous loop or marathon session running past the 24h window would
//     otherwise silently lose its own signal. Foreign-author skips stay
//     silent per the reframe — the operator chose to not be that author.
//
// Non-blocking — never returns an error. Errors from the classifier are
// swallowed (hooks must never break git operations).
func runPostCommitHook(cmd *cobra.Command) error {
	actionable, staleSelf := classifyPostCommitState()
	if actionable == 0 && staleSelf == 0 {
		return nil
	}

	printer := output.NewPrinter(cmd.OutOrStdout(), false, useColor(cmd))
	if actionable > 0 {
		printer.Println(
			"[timbers] document this commit — " +
				"timbers log \"what\" --why \"why\" --how \"how\"",
		)
	}
	if staleSelf > 0 {
		printer.Print(
			"[timbers] auto-skipped %d stale commit(s) (>%s old, same author); "+
				"run 'timbers pending --explain' to inspect, "+
				"or 'timbers log --range' to backfill if needed\n",
			staleSelf, ledger.DefaultSessionWindow,
		)
	}
	return nil
}

// classifyPostCommitState walks the pending range and returns counts of
// actionable (in-session blocking) and stale-self (same-author auto-skipped)
// commits. Returns (0, 0) on any storage/classifier error — hooks must never
// break git operations.
func classifyPostCommitState() (actionable, staleSelf int) {
	if envTruthy(envSkipCrossAgentDebt) {
		return 0, 0
	}
	root, err := git.RepoRoot()
	if err != nil {
		return 0, 0
	}
	info, err := os.Stat(filepath.Join(root, ".timbers"))
	if err != nil || !info.IsDir() {
		return 0, 0
	}
	storage, err := ledger.NewDefaultStorage()
	if err != nil {
		return 0, 0
	}
	classified, _, classifyErr := storage.ExplainPending()
	if classifyErr != nil {
		return 0, 0
	}
	for _, item := range classified {
		switch item.Reason {
		case "":
			actionable++
		case "stale":
			staleSelf++
		}
	}
	return actionable, staleSelf
}
