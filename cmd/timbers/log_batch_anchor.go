// Package main provides the entry point for the timbers CLI.
package main

import "github.com/gorewood/timbers/internal/git"

// pickBatchAnchor selects the entry anchor for a batch commit group.
// Prefers the newest commit on HEAD's first-parent line (the "spine" of
// this branch). Falls back to commits[0] only when no commit in the
// group is on the first-parent line — the pure-cross-agent-debt case
// where every commit in the group arrived via merge from elsewhere.
//
// Background: previously this function was inline as `group.commits[0].SHA`,
// which silently selected whichever commit landed first in the group's
// slice. When a Work-item trailer grouped local commits with merged-in
// side-branch commits, the anchor could land on a side-branch SHA — and
// once an entry was anchored to a side-branch commit, the downstream
// linear-anchor assumptions in pending detection broke in confusing
// ways (the v0.22.0 osprey-strike friction reported by Laura).
//
// HEAD is resolved here (best-effort) rather than threaded through the
// caller chain; if git.HEAD() fails, we degrade to the legacy behavior
// of returning commits[0] rather than failing the batch run.
func pickBatchAnchor(commits []git.Commit) string {
	if len(commits) == 0 {
		return ""
	}
	head, err := git.HEAD()
	if err != nil || head == "" {
		return commits[0].SHA
	}
	for _, commit := range commits {
		if git.IsOnFirstParentLine(commit.SHA, head) {
			return commit.SHA
		}
	}
	// Pure cross-agent debt: no commit in the group is on this branch's
	// first-parent line. Fall back to the legacy behavior; the
	// off-first-parent-line diagnostic (shipped in v0.22.1) surfaces
	// the situation to the user via pending and doctor output.
	return commits[0].SHA
}
