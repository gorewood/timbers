package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchesSkipAuthor(t *testing.T) {
	tests := []struct {
		name        string
		globs       []string
		authorEmail string
		authorName  string
		want        bool
	}{
		{
			name:  "no globs returns false",
			globs: nil,
			want:  false,
		},
		{
			name:        "exact email match",
			globs:       []string{"bot@example.com"},
			authorEmail: "bot@example.com",
			want:        true,
		},
		{
			name:        "wildcard domain match",
			globs:       []string{"*@bot.example.com"},
			authorEmail: "q-redshifted@bot.example.com",
			want:        true,
		},
		{
			name:        "exact name match (fallback when email differs)",
			globs:       []string{"q-redshifted"},
			authorEmail: "noreply@anthropic.com",
			authorName:  "q-redshifted",
			want:        true,
		},
		{
			name:        "no match",
			globs:       []string{"q-redshifted", "*@bot.example.com"},
			authorEmail: "alice@example.com",
			authorName:  "Alice",
			want:        false,
		},
		{
			// GitHub bot account names like "dependabot[bot]" contain
			// '[' which filepath.Match treats as a character class
			// opener — the literal glob won't match. Workaround is to
			// use a prefix wildcard: "dependabot*". This case documents
			// the recommended pattern.
			name:       "GitHub bot prefix wildcard",
			globs:      []string{"dependabot*"},
			authorName: "dependabot[bot]",
			want:       true,
		},
		{
			name:        "empty author fields no panic",
			globs:       []string{"*"},
			authorEmail: "",
			authorName:  "",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesSkipAuthor(tt.globs, tt.authorEmail, tt.authorName)
			if got != tt.want {
				t.Errorf("matchesSkipAuthor(%v, %q, %q) = %v, want %v",
					tt.globs, tt.authorEmail, tt.authorName, got, tt.want)
			}
		})
	}
}

func TestLoadSkipConfig_AuthorLines(t *testing.T) {
	t.Run("missing file returns defaults and no authors", func(t *testing.T) {
		dir := t.TempDir()
		rules, authors, err := loadSkipConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(authors) != 0 {
			t.Errorf("expected no authors, got %v", authors)
		}
		// Default path rules should still be loaded.
		if len(rules) == 0 {
			t.Error("expected built-in default rules, got none")
		}
	})

	t.Run("mixed paths and authors parsed correctly", func(t *testing.T) {
		dir := t.TempDir()
		content := `# this repo's skip config
vendor/                          # path: vendored libs
*.lock                           # path: lockfiles by suffix

# bot authors
author:q-redshifted              # name match
author:*@bot.example.com         # email-domain glob
author:dependabot*               # prefix wildcard for [bot] suffix
`
		if err := os.WriteFile(filepath.Join(dir, ".timbersignore"), []byte(content), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}

		rules, authors, err := loadSkipConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Built-in defaults + 2 path rules from the file.
		if len(rules) < len(compiledDefaultSkipRules)+2 {
			t.Errorf("expected built-in rules + at least 2 path rules, got %d", len(rules))
		}

		wantAuthors := []string{"q-redshifted", "*@bot.example.com", "dependabot*"}
		if len(authors) != len(wantAuthors) {
			t.Fatalf("got %d authors, want %d (%v)", len(authors), len(wantAuthors), authors)
		}
		for i := range wantAuthors {
			if authors[i] != wantAuthors[i] {
				t.Errorf("authors[%d] = %q, want %q", i, authors[i], wantAuthors[i])
			}
		}
	})

	t.Run("malformed author glob is silently dropped", func(t *testing.T) {
		dir := t.TempDir()
		// `[` opens a character class that's never closed — filepath.Match
		// returns ErrBadPattern, and our loader drops it without failing.
		content := "author:good-author\nauthor:[broken\nauthor:another-good\n"
		if err := os.WriteFile(filepath.Join(dir, ".timbersignore"), []byte(content), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}

		_, authors, err := loadSkipConfig(dir)
		if err != nil {
			t.Fatalf("loader must not fail on bad globs, got %v", err)
		}
		// Only the valid globs survive; bad one is dropped.
		want := []string{"good-author", "another-good"}
		if len(authors) != len(want) {
			t.Fatalf("got %d authors, want %d (%v)", len(authors), len(want), authors)
		}
	})

	t.Run("author: prefix with empty glob is skipped", func(t *testing.T) {
		dir := t.TempDir()
		// Three lines: empty author glob, whitespace-only author glob,
		// then a real one. First two must be silently dropped.
		//nolint:dupword // "author:" repeats by design across distinct lines
		content := "author:\nauthor:   \nauthor:real-bot\n"
		if err := os.WriteFile(filepath.Join(dir, ".timbersignore"), []byte(content), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}

		_, authors, err := loadSkipConfig(dir)
		if err != nil {
			t.Fatalf("loader: %v", err)
		}
		if len(authors) != 1 || authors[0] != "real-bot" {
			t.Errorf("expected [real-bot], got %v", authors)
		}
	})
}
