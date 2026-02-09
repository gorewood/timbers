// Package main provides the entry point for the timbers CLI.
package main

import (
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
)

// TestFilterEntriesByTags tests the tag filtering function.
func TestFilterEntriesByTags(t *testing.T) {
	now := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)

	// Create test entries with various tag combinations
	entries := []*ledger.Entry{
		createFilterTestEntry("entry1", "First entry", now, []string{"security", "auth"}),
		createFilterTestEntry("entry2", "Second entry", now, []string{"feature", "api"}),
		createFilterTestEntry("entry3", "Third entry", now, []string{"security"}),
		createFilterTestEntry("entry4", "Fourth entry", now, []string{}),
		createFilterTestEntry("entry5", "Fifth entry", now, []string{"bugfix", "critical"}),
		createFilterTestEntry("entry6", "Sixth entry", now, nil),
	}

	tests := []struct {
		name    string
		entries []*ledger.Entry
		tags    []string
		wantIDs []string
		wantLen int
	}{
		{
			name:    "empty tags returns all entries",
			entries: entries,
			tags:    []string{},
			wantLen: 6,
		},
		{
			name:    "nil tags returns all entries",
			entries: entries,
			tags:    nil,
			wantLen: 6,
		},
		{
			name:    "single tag match",
			entries: entries,
			tags:    []string{"security"},
			wantIDs: []string{"entry1", "entry3"},
			wantLen: 2,
		},
		{
			name:    "multiple tags (OR logic)",
			entries: entries,
			tags:    []string{"security", "bugfix"},
			wantIDs: []string{"entry1", "entry3", "entry5"},
			wantLen: 3,
		},
		{
			name:    "no matches",
			entries: entries,
			tags:    []string{"nonexistent"},
			wantIDs: []string{},
			wantLen: 0,
		},
		{
			name:    "match one of multiple tags on entry",
			entries: entries,
			tags:    []string{"auth"},
			wantIDs: []string{"entry1"},
			wantLen: 1,
		},
		{
			name:    "multiple search tags with overlapping matches",
			entries: entries,
			tags:    []string{"security", "auth"},
			wantIDs: []string{"entry1", "entry3"},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterEntriesByTags(tt.entries, tt.tags)

			if len(result) != tt.wantLen {
				t.Errorf("filterEntriesByTags() returned %d entries, want %d", len(result), tt.wantLen)
			}

			// Check that all expected IDs are present
			if tt.wantIDs != nil {
				gotIDs := make(map[string]bool)
				for _, entry := range result {
					gotIDs[entry.Workset.AnchorCommit] = true
				}

				for _, wantID := range tt.wantIDs {
					if !gotIDs[wantID] {
						t.Errorf("filterEntriesByTags() missing expected entry %q", wantID)
					}
				}

				// Check no unexpected entries
				if len(gotIDs) != len(tt.wantIDs) {
					t.Errorf("filterEntriesByTags() returned %d entries, want %d", len(gotIDs), len(tt.wantIDs))
				}
			}
		})
	}
}

// TestEntryHasAnyTag tests the helper function that checks for tag matches.
func TestEntryHasAnyTag(t *testing.T) {
	now := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)

	tests := []struct {
		name       string
		entryTags  []string
		searchTags []string
		want       bool
	}{
		{
			name:       "exact match",
			entryTags:  []string{"security"},
			searchTags: []string{"security"},
			want:       true,
		},
		{
			name:       "match one of many entry tags",
			entryTags:  []string{"security", "auth", "api"},
			searchTags: []string{"auth"},
			want:       true,
		},
		{
			name:       "match one of many search tags",
			entryTags:  []string{"security"},
			searchTags: []string{"feature", "security", "bugfix"},
			want:       true,
		},
		{
			name:       "no match",
			entryTags:  []string{"security"},
			searchTags: []string{"feature"},
			want:       false,
		},
		{
			name:       "entry has no tags",
			entryTags:  []string{},
			searchTags: []string{"security"},
			want:       false,
		},
		{
			name:       "entry has nil tags",
			entryTags:  nil,
			searchTags: []string{"security"},
			want:       false,
		},
		{
			name:       "empty search tags",
			entryTags:  []string{"security"},
			searchTags: []string{},
			want:       false,
		},
		{
			name:       "multiple matches returns true",
			entryTags:  []string{"security", "auth"},
			searchTags: []string{"security", "auth"},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := createFilterTestEntry("test", "Test entry", now, tt.entryTags)
			got := entryHasAnyTag(entry, tt.searchTags)

			if got != tt.want {
				t.Errorf("entryHasAnyTag() = %v, want %v", got, tt.want)
			}
		})
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
