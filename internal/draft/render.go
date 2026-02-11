package draft

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
)

// RenderContext provides data for template rendering.
type RenderContext struct {
	Entries            []*ledger.Entry
	RepoName           string
	Branch             string
	AppendText         string // Optional extra instructions from --append
	TotalEntries       int    // Total entries in repo (for detecting first batch)
	IsFirstBatch       bool   // True if entries include the chronologically earliest
	ProjectDescription string // Brief project description for context
}

// Render substitutes variables in the template content.
func Render(tmpl *Template, ctx *RenderContext) (string, error) {
	vars, err := buildVars(ctx)
	if err != nil {
		return "", err
	}

	result := tmpl.Content
	for key, val := range vars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", val)
	}

	// Append extra instructions if provided
	if ctx.AppendText != "" {
		result = result + "\n\n## Additional Instructions\n\n" + ctx.AppendText
	}

	return result, nil
}

// buildVars creates the variable map for substitution.
func buildVars(ctx *RenderContext) (map[string]string, error) {
	vars := make(map[string]string)

	// entries_json - full JSON array
	entriesJSON, err := json.MarshalIndent(ctx.Entries, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entries: %w", err)
	}
	vars["entries_json"] = string(entriesJSON)

	// entries_summary - compact text format
	vars["entries_summary"] = buildEntriesSummary(ctx.Entries)

	// entry_count
	vars["entry_count"] = strconv.Itoa(len(ctx.Entries))

	// date_range
	vars["date_range"] = buildDateRange(ctx.Entries)

	// repo_name
	vars["repo_name"] = ctx.RepoName

	// branch
	vars["branch"] = ctx.Branch

	// total_entries
	vars["total_entries"] = strconv.Itoa(ctx.TotalEntries)

	// is_first_batch
	if ctx.IsFirstBatch {
		vars["is_first_batch"] = "true"
	} else {
		vars["is_first_batch"] = "false"
	}

	// project_description
	vars["project_description"] = ctx.ProjectDescription

	return vars, nil
}

// buildEntriesSummary creates a compact text representation of entries.
func buildEntriesSummary(entries []*ledger.Entry) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		date := ""
		if !entry.CreatedAt.IsZero() {
			date = entry.CreatedAt.Format("2006-01-02")
		}
		line := fmt.Sprintf("- [%s] %s: %s (Why: %s)",
			date, entry.ID, entry.Summary.What, entry.Summary.Why)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// buildDateRange returns a human-readable date range.
func buildDateRange(entries []*ledger.Entry) string {
	if len(entries) == 0 {
		return "no entries"
	}

	var earliest, latest time.Time
	for _, entry := range entries {
		if earliest.IsZero() || entry.CreatedAt.Before(earliest) {
			earliest = entry.CreatedAt
		}
		if latest.IsZero() || entry.CreatedAt.After(latest) {
			latest = entry.CreatedAt
		}
	}

	if earliest.IsZero() || latest.IsZero() {
		return "unknown date range"
	}

	earliestStr := earliest.Format("2006-01-02")
	latestStr := latest.Format("2006-01-02")

	if earliestStr == latestStr {
		return earliestStr
	}
	return earliestStr + " to " + latestStr
}
