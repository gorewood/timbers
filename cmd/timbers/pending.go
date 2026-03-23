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

	// Stale anchor: don't show the fallback commit list — it's not actionable
	// and confuses agents into re-documenting already-covered work.
	if errors.Is(err, ledger.ErrStaleAnchor) {
		return outputStaleAnchor(printer, latest)
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

// outputStaleAnchor handles the stale anchor case — reports 0 actionable
// pending with clear guidance instead of dumping a confusing commit list.
func outputStaleAnchor(printer *output.Printer, latest *ledger.Entry) error {
	if printer.IsJSON() {
		data := map[string]any{
			"count":  0,
			"status": "stale_anchor",
			"message": "Anchor commit no longer in history (squash merge or rebase). " +
				"No action needed — anchor self-heals on next timbers log.",
		}
		if latest != nil {
			data["last_entry"] = &entryReference{
				ID:           latest.ID,
				AnchorCommit: latest.Workset.AnchorCommit,
				CreatedAt:    latest.CreatedAt.Format("2006-01-02T15:04:05Z"),
			}
		}
		return printer.Success(data)
	}

	printer.Warn("Anchor commit no longer in history (likely squash merge or rebase)")
	printer.Println("No action needed — do not re-document. The anchor self-heals on your next timbers log.")
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
	// No entries yet — report clean state so agents detect fresh install.
	if result.LastEntry == nil {
		return printer.Success(map[string]any{
			"count":   0,
			"status":  "no_entries",
			"commits": []commitSummary{},
		})
	}

	data := map[string]any{
		"count":   result.Count,
		"commits": result.Commits,
	}
	data["last_entry"] = result.LastEntry

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
	// No entries yet — fresh install, show friendly message instead of
	// dumping the entire pre-timbers history as "pending" work.
	if result.LastEntry == nil {
		printer.Println("No entries yet — tracking starts with your first timbers log.")
		printer.Println("Tip: Run 'timbers catchup' to backfill existing history (optional).")
		return
	}

	// Handle no pending commits (all caught up)
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
