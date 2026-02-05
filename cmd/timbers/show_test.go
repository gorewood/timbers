// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/spf13/cobra"
)

// mockGitOpsForShow implements ledger.GitOps for testing show command.
type mockGitOpsForShow struct {
	notes map[string][]byte
}

func (m *mockGitOpsForShow) ReadNote(commit string) ([]byte, error) {
	if data, ok := m.notes[commit]; ok {
		return data, nil
	}
	return nil, nil
}

func (m *mockGitOpsForShow) WriteNote(string, string, bool) error {
	return nil
}

func (m *mockGitOpsForShow) ListNotedCommits() ([]string, error) {
	commits := make([]string, 0, len(m.notes))
	for commit := range m.notes {
		commits = append(commits, commit)
	}
	return commits, nil
}

func (m *mockGitOpsForShow) HEAD() (string, error) {
	return "abc123def456", nil
}

func (m *mockGitOpsForShow) Log(fromRef, toRef string) ([]git.Commit, error) {
	return nil, nil
}

func (m *mockGitOpsForShow) CommitsReachableFrom(sha string) ([]git.Commit, error) {
	return nil, nil
}

func (m *mockGitOpsForShow) GetDiffstat(fromRef, toRef string) (git.Diffstat, error) {
	return git.Diffstat{}, nil
}

func (m *mockGitOpsForShow) PushNotes(remote string) error {
	return nil
}

func TestShowCommand(t *testing.T) {
	now := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)
	testEntry := createShowTestEntry("anchor123456", now)
	testEntryID := ledger.GenerateID("anchor123456", now)

	tests := []struct {
		name           string
		mock           *mockGitOpsForShow
		args           []string
		lastFlag       bool
		jsonOutput     bool
		wantErr        bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "show by ID - found",
			mock: &mockGitOpsForShow{
				notes: map[string][]byte{
					"anchor123456": testEntry,
				},
			},
			args:         []string{testEntryID},
			wantContains: []string{testEntryID, "Test entry", "For testing", "Via test", "anchor1"},
		},
		{
			name: "show by ID - not found",
			mock: &mockGitOpsForShow{
				notes: map[string][]byte{},
			},
			args:         []string{"nonexistent-id"},
			wantErr:      true,
			wantContains: []string{"entry not found"},
		},
		{
			name: "show --latest - found",
			mock: &mockGitOpsForShow{
				notes: map[string][]byte{
					"anchor123456": testEntry,
				},
			},
			lastFlag:     true,
			wantContains: []string{testEntryID, "Test entry"},
		},
		{
			name: "show --latest - no entries",
			mock: &mockGitOpsForShow{
				notes: map[string][]byte{},
			},
			lastFlag:     true,
			wantErr:      true,
			wantContains: []string{"no entries found"},
		},
		{
			name: "no ID and no --latest flag",
			mock: &mockGitOpsForShow{
				notes: map[string][]byte{},
			},
			wantErr:      true,
			wantContains: []string{"specify an entry ID or use --latest"},
		},
		{
			name: "both ID and --latest flag",
			mock: &mockGitOpsForShow{
				notes: map[string][]byte{
					"anchor123456": testEntry,
				},
			},
			args:         []string{testEntryID},
			lastFlag:     true,
			wantErr:      true,
			wantContains: []string{"cannot use both ID argument and --latest flag"},
		},
		{
			name: "show --json - structured output",
			mock: &mockGitOpsForShow{
				notes: map[string][]byte{
					"anchor123456": testEntry,
				},
			},
			args:         []string{testEntryID},
			jsonOutput:   true,
			wantContains: []string{`"id"`, `"summary"`, `"what"`, `"why"`, `"how"`},
		},
		{
			name: "show --latest --json",
			mock: &mockGitOpsForShow{
				notes: map[string][]byte{
					"anchor123456": testEntry,
				},
			},
			lastFlag:     true,
			jsonOutput:   true,
			wantContains: []string{`"id"`, `"schema"`, `"workset"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flag
			jsonFlag = tt.jsonOutput

			// Create storage with mock
			storage := ledger.NewStorage(tt.mock)

			// Create command
			cmd := newShowCmdWithStorage(storage)

			// Set flags
			if tt.lastFlag {
				if err := cmd.Flags().Set("latest", "true"); err != nil {
					t.Fatalf("failed to set last flag: %v", err)
				}
			}

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

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
				// Verify entry fields are present
				if _, ok := result["id"]; !ok {
					t.Error("JSON output missing 'id' field")
				}
				if _, ok := result["summary"]; !ok {
					t.Error("JSON output missing 'summary' field")
				}
			}
		})
	}
}

func TestShowWithTags(t *testing.T) {
	now := time.Now().UTC()
	entry := &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        ledger.GenerateID("anchor123456", now),
		CreatedAt: now,
		UpdatedAt: now,
		Workset: ledger.Workset{
			AnchorCommit: "anchor123456",
			Commits:      []string{"anchor123456"},
			Diffstat: &ledger.Diffstat{
				Files:      3,
				Insertions: 45,
				Deletions:  12,
			},
		},
		Summary: ledger.Summary{
			What: "Test with tags",
			Why:  "Testing tags display",
			How:  "Added tags",
		},
		Tags: []string{"feature", "docs"},
		WorkItems: []ledger.WorkItem{
			{System: "jira", ID: "PROJ-123"},
			{System: "github", ID: "456"},
		},
	}
	data, _ := entry.ToJSON()

	jsonFlag = false
	storage := ledger.NewStorage(&mockGitOpsForShow{
		notes: map[string][]byte{
			"anchor123456": data,
		},
	})

	cmd := newShowCmdWithStorage(storage)
	if err := cmd.Flags().Set("latest", "true"); err != nil {
		t.Fatalf("failed to set last flag: %v", err)
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Check tags are displayed
	if !strings.Contains(output, "Tags: feature, docs") {
		t.Errorf("output missing tags\noutput: %s", output)
	}

	// Check work items are displayed
	if !strings.Contains(output, "jira:PROJ-123") {
		t.Errorf("output missing work item jira:PROJ-123\noutput: %s", output)
	}
	if !strings.Contains(output, "github:456") {
		t.Errorf("output missing work item github:456\noutput: %s", output)
	}

	// Check diffstat is displayed
	if !strings.Contains(output, "3 files") {
		t.Errorf("output missing diffstat files\noutput: %s", output)
	}
	if !strings.Contains(output, "+45/-12") {
		t.Errorf("output missing diffstat lines\noutput: %s", output)
	}
}

// createShowTestEntry creates a minimal valid entry for testing show command.
func createShowTestEntry(anchor string, created time.Time) []byte {
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

// newShowCmdWithStorage is a helper for tests that injects a storage.
func newShowCmdWithStorage(storage *ledger.Storage) *cobra.Command {
	return newShowCmdInternal(storage)
}
