// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/steveyegge/timbers/internal/git"
	"github.com/steveyegge/timbers/internal/ledger"
)

// mockGitOpsForPending implements ledger.GitOps for testing pending command.
type mockGitOpsForPending struct {
	head            string
	headErr         error
	commits         []git.Commit
	commitsErr      error
	reachableResult []git.Commit
	reachableErr    error
	notes           map[string][]byte
}

func (m *mockGitOpsForPending) ReadNote(commit string) ([]byte, error) {
	if data, ok := m.notes[commit]; ok {
		return data, nil
	}
	return nil, nil
}

func (m *mockGitOpsForPending) WriteNote(string, string, bool) error {
	return nil
}

func (m *mockGitOpsForPending) ListNotedCommits() ([]string, error) {
	commits := make([]string, 0, len(m.notes))
	for commit := range m.notes {
		commits = append(commits, commit)
	}
	return commits, nil
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

func (m *mockGitOpsForPending) PushNotes(remote string) error {
	return nil
}

func TestPendingCommand(t *testing.T) {
	tests := []struct {
		name           string
		mock           *mockGitOpsForPending
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
				head:  "abc123def456",
				notes: map[string][]byte{},
				reachableResult: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "Third commit"},
					{SHA: "def456789012", Short: "def4567", Subject: "Second commit"},
					{SHA: "789012345678", Short: "7890123", Subject: "First commit"},
				},
			},
			wantCount:    3,
			wantContains: []string{"3 pending", "abc123d", "Third commit"},
		},
		{
			name: "has entry - shows commits since anchor",
			mock: &mockGitOpsForPending{
				head: "abc123def456",
				notes: map[string][]byte{
					"oldanchor1234": createTestEntry("oldanchor1234", time.Now().Add(-1*time.Hour)),
				},
				commits: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "New commit"},
					{SHA: "def456789012", Short: "def4567", Subject: "Another new commit"},
				},
			},
			wantCount:    2,
			wantContains: []string{"2 pending", "abc123d", "New commit"},
		},
		{
			name: "no pending commits",
			mock: &mockGitOpsForPending{
				head: "abc123def456",
				notes: map[string][]byte{
					"abc123def456": createTestEntry("abc123def456", time.Now()),
				},
				commits: []git.Commit{},
			},
			wantCount:    0,
			wantContains: []string{"No pending commits"},
		},
		{
			name: "count flag - shows count only",
			mock: &mockGitOpsForPending{
				head:  "abc123def456",
				notes: map[string][]byte{},
				reachableResult: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "Third commit"},
					{SHA: "def456789012", Short: "def4567", Subject: "Second commit"},
				},
			},
			countOnly:      true,
			wantCount:      2,
			wantContains:   []string{"2"},
			wantNotContain: []string{"abc123d", "Third commit"},
		},
		{
			name: "json output - structured format",
			mock: &mockGitOpsForPending{
				head: "abc123def456",
				notes: map[string][]byte{
					"oldanchor1234": createTestEntry("oldanchor1234", time.Now().Add(-1*time.Hour)),
				},
				commits: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "New commit"},
				},
			},
			jsonOutput:   true,
			wantCount:    1,
			wantContains: []string{`"count": 1`, `"commits":`},
		},
		{
			name: "json output - no pending",
			mock: &mockGitOpsForPending{
				head: "abc123def456",
				notes: map[string][]byte{
					"abc123def456": createTestEntry("abc123def456", time.Now()),
				},
				commits: []git.Commit{},
			},
			jsonOutput:   true,
			wantCount:    0,
			wantContains: []string{`"count": 0`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flag
			jsonFlag = tt.jsonOutput

			// Create storage with mock
			storage := ledger.NewStorage(tt.mock)

			// Create command
			cmd := newPendingCmdWithStorage(storage)

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

// createTestEntry creates a minimal valid entry for testing.
func createTestEntry(anchor string, created time.Time) []byte {
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
			What: "Test entry",
			Why:  "For testing",
			How:  "Via test",
		},
	}
	data, _ := entry.ToJSON()
	return data
}

// newPendingCmdWithStorage is a helper for tests that injects a storage.
func newPendingCmdWithStorage(storage *ledger.Storage) *cobra.Command {
	return newPendingCmdInternal(storage)
}
