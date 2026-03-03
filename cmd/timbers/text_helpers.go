// Package main provides the entry point for the timbers CLI.
package main

// truncateString truncates a string to maxLen, adding "..." if truncated.
func truncateString(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}
