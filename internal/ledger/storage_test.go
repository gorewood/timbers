package ledger

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
)

// --- Test Helpers ---

// mockGitOps implements GitOps for testing (git operations only, no storage).
type mockGitOps struct {
	headSHA              string
	headErr              error
	logCommits           []git.Commit
	logErr               error
	firstParentCommits   []git.Commit // returned by LogFirstParent; falls back to logCommits when nil
	firstParentErr       error        // returned by LogFirstParent; falls back to logErr when nil
	firstParentCalled    bool         // true if LogFirstParent was called (asserts gate path)
	reachableFrom        []git.Commit
	reachableErr         error
	isAncestor           bool
	anchorOffFirstParent bool                // opt-in: when true, IsOnFirstParentLine returns false
	commitFiles          map[string][]string // SHA -> files; nil map = unknown (no filtering)
}

func newMockGitOps() *mockGitOps {
	return &mockGitOps{isAncestor: true}
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

// LogFirstParent simulates first-parent traversal. By default it mirrors
// Log (handy for tests that don't care about the merge case); tests that
// need to distinguish the two paths set firstParentCommits/firstParentErr
// explicitly.
func (m *mockGitOps) LogFirstParent(fromRef, toRef string) ([]git.Commit, error) {
	m.firstParentCalled = true
	if m.firstParentErr != nil {
		return nil, m.firstParentErr
	}
	if m.firstParentCommits != nil {
		return m.firstParentCommits, nil
	}
	// Fall back to Log's mock data so the most common case ("first-parent
	// is the same as Log for these inputs") doesn't need duplicated setup.
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

func (m *mockGitOps) IsAncestorOf(ancestor, descendant string) bool {
	return m.isAncestor
}

func (m *mockGitOps) IsOnFirstParentLine(sha, head string) bool {
	// Default true unless the test opts into the "off the line" case
	// — keeps existing tests unchanged while letting new tests exercise
	// the Laura pathology directly.
	if m.anchorOffFirstParent {
		return false
	}
	return true
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

func (m *mockGitOps) CommitFilesMulti(shas []string) (map[string][]string, error) {
	result := make(map[string][]string, len(shas))
	for _, sha := range shas {
		files, err := m.CommitFiles(sha)
		if err != nil {
			return nil, err
		}
		result[sha] = files
	}
	return result, nil
}

func (m *mockGitOps) DiffNameOnly(fromRef, toRef, pathPrefix string) ([]string, error) {
	return nil, nil
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
//
// Returned Storage has provenance DISABLED (NewStorage doesn't auto-load it;
// only NewDefaultStorage does). Tests that exercise the provenance classifier
// call store.SetProvenance with explicit config.
func newTestStorage(t *testing.T, mock *mockGitOps, entries ...*Entry) *Storage {
	t.Helper()
	dir := t.TempDir()
	for _, entry := range entries {
		writeTestEntryFile(t, dir, entry)
	}
	files := NewFileStorage(dir, noopGitAdd, noopGitCommit)
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

// --- HasPendingCommits Tests ---

func TestHasPendingCommits(t *testing.T) {
	tests := []struct {
		name      string
		entries   []*Entry
		setupMock func(*mockGitOps)
		want      bool
		wantErr   bool
	}{
		{
			name: "HEAD equals anchor - no pending",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "anchorsha12"
				// Log(anchor, head) returns empty when they're equal
			},
			want: false,
		},
		{
			name: "HEAD differs from anchor - code commit pending",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "newheadsha1"
				mock.logCommits = []git.Commit{{SHA: "newheadsha1"}}
				mock.commitFiles = map[string][]string{
					"newheadsha1": {"main.go"},
				}
			},
			want: true,
		},
		{
			name: "HEAD differs from anchor - ledger-only commit not pending",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "ledgersha12"
				mock.logCommits = []git.Commit{{SHA: "ledgersha12"}}
				mock.commitFiles = map[string][]string{
					"ledgersha12": {".timbers/2026/01/entry.json"},
				}
			},
			want: false,
		},
		{
			name:    "no entries - not pending (fresh repos never block)",
			entries: nil,
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "anysha12345"
				mock.reachableFrom = []git.Commit{{SHA: "anysha12345"}}
				mock.commitFiles = map[string][]string{
					"anysha12345": {"main.go"},
				}
			},
			want: false,
		},
		{
			name:    "HEAD error - returns error",
			entries: []*Entry{makeTestEntry("anchor12345", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))},
			setupMock: func(mock *mockGitOps) {
				mock.headErr = output.NewSystemError("git failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			tt.setupMock(mock)
			store := newTestStorage(t, mock, tt.entries...)

			got, err := store.HasPendingCommits()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("HasPendingCommits() = %v, want %v", got, tt.want)
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

// --- isInfrastructureOnlyCommit Tests ---

func TestIsInfrastructureOnlyCommit(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  bool
	}{
		{
			name:  "empty list is not infrastructure-only",
			files: nil,
			want:  false,
		},
		{
			name:  "all .timbers/ files is infrastructure-only",
			files: []string{".timbers/2026/01/15/tb_entry.json"},
			want:  true,
		},
		{
			name:  "multiple .timbers/ files is infrastructure-only",
			files: []string{".timbers/2026/01/15/tb_a.json", ".timbers/2026/01/15/tb_b.json"},
			want:  true,
		},
		{
			name:  "all .beads/ files is infrastructure-only",
			files: []string{".beads/issues.jsonl"},
			want:  true,
		},
		{
			name:  "mixed .timbers/ and .beads/ is infrastructure-only",
			files: []string{".timbers/2026/01/15/tb_a.json", ".beads/issues.jsonl"},
			want:  true,
		},
		{
			name:  "mixed infrastructure and code is not infrastructure-only",
			files: []string{".timbers/2026/01/15/tb_a.json", "cmd/main.go"},
			want:  false,
		},
		{
			name:  "only code files is not infrastructure-only",
			files: []string{"cmd/main.go", "internal/ledger/storage.go"},
			want:  false,
		},
		{
			name:  ".beads/ plus code is not infrastructure-only",
			files: []string{".beads/issues.jsonl", "AGENTS.md"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInfrastructureOnlyCommit(compiledDefaultSkipRules, tt.files)
			if got != tt.want {
				t.Errorf("isInfrastructureOnlyCommit(%v) = %v, want %v", tt.files, got, tt.want)
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

// --- Revert filtering integration ---

func TestGetPendingCommits_FiltersDocumentedReverts(t *testing.T) {
	originalSHA := "a4e80a72c11d2a9d8f2c1a3b4d5e6f7081234567"
	anchorSHA := "anchorsha12"
	revertSHA := "rev1234abcd"
	realSHA := "realchange1"

	// Anchor entry that documents originalSHA in its workset.
	anchor := makeTestEntry(anchorSHA, time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	anchor.Workset.Commits = []string{anchorSHA, originalSHA}

	mock := newMockGitOps()
	mock.headSHA = "headsha1234"
	mock.logCommits = []git.Commit{
		{
			SHA:     revertSHA,
			Subject: `Revert "feat: original"`,
			Body:    "This reverts commit " + originalSHA + ".",
		},
		{
			SHA:     realSHA,
			Subject: "feat: another change",
		},
	}
	mock.commitFiles = map[string][]string{
		revertSHA: {"src/main.go"},
		realSHA:   {"src/other.go"},
	}

	store := newTestStorage(t, mock, anchor)

	commits, _, err := store.GetPendingCommits()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 pending commit (revert filtered), got %d: %+v", len(commits), commits)
	}
	if commits[0].SHA != realSHA {
		t.Errorf("expected real change to remain, got SHA %q", commits[0].SHA)
	}
}

func TestGetPendingCommits_KeepsUndocumentedReverts(t *testing.T) {
	anchorSHA := "anchorsha12"
	revertSHA := "rev1234abcd"

	anchor := makeTestEntry(anchorSHA, time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	anchor.Workset.Commits = []string{anchorSHA}

	mock := newMockGitOps()
	mock.headSHA = "headsha1234"
	mock.logCommits = []git.Commit{
		{
			SHA:     revertSHA,
			Subject: `Revert "feat: original"`,
			Body:    "This reverts commit deadbeef00000000000000000000000000000000.",
		},
	}
	mock.commitFiles = map[string][]string{
		revertSHA: {"src/main.go"},
	}

	store := newTestStorage(t, mock, anchor)

	commits, _, err := store.GetPendingCommits()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 pending commit (undocumented revert kept), got %d", len(commits))
	}
}

// --- CountInfraSkippedSinceLatest Tests ---

func TestCountInfraSkippedSinceLatest(t *testing.T) {
	tests := []struct {
		name         string
		entries      []*Entry
		setupMock    func(*mockGitOps)
		skipAuthors  []string
		skipMessages []string
		want         int
	}{
		{
			name:    "no entries returns zero",
			entries: nil,
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
			},
			want: 0,
		},
		{
			name: "counts only infrastructure-only commits",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.logCommits = []git.Commit{
					{SHA: "infra1"},
					{SHA: "real1"},
					{SHA: "infra2"},
				}
				mock.commitFiles = map[string][]string{
					"infra1": {".timbers/2026/foo.json"},
					"real1":  {"cmd/main.go"},
					"infra2": {".gitignore"},
				}
			},
			want: 2,
		},
		{
			name: "counts commits matched by msg: rule",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.logCommits = []git.Commit{
					{SHA: "changelog01", Subject: "chore: changelog for v0.22.7"},
					{SHA: "realcommit1", Subject: "feat: add widget"},
					{SHA: "changelog02", Subject: "chore: changelog for v0.22.8"},
				}
				mock.commitFiles = map[string][]string{
					"changelog01": {"CHANGELOG.md"},
					"realcommit1": {"cmd/main.go"},
					"changelog02": {"CHANGELOG.md"},
				}
			},
			skipMessages: []string{"chore: changelog for v*"},
			want:         2,
		},
		{
			name: "counts commits matched by author: rule",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.logCommits = []git.Commit{
					{SHA: "botcommit01", AuthorEmail: "dependabot[bot]@users.noreply.github.com"},
					{SHA: "humancommit", AuthorEmail: "human@example.com"},
				}
				mock.commitFiles = map[string][]string{
					"botcommit01": {"go.mod"},
					"humancommit": {"cmd/main.go"},
				}
			},
			skipAuthors: []string{"dependabot*"},
			want:        1,
		},
		{
			name: "stale anchor returns zero (not actionable)",
			entries: []*Entry{
				makeTestEntry("staleanchor", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.isAncestor = false
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			tt.setupMock(mock)
			store := newTestStorage(t, mock, tt.entries...)
			store.skipAuthors = tt.skipAuthors
			store.skipMessages = tt.skipMessages

			got, err := store.CountInfraSkippedSinceLatest()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("count = %d, want %d", got, tt.want)
			}
		})
	}
}

// Ensure our mock satisfies the interface (compile-time check).
var _ GitOps = (*mockGitOps)(nil)

// --- Gate-specific pending tests ---

// TestHasPendingCommits_UsesFirstParentPath confirms that the gate path
// (HasPendingCommits -> GetGatePendingCommits) consults LogFirstParent
// rather than Log. The test wires the two paths to different results so
// any regression that re-routes the gate through the full DAG walk fails
// loudly.
func TestHasPendingCommits_UsesFirstParentPath(t *testing.T) {
	tests := []struct {
		name               string
		fullDAGCommits     []git.Commit
		firstParentCommits []git.Commit
		commitFiles        map[string][]string
		wantPending        bool
	}{
		{
			name: "sibling-merge case: code on side branch is invisible to gate",
			fullDAGCommits: []git.Commit{
				{SHA: "sidecommit1", Short: "side1"}, // brought in via merge
				{SHA: "sidecommit2", Short: "side2"}, // brought in via merge
			},
			firstParentCommits: []git.Commit{}, // first-parent line is empty
			commitFiles: map[string][]string{
				"sidecommit1": {"frontend/app.tsx"},
				"sidecommit2": {"frontend/lib.ts"},
			},
			wantPending: false,
		},
		{
			name: "in-branch debt still blocks the gate",
			fullDAGCommits: []git.Commit{
				{SHA: "mycommit01", Short: "mine"},
			},
			firstParentCommits: []git.Commit{
				{SHA: "mycommit01", Short: "mine"},
			},
			commitFiles: map[string][]string{
				"mycommit01": {"cmd/feature.go"},
			},
			wantPending: true,
		},
		{
			name: "merge commit that itself touches files on first-parent line blocks the gate",
			fullDAGCommits: []git.Commit{
				{SHA: "mergecmt01", Short: "merge"},
				{SHA: "sidecommit", Short: "side"},
			},
			firstParentCommits: []git.Commit{
				{SHA: "mergecmt01", Short: "merge"},
			},
			commitFiles: map[string][]string{
				// A merge whose own diff against the first parent is non-empty —
				// typically a conflict resolution that introduced source changes.
				// dropEmptyFileChanges keeps it; the gate fires correctly.
				"mergecmt01": {"cmd/main.go"},
				"sidecommit": {"frontend/app.tsx"},
			},
			wantPending: true,
		},
	}

	anchor := makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			mock.headSHA = "headsha1234"
			mock.logCommits = tt.fullDAGCommits
			mock.firstParentCommits = tt.firstParentCommits
			mock.commitFiles = tt.commitFiles

			store := newTestStorage(t, mock, anchor)

			got, err := store.HasPendingCommits()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantPending {
				t.Errorf("HasPendingCommits() = %v, want %v", got, tt.wantPending)
			}
			if !mock.firstParentCalled {
				t.Error("HasPendingCommits should route through LogFirstParent, but it was not called")
			}
		})
	}
}

// TestGetPendingCommits_StillUsesFullDAG confirms that the display path
// (GetPendingCommits) keeps the full DAG view so total documentation debt
// stays visible to `timbers pending` even when first-parent would hide it.
func TestGetPendingCommits_StillUsesFullDAG(t *testing.T) {
	mock := newMockGitOps()
	mock.headSHA = "headsha1234"
	mock.logCommits = []git.Commit{
		{SHA: "sidecommit1", Short: "side1"},
		{SHA: "sidecommit2", Short: "side2"},
	}
	mock.firstParentCommits = []git.Commit{} // gate would say zero
	mock.commitFiles = map[string][]string{
		"sidecommit1": {"frontend/app.tsx"},
		"sidecommit2": {"frontend/lib.ts"},
	}

	anchor := makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := newTestStorage(t, mock, anchor)

	commits, _, err := store.GetPendingCommits()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 2 {
		t.Errorf("display path should show both commits, got %d", len(commits))
	}
	if mock.firstParentCalled {
		t.Error("GetPendingCommits must not use LogFirstParent (would hide debt from displays)")
	}
}

// TestFilterCommits_DebugTrace verifies that TIMBERS_DEBUG=1 produces a
// per-commit classification trace on stderr. Format is intentionally
// stable so downstream automation can parse it.
func TestFilterCommits_DebugTrace(t *testing.T) {
	var buf bytes.Buffer
	origWriter := debugWriter
	debugWriter = func() io.Writer { return &buf }
	defer func() { debugWriter = origWriter }()

	t.Setenv("TIMBERS_DEBUG", "1")

	mock := newMockGitOps()
	mock.headSHA = "headsha1234"
	mock.logCommits = []git.Commit{
		{SHA: "keepme00000", Short: "keepme", ParentCount: 1},
		{SHA: "infraonly00", Short: "infra", ParentCount: 1},
		{SHA: "cleanmerge0", Short: "merge", ParentCount: 2},
	}
	mock.commitFiles = map[string][]string{
		"keepme00000": {"cmd/main.go"},
		"infraonly00": {".beads/issues.jsonl"},
		"cleanmerge0": nil,
	}

	anchor := makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := newTestStorage(t, mock, anchor)

	if _, _, err := store.GetPendingCommits(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	wantSubstrs := []string{
		"[timbers] debug: path=display",
		"keepme keep",
		"infra skip infra",
		"merge skip merge-empty",
		"dropped=infra:1,merge-empty:1",
	}
	for _, want := range wantSubstrs {
		if !strings.Contains(out, want) {
			t.Errorf("debug output missing %q\nfull output:\n%s", want, out)
		}
	}
}

// TestFilterCommits_DebugDisabled confirms that no trace output is
// emitted when TIMBERS_DEBUG is unset.
func TestFilterCommits_DebugDisabled(t *testing.T) {
	var buf bytes.Buffer
	origWriter := debugWriter
	debugWriter = func() io.Writer { return &buf }
	defer func() { debugWriter = origWriter }()

	t.Setenv("TIMBERS_DEBUG", "")

	mock := newMockGitOps()
	mock.headSHA = "headsha1234"
	mock.logCommits = []git.Commit{
		{SHA: "keepme00000", Short: "keepme", ParentCount: 1},
	}
	mock.commitFiles = map[string][]string{
		"keepme00000": {"cmd/main.go"},
	}
	anchor := makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := newTestStorage(t, mock, anchor)

	if _, _, err := store.GetPendingCommits(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no debug output when env unset, got: %s", buf.String())
	}
}

// TestGetGatePendingCommits_OffFirstParentFallback verifies the gate
// fallback fix (timbers-lyp): when the latest entry's anchor is
// reachable from HEAD via merge but NOT on HEAD's first-parent line,
// the gate path (firstParent=true) must route through the
// all-reachable + docSet path instead of running LogFirstParent against
// a structurally weird range.
//
// This was Laura's actual gate-blocking failure: a merged-in side-branch
// entry won the latest-by-timestamp race, the gate walked
// LogFirstParent(side-branch-anchor, HEAD), and the result included
// commits from main's first-parent line that confused the user even
// when docSet ultimately filtered them.
func TestGetGatePendingCommits_OffFirstParentFallback(t *testing.T) {
	mock := newMockGitOps()
	mock.headSHA = "headsha1234"
	mock.isAncestor = true           // anchor reachable via merge
	mock.anchorOffFirstParent = true // but NOT on first-parent line
	mock.reachableFrom = []git.Commit{
		{SHA: "ownworkcom1", Short: "own", ParentCount: 1},  // user's own work, in docSet
		{SHA: "sidecommit1", Short: "side", ParentCount: 1}, // side-branch, in docSet from PR entry
	}
	mock.commitFiles = map[string][]string{
		"ownworkcom1": {"cmd/main.go"},
		"sidecommit1": {"README.md"},
	}
	// firstParentCommits is set non-empty to detect routing mistakes:
	// if the gate path incorrectly uses LogFirstParent, we'd see this
	// instead of the reachable-from fallback. With the fix, LogFirstParent
	// should NOT be called in this scenario.
	mock.firstParentCommits = []git.Commit{
		{SHA: "wrongcommit", Short: "wrong", ParentCount: 1},
	}

	// Anchor entry covers the side-branch commit; the older entry covers
	// the user's own work. docSet (union of both entries' workset.commits)
	// should fully cover the two reachable commits — gate result must be 0.
	anchorEntry := makeTestEntry("sidebranch1", time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC))
	anchorEntry.Workset.Commits = []string{"sidebranch1", "sidecommit1"}
	ownEntry := makeTestEntry("ownworkcom1", time.Date(2026, 5, 21, 9, 0, 0, 0, time.UTC))
	ownEntry.Workset.Commits = []string{"ownworkcom1"}
	store := newTestStorage(t, mock, anchorEntry, ownEntry)

	commits, _, err := store.GetGatePendingCommits()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both candidates are in docSet (anchorEntry covers sidebranch1 +
	// ownEntry covers ownworkcom1), so the gate should see ZERO actionable
	// pending — which is the correct behavior for Laura's case after fix.
	if len(commits) != 0 {
		t.Errorf("expected 0 gate pending (all covered), got %d: %+v", len(commits), commits)
	}
	if mock.firstParentCalled {
		t.Error("gate must NOT call LogFirstParent when anchor is off-first-parent — should use all-reachable fallback")
	}
}

// TestGetGatePendingCommits_OnFirstParentUsesLogFirstParent guards the
// normal case from regression: when the anchor IS on the first-parent
// line, the gate must still use LogFirstParent (not the fallback).
func TestGetGatePendingCommits_OnFirstParentUsesLogFirstParent(t *testing.T) {
	mock := newMockGitOps()
	mock.headSHA = "headsha1234"
	mock.isAncestor = true
	mock.anchorOffFirstParent = false // healthy case
	mock.logCommits = []git.Commit{
		{SHA: "newcommit01", Short: "new", ParentCount: 1},
	}
	mock.firstParentCommits = []git.Commit{
		{SHA: "newcommit01", Short: "new", ParentCount: 1},
	}
	mock.commitFiles = map[string][]string{
		"newcommit01": {"cmd/feature.go"},
	}

	anchor := makeTestEntry("anchorsha12", time.Date(2026, 5, 21, 9, 0, 0, 0, time.UTC))
	store := newTestStorage(t, mock, anchor)

	commits, _, err := store.GetGatePendingCommits()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 1 {
		t.Errorf("expected 1 gate pending, got %d", len(commits))
	}
	if !mock.firstParentCalled {
		t.Error("gate must call LogFirstParent in the normal (on-first-parent-line) case")
	}
}

// TestLatestAnchorOffFirstParent verifies the diagnostic helper that
// surfaces the Laura pathology (latest entry's anchor on a merged-in
// side branch). Returns the negative case (anchor IS on the first-parent
// line) by default; flipping the mock's anchorOffFirstParent toggle
// simulates a side-branch anchor.
func TestLatestAnchorOffFirstParent(t *testing.T) {
	tests := []struct {
		name      string
		entries   []*Entry
		setupMock func(*mockGitOps)
		wantOff   bool
		wantNil   bool
	}{
		{
			name:    "no entries returns false",
			entries: nil,
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
			},
			wantOff: false,
			wantNil: true,
		},
		{
			name: "anchor on first-parent line: not off",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.isAncestor = true
				// anchorOffFirstParent defaults to false → IsOnFirstParentLine returns true
			},
			wantOff: false,
		},
		{
			name: "anchor on side branch but reachable: off (Laura pathology)",
			entries: []*Entry{
				makeTestEntry("sidebranch1", time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.isAncestor = true // reachable via merge
				mock.anchorOffFirstParent = true
			},
			wantOff: true,
		},
		{
			name: "stale anchor: not off (different signal)",
			entries: []*Entry{
				makeTestEntry("staleanchor", time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headSHA = "headsha1234"
				mock.isAncestor = false // anchor missing from history entirely
				mock.anchorOffFirstParent = true
			},
			wantOff: false,
		},
		{
			name: "HEAD error: degrades to false (best-effort diagnostic)",
			entries: []*Entry{
				makeTestEntry("anchorsha12", time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)),
			},
			setupMock: func(mock *mockGitOps) {
				mock.headErr = output.NewSystemError("git failed")
			},
			wantOff: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			tt.setupMock(mock)
			store := newTestStorage(t, mock, tt.entries...)

			off, latest, err := store.LatestAnchorOffFirstParent()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if off != tt.wantOff {
				t.Errorf("LatestAnchorOffFirstParent off = %v, want %v", off, tt.wantOff)
			}
			if tt.wantNil && latest != nil {
				t.Errorf("expected nil latest, got %v", latest.ID)
			}
		})
	}
}

// TestGetPendingCommits_SkipsAckedSHAs confirms that acked SHAs are
// dropped from pending. The roundtrip: WriteAck → AckedSet → filterByRules
// → commit dropped from GetPendingCommits.
func TestGetPendingCommits_SkipsAckedSHAs(t *testing.T) {
	mock := newMockGitOps()
	mock.headSHA = "headsha1234"
	mock.logCommits = []git.Commit{
		{SHA: "humancommit", Short: "human", ParentCount: 1},
		{SHA: "ackedcommit", Short: "acked", ParentCount: 1},
	}
	mock.commitFiles = map[string][]string{
		"humancommit": {"cmd/main.go"},
		"ackedcommit": {"web/page.tsx"},
	}

	anchor := makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := newTestStorage(t, mock, anchor)

	// Sanity: both commits are pending before any ack lands.
	if commits, _, err := store.GetPendingCommits(); err != nil {
		t.Fatalf("pre-ack pending: %v", err)
	} else if len(commits) != 2 {
		t.Fatalf("expected 2 pending pre-ack, got %d", len(commits))
	}

	// Write an ack for one of them.
	now := time.Now().UTC()
	ack := &Ack{
		Schema:    SchemaVersion,
		Kind:      KindAck,
		ID:        GenerateAckID("ackedcommit", now),
		AckedAt:   now,
		Acker:     Acker{Name: "Test", Email: "test@example.com"},
		TargetSHA: "ackedcommit",
		Reason:    "Upstream sync; checked, no entry needed",
	}
	if err := store.WriteAck(ack); err != nil {
		t.Fatalf("WriteAck: %v", err)
	}

	// After the ack, only the human commit should remain pending.
	commits, _, err := store.GetPendingCommits()
	if err != nil {
		t.Fatalf("post-ack pending: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 pending post-ack, got %d", len(commits))
	}
	if commits[0].SHA != "humancommit" {
		t.Errorf("commits[0].SHA = %q, want humancommit", commits[0].SHA)
	}
}

// TestGetPendingCommits_SkipsAuthorGlobMatches confirms that commits whose
// author matches a configured skip-author glob are dropped from pending
// (both gate and display paths route through filterByRules). Use case:
// AI auto-review bots opening PRs at a known author identity — operators
// add the bot to .timbers/skip-authors and the bot's commits no longer
// clutter pending.
func TestGetPendingCommits_SkipsAuthorGlobMatches(t *testing.T) {
	mock := newMockGitOps()
	mock.headSHA = "headsha1234"
	mock.logCommits = []git.Commit{
		{SHA: "humancommit", Short: "human", Author: "Alice", AuthorEmail: "alice@example.com", ParentCount: 1},
		{SHA: "botcommit01", Short: "bot1", Author: "q-redshifted", AuthorEmail: "noreply@anthropic.com", ParentCount: 1},
		{SHA: "botcommit02", Short: "bot2", Author: "argocd", AuthorEmail: "argo@bot.example.com", ParentCount: 1},
	}
	mock.commitFiles = map[string][]string{
		"humancommit": {"cmd/main.go"},
		"botcommit01": {"web/page.tsx"},
		"botcommit02": {"k8s/deployment.yml"},
	}

	anchor := makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := newTestStorage(t, mock, anchor)
	// Inject skip-authors directly (real loading is tested in
	// skipauthors_test.go; this verifies the storage-level wiring).
	store.skipAuthors = []string{"q-redshifted", "*@bot.example.com"}

	commits, _, err := store.GetPendingCommits()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("got %d commits, want 1 (only the human commit)", len(commits))
	}
	if commits[0].SHA != "humancommit" {
		t.Errorf("commits[0].SHA = %q, want humancommit", commits[0].SHA)
	}
}

// TestGetPendingCommits_DropsEmptyMerges confirms that display output no
// longer surfaces clean PR merges (the osprey-strike "Merge pull request
// #N" noise). A merge commit with ParentCount>=2 and an empty file list
// is dropped from display; a merge with file changes (conflict resolution
// touched code) is kept; a single-parent commit with an empty file list
// (--allow-empty) is also kept because the user may want to see it.
func TestGetPendingCommits_DropsEmptyMerges(t *testing.T) {
	tests := []struct {
		name        string
		commits     []git.Commit
		commitFiles map[string][]string
		wantSHAs    []string
	}{
		{
			name: "clean merge dropped, single-parent code kept",
			commits: []git.Commit{
				{SHA: "cleanmerge1", Short: "merge", ParentCount: 2},
				{SHA: "realcommit1", Short: "real", ParentCount: 1},
			},
			commitFiles: map[string][]string{
				"cleanmerge1": nil, // empty combined diff
				"realcommit1": {"cmd/main.go"},
			},
			wantSHAs: []string{"realcommit1"},
		},
		{
			name: "merge with conflict-resolution file changes kept",
			commits: []git.Commit{
				{SHA: "resolvemrg1", Short: "resol", ParentCount: 2},
			},
			commitFiles: map[string][]string{
				"resolvemrg1": {"src/conflict.go"},
			},
			wantSHAs: []string{"resolvemrg1"},
		},
		{
			name: "single-parent empty commit kept in display (intentional --allow-empty)",
			commits: []git.Commit{
				{SHA: "markercmt01", Short: "marker", ParentCount: 1},
			},
			commitFiles: map[string][]string{
				"markercmt01": nil,
			},
			wantSHAs: []string{"markercmt01"},
		},
		{
			name: "multiple clean merges all dropped",
			commits: []git.Commit{
				{SHA: "mrg203be479", Short: "mrg1", ParentCount: 2},
				{SHA: "mrgeb566051", Short: "mrg2", ParentCount: 2},
				{SHA: "contentcmt1", Short: "code", ParentCount: 1},
			},
			commitFiles: map[string][]string{
				"mrg203be479": nil,
				"mrgeb566051": nil,
				"contentcmt1": {"web/page.tsx"},
			},
			wantSHAs: []string{"contentcmt1"},
		},
	}

	anchor := makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			mock.headSHA = "headsha1234"
			mock.logCommits = tt.commits
			mock.commitFiles = tt.commitFiles
			store := newTestStorage(t, mock, anchor)

			commits, _, err := store.GetPendingCommits()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(commits) != len(tt.wantSHAs) {
				t.Fatalf("got %d commits, want %d (%v)", len(commits), len(tt.wantSHAs), tt.wantSHAs)
			}
			for i, want := range tt.wantSHAs {
				if commits[i].SHA != want {
					t.Errorf("commits[%d].SHA = %q, want %q", i, commits[i].SHA, want)
				}
			}
		})
	}
}
