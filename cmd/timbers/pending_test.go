// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// mockGitOpsForPending implements ledger.GitOps for testing pending command.
type mockGitOpsForPending struct {
	head            string
	headErr         error
	commits         []git.Commit
	commitsErr      error
	reachableResult []git.Commit
	reachableErr    error
}

func (m *mockGitOpsForPending) HEAD() (string, error) {
	return m.head, m.headErr
}

func (m *mockGitOpsForPending) Log(fromRef, toRef string) ([]git.Commit, error) {
	return m.commits, m.commitsErr
}

func (m *mockGitOpsForPending) CommitsReachableFrom(sha string) ([]git.Commit, error) {
	return m.reachableResult, m.reachableErr
}

func (m *mockGitOpsForPending) GetDiffstat(fromRef, toRef string) (git.Diffstat, error) {
	return git.Diffstat{}, nil
}

func (m *mockGitOpsForPending) CommitFiles(sha string) ([]string, error) { return nil, nil }

func TestPendingCommand(t *testing.T) {
	// Helper to create a test entry struct.
	makeEntry := func(anchor string, created time.Time) *ledger.Entry {
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
				What: "Test entry",
				Why:  "For testing",
				How:  "Via test",
			},
		}
	}

	// Helper to write entry files into a temp dir and return FileStorage.
	writeEntries := func(t *testing.T, entries ...*ledger.Entry) *ledger.FileStorage {
		t.Helper()
		dir := t.TempDir()
		for _, entry := range entries {
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
		return ledger.NewFileStorage(dir, func(_ string) error { return nil }, func(_, _ string) error { return nil })
	}

	tests := []struct {
		name           string
		mock           *mockGitOpsForPending
		files          func(t *testing.T) *ledger.FileStorage
		countOnly      bool
		jsonOutput     bool
		wantCount      int
		wantErr        bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "no entries - shows all reachable commits",
			mock: &mockGitOpsForPending{
				head: "abc123def456",
				reachableResult: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "Third commit"},
					{SHA: "def456789012", Short: "def4567", Subject: "Second commit"},
					{SHA: "789012345678", Short: "7890123", Subject: "First commit"},
				},
			},
			files:        nil,
			wantCount:    3,
			wantContains: []string{"Pending Commits", "Count: 3", "abc123d", "Third commit"},
		},
		{
			name: "has entry - shows commits since anchor",
			mock: &mockGitOpsForPending{
				head: "abc123def456",
				commits: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "New commit"},
					{SHA: "def456789012", Short: "def4567", Subject: "Another new commit"},
				},
			},
			files: func(t *testing.T) *ledger.FileStorage {
				return writeEntries(t, makeEntry("oldanchor1234", time.Now().Add(-1*time.Hour)))
			},
			wantCount:    2,
			wantContains: []string{"Pending Commits", "Count: 2", "abc123d", "New commit"},
		},
		{
			name: "no pending commits",
			mock: &mockGitOpsForPending{
				head:    "abc123def456",
				commits: []git.Commit{},
			},
			files: func(t *testing.T) *ledger.FileStorage {
				return writeEntries(t, makeEntry("abc123def456", time.Now()))
			},
			wantCount:    0,
			wantContains: []string{"No pending commits"},
		},
		{
			name: "count flag - shows count only",
			mock: &mockGitOpsForPending{
				head: "abc123def456",
				reachableResult: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "Third commit"},
					{SHA: "def456789012", Short: "def4567", Subject: "Second commit"},
				},
			},
			files:          nil,
			countOnly:      true,
			wantCount:      2,
			wantContains:   []string{"2"},
			wantNotContain: []string{"abc123d", "Third commit"},
		},
		{
			name: "json output - structured format",
			mock: &mockGitOpsForPending{
				head: "abc123def456",
				commits: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "New commit"},
				},
			},
			files: func(t *testing.T) *ledger.FileStorage {
				return writeEntries(t, makeEntry("oldanchor1234", time.Now().Add(-1*time.Hour)))
			},
			jsonOutput:   true,
			wantCount:    1,
			wantContains: []string{`"count": 1`, `"commits":`},
		},
		{
			name: "json output - no pending",
			mock: &mockGitOpsForPending{
				head:    "abc123def456",
				commits: []git.Commit{},
			},
			files: func(t *testing.T) *ledger.FileStorage {
				return writeEntries(t, makeEntry("abc123def456", time.Now()))
			},
			jsonOutput:   true,
			wantCount:    0,
			wantContains: []string{`"count": 0`},
		},
		{
			name: "stale anchor - falls back with warning",
			mock: &mockGitOpsForPending{
				head:       "abc123def456",
				commitsErr: errors.New("bad revision 'staleanchor..abc123def456'"),
				reachableResult: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "Recent commit"},
					{SHA: "def456789012", Short: "def4567", Subject: "Older commit"},
				},
			},
			files: func(t *testing.T) *ledger.FileStorage {
				return writeEntries(t, makeEntry("staleanchor1234", time.Now().Add(-1*time.Hour)))
			},
			wantCount:    2,
			wantContains: []string{"anchor commit is no longer in git history", "Pending Commits", "abc123d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build FileStorage from entries if provided
			var files *ledger.FileStorage
			if tt.files != nil {
				files = tt.files(t)
			}

			// Create storage with mock and file storage
			storage := ledger.NewStorage(tt.mock, files)

			// Create command
			cmd := newPendingCmdWithStorage(storage)

			// Set JSON mode for testing
			if tt.jsonOutput {
				cmd.PersistentFlags().Bool("json", false, "")
				_ = cmd.PersistentFlags().Set("json", "true")
			}

			// Set flags
			if tt.countOnly {
				if err := cmd.Flags().Set("count", "true"); err != nil {
					t.Fatalf("failed to set count flag: %v", err)
				}
			}

			// Capture output
			var buf bytes.Buffer
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

			// For JSON output, verify structure
			if tt.jsonOutput && err == nil {
				var result map[string]any
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("failed to parse JSON output: %v\noutput: %s", err, output)
				}
				if count, ok := result["count"].(float64); ok {
					if int(count) != tt.wantCount {
						t.Errorf("JSON count = %v, want %v", count, tt.wantCount)
					}
				}
			}
		})
	}
}

// newPendingCmdWithStorage is a helper for tests that injects a storage.
func newPendingCmdWithStorage(storage *ledger.Storage) *cobra.Command {
	return newPendingCmdInternal(storage)
}
