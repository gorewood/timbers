package ledger

import (
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/git"
)

// TestExplainPending verifies that ExplainPending pairs each commit in the
// display range with the same classification the gate/display filter would
// apply, covering every Reason value classifyCommit can return. The keep
// case is essential because ExplainPending is the user-facing surface for
// debugging "why isn't this commit pending?" — a regression that drops the
// keep case (or mislabels a reason) silently misleads the user.
func TestExplainPending(t *testing.T) {
	const anchorSHA = "anchorsha12"
	const documentedSHA = "abc123def456" // 12 hex chars — meets the revert-trailer minimum width

	// docEntry pulls documentedSHA into the docSet so the "documented"
	// and "revert" reasons can be exercised. anchorEntry plays the role
	// of "latest" so pendingRange walks anchor..HEAD.
	docEntry := makeTestEntry(documentedSHA, time.Date(2026, 1, 14, 10, 0, 0, 0, time.UTC))
	anchorEntry := makeTestEntry(anchorSHA, time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))

	tests := []struct {
		name         string
		commit       git.Commit
		commitFiles  []string
		skipAuthors  []string
		skipMessages []string
		ackSHA       string
		wantReason   string
	}{
		{
			name:        "keep — undocumented code commit",
			commit:      git.Commit{SHA: "keepcommit1", Subject: "feat: new widget", ParentCount: 1},
			commitFiles: []string{"cmd/main.go"},
			wantReason:  "",
		},
		{
			name:        "infra — only matches skip rules",
			commit:      git.Commit{SHA: "infracommit", Subject: "chore: timbers entry", ParentCount: 1},
			commitFiles: []string{".timbers/2026/foo.json"},
			wantReason:  "infra",
		},
		{
			name: "author — matches skip-author glob",
			commit: git.Commit{
				SHA:         "botcommit01",
				Subject:     "deps: bump",
				Author:      "dependabot[bot]",
				AuthorEmail: "dependabot[bot]@users.noreply.github.com",
				ParentCount: 1,
			},
			commitFiles: []string{"go.mod"},
			skipAuthors: []string{"dependabot*"},
			wantReason:  "author",
		},
		{
			name:         "message — matches skip-message glob",
			commit:       git.Commit{SHA: "changelog01", Subject: "chore: changelog for v0.22.7", ParentCount: 1},
			commitFiles:  []string{"CHANGELOG.md"},
			skipMessages: []string{"chore: changelog for v*"},
			wantReason:   "message",
		},
		{
			name:        "documented — SHA appears in another entry's workset",
			commit:      git.Commit{SHA: documentedSHA, Subject: "feat: prior work", ParentCount: 1},
			commitFiles: []string{"cmd/main.go"},
			wantReason:  "documented",
		},
		{
			name:        "ack — SHA has an ack record",
			commit:      git.Commit{SHA: "ackedcommit", Subject: "chore: upstream sync", ParentCount: 1},
			commitFiles: []string{"vendor/dep.go"},
			ackSHA:      "ackedcommit",
			wantReason:  "ack",
		},
		{
			name: "revert — documented revert (referenced SHA in docSet)",
			commit: git.Commit{
				SHA:         "revertcommit",
				Subject:     `Revert "feat: prior work"`,
				Body:        "This reverts commit " + documentedSHA + ".",
				ParentCount: 1,
			},
			commitFiles: []string{"cmd/main.go"},
			wantReason:  "revert",
		},
		{
			name:        "merge-empty — merge commit with empty diff",
			commit:      git.Commit{SHA: "mergecommit", Subject: "Merge branch 'feature'", ParentCount: 2},
			commitFiles: nil, // empty file list triggers merge-empty in display path
			wantReason:  "merge-empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitOps()
			mock.headSHA = "headsha1234"
			mock.logCommits = []git.Commit{tt.commit}
			mock.commitFiles = map[string][]string{tt.commit.SHA: tt.commitFiles}

			store := newTestStorage(t, mock, docEntry, anchorEntry)
			store.skipAuthors = tt.skipAuthors
			store.skipMessages = tt.skipMessages
			if tt.ackSHA != "" {
				now := time.Now().UTC()
				ack := &Ack{
					Schema:    SchemaVersion,
					Kind:      KindAck,
					ID:        GenerateAckID(tt.ackSHA, now),
					AckedAt:   now,
					Acker:     Acker{Name: "Test", Email: "test@example.com"},
					TargetSHA: tt.ackSHA,
					Reason:    "test ack",
				}
				if err := store.WriteAck(ack); err != nil {
					t.Fatalf("WriteAck: %v", err)
				}
			}

			classified, latest, err := store.ExplainPending()
			if err != nil {
				t.Fatalf("ExplainPending: %v", err)
			}
			if latest == nil || latest.Workset.AnchorCommit != anchorSHA {
				t.Fatalf("latest entry mismatch: got %+v, want anchor %q", latest, anchorSHA)
			}
			if len(classified) != 1 {
				t.Fatalf("expected 1 classified commit, got %d", len(classified))
			}
			got := classified[0]
			if got.Commit.SHA != tt.commit.SHA {
				t.Errorf("commit.SHA = %q, want %q", got.Commit.SHA, tt.commit.SHA)
			}
			if got.Reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", got.Reason, tt.wantReason)
			}
		})
	}
}

// TestExplainPending_PairsEveryCommit confirms the function returns one
// ClassifiedCommit per input commit in order, so kept and skipped commits
// are both surfaced for the explain UI. A regression that drops skipped
// commits would defeat the whole point of --explain.
func TestExplainPending_PairsEveryCommit(t *testing.T) {
	mock := newMockGitOps()
	mock.headSHA = "headsha1234"
	mock.logCommits = []git.Commit{
		{SHA: "keep0000001", Subject: "feat: a", ParentCount: 1},
		{SHA: "skip0000001", Subject: "chore: changelog for v1.0.0", ParentCount: 1},
		{SHA: "keep0000002", Subject: "feat: b", ParentCount: 1},
	}
	mock.commitFiles = map[string][]string{
		"keep0000001": {"cmd/a.go"},
		"skip0000001": {"CHANGELOG.md"},
		"keep0000002": {"cmd/b.go"},
	}

	anchor := makeTestEntry("anchorsha12", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := newTestStorage(t, mock, anchor)
	store.skipMessages = []string{"chore: changelog for v*"}

	classified, _, err := store.ExplainPending()
	if err != nil {
		t.Fatalf("ExplainPending: %v", err)
	}
	if len(classified) != 3 {
		t.Fatalf("expected 3 classified commits, got %d", len(classified))
	}
	wantReasons := []string{"", "message", ""}
	for i, want := range wantReasons {
		if classified[i].Reason != want {
			t.Errorf("classified[%d].Reason = %q, want %q (SHA %s)",
				i, classified[i].Reason, want, classified[i].Commit.SHA)
		}
	}
}
