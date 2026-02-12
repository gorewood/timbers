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

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// mockGitOpsForShow implements ledger.GitOps for testing show command.
type mockGitOpsForShow struct{}

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

func (m *mockGitOpsForShow) CommitFiles(sha string) ([]string, error) { return nil, nil }

// writeShowEntryFile writes an entry JSON file to the correct date subdirectory.
func writeShowEntryFile(t *testing.T, dir string, entry *ledger.Entry) {
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

func TestShowCommand(t *testing.T) {
	now := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)
	testEntry := createShowTestEntryStruct("anchor123456", now)
	testEntryID := ledger.GenerateID("anchor123456", now)

	tests := []struct {
		name           string
		entries        []*ledger.Entry // entries to write to temp dir; nil means no entries
		args           []string
		lastFlag       bool
		jsonOutput     bool
		wantErr        bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:         "show by ID - found",
			entries:      []*ledger.Entry{testEntry},
			args:         []string{testEntryID},
			wantContains: []string{testEntryID, "Test entry", "For testing", "Via test", "anchor1"},
		},
		{
			name:         "show by ID - not found",
			entries:      nil,
			args:         []string{"nonexistent-id"},
			wantErr:      true,
			wantContains: []string{"entry not found"},
		},
		{
			name:         "show --latest - found",
			entries:      []*ledger.Entry{testEntry},
			lastFlag:     true,
			wantContains: []string{testEntryID, "Test entry"},
		},
		{
			name:         "show --latest - no entries",
			entries:      nil,
			lastFlag:     true,
			wantErr:      true,
			wantContains: []string{"no entries found"},
		},
		{
			name:         "no ID and no --latest flag",
			entries:      nil,
			wantErr:      true,
			wantContains: []string{"specify an entry ID or use --latest"},
		},
		{
			name:         "both ID and --latest flag",
			entries:      []*ledger.Entry{testEntry},
			args:         []string{testEntryID},
			lastFlag:     true,
			wantErr:      true,
			wantContains: []string{"cannot use both ID argument and --latest flag"},
		},
		{
			name:         "show --json - structured output",
			entries:      []*ledger.Entry{testEntry},
			args:         []string{testEntryID},
			jsonOutput:   true,
			wantContains: []string{`"id"`, `"summary"`, `"what"`, `"why"`, `"how"`},
		},
		{
			name:         "show --latest --json",
			entries:      []*ledger.Entry{testEntry},
			lastFlag:     true,
			jsonOutput:   true,
			wantContains: []string{`"id"`, `"schema"`, `"workset"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create storage with file-backed entries
			var files *ledger.FileStorage
			if tt.entries != nil {
				dir := t.TempDir()
				for _, entry := range tt.entries {
					writeShowEntryFile(t, dir, entry)
				}
				files = ledger.NewFileStorage(dir, func(_ string) error { return nil })
			}
			storage := ledger.NewStorage(&mockGitOpsForShow{}, files)

			// Create command
			cmd := newShowCmdWithStorage(storage)

			// Set JSON mode for testing
			if tt.jsonOutput {
				cmd.PersistentFlags().Bool("json", false, "")
				_ = cmd.PersistentFlags().Set("json", "true")
			}

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

	dir := t.TempDir()
	writeShowEntryFile(t, dir, entry)
	files := ledger.NewFileStorage(dir, func(_ string) error { return nil })
	storage := ledger.NewStorage(&mockGitOpsForShow{}, files)

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

func TestShowWithNotes(t *testing.T) {
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
		},
		Summary: ledger.Summary{
			What: "Test with notes",
			Why:  "Testing notes display",
			How:  "Added notes field",
		},
		Notes: "Debated putting notes in summary vs top-level. Top-level keeps summary focused.",
	}

	dir := t.TempDir()
	writeShowEntryFile(t, dir, entry)
	files := ledger.NewFileStorage(dir, func(_ string) error { return nil })
	storage := ledger.NewStorage(&mockGitOpsForShow{}, files)

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
	if !strings.Contains(output, "Notes") {
		t.Errorf("output missing Notes section header\noutput: %s", output)
	}
	if !strings.Contains(output, "Debated putting notes") {
		t.Errorf("output missing notes content\noutput: %s", output)
	}
}

func TestShowWithoutNotes(t *testing.T) {
	now := time.Now().UTC()
	entry := createShowTestEntryStruct("anchor123456", now)

	dir := t.TempDir()
	writeShowEntryFile(t, dir, entry)
	files := ledger.NewFileStorage(dir, func(_ string) error { return nil })
	storage := ledger.NewStorage(&mockGitOpsForShow{}, files)

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
	if strings.Contains(output, "Notes") {
		t.Errorf("output should not contain Notes section when notes is empty\noutput: %s", output)
	}
}

func TestAnchorDisplay(t *testing.T) {
	origSHAExists := shaExistsFunc
	t.Cleanup(func() { shaExistsFunc = origSHAExists })

	tests := []struct {
		name      string
		sha       string
		exists    bool
		want      string
		wantAnnot bool
	}{
		{
			name:   "existing SHA - no annotation",
			sha:    "abc123def456789",
			exists: true,
			want:   "abc123d",
		},
		{
			name:      "missing SHA - annotated",
			sha:       "abc123def456789",
			exists:    false,
			want:      "abc123d (not in current history)",
			wantAnnot: true,
		},
		{
			name:      "empty SHA - annotated",
			sha:       "",
			exists:    false,
			want:      "",
			wantAnnot: false, // empty SHA returns empty string, no annotation
		},
		{
			name:      "short SHA - annotated when missing",
			sha:       "abc12",
			exists:    false,
			want:      "abc12 (not in current history)",
			wantAnnot: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shaExistsFunc = func(_ string) bool { return tt.exists }
			got := anchorDisplay(tt.sha)
			if got != tt.want {
				t.Errorf("anchorDisplay(%q) = %q, want %q", tt.sha, got, tt.want)
			}
		})
	}
}

func TestShowAnchorAnnotation(t *testing.T) {
	// Save and restore the original shaExistsFunc.
	origSHAExists := shaExistsFunc
	t.Cleanup(func() { shaExistsFunc = origSHAExists })

	now := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)

	t.Run("stale anchor shows annotation", func(t *testing.T) {
		shaExistsFunc = func(_ string) bool { return false }

		entry := createShowTestEntryStruct("staleanchor12345", now)
		dir := t.TempDir()
		writeShowEntryFile(t, dir, entry)
		files := ledger.NewFileStorage(dir, func(_ string) error { return nil })
		storage := ledger.NewStorage(&mockGitOpsForShow{}, files)

		cmd := newShowCmdWithStorage(storage)
		_ = cmd.Flags().Set("latest", "true")

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "not in current history") {
			t.Errorf("expected stale anchor annotation\noutput: %s", output)
		}
	})

	t.Run("valid anchor shows no annotation", func(t *testing.T) {
		shaExistsFunc = func(_ string) bool { return true }

		entry := createShowTestEntryStruct("validanchor12345", now)
		dir := t.TempDir()
		writeShowEntryFile(t, dir, entry)
		files := ledger.NewFileStorage(dir, func(_ string) error { return nil })
		storage := ledger.NewStorage(&mockGitOpsForShow{}, files)

		cmd := newShowCmdWithStorage(storage)
		_ = cmd.Flags().Set("latest", "true")

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		output := buf.String()
		if strings.Contains(output, "not in current history") {
			t.Errorf("expected no stale anchor annotation for valid SHA\noutput: %s", output)
		}
		if !strings.Contains(output, "validan") {
			t.Errorf("expected short SHA in output\noutput: %s", output)
		}
	})
}

// createShowTestEntryStruct creates a minimal valid entry struct for testing show command.
func createShowTestEntryStruct(anchor string, created time.Time) *ledger.Entry {
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

// newShowCmdWithStorage is a helper for tests that injects a storage.
func newShowCmdWithStorage(storage *ledger.Storage) *cobra.Command {
	return newShowCmdInternal(storage)
}
