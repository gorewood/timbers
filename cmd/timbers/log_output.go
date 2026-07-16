// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// outputDryRun outputs what would be written without actually writing.
// The human path renders the entry as an aligned panel (rounded box at a TTY,
// borderless plain text when piped) with substance leading and bookkeeping
// (ID, Anchor) at the bottom.
func outputDryRun(printer *output.Printer, entry *ledger.Entry) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status": "dry_run",
			"entry":  entryToMap(entry),
		})
	}

	printer.FieldsBox("Dry Run Preview", dryRunFields(entry))
	return nil
}

// formatDiffstat formats a diffstat as a human-readable string.
func formatDiffstat(ds *ledger.Diffstat) string {
	if ds == nil {
		return "0 changed"
	}
	return fmt.Sprintf("%d changed, +%d -%d", ds.Files, ds.Insertions, ds.Deletions)
}

// outputLogSuccess outputs the success result.
func outputLogSuccess(printer *output.Printer, entry *ledger.Entry) error {
	if printer.IsJSON() {
		commitSHAs := make([]string, len(entry.Workset.Commits))
		copy(commitSHAs, entry.Workset.Commits)
		return printer.Success(map[string]any{
			"status":  "created",
			"id":      entry.ID,
			"anchor":  entry.Workset.AnchorCommit,
			"commits": commitSHAs,
			"suggested_commands": []string{
				"timbers show --latest",
			},
		})
	}

	_ = printer.Success(map[string]any{"message": "Created entry " + entry.ID})
	printer.Println("  " + entry.Summary.What)

	return nil
}

// entryToMap converts an Entry to a map for JSON output.
func entryToMap(entry *ledger.Entry) map[string]any {
	workset := map[string]any{
		"anchor_commit": entry.Workset.AnchorCommit,
		"commits":       entry.Workset.Commits,
		"range":         entry.Workset.Range,
	}

	if entry.Workset.Diffstat != nil {
		workset["diffstat"] = map[string]any{
			"files":      entry.Workset.Diffstat.Files,
			"insertions": entry.Workset.Diffstat.Insertions,
			"deletions":  entry.Workset.Diffstat.Deletions,
		}
	}

	result := map[string]any{
		"schema":     entry.Schema,
		"kind":       entry.Kind,
		"id":         entry.ID,
		"created_at": entry.CreatedAt.Format(time.RFC3339),
		"workset":    workset,
		"summary": map[string]any{
			"what": entry.Summary.What,
			"why":  entry.Summary.Why,
			"how":  entry.Summary.How,
		},
	}

	if len(entry.Tags) > 0 {
		result["tags"] = entry.Tags
	}

	if len(entry.WorkItems) > 0 {
		items := make([]map[string]string, len(entry.WorkItems))
		for i, wi := range entry.WorkItems {
			items[i] = map[string]string{"system": wi.System, "id": wi.ID}
		}
		result["work_items"] = items
	}

	if len(entry.Contributors) > 0 {
		result["contributors"] = entry.Contributors
	}

	return result
}
