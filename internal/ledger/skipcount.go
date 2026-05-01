package ledger

import (
	"errors"

	"github.com/gorewood/timbers/internal/git"
)

// CountInfraSkippedSinceLatest returns the number of commits between the
// latest entry's anchor and HEAD that pending detection auto-skips: both
// infrastructure-only commits (.timbers/, .beads/, .timbersignore matches)
// and reverts of already-documented commits. The count uses the same
// predicates as filterByRules so the visibility surface and the gate stay
// in sync.
//
// Returns 0 when no entries exist, when the anchor is stale, or when any
// underlying git call fails — those cases are not actionable signal for
// the caller. Status visibility is best-effort by design.
func (s *Storage) CountInfraSkippedSinceLatest() (int, error) {
	commits, ok, err := s.commitsSinceLatest()
	if err != nil || !ok {
		return 0, err
	}
	return s.countAutoSkipped(commits), nil
}

// commitsSinceLatest returns commits in (latestAnchor..HEAD) when both ends
// are valid and reachable. ok=false signals "no actionable signal" for any
// non-error degraded case (no entries, stale anchor, log failure).
func (s *Storage) commitsSinceLatest() ([]git.Commit, bool, error) {
	latest, err := s.GetLatestEntry()
	if err != nil {
		if errors.Is(err, ErrNoEntries) {
			return nil, false, nil
		}
		return nil, false, err
	}

	head, err := s.git.HEAD()
	if err != nil {
		return nil, false, err
	}

	anchor := latest.Workset.AnchorCommit
	if !s.git.IsAncestorOf(anchor, head) {
		return nil, false, nil
	}

	commits, err := s.git.Log(anchor, head)
	if err != nil {
		return nil, false, nil //nolint:nilerr // stale-anchor counts are best-effort
	}
	return commits, true, nil
}

// countAutoSkipped returns how many of the given commits would be filtered
// from pending — covers both infrastructure-only commits and documented
// reverts. Uses the same predicate set as filterByRules.
func (s *Storage) countAutoSkipped(commits []git.Commit) int {
	if len(commits) == 0 {
		return 0
	}
	fileMap, err := s.git.CommitFilesMulti(commitSHAs(commits))
	if err != nil {
		return 0
	}
	rules := s.rulesOrDefault()
	docSet := s.documentedSHASet()
	var count int
	for _, commit := range commits {
		if isInfrastructureOnlyCommit(rules, fileMap[commit.SHA]) {
			count++
			continue
		}
		if isDocumentedRevert(commit, docSet) {
			count++
		}
	}
	return count
}

// commitSHAs extracts the SHA from each commit for batch lookups.
func commitSHAs(commits []git.Commit) []string {
	shas := make([]string, len(commits))
	for i, commit := range commits {
		shas[i] = commit.SHA
	}
	return shas
}

// filterByRules removes infrastructure-only commits and documented reverts
// from the input list, preserving order. The docSet is supplied by the
// caller so callers can build it once and reuse — avoids re-scanning the
// ledger on every invocation. Pass nil to disable revert auto-skipping.
func (s *Storage) filterByRules(
	commits []git.Commit,
	fileMap map[string][]string,
	docSet map[string]bool,
) []git.Commit {
	rules := s.rulesOrDefault()
	filtered := make([]git.Commit, 0, len(commits))
	for _, commit := range commits {
		if isInfrastructureOnlyCommit(rules, fileMap[commit.SHA]) {
			continue
		}
		if docSet != nil && isDocumentedRevert(commit, docSet) {
			continue
		}
		filtered = append(filtered, commit)
	}
	return filtered
}
