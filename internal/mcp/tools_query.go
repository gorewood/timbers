package mcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gorewood/timbers/internal/ledger"
)

// QueryInput is the input for the query tool.
type QueryInput struct {
	Last  int      `json:"last,omitempty"  jsonschema:"retrieve last N entries"`
	Since string   `json:"since,omitempty" jsonschema:"retrieve entries since duration (24h, 7d) or ISO date"`
	Until string   `json:"until,omitempty" jsonschema:"retrieve entries until duration (24h, 7d) or ISO date"`
	Tags  []string `json:"tags,omitempty"  jsonschema:"filter by tags (OR logic)"`
}

// QueryOutput is the output for the query tool.
type QueryOutput struct {
	Count   int             `json:"count"   jsonschema:"number of entries returned"`
	Entries []*ledger.Entry `json:"entries" jsonschema:"matching ledger entries"`
}

func handleQuery(storage *ledger.Storage) mcp.ToolHandlerFor[QueryInput, QueryOutput] {
	return func(_ context.Context, _ *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, QueryOutput, error) {
		if input.Last == 0 && input.Since == "" && input.Until == "" {
			return nil, QueryOutput{}, errors.New("specify last, since, or until to retrieve entries")
		}

		sinceCutoff, untilCutoff, err := parseQueryCutoffs(input)
		if err != nil {
			return nil, QueryOutput{}, err
		}

		entries, err := queryEntries(storage, input, sinceCutoff, untilCutoff)
		if err != nil {
			return nil, QueryOutput{}, err
		}

		return nil, QueryOutput{Count: len(entries), Entries: entries}, nil
	}
}

// parseQueryCutoffs parses since/until input strings into time values.
func parseQueryCutoffs(input QueryInput) (time.Time, time.Time, error) {
	var sinceCutoff, untilCutoff time.Time
	if input.Since != "" {
		parsed, err := parseDurationOrDate(input.Since)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid since value: %w", err)
		}
		sinceCutoff = parsed
	}
	if input.Until != "" {
		parsed, err := parseDurationOrDate(input.Until)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid until value: %w", err)
		}
		untilCutoff = parsed
	}
	return sinceCutoff, untilCutoff, nil
}

// queryEntries retrieves and filters entries based on query input.
func queryEntries(
	storage *ledger.Storage, input QueryInput,
	sinceCutoff, untilCutoff time.Time,
) ([]*ledger.Entry, error) {
	// If only --last with no other filters, use optimized path
	if canUseLastOnlyPath(input, sinceCutoff, untilCutoff) {
		return storage.GetLastNEntries(input.Last)
	}

	entries, err := storage.ListEntries()
	if err != nil {
		return nil, fmt.Errorf("listing entries: %w", err)
	}

	entries = applyQueryFilters(entries, sinceCutoff, untilCutoff, input.Tags)

	ledger.SortEntriesByCreatedAt(entries)

	if input.Last > 0 && len(entries) > input.Last {
		entries = entries[:input.Last]
	}

	return entries, nil
}

// canUseLastOnlyPath checks if we can use the optimized GetLastNEntries path.
func canUseLastOnlyPath(input QueryInput, sinceCutoff, untilCutoff time.Time) bool {
	return sinceCutoff.IsZero() && untilCutoff.IsZero() && len(input.Tags) == 0 && input.Last > 0
}

// applyQueryFilters applies time range and tag filters to entries.
func applyQueryFilters(
	entries []*ledger.Entry,
	sinceCutoff, untilCutoff time.Time,
	tags []string,
) []*ledger.Entry {
	if !sinceCutoff.IsZero() {
		entries = ledger.FilterEntriesSince(entries, sinceCutoff)
	}
	if !untilCutoff.IsZero() {
		entries = ledger.FilterEntriesUntil(entries, untilCutoff)
	}
	if len(tags) > 0 {
		entries = ledger.FilterEntriesByTags(entries, tags)
	}
	return entries
}
