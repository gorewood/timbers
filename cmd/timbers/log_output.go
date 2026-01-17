// Package main provides the entry point for the timbers CLI.
package main

import (
	"strings"
	"time"

	"github.com/rbergman/timbers/internal/ledger"
	"github.com/rbergman/timbers/internal/output"
)

// outputDryRun outputs what would be written without actually writing.
func outputDryRun(printer *output.Printer, entry *ledger.Entry) error {
	if jsonFlag {
		return printer.Success(map[string]any{
			"status": "dry_run",
			"entry":  entryToMap(entry),
		})
	}

	printer.Println("Dry run - would create entry:")
	printer.Println()
	printer.Print("  ID:     %s\n", entry.ID)
	printer.Print("  Anchor: %s\n", entry.Workset.AnchorCommit)
	printer.Print("  What:   %s\n", entry.Summary.What)
	printer.Print("  Why:    %s\n", entry.Summary.Why)
	printer.Print("  How:    %s\n", entry.Summary.How)
	outputDryRunOptionalFields(printer, entry)
	printer.Print("  Files:  %d changed, +%d -%d\n",
		entry.Workset.Diffstat.Files,
		entry.Workset.Diffstat.Insertions,
		entry.Workset.Diffstat.Deletions)
	printer.Println()

	return nil
}

// outputDryRunOptionalFields outputs optional entry fields in dry-run mode.
func outputDryRunOptionalFields(printer *output.Printer, entry *ledger.Entry) {
	if len(entry.Tags) > 0 {
		printer.Print("  Tags:   %s\n", strings.Join(entry.Tags, ", "))
	}
	if len(entry.WorkItems) > 0 {
		items := make([]string, len(entry.WorkItems))
		for i, wi := range entry.WorkItems {
			items[i] = wi.System + ":" + wi.ID
		}
		printer.Print("  Work:   %s\n", strings.Join(items, ", "))
	}
}

// outputLogSuccess outputs the success result.
func outputLogSuccess(printer *output.Printer, entry *ledger.Entry, pushedMsg string) error {
	if jsonFlag {
		commitSHAs := make([]string, len(entry.Workset.Commits))
		copy(commitSHAs, entry.Workset.Commits)
		return printer.Success(map[string]any{
			"status":  "created",
			"id":      entry.ID,
			"anchor":  entry.Workset.AnchorCommit,
			"commits": commitSHAs,
		})
	}

	printer.Print("Created entry %s%s\n", entry.ID, pushedMsg)
	printer.Print("  %s\n", entry.Summary.What)

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
