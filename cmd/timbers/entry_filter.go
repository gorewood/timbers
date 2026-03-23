// Package main provides the entry point for the timbers CLI.
package main

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// filterEntriesSince filters entries to those created at or after the cutoff.
func filterEntriesSince(entries []*ledger.Entry, cutoff time.Time) []*ledger.Entry {
	return ledger.FilterEntriesSince(entries, cutoff)
}

// filterEntriesUntil filters entries to those created before or at the cutoff.
func filterEntriesUntil(entries []*ledger.Entry, cutoff time.Time) []*ledger.Entry {
	return ledger.FilterEntriesUntil(entries, cutoff)
}

// filterEntriesByTags filters entries to those that have at least one matching tag.
// Uses OR logic: entries matching ANY of the specified tags are included.
func filterEntriesByTags(entries []*ledger.Entry, tags []string) []*ledger.Entry {
	return ledger.FilterEntriesByTags(entries, tags)
}

// entryHasAnyTag checks if the entry has any of the specified tags.
func entryHasAnyTag(entry *ledger.Entry, tags []string) bool {
	return ledger.EntryHasAnyTag(entry, tags)
}

// sortEntriesByCreatedAt sorts entries by created_at descending (most recent first).
func sortEntriesByCreatedAt(entries []*ledger.Entry) {
	ledger.SortEntriesByCreatedAt(entries)
}

// getEntriesByTimeRange retrieves entries within the time range, with optional limit and tag filtering.
//
//nolint:unparam // tagFlags will be used by callers beyond export
func getEntriesByTimeRange(
	printer *output.Printer, storage *ledger.Storage,
	sinceCutoff, untilCutoff time.Time, lastFlag string, tagFlags []string,
) ([]*ledger.Entry, error) {
	entries, err := storage.ListEntries()
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	if !sinceCutoff.IsZero() {
		entries = ledger.FilterEntriesSince(entries, sinceCutoff)
	}
	if !untilCutoff.IsZero() {
		entries = ledger.FilterEntriesUntil(entries, untilCutoff)
	}
	if len(tagFlags) > 0 {
		entries = ledger.FilterEntriesByTags(entries, tagFlags)
	}

	ledger.SortEntriesByCreatedAt(entries)

	if lastFlag != "" {
		count, parseErr := strconv.Atoi(lastFlag)
		if parseErr == nil && count > 0 && len(entries) > count {
			entries = entries[:count]
		}
	}

	return entries, nil
}

// getEntriesByLast retrieves the last N entries with optional tag filtering.
func getEntriesByLast(printer *output.Printer, storage *ledger.Storage, lastFlag string, tagFlags []string) ([]*ledger.Entry, error) {
	count, parseErr := strconv.Atoi(lastFlag)
	if parseErr != nil || count <= 0 {
		err := output.NewUserError("--last must be a positive integer")
		printer.Error(err)
		return nil, err
	}

	// If tag filtering is needed, we can't use the optimized path
	if len(tagFlags) > 0 {
		entries, err := storage.ListEntries()
		if err != nil {
			printer.Error(err)
			return nil, err
		}
		entries = ledger.FilterEntriesByTags(entries, tagFlags)
		ledger.SortEntriesByCreatedAt(entries)
		if len(entries) > count {
			entries = entries[:count]
		}
		return entries, nil
	}

	// Optimized path when no tag filtering
	entries, err := storage.GetLastNEntries(count)
	if err != nil {
		printer.Error(err)
		return nil, err
	}
	return entries, nil
}

// getEntriesByRange retrieves entries whose commits fall within the given range.
// First tries anchor-based matching (entry's workset commits in the range).
// Falls back to file-based discovery via git diff when anchor matching returns
// nothing — this handles squash merges where entry files are present but their
// anchor commits reference feature branch SHAs no longer in main's history.
func getEntriesByRange(printer *output.Printer, storage *ledger.Storage, rangeFlag string) ([]*ledger.Entry, error) {
	parts := strings.Split(rangeFlag, "..")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		err := output.NewUserError("--range must be in format A..B")
		printer.Error(err)
		return nil, err
	}

	fromRef, toRef := parts[0], parts[1]

	allEntries, err := storage.ListEntries()
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	// Primary path: match entries by anchor commit ancestry
	commits, err := storage.LogRange(fromRef, toRef)
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	commitSet := make(map[string]bool, len(commits))
	for _, commit := range commits {
		commitSet[commit.SHA] = true
	}

	entries := filterEntriesByCommits(allEntries, commitSet)
	if len(entries) > 0 {
		return entries, nil
	}

	// Fallback: discover entries by file presence in the diff.
	// Handles squash merges where anchor commits aren't in the current branch.
	return getEntriesByDiff(printer, storage, allEntries, fromRef, toRef)
}

// getEntriesByDiff discovers entries introduced in a commit range by checking
// which .timbers/ files were added or changed. This is the fallback path for
// squash merges where entry anchor commits aren't in the current branch history.
func getEntriesByDiff(
	printer *output.Printer, storage *ledger.Storage,
	allEntries []*ledger.Entry, fromRef, toRef string,
) ([]*ledger.Entry, error) {
	files, err := storage.DiffNameOnly(fromRef, toRef, ".timbers/")
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	// Build a set of entry IDs from the diff file paths
	idSet := make(map[string]bool, len(files))
	for _, f := range files {
		base := filepath.Base(f)
		if id, ok := strings.CutSuffix(base, ".json"); ok {
			idSet[id] = true
		}
	}

	var entries []*ledger.Entry
	for _, entry := range allEntries {
		if idSet[entry.ID] {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

// filterEntriesByCommits returns entries that have at least one commit in the given set.
func filterEntriesByCommits(allEntries []*ledger.Entry, commitSet map[string]bool) []*ledger.Entry {
	var entries []*ledger.Entry
	for _, entry := range allEntries {
		if entryInCommitSet(entry, commitSet) {
			entries = append(entries, entry)
		}
	}
	return entries
}

// entryInCommitSet checks if any commit in entry's workset is in the set.
func entryInCommitSet(entry *ledger.Entry, commitSet map[string]bool) bool {
	for _, commitSHA := range entry.Workset.Commits {
		if commitSet[commitSHA] {
			return true
		}
	}
	return false
}
