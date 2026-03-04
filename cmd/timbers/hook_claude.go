// Package main provides the entry point for the timbers CLI.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// --- Claude Code hook JSON types ---

// hookInput is the JSON payload Claude Code sends to hook stdin.
type hookInput struct {
	ToolName       string        `json:"tool_name"`
	ToolInput      hookToolInput `json:"tool_input"`
	StopHookActive bool          `json:"stop_hook_active"`
}

// hookToolInput contains the tool_input fields from Claude Code.
type hookToolInput struct {
	Command string `json:"command"`
}

// preToolUseOutput is the structured JSON response for PreToolUse hooks.
type preToolUseOutput struct {
	HookSpecificOutput hookPermission `json:"hookSpecificOutput"`
}

// hookPermission carries the permission decision and reason.
type hookPermission struct {
	Decision string `json:"permissionDecision"`
	Reason   string `json:"permissionDecisionReason"`
}

// stopOutput is the structured JSON response for Stop hooks.
type stopOutput struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// pendingChecker abstracts the pending-commits check for testability.
type pendingChecker interface {
	HasPendingCommits() (bool, error)
}

// --- Handler implementations ---

// runClaudePreToolUse handles the PreToolUse hook for Claude Code.
// Blocks git commit when undocumented commits exist.
// Any error silently allows the operation (hooks must never break workflows).
func runClaudePreToolUse(cmd *cobra.Command) error {
	return runClaudePreToolUseWith(cmd.InOrStdin(), cmd.OutOrStdout(), nil)
}

// runClaudePreToolUseWith is the testable implementation of PreToolUse.
func runClaudePreToolUseWith(stdin io.Reader, stdout io.Writer, checker pendingChecker) error {
	input, err := parseHookInput(stdin)
	if err != nil {
		return nil //nolint:nilerr // hooks must never block on malformed input
	}

	command := input.ToolInput.Command
	if !isGitCommitCommand(command) {
		return nil // not a git commit → allow
	}

	if strings.Contains(command, "timbers") {
		return nil // never block timbers' own commits
	}

	pending, ok := checkPending(checker)
	if !ok || !pending {
		return nil
	}

	return writePreToolUseResponse(stdout)
}

// runClaudeStop handles the Stop hook for Claude Code.
// Blocks session end when undocumented commits exist.
func runClaudeStop(cmd *cobra.Command) error {
	return runClaudeStopWith(cmd.InOrStdin(), cmd.OutOrStdout(), nil)
}

// runClaudeStopWith is the testable implementation of Stop.
func runClaudeStopWith(stdin io.Reader, stdout io.Writer, checker pendingChecker) error {
	input, err := parseHookInput(stdin)
	if err != nil {
		return nil //nolint:nilerr // hooks must never block on malformed input
	}

	if input.StopHookActive {
		return nil // prevent infinite loops
	}

	pending, ok := checkPending(checker)
	if !ok || !pending {
		return nil
	}

	return writeStopResponse(stdout)
}

// --- Helpers ---

// checkPending resolves the checker (creating a default if nil) and checks for pending commits.
// Returns (hasPending, ok). ok=false means the check could not be performed (allow through).
func checkPending(checker pendingChecker) (bool, bool) {
	if checker == nil {
		var err error
		checker, err = defaultPendingChecker()
		if err != nil {
			return false, false
		}
	}

	pending, err := checker.HasPendingCommits()
	if err != nil {
		return false, false
	}
	return pending, true
}

// writePreToolUseResponse writes the deny JSON to stdout.
func writePreToolUseResponse(w io.Writer) error {
	resp := preToolUseOutput{
		HookSpecificOutput: hookPermission{
			Decision: "deny",
			Reason:   "Undocumented commit(s) exist. Run 'timbers log' before committing.",
		},
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return fmt.Errorf("writing hook response: %w", err)
	}
	return nil
}

// writeStopResponse writes the block JSON to stdout.
func writeStopResponse(w io.Writer) error {
	resp := stopOutput{
		Decision: "block",
		Reason:   "Undocumented commit(s) exist. Run 'timbers pending' to review, then 'timbers log' to document.",
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return fmt.Errorf("writing hook response: %w", err)
	}
	return nil
}

// parseHookInput reads and decodes the JSON payload from stdin.
func parseHookInput(r io.Reader) (*hookInput, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	var input hookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("parsing hook input: %w", err)
	}
	return &input, nil
}

// isGitCommitCommand checks if a shell command string contains a git commit invocation.
func isGitCommitCommand(cmd string) bool {
	return strings.Contains(cmd, "git commit")
}

// defaultPendingChecker creates a real storage-backed pending checker.
func defaultPendingChecker() (pendingChecker, error) {
	if !git.IsRepo() {
		return nil, errors.New("not a git repo")
	}
	storage, err := ledger.NewDefaultStorage()
	if err != nil {
		return nil, err
	}
	return storage, nil
}
