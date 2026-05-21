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
	return pickBatchAnchorWith(commits, func(sha string) bool {
		return git.IsOnFirstParentLine(sha, head)
	})
}

// pickBatchAnchorWith is the pure-function core of pickBatchAnchor —
// dependency on git is injected so the topology behavior is unit-testable
// without spinning up a real repo. Returns the first commit whose SHA
// satisfies isOnFirstParent, or commits[0] if none qualify. Empty input
// returns "".
//
// Group ordering invariant: callers pass newest-first slices (mirrors
// git log's reverse-chronological output via getBatchCommits →
// GetPendingCommits). Combined with "first match wins," this means the
// returned SHA is the NEWEST commit on the first-parent line — the
// correct anchor semantics for a batch entry covering a range of work.
func pickBatchAnchorWith(commits []git.Commit, isOnFirstParent func(sha string) bool) string {
	if len(commits) == 0 {
		return ""
	}
	for _, commit := range commits {
		if isOnFirstParent(commit.SHA) {
			return commit.SHA
		}
	}
	// Pure cross-agent debt: no commit in the group is on this branch's
	// first-parent line. Fall back to the legacy behavior; the
	// off-first-parent-line diagnostic (shipped in v0.22.1) surfaces
	// the situation to the user via pending and doctor output.
	return commits[0].SHA
}
