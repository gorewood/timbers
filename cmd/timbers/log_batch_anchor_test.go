package main

import (
	"testing"

	"github.com/gorewood/timbers/internal/git"
)

// TestPickBatchAnchorWith exercises the mixed-topology cases that
// pickBatchAnchor was added to handle (Laura's v0.22.0 osprey-strike
// pathology). The pure-function variant lets us check topology
// behavior without spinning up a real git repo.
func TestPickBatchAnchorWith(t *testing.T) {
	tests := []struct {
		name            string
		commits         []git.Commit
		firstParentLine map[string]bool // SHA → on first-parent line?
		wantAnchor      string
	}{
		{
			name:       "empty group returns empty",
			commits:    nil,
			wantAnchor: "",
		},
		{
			name: "single commit on first-parent line returns it",
			commits: []git.Commit{
				{SHA: "mainwork1"},
			},
			firstParentLine: map[string]bool{"mainwork1": true},
			wantAnchor:      "mainwork1",
		},
		{
			name: "mixed: newest is local — returns newest",
			commits: []git.Commit{
				{SHA: "mainwork2"}, // newest, on first-parent line
				{SHA: "sidework1"}, // on side branch
			},
			firstParentLine: map[string]bool{
				"mainwork2": true,
				"sidework1": false,
			},
			wantAnchor: "mainwork2",
		},
		{
			name: "mixed: newest is side-branch — returns older local (the fix)",
			commits: []git.Commit{
				{SHA: "sidewrk2"}, // newest by walk order, but on side branch
				{SHA: "mainwrk1"}, // older but on first-parent line
			},
			firstParentLine: map[string]bool{
				"sidewrk2": false,
				"mainwrk1": true,
			},
			// Pre-fix bug would have returned "sidewrk2" (commits[0]).
			// The fix returns "mainwrk1" — the older but correctly-on-line commit.
			wantAnchor: "mainwrk1",
		},
		{
			name: "multiple side-branch + one local — returns the local",
			commits: []git.Commit{
				{SHA: "side1"},
				{SHA: "side2"},
				{SHA: "side3"},
				{SHA: "mainx"},
			},
			firstParentLine: map[string]bool{
				"side1": false,
				"side2": false,
				"side3": false,
				"mainx": true,
			},
			wantAnchor: "mainx",
		},
		{
			name: "pure cross-agent debt: nothing on first-parent line — falls back to commits[0]",
			commits: []git.Commit{
				{SHA: "side1"},
				{SHA: "side2"},
			},
			firstParentLine: map[string]bool{
				"side1": false,
				"side2": false,
			},
			// All commits arrived via merge from elsewhere. The
			// off-first-parent-line diagnostic shipped in v0.22.1
			// surfaces this case to the user via pending and doctor.
			wantAnchor: "side1",
		},
		{
			name: "Laura's exact pathology: PR-merged commits + own work, own work newest",
			commits: []git.Commit{
				// Walk order from GetPendingCommits is newest-first.
				// Laura's session commit was newest, but a PR's
				// merged-in commits also shared the Work-item trailer
				// and ended up in the same group.
				{SHA: "lauracom1"}, // her own work, first-parent line
				{SHA: "prcommit1"}, // PR #204 README commit, side branch
				{SHA: "prcommit2"}, // PR #204 README commit, side branch
				{SHA: "prcommit3"}, // PR #204 README commit, side branch
			},
			firstParentLine: map[string]bool{
				"lauracom1": true,
				"prcommit1": false,
				"prcommit2": false,
				"prcommit3": false,
			},
			wantAnchor: "lauracom1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isOnFirstParent := func(sha string) bool {
				return tt.firstParentLine[sha]
			}
			got := pickBatchAnchorWith(tt.commits, isOnFirstParent)
			if got != tt.wantAnchor {
				t.Errorf("pickBatchAnchorWith(%v) = %q, want %q", tt.commits, got, tt.wantAnchor)
			}
		})
	}
}
