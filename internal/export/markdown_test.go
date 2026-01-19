package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rbergman/timbers/internal/ledger"
)

func TestFormatMarkdown(t *testing.T) {
	tests := []struct {
		name         string
		entry        *ledger.Entry
		wantContains []string
	}{
		{
			name:  "full entry with all fields",
			entry: testEntry(),
			wantContains: []string{
				"---",
				"schema: timbers.export/v1",
				"id: tb_2026-01-15T15:04:05Z_8f2c1a",
				"date: 2026-01-15",
				"anchor_commit: 8f2c1a9d7b0c",
				"commit_count: 2",
				"tags: [security, auth]",
				"---",
				"# Fixed authentication bypass vulnerability",
				"**What:** Fixed authentication bypass vulnerability",
				"**Why:** User input wasn't being sanitized before JWT validation",
				"**How:** Added input validation middleware before auth handler",
				"## Evidence",
				"- Commits: 2 (abc123..8f2c1a)",
				"- Files changed: 3 (+45/-12)",
			},
		},
		{
			name:  "minimal entry",
			entry: minimalEntry(),
			wantContains: []string{
				"---",
				"schema: timbers.export/v1",
				"id: tb_2026-01-15T15:04:05Z_abc123",
				"date: 2026-01-15",
				"anchor_commit: abc123def456",
				"commit_count: 1",
				"---",
				"# Simple change",
				"**What:** Simple change",
				"**Why:** Needed it",
				"**How:** Did it",
				"## Evidence",
				"- Commits: 1",
			},
		},
		{
			name:  "entry with special characters",
			entry: specialCharsEntry(),
			wantContains: []string{
				`**What:** Fixed "quotes" and <angle> brackets & ampersands`,
				"**Why:** Contains\nnewlines\tand\ttabs",
				`**How:** Used unicode: 日本語 emoji: 🎉`,
			},
		},
		{
			name: "entry without tags",
			entry: &ledger.Entry{
				Schema:    ledger.SchemaVersion,
				Kind:      ledger.KindEntry,
				ID:        "tb_2026-01-15T15:04:05Z_notags",
				CreatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
				UpdatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
				Workset: ledger.Workset{
					AnchorCommit: "notags123456",
					Commits:      []string{"notags123456"},
				},
				Summary: ledger.Summary{
					What: "No tags entry",
					Why:  "Testing",
					How:  "Testing",
				},
			},
			wantContains: []string{
				"id: tb_2026-01-15T15:04:05Z_notags",
				"# No tags entry",
			},
		},
		{
			name: "entry without diffstat",
			entry: &ledger.Entry{
				Schema:    ledger.SchemaVersion,
				Kind:      ledger.KindEntry,
				ID:        "tb_2026-01-15T15:04:05Z_nodiff",
				CreatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
				UpdatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
				Workset: ledger.Workset{
					AnchorCommit: "nodiff123456",
					Commits:      []string{"nodiff123456"},
				},
				Summary: ledger.Summary{
					What: "No diffstat entry",
					Why:  "Testing",
					How:  "Testing",
				},
			},
			wantContains: []string{
				"- Commits: 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMarkdown(tt.entry)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatMarkdown() missing expected content: %q\nGot:\n%s", want, result)
				}
			}
		})
	}
}

func TestFormatMarkdown_NoTagsField(t *testing.T) {
	entry := &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        "tb_2026-01-15T15:04:05Z_notags",
		CreatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		Workset: ledger.Workset{
			AnchorCommit: "notags123456",
			Commits:      []string{"notags123456"},
		},
		Summary: ledger.Summary{
			What: "No tags",
			Why:  "Testing",
			How:  "Testing",
		},
		Tags: nil, // No tags
	}

	result := FormatMarkdown(entry)

	// Should not contain tags line
	if strings.Contains(result, "tags:") {
		t.Errorf("FormatMarkdown() should not include tags line when empty\nGot:\n%s", result)
	}
}

func TestFormatMarkdown_NoDiffstatField(t *testing.T) {
	entry := &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        "tb_2026-01-15T15:04:05Z_nodiff",
		CreatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		Workset: ledger.Workset{
			AnchorCommit: "nodiff123456",
			Commits:      []string{"nodiff123456"},
			Diffstat:     nil, // No diffstat
		},
		Summary: ledger.Summary{
			What: "No diffstat",
			Why:  "Testing",
			How:  "Testing",
		},
	}

	result := FormatMarkdown(entry)

	// Should not contain Files changed line
	if strings.Contains(result, "Files changed") {
		t.Errorf("FormatMarkdown() should not include Files changed when diffstat is nil\nGot:\n%s", result)
	}
}

func TestComputeCommitRange(t *testing.T) {
	tests := []struct {
		name  string
		entry *ledger.Entry
		want  string
	}{
		{
			name: "uses explicit range when present",
			entry: &ledger.Entry{
				Workset: ledger.Workset{
					Range:   "explicit..range",
					Commits: []string{"abc1234", "def5678"},
				},
			},
			want: "explicit..range",
		},
		{
			name: "computes range from commits",
			entry: &ledger.Entry{
				Workset: ledger.Workset{
					Commits: []string{"abc123456789", "def567890123"},
				},
			},
			want: "abc1234..def5678",
		},
		{
			name: "empty commits returns empty",
			entry: &ledger.Entry{
				Workset: ledger.Workset{
					Commits: []string{},
				},
			},
			want: "",
		},
		{
			name: "single commit returns same..same",
			entry: &ledger.Entry{
				Workset: ledger.Workset{
					Commits: []string{"abc123456789"},
				},
			},
			want: "abc1234..abc1234",
		},
		{
			name: "short commits preserved",
			entry: &ledger.Entry{
				Workset: ledger.Workset{
					Commits: []string{"abc", "def"},
				},
			},
			want: "abc..def",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeCommitRange(tt.entry)
			if got != tt.want {
				t.Errorf("computeCommitRange() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteMarkdownFiles(t *testing.T) {
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

			err := WriteMarkdownFiles(tt.entries, tmpDir)
			if err != nil {
				t.Fatalf("WriteMarkdownFiles() error = %v", err)
			}

			// Verify files were created
			for _, entry := range tt.entries {
				filename := filepath.Join(tmpDir, entry.ID+".md")

				// Check file exists
				data, err := os.ReadFile(filename)
				if err != nil {
					t.Errorf("failed to read file %s: %v", filename, err)
					continue
				}

				content := string(data)

				// Verify basic structure
				if !strings.Contains(content, "---") {
					t.Errorf("file %s missing frontmatter delimiters", filename)
				}
				if !strings.Contains(content, entry.ID) {
					t.Errorf("file %s missing entry ID", filename)
				}
				if !strings.Contains(content, entry.Summary.What) {
					t.Errorf("file %s missing What summary", filename)
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

func TestWriteMarkdownFiles_InvalidDirectory(t *testing.T) {
	entries := []*ledger.Entry{testEntry()}

	// Try to write to a non-existent directory
	err := WriteMarkdownFiles(entries, "/nonexistent/directory/path")
	if err == nil {
		t.Error("WriteMarkdownFiles() expected error for invalid directory")
	}
}

func TestWriteMarkdownFiles_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	entries := []*ledger.Entry{testEntry()}

	err := WriteMarkdownFiles(entries, tmpDir)
	if err != nil {
		t.Fatalf("WriteMarkdownFiles() error = %v", err)
	}

	// Check file permissions
	filename := filepath.Join(tmpDir, entries[0].ID+".md")
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

func TestWriteMarkdownFiles_ContentMatchesFormatMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	entry := testEntry()

	err := WriteMarkdownFiles([]*ledger.Entry{entry}, tmpDir)
	if err != nil {
		t.Fatalf("WriteMarkdownFiles() error = %v", err)
	}

	// Read the file
	filename := filepath.Join(tmpDir, entry.ID+".md")
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// Compare with FormatMarkdown output
	expected := FormatMarkdown(entry)
	if string(data) != expected {
		t.Errorf("WriteMarkdownFiles content doesn't match FormatMarkdown\nGot:\n%s\nWant:\n%s", string(data), expected)
	}
}

func TestFormatMarkdown_AnchorCommitTruncation(t *testing.T) {
	tests := []struct {
		name         string
		anchorCommit string
		wantAnchor   string
	}{
		{
			name:         "long SHA truncated to 12",
			anchorCommit: "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f",
			wantAnchor:   "anchor_commit: 8f2c1a9d7b0c",
		},
		{
			name:         "short SHA preserved",
			anchorCommit: "abc123",
			wantAnchor:   "anchor_commit: abc123",
		},
		{
			name:         "exactly 12 chars preserved",
			anchorCommit: "abcdef123456",
			wantAnchor:   "anchor_commit: abcdef123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := minimalEntry()
			entry.Workset.AnchorCommit = tt.anchorCommit

			result := FormatMarkdown(entry)

			if !strings.Contains(result, tt.wantAnchor) {
				t.Errorf("FormatMarkdown() anchor_commit = %q not found in output\nGot:\n%s", tt.wantAnchor, result)
			}
		})
	}
}
