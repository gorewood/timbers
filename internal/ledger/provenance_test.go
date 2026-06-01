package ledger

import (
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/git"
)

// TestClassifyByProvenance covers the pure-function behavior of the
// cross-agent debt classifier: every combination of email vs staleness,
// plus the safe-degradation paths (empty UserEmail, zero StaleWindow,
// zero CommitDate, future-dated CommitDate).
func TestClassifyByProvenance(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-2 * time.Hour)
	stale := now.Add(-48 * time.Hour)
	future := now.Add(2 * time.Hour)

	cases := []struct {
		name       string
		commit     git.Commit
		cfg        ProvenanceConfig
		wantReason string
	}{
		{
			name:       "disabled config returns kept",
			commit:     git.Commit{AuthorEmail: "anyone@elsewhere.test", CommitDate: stale},
			cfg:        ProvenanceConfig{},
			wantReason: "",
		},
		{
			name:       "fresh commit by user is in-session",
			commit:     git.Commit{AuthorEmail: "me@example.com", CommitDate: fresh},
			cfg:        ProvenanceConfig{UserEmail: "me@example.com", StaleWindow: 24 * time.Hour, Now: now},
			wantReason: "",
		},
		{
			name:       "fresh commit by someone else is foreign-author",
			commit:     git.Commit{AuthorEmail: "bot@example.com", CommitDate: fresh},
			cfg:        ProvenanceConfig{UserEmail: "me@example.com", StaleWindow: 24 * time.Hour, Now: now},
			wantReason: reasonForeignAuthor,
		},
		{
			name:       "stale commit by user is stale (signal lost)",
			commit:     git.Commit{AuthorEmail: "me@example.com", CommitDate: stale},
			cfg:        ProvenanceConfig{UserEmail: "me@example.com", StaleWindow: 24 * time.Hour, Now: now},
			wantReason: reasonStale,
		},
		{
			name:       "stale commit by someone else is composite",
			commit:     git.Commit{AuthorEmail: "bot@example.com", CommitDate: stale},
			cfg:        ProvenanceConfig{UserEmail: "me@example.com", StaleWindow: 24 * time.Hour, Now: now},
			wantReason: reasonForeignAuthorStale,
		},
		{
			name:       "empty UserEmail disables email check (safe fallback for unset user.email)",
			commit:     git.Commit{AuthorEmail: "stranger@example.com", CommitDate: fresh},
			cfg:        ProvenanceConfig{UserEmail: "", StaleWindow: 24 * time.Hour, Now: now},
			wantReason: "",
		},
		{
			name:       "zero StaleWindow disables staleness check",
			commit:     git.Commit{AuthorEmail: "me@example.com", CommitDate: stale},
			cfg:        ProvenanceConfig{UserEmail: "me@example.com", StaleWindow: 0, Now: now},
			wantReason: "",
		},
		{
			name:       "zero CommitDate is not stale (defensive fallback)",
			commit:     git.Commit{AuthorEmail: "me@example.com"},
			cfg:        ProvenanceConfig{UserEmail: "me@example.com", StaleWindow: 24 * time.Hour, Now: now},
			wantReason: "",
		},
		{
			name:       "future CommitDate (clock skew) is not stale",
			commit:     git.Commit{AuthorEmail: "me@example.com", CommitDate: future},
			cfg:        ProvenanceConfig{UserEmail: "me@example.com", StaleWindow: 24 * time.Hour, Now: now},
			wantReason: "",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := classifyByProvenance(testCase.commit, testCase.cfg)
			if got != testCase.wantReason {
				t.Errorf("classifyByProvenance = %q, want %q", got, testCase.wantReason)
			}
		})
	}
}

// TestClassifyCommit_ProvenanceIsLastInChain locks in that earlier reasons
// (infra, identity, content) take precedence over provenance. A documented
// or acked commit must not relabel as foreign-author just because its
// author email differs — those reasons carry decision-relevance the
// operator cares about more than provenance. Plan acceptance criteria
// item: "foreign-author + documented → documented (not foreign-author)".
func TestClassifyCommit_ProvenanceIsLastInChain(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-48 * time.Hour)

	provenance := ProvenanceConfig{
		UserEmail:   "me@example.com",
		StaleWindow: 24 * time.Hour,
		Now:         now,
	}
	foreignCommit := git.Commit{SHA: "foreign01", AuthorEmail: "stranger@example.com", CommitDate: stale}

	cases := []struct {
		name       string
		fileMap    map[string][]string
		docSet     map[string]bool
		ackedSet   map[string]bool
		wantReason string
	}{
		{
			name:       "foreign-author + infra-only files → infra wins",
			fileMap:    map[string][]string{"foreign01": {".timbers/2026/foo.json"}},
			wantReason: "infra",
		},
		{
			name:       "foreign-author + documented SHA → documented wins",
			fileMap:    map[string][]string{"foreign01": {"cmd/main.go"}},
			docSet:     map[string]bool{"foreign01": true},
			wantReason: "documented",
		},
		{
			name:       "foreign-author + acked SHA → ack wins",
			fileMap:    map[string][]string{"foreign01": {"cmd/main.go"}},
			ackedSet:   map[string]bool{"foreign01": true},
			wantReason: "ack",
		},
		{
			name:       "foreign-author + no earlier match → falls through to provenance",
			fileMap:    map[string][]string{"foreign01": {"cmd/main.go"}},
			wantReason: reasonForeignAuthorStale,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			store := newTestStorage(t, newMockGitOps())
			store.provenance = provenance
			got := store.classifyCommit(foreignCommit, testCase.fileMap, testCase.docSet, testCase.ackedSet, false)
			if got != testCase.wantReason {
				t.Errorf("classifyCommit = %q, want %q (provenance must be LAST in chain)",
					got, testCase.wantReason)
			}
		})
	}
}
