package ledger

import (
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/git"
)

// TestGetGatePendingCommits_ProvenanceMatrix is the end-to-end gate
// regression for v0.23.0. Walks the four provenance buckets against the
// gate path (GetGatePendingCommits → filterCommits → filterByRules with
// gateStrict=true) and asserts each commit lands in the expected place.
//
// The acceptance criteria from the plan that this test locks in:
//   - foreign-author commits do NOT block the gate (silent skip)
//   - stale commits do NOT block the gate (silent skip)
//   - in-session commits DO block the gate (the only correct case)
func TestGetGatePendingCommits_ProvenanceMatrix(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-1 * time.Hour)
	stale := now.Add(-48 * time.Hour)
	const meEmail = "me@example.com"

	mock := newMockGitOps()
	mock.headSHA = "head00000001"
	mock.firstParentCommits = []git.Commit{
		{SHA: "in-session1", AuthorEmail: meEmail, CommitDate: fresh, ParentCount: 1},
		{SHA: "foreign0001", AuthorEmail: "bot@example.com", CommitDate: fresh, ParentCount: 1},
		{SHA: "stale-self1", AuthorEmail: meEmail, CommitDate: stale, ParentCount: 1},
		{SHA: "foreign-old", AuthorEmail: "bot@example.com", CommitDate: stale, ParentCount: 1},
	}
	mock.commitFiles = map[string][]string{
		"in-session1": {"cmd/main.go"},
		"foreign0001": {"cmd/main.go"},
		"stale-self1": {"cmd/main.go"},
		"foreign-old": {"cmd/main.go"},
	}

	anchor := makeTestEntry("anchorsha12", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	store := newTestStorage(t, mock, anchor)
	store.SetProvenance(ProvenanceConfig{
		UserEmail:   meEmail,
		StaleWindow: 24 * time.Hour,
		Now:         now,
	})

	commits, _, err := store.GetGatePendingCommits()
	if err != nil {
		t.Fatalf("GetGatePendingCommits: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 gate-blocking commit (in-session only), got %d: %+v",
			len(commits), commits)
	}
	if commits[0].SHA != "in-session1" {
		t.Errorf("commits[0].SHA = %q, want %q (only the recent same-author commit should block)",
			commits[0].SHA, "in-session1")
	}
}

// TestCountInfraSkippedSinceLatest_IncludesProvenance asserts that the
// status surface ('timbers status' housekeeping-skipped tally) counts
// commits dropped by the provenance classifier, not just infra and
// identity skips. Without this, an operator looking at the status would
// see N pending and 0 skipped while the gate silently passes N commits —
// the same misreporting failure mode v0.22.7 fixed for msg: rules.
func TestCountInfraSkippedSinceLatest_IncludesProvenance(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-1 * time.Hour)
	stale := now.Add(-48 * time.Hour)
	const meEmail = "me@example.com"

	mock := newMockGitOps()
	mock.headSHA = "head00000001"
	mock.logCommits = []git.Commit{
		{SHA: "real0000001", AuthorEmail: meEmail, CommitDate: fresh},
		{SHA: "foreign0001", AuthorEmail: "bot@example.com", CommitDate: fresh},
		{SHA: "stale-self1", AuthorEmail: meEmail, CommitDate: stale},
	}
	mock.commitFiles = map[string][]string{
		"real0000001": {"cmd/main.go"},
		"foreign0001": {"cmd/main.go"},
		"stale-self1": {"cmd/main.go"},
	}

	anchor := makeTestEntry("anchorsha12", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	store := newTestStorage(t, mock, anchor)
	store.SetProvenance(ProvenanceConfig{
		UserEmail:   meEmail,
		StaleWindow: 24 * time.Hour,
		Now:         now,
	})

	got, err := store.CountInfraSkippedSinceLatest()
	if err != nil {
		t.Fatalf("CountInfraSkippedSinceLatest: %v", err)
	}
	if got != 2 {
		t.Errorf("count = %d, want 2 (one foreign-author + one stale-self both auto-skip)", got)
	}
}
