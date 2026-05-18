// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
)

const postCommitReminder = "[timbers] document this commit"

// hookRepo bundles a temp git repo wired up with .timbers/ and one anchor
// entry so HasPendingCommits has a baseline to compare against. Tests then
// make additional commits and assert what the post-commit hook prints.
type hookRepo struct {
	dir       string
	anchorSHA string
}

// seedFile is an extra file written into the seed commit before the anchor
// entry is captured. Tests use this to bake .timbersignore (or other repo
// configuration) into the baseline so it doesn't show up as actionable
// pending work later.
type seedFile struct {
	relPath string
	content string
}

func newHookRepo(t *testing.T, extraSeeds ...seedFile) *hookRepo {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test User")

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("seed\n"), 0o600); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	runGit(t, dir, "add", "README.md")
	for _, seed := range extraSeeds {
		full := filepath.Join(dir, seed.relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir seed %s: %v", seed.relPath, err)
		}
		if err := os.WriteFile(full, []byte(seed.content), 0o600); err != nil {
			t.Fatalf("write seed %s: %v", seed.relPath, err)
		}
		runGit(t, dir, "add", seed.relPath)
	}
	runGit(t, dir, "commit", "-m", "initial")

	anchor := strings.TrimSpace(runGitOutput(t, dir, "rev-parse", "HEAD"))

	// Write a single anchor entry so the ledger has a starting point. Without
	// this, HasPendingCommits returns false (no entries = no nagging) and
	// post-commit tests can't tell the difference between the bug and a fresh
	// repo.
	entry := makePrimeTestEntry(anchor, time.Now().UTC(), "seed entry")
	entryDir := filepath.Join(dir, ".timbers", ledger.EntryDateDir(entry.ID))
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("mkdir entry: %v", err)
	}
	data, err := entry.ToJSON()
	if err != nil {
		t.Fatalf("marshal entry: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, entry.ID+".json"), data, 0o600); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	return &hookRepo{dir: dir, anchorSHA: anchor}
}

// commitFile stages and commits a single file with the given content.
func (r *hookRepo) commitFile(t *testing.T, relPath, content, msg string) {
	t.Helper()
	full := filepath.Join(r.dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
	runGit(t, r.dir, "add", relPath)
	runGit(t, r.dir, "commit", "-m", msg)
}

// runHook invokes `timbers hook run <name>` against the repo and returns the
// combined stdout/stderr output and the command error.
func (r *hookRepo) runHook(t *testing.T, name string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	var execErr error
	runInDir(t, r.dir, func() {
		cmd := newRootCmd()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"hook", "run", name})
		execErr = cmd.Execute()
	})
	return buf.String(), execErr
}

func TestPostCommitHookGating(t *testing.T) {
	// Pre-state: any infrastructure-only commit (matched by built-in skip
	// rules or .timbersignore) MUST NOT trigger the post-commit reminder,
	// because timbers pending will rightly report it as non-actionable.
	// A regression of this contract was the v0.20.0 bug we're guarding.
	tests := []struct {
		name        string
		seeds       []seedFile
		setup       func(t *testing.T, r *hookRepo)
		wantPrinted bool
	}{
		{
			name: "beads-only commit is not actionable",
			setup: func(t *testing.T, r *hookRepo) {
				r.commitFile(t, ".beads/issues.jsonl", "{\"_type\":\"issue\"}\n", "beads sync")
			},
			wantPrinted: false,
		},
		{
			name: "timbers ledger-only commit is not actionable",
			setup: func(t *testing.T, r *hookRepo) {
				r.commitFile(t, ".timbers/2026/05/10/tb_test.json", "{}\n", "timbers entry")
			},
			wantPrinted: false,
		},
		{
			name: "lockfile-only commit is not actionable",
			setup: func(t *testing.T, r *hookRepo) {
				r.commitFile(t, "package-lock.json", "{}\n", "chore: bump deps")
			},
			wantPrinted: false,
		},
		{
			name:  "timbersignore-skipped commit is not actionable",
			seeds: []seedFile{{relPath: ".timbersignore", content: "vendor/\n"}},
			setup: func(t *testing.T, r *hookRepo) {
				r.commitFile(t, "vendor/lib.go", "package vendor\n", "vendor update")
			},
			wantPrinted: false,
		},
		{
			name: "substantive code commit is actionable",
			setup: func(t *testing.T, r *hookRepo) {
				r.commitFile(t, "internal/feature.go", "package internal\n", "feat: new code")
			},
			wantPrinted: true,
		},
		{
			name: "no .timbers/ directory means hook stays silent",
			setup: func(t *testing.T, r *hookRepo) {
				if err := os.RemoveAll(filepath.Join(r.dir, ".timbers")); err != nil {
					t.Fatalf("remove .timbers: %v", err)
				}
				r.commitFile(t, "internal/feature.go", "package internal\n", "feat: would-be actionable")
			},
			wantPrinted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newHookRepo(t, tt.seeds...)
			tt.setup(t, repo)

			out, err := repo.runHook(t, "post-commit")
			if err != nil {
				t.Fatalf("post-commit hook errored: %v\noutput: %s", err, out)
			}

			gotPrinted := strings.Contains(out, postCommitReminder)
			if gotPrinted != tt.wantPrinted {
				t.Errorf("post-commit reminder printed=%v, want %v\noutput: %s",
					gotPrinted, tt.wantPrinted, out)
			}
		})
	}
}

// TestPreCommitHookGating verifies the pre-commit blocker uses the same
// definition of "actionable pending" as the post-commit reminder. They must
// agree — otherwise agents see contradictory signals (blocked but no nudge,
// or nudge but no block).
func TestPreCommitHookGating(t *testing.T) {
	t.Run("infrastructure-only commit does not block subsequent commit", func(t *testing.T) {
		repo := newHookRepo(t)
		repo.commitFile(t, ".beads/issues.jsonl", "{}\n", "beads sync")

		out, err := repo.runHook(t, "pre-commit")
		if err != nil {
			t.Fatalf("pre-commit unexpectedly errored on infra-only history: %v\noutput: %s", err, out)
		}
		if strings.Contains(out, "Commit blocked") {
			t.Errorf("pre-commit blocked on infra-only history; output: %s", out)
		}
	})

	t.Run("substantive commit blocks subsequent commit", func(t *testing.T) {
		repo := newHookRepo(t)
		repo.commitFile(t, "internal/feature.go", "package internal\n", "feat: new code")

		out, err := repo.runHook(t, "pre-commit")
		if err == nil {
			t.Fatalf("pre-commit expected to error on undocumented work; got nil\noutput: %s", out)
		}
		if !strings.Contains(out, "Commit blocked") {
			t.Errorf("pre-commit did not announce block; output: %s", out)
		}
	})

	// TIMBERS_SKIP_CROSS_AGENT_DEBT escape hatch: when set, the gate must
	// stand down even if there is undocumented work on the current branch.
	// Intended for parallel-agent flows where one agent will run timbers
	// catchup later; not a replacement for documenting work.
	t.Run("env var bypasses the gate", func(t *testing.T) {
		repo := newHookRepo(t)
		repo.commitFile(t, "internal/feature.go", "package internal\n", "feat: new code")

		t.Setenv("TIMBERS_SKIP_CROSS_AGENT_DEBT", "1")
		out, err := repo.runHook(t, "pre-commit")
		if err != nil {
			t.Fatalf("pre-commit must not error when env var is set: %v\noutput: %s", err, out)
		}
		if strings.Contains(out, "Commit blocked") {
			t.Errorf("pre-commit blocked despite env var; output: %s", out)
		}
	})

	t.Run("env var accepts case-insensitive truthy values", func(t *testing.T) {
		for _, val := range []string{"true", "YES", "On", "1"} {
			t.Run(val, func(t *testing.T) {
				repo := newHookRepo(t)
				repo.commitFile(t, "internal/feature.go", "package internal\n", "feat: new code")
				t.Setenv("TIMBERS_SKIP_CROSS_AGENT_DEBT", val)
				out, err := repo.runHook(t, "pre-commit")
				if err != nil || strings.Contains(out, "Commit blocked") {
					t.Errorf("pre-commit must bypass for %q; err=%v output=%s", val, err, out)
				}
			})
		}
	})

	t.Run("env var with falsy value still blocks", func(t *testing.T) {
		repo := newHookRepo(t)
		repo.commitFile(t, "internal/feature.go", "package internal\n", "feat: new code")

		t.Setenv("TIMBERS_SKIP_CROSS_AGENT_DEBT", "0")
		out, err := repo.runHook(t, "pre-commit")
		if err == nil {
			t.Fatalf("pre-commit must still block with TIMBERS_SKIP_CROSS_AGENT_DEBT=0; output: %s", out)
		}
		if !strings.Contains(out, "Commit blocked") {
			t.Errorf("expected block message; got: %s", out)
		}
	})
}

// TestPreCommitHookGating_SiblingMerge is the end-to-end regression for the
// parallel-agent scenario: agent B authored undocumented commits on branch Y,
// agent A is on branch X and merges Y in. Before the first-parent fix, the
// gate fired on B's commits even though A had no work to document. With the
// fix, A's gate stays silent — B owns B's documentation.
func TestPreCommitHookGating_SiblingMerge(t *testing.T) {
	repo := newHookRepo(t)

	// Branch X is the current branch (where A is committing). Branch Y
	// receives B's undocumented work, then gets merged back into X.
	currentBranch := strings.TrimSpace(runGitOutput(t, repo.dir, "rev-parse", "--abbrev-ref", "HEAD"))

	// Create branch Y from the anchor and add two undocumented code commits.
	runGit(t, repo.dir, "checkout", "-b", "branch-y")
	repo.commitFile(t, "frontend/app.tsx", "// frontend agent work\n", "fix(web): v6.x first pass")
	repo.commitFile(t, "frontend/lib.ts", "// more frontend agent work\n", "fix(web): v6.x second pass")

	// Back to branch X and merge Y in (no-ff to guarantee a merge commit).
	runGit(t, repo.dir, "checkout", currentBranch)
	runGit(t, repo.dir, "merge", "--no-ff", "-m", "Merge branch-y", "branch-y")

	out, err := repo.runHook(t, "pre-commit")
	if err != nil {
		t.Fatalf("pre-commit must not block on sibling-branch debt brought in via merge: %v\noutput: %s", err, out)
	}
	if strings.Contains(out, "Commit blocked") {
		t.Errorf("pre-commit blocked despite first-parent scope; output: %s", out)
	}
}
