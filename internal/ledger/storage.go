// Package ledger provides the entry schema, validation, and serialization for the timbers development ledger.
package ledger

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"

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
	CommitsReachableFrom(sha string) ([]git.Commit, error)
	GetDiffstat(fromRef, toRef string) (git.Diffstat, error)
}

// realGitOps implements GitOps using the actual git package functions.
type realGitOps struct{}

func (realGitOps) HEAD() (string, error) {
	return git.HEAD()
}

func (realGitOps) Log(fromRef, toRef string) ([]git.Commit, error) {
	return git.Log(fromRef, toRef)
}

func (realGitOps) CommitsReachableFrom(sha string) ([]git.Commit, error) {
	return git.CommitsReachableFrom(sha)
}

func (realGitOps) GetDiffstat(fromRef, toRef string) (git.Diffstat, error) {
	return git.GetDiffstat(fromRef, toRef)
}

// Storage provides read/write access to ledger entries stored as files in .timbers/.
type Storage struct {
	git   GitOps
	files *FileStorage
}

// NewStorage creates a Storage with the given git operations and file storage.
// If ops is nil, uses real git operations.
// If files is nil, entry operations return empty results.
func NewStorage(ops GitOps, files *FileStorage) *Storage {
	if ops == nil {
		ops = realGitOps{}
	}
	return &Storage{git: ops, files: files}
}

// NewDefaultStorage creates a Storage using real git operations
// and the .timbers/ directory in the repository root.
func NewDefaultStorage() (*Storage, error) {
	root, err := git.RepoRoot()
	if err != nil {
		return nil, err
	}
	files := NewFileStorage(filepath.Join(root, ".timbers"), DefaultGitAdd)
	return NewStorage(nil, files), nil
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
// If no entries exist, returns all commits reachable from HEAD.
func (s *Storage) GetPendingCommits() ([]git.Commit, *Entry, error) {
	head, err := s.git.HEAD()
	if err != nil {
		return nil, nil, err
	}

	latest, latestErr := s.GetLatestEntry()
	if latestErr != nil && !errors.Is(latestErr, ErrNoEntries) {
		return nil, nil, latestErr
	}

	// If no entries exist, return all commits reachable from HEAD
	if errors.Is(latestErr, ErrNoEntries) {
		commits, reachErr := s.git.CommitsReachableFrom(head)
		if reachErr != nil {
			return nil, nil, reachErr
		}
		return commits, nil, nil
	}

	// Get commits from anchor (exclusive) to HEAD (inclusive).
	// If the anchor no longer exists (squash merge, rebase, GC), fall back
	// to all reachable commits from HEAD and wrap ErrStaleAnchor so the
	// caller can emit a warning while still returning useful data.
	commits, logErr := s.git.Log(latest.Workset.AnchorCommit, head)
	if logErr != nil {
		fallback, reachErr := s.git.CommitsReachableFrom(head)
		if reachErr != nil {
			return nil, nil, reachErr
		}
		return fallback, latest, fmt.Errorf("%w: %s", ErrStaleAnchor, latest.Workset.AnchorCommit)
	}

	return commits, latest, nil
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
