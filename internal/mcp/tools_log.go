package mcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// LogInput is the input for the log tool.
type LogInput struct {
	What     string   `json:"what"               jsonschema:"what was done (required)"`
	Why      string   `json:"why"                jsonschema:"why - design decision, not feature description (required)"`
	How      string   `json:"how"                jsonschema:"how - approach and implementation (required)"`
	Tags     []string `json:"tags,omitempty"      jsonschema:"tags for categorization"`
	WorkItem string   `json:"work_item,omitempty" jsonschema:"work item reference in system:id format"`
}

// LogOutput is the output for the log tool.
type LogOutput struct {
	Entry *ledger.Entry `json:"entry" jsonschema:"the created ledger entry"`
}

func handleLog(storage *ledger.Storage) mcp.ToolHandlerFor[LogInput, LogOutput] {
	return func(_ context.Context, _ *mcp.CallToolRequest, input LogInput) (*mcp.CallToolResult, LogOutput, error) {
		if err := validateLogInput(input); err != nil {
			return nil, LogOutput{}, err
		}

		commits, _, pendingErr := storage.GetPendingCommits()
		if pendingErr != nil && !errors.Is(pendingErr, ledger.ErrStaleAnchor) {
			return nil, LogOutput{}, fmt.Errorf("getting pending commits: %w", pendingErr)
		}
		if len(commits) == 0 {
			return nil, LogOutput{}, errors.New("no pending commits to document")
		}

		entry, err := buildLogEntry(storage, commits, input)
		if err != nil {
			return nil, LogOutput{}, err
		}

		if err := storage.WriteEntry(entry, false); err != nil {
			return nil, LogOutput{}, fmt.Errorf("writing entry: %w", err)
		}

		return nil, LogOutput{Entry: entry}, nil
	}
}

// validateLogInput checks that required fields are present.
func validateLogInput(input LogInput) error {
	if input.What == "" {
		return errors.New("what is required")
	}
	if input.Why == "" {
		return errors.New("why is required")
	}
	if input.How == "" {
		return errors.New("how is required")
	}
	return nil
}

// buildLogEntry creates a ledger entry from pending commits and user input.
func buildLogEntry(
	storage *ledger.Storage,
	commits []git.Commit,
	input LogInput,
) (*ledger.Entry, error) {
	anchor := commits[0].SHA
	commitSHAs := make([]string, len(commits))
	for idx, commit := range commits {
		commitSHAs[idx] = commit.SHA
	}

	rangeStr := ""
	if len(commits) > 1 {
		rangeStr = commits[len(commits)-1].Short + ".." + commits[0].Short
	}

	fromRef := commits[len(commits)-1].SHA + "^"
	diffstat, _ := storage.GetDiffstat(fromRef, anchor)
	now := time.Now().UTC()

	var workItems []ledger.WorkItem
	if input.WorkItem != "" {
		parsed, err := parseWorkItem(input.WorkItem)
		if err != nil {
			return nil, err
		}
		workItems = []ledger.WorkItem{parsed}
	}

	return &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        ledger.GenerateID(anchor, now),
		CreatedAt: now,
		UpdatedAt: now,
		Workset: ledger.Workset{
			AnchorCommit: anchor,
			Commits:      commitSHAs,
			Range:        rangeStr,
			Diffstat: &ledger.Diffstat{
				Files:      diffstat.Files,
				Insertions: diffstat.Insertions,
				Deletions:  diffstat.Deletions,
			},
		},
		Summary: ledger.Summary{
			What: input.What,
			Why:  input.Why,
			How:  input.How,
		},
		Tags:      input.Tags,
		WorkItems: workItems,
	}, nil
}
