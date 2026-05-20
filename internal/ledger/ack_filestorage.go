package ledger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorewood/timbers/internal/output"
)

// ackDir returns the directory for an ack ID (root/YYYY/MM/DD). Falls
// back to the root storage directory if the ID doesn't parse.
func (fs *FileStorage) ackDir(id string) string {
	if sub := AckDateDir(id); sub != "" {
		return filepath.Join(fs.dir, sub)
	}
	return fs.dir
}

// ackPath returns the canonical file path for an ack ID. Colons in the
// timestamp portion are replaced with dashes to match the entry-storage
// convention (Windows-safe; matches existing IDToFilename behavior).
func (fs *FileStorage) ackPath(id string) string {
	return filepath.Join(fs.ackDir(id), IDToFilename(id)+".json")
}

// WriteAck writes an ack record to the storage directory and stages +
// commits it. Validates the ack before writing. Uses the same atomic
// write-and-rename pattern as WriteEntry; commit message follows the
// existing "timbers: document <id>" convention but with "ack" in place
// of the entry id-prefix.
func (fs *FileStorage) WriteAck(ack *Ack) error {
	if err := ack.Validate(); err != nil {
		return output.NewUserError(err.Error())
	}

	path := fs.ackPath(ack.ID)

	if _, err := os.Stat(path); err == nil {
		return output.NewConflictError("ack already exists: " + ack.ID)
	}

	data, err := ack.ToJSON()
	if err != nil {
		return output.NewSystemError("failed to serialize ack: " + err.Error())
	}

	if err = os.MkdirAll(fs.ackDir(ack.ID), 0o755); err != nil {
		return output.NewSystemErrorWithCause("failed to create ack directory", err)
	}
	if err = atomicWrite(path, data); err != nil {
		return output.NewSystemErrorWithCause("failed to write ack", err)
	}
	if err = fs.gitAdd(path); err != nil {
		return output.NewSystemErrorWithCause("failed to stage ack file", err)
	}
	if err = fs.gitCommit(path, "timbers: ack "+ack.ID); err != nil {
		return output.NewSystemErrorWithCause("failed to commit ack file", err)
	}
	return nil
}

// ListAcks returns every ack record under the storage directory. Skips
// files that don't look like ack files (don't start with "ack_") so
// entries and acks can share the same date-dir layout.
func (fs *FileStorage) ListAcks() ([]*Ack, error) {
	var acks []*Ack
	walkErr := filepath.WalkDir(fs.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		name := strings.TrimSuffix(d.Name(), ".json")
		if !strings.HasPrefix(name, ackIDPrefix) {
			return nil
		}
		//nolint:gosec // path comes from WalkDir under fs.dir
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			//nolint:nilerr // ListAcks is best-effort; unreadable files are silently skipped so a single bad file doesn't break pending detection
			return nil
		}
		ack, parseErr := FromJSONAck(data)
		if parseErr != nil {
			//nolint:nilerr // not an ack record (e.g., legacy file with same prefix) — silently skip
			return nil
		}
		acks = append(acks, ack)
		return nil
	})
	if walkErr != nil {
		if errors.Is(walkErr, os.ErrNotExist) {
			return nil, nil
		}
		return nil, output.NewSystemErrorWithCause("failed to walk ack directory", walkErr)
	}
	return acks, nil
}
