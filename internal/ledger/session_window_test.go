package ledger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadSessionWindow covers the .timbersignore session-window directive
// across the supported grammar (Go time.ParseDuration), the safe-degradation
// paths (missing file, no directive, malformed value), and the multi-entry
// last-wins rule.
func TestLoadSessionWindow(t *testing.T) {
	cases := []struct {
		name      string
		content   string // ".timbersignore" body, "" = no file
		wantWin   time.Duration
		wantRaw   string
		wantError bool
	}{
		{
			name:    "no file uses default",
			content: "",
			wantWin: DefaultSessionWindow,
		},
		{
			name:    "no directive uses default",
			content: "vendor/\nauthor:dependabot*\nmsg:chore: changelog for v*\n",
			wantWin: DefaultSessionWindow,
		},
		{
			name:    "valid 4h override",
			content: "session-window: 4h\n",
			wantWin: 4 * time.Hour,
			wantRaw: "4h",
		},
		{
			name:    "valid 90m override (sub-hour)",
			content: "session-window: 90m\n",
			wantWin: 90 * time.Minute,
			wantRaw: "90m",
		},
		{
			name:    "valid composite 2h30m",
			content: "session-window:2h30m\n",
			wantWin: 2*time.Hour + 30*time.Minute,
			wantRaw: "2h30m",
		},
		{
			name:      "malformed 1d falls back to default with error",
			content:   "session-window: 1d\n",
			wantWin:   DefaultSessionWindow,
			wantRaw:   "1d",
			wantError: true,
		},
		{
			name:      "malformed '4 hours' falls back to default",
			content:   "session-window: 4 hours\n",
			wantWin:   DefaultSessionWindow,
			wantRaw:   "4 hours",
			wantError: true,
		},
		{
			name:      "zero value falls back to default",
			content:   "session-window: 0\n",
			wantWin:   DefaultSessionWindow,
			wantRaw:   "0",
			wantError: false, // 0 parses as zero duration; we coerce to default but don't flag as error per current impl
		},
		{
			name:      "negative value falls back to default",
			content:   "session-window: -1h\n",
			wantWin:   DefaultSessionWindow,
			wantRaw:   "-1h",
			wantError: false,
		},
		{
			name:    "comment lines are ignored",
			content: "# session-window: 1h\nsession-window: 6h\n",
			wantWin: 6 * time.Hour,
			wantRaw: "6h",
		},
		{
			name:    "inline trailing comment is stripped",
			content: "session-window: 12h  # half a day\n",
			wantWin: 12 * time.Hour,
			wantRaw: "12h",
		},
		{
			name:    "last directive wins",
			content: "session-window: 1h\nsession-window: 8h\n",
			wantWin: 8 * time.Hour,
			wantRaw: "8h",
		},
		{
			name: "empty value is ignored",
			//nolint:dupword // adjacent directives intentional for last-wins test
			content: "session-window:\n" + "session-window: 5h\n",
			wantWin: 5 * time.Hour,
			wantRaw: "5h",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if testCase.content != "" {
				path := filepath.Join(dir, timbersIgnoreFilename)
				if err := os.WriteFile(path, []byte(testCase.content), 0o600); err != nil {
					t.Fatalf("write .timbersignore: %v", err)
				}
			}
			got := LoadSessionWindow(dir)
			if got.Window != testCase.wantWin {
				t.Errorf("Window = %v, want %v", got.Window, testCase.wantWin)
			}
			if got.Raw != testCase.wantRaw {
				t.Errorf("Raw = %q, want %q", got.Raw, testCase.wantRaw)
			}
			if (got.ParseErr != nil) != testCase.wantError {
				t.Errorf("ParseErr = %v, wantError = %v", got.ParseErr, testCase.wantError)
			}
		})
	}
}

// TestLoadSessionWindow_DoesNotBreakSkipRuleParsing asserts that adding a
// session-window: line does not pollute the path/author/message skip rule
// sets — the directive lives on its own classification axis and the main
// readTimbersIgnore parser must skip it.
func TestLoadSessionWindow_DoesNotBreakSkipRuleParsing(t *testing.T) {
	dir := t.TempDir()
	content := "vendor/\nauthor:dependabot*\nmsg:chore: changelog for v*\nsession-window: 4h\n"
	if err := os.WriteFile(filepath.Join(dir, timbersIgnoreFilename), []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	rules, authors, messages, err := loadSkipConfig(dir)
	if err != nil {
		t.Fatalf("loadSkipConfig: %v", err)
	}

	// The session-window line must not appear as a path rule. The path
	// set we configured has only "vendor/", but rules also includes the
	// built-in defaults — we look for a synthetic match against the
	// session-window literal.
	for _, rule := range rules {
		if rule.pattern == "session-window: 4h" || rule.pattern == "session-window:4h" {
			t.Errorf("session-window: directive leaked into path rules as %q", rule.pattern)
		}
	}
	// Authors and messages should be unaffected.
	if len(authors) != 1 || authors[0] != "dependabot*" {
		t.Errorf("authors = %v, want [dependabot*]", authors)
	}
	if len(messages) != 1 || messages[0] != "chore: changelog for v*" {
		t.Errorf("messages = %v, want [chore: changelog for v*]", messages)
	}
}
