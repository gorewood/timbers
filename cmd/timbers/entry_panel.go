// Package main provides the entry point for the timbers CLI.
package main

import (
	"strconv"
	"strings"

	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// substanceFields builds the shared What/Why/How(/Notes/Tags/Work) rows that
// lead both the show and dry-run panels. What and Why are emphasized so the
// substance of the entry reads first; optional rows appear only when set.
func substanceFields(entry *ledger.Entry) []output.Field {
	fields := []output.Field{
		{Key: "What", Value: entry.Summary.What, Emphasis: true},
		{Key: "Why", Value: entry.Summary.Why, Emphasis: true},
		{Key: "How", Value: entry.Summary.How},
	}
	if entry.Notes != "" {
		fields = append(fields, output.Field{Key: "Notes", Value: entry.Notes})
	}
	if len(entry.Tags) > 0 {
		fields = append(fields, output.Field{Key: "Tags", Value: strings.Join(entry.Tags, ", ")})
	}
	if work := formatWorkItems(entry.WorkItems); work != "" {
		fields = append(fields, output.Field{Key: "Work", Value: work})
	}
	return fields
}

// dryRunFields builds the field rows for the `log --dry-run` panel: substance
// first, then the diffstat, a separator, and the bookkeeping (ID, Anchor) at
// the bottom. The box title carries the status, so the ID lives in the body.
func dryRunFields(entry *ledger.Entry) []output.Field {
	fields := substanceFields(entry)
	fields = append(fields,
		output.Field{Key: "Files", Value: formatDiffstat(entry.Workset.Diffstat)},
		output.Separator(),
		output.Field{Key: "ID", Value: entry.ID},
		output.Field{Key: "Anchor", Value: shortSHA(entry.Workset.AnchorCommit)},
	)
	return fields
}

// showFields builds the field rows for `timbers show`: substance first, a
// separator, then the workset bookkeeping. The entry ID is the panel title
// (it is the thing you copy), so it is not repeated in the body.
func showFields(entry *ledger.Entry) []output.Field {
	fields := substanceFields(entry)
	fields = append(fields, output.Separator())
	fields = append(fields, output.Field{Key: "Anchor", Value: anchorDisplay(entry.Workset.AnchorCommit)})
	if len(entry.Workset.Commits) > 0 {
		commits := strconv.Itoa(len(entry.Workset.Commits))
		if entry.Workset.Range != "" {
			commits += " (" + entry.Workset.Range + ")"
		}
		fields = append(fields, output.Field{Key: "Commits", Value: commits})
	}
	if entry.Workset.Diffstat != nil {
		fields = append(fields, output.Field{Key: "Files", Value: formatDiffstat(entry.Workset.Diffstat)})
	}
	fields = append(fields, output.Field{Key: "Created", Value: entry.CreatedAt.Format("2006-01-02 15:04:05 UTC")})
	return fields
}

// formatWorkItems renders work items as "system:id, system:id".
func formatWorkItems(items []ledger.WorkItem) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, len(items))
	for i, wi := range items {
		parts[i] = wi.System + ":" + wi.ID
	}
	return strings.Join(parts, ", ")
}
