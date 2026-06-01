// Package ledger provides the entry schema, validation, and serialization for the timbers development ledger.
package ledger

import (
	"errors"
	"path/filepath"
	"sort"
	"time"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
)

// ErrNoEntries is returned when no ledger entries exist.
var ErrNoEntries = errors.New("no ledger entries found")

// ErrStaleAnchor indicates the latest entry's anchor commit no longer exists in git history.
// This happens after squash merges, rebases, or garbage collection.
// When this occurs, GetPendingCommits falls back to all reachable commits from HEAD.
var ErrStaleAnchor = errors.New("anchor commit not found in current history")

// ListStats contains statistics about listing entries.
type ListStats struct {
	Total       int // Total JSON files found
	Parsed      int // Successfully parsed as timbers entries
	Skipped     int // Skipped (not timbers entries or parse errors)
	NotTimbers  int // Specifically: valid JSON but wrong schema
	ParseErrors int // JSON parse failures
}

// GitOps defines the git operations required by Storage.
// Entry storage is handled by FileStorage; this interface covers
// commit history and diff operations only.
type GitOps interface {
	HEAD() (string, error)
	Log(fromRef, toRef string) ([]git.Commit, error)
	LogFirstParent(fromRef, toRef string) ([]git.Commit, error)
	CommitsReachableFrom(sha string) ([]git.Commit, error)
	IsAncestorOf(ancestor, descendant string) bool
	IsOnFirstParentLine(sha, head string) bool
	GetDiffstat(fromRef, toRef string) (git.Diffstat, error)
	CommitFiles(sha string) ([]string, error)
	CommitFilesMulti(shas []string) (map[string][]string, error)
	DiffNameOnly(fromRef, toRef, pathPrefix string) ([]string, error)
}

// Storage provides read/write access to ledger entries stored as files in .timbers/.
type Storage struct {
	git          GitOps
	files        *FileStorage
	skipRules    []skipRule
	skipAuthors  []string
	skipMessages []string
	provenance   ProvenanceConfig // cross-agent debt classifier; zero-value = disabled
}

// NewStorage creates a Storage with the given git operations and file storage.
// If ops is nil, uses real git operations.
// If files is nil, entry operations return empty results.
//
// Performs disk I/O at construction: when files is non-nil, this opens
// <repoRoot>/.timbersignore (if present) to load per-repo skip rules.
// The repo root is derived as the parent of files.Dir (which is .timbers/).
// Loader errors are not fatal — the built-in defaults are used as a safe
// fallback so a malformed .timbersignore never inverts the gate.
//
// The cross-agent debt provenance config is left at zero (disabled). Use
// NewDefaultStorage for production code — it loads provenance from the
// real git config — or call SetProvenance explicitly when you need to
// pin it (custom configuration, integration tests).
func NewStorage(ops GitOps, files *FileStorage) *Storage {
	if ops == nil {
		ops = realGitOps{}
	}
	rules := compiledDefaultSkipRules
	var authors, messages []string
	if files != nil {
		// One file parse yields path rules, author globs, and message
		// globs. A malformed or unreadable .timbersignore must not break
		// pending detection, so loader errors fall through to the defaults.
		if loadedRules, loadedAuthors, loadedMessages, err := loadSkipConfig(filepath.Dir(files.Dir())); err == nil {
			rules = loadedRules
			authors = loadedAuthors
			messages = loadedMessages
		}
	}
	return &Storage{git: ops, files: files, skipRules: rules, skipAuthors: authors, skipMessages: messages}
}

// NewDefaultStorage creates a Storage using real git operations
// and the .timbers/ directory in the repository root.
//
// Production entry point: loads the cross-agent debt provenance config
// from the real environment (git config user.email + the .timbersignore
// session-window directive when present, falling back to
// DefaultSessionWindow otherwise). Empty user.email and a malformed
// session-window value both degrade safely — see LoadProvenanceConfig
// and LoadSessionWindow.
func NewDefaultStorage() (*Storage, error) {
	root, err := git.RepoRoot()
	if err != nil {
		return nil, err
	}
	files := NewFileStorage(filepath.Join(root, ".timbers"), DefaultGitAdd, DefaultGitCommit)
	store := NewStorage(nil, files)
	cfg := LoadProvenanceConfig(time.Now())
	cfg.StaleWindow = LoadSessionWindow(root).Window
	store.SetProvenance(cfg)
	return store, nil
}

// --- Entry CRUD (delegated to FileStorage) ---

// ListEntries returns all entries in the ledger.
// Entries with parse errors are skipped.
// Returns an empty slice if no entries exist or file storage is not configured.
func (s *Storage) ListEntries() ([]*Entry, error) {
	if s.files == nil {
		return nil, nil
	}
	return s.files.ListEntries()
}

// ListEntriesWithStats returns all entries plus statistics about skipped files.
func (s *Storage) ListEntriesWithStats() ([]*Entry, *ListStats, error) {
	if s.files == nil {
		return nil, &ListStats{}, nil
	}
	return s.files.ListEntriesWithStats()
}

// WriteEntry writes an entry to the .timbers/ directory and stages it.
// Validates the entry before writing.
// If force is false and the entry file already exists, returns a conflict error.
// If force is true, overwrites any existing file.
func (s *Storage) WriteEntry(entry *Entry, force bool) error {
	return s.files.WriteEntry(entry, force)
}

// GetEntryByID returns the entry with the given ID.
// Returns a user error (exit code 1) if the entry is not found.
func (s *Storage) GetEntryByID(id string) (*Entry, error) {
	if s.files == nil {
		return nil, output.NewUserError("entry not found: " + id)
	}
	return s.files.ReadEntry(id)
}

// GetLatestEntry returns the entry with the most recent created_at timestamp.
// Returns ErrNoEntries if no entries exist.
func (s *Storage) GetLatestEntry() (*Entry, error) {
	entries, err := s.ListEntries()
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, ErrNoEntries
	}

	latest := entries[0]
	for _, entry := range entries[1:] {
		if entry.CreatedAt.After(latest.CreatedAt) {
			latest = entry
		}
	}

	return latest, nil
}

// GetLastNEntries returns the last N entries sorted by created_at descending.
// Returns entries up to N; if fewer than N exist, returns all entries.
// Returns an empty slice if no entries exist.
func (s *Storage) GetLastNEntries(count int) ([]*Entry, error) {
	entries, err := s.ListEntries()
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return []*Entry{}, nil
	}

	// Sort entries by CreatedAt descending (most recent first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[j].CreatedAt.Before(entries[i].CreatedAt)
	})

	// Return last N entries
	if count >= len(entries) {
		return entries, nil
	}
	return entries[:count], nil
}

// --- Git operations ---

// GetPendingCommits returns commits that have not been documented since the last entry.
// Returns:
//   - commits: commits from the latest entry's anchor (exclusive) to HEAD (inclusive)
//   - latest: the latest entry by created_at (nil if no entries exist)
//   - err: any error that occurred
//
// If no entries exist, returns all commits reachable from HEAD (latest will be nil).
// Callers that display pending counts should check latest == nil to distinguish
// "no entries yet" from "all caught up" and show appropriate messaging.
//
// Walks the full DAG — commits brought into HEAD via merge are included.
// This is the right answer for display commands (timbers pending, prime,
// status, catchup) that surface total documentation debt. The hook gate
// uses GetGatePendingCommits instead, which excludes merged-in work.
func (s *Storage) GetPendingCommits() ([]git.Commit, *Entry, error) {
	return s.getPendingCommits(false)
}

// GetGatePendingCommits returns commits that should gate a new commit —
// i.e., undocumented work on the current branch's first-parent line since
// the latest entry's anchor.
//
// Unlike GetPendingCommits, this skips commits brought in by merges. The
// motivation is parallel-agent flows: when an agent on branch X merges main
// in (and main carries undocumented commits authored by a sibling agent on
// branch Y), those merged-in commits should not block A from committing on
// X. The sibling agent owns documenting their own work; this agent's gate
// should only fire on their own branch's first-parent history.
//
// For single-actor flows on linear history, first-parent collapses to the
// same answer as the full DAG walk, so behavior is unchanged.
func (s *Storage) GetGatePendingCommits() ([]git.Commit, *Entry, error) {
	return s.getPendingCommits(true)
}

// getPendingCommits is the shared implementation of GetPendingCommits and
// GetGatePendingCommits. When firstParent is true, the anchor..HEAD walk
// uses --first-parent so merged-in commits are excluded; in addition,
// commits with no first-parent file changes (clean merges or empty commits)
// are dropped, since they add no new work to this branch's line.
func (s *Storage) getPendingCommits(firstParent bool) ([]git.Commit, *Entry, error) {
	commits, latest, docSet, ackedSet, err := s.pendingRange(firstParent)
	if commits == nil {
		// Hard error (HEAD/reach failure) — nothing to filter.
		return nil, latest, err
	}
	// On stale anchor, commits is the all-reachable fallback; still filter it
	// (callers that care distinguish via errors.Is(err, ErrStaleAnchor)).
	return s.filterCommits(commits, docSet, ackedSet, firstParent), latest, err
}

// latestEntry returns the entry with the most recent CreatedAt, or nil
// when entries is empty.
func latestEntry(entries []*Entry) *Entry {
	if len(entries) == 0 {
		return nil
	}
	latest := entries[0]
	for _, e := range entries[1:] {
		if e.CreatedAt.After(latest.CreatedAt) {
			latest = e
		}
	}
	return latest
}

// HasPendingCommits checks whether undocumented commits exist on the current
// branch's first-parent line — the gate's notion of "this agent's debt."
//
// Delegates to GetGatePendingCommits so that commits brought in by a merge
// (typically authored by a sibling agent on another branch) do not block
// commits on this branch. Display commands (timbers pending) use
// GetPendingCommits, which keeps the full-DAG view for total debt awareness.
//
// Returns false when no entries exist (fresh repos never trigger blocking).
// Returns false on stale anchor (squash/rebase) — the commits shown in the
// fallback are not actionable, and blocking on them causes agents to create
// duplicate entries.
func (s *Storage) HasPendingCommits() (bool, error) {
	commits, latest, err := s.GetGatePendingCommits()
	if err != nil {
		if errors.Is(err, ErrStaleAnchor) {
			return false, nil // stale anchor is not actionable pending
		}
		return false, err
	}
	// No entries yet — don't nag about pre-timbers history.
	if latest == nil {
		return false, nil
	}
	return len(commits) > 0, nil
}

// LogRange returns commits in the given range (fromRef..toRef).
// The 'fromRef' ref is exclusive, 'toRef' is inclusive.
func (s *Storage) LogRange(fromRef, toRef string) ([]git.Commit, error) {
	return s.git.Log(fromRef, toRef)
}

// GetDiffstat returns the change statistics for the given commit range.
func (s *Storage) GetDiffstat(fromRef, toRef string) (git.Diffstat, error) {
	return s.git.GetDiffstat(fromRef, toRef)
}

// DiffNameOnly returns file paths changed between fromRef and toRef,
// optionally filtered to a path prefix.
func (s *Storage) DiffNameOnly(fromRef, toRef, pathPrefix string) ([]string, error) {
	return s.git.DiffNameOnly(fromRef, toRef, pathPrefix)
}
