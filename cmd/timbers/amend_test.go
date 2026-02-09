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
	"github.com/gorewood/timbers/internal/output"
)

// mockGitOpsForAmend implements ledger.GitOps for testing amend command.
type mockGitOpsForAmend struct {
	notes        map[string]*ledger.Entry
	writtenEntry *ledger.Entry
	writeForce   bool
	writeErr     error
}

func newMockGitOpsForAmend() *mockGitOpsForAmend {
	return &mockGitOpsForAmend{
		notes: make(map[string]*ledger.Entry),
	}
}

func (m *mockGitOpsForAmend) ReadNote(commit string) ([]byte, error) {
	if entry, ok := m.notes[commit]; ok {
		return entry.ToJSON()
	}
	return nil, output.NewUserError("note not found for commit: " + commit)
}

func (m *mockGitOpsForAmend) WriteNote(commit string, content string, force bool) error {
	if m.writeErr != nil {
		return m.writeErr
	}

	// Parse the JSON to track what was written
	var entry ledger.Entry
	if err := json.Unmarshal([]byte(content), &entry); err != nil {
		return output.NewSystemError("failed to unmarshal entry: " + err.Error())
	}

	m.writtenEntry = &entry
	m.writeForce = force
	m.notes[commit] = &entry
	return nil
}

func (m *mockGitOpsForAmend) ListNotedCommits() ([]string, error) {
	commits := make([]string, 0, len(m.notes))
	for commit := range m.notes {
		commits = append(commits, commit)
	}
	return commits, nil
}

func (m *mockGitOpsForAmend) HEAD() (string, error) {
	return "abc123", nil
}

func (m *mockGitOpsForAmend) Log(_, _ string) ([]git.Commit, error) {
	return nil, nil
}

func (m *mockGitOpsForAmend) CommitsReachableFrom(_ string) ([]git.Commit, error) {
	return nil, nil
}

func (m *mockGitOpsForAmend) GetDiffstat(_, _ string) (git.Diffstat, error) {
	return git.Diffstat{}, nil
}

func (m *mockGitOpsForAmend) PushNotes(_ string) error {
	return nil
}

func TestAmendCommand(t *testing.T) {
	baseTime := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)
	anchorSHA := "abc123def456"

	tests := []struct {
		name         string
		setupEntry   *ledger.Entry
		args         []string
		wantErr      bool
		wantContains []string
		checkResult  func(t *testing.T, mock *mockGitOpsForAmend)
	}{
		{
			name: "amend what field",
			setupEntry: &ledger.Entry{
				Schema:    ledger.SchemaVersion,
				Kind:      ledger.KindEntry,
				ID:        ledger.GenerateID(anchorSHA, baseTime),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				Workset: ledger.Workset{
					AnchorCommit: anchorSHA,
					Commits:      []string{anchorSHA},
				},
				Summary: ledger.Summary{
					What: "Original what",
					Why:  "Original why",
					How:  "Original how",
				},
			},
			args:         []string{"tb_2026-01-15T15:04:05Z_abc123", "--what", "Updated what"},
			wantContains: []string{"Entry amended successfully", "tb_2026-01-15T15:04:05Z_abc123"},
			checkResult: func(t *testing.T, mock *mockGitOpsForAmend) {
				if mock.writtenEntry == nil {
					t.Fatal("expected entry to be written")
				}
				if mock.writtenEntry.Summary.What != "Updated what" {
					t.Errorf("expected what='Updated what', got %q", mock.writtenEntry.Summary.What)
				}
				if mock.writtenEntry.Summary.Why != "Original why" {
					t.Errorf("expected why='Original why', got %q", mock.writtenEntry.Summary.Why)
				}
				if mock.writtenEntry.Summary.How != "Original how" {
					t.Errorf("expected how='Original how', got %q", mock.writtenEntry.Summary.How)
				}
				if !mock.writeForce {
					t.Error("expected force=true when writing")
				}
				if !mock.writtenEntry.UpdatedAt.After(baseTime) {
					t.Error("expected updated_at to be after original time")
				}
			},
		},
		{
			name: "amend why field",
			setupEntry: &ledger.Entry{
				Schema:    ledger.SchemaVersion,
				Kind:      ledger.KindEntry,
				ID:        ledger.GenerateID(anchorSHA, baseTime),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				Workset: ledger.Workset{
					AnchorCommit: anchorSHA,
					Commits:      []string{anchorSHA},
				},
				Summary: ledger.Summary{
					What: "Original what",
					Why:  "Original why",
					How:  "Original how",
				},
			},
			args:         []string{"tb_2026-01-15T15:04:05Z_abc123", "--why", "Updated why"},
			wantContains: []string{"Entry amended successfully"},
			checkResult: func(t *testing.T, mock *mockGitOpsForAmend) {
				if mock.writtenEntry.Summary.Why != "Updated why" {
					t.Errorf("expected why='Updated why', got %q", mock.writtenEntry.Summary.Why)
				}
				if mock.writtenEntry.Summary.What != "Original what" {
					t.Errorf("expected what unchanged, got %q", mock.writtenEntry.Summary.What)
				}
			},
		},
		{
			name: "amend how field",
			setupEntry: &ledger.Entry{
				Schema:    ledger.SchemaVersion,
				Kind:      ledger.KindEntry,
				ID:        ledger.GenerateID(anchorSHA, baseTime),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				Workset: ledger.Workset{
					AnchorCommit: anchorSHA,
					Commits:      []string{anchorSHA},
				},
				Summary: ledger.Summary{
					What: "Original what",
					Why:  "Original why",
					How:  "Original how",
				},
			},
			args:         []string{"tb_2026-01-15T15:04:05Z_abc123", "--how", "Updated how"},
			wantContains: []string{"Entry amended successfully"},
			checkResult: func(t *testing.T, mock *mockGitOpsForAmend) {
				if mock.writtenEntry.Summary.How != "Updated how" {
					t.Errorf("expected how='Updated how', got %q", mock.writtenEntry.Summary.How)
				}
			},
		},
		{
			name: "amend multiple fields",
			setupEntry: &ledger.Entry{
				Schema:    ledger.SchemaVersion,
				Kind:      ledger.KindEntry,
				ID:        ledger.GenerateID(anchorSHA, baseTime),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				Workset: ledger.Workset{
					AnchorCommit: anchorSHA,
					Commits:      []string{anchorSHA},
				},
				Summary: ledger.Summary{
					What: "Original what",
					Why:  "Original why",
					How:  "Original how",
				},
			},
			args:         []string{"tb_2026-01-15T15:04:05Z_abc123", "--what", "New what", "--why", "New why"},
			wantContains: []string{"Entry amended successfully"},
			checkResult: func(t *testing.T, mock *mockGitOpsForAmend) {
				if mock.writtenEntry.Summary.What != "New what" {
					t.Errorf("expected what='New what', got %q", mock.writtenEntry.Summary.What)
				}
				if mock.writtenEntry.Summary.Why != "New why" {
					t.Errorf("expected why='New why', got %q", mock.writtenEntry.Summary.Why)
				}
				if mock.writtenEntry.Summary.How != "Original how" {
					t.Errorf("expected how unchanged, got %q", mock.writtenEntry.Summary.How)
				}
			},
		},
		{
			name: "amend tags",
			setupEntry: &ledger.Entry{
				Schema:    ledger.SchemaVersion,
				Kind:      ledger.KindEntry,
				ID:        ledger.GenerateID(anchorSHA, baseTime),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				Workset: ledger.Workset{
					AnchorCommit: anchorSHA,
					Commits:      []string{anchorSHA},
				},
				Summary: ledger.Summary{
					What: "Original what",
					Why:  "Original why",
					How:  "Original how",
				},
				Tags: []string{"old-tag"},
			},
			args:         []string{"tb_2026-01-15T15:04:05Z_abc123", "--tag", "new-tag", "--tag", "another-tag"},
			wantContains: []string{"Entry amended successfully"},
			checkResult: func(t *testing.T, mock *mockGitOpsForAmend) {
				if len(mock.writtenEntry.Tags) != 2 {
					t.Errorf("expected 2 tags, got %d", len(mock.writtenEntry.Tags))
				}
				if mock.writtenEntry.Tags[0] != "new-tag" || mock.writtenEntry.Tags[1] != "another-tag" {
					t.Errorf("expected tags [new-tag, another-tag], got %v", mock.writtenEntry.Tags)
				}
			},
		},
		{
			name: "replace with single tag",
			setupEntry: &ledger.Entry{
				Schema:    ledger.SchemaVersion,
				Kind:      ledger.KindEntry,
				ID:        ledger.GenerateID(anchorSHA, baseTime),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				Workset: ledger.Workset{
					AnchorCommit: anchorSHA,
					Commits:      []string{anchorSHA},
				},
				Summary: ledger.Summary{
					What: "Original what",
					Why:  "Original why",
					How:  "Original how",
				},
				Tags: []string{"tag1", "tag2"},
			},
			args:         []string{"tb_2026-01-15T15:04:05Z_abc123", "--tag", "new-tag"},
			wantContains: []string{"Entry amended successfully"},
			checkResult: func(t *testing.T, mock *mockGitOpsForAmend) {
				if len(mock.writtenEntry.Tags) != 1 || mock.writtenEntry.Tags[0] != "new-tag" {
					t.Errorf("expected tags ['new-tag'], got %v", mock.writtenEntry.Tags)
				}
			},
		},
		{
			name: "dry run",
			setupEntry: &ledger.Entry{
				Schema:    ledger.SchemaVersion,
				Kind:      ledger.KindEntry,
				ID:        ledger.GenerateID(anchorSHA, baseTime),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				Workset: ledger.Workset{
					AnchorCommit: anchorSHA,
					Commits:      []string{anchorSHA},
				},
				Summary: ledger.Summary{
					What: "Original what",
					Why:  "Original why",
					How:  "Original how",
				},
			},
			args:         []string{"tb_2026-01-15T15:04:05Z_abc123", "--what", "Updated what", "--dry-run"},
			wantContains: []string{"Dry run", "Before:", "After:", "Original what", "Updated what"},
			checkResult: func(t *testing.T, mock *mockGitOpsForAmend) {
				if mock.writtenEntry != nil {
					t.Error("expected no write in dry-run mode")
				}
			},
		},
		{
			name: "error: no fields specified",
			setupEntry: &ledger.Entry{
				Schema:    ledger.SchemaVersion,
				Kind:      ledger.KindEntry,
				ID:        ledger.GenerateID(anchorSHA, baseTime),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				Workset: ledger.Workset{
					AnchorCommit: anchorSHA,
					Commits:      []string{anchorSHA},
				},
				Summary: ledger.Summary{
					What: "Original what",
					Why:  "Original why",
					How:  "Original how",
				},
			},
			args:         []string{"tb_2026-01-15T15:04:05Z_abc123"},
			wantErr:      true,
			wantContains: []string{"at least one field must be specified"},
			checkResult: func(t *testing.T, mock *mockGitOpsForAmend) {
				if mock.writtenEntry != nil {
					t.Error("expected no write when no fields specified")
				}
			},
		},
		{
			name:         "error: entry not found",
			setupEntry:   nil, // No entry in storage
			args:         []string{"tb_2026-01-15T15:04:05Z_nonexistent", "--what", "New what"},
			wantErr:      true,
			wantContains: []string{"entry not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOpsForAmend()

			// Setup entry in mock storage if provided
			if tt.setupEntry != nil {
				mock.notes[tt.setupEntry.Workset.AnchorCommit] = tt.setupEntry
			}

			storage := ledger.NewStorage(mock)
			cmd := newAmendCmdInternal(storage)

			// Capture output
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			// Execute command
			err := cmd.Execute()

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got error=%v: %v", tt.wantErr, err != nil, err)
			}

			// Check output contains expected strings
			output := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, output)
				}
			}

			// Run custom check if provided
			if tt.checkResult != nil {
				tt.checkResult(t, mock)
			}
		})
	}
}

func TestAmendCommandJSON(t *testing.T) {
	baseTime := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)
	anchorSHA := "abc123def456"

	tests := []struct {
		name        string
		setupEntry  *ledger.Entry
		args        []string
		wantErr     bool
		checkResult func(t *testing.T, result map[string]any)
	}{
		{
			name: "amend with JSON output",
			setupEntry: &ledger.Entry{
				Schema:    ledger.SchemaVersion,
				Kind:      ledger.KindEntry,
				ID:        ledger.GenerateID(anchorSHA, baseTime),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				Workset: ledger.Workset{
					AnchorCommit: anchorSHA,
					Commits:      []string{anchorSHA},
				},
				Summary: ledger.Summary{
					What: "Original what",
					Why:  "Original why",
					How:  "Original how",
				},
			},
			args: []string{"tb_2026-01-15T15:04:05Z_abc123", "--what", "Updated what"},
			checkResult: func(t *testing.T, result map[string]any) {
				if status, ok := result["status"].(string); !ok || status != "amended" {
					t.Errorf("expected status='amended' in JSON output, got %v", result["status"])
				}
				if id, ok := result["id"].(string); !ok || !strings.Contains(id, "tb_") {
					t.Error("expected id in JSON output")
				}
			},
		},
		{
			name: "dry run with JSON output",
			setupEntry: &ledger.Entry{
				Schema:    ledger.SchemaVersion,
				Kind:      ledger.KindEntry,
				ID:        ledger.GenerateID(anchorSHA, baseTime),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				Workset: ledger.Workset{
					AnchorCommit: anchorSHA,
					Commits:      []string{anchorSHA},
				},
				Summary: ledger.Summary{
					What: "Original what",
					Why:  "Original why",
					How:  "Original how",
				},
			},
			args: []string{"tb_2026-01-15T15:04:05Z_abc123", "--what", "Updated what", "--dry-run"},
			checkResult: func(t *testing.T, result map[string]any) {
				if dryRun, ok := result["dry_run"].(bool); !ok || !dryRun {
					t.Error("expected dry_run=true in JSON output")
				}
				changes, ok := result["changes"].(map[string]any)
				if !ok {
					t.Error("expected changes object in JSON output")
					return
				}
				whatChange, ok := changes["what"].(map[string]any)
				if !ok {
					t.Error("expected what change object")
					return
				}
				if whatChange["before"] != "Original what" {
					t.Errorf("expected before='Original what', got %v", whatChange["before"])
				}
				if whatChange["after"] != "Updated what" {
					t.Errorf("expected after='Updated what', got %v", whatChange["after"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOpsForAmend()

			// Setup entry in mock storage if provided
			if tt.setupEntry != nil {
				mock.notes[tt.setupEntry.Workset.AnchorCommit] = tt.setupEntry
			}

			storage := ledger.NewStorage(mock)
			cmd := newAmendCmdInternal(storage)

			// Set JSON mode for testing
			cmd.PersistentFlags().Bool("json", false, "")
			_ = cmd.PersistentFlags().Set("json", "true")

			// Capture output
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			// Execute command
			err := cmd.Execute()

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got error=%v: %v", tt.wantErr, err != nil, err)
			}

			if !tt.wantErr && tt.checkResult != nil {
				// Parse JSON output
				var result map[string]any
				if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
					t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
				}
				tt.checkResult(t, result)
			}
		})
	}
}
