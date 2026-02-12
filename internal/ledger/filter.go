// Package ledger provides the entry schema, validation, and serialization for the timbers development ledger.
package ledger

import (
	"slices"
	"sort"
	"time"
)

// FilterEntriesSince filters entries to those created at or after the cutoff.
func FilterEntriesSince(entries []*Entry, cutoff time.Time) []*Entry {
	var result []*Entry
	for _, entry := range entries {
		if entry.CreatedAt.After(cutoff) || entry.CreatedAt.Equal(cutoff) {
			result = append(result, entry)
		}
	}
	return result
}

// FilterEntriesUntil filters entries to those created before or at the cutoff.
func FilterEntriesUntil(entries []*Entry, cutoff time.Time) []*Entry {
	var result []*Entry
	for _, entry := range entries {
		if entry.CreatedAt.Before(cutoff) || entry.CreatedAt.Equal(cutoff) {
			result = append(result, entry)
		}
	}
	return result
}

// FilterEntriesByTags filters entries to those that have at least one matching tag.
// Uses OR logic: entries matching ANY of the specified tags are included.
func FilterEntriesByTags(entries []*Entry, tags []string) []*Entry {
	if len(tags) == 0 {
		return entries
	}

	var result []*Entry
	for _, entry := range entries {
		if EntryHasAnyTag(entry, tags) {
			result = append(result, entry)
		}
	}
	return result
}

// EntryHasAnyTag checks if the entry has any of the specified tags.
func EntryHasAnyTag(entry *Entry, tags []string) bool {
	for _, entryTag := range entry.Tags {
		if slices.Contains(tags, entryTag) {
			return true
		}
	}
	return false
}

// SortEntriesByCreatedAt sorts entries by created_at descending (most recent first).
func SortEntriesByCreatedAt(entries []*Entry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})
}
