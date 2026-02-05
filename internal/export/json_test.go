package export

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// testEntry creates a fully populated entry for testing.
func testEntry() *ledger.Entry {
	return &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        "tb_2026-01-15T15:04:05Z_8f2c1a",
		CreatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		Workset: ledger.Workset{
			AnchorCommit: "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f",
			Commits:      []string{"8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f", "abc123def456"},
			Range:        "abc123..8f2c1a",
			Diffstat: &ledger.Diffstat{
				Files:      3,
				Insertions: 45,
				Deletions:  12,
			},
		},
		Summary: ledger.Summary{
			What: "Fixed authentication bypass vulnerability",
			Why:  "User input wasn't being sanitized before JWT validation",
			How:  "Added input validation middleware before auth handler",
		},
		Tags: []string{"security", "auth"},
		WorkItems: []ledger.WorkItem{
			{System: "beads", ID: "bd-a1b2c3"},
		},
	}
}

// minimalEntry creates an entry with only required fields.
func minimalEntry() *ledger.Entry {
	return &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        "tb_2026-01-15T15:04:05Z_abc123",
		CreatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		Workset: ledger.Workset{
			AnchorCommit: "abc123def456",
			Commits:      []string{"abc123def456"},
		},
		Summary: ledger.Summary{
			What: "Simple change",
			Why:  "Needed it",
			How:  "Did it",
		},
	}
}

// specialCharsEntry creates an entry with special characters in summary fields.
func specialCharsEntry() *ledger.Entry {
	return &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        "tb_2026-01-15T15:04:05Z_special",
		CreatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		Workset: ledger.Workset{
			AnchorCommit: "special123",
			Commits:      []string{"special123"},
		},
		Summary: ledger.Summary{
			What: `Fixed "quotes" and <angle> brackets & ampersands`,
			Why:  "Contains\nnewlines\tand\ttabs",
			How:  `Used unicode: 日本語 emoji: 🎉`,
		},
	}
}

func TestFormatJSON(t *testing.T) {
	tests := []struct {
		name       string
		entries    []*ledger.Entry
		wantFields []string
	}{
		{
			name:    "single entry with all fields",
			entries: []*ledger.Entry{testEntry()},
			wantFields: []string{
				`"schema": "timbers.devlog/v1"`,
				`"id": "tb_2026-01-15T15:04:05Z_8f2c1a"`,
				`"anchor_commit": "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f"`,
				`"what": "Fixed authentication bypass vulnerability"`,
				`"why": "User input wasn't being sanitized before JWT validation"`,
				`"how": "Added input validation middleware before auth handler"`,
				`"security"`,
				`"auth"`,
			},
		},
		{
			name:    "minimal entry",
			entries: []*ledger.Entry{minimalEntry()},
			wantFields: []string{
				`"schema": "timbers.devlog/v1"`,
				`"id": "tb_2026-01-15T15:04:05Z_abc123"`,
				`"what": "Simple change"`,
			},
		},
		{
			name:    "multiple entries",
			entries: []*ledger.Entry{testEntry(), minimalEntry()},
			wantFields: []string{
				`"id": "tb_2026-01-15T15:04:05Z_8f2c1a"`,
				`"id": "tb_2026-01-15T15:04:05Z_abc123"`,
			},
		},
		{
			name:       "empty entry list",
			entries:    []*ledger.Entry{},
			wantFields: []string{"[]"},
		},
		{
			name:    "entry with special characters",
			entries: []*ledger.Entry{specialCharsEntry()},
			wantFields: []string{
				`Fixed \"quotes\"`,
				`\u003cangle\u003e`,
				`\u0026 ampersands`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printer := output.NewPrinter(&buf, true, false)

			err := FormatJSON(printer, tt.entries)
			if err != nil {
				t.Fatalf("FormatJSON() error = %v", err)
			}

			result := buf.String()
			for _, field := range tt.wantFields {
				if !containsString(result, field) {
					t.Errorf("FormatJSON() output missing expected field: %s\nGot: %s", field, result)
				}
			}

			// Verify output is valid JSON
			var parsed any
			if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
				t.Errorf("FormatJSON() output is not valid JSON: %v", err)
			}
		})
	}
}

func TestFormatJSON_NilEntries(t *testing.T) {
	var buf bytes.Buffer
	printer := output.NewPrinter(&buf, true, false)

	err := FormatJSON(printer, nil)
	if err != nil {
		t.Fatalf("FormatJSON() error = %v", err)
	}

	// Should output null
	result := buf.String()
	if result != "null\n" {
		t.Errorf("FormatJSON(nil) = %q, want \"null\\n\"", result)
	}
}

func TestWriteJSONFiles(t *testing.T) {
	tests := []struct {
		name    string
		entries []*ledger.Entry
	}{
		{
			name:    "single entry",
			entries: []*ledger.Entry{testEntry()},
		},
		{
			name:    "multiple entries",
			entries: []*ledger.Entry{testEntry(), minimalEntry()},
		},
		{
			name:    "entry with special characters",
			entries: []*ledger.Entry{specialCharsEntry()},
		},
		{
			name:    "empty entry list",
			entries: []*ledger.Entry{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			err := WriteJSONFiles(tt.entries, tmpDir)
			if err != nil {
				t.Fatalf("WriteJSONFiles() error = %v", err)
			}

			// Verify files were created
			for _, entry := range tt.entries {
				filename := filepath.Join(tmpDir, entry.ID+".json")

				// Check file exists
				data, err := os.ReadFile(filename)
				if err != nil {
					t.Errorf("failed to read file %s: %v", filename, err)
					continue
				}

				// Verify valid JSON
				var parsed ledger.Entry
				if err := json.Unmarshal(data, &parsed); err != nil {
					t.Errorf("file %s is not valid JSON: %v", filename, err)
					continue
				}

				// Verify content matches
				if parsed.ID != entry.ID {
					t.Errorf("file %s has wrong ID: got %q, want %q", filename, parsed.ID, entry.ID)
				}
				if parsed.Summary.What != entry.Summary.What {
					t.Errorf("file %s has wrong What: got %q, want %q", filename, parsed.Summary.What, entry.Summary.What)
				}
			}

			// Verify no extra files for empty list
			if len(tt.entries) == 0 {
				files, err := os.ReadDir(tmpDir)
				if err != nil {
					t.Fatalf("failed to read temp dir: %v", err)
				}
				if len(files) != 0 {
					t.Errorf("expected no files for empty entry list, got %d", len(files))
				}
			}
		})
	}
}

func TestWriteJSONFiles_InvalidDirectory(t *testing.T) {
	entries := []*ledger.Entry{testEntry()}

	// Try to write to a non-existent directory
	err := WriteJSONFiles(entries, "/nonexistent/directory/path")
	if err == nil {
		t.Error("WriteJSONFiles() expected error for invalid directory")
	}
}

func TestWriteJSONFiles_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	entries := []*ledger.Entry{testEntry()}

	err := WriteJSONFiles(entries, tmpDir)
	if err != nil {
		t.Fatalf("WriteJSONFiles() error = %v", err)
	}

	// Check file permissions
	filename := filepath.Join(tmpDir, entries[0].ID+".json")
	info, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	// Should be 0600 (owner read/write only)
	expectedPerm := os.FileMode(0600)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("file permissions = %o, want %o", info.Mode().Perm(), expectedPerm)
	}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
