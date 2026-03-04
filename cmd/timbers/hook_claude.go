// Package main provides the entry point for the timbers CLI.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// --- Claude Code hook JSON types ---

// hookInput is the JSON payload Claude Code sends to hook stdin.
type hookInput struct {
	StopHookActive bool `json:"stop_hook_active"`
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

// --- Handler implementation ---

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
