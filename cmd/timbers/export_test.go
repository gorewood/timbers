// Package main provides the entry point for the timbers CLI.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/ledger"
)

// mockGitOpsForExport implements ledger.GitOps for testing export command.
type mockGitOpsForExport struct {
	notes   map[string][]byte
	commits map[string]git.Commit // Map commit SHA to commit
}

func (m *mockGitOpsForExport) ReadNote(commit string) ([]byte, error) {
	if data, ok := m.notes[commit]; ok {
		return data, nil
	}
	return nil, nil
}

func (m *mockGitOpsForExport) WriteNote(string, string, bool) error {
	return nil
}

func (m *mockGitOpsForExport) ListNotedCommits() ([]string, error) {
	commits := make([]string, 0, len(m.notes))
	for commit := range m.notes {
		commits = append(commits, commit)
	}
	return commits, nil
}

func (m *mockGitOpsForExport) HEAD() (string, error) {
	return "head123", nil
}

func (m *mockGitOpsForExport) Log(fromRef, toRef string) ([]git.Commit, error) {
	// Simple implementation: return commits in the range
	// For test purposes, we'll match based on being in the commits map
	result := make([]git.Commit, 0, len(m.commits))
	for _, commit := range m.commits {
		result = append(result, commit)
	}
	return result, nil
}

func (m *mockGitOpsForExport) CommitsReachableFrom(sha string) ([]git.Commit, error) {
	return nil, nil
}

func (m *mockGitOpsForExport) GetDiffstat(fromRef, toRef string) (git.Diffstat, error) {
	return git.Diffstat{}, nil
}

func (m *mockGitOpsForExport) PushNotes(remote string) error {
	return nil
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
		commits      map[string]git.Commit
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
			wantContains: []string{"specify --last N or --range A..B"},
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
			// Reset global flag
			jsonFlag = tt.jsonOutput

			// Create storage with mock
			storage := ledger.NewStorage(&mockGitOpsForExport{
				notes:   tt.notes,
				commits: tt.commits,
			})

			// Create command
			cmd := newExportCmdInternal(storage)

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

	// Create storage with mock
	storage := ledger.NewStorage(&mockGitOpsForExport{
		notes:   notes,
		commits: map[string]git.Commit{},
	})

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

	// Create storage with mock
	storage := ledger.NewStorage(&mockGitOpsForExport{
		notes:   notes,
		commits: map[string]git.Commit{},
	})

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

// createExportTestEntry creates a minimal valid entry for testing export command.
func createExportTestEntry(anchor, what string, created time.Time) []byte {
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
		Tags: []string{"test"},
	}
	data, _ := entry.ToJSON()
	return data
}
