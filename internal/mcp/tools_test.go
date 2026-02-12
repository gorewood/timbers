package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// --- Mock GitOps ---

type mockGitOps struct {
	headSHA       string
	logCommits    []git.Commit
	logErr        error
	reachableFrom []git.Commit
	diffstat      git.Diffstat
	commitFiles   map[string][]string
}

func (m *mockGitOps) HEAD() (string, error) {
	return m.headSHA, nil
}

func (m *mockGitOps) Log(_, _ string) ([]git.Commit, error) {
	return m.logCommits, m.logErr
}

func (m *mockGitOps) CommitsReachableFrom(_ string) ([]git.Commit, error) {
	return m.reachableFrom, nil
}

func (m *mockGitOps) GetDiffstat(_, _ string) (git.Diffstat, error) {
	return m.diffstat, nil
}

func (m *mockGitOps) CommitFiles(sha string) ([]string, error) {
	if m.commitFiles == nil {
		return nil, nil
	}
	return m.commitFiles[sha], nil
}

// --- Test helpers ---

func makeTestStorage(t *testing.T, gitOps *mockGitOps, entries []*ledger.Entry) *ledger.Storage {
	t.Helper()
	tmpDir := t.TempDir()
	fileStore := ledger.NewFileStorage(tmpDir, noopGitAdd)
	for _, entry := range entries {
		if err := fileStore.WriteEntry(entry, false); err != nil {
			t.Fatalf("writing test entry: %v", err)
		}
	}
	return ledger.NewStorage(gitOps, fileStore)
}

func noopGitAdd(_ string) error { return nil }

func makeEntry(anchor, what, why, how string, created time.Time, tags []string) *ledger.Entry {
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
		Summary: ledger.Summary{What: what, Why: why, How: how},
		Tags:    tags,
	}
}

// --- Pending handler tests ---

func TestHandlePending_NoEntries(t *testing.T) {
	gitOps := &mockGitOps{
		headSHA: "abc123",
		reachableFrom: []git.Commit{
			{SHA: "abc123", Short: "abc123", Subject: "initial commit"},
		},
	}
	storage := makeTestStorage(t, gitOps, nil)
	handler := handlePending(storage)

	_, out, err := handler(context.Background(), &mcp.CallToolRequest{}, PendingInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Count != 1 {
		t.Errorf("Count = %d, want 1", out.Count)
	}
	if len(out.Commits) != 1 {
		t.Errorf("len(Commits) = %d, want 1", len(out.Commits))
	}
}

func TestHandlePending_AllDocumented(t *testing.T) {
	now := time.Now().UTC()
	gitOps := &mockGitOps{
		headSHA:    "abc123",
		logCommits: []git.Commit{}, // no commits since anchor
	}
	entries := []*ledger.Entry{
		makeEntry("abc123", "test", "why", "how", now, nil),
	}
	storage := makeTestStorage(t, gitOps, entries)
	handler := handlePending(storage)

	_, out, err := handler(context.Background(), &mcp.CallToolRequest{}, PendingInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Count != 0 {
		t.Errorf("Count = %d, want 0", out.Count)
	}
	if out.LastEntry == nil {
		t.Error("LastEntry is nil, want non-nil")
	}
}

// --- Query handler tests ---

func TestHandleQuery_LastN(t *testing.T) {
	now := time.Now().UTC()
	gitOps := &mockGitOps{headSHA: "def456"}
	entries := []*ledger.Entry{
		makeEntry("aaa111", "first", "w1", "h1", now.Add(-3*time.Hour), nil),
		makeEntry("bbb222", "second", "w2", "h2", now.Add(-2*time.Hour), nil),
		makeEntry("ccc333", "third", "w3", "h3", now.Add(-1*time.Hour), nil),
	}
	storage := makeTestStorage(t, gitOps, entries)
	handler := handleQuery(storage)

	_, out, err := handler(context.Background(), &mcp.CallToolRequest{}, QueryInput{Last: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Count != 2 {
		t.Errorf("Count = %d, want 2", out.Count)
	}
	// Should be most recent first
	if out.Entries[0].Summary.What != "third" {
		t.Errorf("first entry What = %q, want %q", out.Entries[0].Summary.What, "third")
	}
}

func TestHandleQuery_NoFilter(t *testing.T) {
	gitOps := &mockGitOps{headSHA: "def456"}
	storage := makeTestStorage(t, gitOps, nil)
	handler := handleQuery(storage)

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, QueryInput{})
	if err == nil {
		t.Error("expected error for no filter, got nil")
	}
}

func TestHandleQuery_WithTags(t *testing.T) {
	now := time.Now().UTC()
	gitOps := &mockGitOps{headSHA: "def456"}
	entries := []*ledger.Entry{
		makeEntry("aaa111", "security fix", "w1", "h1", now.Add(-2*time.Hour), []string{"security"}),
		makeEntry("bbb222", "feature add", "w2", "h2", now.Add(-1*time.Hour), []string{"feature"}),
	}
	storage := makeTestStorage(t, gitOps, entries)
	handler := handleQuery(storage)

	_, out, err := handler(context.Background(), &mcp.CallToolRequest{}, QueryInput{Last: 10, Tags: []string{"security"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Count != 1 {
		t.Errorf("Count = %d, want 1", out.Count)
	}
}

// --- Show handler tests ---

func TestHandleShow_ByID(t *testing.T) {
	now := time.Now().UTC()
	gitOps := &mockGitOps{headSHA: "def456"}
	entry := makeEntry("aaa111", "test entry", "why", "how", now, nil)
	storage := makeTestStorage(t, gitOps, []*ledger.Entry{entry})
	handler := handleShow(storage)

	_, out, err := handler(context.Background(), &mcp.CallToolRequest{}, ShowInput{ID: entry.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Entry.Summary.What != "test entry" {
		t.Errorf("What = %q, want %q", out.Entry.Summary.What, "test entry")
	}
}

func TestHandleShow_Latest(t *testing.T) {
	now := time.Now().UTC()
	gitOps := &mockGitOps{headSHA: "def456"}
	entries := []*ledger.Entry{
		makeEntry("aaa111", "older", "w1", "h1", now.Add(-1*time.Hour), nil),
		makeEntry("bbb222", "newer", "w2", "h2", now, nil),
	}
	storage := makeTestStorage(t, gitOps, entries)
	handler := handleShow(storage)

	_, out, err := handler(context.Background(), &mcp.CallToolRequest{}, ShowInput{Latest: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Entry.Summary.What != "newer" {
		t.Errorf("What = %q, want %q", out.Entry.Summary.What, "newer")
	}
}

func TestHandleShow_NoArgs(t *testing.T) {
	gitOps := &mockGitOps{headSHA: "def456"}
	storage := makeTestStorage(t, gitOps, nil)
	handler := handleShow(storage)

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, ShowInput{})
	if err == nil {
		t.Error("expected error for no args, got nil")
	}
}

func TestHandleShow_BothIDAndLatest(t *testing.T) {
	gitOps := &mockGitOps{headSHA: "def456"}
	storage := makeTestStorage(t, gitOps, nil)
	handler := handleShow(storage)

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, ShowInput{ID: "tb_foo", Latest: true})
	if err == nil {
		t.Error("expected error for both id and latest, got nil")
	}
}

// --- Log handler tests ---

func TestHandleLog_Success(t *testing.T) {
	gitOps := &mockGitOps{
		headSHA: "abc123",
		reachableFrom: []git.Commit{
			{SHA: "abc123", Short: "abc123", Subject: "test commit"},
		},
		diffstat: git.Diffstat{Files: 2, Insertions: 10, Deletions: 3},
	}
	tmpDir := t.TempDir()
	fileStore := ledger.NewFileStorage(tmpDir, noopGitAdd)
	storage := ledger.NewStorage(gitOps, fileStore)
	handler := handleLog(storage)

	_, out, err := handler(context.Background(), &mcp.CallToolRequest{}, LogInput{
		What: "implemented feature",
		Why:  "users needed it",
		How:  "added new module",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Entry == nil {
		t.Fatal("Entry is nil")
	}
	if out.Entry.Summary.What != "implemented feature" {
		t.Errorf("What = %q, want %q", out.Entry.Summary.What, "implemented feature")
	}
	if out.Entry.Workset.AnchorCommit != "abc123" {
		t.Errorf("AnchorCommit = %q, want %q", out.Entry.Workset.AnchorCommit, "abc123")
	}
}

func TestHandleLog_MissingFields(t *testing.T) {
	gitOps := &mockGitOps{headSHA: "abc123"}
	storage := makeTestStorage(t, gitOps, nil)
	handler := handleLog(storage)

	tests := []struct {
		name  string
		input LogInput
	}{
		{"missing what", LogInput{Why: "w", How: "h"}},
		{"missing why", LogInput{What: "w", How: "h"}},
		{"missing how", LogInput{What: "w", Why: "w"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, tt.input)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestHandleLog_NoPendingCommits(t *testing.T) {
	now := time.Now().UTC()
	gitOps := &mockGitOps{
		headSHA:    "abc123",
		logCommits: []git.Commit{}, // no pending
	}
	entries := []*ledger.Entry{
		makeEntry("abc123", "already documented", "w", "h", now, nil),
	}
	storage := makeTestStorage(t, gitOps, entries)
	handler := handleLog(storage)

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, LogInput{
		What: "test", Why: "test", How: "test",
	})
	if err == nil {
		t.Error("expected error for no pending commits, got nil")
	}
}

func TestHandleLog_WithWorkItem(t *testing.T) {
	gitOps := &mockGitOps{
		headSHA: "abc123",
		reachableFrom: []git.Commit{
			{SHA: "abc123", Short: "abc123", Subject: "test commit"},
		},
	}
	tmpDir := t.TempDir()
	fileStore := ledger.NewFileStorage(tmpDir, noopGitAdd)
	storage := ledger.NewStorage(gitOps, fileStore)
	handler := handleLog(storage)

	_, out, err := handler(context.Background(), &mcp.CallToolRequest{}, LogInput{
		What:     "test",
		Why:      "test",
		How:      "test",
		WorkItem: "beads:abc123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Entry.WorkItems) != 1 {
		t.Fatalf("WorkItems len = %d, want 1", len(out.Entry.WorkItems))
	}
	if out.Entry.WorkItems[0].System != "beads" {
		t.Errorf("System = %q, want %q", out.Entry.WorkItems[0].System, "beads")
	}
}

// --- Helper function tests ---

func TestParseDurationOrDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"go duration hours", "24h", false},
		{"go duration minutes", "30m", false},
		{"day duration", "7d", false},
		{"iso date", "2026-01-15", false},
		{"rfc3339", "2026-01-15T10:30:00Z", false},
		{"invalid", "not-a-date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDurationOrDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDurationOrDate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestParseWorkItem(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSystem string
		wantID     string
		wantErr    bool
	}{
		{"valid beads", "beads:abc123", "beads", "abc123", false},
		{"valid jira", "jira:PROJ-456", "jira", "PROJ-456", false},
		{"valid github", "gh:owner/repo#42", "gh", "owner/repo#42", false},
		{"missing colon", "beadsabc123", "", "", true},
		{"empty system", ":abc123", "", "", true},
		{"empty id", "beads:", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workItem, err := parseWorkItem(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseWorkItem(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if workItem.System != tt.wantSystem {
					t.Errorf("System = %q, want %q", workItem.System, tt.wantSystem)
				}
				if workItem.ID != tt.wantID {
					t.Errorf("ID = %q, want %q", workItem.ID, tt.wantID)
				}
			}
		})
	}
}

// --- Server registration test ---

func TestNewServer_RegistersTools(t *testing.T) {
	gitOps := &mockGitOps{headSHA: "abc123"}
	storage := makeTestStorage(t, gitOps, nil)

	// Should not panic
	server := NewServer("test-version", storage)
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
}
