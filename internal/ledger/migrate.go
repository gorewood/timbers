package ledger

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorewood/timbers/internal/output"
)

// MigrateLegacyFilenames renames pre-v0.18 colon-encoded entry files in the
// underlying FileStorage to the canonical (dashed) form. Returns the IDs that
// were migrated, or an empty slice if file storage is not configured.
func (s *Storage) MigrateLegacyFilenames() ([]string, error) {
	if s.files == nil {
		return nil, nil
	}
	return s.files.MigrateLegacyFilenames()
}

// MigrateLegacyFilenames walks the storage directory and renames any
// pre-v0.18 colon-encoded entry files to the canonical (dashed) form.
// Returns the IDs that were migrated. Idempotent: a no-op if everything is
// already canonical, and tolerates a canonical sibling already existing
// (the legacy file is removed in that case).
func (fs *FileStorage) MigrateLegacyFilenames() ([]string, error) {
	var migrated []string
	walkErr := filepath.WalkDir(fs.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		base := strings.TrimSuffix(d.Name(), ".json")
		if !strings.Contains(base, ":") {
			return nil
		}
		canonicalName := IDToFilename(base) + ".json"
		canonicalPath := filepath.Join(filepath.Dir(path), canonicalName)
		if _, statErr := os.Stat(canonicalPath); statErr == nil {
			// Canonical exists already — drop the legacy duplicate.
			if rmErr := os.Remove(path); rmErr != nil {
				return fmt.Errorf("remove duplicate legacy %s: %w", path, rmErr)
			}
		} else if rnErr := os.Rename(path, canonicalPath); rnErr != nil {
			return fmt.Errorf("rename %s: %w", path, rnErr)
		}
		migrated = append(migrated, FilenameToID(base))
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, os.ErrNotExist) {
		return migrated, output.NewSystemErrorWithCause("filename migration walk failed", walkErr)
	}
	return migrated, nil
}
