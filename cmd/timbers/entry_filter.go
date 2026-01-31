// Package main provides the entry point for the timbers CLI.
package main

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rbergman/timbers/internal/ledger"
	"github.com/rbergman/timbers/internal/output"
)

// filterEntriesSince filters entries to those created at or after the cutoff.
func filterEntriesSince(entries []*ledger.Entry, cutoff time.Time) []*ledger.Entry {
	var result []*ledger.Entry
	for _, entry := range entries {
		if entry.CreatedAt.After(cutoff) || entry.CreatedAt.Equal(cutoff) {
			result = append(result, entry)
		}
	}
	return result
}

// filterEntriesUntil filters entries to those created before or at the cutoff.
func filterEntriesUntil(entries []*ledger.Entry, cutoff time.Time) []*ledger.Entry {
	var result []*ledger.Entry
	for _, entry := range entries {
		if entry.CreatedAt.Before(cutoff) || entry.CreatedAt.Equal(cutoff) {
			result = append(result, entry)
		}
	}
	return result
}

// sortEntriesByCreatedAt sorts entries by created_at descending (most recent first).
func sortEntriesByCreatedAt(entries []*ledger.Entry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})
}

// getEntriesByTimeRange retrieves entries within the time range, with optional limit.
func getEntriesByTimeRange(
	printer *output.Printer, storage *ledger.Storage,
	sinceCutoff, untilCutoff time.Time, lastFlag string,
) ([]*ledger.Entry, error) {
	entries, err := storage.ListEntries()
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	if !sinceCutoff.IsZero() {
		entries = filterEntriesSince(entries, sinceCutoff)
	}
	if !untilCutoff.IsZero() {
		entries = filterEntriesUntil(entries, untilCutoff)
	}

	sortEntriesByCreatedAt(entries)

	if lastFlag != "" {
		count, parseErr := strconv.Atoi(lastFlag)
		if parseErr == nil && count > 0 && len(entries) > count {
			entries = entries[:count]
		}
	}

	return entries, nil
}

// getEntriesByLast retrieves the last N entries.
func getEntriesByLast(printer *output.Printer, storage *ledger.Storage, lastFlag string) ([]*ledger.Entry, error) {
	count, parseErr := strconv.Atoi(lastFlag)
	if parseErr != nil || count <= 0 {
		err := output.NewUserError("--last must be a positive integer")
		printer.Error(err)
		return nil, err
	}

	entries, err := storage.GetLastNEntries(count)
	if err != nil {
		printer.Error(err)
		return nil, err
	}
	return entries, nil
}

// getEntriesByRange retrieves entries whose commits fall within the given range.
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

	commits, err := storage.LogRange(fromRef, toRef)
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	commitSet := make(map[string]bool, len(commits))
	for _, commit := range commits {
		commitSet[commit.SHA] = true
	}

	return filterEntriesByCommits(allEntries, commitSet), nil
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
