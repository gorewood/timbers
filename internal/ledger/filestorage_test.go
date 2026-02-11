package ledger

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/output"
)

// --- Test Helpers ---

func noopGitAdd(_ string) error { return nil }

type gitAddRecorder struct {
	paths []string
}

func (r *gitAddRecorder) add(path string) error {
	r.paths = append(r.paths, path)
	return nil
}

func writeTestEntryFile(t *testing.T, dir string, entry *Entry) {
	t.Helper()
	data, err := entry.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize test entry: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, entry.ID+".json"), data, 0o600); err != nil {
		t.Fatalf("failed to write test entry file: %v", err)
	}
}

func writeRawFile(t *testing.T, dir, name string, content []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
		t.Fatalf("failed to write test file %s: %v", name, err)
	}
}

// --- ReadEntry Tests ---

func TestFileStorage_ReadEntry(t *testing.T) {
	tests := []struct {
		name        string
		setupDir    func(t *testing.T, dir string)
		entryID     string
		wantErr     bool
		errContains string
		wantAnchor  string
	}{
		{
			name: "reads entry successfully",
			setupDir: func(t *testing.T, dir string) {
				entry := makeTestEntry("abc123def456", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				writeTestEntryFile(t, dir, entry)
			},
			entryID:    GenerateID("abc123def456", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			wantAnchor: "abc123def456",
		},
		{
			name:        "returns error for missing entry",
			setupDir:    func(t *testing.T, dir string) {},
			entryID:     "tb_2026-01-15T10:00:00Z_nonexs",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "returns error for invalid JSON",
			setupDir: func(t *testing.T, dir string) {
				writeRawFile(t, dir, "tb_2026-01-15T10:00:00Z_baddat.json", []byte("not json"))
			},
			entryID:     "tb_2026-01-15T10:00:00Z_baddat",
			wantErr:     true,
			errContains: "parse",
		},
		{
			name: "returns ErrNotTimbersNote for non-timbers JSON",
			setupDir: func(t *testing.T, dir string) {
				writeRawFile(t, dir, "tb_2026-01-15T10:00:00Z_othert.json",
					[]byte(`{"schema": "other.tool/v1", "type": "annotation"}`))
			},
			entryID:     "tb_2026-01-15T10:00:00Z_othert",
			wantErr:     true,
			errContains: "not a timbers note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupDir(t, dir)
			store := NewFileStorage(dir, noopGitAdd)

			entry, err := store.ReadEntry(tt.entryID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if entry == nil {
				t.Error("expected entry, got nil")
				return
			}
			if entry.Workset.AnchorCommit != tt.wantAnchor {
				t.Errorf("anchor = %q, want %q", entry.Workset.AnchorCommit, tt.wantAnchor)
			}
		})
	}
}

func TestFileStorage_ReadEntry_NotTimbersNote(t *testing.T) {
	dir := t.TempDir()
	writeRawFile(t, dir, "other.json",
		[]byte(`{"schema": "other.tool/v1", "type": "annotation"}`))

	store := NewFileStorage(dir, noopGitAdd)

	_, err := store.ReadEntry("other")
	if err == nil {
		t.Error("expected error, got nil")
		return
	}

	if !errors.Is(err, ErrNotTimbersNote) {
		t.Errorf("expected ErrNotTimbersNote, got %v", err)
	}
}

// --- ListEntries Tests ---

func TestFileStorage_ListEntries(t *testing.T) {
	tests := []struct {
		name      string
		setupDir  func(t *testing.T, dir string)
		wantCount int
		wantErr   bool
	}{
		{
			name: "lists multiple entries",
			setupDir: func(t *testing.T, dir string) {
				writeTestEntryFile(t, dir, makeTestEntry("commit1aaa", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)))
				writeTestEntryFile(t, dir, makeTestEntry("commit2bbb", time.Date(2026, 1, 15, 11, 0, 0, 0, time.UTC)))
			},
			wantCount: 2,
		},
		{
			name:      "returns empty for empty directory",
			setupDir:  func(t *testing.T, dir string) {},
			wantCount: 0,
		},
		{
			name: "skips entries with parse errors",
			setupDir: func(t *testing.T, dir string) {
				writeTestEntryFile(t, dir, makeTestEntry("goodcommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)))
				writeRawFile(t, dir, "bad.json", []byte("invalid json"))
			},
			wantCount: 1,
		},
		{
			name: "ignores non-json files",
			setupDir: func(t *testing.T, dir string) {
				writeTestEntryFile(t, dir, makeTestEntry("goodcommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)))
				writeRawFile(t, dir, "README.md", []byte("# hello"))
			},
			wantCount: 1,
		},
		{
			name: "ignores subdirectories",
			setupDir: func(t *testing.T, dir string) {
				writeTestEntryFile(t, dir, makeTestEntry("goodcommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)))
				if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0o755); err != nil {
					t.Fatalf("failed to create subdir: %v", err)
				}
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupDir(t, dir)
			store := NewFileStorage(dir, noopGitAdd)

			entries, err := store.ListEntries()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(entries) != tt.wantCount {
				t.Errorf("got %d entries, want %d", len(entries), tt.wantCount)
			}
		})
	}
}

func TestFileStorage_ListEntries_NonexistentDir(t *testing.T) {
	store := NewFileStorage("/nonexistent/dir", noopGitAdd)

	entries, err := store.ListEntries()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

// --- ListEntriesWithStats Tests ---

func TestFileStorage_ListEntriesWithStats(t *testing.T) {
	tests := []struct {
		name        string
		setupDir    func(t *testing.T, dir string)
		wantEntries int
		wantStats   *ListStats
	}{
		{
			name: "all valid timbers entries",
			setupDir: func(t *testing.T, dir string) {
				writeTestEntryFile(t, dir, makeTestEntry("commit1aaa", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)))
				writeTestEntryFile(t, dir, makeTestEntry("commit2bbb", time.Date(2026, 1, 15, 11, 0, 0, 0, time.UTC)))
			},
			wantEntries: 2,
			wantStats:   &ListStats{Total: 2, Parsed: 2, Skipped: 0, NotTimbers: 0, ParseErrors: 0},
		},
		{
			name: "mixed timbers and non-timbers files",
			setupDir: func(t *testing.T, dir string) {
				writeTestEntryFile(t, dir, makeTestEntry("timberscommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)))
				writeRawFile(t, dir, "other.json",
					[]byte(`{"schema": "othertool/v1", "type": "annotation"}`))
				writeRawFile(t, dir, "bad.json", []byte("not valid json"))
			},
			wantEntries: 1,
			wantStats:   &ListStats{Total: 3, Parsed: 1, Skipped: 2, NotTimbers: 1, ParseErrors: 1},
		},
		{
			name: "only non-timbers files",
			setupDir: func(t *testing.T, dir string) {
				writeRawFile(t, dir, "other1.json",
					[]byte(`{"schema": "beads/v1", "id": "bd-123"}`))
				writeRawFile(t, dir, "other2.json",
					[]byte(`{"type": "review-comment"}`))
			},
			wantEntries: 0,
			wantStats:   &ListStats{Total: 2, Parsed: 0, Skipped: 2, NotTimbers: 2, ParseErrors: 0},
		},
		{
			name:        "empty directory",
			setupDir:    func(t *testing.T, dir string) {},
			wantEntries: 0,
			wantStats:   &ListStats{Total: 0, Parsed: 0, Skipped: 0, NotTimbers: 0, ParseErrors: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupDir(t, dir)
			store := NewFileStorage(dir, noopGitAdd)

			entries, stats, err := store.ListEntriesWithStats()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(entries) != tt.wantEntries {
				t.Errorf("got %d entries, want %d", len(entries), tt.wantEntries)
			}

			if stats.Total != tt.wantStats.Total {
				t.Errorf("stats.Total = %d, want %d", stats.Total, tt.wantStats.Total)
			}
			if stats.Parsed != tt.wantStats.Parsed {
				t.Errorf("stats.Parsed = %d, want %d", stats.Parsed, tt.wantStats.Parsed)
			}
			if stats.Skipped != tt.wantStats.Skipped {
				t.Errorf("stats.Skipped = %d, want %d", stats.Skipped, tt.wantStats.Skipped)
			}
			if stats.NotTimbers != tt.wantStats.NotTimbers {
				t.Errorf("stats.NotTimbers = %d, want %d", stats.NotTimbers, tt.wantStats.NotTimbers)
			}
			if stats.ParseErrors != tt.wantStats.ParseErrors {
				t.Errorf("stats.ParseErrors = %d, want %d", stats.ParseErrors, tt.wantStats.ParseErrors)
			}
		})
	}
}

// --- WriteEntry Tests ---

func TestFileStorage_WriteEntry(t *testing.T) {
	tests := []struct {
		name        string
		entry       *Entry
		force       bool
		setupDir    func(t *testing.T, dir string)
		wantErr     bool
		wantCode    int
		errContains string
	}{
		{
			name:     "writes valid entry successfully",
			entry:    makeTestEntry("newcommit12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			force:    false,
			setupDir: func(t *testing.T, dir string) {},
			wantErr:  false,
		},
		{
			name: "rejects invalid entry",
			entry: &Entry{
				Schema: SchemaVersion,
				Kind:   KindEntry,
			},
			force:       false,
			setupDir:    func(t *testing.T, dir string) {},
			wantErr:     true,
			errContains: "missing required fields",
		},
		{
			name:  "returns conflict when entry exists and force=false",
			entry: makeTestEntry("existingabc", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			force: false,
			setupDir: func(t *testing.T, dir string) {
				existing := makeTestEntry("existingabc", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				writeTestEntryFile(t, dir, existing)
			},
			wantErr:  true,
			wantCode: output.ExitConflict,
		},
		{
			name:  "overwrites when force=true",
			entry: makeTestEntry("existingdef", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			force: true,
			setupDir: func(t *testing.T, dir string) {
				existing := makeTestEntry("existingdef", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				writeTestEntryFile(t, dir, existing)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupDir(t, dir)
			recorder := &gitAddRecorder{}
			store := NewFileStorage(dir, recorder.add)

			err := store.WriteEntry(tt.entry, tt.force)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else {
					if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
						t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
					}
					if tt.wantCode != 0 {
						code := output.GetExitCode(err)
						if code != tt.wantCode {
							t.Errorf("exit code = %d, want %d", code, tt.wantCode)
						}
					}
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify file exists with correct content
			path := filepath.Join(dir, tt.entry.ID+".json")
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("entry file not found: %v", readErr)
			}

			readBack, parseErr := FromJSON(data)
			if parseErr != nil {
				t.Fatalf("failed to parse written entry: %v", parseErr)
			}
			if readBack.ID != tt.entry.ID {
				t.Errorf("ID = %q, want %q", readBack.ID, tt.entry.ID)
			}

			// Verify git add was called
			if len(recorder.paths) == 0 {
				t.Error("expected git add to be called")
			} else if recorder.paths[0] != path {
				t.Errorf("git add path = %q, want %q", recorder.paths[0], path)
			}
		})
	}
}

func TestFileStorage_WriteEntry_GitAddError(t *testing.T) {
	dir := t.TempDir()
	failGitAdd := func(_ string) error {
		return output.NewSystemError("git add failed")
	}
	store := NewFileStorage(dir, failGitAdd)

	entry := makeTestEntry("failcommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	err := store.WriteEntry(entry, false)

	if err == nil {
		t.Error("expected error, got nil")
		return
	}
	if !containsString(err.Error(), "stage") {
		t.Errorf("error %q should mention staging", err.Error())
	}

	// File should still exist (rename succeeded before git add failed)
	path := filepath.Join(dir, entry.ID+".json")
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("entry file should still exist after git add failure: %v", statErr)
	}
}

func TestFileStorage_WriteEntry_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStorage(dir, noopGitAdd)

	entry := makeTestEntry("roundtrip1", time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC))
	entry.Tags = []string{"security", "auth"}
	entry.Summary.What = "Fixed authentication bypass"
	entry.Summary.Why = "User input wasn't sanitized before JWT validation"
	entry.Summary.How = "Added input validation middleware"

	if err := store.WriteEntry(entry, false); err != nil {
		t.Fatalf("WriteEntry failed: %v", err)
	}

	readBack, err := store.ReadEntry(entry.ID)
	if err != nil {
		t.Fatalf("ReadEntry failed: %v", err)
	}

	if readBack.ID != entry.ID {
		t.Errorf("ID = %q, want %q", readBack.ID, entry.ID)
	}
	if readBack.Summary.What != entry.Summary.What {
		t.Errorf("What = %q, want %q", readBack.Summary.What, entry.Summary.What)
	}
	if readBack.Summary.Why != entry.Summary.Why {
		t.Errorf("Why = %q, want %q", readBack.Summary.Why, entry.Summary.Why)
	}
	if len(readBack.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(readBack.Tags))
	}
}

// --- EntryExists Tests ---

func TestFileStorage_EntryExists(t *testing.T) {
	dir := t.TempDir()
	entry := makeTestEntry("existscommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	writeTestEntryFile(t, dir, entry)

	store := NewFileStorage(dir, noopGitAdd)

	if !store.EntryExists(entry.ID) {
		t.Error("expected entry to exist")
	}

	if store.EntryExists("nonexistent") {
		t.Error("expected entry to not exist")
	}
}

// --- DirExists Tests ---

func TestFileStorage_DirExists(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStorage(dir, noopGitAdd)

	if !store.DirExists() {
		t.Error("expected directory to exist")
	}

	storeNone := NewFileStorage("/nonexistent/path", noopGitAdd)
	if storeNone.DirExists() {
		t.Error("expected directory to not exist")
	}
}

// --- No temp files left behind ---

func TestFileStorage_WriteEntry_NoTempFilesLeftBehind(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStorage(dir, noopGitAdd)

	entry := makeTestEntry("cleancommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	if err := store.WriteEntry(entry, false); err != nil {
		t.Fatalf("WriteEntry failed: %v", err)
	}

	// Check that no temp files remain
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}
	for _, dirEntry := range dirEntries {
		if name := dirEntry.Name(); len(name) > 0 && name[0] == '.' {
			t.Errorf("temp file left behind: %s", name)
		}
	}
}
