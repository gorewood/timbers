package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchesSkipMessage(t *testing.T) {
	tests := []struct {
		name    string
		globs   []string
		subject string
		want    bool
	}{
		{
			name:  "no globs returns false",
			globs: nil,
			want:  false,
		},
		{
			name:    "empty subject returns false",
			globs:   []string{"*"},
			subject: "",
			want:    false,
		},
		{
			name:    "changelog glob matches release commit",
			globs:   []string{"chore: changelog for v*"},
			subject: "chore: changelog for v0.22.4",
			want:    true,
		},
		{
			name:    "changelog glob does not match unrelated commit",
			globs:   []string{"chore: changelog for v*"},
			subject: "feat: add the thing",
			want:    false,
		},
		{
			name:    "glob must match the whole subject, not a prefix",
			globs:   []string{"chore: changelog for v*"},
			subject: "revert: chore: changelog for v0.22.4",
			want:    false,
		},
		{
			name:    "second glob in set matches",
			globs:   []string{"chore: release *", "chore: changelog for v*"},
			subject: "chore: changelog for v1.2.3",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesSkipMessage(tt.globs, tt.subject); got != tt.want {
				t.Errorf("matchesSkipMessage(%v, %q) = %v, want %v", tt.globs, tt.subject, got, tt.want)
			}
		})
	}
}

func TestLoadSkipConfig_MessageLines(t *testing.T) {
	t.Run("mixed paths, authors, and messages parsed into separate sets", func(t *testing.T) {
		dir := t.TempDir()
		content := `vendor/
author:dependabot*
msg:chore: changelog for v*
msg:chore: release *
`
		if err := os.WriteFile(filepath.Join(dir, ".timbersignore"), []byte(content), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}

		_, authors, messages, err := loadSkipConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(authors) != 1 || authors[0] != "dependabot*" {
			t.Errorf("expected [dependabot*], got %v", authors)
		}
		want := []string{"chore: changelog for v*", "chore: release *"}
		if len(messages) != len(want) {
			t.Fatalf("got %d messages, want %d (%v)", len(messages), len(want), messages)
		}
		for i := range want {
			if messages[i] != want[i] {
				t.Errorf("messages[%d] = %q, want %q", i, messages[i], want[i])
			}
		}
	})

	t.Run("malformed message glob is silently dropped", func(t *testing.T) {
		dir := t.TempDir()
		// "[" opens an unterminated character class — filepath.Match rejects it.
		content := "msg:good *\nmsg:[broken\nmsg:also good *\n"
		if err := os.WriteFile(filepath.Join(dir, ".timbersignore"), []byte(content), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		_, _, messages, err := loadSkipConfig(dir)
		if err != nil {
			t.Fatalf("loader must not fail on bad globs, got %v", err)
		}
		want := []string{"good *", "also good *"}
		if len(messages) != len(want) {
			t.Fatalf("got %d messages, want %d (%v)", len(messages), len(want), messages)
		}
	})
}
