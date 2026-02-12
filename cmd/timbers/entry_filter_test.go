// Package main provides the entry point for the timbers CLI.
package main

import (
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
)

// TestFilterEntriesByTags tests the thin wrapper delegates correctly.
func TestFilterEntriesByTags(t *testing.T) {
	now := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)

	entries := []*ledger.Entry{
		createFilterTestEntry("entry1", "First entry", now, []string{"security", "auth"}),
		createFilterTestEntry("entry2", "Second entry", now, []string{"feature"}),
	}

	result := filterEntriesByTags(entries, []string{"security"})
	if len(result) != 1 {
		t.Errorf("filterEntriesByTags() returned %d entries, want 1", len(result))
	}

	result = filterEntriesByTags(entries, nil)
	if len(result) != 2 {
		t.Errorf("filterEntriesByTags(nil) returned %d entries, want 2", len(result))
	}
}

// TestEntryHasAnyTag tests the thin wrapper delegates correctly.
func TestEntryHasAnyTag(t *testing.T) {
	now := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)
	entry := createFilterTestEntry("test", "Test entry", now, []string{"security"})

	if !entryHasAnyTag(entry, []string{"security"}) {
		t.Error("entryHasAnyTag() = false, want true")
	}
	if entryHasAnyTag(entry, []string{"feature"}) {
		t.Error("entryHasAnyTag() = true, want false")
	}
}

// createFilterTestEntry creates a minimal valid entry for testing filters.
func createFilterTestEntry(anchor, what string, created time.Time, tags []string) *ledger.Entry {
	return &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        ledger.GenerateID(anchor, created),
		CreatedAt: created,
		UpdatedAt: created,
		Workset: ledger.Workset{
			AnchorCommit: anchor,
			Commits:      []string{anchor},
		},
		Summary: ledger.Summary{
			What: what,
			Why:  "Testing",
			How:  "Via test",
		},
		Tags: tags,
	}
}
