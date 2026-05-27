package ledger

import "path/filepath"

// matchesSkipMessage reports whether any glob in the set matches the commit
// Subject (first line of the message). Empty glob list returns false. Globs
// are loaded from .timbersignore lines prefixed "msg:" and use filepath.Match
// semantics (*, ?, character classes), so the glob must match the entire
// subject — e.g. "chore: changelog for v*" matches "chore: changelog for
// v1.2.3" but not a subject with a leading prefix.
//
// Match failures from filepath.Match are treated as non-matches so a
// malformed glob can never break pending detection.
func matchesSkipMessage(globs []string, subject string) bool {
	if len(globs) == 0 || subject == "" {
		return false
	}
	for _, glob := range globs {
		if ok, _ := filepath.Match(glob, subject); ok {
			return true
		}
	}
	return false
}
