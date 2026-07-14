package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorewood/timbers/internal/draft"
	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// buildRenderContext creates a RenderContext from entries and flags.
func buildRenderContext(entries []*ledger.Entry, appendFlag string, vars map[string]string) *draft.RenderContext {
	repoName := ""
	if root, rootErr := git.RepoRoot(); rootErr == nil {
		repoName = filepath.Base(root)
	}
	branch, _ := git.CurrentBranch()

	// Get total entries and check if this batch includes the earliest
	var totalEntries int
	var isFirstBatch bool
	storage, storageErr := ledger.NewDefaultStorage()
	if storageErr == nil {
		totalEntries, isFirstBatch = computeFirstBatchInfo(storage, entries)
	}

	// Get project description from CLAUDE.md or default
	projectDesc := getProjectDescription()

	return &draft.RenderContext{
		Entries:            entries,
		RepoName:           repoName,
		Branch:             branch,
		AppendText:         appendFlag,
		TotalEntries:       totalEntries,
		IsFirstBatch:       isFirstBatch,
		ProjectDescription: projectDesc,
		Vars:               vars,
	}
}

// computeFirstBatchInfo returns total entry count and whether entries include the earliest.
func computeFirstBatchInfo(storage *ledger.Storage, entries []*ledger.Entry) (int, bool) {
	allEntries, err := storage.ListEntries()
	if err != nil || len(allEntries) == 0 {
		return len(entries), len(entries) > 0
	}

	// Find the earliest entry in the repo
	var earliest time.Time
	for _, e := range allEntries {
		if earliest.IsZero() || e.CreatedAt.Before(earliest) {
			earliest = e.CreatedAt
		}
	}

	// Check if current entries include the earliest
	isFirstBatch := false
	for _, e := range entries {
		if e.CreatedAt.Equal(earliest) {
			isFirstBatch = true
			break
		}
	}

	return len(allEntries), isFirstBatch
}

// getProjectDescription extracts project description from CLAUDE.md or returns default.
func getProjectDescription() string {
	root, err := git.RepoRoot()
	if err != nil {
		return ""
	}

	claudeMD := filepath.Join(root, "CLAUDE.md")
	content, err := os.ReadFile(claudeMD)
	if err != nil {
		return ""
	}

	// Extract first paragraph after the title (skip # Title line)
	lines := strings.Split(string(content), "\n")
	var desc strings.Builder
	inDesc := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" && inDesc {
			break // End of first paragraph
		}
		if !strings.HasPrefix(line, "#") && line != "" {
			if inDesc {
				desc.WriteString(" ")
			}
			desc.WriteString(line)
			inDesc = true
		}
	}

	return desc.String()
}

// parseTimeCutoffs parses --since and --until flags into time cutoffs.
func parseTimeCutoffs(printer *output.Printer, sinceFlag, untilFlag string) (time.Time, time.Time, error) {
	var sinceCutoff, untilCutoff time.Time
	var err error
	if sinceFlag != "" {
		if sinceCutoff, err = parseSinceValue(sinceFlag); err != nil {
			userErr := output.NewUserError(err.Error())
			printer.Error(userErr)
			return time.Time{}, time.Time{}, userErr
		}
	}
	if untilFlag != "" {
		if untilCutoff, err = parseUntilValue(untilFlag); err != nil {
			userErr := output.NewUserError(err.Error())
			printer.Error(userErr)
			return time.Time{}, time.Time{}, userErr
		}
	}
	return sinceCutoff, untilCutoff, nil
}

// getDraftEntries retrieves and validates the complete ledger before selecting entries.
func getDraftEntries(
	printer *output.Printer, lastFlag, sinceFlag, untilFlag, rangeFlag string,
) ([]*ledger.Entry, error) {
	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return nil, err
	}
	storage, err := ledger.NewDefaultStorage()
	if err != nil {
		printer.Error(err)
		return nil, err
	}
	allEntries, stats, err := storage.ListEntriesWithStats()
	if err != nil {
		printer.Error(err)
		return nil, err
	}
	if integrityErr := corruptEntriesError(stats); integrityErr != nil {
		printer.Error(integrityErr)
		return nil, integrityErr
	}
	return selectDraftEntries(printer, storage, allEntries, lastFlag, sinceFlag, untilFlag, rangeFlag)
}

func selectDraftEntries(
	printer *output.Printer, storage *ledger.Storage, allEntries []*ledger.Entry,
	lastFlag, sinceFlag, untilFlag, rangeFlag string,
) ([]*ledger.Entry, error) {
	sinceCutoff, untilCutoff, err := parseTimeCutoffs(printer, sinceFlag, untilFlag)
	if err != nil {
		return nil, err
	}
	entries := allEntries
	if rangeFlag != "" {
		entries, err = getEntriesByRangeFromEntries(printer, storage, entries, rangeFlag)
		if err != nil {
			return nil, err
		}
	}
	entries = applyQueryFilters(entries, sinceCutoff, untilCutoff, nil)
	sortEntriesByCreatedAt(entries)
	return limitDraftEntries(printer, entries, lastFlag)
}

func limitDraftEntries(printer *output.Printer, entries []*ledger.Entry, lastFlag string) ([]*ledger.Entry, error) {
	if lastFlag == "" {
		return entries, nil
	}
	count, err := strconv.Atoi(lastFlag)
	if err != nil || count <= 0 {
		userErr := output.NewUserError("--last must be a positive integer")
		printer.Error(userErr)
		return nil, userErr
	}
	if len(entries) > count {
		entries = entries[:count]
	}
	return entries, nil
}
