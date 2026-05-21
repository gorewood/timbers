package ledger

import (
	"fmt"

	"github.com/gorewood/timbers/internal/git"
)

// anchorShortCircuit handles two short-circuit cases that bypass the
// normal Log(anchor, head) walk:
//   - Stale anchor: anchor SHA no longer reachable from HEAD (squash/
//     rebase GC'd it). Falls back to all-reachable + docSet filtering
//     and wraps ErrStaleAnchor so display callers can surface a hint.
//   - Off-first-parent anchor in gate path: anchor is reachable but
//     not on HEAD's first-parent line (latest entry was authored on a
//     merged-in side branch). LogFirstParent(anchor, head) would walk
//     a structurally weird range — exclude side is full-DAG, include
//     side is first-parent only. Falls back to the same all-reachable
//     path without wrapping ErrStaleAnchor.
//
// Returns (fallbackCommits, err, true) when a short-circuit applied;
// (nil, nil, false) when the caller should continue with the normal
// log walk.
func (s *Storage) anchorShortCircuit(
	anchor, head string,
	docSet, ackedSet map[string]bool,
	firstParent bool,
) ([]git.Commit, error, bool) {
	if !s.git.IsAncestorOf(anchor, head) {
		fallback, reachErr := s.git.CommitsReachableFrom(head)
		if reachErr != nil {
			return nil, reachErr, true
		}
		return s.filterCommits(fallback, docSet, ackedSet, firstParent),
			fmt.Errorf("%w: %s", ErrStaleAnchor, anchor),
			true
	}
	if firstParent && !s.git.IsOnFirstParentLine(anchor, head) {
		fallback, reachErr := s.git.CommitsReachableFrom(head)
		if reachErr != nil {
			return nil, reachErr, true
		}
		return s.filterCommits(fallback, docSet, ackedSet, firstParent), nil, true
	}
	return nil, nil, false
}

// LatestAnchorOffFirstParent reports whether the latest entry's anchor
// is reachable from HEAD via merge but NOT via first-parent traversal —
// i.e., the entry was authored on a side branch that has been merged in.
//
// Returns (true, latest, nil) when the situation applies. The caller can
// use `latest` to surface entry details in a diagnostic message.
//
// Returns (false, _, _) in every other case, including: no entries, git
// errors, latest anchor on the first-parent line (healthy), or latest
// anchor stale (squash/rebase — different signal, surfaced separately).
//
// Purpose: surface the topology to users when the latest entry came in
// via a merge from a side branch. The pending/gate algorithm handles
// this case correctly via docSet filtering, but the situation is opaque
// to users and reads as "pending is scrambling." A diagnostic hint
// pointing at the existing escape hatches (TIMBERS_SKIP_CROSS_AGENT_DEBT,
// timbers ack, re-log on main) closes the UX gap without algorithm churn.
func (s *Storage) LatestAnchorOffFirstParent() (bool, *Entry, error) {
	entries, err := s.ListEntries()
	if err != nil || len(entries) == 0 {
		return false, nil, err
	}
	latest := latestEntry(entries)
	if latest == nil || latest.Workset.AnchorCommit == "" {
		return false, latest, nil
	}
	head, headErr := s.git.HEAD()
	if headErr != nil {
		//nolint:nilerr // diagnostic is best-effort; never propagate as error
		return false, latest, nil
	}
	// Stale anchor (not reachable at all) is a separate signal — surfaced
	// elsewhere — and would muddy this one if conflated.
	if !s.git.IsAncestorOf(latest.Workset.AnchorCommit, head) {
		return false, latest, nil
	}
	onLine := s.git.IsOnFirstParentLine(latest.Workset.AnchorCommit, head)
	return !onLine, latest, nil
}
