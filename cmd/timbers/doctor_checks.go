// Package main provides the entry point for the timbers CLI.
package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// runCoreChecks performs core infrastructure checks.
func runCoreChecks(flags *doctorFlags) []checkResult {
	checks := make([]checkResult, 0, 6)
	checks = append(checks, checkTimbersDirExists())
	checks = append(checks, checkBinaryInPath())
	checks = append(checks, checkShadowingBinary())
	checks = append(checks, checkVersion())
	checks = append(checks, checkGitattributes())
	checks = append(checks, checkLegacyFilenames(flags))
	return checks
}

// checkTimbersDirExists checks if the .timbers/ directory exists.
func checkTimbersDirExists() checkResult {
	root, err := git.RepoRoot()
	if err != nil {
		return checkResult{
			Name:    "Timbers Directory",
			Status:  checkWarn,
			Message: "could not determine repo root: " + err.Error(),
		}
	}

	timbersDir := filepath.Join(root, ".timbers")
	info, statErr := os.Stat(timbersDir)
	if statErr == nil && info.IsDir() {
		return checkResult{
			Name:    "Timbers Directory",
			Status:  checkPass,
			Message: ".timbers/ directory exists",
		}
	}

	return checkResult{
		Name:    "Timbers Directory",
		Status:  checkWarn,
		Message: ".timbers/ directory not found",
		Hint:    "Run 'timbers init' to initialize",
	}
}

// checkBinaryInPath checks if timbers binary is in PATH.
func checkBinaryInPath() checkResult {
	execPath, err := os.Executable()
	if err != nil {
		return checkResult{
			Name:    "Binary in PATH",
			Status:  checkWarn,
			Message: "could not determine executable path",
		}
	}

	resolvedPath, resolveErr := filepath.EvalSymlinks(execPath)
	if resolveErr != nil {
		return checkResult{
			Name:    "Binary in PATH",
			Status:  checkWarn,
			Message: "could not resolve executable path",
		}
	}

	return checkResult{
		Name:    "Binary in PATH",
		Status:  checkPass,
		Message: resolvedPath,
	}
}

// runWorkflowChecks performs workflow-related checks.
func runWorkflowChecks() []checkResult {
	checks := make([]checkResult, 0, 4)
	checks = append(checks, checkPendingCommits())
	checks = append(checks, checkLatestAnchorTopology())
	checks = append(checks, checkRecentEntries())
	checks = append(checks, checkMergeStrategy())
	return checks
}

// checkLatestAnchorTopology surfaces the Laura pathology: latest entry's
// anchor is on a merged-in side branch rather than HEAD's first-parent
// line. The pending algorithm handles this case correctly via docSet
// filtering, but the situation is opaque to users — pending output reads
// as "scrambled" even though it isn't. The check tells the user what's
// going on and points at the existing escape hatches.
func checkLatestAnchorTopology() checkResult {
	storage, err := ledger.NewDefaultStorage()
	if err != nil {
		return checkResult{
			Name:    "Latest Anchor Topology",
			Status:  checkPass,
			Message: "skipped: " + err.Error(),
		}
	}
	off, latest, offErr := storage.LatestAnchorOffFirstParent()
	if offErr != nil || latest == nil {
		return checkResult{
			Name:    "Latest Anchor Topology",
			Status:  checkPass,
			Message: "no entries or check unavailable",
		}
	}
	if !off {
		return checkResult{
			Name:    "Latest Anchor Topology",
			Status:  checkPass,
			Message: "latest entry's anchor is on this branch's first-parent line",
		}
	}
	return checkResult{
		Name:   "Latest Anchor Topology",
		Status: checkWarn,
		Message: "latest entry (" + latest.ID + ") is anchored to a merged-in side branch (" +
			shortSHA(latest.Workset.AnchorCommit) + ")",
		Hint: "Coverage from side-branch entries still applies via docSet — pending is structurally " +
			"correct but the linear `since-anchor` mental model is opaque here. Escape hatches: " +
			"set TIMBERS_SKIP_CROSS_AGENT_DEBT=1 to bypass the gate, run " +
			"'timbers ack <sha> --reason \"rebased; content in <entry-id>\"' to clear a " +
			"content-preserved commit honestly, or run timbers log on the current branch to re-anchor.",
	}
}

// checkPendingCommits checks for undocumented commits.
func checkPendingCommits() checkResult {
	storage, storageErr := ledger.NewDefaultStorage()
	if storageErr != nil {
		return checkResult{
			Name:    "Pending Commits",
			Status:  checkWarn,
			Message: "could not check: " + storageErr.Error(),
		}
	}

	// Fast path: check for entries before expensive GetPendingCommits.
	// In repos with no entries, GetPendingCommits would fetch and filter
	// the entire commit history only for us to ignore the result.
	entries, listErr := storage.ListEntries()
	if listErr != nil {
		return checkResult{
			Name:    "Pending Commits",
			Status:  checkWarn,
			Message: "could not check: " + listErr.Error(),
		}
	}
	if len(entries) == 0 {
		return checkResult{
			Name:    "Pending Commits",
			Status:  checkPass,
			Message: "tracking starts with your first timbers log",
		}
	}

	commits, _, err := storage.GetPendingCommits()
	if err != nil {
		if errors.Is(err, ledger.ErrStaleAnchor) {
			return checkResult{
				Name:    "Pending Commits",
				Status:  checkWarn,
				Message: "stale anchor — last entry references a commit no longer in history (squash merge or rebase)",
				Hint:    "No action needed. The anchor self-heals on your next timbers log. Prefer merge commits over squash/rebase to avoid this.",
			}
		}
		return checkResult{
			Name:    "Pending Commits",
			Status:  checkWarn,
			Message: "could not check pending commits: " + err.Error(),
		}
	}

	count := len(commits)
	if count == 0 {
		return checkResult{
			Name:    "Pending Commits",
			Status:  checkPass,
			Message: "no undocumented commits",
		}
	}

	return checkResult{
		Name:    "Pending Commits",
		Status:  checkWarn,
		Message: strconv.Itoa(count) + " undocumented commit(s)",
		Hint:    "Run 'timbers pending' to review, then 'timbers log' to document",
	}
}

// checkMergeStrategy checks git config for timbers-friendly merge settings.
// Squash merges and rebases break anchor tracking; merge commits preserve it.
func checkMergeStrategy() checkResult {
	var warnings []string

	// Check pull.rebase
	pullRebase, _ := git.Run("config", "--get", "pull.rebase")
	if pullRebase == "true" {
		warnings = append(warnings, "pull.rebase=true (rebases break anchor tracking)")
	}

	// Check merge.ff (only/true means fast-forward, which avoids merge commits)
	mergeFF, _ := git.Run("config", "--get", "merge.ff")
	if mergeFF == "only" {
		warnings = append(warnings, "merge.ff=only (no merge commits created)")
	}

	if len(warnings) > 0 {
		return checkResult{
			Name:    "Merge Strategy",
			Status:  checkWarn,
			Message: strings.Join(warnings, "; "),
			Hint:    "Timbers works best with merge commits. Run: git config pull.rebase false && git config merge.ff false",
		}
	}

	return checkResult{
		Name:    "Merge Strategy",
		Status:  checkPass,
		Message: "merge-friendly configuration",
	}
}

// checkRecentEntries checks if any ledger entries exist.
func checkRecentEntries() checkResult {
	storage, storageErr := ledger.NewDefaultStorage()
	if storageErr != nil {
		return checkResult{
			Name:    "Recent Entries",
			Status:  checkWarn,
			Message: "could not check: " + storageErr.Error(),
		}
	}

	entries, err := storage.ListEntries()
	if err != nil {
		return checkResult{
			Name:    "Recent Entries",
			Status:  checkWarn,
			Message: "could not list entries: " + err.Error(),
		}
	}

	count := len(entries)
	if count == 0 {
		return checkResult{
			Name:    "Recent Entries",
			Status:  checkPass,
			Message: "no entries yet — tracking starts with your first timbers log",
		}
	}

	return checkResult{
		Name:    "Recent Entries",
		Status:  checkPass,
		Message: strconv.Itoa(count) + " entry(ies) in ledger",
	}
}

// checkGitattributes checks if .gitattributes has the linguist-generated
// rule for .timbers/.
func checkGitattributes() checkResult {
	root, err := git.RepoRoot()
	if err != nil {
		return checkResult{
			Name:    "Gitattributes",
			Status:  checkWarn,
			Message: "could not determine repo root: " + err.Error(),
		}
	}

	data, readErr := os.ReadFile(filepath.Join(root, ".gitattributes"))
	if readErr != nil {
		return checkResult{
			Name:    "Gitattributes",
			Status:  checkWarn,
			Message: ".gitattributes missing linguist-generated rule",
			Hint:    "Run 'timbers init' to configure",
		}
	}

	content := string(data)
	if strings.Contains(content, "/.timbers/**") &&
		strings.Contains(content, "linguist-generated") {
		return checkResult{
			Name:    "Gitattributes",
			Status:  checkPass,
			Message: ".gitattributes configured for timbers",
		}
	}

	return checkResult{
		Name:    "Gitattributes",
		Status:  checkWarn,
		Message: ".gitattributes missing linguist-generated rule",
		Hint:    "Run 'timbers init' to configure",
	}
}

// runIntegrationChecks performs integration-related checks.
func runIntegrationChecks(flags *doctorFlags) []checkResult {
	checks := make([]checkResult, 0, 4)
	checks = append(checks, checkGitHooks(flags))
	checks = append(checks, checkPostCommitHook(flags))
	checks = append(checks, checkPostRewriteHookDrift(flags))
	checks = append(checks, checkAgentIntegrations(flags)...)
	return checks
}
