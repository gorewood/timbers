// Package main provides the entry point for the timbers CLI.
package main

import (
	"strings"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// mockGitOpsForQuery implements ledger.GitOps for testing query command.
type mockGitOpsForQuery struct {
	notes map[string][]byte
}

func (m *mockGitOpsForQuery) ReadNote(commit string) ([]byte, error) {
	if data, ok := m.notes[commit]; ok {
		return data, nil
	}
	return nil, nil
}

func (m *mockGitOpsForQuery) WriteNote(string, string, bool) error {
	return nil
}

func (m *mockGitOpsForQuery) ListNotedCommits() ([]string, error) {
	commits := make([]string, 0, len(m.notes))
	for commit := range m.notes {
		commits = append(commits, commit)
	}
	return commits, nil
}

func (m *mockGitOpsForQuery) HEAD() (string, error) {
	return "abc123def456", nil
}

func (m *mockGitOpsForQuery) Log(fromRef, toRef string) ([]git.Commit, error) {
	return nil, nil
}

func (m *mockGitOpsForQuery) CommitsReachableFrom(sha string) ([]git.Commit, error) {
	return nil, nil
}

func (m *mockGitOpsForQuery) GetDiffstat(fromRef, toRef string) (git.Diffstat, error) {
	return git.Diffstat{}, nil
}

func (m *mockGitOpsForQuery) PushNotes(remote string) error {
	return nil
}

// TestQueryCommand tests the query command with various inputs.
func TestQueryCommand(t *testing.T) {
	now := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)

	tests := []struct {
		name           string
		lastFlag       string
		tagFlags       []string
		onelineFlag    bool
		jsonOutput     bool
		notes          map[string][]byte
		wantErr        bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:         "no --last flag",
			lastFlag:     "",
			notes:        map[string][]byte{},
			wantErr:      true,
			wantContains: []string{"specify --last N, --since"},
		},
		{
			name:         "--last with zero",
			lastFlag:     "0",
			notes:        map[string][]byte{},
			wantErr:      true,
			wantContains: []string{"--last must be a positive integer"},
		},
		{
			name:         "--last with negative",
			lastFlag:     "-5",
			notes:        map[string][]byte{},
			wantErr:      true,
			wantContains: []string{"--last must be a positive integer"},
		},
		{
			name:         "--last with non-integer",
			lastFlag:     "abc",
			notes:        map[string][]byte{},
			wantErr:      true,
			wantContains: []string{"--last must be a positive integer"},
		},
		{
			name:     "empty entries",
			lastFlag: "5",
			notes:    map[string][]byte{},
			wantErr:  false,
		},
		{
			name:     "--last 1 with 5 entries",
			lastFlag: "1",
			notes: map[string][]byte{
				"anchor1": createQueryTestEntry("anchor1", "first", now.Add(-4*time.Hour)),
				"anchor2": createQueryTestEntry("anchor2", "second", now.Add(-3*time.Hour)),
				"anchor3": createQueryTestEntry("anchor3", "third", now.Add(-2*time.Hour)),
				"anchor4": createQueryTestEntry("anchor4", "fourth", now.Add(-1*time.Hour)),
				"anchor5": createQueryTestEntry("anchor5", "fifth", now),
			},
			wantErr:      false,
			wantContains: []string{"fifth"},
		},
		{
			name:     "--last 3 with 5 entries",
			lastFlag: "3",
			notes: map[string][]byte{
				"anchor1": createQueryTestEntry("anchor1", "first", now.Add(-4*time.Hour)),
				"anchor2": createQueryTestEntry("anchor2", "second", now.Add(-3*time.Hour)),
				"anchor3": createQueryTestEntry("anchor3", "third", now.Add(-2*time.Hour)),
				"anchor4": createQueryTestEntry("anchor4", "fourth", now.Add(-1*time.Hour)),
				"anchor5": createQueryTestEntry("anchor5", "fifth", now),
			},
			wantErr:        false,
			wantContains:   []string{"third", "fourth", "fifth"},
			wantNotContain: []string{"first", "second"},
		},
		{
			name:     "--last 10 with 5 entries",
			lastFlag: "10",
			notes: map[string][]byte{
				"anchor1": createQueryTestEntry("anchor1", "first", now.Add(-4*time.Hour)),
				"anchor2": createQueryTestEntry("anchor2", "second", now.Add(-3*time.Hour)),
				"anchor3": createQueryTestEntry("anchor3", "third", now.Add(-2*time.Hour)),
				"anchor4": createQueryTestEntry("anchor4", "fourth", now.Add(-1*time.Hour)),
				"anchor5": createQueryTestEntry("anchor5", "fifth", now),
			},
			wantErr:      false,
			wantContains: []string{"first", "second", "third", "fourth", "fifth"},
		},
		{
			name:        "--oneline output",
			lastFlag:    "2",
			onelineFlag: true,
			notes: map[string][]byte{
				"anchor1": createQueryTestEntry("anchor1", "first", now.Add(-1*time.Hour)),
				"anchor2": createQueryTestEntry("anchor2", "second", now),
			},
			wantErr:      false,
			wantContains: []string{"first", "second"},
		},
		{
			name:       "--json output",
			lastFlag:   "2",
			jsonOutput: true,
			notes: map[string][]byte{
				"anchor1": createQueryTestEntry("anchor1", "first", now.Add(-1*time.Hour)),
				"anchor2": createQueryTestEntry("anchor2", "second", now),
			},
			wantErr:      false,
			wantContains: []string{`"id"`, `"summary"`},
		},
		{
			name:     "filter by single tag",
			lastFlag: "10",
			tagFlags: []string{"security"},
			notes: map[string][]byte{
				"anchor1": createQueryTestEntryWithTags("anchor1", "first", now.Add(-4*time.Hour), []string{"security", "auth"}),
				"anchor2": createQueryTestEntryWithTags("anchor2", "second", now.Add(-3*time.Hour), []string{"feature"}),
				"anchor3": createQueryTestEntryWithTags("anchor3", "third", now.Add(-2*time.Hour), []string{"security"}),
				"anchor4": createQueryTestEntry("anchor4", "fourth", now.Add(-1*time.Hour)),
				"anchor5": createQueryTestEntryWithTags("anchor5", "fifth", now, []string{"bugfix"}),
			},
			wantErr:        false,
			wantContains:   []string{"first", "third"},
			wantNotContain: []string{"second", "fourth", "fifth"},
		},
		{
			name:     "filter by multiple tags (OR logic)",
			lastFlag: "10",
			tagFlags: []string{"security", "bugfix"},
			notes: map[string][]byte{
				"anchor1": createQueryTestEntryWithTags("anchor1", "first", now.Add(-4*time.Hour), []string{"security", "auth"}),
				"anchor2": createQueryTestEntryWithTags("anchor2", "second", now.Add(-3*time.Hour), []string{"feature"}),
				"anchor3": createQueryTestEntryWithTags("anchor3", "third", now.Add(-2*time.Hour), []string{"security"}),
				"anchor4": createQueryTestEntry("anchor4", "fourth", now.Add(-1*time.Hour)),
				"anchor5": createQueryTestEntryWithTags("anchor5", "fifth", now, []string{"bugfix"}),
			},
			wantErr:        false,
			wantContains:   []string{"first", "third", "fifth"},
			wantNotContain: []string{"second", "fourth"},
		},
		{
			name:     "filter by tag with no matches",
			lastFlag: "10",
			tagFlags: []string{"nonexistent"},
			notes: map[string][]byte{
				"anchor1": createQueryTestEntryWithTags("anchor1", "first", now, []string{"security"}),
				"anchor2": createQueryTestEntryWithTags("anchor2", "second", now, []string{"feature"}),
			},
			wantErr: false,
		},
		{
			name:     "filter by tag with entries that have no tags",
			lastFlag: "10",
			tagFlags: []string{"security"},
			notes: map[string][]byte{
				"anchor1": createQueryTestEntry("anchor1", "first", now.Add(-1*time.Hour)),
				"anchor2": createQueryTestEntryWithTags("anchor2", "second", now, []string{"security"}),
			},
			wantErr:        false,
			wantContains:   []string{"second"},
			wantNotContain: []string{"first"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create storage with mock
			storage := ledger.NewStorage(&mockGitOpsForQuery{notes: tt.notes})

			// Create command
			cmd := newQueryCmdInternal(storage)

			// Set JSON mode for testing
			if tt.jsonOutput {
				cmd.PersistentFlags().Bool("json", false, "")
				_ = cmd.PersistentFlags().Set("json", "true")
			}

			// Set flags
			if err := cmd.Flags().Set("last", tt.lastFlag); err != nil {
				t.Fatalf("failed to set last flag: %v", err)
			}
			if tt.onelineFlag {
				if err := cmd.Flags().Set("oneline", "true"); err != nil {
					t.Fatalf("failed to set oneline flag: %v", err)
				}
			}
			for _, tag := range tt.tagFlags {
				if err := cmd.Flags().Set("tag", tag); err != nil {
					t.Fatalf("failed to set tag flag: %v", err)
				}
			}

			// Capture output
			var buf strings.Builder
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Execute
			err := cmd.Execute()

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()

			// Check expected content
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected content %q\noutput: %s", want, output)
				}
			}

			// Check content that should not appear
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(output, notWant) {
					t.Errorf("output contains unexpected content %q\noutput: %s", notWant, output)
				}
			}
		})
	}
}

// createQueryTestEntry creates a minimal valid entry for testing query command.
func createQueryTestEntry(anchor, what string, created time.Time) []byte {
	return createQueryTestEntryWithTags(anchor, what, created, nil)
}

// createQueryTestEntryWithTags creates a valid entry with tags for testing query command.
func createQueryTestEntryWithTags(anchor, what string, created time.Time, tags []string) []byte {
	entry := &ledger.Entry{
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
			Why:  "Testing query",
			How:  "Via test",
		},
		Tags: tags,
	}
	data, _ := entry.ToJSON()
	return data
}
