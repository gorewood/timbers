package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorewood/timbers/internal/setup"
)

// writePostRewriteHook writes hookContent to .git/hooks/post-rewrite under dir.
func writePostRewriteHook(t *testing.T, dir, hookContent string) string {
	t.Helper()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	path := filepath.Join(hooksDir, "post-rewrite")
	if err := os.WriteFile(path, []byte(hookContent), 0o600); err != nil {
		t.Fatalf("write hook: %v", err)
	}
	return path
}

// TestCheckPostRewriteHookDrift verifies the doctor check reports drift between
// an installed post-rewrite section and the current generator output, and
// regenerates it under --fix.
func TestCheckPostRewriteHookDrift(t *testing.T) {
	currentSection := postRewriteTimbersSection()
	delimited := func(section string) string {
		return "#!/bin/sh\n# --- timbers section (do not edit) ---\n" + section + "# --- end timbers section ---\n"
	}

	t.Run("not installed is an informational pass", func(t *testing.T) {
		dir := t.TempDir()
		runGit(t, dir, "init")
		runInDir(t, dir, func() {
			res := checkPostRewriteHookDrift(&doctorFlags{})
			if res.Status != checkPass {
				t.Errorf("status = %v, want pass", res.Status)
			}
			if !strings.Contains(res.Message, "not installed") {
				t.Errorf("message = %q, want 'not installed'", res.Message)
			}
		})
	})

	t.Run("up-to-date section passes", func(t *testing.T) {
		dir := t.TempDir()
		runGit(t, dir, "init")
		writePostRewriteHook(t, dir, delimited(currentSection))
		runInDir(t, dir, func() {
			res := checkPostRewriteHookDrift(&doctorFlags{})
			if res.Status != checkPass {
				t.Errorf("status = %v, want pass", res.Status)
			}
			if !strings.Contains(res.Message, "up to date") {
				t.Errorf("message = %q, want 'up to date'", res.Message)
			}
		})
	})

	t.Run("drifted section warns without fix", func(t *testing.T) {
		dir := t.TempDir()
		runGit(t, dir, "init")
		writePostRewriteHook(t, dir, delimited("# old timbers post-rewrite hook\necho stale\n"))
		runInDir(t, dir, func() {
			res := checkPostRewriteHookDrift(&doctorFlags{})
			if res.Status != checkWarn {
				t.Errorf("status = %v, want warn", res.Status)
			}
			if !strings.Contains(res.Message, "outdated") {
				t.Errorf("message = %q, want 'outdated'", res.Message)
			}
		})
	})

	t.Run("drifted section regenerated under --fix", func(t *testing.T) {
		dir := t.TempDir()
		runGit(t, dir, "init")
		path := writePostRewriteHook(t, dir, delimited("# old timbers post-rewrite hook\necho stale\n"))
		runInDir(t, dir, func() {
			res := checkPostRewriteHookDrift(&doctorFlags{fix: true})
			if res.Status != checkPass {
				t.Errorf("status = %v, want pass; msg=%q", res.Status, res.Message)
			}
			if !setup.SectionUpToDate(path, currentSection) {
				t.Error("hook should be up to date after --fix")
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(got), "echo stale") {
				t.Error("stale section content should be gone after --fix")
			}
		})
	})
}
