// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"strings"

	"github.com/rbergman/timbers/internal/ledger"
	"github.com/rbergman/timbers/internal/output"
)

// validateLogInput validates the input arguments and flags.
func validateLogInput(args []string, flags logFlags) (string, error) {
	if len(args) == 0 {
		return "", output.NewUserError(
			"missing required <what> argument; usage: timbers log \"<what>\" --why \"<why>\" --how \"<how>\"")
	}
	what := args[0]
	if strings.TrimSpace(what) == "" {
		return "", output.NewUserError(
			"missing required <what> argument; usage: timbers log \"<what>\" --why \"<why>\" --how \"<how>\"")
	}

	if flags.rangeStr != "" {
		if err := validateRangeFormat(flags.rangeStr); err != nil {
			return "", err
		}
	}

	if !flags.minor {
		if flags.why == "" {
			return "", output.NewUserError("--why flag is required (use --minor for trivial changes)")
		}
		if flags.how == "" {
			return "", output.NewUserError("--how flag is required (use --minor for trivial changes)")
		}
	}

	return what, nil
}

// validateRangeFormat checks that a range string contains ".."
func validateRangeFormat(rangeStr string) error {
	if !strings.Contains(rangeStr, "..") {
		return output.NewUserError("--range must contain '..' (e.g., abc123..def456)")
	}
	return nil
}

// parseWorkItems parses and validates work item strings.
func parseWorkItems(items []string) ([]ledger.WorkItem, error) {
	result := make([]ledger.WorkItem, 0, len(items))
	for _, item := range items {
		system, itemID, err := parseWorkItem(item)
		if err != nil {
			return nil, err
		}
		result = append(result, ledger.WorkItem{System: system, ID: itemID})
	}
	return result, nil
}

// parseWorkItem parses a single work item string in format "system:id".
func parseWorkItem(item string) (string, string, error) {
	if item == "" {
		return "", "", output.NewUserError("--work-item cannot be empty")
	}

	parts := strings.SplitN(item, ":", 2)
	if len(parts) != 2 {
		return "", "", output.NewUserError(
			fmt.Sprintf("--work-item must be in format system:id, got %q", item))
	}

	system := strings.TrimSpace(parts[0])
	itemID := strings.TrimSpace(parts[1])

	if system == "" {
		return "", "", output.NewUserError(
			fmt.Sprintf("--work-item system cannot be empty in %q", item))
	}
	if itemID == "" {
		return "", "", output.NewUserError(
			fmt.Sprintf("--work-item id cannot be empty in %q", item))
	}

	return system, itemID, nil
}
