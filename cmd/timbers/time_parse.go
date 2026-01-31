// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// durationRegex matches duration strings like "24h", "7d", "30d".
var durationRegex = regexp.MustCompile(`^(\d+)([hdwm])$`)

// parseSinceValue parses a --since value into a time.Time cutoff.
// Accepts:
//   - Durations: "24h", "48h", "7d", "2w", "1m" (hours, days, weeks, months)
//   - Dates: "2026-01-17" (YYYY-MM-DD format)
//
// Returns the cutoff time (entries created after this time should be included).
func parseSinceValue(value string) (time.Time, error) {
	t, err := parseTimeValue(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --since value %q; use duration (24h, 7d, 2w) or date (2026-01-17)", value)
	}
	return t, nil
}

// parseUntilValue parses a --until value into a time.Time cutoff.
// Accepts:
//   - Durations: "24h", "48h", "7d", "2w", "1m" (hours, days, weeks, months)
//   - Dates: "2026-01-17" (YYYY-MM-DD format)
//
// Returns the cutoff time (entries created before this time should be included).
// For dates, returns end of day (23:59:59) to include the full day.
func parseUntilValue(value string) (time.Time, error) {
	cutoff, err := parseTimeValue(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --until value %q; use duration (24h, 7d, 2w) or date (2026-01-17)", value)
	}
	// For date-only values (YYYY-MM-DD), extend to end of day
	if len(value) == 10 && value[4] == '-' && value[7] == '-' {
		cutoff = cutoff.Add(24*time.Hour - time.Second)
	}
	return cutoff, nil
}

// parseTimeValue parses a time value (duration or date) into a time.Time.
func parseTimeValue(value string) (time.Time, error) {
	// Try parsing as duration first
	if matches := durationRegex.FindStringSubmatch(value); len(matches) == 3 {
		return parseDuration(matches[1], matches[2])
	}

	// Try parsing as date (YYYY-MM-DD)
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}

	// Try parsing as datetime (YYYY-MM-DDTHH:MM:SS)
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time value: %s", value)
}

// parseDuration converts a numeric value and unit to a time cutoff.
func parseDuration(numStr, unit string) (time.Time, error) {
	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 {
		return time.Time{}, fmt.Errorf("invalid duration number: %s", numStr)
	}

	now := time.Now().UTC()

	switch unit {
	case "h":
		return now.Add(-time.Duration(num) * time.Hour), nil
	case "d":
		return now.AddDate(0, 0, -num), nil
	case "w":
		return now.AddDate(0, 0, -num*7), nil
	case "m":
		return now.AddDate(0, -num, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unknown duration unit: %s", unit)
	}
}
