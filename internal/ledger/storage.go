// Package ledger provides the entry schema, validation, and serialization for the timbers development ledger.
package ledger

import (
	"errors"

	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/output"
)

// ErrNoEntries is returned when no ledger entries exist.
var ErrNoEntries = errors.New("no ledger entries found")

// GitOps defines the git operations required by Storage.
// This allows injection of mock implementations for testing.
type GitOps interface {
	ReadNote(commit string) ([]byte, error)
	WriteNote(commit string, content string, force bool) error
	ListNotedCommits() ([]string, error)
	HEAD() (string, error)
	Log(fromRef, toRef string) ([]git.Commit, error)
	CommitsReachableFrom(sha string) ([]git.Commit, error)
	GetDiffstat(fromRef, toRef string) (git.Diffstat, error)
	PushNotes(remote string) error
}

// realGitOps implements GitOps using the actual git package functions.
type realGitOps struct{}

func (realGitOps) ReadNote(commit string) ([]byte, error) {
	return git.ReadNote(commit)
}

func (realGitOps) WriteNote(commit string, content string, force bool) error {
	return git.WriteNote(commit, content, force)
}

func (realGitOps) ListNotedCommits() ([]string, error) {
	return git.ListNotedCommits()
}

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

func (realGitOps) PushNotes(remote string) error {
	return git.PushNotes(remote)
}

// Storage provides read/write access to ledger entries stored in git notes.
type Storage struct {
	git GitOps
}

// NewStorage creates a Storage that uses real git operations.
func NewStorage(ops GitOps) *Storage {
	if ops == nil {
		ops = realGitOps{}
	}
	return &Storage{git: ops}
}

// ReadEntry reads the entry attached to the given anchor commit.
// Returns a user error (exit code 1) if the entry is not found.
// Returns a user error if the note content cannot be parsed as an Entry.
func (s *Storage) ReadEntry(anchor string) (*Entry, error) {
	data, err := s.git.ReadNote(anchor)
	if err != nil {
		return nil, err
	}

	entry, err := FromJSON(data)
	if err != nil {
		return nil, output.NewUserError("failed to parse entry: " + err.Error())
	}

	return entry, nil
}

// ListEntries returns all entries in the ledger.
// Entries with parse errors are skipped (logged but not returned).
// Returns an empty slice if no entries exist.
func (s *Storage) ListEntries() ([]*Entry, error) {
	commits, err := s.git.ListNotedCommits()
	if err != nil {
		return nil, err
	}

	var entries []*Entry
	for _, commit := range commits {
		entry, err := s.ReadEntry(commit)
		if err != nil {
			// Skip entries with parse errors
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
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

// WriteEntry writes an entry to git notes on its anchor commit.
// Validates the entry before writing.
// If force is false and a note already exists, returns a conflict error (exit code 3).
// If force is true, overwrites any existing note.
func (s *Storage) WriteEntry(entry *Entry, force bool) error {
	// Validate before writing
	if err := entry.Validate(); err != nil {
		return output.NewUserError(err.Error())
	}

	// Check for existing note if not forcing
	if !force {
		_, err := s.git.ReadNote(entry.Workset.AnchorCommit)
		if err == nil {
			// Note exists
			return output.NewConflictError("entry already exists for commit: " + entry.Workset.AnchorCommit)
		}
		// Only proceed if error is "not found"
		var exitErr *output.ExitError
		if errors.As(err, &exitErr) && exitErr.Code != output.ExitUserError {
			// Some other error (not "not found")
			return err
		}
	}

	// Serialize entry
	data, err := entry.ToJSON()
	if err != nil {
		return output.NewSystemError("failed to serialize entry: " + err.Error())
	}

	// Write to git notes
	if err := s.git.WriteNote(entry.Workset.AnchorCommit, string(data), force); err != nil {
		return err
	}

	return nil
}

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

	// Get commits from anchor (exclusive) to HEAD (inclusive)
	commits, logErr := s.git.Log(latest.Workset.AnchorCommit, head)
	if logErr != nil {
		return nil, nil, logErr
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

// PushNotes pushes the notes ref to the given remote.
func (s *Storage) PushNotes(remote string) error {
	return s.git.PushNotes(remote)
}

// GetEntryByID returns the entry with the given ID.
// Returns a user error (exit code 1) if the entry is not found.
func (s *Storage) GetEntryByID(id string) (*Entry, error) {
	entries, err := s.ListEntries()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.ID == id {
			return entry, nil
		}
	}

	return nil, output.NewUserError("entry not found: " + id)
}
