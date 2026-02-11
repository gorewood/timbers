// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// outputDryRun outputs what would be written without actually writing.
func outputDryRun(printer *output.Printer, entry *ledger.Entry) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status": "dry_run",
			"entry":  entryToMap(entry),
		})
	}

	printer.Section("Dry Run Preview")
	printer.KeyValue("ID", entry.ID)
	printer.KeyValue("Anchor", entry.Workset.AnchorCommit)
	printer.KeyValue("What", entry.Summary.What)
	printer.KeyValue("Why", entry.Summary.Why)
	printer.KeyValue("How", entry.Summary.How)
	outputDryRunOptionalFields(printer, entry)
	printer.KeyValue("Files", formatDiffstat(entry.Workset.Diffstat))

	return nil
}

// outputDryRunOptionalFields outputs optional entry fields in dry-run mode.
func outputDryRunOptionalFields(printer *output.Printer, entry *ledger.Entry) {
	if len(entry.Tags) > 0 {
		printer.KeyValue("Tags", strings.Join(entry.Tags, ", "))
	}
	if len(entry.WorkItems) > 0 {
		items := make([]string, len(entry.WorkItems))
		for i, wi := range entry.WorkItems {
			items[i] = wi.System + ":" + wi.ID
		}
		printer.KeyValue("Work", strings.Join(items, ", "))
	}
}

// formatDiffstat formats a diffstat as a human-readable string.
func formatDiffstat(ds *ledger.Diffstat) string {
	if ds == nil {
		return "0 changed"
	}
	return fmt.Sprintf("%d changed, +%d -%d", ds.Files, ds.Insertions, ds.Deletions)
}

// outputLogSuccess outputs the success result.
func outputLogSuccess(printer *output.Printer, entry *ledger.Entry, pushedMsg string) error {
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

	_ = printer.Success(map[string]any{"message": "Created entry " + entry.ID + pushedMsg})
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

	return result
}
