package ledger

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
)

// GitAddFunc stages a file at the given path.
type GitAddFunc func(path string) error

// DefaultGitAdd stages a file using git add.
func DefaultGitAdd(path string) error {
	_, err := git.Run("add", path)
	return err
}

// GitCommitFunc commits a file at the given path with the given message.
type GitCommitFunc func(path string, message string) error

// DefaultGitCommit commits a specific file using git commit with pathspec.
// The -- before the path ensures only the entry file is committed,
// not other staged files.
func DefaultGitCommit(path string, message string) error {
	_, err := git.Run("commit", "-m", message, "--", path)
	return err
}

// FileStorage provides file-based storage for ledger entries in YYYY/MM/DD subdirectories.
// Each entry is stored as a JSON file at YYYY/MM/DD/<entry-id>.json.
type FileStorage struct {
	dir       string
	gitAdd    GitAddFunc
	gitCommit GitCommitFunc
}

// NewFileStorage creates a FileStorage for the given directory.
// If gitAdd is nil, uses DefaultGitAdd.
// If gitCommit is nil, uses DefaultGitCommit.
func NewFileStorage(dir string, gitAdd GitAddFunc, gitCommit GitCommitFunc) *FileStorage {
	if gitAdd == nil {
		gitAdd = DefaultGitAdd
	}
	if gitCommit == nil {
		gitCommit = DefaultGitCommit
	}
	return &FileStorage{dir: dir, gitAdd: gitAdd, gitCommit: gitCommit}
}

// Dir returns the storage directory path.
func (fs *FileStorage) Dir() string {
	return fs.dir
}

// DirExists returns true if the storage directory exists.
func (fs *FileStorage) DirExists() bool {
	info, err := os.Stat(fs.dir)
	return err == nil && info.IsDir()
}

// EntryDateDir extracts the YYYY/MM/DD relative path from an entry ID.
// Entry IDs have the format: tb_YYYY-MM-DDT...
// Returns empty string if the ID format is unexpected.
func EntryDateDir(id string) string {
	if len(id) >= 13 && id[:3] == "tb_" {
		datePart := id[3:13] // "2026-01-19"
		parts := strings.SplitN(datePart, "-", 3)
		if len(parts) == 3 {
			return filepath.Join(parts[0], parts[1], parts[2])
		}
	}
	return ""
}

// entryDir returns the directory for an entry ID (root/YYYY/MM/DD).
// Falls back to the root storage directory if the ID format is unexpected.
func (fs *FileStorage) entryDir(id string) string {
	if sub := EntryDateDir(id); sub != "" {
		return filepath.Join(fs.dir, sub)
	}
	return fs.dir
}

// entryPath returns the file path for an entry ID.
func (fs *FileStorage) entryPath(id string) string {
	return filepath.Join(fs.entryDir(id), id+".json")
}

// ReadEntry reads the entry with the given ID from the storage directory.
// Returns a user error if the entry file does not exist.
// Returns ErrNotTimbersNote if the file is valid JSON but not a timbers entry.
func (fs *FileStorage) ReadEntry(id string) (*Entry, error) {
	path := fs.entryPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, output.NewUserError("entry not found: " + id)
		}
		return nil, output.NewSystemErrorWithCause("failed to read entry file: "+path, err)
	}

	entry, err := FromJSON(data)
	if err != nil {
		if errors.Is(err, ErrNotTimbersNote) {
			return nil, err
		}
		return nil, output.NewUserError("failed to parse entry: " + err.Error())
	}

	return entry, nil
}

// ListEntries returns all entries in the storage directory.
// Entries with parse errors are skipped.
// Returns an empty slice if the directory does not exist or is empty.
func (fs *FileStorage) ListEntries() ([]*Entry, error) {
	entries, _, err := fs.ListEntriesWithStats()
	return entries, err
}

// ListEntriesWithStats returns all entries plus statistics about skipped files.
// Only .json files are considered; directories and other files are ignored.
// Returns empty results if the directory does not exist.
func (fs *FileStorage) ListEntriesWithStats() ([]*Entry, *ListStats, error) {
	stats := &ListStats{}
	var entries []*Entry

	err := filepath.WalkDir(fs.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}

		stats.Total++
		id := strings.TrimSuffix(d.Name(), ".json")
		entry, readErr := fs.ReadEntry(id)
		if readErr != nil {
			stats.Skipped++
			if errors.Is(readErr, ErrNotTimbersNote) {
				stats.NotTimbers++
			} else {
				stats.ParseErrors++
			}
			return nil
		}
		entries = append(entries, entry)
		stats.Parsed++
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &ListStats{}, nil
		}
		return nil, nil, output.NewSystemErrorWithCause("failed to walk storage directory", err)
	}

	return entries, stats, nil
}

// WriteEntry writes an entry to the storage directory and stages it with git add.
// Validates the entry before writing. Uses write-to-temp-then-rename for atomicity.
// If force is false and the entry file already exists, returns a conflict error.
// If force is true, overwrites any existing file.
func (fs *FileStorage) WriteEntry(entry *Entry, force bool) error {
	if err := entry.Validate(); err != nil {
		return output.NewUserError(err.Error())
	}

	path := fs.entryPath(entry.ID)

	// Check for existing entry if not forcing
	if !force {
		if _, err := os.Stat(path); err == nil {
			return output.NewConflictError("entry already exists: " + entry.ID)
		}
	}

	data, err := entry.ToJSON()
	if err != nil {
		return output.NewSystemError("failed to serialize entry: " + err.Error())
	}

	// Ensure the date directory exists
	if err = os.MkdirAll(fs.entryDir(entry.ID), 0o755); err != nil {
		return output.NewSystemErrorWithCause("failed to create entry directory", err)
	}

	if err = atomicWrite(path, data); err != nil {
		return output.NewSystemErrorWithCause("failed to write entry", err)
	}

	if err = fs.gitAdd(path); err != nil {
		return output.NewSystemErrorWithCause("failed to stage entry file", err)
	}

	if err = fs.gitCommit(path, "timbers: document "+entry.ID); err != nil {
		return output.NewSystemErrorWithCause("failed to commit entry file", err)
	}

	return nil
}

// atomicWrite writes data to path using write-to-temp-then-rename.
// The temp file is created in the same directory as path.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write data: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// EntryExists returns true if an entry file exists for the given ID.
func (fs *FileStorage) EntryExists(id string) bool {
	_, err := os.Stat(fs.entryPath(id))
	return err == nil
}
