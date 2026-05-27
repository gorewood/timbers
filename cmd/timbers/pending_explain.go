package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// runPendingExplain handles `timbers pending --explain`: classify every commit
// in the display range (kept vs skip reason) so the user can see why each is
// or isn't pending — e.g. confirm a new .timbersignore rule exempts a commit.
func runPendingExplain(storage *ledger.Storage, printer *output.Printer) error {
	classified, latest, err := storage.ExplainPending()
	if err != nil && !errors.Is(err, ledger.ErrStaleAnchor) {
		printer.Error(err)
		return err
	}
	if errors.Is(err, ledger.ErrStaleAnchor) {
		return outputStaleAnchor(printer, latest)
	}

	if printer.IsJSON() {
		rows := make([]map[string]any, 0, len(classified))
		for _, item := range classified {
			rows = append(rows, map[string]any{
				"sha":     item.Commit.SHA,
				"short":   item.Commit.Short,
				"subject": item.Commit.Subject,
				"kept":    item.Reason == "",
				"reason":  item.Reason,
			})
		}
		return printer.Success(map[string]any{"commits": rows, "count": len(classified)})
	}

	outputExplainHuman(printer, classified)
	return nil
}

// outputExplainHuman prints the per-commit classification table plus a summary.
func outputExplainHuman(printer *output.Printer, classified []ledger.ClassifiedCommit) {
	if len(classified) == 0 {
		printer.Println("No commits in range — all caught up.")
		return
	}
	printer.Println("Pending classification (since last entry):")
	printer.Println()
	kept := 0
	reasons := map[string]int{}
	for _, item := range classified {
		label := "KEEP"
		if item.Reason == "" {
			kept++
		} else {
			label = "skip " + item.Reason
			reasons[item.Reason]++
		}
		printer.Println("  " + item.Commit.Short + "  " + padLabel(label) + "  " + item.Commit.Subject)
	}
	printer.Println()
	printer.Println("Kept (pending): " + strconv.Itoa(kept) +
		"   Skipped: " + strconv.Itoa(len(classified)-kept) + formatReasonBreakdown(reasons))
}

// padLabel right-pads a classification label to a fixed width for column
// alignment in the explain output.
func padLabel(label string) string {
	const width = 16
	if len(label) >= width {
		return label
	}
	return label + strings.Repeat(" ", width-len(label))
}

// formatReasonBreakdown renders a skip-reason tally as " (author:2, infra:1)",
// or "" when nothing was skipped.
func formatReasonBreakdown(reasons map[string]int) string {
	order := []string{"infra", "author", "message", "documented", "ack", "revert", "merge-empty", "empty"}
	var parts []string
	for _, r := range order {
		if n := reasons[r]; n > 0 {
			parts = append(parts, r+":"+strconv.Itoa(n))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}
