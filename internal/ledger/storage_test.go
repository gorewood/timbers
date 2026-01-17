package ledger

import (
	"errors"
	"testing"
	"time"

	"github.com/steveyegge/timbers/internal/git"
	"github.com/steveyegge/timbers/internal/output"
)

// --- Test Helpers ---

// mockGitOps implements GitOps for testing.
type mockGitOps struct {
	notes         map[string][]byte // commit -> note content
	notedCommits  []string
	listNotesErr  error
	readNoteErr   map[string]error
	writeNoteErr  error
	headSHA       string
	headErr       error
	logCommits    []git.Commit
	logErr        error
	reachableFrom []git.Commit
	reachableErr  error
	writtenNotes  []writtenNote // track writes for verification
	existingNotes map[string]bool
}

type writtenNote struct {
	commit  string
	content string
	force   bool
}

func newMockGitOps() *mockGitOps {
	return &mockGitOps{
		notes:         make(map[string][]byte),
		readNoteErr:   make(map[string]error),
		existingNotes: make(map[string]bool),
	}
}

func (m *mockGitOps) ReadNote(commit string) ([]byte, error) {
	if err, ok := m.readNoteErr[commit]; ok {
		return nil, err
	}
	content, ok := m.notes[commit]
	if !ok {
		return nil, output.NewUserError("note not found for commit: " + commit)
	}
	return content, nil
}

func (m *mockGitOps) WriteNote(commit string, content string, force bool) error {
	m.writtenNotes = append(m.writtenNotes, writtenNote{commit, content, force})
	if m.writeNoteErr != nil {
		return m.writeNoteErr
	}
	// Simulate conflict: if note exists and force=false
	if m.existingNotes[commit] && !force {
		return output.NewSystemError("note already exists")
	}
	m.notes[commit] = []byte(content)
	m.existingNotes[commit] = true
	return nil
}

func (m *mockGitOps) ListNotedCommits() ([]string, error) {
	if m.listNotesErr != nil {
		return nil, m.listNotesErr
	}
	return m.notedCommits, nil
}

func (m *mockGitOps) HEAD() (string, error) {
	if m.headErr != nil {
		return "", m.headErr
	}
	return m.headSHA, nil
}

func (m *mockGitOps) Log(fromRef, toRef string) ([]git.Commit, error) {
	if m.logErr != nil {
		return nil, m.logErr
	}
	return m.logCommits, nil
}

func (m *mockGitOps) CommitsReachableFrom(sha string) ([]git.Commit, error) {
	if m.reachableErr != nil {
		return nil, m.reachableErr
	}
	return m.reachableFrom, nil
}

func (m *mockGitOps) GetDiffstat(fromRef, toRef string) (git.Diffstat, error) {
	return git.Diffstat{}, nil
}

func (m *mockGitOps) PushNotes(remote string) error {
	return nil
}

// makeTestEntry creates a valid entry for testing.
func makeTestEntry(anchor string, createdAt time.Time) *Entry {
	return &Entry{
		Schema:    SchemaVersion,
		Kind:      KindEntry,
		ID:        GenerateID(anchor, createdAt),
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Workset: Workset{
			AnchorCommit: anchor,
			Commits:      []string{anchor},
		},
		Summary: Summary{
			What: "test what",
			Why:  "test why",
			How:  "test how",
		},
	}
}

// --- ReadEntry Tests ---

func TestReadEntry(t *testing.T) {
	tests := []struct {
		name        string
		anchor      string
		setupMock   func(*mockGitOps)
		wantErr     bool
		errContains string
	}{
		{
			name:   "reads entry successfully",
			anchor: "abc123def456",
			setupMock: func(mock *mockGitOps) {
				entry := makeTestEntry("abc123def456", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				data, _ := entry.ToJSON()
				mock.notes["abc123def456"] = data
			},
			wantErr: false,
		},
		{
			name:   "returns error for missing entry",
			anchor: "nonexistent",
			setupMock: func(mock *mockGitOps) {
				// no note added
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:   "returns error for invalid JSON",
			anchor: "baddata123",
			setupMock: func(mock *mockGitOps) {
				mock.notes["baddata123"] = []byte("not valid json")
			},
			wantErr:     true,
			errContains: "parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			tt.setupMock(mock)
			store := NewStorage(mock)

			entry, err := store.ReadEntry(tt.anchor)

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
			if entry.Workset.AnchorCommit != tt.anchor {
				t.Errorf("anchor = %q, want %q", entry.Workset.AnchorCommit, tt.anchor)
			}
		})
	}
}

// --- ListEntries Tests ---

func TestListEntries(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mockGitOps)
		wantCount   int
		wantErr     bool
		errContains string
	}{
		{
			name: "lists multiple entries",
			setupMock: func(mock *mockGitOps) {
				e1 := makeTestEntry("commit1aaa", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				e2 := makeTestEntry("commit2bbb", time.Date(2026, 1, 15, 11, 0, 0, 0, time.UTC))
				d1, _ := e1.ToJSON()
				d2, _ := e2.ToJSON()
				mock.notes["commit1aaa"] = d1
				mock.notes["commit2bbb"] = d2
				mock.notedCommits = []string{"commit1aaa", "commit2bbb"}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "returns empty list for empty ledger",
			setupMock: func(mock *mockGitOps) {
				mock.notedCommits = []string{}
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "handles list error",
			setupMock: func(mock *mockGitOps) {
				mock.listNotesErr = output.NewSystemError("git notes list failed")
			},
			wantErr:     true,
			errContains: "failed",
		},
		{
			name: "skips entries with parse errors",
			setupMock: func(mock *mockGitOps) {
				e1 := makeTestEntry("goodcommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				d1, _ := e1.ToJSON()
				mock.notes["goodcommit"] = d1
				mock.notes["badcommit"] = []byte("invalid json")
				mock.notedCommits = []string{"goodcommit", "badcommit"}
			},
			wantCount: 1, // only the valid one
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			tt.setupMock(mock)
			store := NewStorage(mock)

			entries, err := store.ListEntries()

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
			if len(entries) != tt.wantCount {
				t.Errorf("got %d entries, want %d", len(entries), tt.wantCount)
			}
		})
	}
}

// --- GetLatestEntry Tests ---

func TestGetLatestEntry(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mockGitOps)
		wantAnchor  string
		wantNil     bool
		wantErr     bool
		errContains string
	}{
		{
			name: "returns entry with latest created_at",
			setupMock: func(mock *mockGitOps) {
				older := makeTestEntry("oldercommit", time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC))
				newer := makeTestEntry("newercommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				d1, _ := older.ToJSON()
				d2, _ := newer.ToJSON()
				mock.notes["oldercommit"] = d1
				mock.notes["newercommit"] = d2
				mock.notedCommits = []string{"oldercommit", "newercommit"}
			},
			wantAnchor: "newercommit",
			wantNil:    false,
			wantErr:    false,
		},
		{
			name: "returns ErrNoEntries for empty ledger",
			setupMock: func(mock *mockGitOps) {
				mock.notedCommits = []string{}
			},
			wantNil:     true,
			wantErr:     true,
			errContains: "no ledger entries",
		},
		{
			name: "handles single entry",
			setupMock: func(mock *mockGitOps) {
				entry := makeTestEntry("onlycommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				data, _ := entry.ToJSON()
				mock.notes["onlycommit"] = data
				mock.notedCommits = []string{"onlycommit"}
			},
			wantAnchor: "onlycommit",
			wantNil:    false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			tt.setupMock(mock)
			store := NewStorage(mock)

			entry, err := store.GetLatestEntry()

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

			if tt.wantNil {
				if entry != nil {
					t.Errorf("expected nil, got entry with anchor %q", entry.Workset.AnchorCommit)
				}
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

// --- WriteEntry Tests ---

func TestWriteEntry(t *testing.T) {
	tests := []struct {
		name        string
		entry       *Entry
		force       bool
		setupMock   func(*mockGitOps)
		wantErr     bool
		wantCode    int
		errContains string
	}{
		{
			name:  "writes valid entry successfully",
			entry: makeTestEntry("newcommit12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			force: false,
			setupMock: func(mock *mockGitOps) {
				// no existing note
			},
			wantErr: false,
		},
		{
			name: "rejects invalid entry",
			entry: &Entry{
				// Missing required fields
				Schema: SchemaVersion,
				Kind:   KindEntry,
			},
			force:       false,
			setupMock:   func(mock *mockGitOps) {},
			wantErr:     true,
			errContains: "missing required fields",
		},
		{
			name:  "returns conflict when note exists and force=false",
			entry: makeTestEntry("existingabc", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			force: false,
			setupMock: func(mock *mockGitOps) {
				// Need actual note content for ReadNote to succeed
				existing := makeTestEntry("existingabc", time.Date(2026, 1, 14, 10, 0, 0, 0, time.UTC))
				data, _ := existing.ToJSON()
				mock.notes["existingabc"] = data
				mock.existingNotes["existingabc"] = true
			},
			wantErr:  true,
			wantCode: output.ExitConflict,
		},
		{
			name:  "overwrites when force=true",
			entry: makeTestEntry("existingdef", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			force: true,
			setupMock: func(mock *mockGitOps) {
				// Need actual note content for the existing entry
				existing := makeTestEntry("existingdef", time.Date(2026, 1, 14, 10, 0, 0, 0, time.UTC))
				data, _ := existing.ToJSON()
				mock.notes["existingdef"] = data
				mock.existingNotes["existingdef"] = true
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			tt.setupMock(mock)
			store := NewStorage(mock)

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

			// Verify the note was written
			if len(mock.writtenNotes) == 0 {
				t.Error("expected note to be written")
			}
		})
	}
}

// --- GetPendingCommits Tests ---

func TestGetPendingCommits(t *testing.T) {
	tests := []struct {
		name            string
		setupMock       func(*mockGitOps)
		wantCommitCount int
		wantLatestNil   bool
		wantErr         bool
		errContains     string
	}{
		{
			name: "returns commits since latest entry",
			setupMock: func(mock *mockGitOps) {
				// Setup latest entry
				entry := makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				data, _ := entry.ToJSON()
				mock.notes["anchorsha12"] = data
				mock.notedCommits = []string{"anchorsha12"}
				mock.headSHA = "headsha1234"
				mock.logCommits = []git.Commit{
					{SHA: "commit1abc", Short: "commit1"},
					{SHA: "commit2def", Short: "commit2"},
				}
			},
			wantCommitCount: 2,
			wantLatestNil:   false,
			wantErr:         false,
		},
		{
			name: "returns all commits when no entries exist",
			setupMock: func(mock *mockGitOps) {
				mock.notedCommits = []string{}
				mock.headSHA = "headsha1234"
				mock.reachableFrom = []git.Commit{
					{SHA: "commit1abc", Short: "commit1"},
					{SHA: "commit2def", Short: "commit2"},
					{SHA: "commit3ghi", Short: "commit3"},
				}
			},
			wantCommitCount: 3,
			wantLatestNil:   true,
			wantErr:         false,
		},
		{
			name: "returns empty when HEAD is the anchor",
			setupMock: func(mock *mockGitOps) {
				entry := makeTestEntry("headisanchr", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				data, _ := entry.ToJSON()
				mock.notes["headisanchr"] = data
				mock.notedCommits = []string{"headisanchr"}
				mock.headSHA = "headisanchr"
				mock.logCommits = []git.Commit{} // no commits in range
			},
			wantCommitCount: 0,
			wantLatestNil:   false,
			wantErr:         false,
		},
		{
			name: "handles HEAD error",
			setupMock: func(mock *mockGitOps) {
				mock.headErr = output.NewSystemError("failed to get HEAD")
			},
			wantErr:     true,
			errContains: "HEAD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			tt.setupMock(mock)
			store := NewStorage(mock)

			commits, latest, err := store.GetPendingCommits()

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

			if len(commits) != tt.wantCommitCount {
				t.Errorf("got %d commits, want %d", len(commits), tt.wantCommitCount)
			}

			if tt.wantLatestNil {
				if latest != nil {
					t.Errorf("expected nil latest, got entry with anchor %q", latest.Workset.AnchorCommit)
				}
			} else {
				if latest == nil {
					t.Error("expected latest entry, got nil")
				}
			}
		})
	}
}

// Ensure our mock satisfies the interface (compile-time check).
var _ GitOps = (*mockGitOps)(nil)

// Suppress unused import warning.
var _ = errors.New
