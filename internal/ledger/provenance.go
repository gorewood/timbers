package ledger

import (
	"time"

	"github.com/gorewood/timbers/internal/git"
)

// Provenance reason values returned by classifyByProvenance. Layered on top
// of the existing skip chain (infra → identity → content) — provenance fires
// only on commits that would otherwise be kept.
const (
	reasonForeignAuthor      = "foreign-author"
	reasonStale              = "stale"
	reasonForeignAuthorStale = "foreign-author+stale"
)

// ProvenanceConfig drives the cross-agent debt classifier. A commit is
// classified as out-of-session — and silently skipped by the gate — when
// its mailmap-resolved AuthorEmail differs from UserEmail OR its CommitDate
// is older than StaleWindow.
//
// Safe-degradation rules (intentional — see plan doc):
//
//   - Empty UserEmail disables the email check (e.g. when `git config
//     user.email` is unset). Otherwise every commit would mismatch and the
//     gate would silently disable itself.
//   - Zero StaleWindow disables the staleness check.
//   - Zero CommitDate on the commit treats it as not-stale (defensive
//     fallback for malformed git output).
//   - Negative ages (CommitDate > Now, e.g. clock skew) do NOT classify as
//     stale; the heuristic is one-sided.
//
// Now is the comparison reference. Callers set it to time.Now() in
// production and to a fixed point in tests so behavior is reproducible.
type ProvenanceConfig struct {
	UserEmail   string
	StaleWindow time.Duration
	Now         time.Time
}

// Enabled reports whether at least one of the two provenance checks can
// fire. When both are disabled the classifier is a no-op.
func (p ProvenanceConfig) Enabled() bool {
	return p.UserEmail != "" || p.StaleWindow > 0
}

// classifyByProvenance returns one of the provenance reason constants
// (or "" when the commit is in-session). Composite reason
// "foreign-author+stale" is returned when both checks fire — it preserves
// both diagnostic signals in --explain output instead of arbitrarily
// picking one.
func classifyByProvenance(commit git.Commit, cfg ProvenanceConfig) string {
	if !cfg.Enabled() {
		return ""
	}
	foreignAuthor := cfg.UserEmail != "" && commit.AuthorEmail != cfg.UserEmail
	stale := isStale(commit.CommitDate, cfg.StaleWindow, cfg.Now)
	switch {
	case foreignAuthor && stale:
		return reasonForeignAuthorStale
	case foreignAuthor:
		return reasonForeignAuthor
	case stale:
		return reasonStale
	}
	return ""
}

// isStale returns true when the commit's CommitDate is older than window
// relative to now. Zero window or zero commitDate → false (defensive
// fallback; see ProvenanceConfig safe-degradation rules).
func isStale(commitDate time.Time, window time.Duration, now time.Time) bool {
	if window <= 0 || commitDate.IsZero() {
		return false
	}
	age := now.Sub(commitDate)
	return age > window
}
