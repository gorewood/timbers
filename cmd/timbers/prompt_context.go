package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/ledger"
	"github.com/rbergman/timbers/internal/prompt"
)

// buildRenderContext creates a RenderContext from entries and flags.
func buildRenderContext(entries []*ledger.Entry, appendFlag string) *prompt.RenderContext {
	repoName := ""
	if root, rootErr := git.RepoRoot(); rootErr == nil {
		repoName = filepath.Base(root)
	}
	branch, _ := git.CurrentBranch()

	// Get total entries and check if this batch includes the earliest
	totalEntries, isFirstBatch := computeFirstBatchInfo(entries)

	// Get project description from CLAUDE.md or default
	projectDesc := getProjectDescription()

	return &prompt.RenderContext{
		Entries:            entries,
		RepoName:           repoName,
		Branch:             branch,
		AppendText:         appendFlag,
		TotalEntries:       totalEntries,
		IsFirstBatch:       isFirstBatch,
		ProjectDescription: projectDesc,
	}
}

// computeFirstBatchInfo returns total entry count and whether entries include the earliest.
func computeFirstBatchInfo(entries []*ledger.Entry) (int, bool) {
	storage := ledger.NewStorage(nil)
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
