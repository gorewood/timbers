// Package main provides the entry point for the timbers CLI.
package main

import (
	"github.com/spf13/cobra"

	"github.com/steveyegge/timbers/internal/git"
	"github.com/steveyegge/timbers/internal/ledger"
	"github.com/steveyegge/timbers/internal/output"
)

// pendingResult holds the data for pending output.
type pendingResult struct {
	Count     int             `json:"count"`
	LastEntry *entryReference `json:"last_entry,omitempty"`
	Commits   []commitSummary `json:"commits,omitempty"`
}

// entryReference is a simplified reference to a ledger entry.
type entryReference struct {
	ID           string `json:"id"`
	AnchorCommit string `json:"anchor_commit"`
	CreatedAt    string `json:"created_at"`
}

// commitSummary is a simplified commit for output.
type commitSummary struct {
	SHA     string `json:"sha"`
	Short   string `json:"short"`
	Subject string `json:"subject"`
}

// newPendingCmd creates the pending command.
func newPendingCmd() *cobra.Command {
	return newPendingCmdInternal(nil)
}

// newPendingCmdInternal creates the pending command with optional storage injection.
// If storage is nil, a real storage is created when the command runs.
func newPendingCmdInternal(storage *ledger.Storage) *cobra.Command {
	var countOnly bool

	cmd := &cobra.Command{
		Use:   "pending",
		Short: "Show undocumented commits since last entry",
		Long:  `Show commits that have not been documented since the last ledger entry.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPending(cmd, storage, countOnly)
		},
	}

	cmd.Flags().BoolVar(&countOnly, "count", false, "Show count only, without commit list")

	return cmd
}

// runPending executes the pending command.
func runPending(cmd *cobra.Command, storage *ledger.Storage, countOnly bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	// Check if we're in a git repo (only when using real git)
	if storage == nil && !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	// Create storage if not injected
	if storage == nil {
		storage = ledger.NewStorage(nil)
	}

	// Get pending commits
	commits, latest, err := storage.GetPendingCommits()
	if err != nil {
		printer.Error(err)
		return err
	}

	// Build result
	result := buildPendingResult(commits, latest)

	// Output based on mode
	if jsonFlag {
		return outputPendingJSON(printer, result)
	}

	outputPendingHuman(printer, result, countOnly)
	return nil
}

// buildPendingResult constructs the result from commits and latest entry.
func buildPendingResult(commits []git.Commit, latest *ledger.Entry) *pendingResult {
	result := &pendingResult{
		Count:   len(commits),
		Commits: make([]commitSummary, 0, len(commits)),
	}

	// Add entry reference if exists
	if latest != nil {
		result.LastEntry = &entryReference{
			ID:           latest.ID,
			AnchorCommit: latest.Workset.AnchorCommit,
			CreatedAt:    latest.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	// Add commit summaries
	for _, c := range commits {
		result.Commits = append(result.Commits, commitSummary{
			SHA:     c.SHA,
			Short:   c.Short,
			Subject: c.Subject,
		})
	}

	return result
}

// outputPendingJSON outputs the result as JSON.
func outputPendingJSON(printer *output.Printer, result *pendingResult) error {
	data := map[string]any{
		"count":   result.Count,
		"commits": result.Commits,
	}

	if result.LastEntry != nil {
		data["last_entry"] = result.LastEntry
	}

	return printer.Success(data)
}

// outputPendingHuman outputs the result in human-readable format.
func outputPendingHuman(printer *output.Printer, result *pendingResult, countOnly bool) {
	// Handle no pending commits
	if result.Count == 0 {
		printer.Println("No pending commits - all work is documented")
		return
	}

	// Count-only mode
	if countOnly {
		printer.Print("%d\n", result.Count)
		return
	}

	// Full output with commit list
	printer.Print("%d pending commit", result.Count)
	if result.Count != 1 {
		printer.Print("s")
	}
	if result.LastEntry != nil {
		printer.Print(" since %s", result.LastEntry.ID)
	}
	printer.Println()
	printer.Println()

	// List commits
	for _, c := range result.Commits {
		printer.Print("  %s %s\n", c.Short, c.Subject)
	}

	// Suggest command
	printer.Println()
	printer.Println("Run 'timbers log \"<what>\" --why \"<why>\" --how \"<how>\"' to document this work")
}
