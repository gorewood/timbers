package draft

import "testing"

func TestSanitizeLLMOutput(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "clean output unchanged",
			in:   "## [0.12.0] - 2026-02-28\n\n### Added\n- New feature",
			want: "## [0.12.0] - 2026-02-28\n\n### Added\n- New feature",
		},
		{
			name: "strips here is preamble",
			in:   "Here is the changelog:\n\n## [0.12.0] - 2026-02-28",
			want: "## [0.12.0] - 2026-02-28",
		},
		{
			name: "strips now I have enough context",
			in:   "Now I have enough context to generate the changelog.\n\n## [0.12.0]",
			want: "## [0.12.0]",
		},
		{
			name: "strips I'll generate preamble",
			in:   "I'll generate the release notes based on the entries.\n\n## New Features",
			want: "## New Features",
		},
		{
			name: "strips certainly preamble",
			in:   "Certainly! Here's the decision log:\n\n## ADR-1: Storage",
			want: "## ADR-1: Storage",
		},
		{
			name: "strips let me know signoff",
			in:   "## Standup\n- Fixed auth\n\nLet me know if you need anything else!",
			want: "## Standup\n- Fixed auth",
		},
		{
			name: "strips would you like signoff",
			in:   "## Report\n- Done\n\nWould you like me to adjust the format?",
			want: "## Report\n- Done",
		},
		{
			name: "strips both preamble and signoff",
			in:   "Here is the changelog:\n\n## Changes\n- Fix\n\nLet me know if you want changes.",
			want: "## Changes\n- Fix",
		},
		{
			name: "case insensitive",
			in:   "HERE IS the report:\n\n## Report",
			want: "## Report",
		},
		{
			name: "based on preamble",
			in:   "Based on the entries provided, here's the changelog:\n\n## Changes",
			want: "## Changes",
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
		{
			name: "preserves content that starts with common words",
			in:   "## Here Is How Auth Works\n\nDetails follow.",
			want: "## Here Is How Auth Works\n\nDetails follow.",
		},
		{
			name: "only strips first few preamble lines",
			in:   "Sure!\nI'll generate this now.\nLooking at the entries:\n\n## Actual Content",
			want: "## Actual Content",
		},
		{
			name: "does not strip body content matching patterns",
			in:   "## Report\n\nBased on the auth module changes, we refactored.\n\nLet me explain the design.",
			want: "## Report\n\nBased on the auth module changes, we refactored.\n\nLet me explain the design.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeLLMOutput(tt.in)
			if got != tt.want {
				t.Errorf("SanitizeLLMOutput():\n  got:  %q\n  want: %q", got, tt.want)
			}
		})
	}
}

func TestMatchesAnyPrefix(t *testing.T) {
	tests := []struct {
		line     string
		patterns []string
		want     bool
	}{
		{"Here is the output", preamblePatterns, true},
		{"here's what I found", preamblePatterns, true},
		{"## Changelog", preamblePatterns, false},
		{"Let me know if", signoffPatterns, true},
		{"- Let me fix this", signoffPatterns, false}, // bullet point, not sign-off
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := matchesAnyPrefix(tt.line, tt.patterns)
			if got != tt.want {
				t.Errorf("matchesAnyPrefix(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
