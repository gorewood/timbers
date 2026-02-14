// Package main provides the entry point for the timbers CLI.
package main

import (
	"errors"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
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
		Long: `Show commits that have not been documented since the last ledger entry.

This command identifies work that needs to be documented by finding all commits
made after the most recent ledger entry's anchor commit.

Examples:
  timbers pending              # List all undocumented commits
  timbers pending --count      # Show only the count of pending commits
  timbers pending --json       # Output pending commits as JSON`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPending(cmd, storage, countOnly)
		},
	}

	cmd.Flags().BoolVar(&countOnly, "count", false, "Show count only, without commit list")

	return cmd
}

// runPending executes the pending command.
func runPending(cmd *cobra.Command, storage *ledger.Storage, countOnly bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd))

	// Check if we're in a git repo (only when using real git)
	if storage == nil && !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	// Create storage if not injected
	if storage == nil {
		var err error
		storage, err = ledger.NewDefaultStorage()
		if err != nil {
			printer.Error(err)
			return err
		}
	}

	// Get pending commits
	commits, latest, err := storage.GetPendingCommits()
	if err != nil && !errors.Is(err, ledger.ErrStaleAnchor) {
		printer.Error(err)
		return err
	}
	if errors.Is(err, ledger.ErrStaleAnchor) {
		printer.Warn("last entry's anchor commit is no longer in git history (squash merge or rebase?); showing all reachable commits")
	}

	// Build result
	result := buildPendingResult(commits, latest)

	// Output based on mode
	if printer.IsJSON() {
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

	// Add suggested commands based on state
	if result.Count > 0 {
		data["suggested_commands"] = []string{
			"timbers log \"<what>\" --why \"<why>\" --how \"<how>\"",
		}
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

	// Section header
	printer.Section("Pending Commits")

	// Build table rows from commits
	rows := make([][]string, 0, len(result.Commits))
	for _, c := range result.Commits {
		rows = append(rows, []string{c.Short, c.Subject})
	}

	// Render commit table
	printer.Table([]string{"SHA", "Subject"}, rows)

	// Summary with count
	printer.Println()
	printer.KeyValue("Count", strconv.Itoa(result.Count))

	if result.LastEntry != nil {
		printer.KeyValue("Since", result.LastEntry.ID)
	}

	// Suggest command
	printer.Println()
	printer.Println("Run 'timbers log \"<what>\" --why \"<why>\" --how \"<how>\"' to document this work")
}
