// Package main provides the entry point for the timbers CLI.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// mockGitOpsForQuery implements ledger.GitOps for testing query command.
type mockGitOpsForQuery struct{}

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

func (m *mockGitOpsForQuery) CommitFiles(sha string) ([]string, error) { return nil, nil }

// writeQueryEntryFile writes an entry JSON file to the correct date subdirectory.
func writeQueryEntryFile(t *testing.T, dir string, entry *ledger.Entry) {
	t.Helper()
	data, err := entry.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize entry: %v", err)
	}
	entryDir := dir
	if sub := ledger.EntryDateDir(entry.ID); sub != "" {
		entryDir = filepath.Join(dir, sub)
	}
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("failed to create entry dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, entry.ID+".json"), data, 0o600); err != nil {
		t.Fatalf("failed to write entry file: %v", err)
	}
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
		entries        []*ledger.Entry
		wantErr        bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:         "no --last flag",
			lastFlag:     "",
			entries:      nil,
			wantErr:      true,
			wantContains: []string{"specify --last N, --since"},
		},
		{
			name:         "--last with zero",
			lastFlag:     "0",
			entries:      nil,
			wantErr:      true,
			wantContains: []string{"--last must be a positive integer"},
		},
		{
			name:         "--last with negative",
			lastFlag:     "-5",
			entries:      nil,
			wantErr:      true,
			wantContains: []string{"--last must be a positive integer"},
		},
		{
			name:         "--last with non-integer",
			lastFlag:     "abc",
			entries:      nil,
			wantErr:      true,
			wantContains: []string{"--last must be a positive integer"},
		},
		{
			name:     "empty entries",
			lastFlag: "5",
			entries:  nil,
			wantErr:  false,
		},
		{
			name:     "--last 1 with 5 entries",
			lastFlag: "1",
			entries: []*ledger.Entry{
				createQueryTestEntryStruct("anchor1", "first", now.Add(-4*time.Hour)),
				createQueryTestEntryStruct("anchor2", "second", now.Add(-3*time.Hour)),
				createQueryTestEntryStruct("anchor3", "third", now.Add(-2*time.Hour)),
				createQueryTestEntryStruct("anchor4", "fourth", now.Add(-1*time.Hour)),
				createQueryTestEntryStruct("anchor5", "fifth", now),
			},
			wantErr:      false,
			wantContains: []string{"fifth"},
		},
		{
			name:     "--last 3 with 5 entries",
			lastFlag: "3",
			entries: []*ledger.Entry{
				createQueryTestEntryStruct("anchor1", "first", now.Add(-4*time.Hour)),
				createQueryTestEntryStruct("anchor2", "second", now.Add(-3*time.Hour)),
				createQueryTestEntryStruct("anchor3", "third", now.Add(-2*time.Hour)),
				createQueryTestEntryStruct("anchor4", "fourth", now.Add(-1*time.Hour)),
				createQueryTestEntryStruct("anchor5", "fifth", now),
			},
			wantErr:        false,
			wantContains:   []string{"third", "fourth", "fifth"},
			wantNotContain: []string{"first", "second"},
		},
		{
			name:     "--last 10 with 5 entries",
			lastFlag: "10",
			entries: []*ledger.Entry{
				createQueryTestEntryStruct("anchor1", "first", now.Add(-4*time.Hour)),
				createQueryTestEntryStruct("anchor2", "second", now.Add(-3*time.Hour)),
				createQueryTestEntryStruct("anchor3", "third", now.Add(-2*time.Hour)),
				createQueryTestEntryStruct("anchor4", "fourth", now.Add(-1*time.Hour)),
				createQueryTestEntryStruct("anchor5", "fifth", now),
			},
			wantErr:      false,
			wantContains: []string{"first", "second", "third", "fourth", "fifth"},
		},
		{
			name:        "--oneline output",
			lastFlag:    "2",
			onelineFlag: true,
			entries: []*ledger.Entry{
				createQueryTestEntryStruct("anchor1", "first", now.Add(-1*time.Hour)),
				createQueryTestEntryStruct("anchor2", "second", now),
			},
			wantErr:      false,
			wantContains: []string{"first", "second"},
		},
		{
			name:       "--json output",
			lastFlag:   "2",
			jsonOutput: true,
			entries: []*ledger.Entry{
				createQueryTestEntryStruct("anchor1", "first", now.Add(-1*time.Hour)),
				createQueryTestEntryStruct("anchor2", "second", now),
			},
			wantErr:      false,
			wantContains: []string{`"id"`, `"summary"`},
		},
		{
			name:     "filter by single tag",
			lastFlag: "10",
			tagFlags: []string{"security"},
			entries: []*ledger.Entry{
				createQueryTestEntryStructWithTags("anchor1", "first", now.Add(-4*time.Hour), []string{"security", "auth"}),
				createQueryTestEntryStructWithTags("anchor2", "second", now.Add(-3*time.Hour), []string{"feature"}),
				createQueryTestEntryStructWithTags("anchor3", "third", now.Add(-2*time.Hour), []string{"security"}),
				createQueryTestEntryStruct("anchor4", "fourth", now.Add(-1*time.Hour)),
				createQueryTestEntryStructWithTags("anchor5", "fifth", now, []string{"bugfix"}),
			},
			wantErr:        false,
			wantContains:   []string{"first", "third"},
			wantNotContain: []string{"second", "fourth", "fifth"},
		},
		{
			name:     "filter by multiple tags (OR logic)",
			lastFlag: "10",
			tagFlags: []string{"security", "bugfix"},
			entries: []*ledger.Entry{
				createQueryTestEntryStructWithTags("anchor1", "first", now.Add(-4*time.Hour), []string{"security", "auth"}),
				createQueryTestEntryStructWithTags("anchor2", "second", now.Add(-3*time.Hour), []string{"feature"}),
				createQueryTestEntryStructWithTags("anchor3", "third", now.Add(-2*time.Hour), []string{"security"}),
				createQueryTestEntryStruct("anchor4", "fourth", now.Add(-1*time.Hour)),
				createQueryTestEntryStructWithTags("anchor5", "fifth", now, []string{"bugfix"}),
			},
			wantErr:        false,
			wantContains:   []string{"first", "third", "fifth"},
			wantNotContain: []string{"second", "fourth"},
		},
		{
			name:     "filter by tag with no matches",
			lastFlag: "10",
			tagFlags: []string{"nonexistent"},
			entries: []*ledger.Entry{
				createQueryTestEntryStructWithTags("anchor1", "first", now, []string{"security"}),
				createQueryTestEntryStructWithTags("anchor2", "second", now, []string{"feature"}),
			},
			wantErr: false,
		},
		{
			name:     "filter by tag with entries that have no tags",
			lastFlag: "10",
			tagFlags: []string{"security"},
			entries: []*ledger.Entry{
				createQueryTestEntryStruct("anchor1", "first", now.Add(-1*time.Hour)),
				createQueryTestEntryStructWithTags("anchor2", "second", now, []string{"security"}),
			},
			wantErr:        false,
			wantContains:   []string{"second"},
			wantNotContain: []string{"first"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create storage with file-backed entries
			var files *ledger.FileStorage
			if tt.entries != nil {
				dir := t.TempDir()
				for _, entry := range tt.entries {
					writeQueryEntryFile(t, dir, entry)
				}
				files = ledger.NewFileStorage(dir, func(_ string) error { return nil })
			}
			storage := ledger.NewStorage(&mockGitOpsForQuery{}, files)

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

// createQueryTestEntryStruct creates a minimal valid entry struct for testing query command.
func createQueryTestEntryStruct(anchor, what string, created time.Time) *ledger.Entry {
	return createQueryTestEntryStructWithTags(anchor, what, created, nil)
}

// createQueryTestEntryStructWithTags creates a valid entry struct with tags for testing query command.
func createQueryTestEntryStructWithTags(anchor, what string, created time.Time, tags []string) *ledger.Entry {
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
			Why:  "Testing query",
			How:  "Via test",
		},
		Tags: tags,
	}
}
