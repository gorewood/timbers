// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// mockGitOpsForLog implements ledger.GitOps for testing log command.
type mockGitOpsForLog struct {
	head            string
	headErr         error
	commits         []git.Commit
	commitsErr      error
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

func (m *mockGitOpsForLog) Log(_, _ string) ([]git.Commit, error) {
	return m.commits, m.commitsErr
}

func (m *mockGitOpsForLog) CommitsReachableFrom(_ string) ([]git.Commit, error) {
	return m.reachableResult, m.reachableErr
}

func (m *mockGitOpsForLog) GetDiffstat(_, _ string) (git.Diffstat, error) {
	return m.diffstat, m.diffstatErr
}

// newLogTestStorage creates a Storage with a temp dir for writing entries.
func newLogTestStorage(t *testing.T, mock *mockGitOpsForLog) (*ledger.Storage, string) {
	t.Helper()
	dir := t.TempDir()
	files := ledger.NewFileStorage(dir, func(_ string) error { return nil })
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
				entries, _ := os.ReadDir(dir)
				jsonFiles := 0
				for _, e := range entries {
					if strings.HasSuffix(e.Name(), ".json") {
						jsonFiles++
					}
				}
				if jsonFiles != 1 {
					t.Errorf("expected 1 entry file written, got %d", jsonFiles)
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
				entries, _ := os.ReadDir(dir)
				for _, e := range entries {
					if !strings.HasSuffix(e.Name(), ".json") {
						continue
					}
					data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
					content := string(data)
					if !strings.Contains(content, "security") || !strings.Contains(content, "auth") {
						t.Error("expected tags to be in written entry")
					}
				}
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
				entries, _ := os.ReadDir(dir)
				for _, e := range entries {
					if !strings.HasSuffix(e.Name(), ".json") {
						continue
					}
					data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
					content := string(data)
					if !strings.Contains(content, "beads") || !strings.Contains(content, "bd-abc123") {
						t.Error("expected work items to be in written entry")
					}
				}
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
				entries, _ := os.ReadDir(dir)
				for _, e := range entries {
					if !strings.HasSuffix(e.Name(), ".json") {
						continue
					}
					data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
					content := string(data)
					if !strings.Contains(content, "Minor change") {
						t.Error("expected 'Minor change' to be in written entry for --minor mode")
					}
				}
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
				entries, _ := os.ReadDir(dir)
				jsonFiles := 0
				for _, e := range entries {
					if strings.HasSuffix(e.Name(), ".json") {
						jsonFiles++
					}
				}
				if jsonFiles != 0 {
					t.Errorf("expected no entries written in dry-run mode, got %d", jsonFiles)
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
				entries, _ := os.ReadDir(dir)
				for _, e := range entries {
					if !strings.HasSuffix(e.Name(), ".json") {
						continue
					}
					data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
					content := string(data)
					if !strings.Contains(content, "custom123456") {
						t.Error("expected entry written with custom anchor commit")
					}
				}
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

func TestLogDirtyTreeWarning(t *testing.T) {
	mock := newMockGitOpsForLog()
	mock.head = "abc123def456789"
	mock.reachableResult = []git.Commit{
		{SHA: "abc123def456789", Short: "abc123d", Subject: "Latest commit"},
	}
	mock.diffstat = git.Diffstat{Files: 1, Insertions: 10, Deletions: 0}

	storage, _ := newLogTestStorage(t, mock)

	// Test with dirty tree
	cmd := newLogCmdInternal(storage, func() bool { return true })
	cmd.SetArgs([]string{"Test entry", "--why", "Testing", "--how", "Via test"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	_ = cmd.Execute()
	out := buf.String()

	if !strings.Contains(out, "Warning") || !strings.Contains(out, "uncommitted changes") {
		t.Errorf("expected dirty-tree warning in output, got: %s", out)
	}

	// Test with clean tree - need fresh storage since entry ID may conflict
	mock2 := newMockGitOpsForLog()
	mock2.head = "def456789012345"
	mock2.reachableResult = []git.Commit{
		{SHA: "def456789012345", Short: "def4567", Subject: "Another commit"},
	}
	mock2.diffstat = git.Diffstat{Files: 1, Insertions: 10, Deletions: 0}

	storage2, _ := newLogTestStorage(t, mock2)
	cmd2 := newLogCmdInternal(storage2, func() bool { return false })
	cmd2.SetArgs([]string{"Test entry 2", "--why", "Testing", "--how", "Via test"})

	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)

	_ = cmd2.Execute()
	out2 := buf2.String()

	if strings.Contains(out2, "Warning") {
		t.Errorf("expected no warning with clean tree, got: %s", out2)
	}
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
				entries, _ := os.ReadDir(dir)
				for _, e := range entries {
					if !strings.HasSuffix(e.Name(), ".json") {
						continue
					}
					data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
					content := string(data)
					if !strings.Contains(content, "Add feature X; Fix tests") {
						t.Errorf("expected combined subjects in entry, got: %s", content)
					}
					if !strings.Contains(content, "Auto-documented") {
						t.Errorf("expected Auto-documented default in entry, got: %s", content)
					}
				}
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
				entries, _ := os.ReadDir(dir)
				for _, e := range entries {
					if !strings.HasSuffix(e.Name(), ".json") {
						continue
					}
					data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
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
				}
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
				entries, _ := os.ReadDir(dir)
				for _, e := range entries {
					if !strings.HasSuffix(e.Name(), ".json") {
						continue
					}
					data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
					content := string(data)
					if !strings.Contains(content, "Custom what") {
						t.Errorf("expected custom what in entry, got: %s", content)
					}
					if strings.Contains(content, "Original subject") {
						t.Errorf("should not contain original subject when overridden, got: %s", content)
					}
				}
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
				entries, _ := os.ReadDir(dir)
				for _, e := range entries {
					if !strings.HasSuffix(e.Name(), ".json") {
						continue
					}
					data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
					content := string(data)
					if !strings.Contains(content, "Custom why") {
						t.Errorf("expected custom why in entry, got: %s", content)
					}
					if !strings.Contains(content, "Custom how") {
						t.Errorf("expected custom how in entry, got: %s", content)
					}
				}
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
				entries, _ := os.ReadDir(dir)
				jsonFiles := 0
				for _, e := range entries {
					if strings.HasSuffix(e.Name(), ".json") {
						jsonFiles++
					}
				}
				if jsonFiles != 1 {
					t.Errorf("expected 1 entry file written, got %d", jsonFiles)
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
				entries, _ := os.ReadDir(dir)
				jsonFiles := 0
				for _, e := range entries {
					if strings.HasSuffix(e.Name(), ".json") {
						jsonFiles++
					}
				}
				if jsonFiles != 0 {
					t.Errorf("expected no entries in dry-run mode, got %d", jsonFiles)
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
	files := ledger.NewFileStorage(dir, failAdd)
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
