package ledger

import (
	"fmt"

	"github.com/gorewood/timbers/internal/git"
)

// ClassifiedCommit pairs a commit with its pending classification: an empty
// Reason means the commit is kept (counts as pending); a non-empty Reason is
// the skip reason (infra/author/message/documented/ack/revert/merge-empty/empty).
type ClassifiedCommit struct {
	Commit git.Commit
	Reason string
}

// ExplainPending returns every commit in the display pending range (the same
// range `timbers pending` shows) paired with its keep/skip classification.
// Unlike GetPendingCommits, which drops skipped commits, this keeps them so
// callers can see *why* each is or isn't pending — e.g. verifying that a new
// .timbersignore author:/msg: rule actually exempts a commit.
func (s *Storage) ExplainPending() ([]ClassifiedCommit, *Entry, error) {
	commits, latest, docSet, ackedSet, err := s.pendingRange(false)
	if commits == nil {
		return nil, latest, err
	}
	fileMap, ferr := s.git.CommitFilesMulti(commitSHAs(commits))
	if ferr != nil {
		fileMap = map[string][]string{} // degrade: classify without file data
	}
	out := make([]ClassifiedCommit, 0, len(commits))
	for _, c := range commits {
		out = append(out, ClassifiedCommit{
			Commit: c,
			Reason: s.classifyCommit(c, fileMap, docSet, ackedSet, false),
		})
	}
	return out, latest, err
}

// pendingRange resolves the raw (unfiltered) commit range to consider for
// pending detection, plus the documented/acked sets used to classify or
// filter it. Shared by getPendingCommits (which filters) and ExplainPending
// (which classifies). Returns the wrapped ErrStaleAnchor when the anchor was
// GC'd (commits is then the all-reachable fallback). On a hard git failure,
// commits is nil and err is the underlying error.
//
// One disk scan per call: ListEntries feeds both `latest` and the documented-
// SHA set; AckedSet is a parallel scan. Both are built once and returned so a
// single pending check sees one consistent snapshot.
func (s *Storage) pendingRange(firstParent bool) (commits []git.Commit, latest *Entry, docSet, ackedSet map[string]bool, err error) {
	head, headErr := s.git.HEAD()
	if headErr != nil {
		return nil, nil, nil, nil, headErr
	}
	entries, listErr := s.ListEntries()
	if listErr != nil {
		return nil, nil, nil, nil, listErr
	}
	latest = latestEntry(entries)
	docSet = documentedSHASetFromEntries(entries)
	ackedSet = s.AckedSet()

	// No entries yet — all reachable commits, nil latest. Display callers
	// (pending, doctor) check latest == nil to show friendly messaging.
	if latest == nil {
		return s.reachableFallback(head, nil, docSet, ackedSet, nil)
	}

	anchor := latest.Workset.AnchorCommit
	staleErr := fmt.Errorf("%w: %s", ErrStaleAnchor, anchor)

	// Short-circuit 1 — stale anchor (squash/rebase GC'd the SHA): fall back
	// to all-reachable and wrap ErrStaleAnchor so display callers surface it.
	if !s.git.IsAncestorOf(anchor, head) {
		return s.reachableFallback(head, latest, docSet, ackedSet, staleErr)
	}

	// Short-circuit 2 — off-first-parent anchor in gate path: anchor is
	// reachable but not on HEAD's first-parent line (latest entry authored on
	// a merged-in side branch). LogFirstParent(anchor, head) would walk a
	// structurally weird range; fall back to all-reachable (no ErrStaleAnchor).
	if firstParent && !s.git.IsOnFirstParentLine(anchor, head) {
		return s.reachableFallback(head, latest, docSet, ackedSet, nil)
	}

	// Normal path: commits from anchor (exclusive) to HEAD (inclusive). If the
	// anchor lookup fails (GC'd), fall back to all-reachable + ErrStaleAnchor.
	logFn := s.git.Log
	if firstParent {
		logFn = s.git.LogFirstParent
	}
	rangeCommits, logErr := logFn(anchor, head)
	if logErr != nil {
		return s.reachableFallback(head, latest, docSet, ackedSet, staleErr)
	}
	return rangeCommits, latest, docSet, ackedSet, nil
}

// reachableFallback returns all commits reachable from head paired with the
// supplied sets and wrapErr (the pending fallback when anchor..HEAD can't be
// walked). A hard CommitsReachableFrom failure overrides wrapErr with nil
// commits so callers treat it as a real error, not a fallback list.
func (s *Storage) reachableFallback(
	head string, latest *Entry, docSet, ackedSet map[string]bool, wrapErr error,
) (commits []git.Commit, _ *Entry, _, _ map[string]bool, err error) {
	fallback, reachErr := s.git.CommitsReachableFrom(head)
	if reachErr != nil {
		return nil, latest, docSet, ackedSet, reachErr
	}
	return fallback, latest, docSet, ackedSet, wrapErr
}
