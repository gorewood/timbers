// Package main provides the entry point for the timbers CLI.
package main

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// workItemTrailerRegex matches Work-item trailers in commit bodies.
// Format: Work-item: system:id (case-insensitive)
var workItemTrailerRegex = regexp.MustCompile(`(?i)^work-item:\s*(\S+:\S+)\s*$`)

// commitGroup represents a group of commits to process as one entry.
type commitGroup struct {
	key     string       // Group identifier (work-item or date)
	commits []git.Commit // Commits in this group (newest first)
}

// batchResult represents the result of a batch operation.
type batchResult struct {
	Status  string          `json:"status"`
	Count   int             `json:"count"`
	Entries []batchEntryRef `json:"entries"`
}

// batchEntryRef is a lightweight reference to a created entry.
type batchEntryRef struct {
	ID       string `json:"id"`
	Anchor   string `json:"anchor"`
	GroupKey string `json:"group_key"`
	What     string `json:"what"`
}

// runBatchLog processes pending commits in batches grouped by work-item or day.
func runBatchLog(storage *ledger.Storage, flags logFlags, printer *output.Printer) error {
	// Get pending commits
	commits, err := getBatchCommits(storage, flags)
	if err != nil {
		printer.Error(err)
		return err
	}

	if len(commits) == 0 {
		err := output.NewUserError("no pending commits to document; run 'timbers pending' to check status")
		printer.Error(err)
		return err
	}

	// Group commits by work-item trailer or by day
	groups := groupCommits(commits)

	if len(groups) == 0 {
		err := output.NewUserError("no groups found for batch processing")
		printer.Error(err)
		return err
	}

	// Process each group
	return processBatchGroups(storage, groups, flags, printer)
}

// getBatchCommits retrieves pending commits for batch processing.
func getBatchCommits(storage *ledger.Storage, flags logFlags) ([]git.Commit, error) {
	if flags.rangeStr != "" {
		parts := strings.SplitN(flags.rangeStr, "..", 2)
		fromRef := parts[0]
		toRef := parts[1]
		return storage.LogRange(fromRef, toRef)
	}

	commits, _, err := storage.GetPendingCommits()
	return commits, err
}

// GroupStrategy defines how commits are grouped: auto, day, or work-item.
type GroupStrategy string

const (
	GroupStrategyAuto     GroupStrategy = "auto"      // work-item first, fallback to day
	GroupStrategyDay      GroupStrategy = "day"       // group by YYYY-MM-DD
	GroupStrategyWorkItem GroupStrategy = "work-item" // group by Work-item trailer
)

// groupCommits groups commits using auto strategy (work-item first, fallback to day).
func groupCommits(commits []git.Commit) []commitGroup {
	return groupCommitsByStrategy(commits, GroupStrategyAuto)
}

// groupCommitsByStrategy groups commits using the specified strategy.
func groupCommitsByStrategy(commits []git.Commit, strategy GroupStrategy) []commitGroup {
	switch strategy {
	case GroupStrategyDay:
		return groupCommitsByDay(commits)
	case GroupStrategyWorkItem:
		return groupCommitsByTrailer(commits)
	case GroupStrategyAuto:
		if groups := groupCommitsByTrailer(commits); len(groups) > 0 {
			return groups
		}
		return groupCommitsByDay(commits)
	}
	return nil // unreachable with valid strategy
}

// groupCommitsByTrailer groups commits by Work-item trailer found in commit body.
// Returns empty slice if no trailers found.
func groupCommitsByTrailer(commits []git.Commit) []commitGroup {
	groups := make(map[string][]git.Commit)

	for _, commit := range commits {
		workItem := extractWorkItemTrailer(commit.Body)
		if workItem != "" {
			groups[workItem] = append(groups[workItem], commit)
		}
	}

	// If no trailers found at all, return empty
	if len(groups) == 0 {
		return nil
	}

	// Handle commits without trailers - add to "untracked" group
	for _, commit := range commits {
		workItem := extractWorkItemTrailer(commit.Body)
		if workItem == "" {
			groups["untracked"] = append(groups["untracked"], commit)
		}
	}

	return mapToSortedGroups(groups)
}

// extractWorkItemTrailer extracts the Work-item trailer value from a commit body.
func extractWorkItemTrailer(body string) string {
	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		matches := workItemTrailerRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return matches[1]
		}
	}
	return ""
}

// groupCommitsByDay groups commits by their date (YYYY-MM-DD format).
func groupCommitsByDay(commits []git.Commit) []commitGroup {
	groups := make(map[string][]git.Commit)

	for _, commit := range commits {
		day := commit.Date.Format("2006-01-02")
		groups[day] = append(groups[day], commit)
	}

	return mapToSortedGroups(groups)
}

// mapToSortedGroups converts a map of groups to a sorted slice (newest/highest keys first).
func mapToSortedGroups(groups map[string][]git.Commit) []commitGroup {
	result := make([]commitGroup, 0, len(groups))
	for key, groupCommits := range groups {
		result = append(result, commitGroup{key: key, commits: groupCommits})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].key > result[j].key })
	return result
}

// processBatchGroups processes each group and creates entries.
func processBatchGroups(
	storage *ledger.Storage,
	groups []commitGroup,
	flags logFlags,
	printer *output.Printer,
) error {
	var entries []batchEntryRef

	for _, group := range groups {
		entry, err := processBatchGroup(storage, group, flags, printer)
		if err != nil {
			return err
		}

		entries = append(entries, batchEntryRef{
			ID:       entry.ID,
			Anchor:   entry.Workset.AnchorCommit,
			GroupKey: group.key,
			What:     entry.Summary.What,
		})
	}

	return outputBatchResult(printer, entries, flags.dryRun)
}

// processBatchGroup creates an entry for a single group of commits.
func processBatchGroup(
	storage *ledger.Storage,
	group commitGroup,
	flags logFlags,
	printer *output.Printer,
) (*ledger.Entry, error) {
	entry := buildBatchEntry(storage, group, flags.tags)

	if flags.dryRun {
		return entry, nil
	}

	if err := storage.WriteEntry(entry, false); err != nil {
		printer.Error(err)
		return nil, err
	}

	return entry, nil
}

// buildBatchEntry constructs a ledger entry from a commit group.
func buildBatchEntry(storage *ledger.Storage, group commitGroup, tags []string) *ledger.Entry {
	what, why, how := extractAutoContent(group.commits)
	workItems := extractWorkItemsFromKey(group.key)
	anchor := group.commits[0].SHA
	diffstat := getBatchDiffstat(storage, group.commits, anchor)

	now := time.Now().UTC()
	commitSHAs := extractCommitSHAs(group.commits)
	rangeStr := buildCommitRange(group.commits)

	return &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        ledger.GenerateID(anchor, now),
		CreatedAt: now,
		UpdatedAt: now,
		Workset: ledger.Workset{
			AnchorCommit: anchor,
			Commits:      commitSHAs,
			Range:        rangeStr,
			Diffstat: &ledger.Diffstat{
				Files:      diffstat.Files,
				Insertions: diffstat.Insertions,
				Deletions:  diffstat.Deletions,
			},
		},
		Summary: ledger.Summary{
			What: what,
			Why:  why,
			How:  how,
		},
		Tags:      tags,
		WorkItems: workItems,
	}
}

// extractWorkItemsFromKey extracts work items from a group key if applicable.
func extractWorkItemsFromKey(key string) []ledger.WorkItem {
	if !isWorkItemKey(key) {
		return nil
	}
	system, id, err := parseWorkItem(key)
	if err != nil {
		return nil
	}
	return []ledger.WorkItem{{System: system, ID: id}}
}

// getBatchDiffstat retrieves diffstat for a batch group.
func getBatchDiffstat(storage *ledger.Storage, commits []git.Commit, anchor string) git.Diffstat {
	if len(commits) == 0 {
		return git.Diffstat{}
	}
	fromRef := commits[len(commits)-1].SHA + "^"
	diffstat, err := storage.GetDiffstat(fromRef, anchor)
	if err != nil {
		return git.Diffstat{}
	}
	return diffstat
}

// extractCommitSHAs extracts SHA strings from commits.
func extractCommitSHAs(commits []git.Commit) []string {
	shas := make([]string, len(commits))
	for i, commit := range commits {
		shas[i] = commit.SHA
	}
	return shas
}

// buildCommitRange builds a commit range string.
func buildCommitRange(commits []git.Commit) string {
	if len(commits) <= 1 {
		return ""
	}
	return commits[len(commits)-1].Short + ".." + commits[0].Short
}

// isWorkItemKey checks if a group key represents a work-item (vs a date or "untracked").
func isWorkItemKey(key string) bool {
	if key == "untracked" {
		return false
	}
	// Check if it looks like a date (YYYY-MM-DD)
	if len(key) == 10 && key[4] == '-' && key[7] == '-' {
		return false
	}
	// Must contain : to be a work-item
	return strings.Contains(key, ":")
}

// outputBatchResult outputs the batch processing result.
func outputBatchResult(printer *output.Printer, entries []batchEntryRef, isDryRun bool) error {
	status := "created"
	if isDryRun {
		status = "dry_run"
	}

	if printer.IsJSON() {
		return printer.WriteJSON(batchResult{
			Status:  status,
			Count:   len(entries),
			Entries: entries,
		})
	}

	// Human-readable output
	if isDryRun {
		printer.Print("Dry run - would create %d entries:\n", len(entries))
	} else {
		printer.Print("Created %d entries:\n", len(entries))
	}

	for _, e := range entries {
		printer.Print("  %s [%s] %s\n", e.ID, e.GroupKey, truncateString(e.What, 50))
	}

	return nil
}

// truncateString truncates a string to maxLen, adding "..." if truncated.
func truncateString(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}
