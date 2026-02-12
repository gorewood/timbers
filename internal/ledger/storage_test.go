package ledger

import (
	"errors"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
)

// --- Test Helpers ---

// mockGitOps implements GitOps for testing (git operations only, no storage).
type mockGitOps struct {
	headSHA       string
	headErr       error
	logCommits    []git.Commit
	logErr        error
	reachableFrom []git.Commit
	reachableErr  error
	commitFiles   map[string][]string // SHA -> files; nil map = unknown (no filtering)
}

func newMockGitOps() *mockGitOps {
	return &mockGitOps{}
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

func (m *mockGitOps) CommitFiles(sha string) ([]string, error) {
	if m.commitFiles == nil {
		return nil, nil
	}
	files, ok := m.commitFiles[sha]
	if !ok {
		return nil, nil
	}
	return files, nil
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

// newTestStorage creates a Storage with a temp dir containing the given entries.
func newTestStorage(t *testing.T, mock *mockGitOps, entries ...*Entry) *Storage {
	t.Helper()
	dir := t.TempDir()
	for _, entry := range entries {
		writeTestEntryFile(t, dir, entry)
	}
	files := NewFileStorage(dir, noopGitAdd)
	return NewStorage(mock, files)
}

// --- GetLatestEntry Tests ---

func TestGetLatestEntry(t *testing.T) {
	tests := []struct {
		name        string
		entries     []*Entry
		wantAnchor  string
		wantNil     bool
		wantErr     bool
		errContains string
	}{
		{
			name: "returns entry with latest created_at",
			entries: []*Entry{
				makeTestEntry("oldercommit", time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)),
				makeTestEntry("newercommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			wantAnchor: "newercommit",
		},
		{
			name:        "returns ErrNoEntries for empty ledger",
			entries:     nil,
			wantNil:     true,
			wantErr:     true,
			errContains: "no ledger entries",
		},
		{
			name: "handles single entry",
			entries: []*Entry{
				makeTestEntry("onlycommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			wantAnchor: "onlycommit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStorage(t, newMockGitOps(), tt.entries...)

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

// --- GetPendingCommits Tests ---

func TestGetPendingCommits(t *testing.T) {
	tests := []struct {
		name            string
		entries         []*Entry
		setupMock       func(*mockGitOps)
		wantCommitCount int
		wantLatestNil   bool
		wantErr         bool
		wantStaleAnchor bool
		errContains     string
	}{
		{
			name: "returns commits since latest entry",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.logCommits = []git.Commit{
					{SHA: "commit1abc", Short: "commit1"},
					{SHA: "commit2def", Short: "commit2"},
				}
			},
			wantCommitCount: 2,
			wantLatestNil:   false,
		},
		{
			name:    "returns all commits when no entries exist",
			entries: nil,
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.reachableFrom = []git.Commit{
					{SHA: "commit1abc", Short: "commit1"},
					{SHA: "commit2def", Short: "commit2"},
					{SHA: "commit3ghi", Short: "commit3"},
				}
			},
			wantCommitCount: 3,
			wantLatestNil:   true,
		},
		{
			name: "returns empty when HEAD is the anchor",
			entries: []*Entry{
				makeTestEntry("headisanchr", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headisanchr"
				mock.logCommits = []git.Commit{} // no commits in range
			},
			wantCommitCount: 0,
			wantLatestNil:   false,
		},
		{
			name:    "handles HEAD error",
			entries: nil,
			setupMock: func(mock *mockGitOps) {
				mock.headErr = output.NewSystemError("failed to get HEAD")
			},
			wantErr:     true,
			errContains: "HEAD",
		},
		{
			name: "stale anchor falls back to all reachable commits",
			entries: []*Entry{
				makeTestEntry("staleanchor", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.logErr = output.NewSystemError("bad revision 'staleanchor..headsha1234'")
				mock.reachableFrom = []git.Commit{
					{SHA: "commit1abc", Short: "commit1"},
					{SHA: "commit2def", Short: "commit2"},
				}
			},
			wantCommitCount: 2,
			wantLatestNil:   false,
			wantStaleAnchor: true,
		},
		{
			name: "stale anchor with reachable error returns error",
			entries: []*Entry{
				makeTestEntry("staleanchor", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.logErr = output.NewSystemError("bad revision")
				mock.reachableErr = output.NewSystemError("reachable failed")
			},
			wantErr:     true,
			errContains: "reachable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			tt.setupMock(mock)
			store := newTestStorage(t, mock, tt.entries...)

			commits, latest, err := store.GetPendingCommits()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if tt.wantStaleAnchor {
				if !errors.Is(err, ErrStaleAnchor) {
					t.Errorf("expected ErrStaleAnchor, got %v", err)
				}
			} else if err != nil {
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

// --- GetLastNEntries Tests ---

func TestGetLastNEntries(t *testing.T) {
	older := makeTestEntry("oldercommit", time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC))
	newer := makeTestEntry("newercommit", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))

	tests := []struct {
		name      string
		entries   []*Entry
		count     int
		wantCount int
	}{
		{
			name:      "returns last N entries",
			entries:   []*Entry{older, newer},
			count:     1,
			wantCount: 1,
		},
		{
			name:      "returns all when count exceeds total",
			entries:   []*Entry{older, newer},
			count:     10,
			wantCount: 2,
		},
		{
			name:      "returns empty for no entries",
			entries:   nil,
			count:     5,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStorage(t, newMockGitOps(), tt.entries...)

			entries, err := store.GetLastNEntries(tt.count)
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

// --- Nil FileStorage Tests ---

func TestStorage_NilFiles(t *testing.T) {
	store := NewStorage(newMockGitOps(), nil)

	entries, err := store.ListEntries()
	if err != nil {
		t.Errorf("ListEntries with nil files: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries, got %d", len(entries))
	}

	_, latestErr := store.GetLatestEntry()
	if !errors.Is(latestErr, ErrNoEntries) {
		t.Errorf("expected ErrNoEntries, got %v", latestErr)
	}

	last, lastErr := store.GetLastNEntries(5)
	if lastErr != nil {
		t.Errorf("GetLastNEntries with nil files: %v", lastErr)
	}
	if len(last) != 0 {
		t.Errorf("expected 0 entries, got %d", len(last))
	}
}

// --- isLedgerOnlyCommit Tests ---

func TestIsLedgerOnlyCommit(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  bool
	}{
		{
			name:  "empty list is not ledger-only",
			files: nil,
			want:  false,
		},
		{
			name:  "all .timbers/ files is ledger-only",
			files: []string{".timbers/2026/01/15/tb_entry.json"},
			want:  true,
		},
		{
			name:  "multiple .timbers/ files is ledger-only",
			files: []string{".timbers/2026/01/15/tb_a.json", ".timbers/2026/01/15/tb_b.json"},
			want:  true,
		},
		{
			name:  "mixed files is not ledger-only",
			files: []string{".timbers/2026/01/15/tb_a.json", "cmd/main.go"},
			want:  false,
		},
		{
			name:  "only real files is not ledger-only",
			files: []string{"cmd/main.go", "internal/ledger/storage.go"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLedgerOnlyCommit(tt.files)
			if got != tt.want {
				t.Errorf("isLedgerOnlyCommit(%v) = %v, want %v", tt.files, got, tt.want)
			}
		})
	}
}

// --- Ledger-only commit filtering in GetPendingCommits ---

func TestGetPendingCommits_FiltersLedgerOnlyCommits(t *testing.T) {
	tests := []struct {
		name            string
		entries         []*Entry
		setupMock       func(*mockGitOps)
		wantCommitCount int
		wantSHAs        []string
	}{
		{
			name: "filters out ledger-only commit",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.logCommits = []git.Commit{
					{SHA: "realcommit1", Short: "real1"},
					{SHA: "ledgercommit", Short: "ledger"},
				}
				mock.commitFiles = map[string][]string{
					"realcommit1":  {"cmd/main.go", "internal/ledger/storage.go"},
					"ledgercommit": {".timbers/2026/01/15/tb_entry.json"},
				}
			},
			wantCommitCount: 1,
			wantSHAs:        []string{"realcommit1"},
		},
		{
			name: "keeps mixed commit",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.logCommits = []git.Commit{
					{SHA: "mixedcommit", Short: "mixed"},
				}
				mock.commitFiles = map[string][]string{
					"mixedcommit": {".timbers/2026/01/15/tb_entry.json", "README.md"},
				}
			},
			wantCommitCount: 1,
			wantSHAs:        []string{"mixedcommit"},
		},
		{
			name: "keeps commit when CommitFiles returns nil (unknown)",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.logCommits = []git.Commit{
					{SHA: "unknownsha1", Short: "unkn"},
				}
				// commitFiles is nil map — unknown = no filtering
			},
			wantCommitCount: 1,
			wantSHAs:        []string{"unknownsha1"},
		},
		{
			name: "filters all ledger-only commits to zero pending",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.logCommits = []git.Commit{
					{SHA: "ledger1", Short: "l1"},
					{SHA: "ledger2", Short: "l2"},
				}
				mock.commitFiles = map[string][]string{
					"ledger1": {".timbers/2026/01/15/tb_a.json"},
					"ledger2": {".timbers/2026/01/15/tb_b.json"},
				}
			},
			wantCommitCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			tt.setupMock(mock)
			store := newTestStorage(t, mock, tt.entries...)

			commits, _, err := store.GetPendingCommits()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(commits) != tt.wantCommitCount {
				t.Errorf("got %d commits, want %d", len(commits), tt.wantCommitCount)
			}

			for i, wantSHA := range tt.wantSHAs {
				if i < len(commits) && commits[i].SHA != wantSHA {
					t.Errorf("commits[%d].SHA = %q, want %q", i, commits[i].SHA, wantSHA)
				}
			}
		})
	}
}

// Ensure our mock satisfies the interface (compile-time check).
var _ GitOps = (*mockGitOps)(nil)
