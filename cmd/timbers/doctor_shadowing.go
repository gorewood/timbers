package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// checkShadowingBinary detects multiple distinct `timbers` binaries on PATH.
// The git pre-commit hook runs whichever `timbers` resolves first on PATH;
// if a stale binary (e.g. a `dev` build left in a Go bin dir) shadows the
// installed release, the gate can misbehave — most visibly, an older binary
// won't treat `ack` records as documented (that landed in v0.22.0), so it
// blocks commits the current binary considers clean. This check surfaces that
// divergence before it bites.
func checkShadowingBinary() checkResult {
	const name = "Binary Shadowing"

	bins := timbersBinariesOnPath()
	if len(bins) <= 1 {
		return checkResult{
			Name:    name,
			Status:  checkPass,
			Message: "single timbers on PATH",
		}
	}

	winner := bins[0]
	winnerVer := binaryVersionToken(winner)
	if winnerVer == "?" {
		// Can't read the winner's version (unusual) — don't cry wolf.
		return checkResult{
			Name:    name,
			Status:  checkPass,
			Message: "multiple timbers on PATH (versions undetermined)",
		}
	}
	for _, other := range bins[1:] {
		otherVer := binaryVersionToken(other)
		// Only flag a concrete, different version. A "?" (broken shim, a
		// binary that can't run --version) isn't evidence of shadowing.
		if otherVer != "?" && otherVer != winnerVer {
			return checkResult{
				Name:   name,
				Status: checkWarn,
				Message: "multiple timbers on PATH report different versions; " +
					winner + " (" + winnerVer + ") shadows " + other + " (" + otherVer + ")",
				Hint: "The git hook runs the first timbers on PATH — a stale one blocks " +
					"commits the current binary considers clean (e.g. acks, added in v0.22.0). " +
					"Remove the stale binary (often a 'dev' build from 'go install' in a Go bin " +
					"dir) or fix PATH order, then re-run 'timbers doctor'.",
			}
		}
	}

	return checkResult{
		Name:    name,
		Status:  checkPass,
		Message: "multiple timbers on PATH, same version",
	}
}

// timbersBinariesOnPath returns the resolved paths of every executable named
// "timbers" found on PATH, in PATH order, deduplicated by resolved real path
// so a symlink and its target count once.
func timbersBinariesOnPath() []string {
	var ordered []string
	seen := map[string]bool{}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, "timbers")
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			continue
		}
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			resolved = candidate
		}
		if seen[resolved] {
			continue
		}
		seen[resolved] = true
		ordered = append(ordered, resolved)
	}
	return ordered
}

// binaryVersionToken runs `<path> --version` and returns the version token
// (e.g. "v0.22.3" or "dev"), ignoring the commit/date suffix so two builds of
// the same version don't read as different. Returns "?" if the binary can't be
// run or parsed.
func binaryVersionToken(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	//nolint:gosec // path is a resolved timbers binary already on the user's PATH
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return "?"
	}
	return parseVersionToken(string(out))
}

// parseVersionToken extracts the version token from `timbers --version`
// output ("timbers version <token> (<commit>, <date>)"), returning just
// <token> so two builds of the same version with different commit/date
// suffixes compare equal. Returns "?" when no token is found.
func parseVersionToken(output string) string {
	fields := strings.Fields(output)
	for i, f := range fields {
		if f == "version" && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return "?"
}
