// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// mockGitOpsForAmend implements ledger.GitOps for testing amend command.
type mockGitOpsForAmend struct{}

func newMockGitOpsForAmend() *mockGitOpsForAmend {
	return &mockGitOpsForAmend{}
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

func (m *mockGitOpsForAmend) CommitFiles(sha string) ([]string, error) { return nil, nil }

// setupAmendTestStorage creates a temp dir, writes the entry file if non-nil,
// and returns the storage and dir path. The gitAdd function is a no-op by default.
func setupAmendTestStorage(t *testing.T, mock *mockGitOpsForAmend, entry *ledger.Entry) (*ledger.Storage, string) {
	t.Helper()
	dir := t.TempDir()
	if entry != nil {
		data, err := entry.ToJSON()
		if err != nil {
			t.Fatalf("failed to serialize setup entry: %v", err)
		}
		entryDir := dir
		if sub := ledger.EntryDateDir(entry.ID); sub != "" {
			entryDir = filepath.Join(dir, sub)
		}
		if err := os.MkdirAll(entryDir, 0o755); err != nil {
			t.Fatalf("failed to create entry dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(entryDir, entry.ID+".json"), data, 0o600); err != nil {
			t.Fatalf("failed to write setup entry file: %v", err)
		}
	}
	files := ledger.NewFileStorage(dir, func(_ string) error { return nil }, func(_, _ string) error { return nil })
	return ledger.NewStorage(mock, files), dir
}

// readEntryFromDir reads and parses the entry file for the given ID from the dir.
func readEntryFromDir(t *testing.T, dir, id string) *ledger.Entry {
	t.Helper()
	entryDir := dir
	if sub := ledger.EntryDateDir(id); sub != "" {
		entryDir = filepath.Join(dir, sub)
	}
	data, err := os.ReadFile(filepath.Join(entryDir, id+".json"))
	if err != nil {
		t.Fatalf("failed to read entry file: %v", err)
	}
	entry, err := ledger.FromJSON(data)
	if err != nil {
		t.Fatalf("failed to parse entry file: %v", err)
	}
	return entry
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
		failAdd      bool // if true, use a failing gitAdd to simulate write errors
		checkResult  func(t *testing.T, dir string, entryID string)
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
			checkResult: func(t *testing.T, dir string, entryID string) {
				entry := readEntryFromDir(t, dir, entryID)
				if entry.Summary.What != "Updated what" {
					t.Errorf("expected what='Updated what', got %q", entry.Summary.What)
				}
				if entry.Summary.Why != "Original why" {
					t.Errorf("expected why='Original why', got %q", entry.Summary.Why)
				}
				if entry.Summary.How != "Original how" {
					t.Errorf("expected how='Original how', got %q", entry.Summary.How)
				}
				if !entry.UpdatedAt.After(baseTime) {
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
			checkResult: func(t *testing.T, dir string, entryID string) {
				entry := readEntryFromDir(t, dir, entryID)
				if entry.Summary.Why != "Updated why" {
					t.Errorf("expected why='Updated why', got %q", entry.Summary.Why)
				}
				if entry.Summary.What != "Original what" {
					t.Errorf("expected what unchanged, got %q", entry.Summary.What)
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
			checkResult: func(t *testing.T, dir string, entryID string) {
				entry := readEntryFromDir(t, dir, entryID)
				if entry.Summary.How != "Updated how" {
					t.Errorf("expected how='Updated how', got %q", entry.Summary.How)
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
			checkResult: func(t *testing.T, dir string, entryID string) {
				entry := readEntryFromDir(t, dir, entryID)
				if entry.Summary.What != "New what" {
					t.Errorf("expected what='New what', got %q", entry.Summary.What)
				}
				if entry.Summary.Why != "New why" {
					t.Errorf("expected why='New why', got %q", entry.Summary.Why)
				}
				if entry.Summary.How != "Original how" {
					t.Errorf("expected how unchanged, got %q", entry.Summary.How)
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
			checkResult: func(t *testing.T, dir string, entryID string) {
				entry := readEntryFromDir(t, dir, entryID)
				if len(entry.Tags) != 2 {
					t.Errorf("expected 2 tags, got %d", len(entry.Tags))
				}
				if entry.Tags[0] != "new-tag" || entry.Tags[1] != "another-tag" {
					t.Errorf("expected tags [new-tag, another-tag], got %v", entry.Tags)
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
			checkResult: func(t *testing.T, dir string, entryID string) {
				entry := readEntryFromDir(t, dir, entryID)
				if len(entry.Tags) != 1 || entry.Tags[0] != "new-tag" {
					t.Errorf("expected tags ['new-tag'], got %v", entry.Tags)
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
			checkResult: func(t *testing.T, dir string, entryID string) {
				// In dry-run mode, the file should still contain the original entry
				entry := readEntryFromDir(t, dir, entryID)
				if entry.Summary.What != "Original what" {
					t.Errorf("expected no write in dry-run mode, but what was changed to %q", entry.Summary.What)
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
			checkResult: func(t *testing.T, dir string, entryID string) {
				// File should still contain the original entry (no write occurred)
				entry := readEntryFromDir(t, dir, entryID)
				if entry.Summary.What != "Original what" {
					t.Errorf("expected no write when no fields specified, but what was changed to %q", entry.Summary.What)
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
		{
			name: "error: write failure",
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
			failAdd:      true,
			wantErr:      true,
			wantContains: []string{"failed to stage entry file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOpsForAmend()

			var storage *ledger.Storage
			var dir string

			if tt.failAdd {
				// Use a failing gitAdd to simulate write errors
				dir = t.TempDir()
				if tt.setupEntry != nil {
					data, err := tt.setupEntry.ToJSON()
					if err != nil {
						t.Fatalf("failed to serialize setup entry: %v", err)
					}
					entryDir := dir
					if sub := ledger.EntryDateDir(tt.setupEntry.ID); sub != "" {
						entryDir = filepath.Join(dir, sub)
					}
					if err := os.MkdirAll(entryDir, 0o755); err != nil {
						t.Fatalf("failed to create entry dir: %v", err)
					}
					if err := os.WriteFile(filepath.Join(entryDir, tt.setupEntry.ID+".json"), data, 0o600); err != nil {
						t.Fatalf("failed to write setup entry file: %v", err)
					}
				}
				failAdd := func(_ string) error { return output.NewSystemError("write failed") }
				files := ledger.NewFileStorage(dir, failAdd, func(_, _ string) error { return nil })
				storage = ledger.NewStorage(mock, files)
			} else {
				storage, dir = setupAmendTestStorage(t, mock, tt.setupEntry)
			}

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
			cmdOutput := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(cmdOutput, want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, cmdOutput)
				}
			}

			// Run custom check if provided
			if tt.checkResult != nil {
				entryID := ""
				if tt.setupEntry != nil {
					entryID = tt.setupEntry.ID
				}
				tt.checkResult(t, dir, entryID)
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
			storage, _ := setupAmendTestStorage(t, mock, tt.setupEntry)

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
