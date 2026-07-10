package main

import (
	"os"
	"path/filepath"
	"testing"
)

// installBlockingPreCommitHook writes a pre-commit hook that mimics timbers'
// cross-agent-debt gate: it blocks every commit UNLESS
// TIMBERS_SKIP_CROSS_AGENT_DEBT=1 is present in the hook's environment. This
// exercises the exact propagation path that matters — a git pre-commit hook
// runs as a child of `git commit` and inherits that process's environment — so
// the test proves whether `timbers log`'s own entry commit carries the
// exemption, without needing the timbers binary on PATH.
func installBlockingPreCommitHook(t *testing.T, dir string) {
	t.Helper()
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	script := "#!/bin/sh\n" +
		"[ \"$TIMBERS_SKIP_CROSS_AGENT_DEBT\" = \"1\" ] && exit 0\n" +
		"echo 'gate: undocumented commits exist' >&2\n" +
		"exit 1\n"
	// #nosec G306 -- git hook needs execute permission
	if err := os.WriteFile(hookPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write pre-commit hook: %v", err)
	}
}

// TestLogEntryCommitBypassesDebtGate is the regression for the "failed to commit
// entry file" defect: `timbers log` committed its entry via a plain `git commit`
// that fired the pre-commit gate, which in a parallel-agent repo blocked on
// sibling debt — so logging (the act of paying the debt) itself failed. The
// entry commit must carry TIMBERS_SKIP_CROSS_AGENT_DEBT so the timbers gate
// stands down for the one commit that documents work.
func TestLogEntryCommitBypassesDebtGate(t *testing.T) {
	dir := newLogAnchorRepo(t)
	installBlockingPreCommitHook(t, dir)

	out, err := runLogCmd(t, dir, "documented work",
		"--why", "sibling debt must not block the documenting commit",
		"--how", "entry commit self-exempts the debt gate")
	if err != nil {
		t.Fatalf("timbers log failed under a debt-gating pre-commit hook: %v\noutput: %s", err, out)
	}

	if countJSONFilesInDir(filepath.Join(dir, ".timbers")) != 1 {
		t.Errorf("expected exactly one entry to be written and committed")
	}
}
