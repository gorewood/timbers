// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// mockGitOpsForPrime implements ledger.GitOps for testing prime command.
type mockGitOpsForPrime struct {
	head            string
	headErr         error
	commits         []git.Commit
	commitsErr      error
	reachableResult []git.Commit
	reachableErr    error
}

func (m *mockGitOpsForPrime) HEAD() (string, error) {
	return m.head, m.headErr
}

func (m *mockGitOpsForPrime) Log(fromRef, toRef string) ([]git.Commit, error) {
	return m.commits, m.commitsErr
}

func (m *mockGitOpsForPrime) CommitsReachableFrom(sha string) ([]git.Commit, error) {
	return m.reachableResult, m.reachableErr
}

func (m *mockGitOpsForPrime) GetDiffstat(fromRef, toRef string) (git.Diffstat, error) {
	return git.Diffstat{}, nil
}

func (m *mockGitOpsForPrime) CommitFiles(sha string) ([]string, error) { return nil, nil }

func TestPrimeCommand(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	// Helper to write entry files into a temp dir and return FileStorage.
	writeEntries := func(t *testing.T, entries ...*ledger.Entry) *ledger.FileStorage {
		t.Helper()
		dir := t.TempDir()
		for _, entry := range entries {
			data, err := entry.ToJSON()
			if err != nil {
				t.Fatalf("failed to serialize entry: %v", err)
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
		return ledger.NewFileStorage(dir, func(_ string) error { return nil })
	}

	tests := []struct {
		name           string
		mock           *mockGitOpsForPrime
		files          func(t *testing.T) *ledger.FileStorage
		lastN          int
		jsonOutput     bool
		wantContains   []string
		wantNotContain []string
		wantErr        bool
	}{
		{
			name: "no entries - shows pending commits and workflow",
			mock: &mockGitOpsForPrime{
				head: "abc123def456",
				reachableResult: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "Third commit"},
					{SHA: "def456789012", Short: "def4567", Subject: "Second commit"},
				},
			},
			files: nil,
			lastN: 3,
			wantContains: []string{
				"Timbers Session Context", "2 undocumented", "(no entries)",
				"CRITICAL: Session Protocol", "Core Rules", "Writing Good Why Fields",
			},
		},
		{
			name: "has entries and pending",
			mock: &mockGitOpsForPrime{
				head: "abc123def456",
				commits: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "New commit"},
				},
			},
			files: func(t *testing.T) *ledger.FileStorage {
				return writeEntries(t,
					makePrimeTestEntry("oldanchor1234", oneHourAgo, "Fixed bug"),
					makePrimeTestEntry("oldanchor5678", twoHoursAgo, "Added feature"),
				)
			},
			lastN: 3,
			wantContains: []string{
				"Timbers Session Context", "Entries: 2", "1 undocumented commit",
				"Fixed bug", "Added feature", "Essential Commands", "commit code FIRST",
			},
		},
		{
			name: "no pending commits",
			mock: &mockGitOpsForPrime{
				head:    "abc123def456",
				commits: []git.Commit{},
			},
			files: func(t *testing.T) *ledger.FileStorage {
				return writeEntries(t, makePrimeTestEntry("abc123def456", now, "Latest work"))
			},
			lastN:        3,
			wantContains: []string{"all work documented", "Latest work", "CRITICAL: Session Protocol"},
		},
		{
			name: "respects lastN flag",
			mock: &mockGitOpsForPrime{
				head:    "abc123def456",
				commits: []git.Commit{},
			},
			files: func(t *testing.T) *ledger.FileStorage {
				return writeEntries(t,
					makePrimeTestEntry("abc123def456", now, "Latest"),
					makePrimeTestEntry("def456789012", oneHourAgo, "Middle"),
					makePrimeTestEntry("789012345678", twoHoursAgo, "Oldest"),
				)
			},
			lastN:          1,
			wantContains:   []string{"Latest"},
			wantNotContain: []string{"Middle", "Oldest"},
		},
		{
			name: "json output - structured format with workflow",
			mock: &mockGitOpsForPrime{
				head: "abc123def456",
				commits: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "New commit"},
				},
			},
			files: func(t *testing.T) *ledger.FileStorage {
				return writeEntries(t, makePrimeTestEntry("oldanchor1234", oneHourAgo, "Test entry"))
			},
			lastN:        3,
			jsonOutput:   true,
			wantContains: []string{`"entry_count": 1`, `"pending":`, `"count": 1`, `"recent_entries":`, `"workflow":`},
		},
		{
			name: "json output - no entries",
			mock: &mockGitOpsForPrime{
				head: "abc123def456",
				reachableResult: []git.Commit{
					{SHA: "abc123def456", Short: "abc123d", Subject: "First commit"},
				},
			},
			files:        nil,
			lastN:        3,
			jsonOutput:   true,
			wantContains: []string{`"entry_count": 0`, `"recent_entries": []`, `"workflow":`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build FileStorage from entries if provided
			var files *ledger.FileStorage
			if tt.files != nil {
				files = tt.files(t)
			}

			// Create storage with mock and file storage
			storage := ledger.NewStorage(tt.mock, files)

			// Create command
			cmd := newPrimeCmdInternal(storage)

			// Set JSON mode for testing
			if tt.jsonOutput {
				cmd.PersistentFlags().Bool("json", false, "")
				_ = cmd.PersistentFlags().Set("json", "true")
			}

			// Set flags
			if tt.lastN != 3 {
				if err := cmd.Flags().Set("last", string(rune('0'+tt.lastN))); err != nil {
					t.Fatalf("failed to set last flag: %v", err)
				}
			}

			// Capture output
			var buf bytes.Buffer
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

			// Check content that should not appear
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(output, notWant) {
					t.Errorf("output contains unexpected content %q\noutput: %s", notWant, output)
				}
			}

			// For JSON output, verify structure
			if tt.jsonOutput && err == nil {
				var result map[string]any
				if parseErr := json.Unmarshal([]byte(output), &result); parseErr != nil {
					t.Errorf("failed to parse JSON output: %v\noutput: %s", parseErr, output)
				}
				// Verify required fields exist
				requiredFields := []string{
					"repo", "branch", "head", "timbers_dir",
					"entry_count", "pending", "recent_entries", "workflow",
				}
				for _, field := range requiredFields {
					if _, ok := result[field]; !ok {
						t.Errorf("JSON missing required field %q", field)
					}
				}
			}
		})
	}
}

// makePrimeTestEntry creates a minimal valid entry for testing.
func makePrimeTestEntry(anchor string, created time.Time, what string) *ledger.Entry {
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
		Summary: ledger.Summary{
			What: what,
			Why:  "For testing",
			How:  "Via test",
		},
	}
}

func TestPrimeResultJSON(t *testing.T) {
	// Test that primeResult serializes correctly
	result := &primeResult{
		Repo:       "test-repo",
		Branch:     "main",
		Head:       "abc123def456",
		TimbersDir: "/tmp/test-repo/.timbers",
		EntryCount: 2,
		Pending: primePending{
			Count: 1,
			Commits: []commitSummary{
				{SHA: "abc123def456", Short: "abc123d", Subject: "Test commit"},
			},
		},
		RecentEntries: []primeEntry{
			{ID: "tb_2026-01-15T10:00:00Z_abc123", What: "Test entry", CreatedAt: "2026-01-15T10:00:00Z"},
		},
		Workflow: "# Test Workflow",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal primeResult: %v", err)
	}

	// Unmarshal and verify
	var unmarshaled map[string]any
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal primeResult: %v", err)
	}

	// Verify structure
	if unmarshaled["repo"] != "test-repo" {
		t.Errorf("repo = %v, want test-repo", unmarshaled["repo"])
	}
	if unmarshaled["branch"] != "main" {
		t.Errorf("branch = %v, want main", unmarshaled["branch"])
	}
	entryCount, ok := unmarshaled["entry_count"].(float64)
	if !ok || entryCount != 2 {
		t.Errorf("entry_count = %v, want 2", unmarshaled["entry_count"])
	}

	pending, ok := unmarshaled["pending"].(map[string]any)
	if !ok {
		t.Fatalf("pending is not a map")
	}
	pendingCount, ok := pending["count"].(float64)
	if !ok || pendingCount != 1 {
		t.Errorf("pending.count = %v, want 1", pending["count"])
	}

	recentEntries, ok := unmarshaled["recent_entries"].([]any)
	if !ok {
		t.Fatalf("recent_entries is not an array")
	}
	if len(recentEntries) != 1 {
		t.Errorf("recent_entries length = %v, want 1", len(recentEntries))
	}

	workflow, ok := unmarshaled["workflow"].(string)
	if !ok {
		t.Fatalf("workflow is not a string")
	}
	if workflow != "# Test Workflow" {
		t.Errorf("workflow = %v, want # Test Workflow", workflow)
	}
}

func TestPrimeExportFlag(t *testing.T) {
	cmd := newPrimeCmdInternal(nil)

	// Set export flag
	if err := cmd.Flags().Set("export", "true"); err != nil {
		t.Fatalf("failed to set export flag: %v", err)
	}

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Execute
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Verify it contains default workflow content
	expectedParts := []string{
		"CRITICAL: Session Protocol",
		"Core Rules",
		"Essential Commands",
		"timbers pending",
		"timbers log",
		"git push",
	}

	for _, want := range expectedParts {
		if !strings.Contains(output, want) {
			t.Errorf("export output missing expected content %q", want)
		}
	}
}

func TestPrimeVerboseFlag(t *testing.T) {
	now := time.Now()

	mock := &mockGitOpsForPrime{
		head:    "abc123def456",
		commits: []git.Commit{},
	}

	entry := makePrimeTestEntry("anchor1234", now, "Fixed auth bug")
	dir := t.TempDir()
	data, _ := entry.ToJSON()
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
	files := ledger.NewFileStorage(dir, func(_ string) error { return nil })
	storage := ledger.NewStorage(mock, files)

	// Without verbose: should show what but not why/how
	cmd := newPrimeCmdInternal(storage)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Fixed auth bug") {
		t.Error("expected 'what' in non-verbose output")
	}
	if strings.Contains(out, "Why:") || strings.Contains(out, "How:") {
		t.Error("expected no why/how in non-verbose output")
	}

	// With verbose: should show why and how
	cmd2 := newPrimeCmdInternal(storage)
	if err := cmd2.Flags().Set("verbose", "true"); err != nil {
		t.Fatalf("failed to set verbose flag: %v", err)
	}
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	out2 := buf2.String()
	if !strings.Contains(out2, "Why: For testing") {
		t.Errorf("expected 'Why: For testing' in verbose output, got: %s", out2)
	}
	if !strings.Contains(out2, "How: Via test") {
		t.Errorf("expected 'How: Via test' in verbose output, got: %s", out2)
	}
}

func TestPrimeSilentInUninitRepo(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", ".")
	runGit(t, tempDir, "commit", "-m", "initial")

	// No timbers init — .timbers/ directory does not exist
	runInDir(t, tempDir, func() {
		cmd := newPrimeCmdInternal(nil)
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("prime should not error in uninitiated repo: %v", err)
		}

		// Should produce no stdout output (silent exit)
		if buf.Len() > 0 {
			t.Errorf("prime should be silent in uninitiated repo, got: %s", buf.String())
		}
	})
}

func TestPrimeVerboseJSON(t *testing.T) {
	now := time.Now()

	mock := &mockGitOpsForPrime{
		head:    "abc123def456",
		commits: []git.Commit{},
	}

	entry := makePrimeTestEntry("anchor1234", now, "Fixed auth bug")
	dir := t.TempDir()
	data, _ := entry.ToJSON()
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
	files := ledger.NewFileStorage(dir, func(_ string) error { return nil })
	storage := ledger.NewStorage(mock, files)

	// JSON with verbose: should include why/how fields
	cmd := newPrimeCmdInternal(storage)
	cmd.PersistentFlags().Bool("json", false, "")
	_ = cmd.PersistentFlags().Set("json", "true")
	if err := cmd.Flags().Set("verbose", "true"); err != nil {
		t.Fatalf("failed to set verbose flag: %v", err)
	}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var result primeResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, buf.String())
	}
	if len(result.RecentEntries) == 0 {
		t.Fatal("expected at least one recent entry")
	}
	recentEntry := result.RecentEntries[0]
	if recentEntry.Why != "For testing" {
		t.Errorf("why = %q, want %q", recentEntry.Why, "For testing")
	}
	if recentEntry.How != "Via test" {
		t.Errorf("how = %q, want %q", recentEntry.How, "Via test")
	}

	// JSON without verbose: why/how should be empty (omitted)
	cmd2 := newPrimeCmdInternal(storage)
	cmd2.PersistentFlags().Bool("json", false, "")
	_ = cmd2.PersistentFlags().Set("json", "true")
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	out2 := buf2.String()
	if strings.Contains(out2, `"why"`) || strings.Contains(out2, `"how"`) {
		t.Errorf("expected no why/how in non-verbose JSON, got: %s", out2)
	}
}

func TestTruncateNotes(t *testing.T) {
	tests := []struct {
		name   string
		notes  string
		maxLen int
		want   string
	}{
		{"empty", "", 200, ""},
		{"under limit", "short note", 200, "short note"},
		{"exactly at limit", "abc", 3, "abc"},
		{"over limit", "abcdef", 3, "abc..."},
		{"long notes truncated", strings.Repeat("x", 300), 200, strings.Repeat("x", 200) + "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateNotes(tt.notes, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateNotes(%q, %d) = %q, want %q", tt.notes, tt.maxLen, got, tt.want)
			}
		})
	}
}
