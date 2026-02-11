//go:build integration

// Package integration provides integration tests for the timbers CLI.
// These tests create real git repositories and run full command workflows.
//
// Run with: go test -tags=integration ./internal/integration/...
package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testRepo is a helper for creating and managing test git repositories.
type testRepo struct {
	t       *testing.T
	dir     string
	binary  string
	commits []string // SHAs of created commits
}

// newTestRepo creates a new git repository in a temp directory.
// It builds the timbers binary and initializes a git repo.
func newTestRepo(t *testing.T) *testRepo {
	t.Helper()

	dir := t.TempDir()

	// Build the timbers binary
	binary := filepath.Join(dir, "timbers")
	buildCmd := exec.Command("go", "build", "-o", binary, "./cmd/timbers")
	buildCmd.Dir = findProjectRoot(t)
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build timbers: %v\n%s", err, output)
	}

	// Initialize git repo
	repo := &testRepo{
		t:       t,
		dir:     dir,
		binary:  binary,
		commits: make([]string, 0),
	}

	repo.git("init", "--initial-branch=main")
	repo.git("config", "user.email", "test@example.com")
	repo.git("config", "user.name", "Test User")

	return repo
}

// findProjectRoot locates the project root by finding go.mod.
func findProjectRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// git runs a git command in the test repo.
func (r *testRepo) git(args ...string) string {
	r.t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

// gitMayFail runs a git command that may fail.
func (r *testRepo) gitMayFail(args ...string) (string, error) {
	r.t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// createFile creates a file with the given content.
func (r *testRepo) createFile(name, content string) {
	r.t.Helper()

	path := filepath.Join(r.dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		r.t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		r.t.Fatalf("failed to write file %s: %v", name, err)
	}
}

// commit creates a commit with the given message.
func (r *testRepo) commit(msg string) string {
	r.t.Helper()

	r.git("add", "-A")
	r.git("commit", "-m", msg)
	sha := r.git("rev-parse", "HEAD")
	r.commits = append(r.commits, sha)
	return sha
}

// commitWithBody creates a commit with subject and body.
func (r *testRepo) commitWithBody(subject, body string) string {
	r.t.Helper()

	r.git("add", "-A")
	r.git("commit", "-m", subject, "-m", body)
	sha := r.git("rev-parse", "HEAD")
	r.commits = append(r.commits, sha)
	return sha
}

// commitWithTrailer creates a commit with a Work-item trailer.
func (r *testRepo) commitWithTrailer(subject, trailerKey, trailerValue string) string {
	r.t.Helper()

	r.git("add", "-A")
	// Use trailer format: subject\n\nTrailer: value
	msg := subject + "\n\n" + trailerKey + ": " + trailerValue
	r.git("commit", "-m", msg)
	sha := r.git("rev-parse", "HEAD")
	r.commits = append(r.commits, sha)
	return sha
}

// timbers runs the timbers command with the given args.
// Returns stdout, stderr, and error.
func (r *testRepo) timbers(args ...string) (string, string, error) {
	r.t.Helper()

	cmd := exec.Command(r.binary, args...)
	cmd.Dir = r.dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// timbersOK runs timbers and expects success.
func (r *testRepo) timbersOK(args ...string) string {
	r.t.Helper()

	stdout, stderr, err := r.timbers(args...)
	if err != nil {
		r.t.Fatalf("timbers %v failed: %v\nstdout: %s\nstderr: %s", args, err, stdout, stderr)
	}
	return stdout
}

// timbersErr runs timbers and expects failure.
func (r *testRepo) timbersErr(args ...string) (string, string) {
	r.t.Helper()

	stdout, stderr, err := r.timbers(args...)
	if err == nil {
		r.t.Fatalf("timbers %v expected to fail but succeeded\nstdout: %s", args, stdout)
	}
	return stdout, stderr
}

// TestLogPendingQueryCycle tests the full workflow:
// create repo -> pending shows commits -> log -> pending shows 0 -> query returns entry.
func TestLogPendingQueryCycle(t *testing.T) {
	repo := newTestRepo(t)

	// Create some commits
	repo.createFile("README.md", "# Test Project")
	repo.commit("Initial commit")

	repo.createFile("main.go", "package main\nfunc main() {}")
	repo.commit("Add main.go")

	repo.createFile("util.go", "package main\nfunc helper() {}")
	repo.commit("Add util.go")

	// Step 1: Check pending shows 3 commits
	pendingOut := repo.timbersOK("pending", "--json")
	var pendingResult struct {
		Count   int `json:"count"`
		Commits []struct {
			SHA     string `json:"sha"`
			Subject string `json:"subject"`
		} `json:"commits"`
	}
	if err := json.Unmarshal([]byte(pendingOut), &pendingResult); err != nil {
		t.Fatalf("failed to parse pending JSON: %v", err)
	}
	if pendingResult.Count != 3 {
		t.Errorf("expected 3 pending commits, got %d", pendingResult.Count)
	}

	// Step 2: Log the work
	logOut := repo.timbersOK("log", "Added project structure",
		"--why", "Setting up the codebase",
		"--how", "Created main and utility files",
		"--json")

	var logResult struct {
		Status string `json:"status"`
		ID     string `json:"id"`
		Anchor string `json:"anchor"`
	}
	if err := json.Unmarshal([]byte(logOut), &logResult); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}
	if logResult.Status != "created" {
		t.Errorf("expected status 'created', got %q", logResult.Status)
	}
	if logResult.ID == "" {
		t.Error("expected non-empty entry ID")
	}

	// Step 3: Pending should now show 0
	pendingOut2 := repo.timbersOK("pending", "--json")
	var pendingResult2 struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(pendingOut2), &pendingResult2); err != nil {
		t.Fatalf("failed to parse pending JSON: %v", err)
	}
	if pendingResult2.Count != 0 {
		t.Errorf("expected 0 pending commits after log, got %d", pendingResult2.Count)
	}

	// Step 4: Query should return the entry
	queryOut := repo.timbersOK("query", "--last", "1", "--json")
	var entries []struct {
		ID      string `json:"id"`
		Summary struct {
			What string `json:"what"`
			Why  string `json:"why"`
			How  string `json:"how"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(queryOut), &entries); err != nil {
		t.Fatalf("failed to parse query JSON: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID != logResult.ID {
		t.Errorf("expected entry ID %q, got %q", logResult.ID, entries[0].ID)
	}
	if entries[0].Summary.What != "Added project structure" {
		t.Errorf("expected what 'Added project structure', got %q", entries[0].Summary.What)
	}

	// Step 5: Export should output valid JSON
	exportOut := repo.timbersOK("export", "--last", "1", "--format", "json")
	var exportedEntries []map[string]any
	if err := json.Unmarshal([]byte(exportOut), &exportedEntries); err != nil {
		t.Fatalf("failed to parse export JSON: %v\noutput: %s", err, exportOut)
	}
	if len(exportedEntries) != 1 {
		t.Errorf("expected 1 exported entry, got %d", len(exportedEntries))
	}
}

// TestAutoMode tests the --auto flag extracts content from commits.
func TestAutoMode(t *testing.T) {
	repo := newTestRepo(t)

	// Create commit with body
	repo.createFile("feature.go", "package main\nfunc feature() {}")
	repo.commitWithBody("Add new feature",
		"Users requested this feature.\n\nImplemented using new design pattern.")

	// Run log with --auto --dry-run to see what would be created
	logOut := repo.timbersOK("log", "--auto", "--dry-run", "--json")

	var logResult struct {
		Status string `json:"status"`
		Entry  struct {
			Summary struct {
				What string `json:"what"`
				Why  string `json:"why"`
				How  string `json:"how"`
			} `json:"summary"`
		} `json:"entry"`
	}
	if err := json.Unmarshal([]byte(logOut), &logResult); err != nil {
		t.Fatalf("failed to parse log JSON: %v\noutput: %s", err, logOut)
	}

	if logResult.Status != "dry_run" {
		t.Errorf("expected status 'dry_run', got %q", logResult.Status)
	}

	// Verify auto-extraction worked
	if !strings.Contains(logResult.Entry.Summary.What, "Add new feature") {
		t.Errorf("expected what to contain 'Add new feature', got %q", logResult.Entry.Summary.What)
	}
	if !strings.Contains(logResult.Entry.Summary.Why, "Users requested") {
		t.Errorf("expected why to contain 'Users requested', got %q", logResult.Entry.Summary.Why)
	}
	if !strings.Contains(logResult.Entry.Summary.How, "design pattern") {
		t.Errorf("expected how to contain 'design pattern', got %q", logResult.Entry.Summary.How)
	}
}

// TestMinorMode tests the --minor flag.
func TestMinorMode(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("typo.txt", "fixed typo")
	repo.commit("Fix typo")

	// Run log with --minor (no --why/--how required)
	logOut := repo.timbersOK("log", "Fixed typo in docs", "--minor", "--json")

	var logResult struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(logOut), &logResult); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}
	if logResult.Status != "created" {
		t.Errorf("expected status 'created', got %q", logResult.Status)
	}
}

// TestDryRun tests the --dry-run flag.
func TestDryRun(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("test.txt", "test content")
	repo.commit("Test commit")

	// Run log with --dry-run
	logOut := repo.timbersOK("log", "Test entry",
		"--why", "Testing",
		"--how", "Test method",
		"--dry-run", "--json")

	var logResult struct {
		Status string `json:"status"`
		Entry  struct {
			ID string `json:"id"`
		} `json:"entry"`
	}
	if err := json.Unmarshal([]byte(logOut), &logResult); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}
	if logResult.Status != "dry_run" {
		t.Errorf("expected status 'dry_run', got %q", logResult.Status)
	}

	// Verify nothing was written - pending should still show 1
	pendingOut := repo.timbersOK("pending", "--json")
	var pendingResult struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(pendingOut), &pendingResult); err != nil {
		t.Fatalf("failed to parse pending JSON: %v", err)
	}
	if pendingResult.Count != 1 {
		t.Errorf("expected 1 pending commit after dry-run, got %d", pendingResult.Count)
	}
}

// TestShowCommand tests the show command.
func TestShowCommand(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("file.txt", "content")
	repo.commit("Add file")

	// Create an entry
	logOut := repo.timbersOK("log", "Test work",
		"--why", "Test reason",
		"--how", "Test method",
		"--json")

	var logResult struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(logOut), &logResult); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}

	// Show by ID
	showOut := repo.timbersOK("show", logResult.ID, "--json")
	var showResult struct {
		ID      string `json:"id"`
		Summary struct {
			What string `json:"what"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(showOut), &showResult); err != nil {
		t.Fatalf("failed to parse show JSON: %v", err)
	}
	if showResult.ID != logResult.ID {
		t.Errorf("show returned wrong ID: expected %q, got %q", logResult.ID, showResult.ID)
	}

	// Show with --last
	showLastOut := repo.timbersOK("show", "--last", "--json")
	var showLastResult struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(showLastOut), &showLastResult); err != nil {
		t.Fatalf("failed to parse show --last JSON: %v", err)
	}
	if showLastResult.ID != logResult.ID {
		t.Errorf("show --last returned wrong ID")
	}
}

// TestErrorNotGitRepo tests error when running outside a git repo.
func TestErrorNotGitRepo(t *testing.T) {
	// Create a directory that is NOT a git repo
	dir := t.TempDir()

	// Build the binary
	binary := filepath.Join(dir, "timbers")
	buildCmd := exec.Command("go", "build", "-o", binary, "./cmd/timbers")
	buildCmd.Dir = findProjectRoot(t)
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build timbers: %v\n%s", err, output)
	}

	// Create a non-git directory
	nonGitDir := filepath.Join(dir, "not-a-repo")
	if err := os.MkdirAll(nonGitDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Run commands and expect errors
	cmds := [][]string{
		{"pending"},
		{"log", "test", "--why", "test", "--how", "test"},
		{"show", "--last"},
		{"query", "--last", "1"},
		{"status"},
	}

	for _, args := range cmds {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			cmd := exec.Command(binary, append(args, "--json")...)
			cmd.Dir = nonGitDir

			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected error for %v outside git repo", args)
			}

			// Verify JSON error format
			var errResult struct {
				Error string `json:"error"`
				Code  int    `json:"code"`
			}
			if jsonErr := json.Unmarshal(output, &errResult); jsonErr != nil {
				t.Fatalf("expected JSON error output, got: %s", output)
			}
			if !strings.Contains(errResult.Error, "git repository") {
				t.Errorf("expected 'git repository' in error, got: %s", errResult.Error)
			}
			if errResult.Code != 2 {
				t.Errorf("expected exit code 2 (system error), got: %d", errResult.Code)
			}
		})
	}
}

// TestErrorMissingArgs tests error handling for missing arguments.
func TestErrorMissingArgs(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("file.txt", "content")
	repo.commit("Initial")

	tests := []struct {
		name    string
		args    []string
		wantErr string
		code    int
	}{
		{
			name:    "log missing what",
			args:    []string{"log", "--why", "reason", "--how", "method"},
			wantErr: "what",
			code:    1,
		},
		{
			name:    "log missing why (not minor)",
			args:    []string{"log", "work done", "--how", "method"},
			wantErr: "--why",
			code:    1,
		},
		{
			name:    "log missing how (not minor)",
			args:    []string{"log", "work done", "--why", "reason"},
			wantErr: "--how",
			code:    1,
		},
		{
			name:    "show missing id or --last",
			args:    []string{"show"},
			wantErr: "specify",
			code:    1,
		},
		{
			name:    "query missing --last",
			args:    []string{"query"},
			wantErr: "--last",
			code:    1,
		},
		{
			name:    "export missing --last or --range",
			args:    []string{"export"},
			wantErr: "--last",
			code:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := repo.timbersErr(append(tt.args, "--json")...)

			// Parse combined output (error might be in stdout for JSON mode)
			output := stdout + stderr
			var errResult struct {
				Error string `json:"error"`
				Code  int    `json:"code"`
			}
			if err := json.Unmarshal([]byte(output), &errResult); err != nil {
				t.Fatalf("expected JSON error, got: %s", output)
			}

			if !strings.Contains(strings.ToLower(errResult.Error), strings.ToLower(tt.wantErr)) {
				t.Errorf("expected error containing %q, got: %s", tt.wantErr, errResult.Error)
			}
			if errResult.Code != tt.code {
				t.Errorf("expected code %d, got %d", tt.code, errResult.Code)
			}
		})
	}
}

// TestErrorShowNonexistent tests error for nonexistent entry ID.
func TestErrorShowNonexistent(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("file.txt", "content")
	repo.commit("Initial")

	stdout, stderr := repo.timbersErr("show", "tb_nonexistent_123456", "--json")
	output := stdout + stderr

	var errResult struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}
	if err := json.Unmarshal([]byte(output), &errResult); err != nil {
		t.Fatalf("expected JSON error, got: %s", output)
	}

	if !strings.Contains(errResult.Error, "not found") {
		t.Errorf("expected 'not found' in error, got: %s", errResult.Error)
	}
	if errResult.Code != 1 {
		t.Errorf("expected code 1 (user error), got: %d", errResult.Code)
	}
}

// TestMultipleEntries tests creating multiple entries and querying them.
func TestMultipleEntries(t *testing.T) {
	repo := newTestRepo(t)

	// Create first entry
	repo.createFile("file1.txt", "content1")
	repo.commit("First commit")
	repo.timbersOK("log", "First entry", "--why", "First reason", "--how", "First method")

	// Create second entry
	repo.createFile("file2.txt", "content2")
	repo.commit("Second commit")
	repo.timbersOK("log", "Second entry", "--why", "Second reason", "--how", "Second method")

	// Create third entry
	repo.createFile("file3.txt", "content3")
	repo.commit("Third commit")
	repo.timbersOK("log", "Third entry", "--why", "Third reason", "--how", "Third method")

	// Query last 2 should return 2 entries
	queryOut := repo.timbersOK("query", "--last", "2", "--json")
	var entries []struct {
		Summary struct {
			What string `json:"what"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(queryOut), &entries); err != nil {
		t.Fatalf("failed to parse query JSON: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	// Most recent should be first
	if entries[0].Summary.What != "Third entry" {
		t.Errorf("expected first entry to be 'Third entry', got %q", entries[0].Summary.What)
	}
	if entries[1].Summary.What != "Second entry" {
		t.Errorf("expected second entry to be 'Second entry', got %q", entries[1].Summary.What)
	}
}

// TestTagsAndWorkItems tests --tag and --work-item flags.
func TestTagsAndWorkItems(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("feature.go", "package main")
	repo.commit("Add feature")

	// Log with tags and work items
	repo.timbersOK("log", "Added security feature",
		"--why", "Security requirement",
		"--how", "Implemented auth middleware",
		"--tag", "security",
		"--tag", "feature",
		"--work-item", "jira:SEC-123",
		"--work-item", "beads:bd-abc123")

	// Query and verify
	queryOut := repo.timbersOK("query", "--last", "1", "--json")
	var entries []struct {
		Tags      []string `json:"tags"`
		WorkItems []struct {
			System string `json:"system"`
			ID     string `json:"id"`
		} `json:"work_items"`
	}
	if err := json.Unmarshal([]byte(queryOut), &entries); err != nil {
		t.Fatalf("failed to parse query JSON: %v", err)
	}

	if len(entries[0].Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(entries[0].Tags))
	}
	if len(entries[0].WorkItems) != 2 {
		t.Errorf("expected 2 work items, got %d", len(entries[0].WorkItems))
	}

	// Check specific values
	foundSecurity := false
	foundFeature := false
	for _, tag := range entries[0].Tags {
		if tag == "security" {
			foundSecurity = true
		}
		if tag == "feature" {
			foundFeature = true
		}
	}
	if !foundSecurity || !foundFeature {
		t.Errorf("expected tags [security, feature], got %v", entries[0].Tags)
	}
}

// TestExportFormats tests export in different formats.
func TestExportFormats(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("code.go", "package main")
	repo.commit("Add code")
	repo.timbersOK("log", "Code work", "--why", "Needed code", "--how", "Wrote code")

	// Test JSON format to stdout
	jsonOut := repo.timbersOK("export", "--last", "1", "--format", "json")
	var jsonEntries []map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &jsonEntries); err != nil {
		t.Fatalf("failed to parse export JSON: %v\noutput: %s", err, jsonOut)
	}
	if len(jsonEntries) != 1 {
		t.Errorf("expected 1 JSON entry, got %d", len(jsonEntries))
	}

	// Test markdown format to stdout
	mdOut := repo.timbersOK("export", "--last", "1", "--format", "md")
	if !strings.Contains(mdOut, "Code work") {
		t.Errorf("expected markdown to contain 'Code work', got: %s", mdOut)
	}
	if !strings.Contains(mdOut, "Needed code") {
		t.Errorf("expected markdown to contain 'Needed code', got: %s", mdOut)
	}
}

// TestStatusCommand tests the status command.
func TestStatusCommand(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("file.txt", "content")
	repo.commit("Initial")

	// Status should work
	statusOut := repo.timbersOK("status", "--json")
	var statusResult struct {
		Repo       string `json:"repo"`
		Branch     string `json:"branch"`
		Head       string `json:"head"`
		EntryCount int    `json:"entry_count"`
	}
	if err := json.Unmarshal([]byte(statusOut), &statusResult); err != nil {
		t.Fatalf("failed to parse status JSON: %v", err)
	}
	if statusResult.Repo == "" {
		t.Error("expected repo to be non-empty")
	}
	if statusResult.Branch != "main" {
		t.Errorf("expected branch 'main', got %q", statusResult.Branch)
	}
	if statusResult.Head == "" {
		t.Error("expected head to be non-empty")
	}
}

// TestBatchModeByDay tests --batch groups commits by day when no Work-item trailers.
func TestBatchModeByDay(t *testing.T) {
	repo := newTestRepo(t)

	// Create commits on the same "day" (in test, they all happen quickly)
	repo.createFile("file1.txt", "content1")
	repo.commit("First change")

	repo.createFile("file2.txt", "content2")
	repo.commit("Second change")

	repo.createFile("file3.txt", "content3")
	repo.commit("Third change")

	// Run batch mode with dry-run to see what would be created
	batchOut := repo.timbersOK("log", "--batch", "--dry-run", "--json")

	var batchResult struct {
		Status  string `json:"status"`
		Count   int    `json:"count"`
		Entries []struct {
			ID       string `json:"id"`
			GroupKey string `json:"group_key"`
			What     string `json:"what"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(batchOut), &batchResult); err != nil {
		t.Fatalf("failed to parse batch JSON: %v\noutput: %s", err, batchOut)
	}

	if batchResult.Status != "dry_run" {
		t.Errorf("expected status 'dry_run', got %q", batchResult.Status)
	}

	// All commits on same day should be grouped together
	if batchResult.Count != 1 {
		t.Errorf("expected 1 group (all commits on same day), got %d", batchResult.Count)
	}
}

// TestBatchModeWithTrailers tests --batch groups commits by Work-item trailer.
func TestBatchModeWithTrailers(t *testing.T) {
	repo := newTestRepo(t)

	// Create commits with Work-item trailers
	repo.createFile("feature1.go", "package main")
	repo.commitWithBody("Add feature 1", "Implementation details\n\nWork-item: jira:PROJ-100")

	repo.createFile("feature2.go", "package main")
	repo.commitWithBody("Add feature 2", "More details\n\nWork-item: jira:PROJ-100")

	repo.createFile("bugfix.go", "package main")
	repo.commitWithBody("Fix bug", "Bug fix details\n\nWork-item: jira:PROJ-200")

	// Run batch mode
	batchOut := repo.timbersOK("log", "--batch", "--dry-run", "--json")

	var batchResult struct {
		Status  string `json:"status"`
		Count   int    `json:"count"`
		Entries []struct {
			ID       string `json:"id"`
			GroupKey string `json:"group_key"`
			What     string `json:"what"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(batchOut), &batchResult); err != nil {
		t.Fatalf("failed to parse batch JSON: %v\noutput: %s", err, batchOut)
	}

	if batchResult.Status != "dry_run" {
		t.Errorf("expected status 'dry_run', got %q", batchResult.Status)
	}

	// Should have 2 groups (one for each work-item)
	if batchResult.Count != 2 {
		t.Errorf("expected 2 groups (2 work-items), got %d", batchResult.Count)
	}

	// Verify group keys include work-items
	foundProj100 := false
	foundProj200 := false
	for _, e := range batchResult.Entries {
		if e.GroupKey == "jira:PROJ-100" {
			foundProj100 = true
		}
		if e.GroupKey == "jira:PROJ-200" {
			foundProj200 = true
		}
	}
	if !foundProj100 || !foundProj200 {
		t.Errorf("expected groups for jira:PROJ-100 and jira:PROJ-200, got %v", batchResult.Entries)
	}
}

// TestInvalidWorkItemFormat tests error for invalid work-item format.
func TestInvalidWorkItemFormat(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("file.txt", "content")
	repo.commit("Initial")

	stdout, stderr := repo.timbersErr("log", "Work",
		"--why", "Reason",
		"--how", "Method",
		"--work-item", "invalid-no-colon",
		"--json")

	output := stdout + stderr
	var errResult struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}
	if err := json.Unmarshal([]byte(output), &errResult); err != nil {
		t.Fatalf("expected JSON error, got: %s", output)
	}

	if !strings.Contains(errResult.Error, "system:id") {
		t.Errorf("expected error about format, got: %s", errResult.Error)
	}
}
