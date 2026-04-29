package ledger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/output"
)

// --- Test Helpers ---

func noopGitAdd(_ string) error       { return nil }
func noopGitCommit(_, _ string) error { return nil }

type gitAddRecorder struct {
	paths []string
}

func (r *gitAddRecorder) add(path string) error {
	r.paths = append(r.paths, path)
	return nil
}

type gitCommitRecorder struct {
	paths    []string
	messages []string
}

func (r *gitCommitRecorder) commit(path string, message string) error {
	r.paths = append(r.paths, path)
	r.messages = append(r.messages, message)
	return nil
}

func writeTestEntryFile(t *testing.T, dir string, entry *Entry) {
	t.Helper()
	data, err := entry.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize test entry: %v", err)
	}
	entryDir := dir
	if sub := EntryDateDir(entry.ID); sub != "" {
		entryDir = filepath.Join(dir, sub)
	}
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("failed to create entry directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, entry.ID+".json"), data, 0o600); err != nil {
		t.Fatalf("failed to write test entry file: %v", err)
	}
}

func writeRawFile(t *testing.T, dir, name string, content []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
		t.Fatalf("failed to write test file %s: %v", name, err)
	}
}

// writeRawEntryFile writes raw content for an entry-like ID in the correct date subdirectory.
func writeRawEntryFile(t *testing.T, dir, id string, content []byte) {
	t.Helper()
	entryDir := dir
	if sub := EntryDateDir(id); sub != "" {
		entryDir = filepath.Join(dir, sub)
	}
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("failed to create entry directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, id+".json"), content, 0o600); err != nil {
		t.Fatalf("failed to write test file %s: %v", id, err)
	}
}

// --- Backward-compat filename Tests ---

// TestFileStorage_ReadsLegacyColonFilename verifies that ledgers written before
// the v0.18 colon-to-dash migration remain readable. The on-disk file uses the
// canonical ID (with colons) directly as its filename, but ReadEntry must still
// find it.
func TestFileStorage_ReadsLegacyColonFilename(t *testing.T) {
	dir := t.TempDir()
	entry := makeTestEntry("legacy01", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))

	// Write at the legacy (colon-encoded) path on purpose.
	data, _ := entry.ToJSON()
	sub := EntryDateDir(entry.ID)
	if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	legacyPath := filepath.Join(dir, sub, entry.ID+".json")
	if err := os.WriteFile(legacyPath, data, 0o600); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	store := NewFileStorage(dir, noopGitAdd, noopGitCommit)

	// ReadEntry should find the legacy file.
	got, err := store.ReadEntry(entry.ID)
	if err != nil {
		t.Fatalf("ReadEntry: %v", err)
	}
	if got.ID != entry.ID {
		t.Errorf("got ID %q, want %q", got.ID, entry.ID)
	}

	// EntryExists should report true.
	if !store.EntryExists(entry.ID) {
		t.Error("EntryExists returned false for legacy file")
	}

	// ListEntries should include it.
	entries, err := store.ListEntries()
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != entry.ID {
		t.Errorf("ListEntries returned %v, want [%q]", entries, entry.ID)
	}
}

// TestFileStorage_WriteRemovesLegacySibling verifies that writing an entry
// supersedes any pre-v0.18 colon-encoded sibling for the same ID, leaving a
// single canonical file on disk.
func TestFileStorage_WriteRemovesLegacySibling(t *testing.T) {
	dir := t.TempDir()
	entry := makeTestEntry("upgrade01", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))

	// Pre-create legacy file.
	data, _ := entry.ToJSON()
	sub := EntryDateDir(entry.ID)
	if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	legacyPath := filepath.Join(dir, sub, entry.ID+".json")
	if err := os.WriteFile(legacyPath, data, 0o600); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	store := NewFileStorage(dir, noopGitAdd, noopGitCommit)
	if err := store.WriteEntry(entry, true); err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}

	// Legacy file should be gone.
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Errorf("legacy file should be removed after canonical write, stat err = %v", err)
	}

	// Canonical file should exist.
	canonicalPath := filepath.Join(dir, sub, IDToFilename(entry.ID)+".json")
	if _, err := os.Stat(canonicalPath); err != nil {
		t.Errorf("canonical file should exist after write: %v", err)
	}
}

// TestFileStorage_MigrateLegacyFilenames verifies the bulk migration walks
// .timbers/, renames colon files to canonical dashed names, and reports the
// affected IDs.
func TestFileStorage_MigrateLegacyFilenames(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStorage(dir, noopGitAdd, noopGitCommit)

	mkLegacy := func(id string) string {
		t.Helper()
		entry := makeTestEntry("anchor"+id[len(id)-4:], time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
		entry.ID = id
		data, _ := entry.ToJSON()
		sub := EntryDateDir(id)
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		p := filepath.Join(dir, sub, id+".json")
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		return p
	}

	legacyA := mkLegacy("tb_2026-01-15T10:00:00Z_aaaaaa")
	legacyB := mkLegacy("tb_2026-01-15T11:00:00Z_bbbbbb")

	migrated, err := store.MigrateLegacyFilenames()
	if err != nil {
		t.Fatalf("MigrateLegacyFilenames: %v", err)
	}
	if len(migrated) != 2 {
		t.Errorf("migrated count = %d, want 2", len(migrated))
	}

	// Legacy files gone, canonical present.
	for _, p := range []string{legacyA, legacyB} {
		if _, statErr := os.Stat(p); !os.IsNotExist(statErr) {
			t.Errorf("legacy %s should be removed: %v", p, statErr)
		}
	}
	for _, id := range []string{"tb_2026-01-15T10:00:00Z_aaaaaa", "tb_2026-01-15T11:00:00Z_bbbbbb"} {
		canonical := filepath.Join(dir, EntryDateDir(id), IDToFilename(id)+".json")
		if _, statErr := os.Stat(canonical); statErr != nil {
			t.Errorf("canonical %s missing: %v", canonical, statErr)
		}
	}

	// Idempotent: running again is a no-op.
	again, err := store.MigrateLegacyFilenames()
	if err != nil {
		t.Fatalf("MigrateLegacyFilenames (second run): %v", err)
	}
	if len(again) != 0 {
		t.Errorf("second run migrated %d, want 0", len(again))
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
				writeRawEntryFile(t, dir, "tb_2026-01-15T10:00:00Z_baddat", []byte("not json"))
			},
			entryID:     "tb_2026-01-15T10:00:00Z_baddat",
			wantErr:     true,
			errContains: "parse",
		},
		{
			name: "returns ErrNotTimbersNote for non-timbers JSON",
			setupDir: func(t *testing.T, dir string) {
				writeRawEntryFile(t, dir, "tb_2026-01-15T10:00:00Z_othert",
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
			store := NewFileStorage(dir, noopGitAdd, noopGitCommit)

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

	store := NewFileStorage(dir, noopGitAdd, noopGitCommit)

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
			name: "walks into subdirectories",
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
			store := NewFileStorage(dir, noopGitAdd, noopGitCommit)

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
	store := NewFileStorage("/nonexistent/dir", noopGitAdd, noopGitCommit)

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
			store := NewFileStorage(dir, noopGitAdd, noopGitCommit)

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
			addRecorder := &gitAddRecorder{}
			commitRecorder := &gitCommitRecorder{}
			store := NewFileStorage(dir, addRecorder.add, commitRecorder.commit)

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
			sub := EntryDateDir(tt.entry.ID)
			path := filepath.Join(dir, sub, IDToFilename(tt.entry.ID)+".json")
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
			if len(addRecorder.paths) == 0 {
				t.Error("expected git add to be called")
			} else if addRecorder.paths[0] != path {
				t.Errorf("git add path = %q, want %q", addRecorder.paths[0], path)
			}

			// Verify git commit was called
			if len(commitRecorder.paths) == 0 {
				t.Error("expected git commit to be called")
			} else {
				if commitRecorder.paths[0] != path {
					t.Errorf("git commit path = %q, want %q", commitRecorder.paths[0], path)
				}
				wantMsg := "timbers: document " + tt.entry.ID
				if commitRecorder.messages[0] != wantMsg {
					t.Errorf("git commit message = %q, want %q", commitRecorder.messages[0], wantMsg)
				}
			}
		})
	}
}

func TestFileStorage_WriteEntry_GitAddError(t *testing.T) {
	dir := t.TempDir()
	failGitAdd := func(_ string) error {
		return output.NewSystemError("git add failed")
	}
	store := NewFileStorage(dir, failGitAdd, noopGitCommit)

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
	sub := EntryDateDir(entry.ID)
	path := filepath.Join(dir, sub, IDToFilename(entry.ID)+".json")
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("entry file should still exist after git add failure: %v", statErr)
	}
}

func TestFileStorage_WriteEntry_GitCommitError(t *testing.T) {
	dir := t.TempDir()
	failGitCommit := func(_, _ string) error {
		return output.NewSystemError("git commit failed")
	}
	store := NewFileStorage(dir, noopGitAdd, failGitCommit)

	entry := makeTestEntry("commitfail1", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	err := store.WriteEntry(entry, false)

	if err == nil {
		t.Error("expected error, got nil")
		return
	}
	if !containsString(err.Error(), "commit entry") {
		t.Errorf("error %q should mention committing entry", err.Error())
	}
}

func TestFileStorage_WriteEntry_CommitMessageFormat(t *testing.T) {
	dir := t.TempDir()
	commitRecorder := &gitCommitRecorder{}
	store := NewFileStorage(dir, noopGitAdd, commitRecorder.commit)

	entry := makeTestEntry("msgformat1", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	if err := store.WriteEntry(entry, false); err != nil {
		t.Fatalf("WriteEntry failed: %v", err)
	}

	if len(commitRecorder.messages) != 1 {
		t.Fatalf("expected 1 commit call, got %d", len(commitRecorder.messages))
	}

	wantPrefix := "timbers: document "
	if !strings.HasPrefix(commitRecorder.messages[0], wantPrefix) {
		t.Errorf("commit message %q should start with %q", commitRecorder.messages[0], wantPrefix)
	}
	if !strings.Contains(commitRecorder.messages[0], entry.ID) {
		t.Errorf("commit message %q should contain entry ID %q", commitRecorder.messages[0], entry.ID)
	}
}

func TestFileStorage_WriteEntry_CommitPathspec(t *testing.T) {
	dir := t.TempDir()
	commitRecorder := &gitCommitRecorder{}
	store := NewFileStorage(dir, noopGitAdd, commitRecorder.commit)

	entry := makeTestEntry("pathspec01", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	if err := store.WriteEntry(entry, false); err != nil {
		t.Fatalf("WriteEntry failed: %v", err)
	}

	if len(commitRecorder.paths) != 1 {
		t.Fatalf("expected 1 commit call, got %d", len(commitRecorder.paths))
	}

	// The path should be the full entry file path (gitCommit receives it directly;
	// DefaultGitCommit places it after -- in the git command)
	sub := EntryDateDir(entry.ID)
	wantPath := filepath.Join(dir, sub, IDToFilename(entry.ID)+".json")
	if commitRecorder.paths[0] != wantPath {
		t.Errorf("commit path = %q, want %q", commitRecorder.paths[0], wantPath)
	}
}

func TestFileStorage_WriteEntry_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStorage(dir, noopGitAdd, noopGitCommit)

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

	store := NewFileStorage(dir, noopGitAdd, noopGitCommit)

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
	store := NewFileStorage(dir, noopGitAdd, noopGitCommit)

	if !store.DirExists() {
		t.Error("expected directory to exist")
	}

	storeNone := NewFileStorage("/nonexistent/path", noopGitAdd, noopGitCommit)
	if storeNone.DirExists() {
		t.Error("expected directory to not exist")
	}
}

// --- No temp files left behind ---

func TestFileStorage_WriteEntry_NoTempFilesLeftBehind(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStorage(dir, noopGitAdd, noopGitCommit)

	entry := makeTestEntry("cleancommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	if err := store.WriteEntry(entry, false); err != nil {
		t.Fatalf("WriteEntry failed: %v", err)
	}

	// Check that no temp files remain (walk entire tree)
	walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			if name := d.Name(); len(name) > 0 && name[0] == '.' {
				t.Errorf("temp file left behind: %s", path)
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("failed to walk dir: %v", walkErr)
	}
}
