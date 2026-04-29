// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// checkLegacyFilenames detects pre-v0.18 colon-encoded entry files. Colons in
// filenames break Go's module zip format, which prevents 'go install' from
// working on tagged versions of this repo. Auto-fixable: --fix renames legacy
// files to the canonical (dashed) form.
func checkLegacyFilenames(flags *doctorFlags) checkResult {
	dir, dirCheck, ok := legacyFilenamesDir()
	if !ok {
		return dirCheck
	}

	legacyCount, scanErr := countLegacyFilenames(dir)
	if scanErr != nil {
		return checkResult{
			Name:    "Filename Encoding",
			Status:  checkWarn,
			Message: "scan failed: " + scanErr.Error(),
		}
	}
	if legacyCount == 0 {
		return checkResult{
			Name:    "Filename Encoding",
			Status:  checkPass,
			Message: "all entry filenames are canonical",
		}
	}
	if flags != nil && flags.fix {
		if fixed, ok := tryFixLegacyFilenames(); ok {
			return fixed
		}
	}
	return checkResult{
		Name:    "Filename Encoding",
		Status:  checkFail,
		Message: strconv.Itoa(legacyCount) + " legacy colon-encoded filenames break 'go install'",
		Hint:    "Run 'timbers doctor --fix' to rename files, then commit",
	}
}

// legacyFilenamesDir returns the .timbers directory to scan, or a pre-baked
// pass/warn result when the directory is unavailable.
func legacyFilenamesDir() (string, checkResult, bool) {
	root, err := git.RepoRoot()
	if err != nil {
		return "", checkResult{
			Name:    "Filename Encoding",
			Status:  checkWarn,
			Message: "could not determine repo root: " + err.Error(),
		}, false
	}
	dir := filepath.Join(root, ".timbers")
	if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
		return "", checkResult{
			Name:    "Filename Encoding",
			Status:  checkPass,
			Message: "no .timbers/ directory",
		}, false
	}
	return dir, checkResult{}, true
}

// countLegacyFilenames returns the number of `.json` files under dir whose
// basename contains a colon (pre-v0.18 encoding).
func countLegacyFilenames(dir string) (int, error) {
	count := 0
	err := filepath.WalkDir(dir, func(_ string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		if strings.Contains(d.Name(), ":") {
			count++
		}
		return nil
	})
	if err != nil {
		return count, fmt.Errorf("walk %s: %w", dir, err)
	}
	return count, nil
}

// tryFixLegacyFilenames runs the migration and returns a success check, or
// (zero, false) if storage construction or migration failed.
func tryFixLegacyFilenames() (checkResult, bool) {
	store, err := ledger.NewDefaultStorage()
	if err != nil {
		return checkResult{}, false
	}
	migrated, migErr := store.MigrateLegacyFilenames()
	if migErr != nil {
		return checkResult{}, false
	}
	return checkResult{
		Name:    "Filename Encoding",
		Status:  checkPass,
		Message: "renamed " + strconv.Itoa(len(migrated)) + " legacy file(s) to canonical form",
	}, true
}
