// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"strings"

	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/ledger"
	"github.com/rbergman/timbers/internal/output"
)

// validateBasicInput validates basic input before commits are fetched.
// This only validates range format; content validation happens in resolveLogContent.
func validateBasicInput(_ []string, flags logFlags) error {
	if flags.rangeStr != "" {
		if err := validateRangeFormat(flags.rangeStr); err != nil {
			return err
		}
	}
	return nil
}

// resolveLogContent determines what/why/how values based on mode (auto, minor, or manual).
// Returns the what value and potentially modified flags with why/how populated.
func resolveLogContent(args []string, flags logFlags, commits []git.Commit) (string, logFlags, error) {
	if flags.auto {
		return resolveAutoContent(args, flags, commits)
	}
	return resolveManualContent(args, flags)
}

// resolveAutoContent extracts what/why/how from commit messages.
func resolveAutoContent(args []string, flags logFlags, commits []git.Commit) (string, logFlags, error) {
	// Extract content from commits
	what, why, how := extractAutoContent(commits)

	// Allow user to override with explicit args/flags
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		what = args[0]
	}
	if flags.why != "" {
		why = flags.why
	}
	if flags.how != "" {
		how = flags.how
	}

	// Update flags with extracted values
	result := flags
	result.why = why
	result.how = how

	return what, result, nil
}

// resolveManualContent validates and returns manual input content.
func resolveManualContent(args []string, flags logFlags) (string, logFlags, error) {
	if len(args) == 0 {
		return "", flags, output.NewUserError(
			"missing required <what> argument; usage: timbers log \"<what>\" --why \"<why>\" --how \"<how>\"")
	}
	what := args[0]
	if strings.TrimSpace(what) == "" {
		return "", flags, output.NewUserError(
			"missing required <what> argument; usage: timbers log \"<what>\" --why \"<why>\" --how \"<how>\"")
	}

	if !flags.minor {
		if flags.why == "" {
			return "", flags, output.NewUserError("--why flag is required (use --minor or --auto for alternatives)")
		}
		if flags.how == "" {
			return "", flags, output.NewUserError("--how flag is required (use --minor or --auto for alternatives)")
		}
	}

	return what, flags, nil
}

// extractAutoContent extracts what/why/how from commit messages.
// - what: commit subjects joined with "; "
// - why: first body paragraph from first commit with body content
// - how: remaining body content after first paragraph
func extractAutoContent(commits []git.Commit) (what, why, how string) {
	// Build "what" from commit subjects
	subjects := make([]string, 0, len(commits))
	for _, c := range commits {
		if c.Subject != "" {
			subjects = append(subjects, c.Subject)
		}
	}
	what = strings.Join(subjects, "; ")
	if what == "" {
		what = "Auto-documented"
	}

	// Extract why/how from first commit with body content
	for _, c := range commits {
		body := strings.TrimSpace(c.Body)
		if body == "" {
			continue
		}

		// Split body into paragraphs (separated by blank lines)
		paragraphs := splitIntoParagraphs(body)
		if len(paragraphs) == 0 {
			continue
		}

		// First paragraph is "why"
		why = paragraphs[0]

		// Remaining paragraphs form "how"
		if len(paragraphs) > 1 {
			how = strings.Join(paragraphs[1:], "\n\n")
		}
		break
	}

	// Default values if nothing extracted
	if why == "" {
		why = "Auto-documented"
	}
	if how == "" {
		how = "Auto-documented"
	}

	return what, why, how
}

// splitIntoParagraphs splits text into paragraphs separated by blank lines.
func splitIntoParagraphs(text string) []string {
	lines := strings.Split(text, "\n")
	var paragraphs []string
	var current []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(current) > 0 {
				paragraphs = append(paragraphs, strings.Join(current, "\n"))
				current = nil
			}
		} else {
			current = append(current, line)
		}
	}

	if len(current) > 0 {
		paragraphs = append(paragraphs, strings.Join(current, "\n"))
	}

	return paragraphs
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
