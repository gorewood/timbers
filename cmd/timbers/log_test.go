// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// countJSONFilesInDir walks dir recursively and counts .json files.
func countJSONFilesInDir(dir string) int {
	count := 0
	_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
			count++
		}
		return nil
	})
	return count
}

// walkJSONFiles calls callback for each .json file found recursively under dir.
func walkJSONFiles(dir string, callback func(path string, data []byte)) {
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
			data, _ := os.ReadFile(path)
			callback(path, data)
		}
		return nil
	})
}

// mockGitOpsForLog implements ledger.GitOps for testing log command.
type mockGitOpsForLog struct {
	head            string
	headErr         error
	commits         []git.Commit
	commitsErr      error
	rangeCommits    []git.Commit // returned by Log for single-commit range calls (fromRef ends with "^")
	reachableResult []git.Commit
	reachableErr    error
	diffstat        git.Diffstat
	diffstatErr     error
}

func newMockGitOpsForLog() *mockGitOpsForLog {
	return &mockGitOpsForLog{}
}

func (m *mockGitOpsForLog) HEAD() (string, error) {
	return m.head, m.headErr
}

func (m *mockGitOpsForLog) Log(fromRef, _ string) ([]git.Commit, error) {
	// The --anchor fallback resolves a single commit via LogRange(anchor^, anchor);
	// distinguish that from the pending-range call so tests can set them apart.
	if m.rangeCommits != nil && strings.HasSuffix(fromRef, "^") {
		return m.rangeCommits, nil
	}
	return m.commits, m.commitsErr
}

func (m *mockGitOpsForLog) LogFirstParent(_, _ string) ([]git.Commit, error) {
	return m.commits, m.commitsErr
}

func (m *mockGitOpsForLog) ResolveCommit(ref string) (string, error) {
	return ref, nil
}

func (m *mockGitOpsForLog) CommitsReachableFrom(_ string) ([]git.Commit, error) {
	return m.reachableResult, m.reachableErr
}

func (m *mockGitOpsForLog) IsAncestorOf(ancestor, descendant string) bool {
	return true
}

func (m *mockGitOpsForLog) IsOnFirstParentLine(sha, head string) bool {
	return true
}

func (m *mockGitOpsForLog) GetDiffstat(_, _ string) (git.Diffstat, error) {
	return m.diffstat, m.diffstatErr
}

func (m *mockGitOpsForLog) CommitFiles(sha string) ([]string, error) { return nil, nil }
func (m *mockGitOpsForLog) CommitFilesMulti(shas []string) (map[string][]string, error) {
	return make(map[string][]string), nil
}

func (m *mockGitOpsForLog) DiffNameOnly(fromRef, toRef, pathPrefix string) ([]string, error) {
	return nil, nil
}

// newLogTestStorage creates a Storage with a temp dir for writing entries.
func newLogTestStorage(t *testing.T, mock *mockGitOpsForLog) (*ledger.Storage, string) {
	t.Helper()
	dir := t.TempDir()
	files := ledger.NewFileStorage(dir, func(_ string) error { return nil }, func(_, _ string) error { return nil })
	return ledger.NewStorage(mock, files), dir
}

func TestLogCommand(t *testing.T) {
	tests := []struct {
		name           string
		mock           *mockGitOpsForLog
		args           []string
		jsonOutput     bool
		wantErr        bool
		wantContains   []string
		wantNotContain []string
		checkDir       func(t *testing.T, dir string)
	}{
		{
			name: "successful log with all flags",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
					{SHA: "def456789012345", Short: "def4567", Subject: "Previous commit"},
				}
				mock.diffstat = git.Diffstat{Files: 3, Insertions: 45, Deletions: 12}
				return mock
			}(),
			args:         []string{"Fixed authentication bug", "--why", "Security vulnerability", "--how", "Added input validation"},
			wantErr:      false,
			wantContains: []string{"Created entry"},
			checkDir: func(t *testing.T, dir string) {
				if n := countJSONFilesInDir(dir); n != 1 {
					t.Errorf("expected 1 entry file written, got %d", n)
				}
			},
		},
		{
			name: "successful log with tags",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
				}
				mock.diffstat = git.Diffstat{Files: 1, Insertions: 10, Deletions: 5}
				return mock
			}(),
			args: []string{
				"Added feature", "--why", "User request", "--how", "New component",
				"--tag", "security", "--tag", "auth",
			},
			wantErr:      false,
			wantContains: []string{"Created entry"},
			checkDir: func(t *testing.T, dir string) {
				walkJSONFiles(dir, func(_ string, data []byte) {
					content := string(data)
					if !strings.Contains(content, "security") || !strings.Contains(content, "auth") {
						t.Error("expected tags to be in written entry")
					}
				})
			},
		},
		{
			name: "successful log with work items",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
				}
				mock.diffstat = git.Diffstat{Files: 1, Insertions: 10, Deletions: 5}
				return mock
			}(),
			args: []string{
				"Fixed issue", "--why", "Bug report", "--how", "Patch applied",
				"--work-item", "beads:bd-abc123", "--work-item", "jira:PROJ-456",
			},
			wantErr:      false,
			wantContains: []string{"Created entry"},
			checkDir: func(t *testing.T, dir string) {
				walkJSONFiles(dir, func(_ string, data []byte) {
					content := string(data)
					if !strings.Contains(content, "beads") || !strings.Contains(content, "bd-abc123") {
						t.Error("expected work items to be in written entry")
					}
				})
			},
		},
		{
			name: "minor mode - why and how default to Minor change",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
				}
				mock.diffstat = git.Diffstat{Files: 1, Insertions: 2, Deletions: 1}
				return mock
			}(),
			args:         []string{"Updated README", "--minor"},
			wantErr:      false,
			wantContains: []string{"Created entry"},
			checkDir: func(t *testing.T, dir string) {
				walkJSONFiles(dir, func(_ string, data []byte) {
					content := string(data)
					if !strings.Contains(content, "Minor change") {
						t.Error("expected 'Minor change' to be in written entry for --minor mode")
					}
				})
			},
		},
		{
			name: "dry-run mode - shows entry without writing",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
				}
				mock.diffstat = git.Diffstat{Files: 2, Insertions: 20, Deletions: 5}
				return mock
			}(),
			args:         []string{"Test feature", "--why", "Testing", "--how", "Test code", "--dry-run"},
			wantErr:      false,
			wantContains: []string{"Dry Run Preview", "Test feature", "Testing", "Test code"},
			checkDir: func(t *testing.T, dir string) {
				if n := countJSONFilesInDir(dir); n != 0 {
					t.Errorf("expected no entries written in dry-run mode, got %d", n)
				}
			},
		},
		{
			name: "validation error - missing why flag (not minor)",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
				}
				return mock
			}(),
			args:         []string{"Some change", "--how", "Did it"},
			wantErr:      true,
			wantContains: []string{"--why"},
		},
		{
			name: "validation error - missing how flag (not minor)",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
				}
				return mock
			}(),
			args:         []string{"Some change", "--why", "Because"},
			wantErr:      true,
			wantContains: []string{"--how"},
		},
		{
			name: "validation error - missing what argument",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				return mock
			}(),
			args:         []string{"--why", "Because", "--how", "Did it"},
			wantErr:      true,
			wantContains: []string{"what"},
		},
		{
			name: "validation error - invalid work-item format",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
				}
				return mock
			}(),
			args:         []string{"Fix", "--why", "Bug", "--how", "Patch", "--work-item", "invalid-format"},
			wantErr:      true,
			wantContains: []string{"work-item", "system:id"},
		},
		{
			name: "validation error - invalid range format",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				return mock
			}(),
			args:         []string{"Fix", "--why", "Bug", "--how", "Patch", "--range", "invalid"},
			wantErr:      true,
			wantContains: []string{"range", ".."},
		},
		{
			name: "error - no pending commits",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{} // No commits
				return mock
			}(),
			args:         []string{"Something", "--why", "Because", "--how", "Somehow"},
			wantErr:      true,
			wantContains: []string{"no pending commits"},
		},
		{
			name: "JSON output - success",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
				}
				mock.diffstat = git.Diffstat{Files: 1, Insertions: 5, Deletions: 2}
				return mock
			}(),
			args:         []string{"Feature", "--why", "Need it", "--how", "Built it"},
			jsonOutput:   true,
			wantErr:      false,
			wantContains: []string{`"status": "created"`, `"id":`, `"anchor":`},
		},
		{
			name: "JSON output - dry run",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
				}
				mock.diffstat = git.Diffstat{Files: 1, Insertions: 5, Deletions: 2}
				return mock
			}(),
			args:         []string{"Feature", "--why", "Need it", "--how", "Built it", "--dry-run"},
			jsonOutput:   true,
			wantErr:      false,
			wantContains: []string{`"status": "dry_run"`, `"entry":`},
		},
		{
			name: "explicit range flag",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.commits = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
					{SHA: "def456789012345", Short: "def4567", Subject: "Middle commit"},
				}
				mock.diffstat = git.Diffstat{Files: 2, Insertions: 30, Deletions: 10}
				return mock
			}(),
			args: []string{
				"Range work", "--why", "Grouped", "--how", "All together",
				"--range", "start123..abc123def456789",
			},
			wantErr:      false,
			wantContains: []string{"Created entry"},
		},
		{
			name: "explicit anchor flag",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
				}
				mock.diffstat = git.Diffstat{Files: 1, Insertions: 5, Deletions: 2}
				return mock
			}(),
			args: []string{
				"Custom anchor", "--why", "Specific", "--how", "Override",
				"--anchor", "custom123456",
			},
			wantErr:      false,
			wantContains: []string{"Created entry"},
			checkDir: func(t *testing.T, dir string) {
				walkJSONFiles(dir, func(_ string, data []byte) {
					content := string(data)
					if !strings.Contains(content, "custom123456") {
						t.Error("expected entry written with custom anchor commit")
					}
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create storage with mock and temp dir
			storage, dir := newLogTestStorage(t, tt.mock)

			// Create command
			cmd := newLogCmdWithStorage(storage)

			// Set JSON mode for testing
			if tt.jsonOutput {
				cmd.PersistentFlags().Bool("json", false, "")
				_ = cmd.PersistentFlags().Set("json", "true")
			}

			// Set args
			cmd.SetArgs(tt.args)

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
				if jsonErr := json.Unmarshal([]byte(output), &result); jsonErr != nil {
					t.Errorf("failed to parse JSON output: %v\noutput: %s", jsonErr, output)
				}
			}

			// Run dir checks if provided
			if tt.checkDir != nil && err == nil {
				tt.checkDir(t, dir)
			}
		})
	}
}

func TestParseWorkItem(t *testing.T) {
	tests := []struct {
		input      string
		wantSystem string
		wantID     string
		wantErr    bool
	}{
		{"beads:bd-abc123", "beads", "bd-abc123", false},
		{"jira:PROJ-456", "jira", "PROJ-456", false},
		{"github:123", "github", "123", false},
		{"invalid", "", "", true},
		{"no-colon-here", "", "", true},
		{":empty-system", "", "", true},
		{"empty-id:", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			system, itemID, err := parseWorkItem(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseWorkItem(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if system != tt.wantSystem {
				t.Errorf("parseWorkItem(%q) system = %q, want %q", tt.input, system, tt.wantSystem)
			}
			if itemID != tt.wantID {
				t.Errorf("parseWorkItem(%q) id = %q, want %q", tt.input, itemID, tt.wantID)
			}
		})
	}
}

func TestValidateRangeFormat(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"abc123..def456", false},
		{"HEAD~5..HEAD", false},
		{"start..end", false},
		{"invalid", true},
		{"nodots", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := validateRangeFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRangeFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// newLogCmdWithStorage is a helper for tests that injects a storage.
// Dirty checker defaults to always-clean for unit tests.
func newLogCmdWithStorage(storage *ledger.Storage) *cobra.Command {
	return newLogCmdInternal(storage, func() bool { return false })
}

// TestLogDirtyTreeRefuses confirms that `timbers log` on a dirty tree
// returns a UserError without creating or auto-committing an entry. This
// closes the phantom-entry path where the pre-commit gate aborts a commit,
// the caller follows up with `timbers log` (newline-chained, no &&), and
// the entry's auto-commit (pathspec-scoped to .timbers/...) lands on the
// old HEAD while the staged feature changes stay in the index.
//
// --dry-run is intentionally allowed even on a dirty tree because it
// short-circuits before any write — useful for "what would this entry
// look like?" inspections in the middle of a debugging session.
func TestLogDirtyTreeRefuses(t *testing.T) {
	t.Run("dirty tree refuses with no entry created", func(t *testing.T) {
		mock := newMockGitOpsForLog()
		mock.head = "abc123def456789"
		mock.reachableResult = []git.Commit{
			{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
		}
		mock.diffstat = git.Diffstat{Files: 1, Insertions: 10, Deletions: 0}

		storage, dir := newLogTestStorage(t, mock)
		cmd := newLogCmdInternal(storage, func() bool { return true })
		cmd.SetArgs([]string{"Test entry", "--why", "Testing", "--how", "Via test"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		if err := cmd.Execute(); err == nil {
			t.Fatalf("expected refusal error on dirty tree, got nil; output: %s", buf.String())
		}
		out := buf.String()
		if !strings.Contains(out, "uncommitted changes") {
			t.Errorf("expected 'uncommitted changes' in refusal output, got: %s", out)
		}
		if !strings.Contains(out, "phantom") {
			t.Errorf("expected 'phantom' in refusal output (explains the why), got: %s", out)
		}
		// Critical: no entry file should be created when refusing.
		if n := countJSONFilesInDir(dir); n != 0 {
			t.Errorf("expected no entry files created on refusal, got %d", n)
		}
	})

	t.Run("dirty tree with --dry-run succeeds without writing", func(t *testing.T) {
		mock := newMockGitOpsForLog()
		mock.head = "abc123def456789"
		mock.reachableResult = []git.Commit{
			{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
		}
		mock.diffstat = git.Diffstat{Files: 1, Insertions: 10, Deletions: 0}

		storage, dir := newLogTestStorage(t, mock)
		cmd := newLogCmdInternal(storage, func() bool { return true })
		cmd.SetArgs([]string{"Test entry", "--why", "Testing", "--how", "Via test", "--dry-run"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("dry-run on dirty tree should succeed, got error: %v (output: %s)", err, buf.String())
		}
		if n := countJSONFilesInDir(dir); n != 0 {
			t.Errorf("dry-run must not create entry files, got %d", n)
		}
	})

	t.Run("clean tree succeeds without warning", func(t *testing.T) {
		mock := newMockGitOpsForLog()
		mock.head = "def456789012345"
		mock.reachableResult = []git.Commit{
			{SHA: "def456789012345", Short: "def4567", Subject: "Another commit"},
		}
		mock.diffstat = git.Diffstat{Files: 1, Insertions: 10, Deletions: 0}

		storage, _ := newLogTestStorage(t, mock)
		cmd := newLogCmdInternal(storage, func() bool { return false })
		cmd.SetArgs([]string{"Test entry 2", "--why", "Testing", "--how", "Via test"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("clean tree should not error: %v (output: %s)", err, buf.String())
		}
		out := buf.String()
		if strings.Contains(out, "uncommitted changes") {
			t.Errorf("clean tree should not mention uncommitted changes, got: %s", out)
		}
	})
}

func TestExtractAutoContent(t *testing.T) {
	tests := []struct {
		name     string
		commits  []git.Commit
		wantWhat string
		wantWhy  string
		wantHow  string
	}{
		{
			name: "single commit without body",
			commits: []git.Commit{
				{Subject: "Fix bug in parser"},
			},
			wantWhat: "Fix bug in parser",
			wantWhy:  "Auto-documented",
			wantHow:  "Auto-documented",
		},
		{
			name: "multiple commits without body",
			commits: []git.Commit{
				{Subject: "Add feature X"},
				{Subject: "Fix tests"},
				{Subject: "Update docs"},
			},
			wantWhat: "Add feature X; Fix tests; Update docs",
			wantWhy:  "Auto-documented",
			wantHow:  "Auto-documented",
		},
		{
			name: "single commit with one-paragraph body",
			commits: []git.Commit{
				{
					Subject: "Fix authentication bug",
					Body:    "Users were unable to login due to null check missing.",
				},
			},
			wantWhat: "Fix authentication bug",
			wantWhy:  "Users were unable to login due to null check missing.",
			wantHow:  "Auto-documented",
		},
		{
			name: "single commit with multi-paragraph body",
			commits: []git.Commit{
				{
					Subject: "Refactor database layer",
					Body:    "The old implementation was too slow.\n\nAdded connection pooling and query caching.\nAlso optimized indexes.",
				},
			},
			wantWhat: "Refactor database layer",
			wantWhy:  "The old implementation was too slow.",
			wantHow:  "Added connection pooling and query caching.\nAlso optimized indexes.",
		},
		{
			name: "multiple commits, first has body",
			commits: []git.Commit{
				{
					Subject: "Latest commit",
					Body:    "This is why.\n\nThis is how.",
				},
				{
					Subject: "Previous commit",
					Body:    "Different body",
				},
			},
			wantWhat: "Latest commit; Previous commit",
			wantWhy:  "This is why.",
			wantHow:  "This is how.",
		},
		{
			name: "multiple commits, second has body",
			commits: []git.Commit{
				{Subject: "First commit"},
				{
					Subject: "Second commit",
					Body:    "Body from second.\n\nHow from second.",
				},
			},
			wantWhat: "First commit; Second commit",
			wantWhy:  "Body from second.",
			wantHow:  "How from second.",
		},
		{
			name:     "no commits",
			commits:  []git.Commit{},
			wantWhat: "Auto-documented",
			wantWhy:  "Auto-documented",
			wantHow:  "Auto-documented",
		},
		{
			name: "commit with empty subject",
			commits: []git.Commit{
				{Subject: "", Body: "Just a body"},
			},
			wantWhat: "Auto-documented",
			wantWhy:  "Just a body",
			wantHow:  "Auto-documented",
		},
		{
			name: "body with multiple blank lines between paragraphs",
			commits: []git.Commit{
				{
					Subject: "Feature X",
					Body:    "First paragraph.\n\n\n\nSecond paragraph.",
				},
			},
			wantWhat: "Feature X",
			wantWhy:  "First paragraph.",
			wantHow:  "Second paragraph.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			what, why, how := extractAutoContent(tt.commits)

			if what != tt.wantWhat {
				t.Errorf("what = %q, want %q", what, tt.wantWhat)
			}
			if why != tt.wantWhy {
				t.Errorf("why = %q, want %q", why, tt.wantWhy)
			}
			if how != tt.wantHow {
				t.Errorf("how = %q, want %q", how, tt.wantHow)
			}
		})
	}
}

func TestLogCommandAutoMode(t *testing.T) {
	tests := []struct {
		name           string
		mock           *mockGitOpsForLog
		args           []string
		jsonOutput     bool
		wantErr        bool
		wantContains   []string
		wantNotContain []string
		checkDir       func(t *testing.T, dir string)
	}{
		{
			name: "auto mode - extracts from commit subjects",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Add feature X"},
					{SHA: "def456789012345", Short: "def4567", Subject: "Fix tests"},
				}
				mock.diffstat = git.Diffstat{Files: 2, Insertions: 30, Deletions: 10}
				return mock
			}(),
			args:         []string{"--auto"},
			wantErr:      false,
			wantContains: []string{"Created entry"},
			checkDir: func(t *testing.T, dir string) {
				walkJSONFiles(dir, func(_ string, data []byte) {
					content := string(data)
					if !strings.Contains(content, "Add feature X; Fix tests") {
						t.Errorf("expected combined subjects in entry, got: %s", content)
					}
					if !strings.Contains(content, "Auto-documented") {
						t.Errorf("expected Auto-documented default in entry, got: %s", content)
					}
				})
			},
		},
		{
			name: "auto mode - extracts why/how from body",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{
						SHA:     "abc123def456789",
						Short:   "abc123d",
						Subject: "Fix auth bug",
						Body:    "Users couldn't login.\n\nAdded null check to auth handler.",
					},
				}
				mock.diffstat = git.Diffstat{Files: 1, Insertions: 5, Deletions: 2}
				return mock
			}(),
			args:         []string{"--auto"},
			wantErr:      false,
			wantContains: []string{"Created entry"},
			checkDir: func(t *testing.T, dir string) {
				walkJSONFiles(dir, func(_ string, data []byte) {
					content := string(data)
					if !strings.Contains(content, "Fix auth bug") {
						t.Errorf("expected subject as what, got: %s", content)
					}
					if !strings.Contains(content, "Users couldn't login.") {
						t.Errorf("expected first paragraph as why, got: %s", content)
					}
					if !strings.Contains(content, "Added null check to auth handler.") {
						t.Errorf("expected second paragraph as how, got: %s", content)
					}
				})
			},
		},
		{
			name: "auto mode with what override",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Original subject"},
				}
				mock.diffstat = git.Diffstat{Files: 1, Insertions: 5, Deletions: 2}
				return mock
			}(),
			args:         []string{"Custom what", "--auto"},
			wantErr:      false,
			wantContains: []string{"Created entry"},
			checkDir: func(t *testing.T, dir string) {
				walkJSONFiles(dir, func(_ string, data []byte) {
					content := string(data)
					if !strings.Contains(content, "Custom what") {
						t.Errorf("expected custom what in entry, got: %s", content)
					}
					if strings.Contains(content, "Original subject") {
						t.Errorf("should not contain original subject when overridden, got: %s", content)
					}
				})
			},
		},
		{
			name: "auto mode with why/how override",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{
						SHA:     "abc123def456789",
						Short:   "abc123d",
						Subject: "Feature X",
						Body:    "Original why.\n\nOriginal how.",
					},
				}
				mock.diffstat = git.Diffstat{Files: 1, Insertions: 5, Deletions: 2}
				return mock
			}(),
			args:         []string{"--auto", "--why", "Custom why", "--how", "Custom how"},
			wantErr:      false,
			wantContains: []string{"Created entry"},
			checkDir: func(t *testing.T, dir string) {
				walkJSONFiles(dir, func(_ string, data []byte) {
					content := string(data)
					if !strings.Contains(content, "Custom why") {
						t.Errorf("expected custom why in entry, got: %s", content)
					}
					if !strings.Contains(content, "Custom how") {
						t.Errorf("expected custom how in entry, got: %s", content)
					}
				})
			},
		},
		{
			name: "auto mode with --yes flag",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Quick fix"},
				}
				mock.diffstat = git.Diffstat{Files: 1, Insertions: 2, Deletions: 1}
				return mock
			}(),
			args:         []string{"--auto", "--yes"},
			wantErr:      false,
			wantContains: []string{"Created entry"},
			checkDir: func(t *testing.T, dir string) {
				if n := countJSONFilesInDir(dir); n != 1 {
					t.Errorf("expected 1 entry file written, got %d", n)
				}
			},
		},
		{
			name: "auto mode with dry-run",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "Preview this"},
				}
				mock.diffstat = git.Diffstat{Files: 1, Insertions: 5, Deletions: 2}
				return mock
			}(),
			args:         []string{"--auto", "--dry-run"},
			wantErr:      false,
			wantContains: []string{"Dry Run Preview", "Preview this", "Auto-documented"},
			checkDir: func(t *testing.T, dir string) {
				if n := countJSONFilesInDir(dir); n != 0 {
					t.Errorf("expected no entries in dry-run mode, got %d", n)
				}
			},
		},
		{
			name: "auto mode JSON output",
			mock: func() *mockGitOpsForLog {
				mock := newMockGitOpsForLog()
				mock.head = "abc123def456789"
				mock.reachableResult = []git.Commit{
					{SHA: "abc123def456789", Short: "abc123d", Subject: "JSON test"},
				}
				mock.diffstat = git.Diffstat{Files: 1, Insertions: 5, Deletions: 2}
				return mock
			}(),
			args:         []string{"--auto"},
			jsonOutput:   true,
			wantErr:      false,
			wantContains: []string{`"status": "created"`, `"id":`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create storage with mock and temp dir
			storage, dir := newLogTestStorage(t, tt.mock)

			// Create command
			cmd := newLogCmdWithStorage(storage)

			// Set JSON mode for testing
			if tt.jsonOutput {
				cmd.PersistentFlags().Bool("json", false, "")
				_ = cmd.PersistentFlags().Set("json", "true")
			}

			// Set args
			cmd.SetArgs(tt.args)

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
				if jsonErr := json.Unmarshal([]byte(output), &result); jsonErr != nil {
					t.Errorf("failed to parse JSON output: %v\noutput: %s", jsonErr, output)
				}
			}

			// Run dir checks if provided
			if tt.checkDir != nil && err == nil {
				tt.checkDir(t, dir)
			}
		})
	}
}

func TestSplitIntoParagraphs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single paragraph",
			input: "Just one paragraph.",
			want:  []string{"Just one paragraph."},
		},
		{
			name:  "two paragraphs",
			input: "First paragraph.\n\nSecond paragraph.",
			want:  []string{"First paragraph.", "Second paragraph."},
		},
		{
			name:  "multiple blank lines",
			input: "First.\n\n\n\nSecond.",
			want:  []string{"First.", "Second."},
		},
		{
			name:  "multiline paragraph",
			input: "Line one\nLine two\n\nSecond para.",
			want:  []string{"Line one\nLine two", "Second para."},
		},
		{
			name:  "trailing whitespace",
			input: "First.\n\nSecond.\n\n",
			want:  []string{"First.", "Second."},
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "only whitespace",
			input: "   \n\n   ",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitIntoParagraphs(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitIntoParagraphs() returned %d paragraphs, want %d\ngot: %v", len(got), len(tt.want), got)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("paragraph[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLogWriteError(t *testing.T) {
	mock := newMockGitOpsForLog()
	mock.head = "abc123def456789"
	mock.reachableResult = []git.Commit{
		{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
	}
	mock.diffstat = git.Diffstat{Files: 1, Insertions: 5, Deletions: 2}

	dir := t.TempDir()
	failAdd := func(_ string) error { return output.NewSystemError("write failed") }
	files := ledger.NewFileStorage(dir, failAdd, func(_, _ string) error { return nil })
	storage := ledger.NewStorage(mock, files)

	cmd := newLogCmdWithStorage(storage)
	cmd.SetArgs([]string{"Test feature", "--why", "Testing", "--how", "Test code"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when git add fails")
	}
}

func TestLogStaleAnchorSucceeds(t *testing.T) {
	mock := newMockGitOpsForLog()
	mock.head = "abc123def456789"
	mock.commitsErr = errors.New("bad object oldanchor")
	mock.reachableResult = []git.Commit{
		{SHA: "abc123def456789", Short: "abc123d", Subject: "New commit"},
	}
	mock.diffstat = git.Diffstat{Files: 2, Insertions: 10, Deletions: 3}

	storage, dir := newLogTestStorage(t, mock)

	// Write a pre-existing entry so GetPendingCommits hits the stale anchor path
	now := time.Now().UTC()
	entry := &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        ledger.GenerateID("oldanchor1234567890", now),
		CreatedAt: now,
		UpdatedAt: now,
		Workset:   ledger.Workset{AnchorCommit: "oldanchor1234567890", Commits: []string{"oldanchor1234567890"}},
		Summary:   ledger.Summary{What: "Old work", Why: "Old", How: "Old"},
	}
	data, err := entry.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize entry: %v", err)
	}
	entryDir := filepath.Join(dir, ledger.EntryDateDir(entry.ID))
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, entry.ID+".json"), data, 0o600); err != nil {
		t.Fatalf("failed to write entry: %v", err)
	}

	cmd := newLogCmdWithStorage(storage)
	cmd.SetArgs([]string{"Work after squash merge", "--why", "Reason", "--how", "Method"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	execErr := cmd.Execute()
	if execErr != nil {
		t.Fatalf("expected log to succeed with stale anchor, got: %v", execErr)
	}

	out := buf.String()
	if !strings.Contains(out, "Created entry") {
		t.Error("expected entry to be created")
	}
	if !strings.Contains(out, "stale anchor") {
		t.Error("expected stale anchor warning")
	}
}

// TestLogAnchorBypassesZeroPending: when detection finds 0 pending commits but
// --anchor <sha> is given, log should document that single commit rather than
// refusing — the flag's name promises "use this anchor."
func TestLogAnchorBypassesZeroPending(t *testing.T) {
	mock := newMockGitOpsForLog()
	mock.head = "head000000000000"
	mock.commits = []git.Commit{}     // pending range is empty → 0 detected
	mock.rangeCommits = []git.Commit{ // the explicit --anchor commit resolves to one commit
		{SHA: "anchor0000000000", Short: "anchor0", Subject: "explicit anchor commit"},
	}
	mock.diffstat = git.Diffstat{Files: 1, Insertions: 2, Deletions: 0}

	storage, dir := newLogTestStorage(t, mock)

	// A pre-existing entry makes latest != nil so the empty Log(anchor,head)
	// walk yields 0 pending (rather than the no-entries path).
	now := time.Now().UTC()
	prior := &ledger.Entry{
		Schema: ledger.SchemaVersion, Kind: ledger.KindEntry,
		ID:        ledger.GenerateID("prioranchor0000000", now),
		CreatedAt: now, UpdatedAt: now,
		Workset: ledger.Workset{AnchorCommit: "prioranchor0000000", Commits: []string{"prioranchor0000000"}},
		Summary: ledger.Summary{What: "prior", Why: "prior", How: "prior"},
	}
	data, err := prior.ToJSON()
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	entryDir := filepath.Join(dir, ledger.EntryDateDir(prior.ID))
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, prior.ID+".json"), data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := newLogCmdWithStorage(storage)
	cmd.SetArgs([]string{"Document explicit commit", "--why", "real work", "--how", "manual", "--anchor", "anchor0000000000"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if execErr := cmd.Execute(); execErr != nil {
		t.Fatalf("expected --anchor to log at 0 pending, got: %v", execErr)
	}
	if out := buf.String(); !strings.Contains(out, "Created entry") {
		t.Errorf("expected entry created via --anchor at 0 pending, got: %q", out)
	}
	if n := countJSONFilesInDir(dir); n != 2 { // prior + the new one
		t.Errorf("expected 2 entry files, got %d", n)
	}
}
