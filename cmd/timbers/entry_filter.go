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
// Uses two discovery strategies and unions the results:
//  1. Anchor-based: entry's workset commits appear in git rev-list A..B
//  2. File-based: entry file appears in git diff --name-only A..B -- .timbers/
//
// Both paths are always run because a squash merge can leave some entries with
// valid anchors (e.g., from a prior session on main) while others have stale
// anchors from the feature branch. Running only anchor-based and falling back
// on zero results misses the partial-stale case.
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

	// Path 1: match entries by anchor commit ancestry
	commits, err := storage.LogRange(fromRef, toRef)
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	commitSet := make(map[string]bool, len(commits))
	for _, commit := range commits {
		commitSet[commit.SHA] = true
	}

	anchorEntries := filterEntriesByCommits(allEntries, commitSet)

	// Path 2: discover entries by file presence in the diff.
	// On failure, anchor results alone are still valid — don't propagate the error.
	diffEntries, _ := getEntriesByDiff(printer, storage, allEntries, fromRef, toRef)

	// Union both result sets
	return unionEntries(anchorEntries, diffEntries), nil
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

	// Build a set of entry IDs from the diff file paths.
	// Filenames may be in canonical (dashed) or legacy (colon) form; convert
	// both to the canonical ID before comparing against entry.ID.
	idSet := make(map[string]bool, len(files))
	for _, f := range files {
		base := filepath.Base(f)
		if name, ok := strings.CutSuffix(base, ".json"); ok {
			idSet[ledger.FilenameToID(name)] = true
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

// unionEntries merges two entry slices, deduplicating by ID.
func unionEntries(a, b []*ledger.Entry) []*ledger.Entry {
	seen := make(map[string]bool, len(a))
	result := make([]*ledger.Entry, 0, len(a)+len(b))

	for _, e := range a {
		seen[e.ID] = true
		result = append(result, e)
	}
	for _, e := range b {
		if !seen[e.ID] {
			result = append(result, e)
		}
	}
	return result
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
