package ledger

import (
	"errors"
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

// FileStorage provides file-based storage for ledger entries in a flat directory.
// Each entry is stored as a JSON file named <entry-id>.json.
type FileStorage struct {
	dir    string
	gitAdd GitAddFunc
}

// NewFileStorage creates a FileStorage for the given directory.
// If gitAdd is nil, uses DefaultGitAdd.
func NewFileStorage(dir string, gitAdd GitAddFunc) *FileStorage {
	if gitAdd == nil {
		gitAdd = DefaultGitAdd
	}
	return &FileStorage{dir: dir, gitAdd: gitAdd}
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

// entryPath returns the file path for an entry ID.
func (fs *FileStorage) entryPath(id string) string {
	return filepath.Join(fs.dir, id+".json")
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
	dirEntries, err := os.ReadDir(fs.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &ListStats{}, nil
		}
		return nil, nil, output.NewSystemErrorWithCause("failed to read storage directory", err)
	}

	stats := &ListStats{}
	var entries []*Entry

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() || !strings.HasSuffix(dirEntry.Name(), ".json") {
			continue
		}

		stats.Total++
		id := strings.TrimSuffix(dirEntry.Name(), ".json")
		entry, readErr := fs.ReadEntry(id)
		if readErr != nil {
			stats.Skipped++
			if errors.Is(readErr, ErrNotTimbersNote) {
				stats.NotTimbers++
			} else {
				stats.ParseErrors++
			}
			continue
		}
		entries = append(entries, entry)
		stats.Parsed++
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

	// Write to temp file in same directory for atomic rename
	tmpFile, err := os.CreateTemp(fs.dir, ".tmp-*.json")
	if err != nil {
		return output.NewSystemErrorWithCause("failed to create temp file", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on any error (no-op after successful rename)
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return output.NewSystemErrorWithCause("failed to write entry data", err)
	}

	if err := tmpFile.Close(); err != nil {
		return output.NewSystemErrorWithCause("failed to close temp file", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return output.NewSystemErrorWithCause("failed to rename temp file", err)
	}

	if err := fs.gitAdd(path); err != nil {
		return output.NewSystemErrorWithCause("failed to stage entry file", err)
	}

	return nil
}

// EntryExists returns true if an entry file exists for the given ID.
func (fs *FileStorage) EntryExists(id string) bool {
	_, err := os.Stat(fs.entryPath(id))
	return err == nil
}
