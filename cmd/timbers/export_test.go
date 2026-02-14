// Package main provides the entry point for the timbers CLI.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// mockGitOpsForExport implements ledger.GitOps for testing export command.
type mockGitOpsForExport struct{}

func (m *mockGitOpsForExport) HEAD() (string, error) {
	return "head123", nil
}

func (m *mockGitOpsForExport) Log(_, _ string) ([]git.Commit, error) {
	return nil, nil
}

func (m *mockGitOpsForExport) CommitsReachableFrom(_ string) ([]git.Commit, error) {
	return nil, nil
}

func (m *mockGitOpsForExport) GetDiffstat(_, _ string) (git.Diffstat, error) {
	return git.Diffstat{}, nil
}

func (m *mockGitOpsForExport) CommitFiles(sha string) ([]string, error) { return nil, nil }

// writeExportEntryFile writes an entry JSON file to the correct date subdirectory.
func writeExportEntryFile(t *testing.T, dir string, data []byte) {
	t.Helper()
	entry, err := ledger.FromJSON(data)
	if err != nil {
		t.Fatalf("failed to parse entry: %v", err)
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

// newExportTestStorage creates a storage from notes data using file-backed entries.
func newExportTestStorage(t *testing.T, notes map[string][]byte) *ledger.Storage {
	t.Helper()
	dir := t.TempDir()
	for _, data := range notes {
		writeExportEntryFile(t, dir, data)
	}
	files := ledger.NewFileStorage(dir, func(_ string) error { return nil }, func(_, _ string) error { return nil })
	return ledger.NewStorage(&mockGitOpsForExport{}, files)
}

// TestExportCommand tests the export command with various inputs.
func TestExportCommand(t *testing.T) {
	now := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)

	tests := []struct {
		name         string
		lastFlag     string
		rangeFlag    string
		formatFlag   string
		outFlag      string
		jsonOutput   bool
		notes        map[string][]byte
		wantErr      bool
		wantContains []string
	}{
		{
			name:         "no --last or --range flag",
			lastFlag:     "",
			rangeFlag:    "",
			formatFlag:   "",
			outFlag:      "",
			notes:        map[string][]byte{},
			wantErr:      true,
			wantContains: []string{"specify --last N, --since", "--until"},
		},
		{
			name:         "--last with zero",
			lastFlag:     "0",
			rangeFlag:    "",
			formatFlag:   "",
			outFlag:      "",
			notes:        map[string][]byte{},
			wantErr:      true,
			wantContains: []string{"--last must be a positive integer"},
		},
		{
			name:         "--last with negative",
			lastFlag:     "-5",
			rangeFlag:    "",
			formatFlag:   "",
			outFlag:      "",
			notes:        map[string][]byte{},
			wantErr:      true,
			wantContains: []string{"--last must be a positive integer"},
		},
		{
			name:         "--last with non-integer",
			lastFlag:     "abc",
			rangeFlag:    "",
			formatFlag:   "",
			outFlag:      "",
			notes:        map[string][]byte{},
			wantErr:      true,
			wantContains: []string{"--last must be a positive integer"},
		},
		{
			name:       "empty entries with --last",
			lastFlag:   "5",
			rangeFlag:  "",
			formatFlag: "json",
			outFlag:    "",
			jsonOutput: true,
			notes:      map[string][]byte{},
			wantErr:    false,
		},
		{
			name:       "--last 1 with JSON format",
			lastFlag:   "1",
			rangeFlag:  "",
			formatFlag: "json",
			outFlag:    "",
			jsonOutput: true,
			notes: map[string][]byte{
				"anchor1": createExportTestEntry("anchor1", "first", now.Add(-1*time.Hour)),
				"anchor2": createExportTestEntry("anchor2", "second", now),
			},
			wantErr:      false,
			wantContains: []string{"second"},
		},
		{
			name:       "--last 2 with JSON format",
			lastFlag:   "2",
			rangeFlag:  "",
			formatFlag: "json",
			outFlag:    "",
			jsonOutput: true,
			notes: map[string][]byte{
				"anchor1": createExportTestEntry("anchor1", "first", now.Add(-2*time.Hour)),
				"anchor2": createExportTestEntry("anchor2", "second", now.Add(-1*time.Hour)),
				"anchor3": createExportTestEntry("anchor3", "third", now),
			},
			wantErr:      false,
			wantContains: []string{"second", "third"},
		},
		{
			name:         "invalid format",
			lastFlag:     "1",
			rangeFlag:    "",
			formatFlag:   "xml",
			outFlag:      "",
			notes:        map[string][]byte{},
			wantErr:      true,
			wantContains: []string{"--format must be"},
		},
		{
			name:         "invalid range format",
			lastFlag:     "",
			rangeFlag:    "invalid",
			formatFlag:   "",
			outFlag:      "",
			notes:        map[string][]byte{},
			wantErr:      true,
			wantContains: []string{"--range must be in format A..B"},
		},
		{
			name:         "--range with empty parts",
			lastFlag:     "",
			rangeFlag:    "..end",
			formatFlag:   "",
			outFlag:      "",
			notes:        map[string][]byte{},
			wantErr:      true,
			wantContains: []string{"--range must be in format A..B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create storage with file-backed entries
			storage := newExportTestStorage(t, tt.notes)

			// Create command
			cmd := newExportCmdInternal(storage)

			// Set JSON mode for testing
			if tt.jsonOutput {
				cmd.PersistentFlags().Bool("json", false, "")
				_ = cmd.PersistentFlags().Set("json", "true")
			}

			// Set flags
			if tt.lastFlag != "" {
				if err := cmd.Flags().Set("last", tt.lastFlag); err != nil {
					t.Fatalf("failed to set last flag: %v", err)
				}
			}
			if tt.rangeFlag != "" {
				if err := cmd.Flags().Set("range", tt.rangeFlag); err != nil {
					t.Fatalf("failed to set range flag: %v", err)
				}
			}
			if tt.formatFlag != "" {
				if err := cmd.Flags().Set("format", tt.formatFlag); err != nil {
					t.Fatalf("failed to set format flag: %v", err)
				}
			}
			if tt.outFlag != "" {
				if err := cmd.Flags().Set("out", tt.outFlag); err != nil {
					t.Fatalf("failed to set out flag: %v", err)
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
		})
	}
}

// TestExportToDirectory tests exporting to a directory.
func TestExportToDirectory(t *testing.T) {
	now := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)

	// Create temp directory
	tmpDir := t.TempDir()

	notes := map[string][]byte{
		"anchor1": createExportTestEntry("anchor1", "first", now.Add(-1*time.Hour)),
		"anchor2": createExportTestEntry("anchor2", "second", now),
	}

	storage := newExportTestStorage(t, notes)

	// Test JSON export to directory
	cmd := newExportCmdInternal(storage)

	if err := cmd.Flags().Set("last", "2"); err != nil {
		t.Fatalf("failed to set last flag: %v", err)
	}
	if err := cmd.Flags().Set("format", "json"); err != nil {
		t.Fatalf("failed to set format flag: %v", err)
	}
	if err := cmd.Flags().Set("out", tmpDir); err != nil {
		t.Fatalf("failed to set out flag: %v", err)
	}

	// Execute
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Check that files were created
	entries := make([]*ledger.Entry, 0, len(notes))
	for _, noteData := range notes {
		entry, err := ledger.FromJSON(noteData)
		if err != nil {
			t.Fatalf("failed to parse entry: %v", err)
		}
		entries = append(entries, entry)
	}

	for _, entry := range entries {
		filename := filepath.Join(tmpDir, entry.ID+".json")
		if _, err := os.Stat(filename); err != nil {
			t.Errorf("expected file %s to exist, got error: %v", filename, err)
		}

		// Verify file contents
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		var readEntry ledger.Entry
		if err := json.Unmarshal(data, &readEntry); err != nil {
			t.Fatalf("failed to unmarshal entry: %v", err)
		}

		if readEntry.ID != entry.ID {
			t.Errorf("entry ID mismatch: got %s, want %s", readEntry.ID, entry.ID)
		}
	}
}

// TestExportMarkdownToDirectory tests exporting to markdown.
func TestExportMarkdownToDirectory(t *testing.T) {
	now := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)

	// Create temp directory
	tmpDir := t.TempDir()

	notes := map[string][]byte{
		"anchor1": createExportTestEntry("anchor1", "first", now),
	}

	storage := newExportTestStorage(t, notes)

	// Test markdown export to directory
	cmd := newExportCmdInternal(storage)

	if err := cmd.Flags().Set("last", "1"); err != nil {
		t.Fatalf("failed to set last flag: %v", err)
	}
	if err := cmd.Flags().Set("format", "md"); err != nil {
		t.Fatalf("failed to set format flag: %v", err)
	}
	if err := cmd.Flags().Set("out", tmpDir); err != nil {
		t.Fatalf("failed to set out flag: %v", err)
	}

	// Execute
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Check that markdown file was created
	entries := make([]*ledger.Entry, 0, len(notes))
	for _, noteData := range notes {
		entry, err := ledger.FromJSON(noteData)
		if err != nil {
			t.Fatalf("failed to parse entry: %v", err)
		}
		entries = append(entries, entry)
	}

	for _, entry := range entries {
		filename := filepath.Join(tmpDir, entry.ID+".md")
		if _, err := os.Stat(filename); err != nil {
			t.Errorf("expected file %s to exist, got error: %v", filename, err)
		}

		// Verify file contains expected content
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		content := string(data)

		// Check for YAML frontmatter
		if !strings.Contains(content, "schema: timbers.export/v1") {
			t.Errorf("missing schema in frontmatter")
		}

		// Check for entry ID
		if !strings.Contains(content, "id: "+entry.ID) {
			t.Errorf("missing ID in frontmatter")
		}

		// Check for What section
		if !strings.Contains(content, "**What:** first") {
			t.Errorf("missing What section")
		}

		// Check for Evidence section
		if !strings.Contains(content, "## Evidence") {
			t.Errorf("missing Evidence section")
		}
	}
}

// TestExportWithTagFiltering tests export command with --tag flag.
func TestExportWithTagFiltering(t *testing.T) {
	now := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)

	// Create entries with different tags
	entry1 := createExportTestEntryWithTags("anchor1", "security fix", now.Add(-3*time.Hour), []string{"security", "bugfix"})
	entry2 := createExportTestEntryWithTags("anchor2", "new feature", now.Add(-2*time.Hour), []string{"feature"})
	entry3 := createExportTestEntryWithTags("anchor3", "auth improvement", now.Add(-1*time.Hour), []string{"security", "auth"})
	entry4 := createExportTestEntryWithTags("anchor4", "docs update", now, []string{"docs"})

	tests := []struct {
		name         string
		lastFlag     string
		tagFlags     []string
		wantContains []string
		wantExclude  []string
	}{
		{
			name:         "single tag filter",
			lastFlag:     "10",
			tagFlags:     []string{"security"},
			wantContains: []string{"security fix", "auth improvement"},
			wantExclude:  []string{"new feature", "docs update"},
		},
		{
			name:         "multiple tags (OR logic)",
			lastFlag:     "10",
			tagFlags:     []string{"security", "docs"},
			wantContains: []string{"security fix", "auth improvement", "docs update"},
			wantExclude:  []string{"new feature"},
		},
		{
			name:         "tag with --last limit",
			lastFlag:     "1",
			tagFlags:     []string{"security"},
			wantContains: []string{"auth improvement"},
			wantExclude:  []string{"security fix", "new feature", "docs update"},
		},
		{
			name:         "no matching tags",
			lastFlag:     "10",
			tagFlags:     []string{"nonexistent"},
			wantContains: []string{},
			wantExclude:  []string{"security fix", "new feature", "auth improvement", "docs update"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes := map[string][]byte{
				"anchor1": entry1,
				"anchor2": entry2,
				"anchor3": entry3,
				"anchor4": entry4,
			}

			storage := newExportTestStorage(t, notes)

			cmd := newExportCmdInternal(storage)
			cmd.PersistentFlags().Bool("json", false, "")
			_ = cmd.PersistentFlags().Set("json", "true")

			if err := cmd.Flags().Set("last", tt.lastFlag); err != nil {
				t.Fatalf("failed to set last flag: %v", err)
			}

			// Set tag flags
			for _, tag := range tt.tagFlags {
				if err := cmd.Flags().Set("tag", tag); err != nil {
					t.Fatalf("failed to set tag flag: %v", err)
				}
			}

			var buf strings.Builder
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			output := buf.String()

			// Check expected content is present
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected content %q\noutput: %s", want, output)
				}
			}

			// Check excluded content is not present
			for _, exclude := range tt.wantExclude {
				if strings.Contains(output, exclude) {
					t.Errorf("output contains excluded content %q\noutput: %s", exclude, output)
				}
			}
		})
	}
}

// TestExportTagFilteringWithTimeRange tests tag filtering combined with time filters.
func TestExportTagFilteringWithTimeRange(t *testing.T) {
	// Create entries with different tags and times
	// Use absolute dates to avoid relative time calculation issues
	oldDate := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	recentDate := time.Date(2026, 1, 14, 12, 0, 0, 0, time.UTC)
	newestDate := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	entry1 := createExportTestEntryWithTags("anchor1", "old security fix", oldDate, []string{"security"})
	entry2 := createExportTestEntryWithTags("anchor2", "recent security fix", recentDate, []string{"security"})
	entry3 := createExportTestEntryWithTags("anchor3", "recent feature", newestDate, []string{"feature"})

	notes := map[string][]byte{
		"anchor1": entry1,
		"anchor2": entry2,
		"anchor3": entry3,
	}

	storage := newExportTestStorage(t, notes)

	cmd := newExportCmdInternal(storage)
	cmd.PersistentFlags().Bool("json", false, "")
	_ = cmd.PersistentFlags().Set("json", "true")

	// Filter by tag and time: should get only entries since 2026-01-10 with security tag
	if err := cmd.Flags().Set("since", "2026-01-10"); err != nil {
		t.Fatalf("failed to set since flag: %v", err)
	}
	if err := cmd.Flags().Set("tag", "security"); err != nil {
		t.Fatalf("failed to set tag flag: %v", err)
	}

	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Should contain recent security fix
	if !strings.Contains(output, "recent security fix") {
		t.Errorf("output missing recent security fix\noutput: %s", output)
	}

	// Should not contain old security fix or recent feature
	if strings.Contains(output, "old security fix") {
		t.Errorf("output contains old security fix (should be filtered by time)\noutput: %s", output)
	}
	if strings.Contains(output, "recent feature") {
		t.Errorf("output contains recent feature (should be filtered by tag)\noutput: %s", output)
	}
}

// createExportTestEntry creates a minimal valid entry for testing export command.
func createExportTestEntry(anchor, what string, created time.Time) []byte {
	return createExportTestEntryWithTags(anchor, what, created, []string{"test"})
}

// createExportTestEntryWithTags creates an entry with custom tags for testing.
func createExportTestEntryWithTags(anchor, what string, created time.Time, tags []string) []byte {
	entry := &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        ledger.GenerateID(anchor, created),
		CreatedAt: created,
		UpdatedAt: created,
		Workset: ledger.Workset{
			AnchorCommit: anchor,
			Commits:      []string{anchor},
			Range:        anchor + ".." + anchor,
			Diffstat: &ledger.Diffstat{
				Files:      1,
				Insertions: 10,
				Deletions:  5,
			},
		},
		Summary: ledger.Summary{
			What: what,
			Why:  "Testing export",
			How:  "Via test",
		},
		Tags: tags,
	}
	data, _ := entry.ToJSON()
	return data
}
