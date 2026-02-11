// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/git"
)

func TestExtractWorkItemTrailer(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "standard work-item trailer",
			body: "Fixed the bug\n\nWork-item: jira:PROJ-123",
			want: "jira:PROJ-123",
		},
		{
			name: "lowercase work-item",
			body: "Fixed the bug\n\nwork-item: github:456",
			want: "github:456",
		},
		{
			name: "mixed case",
			body: "Fixed the bug\n\nWork-Item: beads:bd-abc123",
			want: "beads:bd-abc123",
		},
		{
			name: "with extra whitespace",
			body: "Fixed the bug\n\nWork-item:   jira:PROJ-789  ",
			want: "jira:PROJ-789",
		},
		{
			name: "no trailer",
			body: "Just a commit message\n\nSome explanation.",
			want: "",
		},
		{
			name: "empty body",
			body: "",
			want: "",
		},
		{
			name: "trailer at start of body",
			body: "Work-item: linear:ABC-123\n\nMore details here.",
			want: "linear:ABC-123",
		},
		{
			name: "multiple trailers - returns first",
			body: "Work-item: jira:PROJ-1\nWork-item: jira:PROJ-2",
			want: "jira:PROJ-1",
		},
		{
			name: "invalid format - no colon in value",
			body: "Work-item: invalid",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWorkItemTrailer(tt.body)
			if got != tt.want {
				t.Errorf("extractWorkItemTrailer() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGroupCommitsByTrailer(t *testing.T) {
	commits := []git.Commit{
		{SHA: "aaa111", Short: "aaa111", Subject: "Fix bug 1", Body: "Work-item: jira:PROJ-1"},
		{SHA: "bbb222", Short: "bbb222", Subject: "Fix bug 2", Body: "Work-item: jira:PROJ-1"},
		{SHA: "ccc333", Short: "ccc333", Subject: "Fix bug 3", Body: "Work-item: jira:PROJ-2"},
		{SHA: "ddd444", Short: "ddd444", Subject: "No trailer", Body: "Just a message"},
	}

	groups := groupCommitsByTrailer(commits)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}

	// Verify groups are sorted by key (reverse order)
	groupMap := make(map[string]int)
	for _, g := range groups {
		groupMap[g.key] = len(g.commits)
	}

	if groupMap["jira:PROJ-1"] != 2 {
		t.Errorf("expected 2 commits in jira:PROJ-1 group, got %d", groupMap["jira:PROJ-1"])
	}
	if groupMap["jira:PROJ-2"] != 1 {
		t.Errorf("expected 1 commit in jira:PROJ-2 group, got %d", groupMap["jira:PROJ-2"])
	}
	if groupMap["untracked"] != 1 {
		t.Errorf("expected 1 commit in untracked group, got %d", groupMap["untracked"])
	}
}

func TestGroupCommitsByTrailer_NoTrailers(t *testing.T) {
	commits := []git.Commit{
		{SHA: "aaa111", Short: "aaa111", Subject: "Commit 1", Body: "No trailers here"},
		{SHA: "bbb222", Short: "bbb222", Subject: "Commit 2", Body: "Also no trailers"},
	}

	groups := groupCommitsByTrailer(commits)

	if len(groups) != 0 {
		t.Errorf("expected empty groups when no trailers, got %d groups", len(groups))
	}
}

func TestGroupCommitsByDay(t *testing.T) {
	day1 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 16, 14, 0, 0, 0, time.UTC)

	commits := []git.Commit{
		{SHA: "aaa111", Short: "aaa111", Subject: "Commit 1", Date: day1},
		{SHA: "bbb222", Short: "bbb222", Subject: "Commit 2", Date: day1},
		{SHA: "ccc333", Short: "ccc333", Subject: "Commit 3", Date: day2},
	}

	groups := groupCommitsByDay(commits)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// Groups should be sorted by date in reverse order (newest first)
	if groups[0].key != "2026-01-16" {
		t.Errorf("expected first group to be 2026-01-16, got %s", groups[0].key)
	}
	if groups[1].key != "2026-01-15" {
		t.Errorf("expected second group to be 2026-01-15, got %s", groups[1].key)
	}

	// Verify commit counts
	if len(groups[0].commits) != 1 {
		t.Errorf("expected 1 commit on day 2, got %d", len(groups[0].commits))
	}
	if len(groups[1].commits) != 2 {
		t.Errorf("expected 2 commits on day 1, got %d", len(groups[1].commits))
	}
}

func TestGroupCommits_FallbackToDay(t *testing.T) {
	day1 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	commits := []git.Commit{
		{SHA: "aaa111", Short: "aaa111", Subject: "Commit 1", Body: "No trailer", Date: day1},
		{SHA: "bbb222", Short: "bbb222", Subject: "Commit 2", Body: "Also no trailer", Date: day1},
	}

	groups := groupCommits(commits)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group (by day), got %d", len(groups))
	}

	if groups[0].key != "2026-01-15" {
		t.Errorf("expected group key to be date, got %s", groups[0].key)
	}
}

func TestGroupCommits_UsesTrailerWhenPresent(t *testing.T) {
	day1 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	commits := []git.Commit{
		{SHA: "aaa111", Short: "aaa111", Subject: "Commit 1", Body: "Work-item: jira:PROJ-1", Date: day1},
		{SHA: "bbb222", Short: "bbb222", Subject: "Commit 2", Body: "No trailer", Date: day1},
	}

	groups := groupCommits(commits)

	// Should use trailer grouping since at least one commit has a trailer
	groupMap := make(map[string]int)
	for _, g := range groups {
		groupMap[g.key] = len(g.commits)
	}

	if _, ok := groupMap["jira:PROJ-1"]; !ok {
		t.Error("expected jira:PROJ-1 group when trailer present")
	}
	if _, ok := groupMap["untracked"]; !ok {
		t.Error("expected untracked group for commits without trailer")
	}
}

func TestIsWorkItemKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"jira:PROJ-123", true},
		{"beads:bd-abc", true},
		{"github:456", true},
		{"2026-01-15", false},
		{"untracked", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := isWorkItemKey(tt.key)
			if got != tt.want {
				t.Errorf("isWorkItemKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestBatchLog_MultipleEntries(t *testing.T) {
	day1 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 16, 14, 0, 0, 0, time.UTC)

	mock := newMockGitOpsForLog()
	mock.head = "abc123def456789"
	mock.reachableResult = []git.Commit{
		{SHA: "abc123def456789", Short: "abc123d", Subject: "Day 2 commit", Date: day2},
		{SHA: "def456789012345", Short: "def4567", Subject: "Day 1 commit 1", Date: day1},
		{SHA: "ghi789012345678", Short: "ghi7890", Subject: "Day 1 commit 2", Date: day1},
	}
	mock.diffstat = git.Diffstat{Files: 2, Insertions: 30, Deletions: 10}

	storage, dir := newLogTestStorage(t, mock)
	cmd := newLogCmdWithStorage(storage)
	cmd.SetArgs([]string{"--batch"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Should create 2 entries (one per day)
	if !strings.Contains(output, "Created 2 entries") {
		t.Errorf("expected 'Created 2 entries' in output, got: %s", output)
	}

	// Verify entries were written to directory
	if n := countJSONFilesInDir(dir); n != 2 {
		t.Errorf("expected 2 entry files written, got %d", n)
	}
}

func TestBatchLog_GroupByWorkItem(t *testing.T) {
	now := time.Now()

	mock := newMockGitOpsForLog()
	mock.head = "abc123def456789"
	mock.reachableResult = []git.Commit{
		{SHA: "abc123def456789", Short: "abc123d", Subject: "PROJ-1 fix", Body: "Work-item: jira:PROJ-1", Date: now},
		{SHA: "def456789012345", Short: "def4567", Subject: "PROJ-2 feature", Body: "Work-item: jira:PROJ-2", Date: now},
		{SHA: "ghi789012345678", Short: "ghi7890", Subject: "Another PROJ-1 fix", Body: "Work-item: jira:PROJ-1", Date: now},
	}
	mock.diffstat = git.Diffstat{Files: 2, Insertions: 30, Deletions: 10}

	storage, _ := newLogTestStorage(t, mock)
	cmd := newLogCmdWithStorage(storage)
	cmd.SetArgs([]string{"--batch"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Should create 2 entries (one per work-item)
	if !strings.Contains(output, "Created 2 entries") {
		t.Errorf("expected 'Created 2 entries' in output, got: %s", output)
	}

	// Verify group keys in output
	if !strings.Contains(output, "jira:PROJ-1") {
		t.Errorf("expected jira:PROJ-1 in output, got: %s", output)
	}
	if !strings.Contains(output, "jira:PROJ-2") {
		t.Errorf("expected jira:PROJ-2 in output, got: %s", output)
	}
}

func TestBatchLog_DryRun(t *testing.T) {
	day1 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	mock := newMockGitOpsForLog()
	mock.head = "abc123def456789"
	mock.reachableResult = []git.Commit{
		{SHA: "abc123def456789", Short: "abc123d", Subject: "Commit 1", Date: day1},
		{SHA: "def456789012345", Short: "def4567", Subject: "Commit 2", Date: day1},
	}
	mock.diffstat = git.Diffstat{Files: 1, Insertions: 10, Deletions: 5}

	storage, dir := newLogTestStorage(t, mock)
	cmd := newLogCmdWithStorage(storage)
	cmd.SetArgs([]string{"--batch", "--dry-run"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Should say "would create"
	if !strings.Contains(output, "would create") {
		t.Errorf("expected 'would create' in dry-run output, got: %s", output)
	}

	// Should NOT have written any entry files
	if n := countJSONFilesInDir(dir); n != 0 {
		t.Errorf("expected no entry files in dry-run mode, got %d", n)
	}
}

func TestBatchLog_JSONOutput(t *testing.T) {
	day1 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 16, 14, 0, 0, 0, time.UTC)

	mock := newMockGitOpsForLog()
	mock.head = "abc123def456789"
	mock.reachableResult = []git.Commit{
		{SHA: "abc123def456789", Short: "abc123d", Subject: "Day 2 commit", Date: day2},
		{SHA: "def456789012345", Short: "def4567", Subject: "Day 1 commit", Date: day1},
	}
	mock.diffstat = git.Diffstat{Files: 2, Insertions: 30, Deletions: 10}

	storage, _ := newLogTestStorage(t, mock)
	cmd := newLogCmdWithStorage(storage)
	cmd.PersistentFlags().Bool("json", false, "")
	_ = cmd.PersistentFlags().Set("json", "true")
	cmd.SetArgs([]string{"--batch"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Parse JSON output
	var result batchResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if result.Status != "created" {
		t.Errorf("expected status 'created', got %q", result.Status)
	}
	if result.Count != 2 {
		t.Errorf("expected count 2, got %d", result.Count)
	}
	if len(result.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result.Entries))
	}
}

func TestBatchLog_JSONDryRun(t *testing.T) {
	day1 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	mock := newMockGitOpsForLog()
	mock.head = "abc123def456789"
	mock.reachableResult = []git.Commit{
		{SHA: "abc123def456789", Short: "abc123d", Subject: "Single commit", Date: day1},
	}
	mock.diffstat = git.Diffstat{Files: 1, Insertions: 5, Deletions: 2}

	storage, _ := newLogTestStorage(t, mock)
	cmd := newLogCmdWithStorage(storage)
	cmd.PersistentFlags().Bool("json", false, "")
	_ = cmd.PersistentFlags().Set("json", "true")
	cmd.SetArgs([]string{"--batch", "--dry-run"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	var result batchResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if result.Status != "dry_run" {
		t.Errorf("expected status 'dry_run', got %q", result.Status)
	}
}

func TestBatchLog_NoPendingCommits(t *testing.T) {
	mock := newMockGitOpsForLog()
	mock.head = "abc123def456789"
	mock.reachableResult = []git.Commit{} // No commits

	storage, _ := newLogTestStorage(t, mock)
	cmd := newLogCmdWithStorage(storage)
	cmd.SetArgs([]string{"--batch"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for no pending commits")
	}

	output := buf.String()
	if !strings.Contains(output, "no pending commits") {
		t.Errorf("expected 'no pending commits' error, got: %s", output)
	}
}

func TestBatchLog_WithUntrackedGroup(t *testing.T) {
	now := time.Now()

	mock := newMockGitOpsForLog()
	mock.head = "abc123def456789"
	mock.reachableResult = []git.Commit{
		{SHA: "abc123def456789", Short: "abc123d", Subject: "Tracked commit", Body: "Work-item: jira:PROJ-1", Date: now},
		{SHA: "def456789012345", Short: "def4567", Subject: "Untracked commit", Body: "No work item here", Date: now},
	}
	mock.diffstat = git.Diffstat{Files: 1, Insertions: 5, Deletions: 2}

	storage, _ := newLogTestStorage(t, mock)
	cmd := newLogCmdWithStorage(storage)
	cmd.SetArgs([]string{"--batch"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Should have both jira:PROJ-1 and untracked groups
	if !strings.Contains(output, "jira:PROJ-1") {
		t.Errorf("expected jira:PROJ-1 in output, got: %s", output)
	}
	if !strings.Contains(output, "untracked") {
		t.Errorf("expected untracked in output, got: %s", output)
	}
}
