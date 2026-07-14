package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/draft"
	"github.com/gorewood/timbers/internal/ledger"
)

func TestResolveReportSelection(t *testing.T) {
	profile := &draft.ReportProfile{Scope: draft.ReportScope{Last: "20"}}

	got, err := resolveReportSelection(profile, draftFlags{})
	if err != nil || got.last != "20" {
		t.Fatalf("default selection = %#v, %v", got, err)
	}
	got, err = resolveReportSelection(profile, draftFlags{since: "7d", until: "1d"})
	if err != nil || got.last != "" || got.since != "7d" || got.until != "1d" {
		t.Fatalf("explicit selection = %#v, %v", got, err)
	}
	if _, err = resolveReportSelection(profile, draftFlags{last: "2", rng: "main..HEAD"}); err == nil {
		t.Fatal("conflicting primary selections succeeded")
	}
	if _, err = resolveReportSelection(profile, draftFlags{until: "1d"}); err == nil ||
		!strings.Contains(err.Error(), "--until requires a --since") {
		t.Fatalf("standalone until error = %v", err)
	}
	sinceProfile := &draft.ReportProfile{Scope: draft.ReportScope{Since: "7d"}}
	got, err = resolveReportSelection(sinceProfile, draftFlags{until: "1d"})
	if err != nil || got.since != "7d" || got.until != "1d" {
		t.Fatalf("default since with until = %#v, %v", got, err)
	}
}

func TestTemplateCommandsSurfaceInvalidOverride(t *testing.T) {
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(".timbers", "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(".timbers", "templates", "decision-digest.md")
	if err := os.WriteFile(path, []byte("---\nname: [invalid\n---\nbody"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{"draft", newDraftCmd(), []string{"decision-digest", "--last", "1"}},
		{"report", newReportCmd(), []string{"decision-digest"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.cmd.SilenceUsage = true
			tc.cmd.SilenceErrors = true
			tc.cmd.SetArgs(tc.args)
			err := tc.cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "invalid frontmatter") {
				t.Fatalf("error = %v, want invalid override error", err)
			}
		})
	}
}

func TestResolveReportSubjectsBestEffort(t *testing.T) {
	entries := []*ledger.Entry{{Workset: ledger.Workset{Commits: []string{"a", "b", "a"}}}}
	lookup := func(sha string) (string, error) {
		if sha == "b" {
			return "", errors.New("rewritten")
		}
		return "subject a", nil
	}
	subjects, resolved, unresolved := resolveReportSubjects(entries, lookup)
	if subjects["a"] != "subject a" || resolved != 1 || unresolved != 1 {
		t.Fatalf("subjects=%v resolved=%d unresolved=%d", subjects, resolved, unresolved)
	}
}

func TestReportDecisionDigestDefaultScopeAndStaleSHA(t *testing.T) {
	dir := newReportRepo(t)
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	sha := strings.TrimSpace(runReportGit(t, dir, "rev-parse", "HEAD"))
	entriesDir := filepath.Join(dir, ".timbers")
	for _, entry := range []*ledger.Entry{
		reportEntry("tb_2026-07-14T12:00:00Z_"+sha[:6], sha, "Initial report work", time.Now().Add(-time.Hour)),
		reportEntry("tb_2026-07-14T13:00:00Z_deadbe", "deadbeefdeadbeef", "Rewritten work", time.Now()),
	} {
		writeReportEntry(t, entriesDir, entry)
	}

	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"report", "decision-digest", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("report error = %v\n%s", err, buf.String())
	}
	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if result["status"] != "rendered" || result["entry_count"] != float64(2) {
		t.Fatalf("result = %#v", result)
	}
	provenance, ok := result["provenance"].(map[string]any)
	if !ok {
		t.Fatalf("provenance type = %T", result["provenance"])
	}
	if provenance["git_resolved"] != float64(1) || provenance["git_unresolved"] != float64(1) {
		t.Fatalf("provenance = %#v", provenance)
	}
	prompt, ok := result["prompt"].(string)
	if !ok {
		t.Fatalf("prompt type = %T", result["prompt"])
	}
	if strings.Contains(prompt, `"how"`) || strings.Contains(prompt, `"diffstat"`) {
		t.Fatalf("prompt contains full entry fields: %s", prompt)
	}
}

func TestReportQuietWithNoEntries(t *testing.T) {
	dir := newReportRepo(t)
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".timbers"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"report", "decision-digest", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("report error = %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), `"status": "quiet"`) || !strings.Contains(buf.String(), `"reason": "no_entries"`) {
		t.Fatalf("quiet output = %s", buf.String())
	}
}

func TestReportFailsClosedOnCorruptEntry(t *testing.T) {
	repo := t.TempDir()
	runReportGit(t, repo, "init", "-q")
	badPath := filepath.Join(repo, ".timbers", "bad.json")
	if err := os.MkdirAll(filepath.Dir(badPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badPath, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}

	cmd := newReportCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"decision-digest"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), badPath) {
		t.Fatalf("error = %v, want corrupt path %q", err, badPath)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no partial report", stdout.String())
	}
}

func TestReportModelExecutionAndSemanticQuiet(t *testing.T) {
	tests := []struct {
		name, content, status, reason string
	}{
		{"generated", "# Decision Digest\n\nA decision.", "generated", ""},
		{"quiet", "_No explicit design decisions in this range._", "quiet", "no_reportable_content"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := newReportRepo(t)
			oldDir, _ := os.Getwd()
			defer func() { _ = os.Chdir(oldDir) }()
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			sha := strings.TrimSpace(runReportGit(t, dir, "rev-parse", "HEAD"))
			writeReportEntry(t, filepath.Join(dir, ".timbers"),
				reportEntry("tb_2026-07-14T12:00:00Z_"+sha[:6], sha, "Initial report work", time.Now()))

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}]}`, tt.content)
			}))
			defer server.Close()
			t.Setenv("LOCAL_LLM_URL", server.URL)

			cmd := newRootCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs([]string{"report", "decision-digest", "--model", "local", "--json"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("report error = %v\n%s", err, buf.String())
			}
			var result map[string]any
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
			}
			if result["status"] != tt.status {
				t.Fatalf("status = %v, want %s: %#v", result["status"], tt.status, result)
			}
			if tt.reason != "" && result["reason"] != tt.reason {
				t.Fatalf("reason = %v, want %s", result["reason"], tt.reason)
			}
		})
	}
}

func newReportRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runReportGit(t, dir, "init", "-q")
	runReportGit(t, dir, "config", "user.email", "test@example.com")
	runReportGit(t, dir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runReportGit(t, dir, "add", "file.txt")
	runReportGit(t, dir, "commit", "-q", "-m", "Initial report work")
	return dir
}

func runReportGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}

func reportEntry(id, sha, what string, created time.Time) *ledger.Entry {
	return &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        id,
		CreatedAt: created, UpdatedAt: created,
		Workset: ledger.Workset{
			AnchorCommit: sha,
			Commits:      []string{sha},
			Diffstat:     &ledger.Diffstat{Files: 1},
		},
		Summary: ledger.Summary{What: what, Why: "Need the report", How: "Implementation detail"},
		Notes:   "Chose the small path",
	}
}

func writeReportEntry(t *testing.T, dir string, entry *ledger.Entry) {
	t.Helper()
	data, err := entry.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	entryDir := filepath.Join(dir, ledger.EntryDateDir(entry.ID))
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(entryDir, ledger.IDToFilename(entry.ID)+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
