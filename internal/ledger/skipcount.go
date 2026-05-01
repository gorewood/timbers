package ledger

import (
	"errors"

	"github.com/gorewood/timbers/internal/git"
)

// CountInfraSkippedSinceLatest returns the number of commits between the
// latest entry's anchor and HEAD that are filtered out as housekeeping
// (e.g., .timbers/, .beads/, files matching .timbersignore).
//
// Returns 0 when no entries exist, when the anchor is stale, or when any
// underlying git call fails — those cases are not actionable signal for
// the caller. Status visibility is best-effort by design.
func (s *Storage) CountInfraSkippedSinceLatest() (int, error) {
	commits, ok, err := s.commitsSinceLatest()
	if err != nil || !ok {
		return 0, err
	}
	return s.countInfraOnly(commits), nil
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

// countInfraOnly returns how many of the given commits touch only files
// matching the storage's skip rules.
func (s *Storage) countInfraOnly(commits []git.Commit) int {
	if len(commits) == 0 {
		return 0
	}
	shas := make([]string, len(commits))
	for i, c := range commits {
		shas[i] = c.SHA
	}
	fileMap, err := s.git.CommitFilesMulti(shas)
	if err != nil {
		return 0
	}
	rules := s.rulesOrDefault()
	var count int
	for _, c := range commits {
		if isInfrastructureOnlyCommit(rules, fileMap[c.SHA]) {
			count++
		}
	}
	return count
}
