package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSkipRules_NoFile(t *testing.T) {
	repoRoot := t.TempDir()
	rules, err := loadSkipRules(repoRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != len(compiledDefaultSkipRules) {
		t.Errorf("expected %d default rules, got %d", len(compiledDefaultSkipRules), len(rules))
	}
}

func TestLoadSkipRules_EmptyRepoRoot(t *testing.T) {
	rules, err := loadSkipRules("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != len(compiledDefaultSkipRules) {
		t.Errorf("expected %d default rules, got %d", len(compiledDefaultSkipRules), len(rules))
	}
}

func TestLoadSkipRules_MergesWithDefaults(t *testing.T) {
	repoRoot := t.TempDir()
	contents := `# project-specific extensions
vendor/
*.lock
third_party/

# blank line above is fine
docs/generated/   # inline comment
`
	if err := os.WriteFile(filepath.Join(repoRoot, ".timbersignore"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	rules, err := loadSkipRules(repoRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantTotal := len(compiledDefaultSkipRules) + 4
	if len(rules) != wantTotal {
		t.Errorf("expected %d rules, got %d", wantTotal, len(rules))
	}

	// Verify match behavior on each loaded pattern.
	cases := map[string]bool{
		"vendor/foo.go":         true,
		"vendor.txt":            false,
		"go.lock":               true,
		"third_party/lib/foo.h": true,
		"third_party_other.txt": false,
		"docs/generated/api.md": true,
		"docs/handwritten.md":   false,
		// Defaults still active
		".gitignore":  true,
		".gitignores": false,
		"cmd/main.go": false,
	}
	for path, want := range cases {
		if got := matchAny(rules, path); got != want {
			t.Errorf("matchAny(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestLoadSkipRules_OnlyComments(t *testing.T) {
	repoRoot := t.TempDir()
	contents := "# nothing here\n\n# just comments\n"
	if err := os.WriteFile(filepath.Join(repoRoot, ".timbersignore"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	rules, err := loadSkipRules(repoRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != len(compiledDefaultSkipRules) {
		t.Errorf("expected %d default rules (no extras), got %d", len(compiledDefaultSkipRules), len(rules))
	}
}

func TestIndexInlineComment(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"vendor/", -1},
		{"vendor/  # libs", 9},
		{"vendor/\t# tab", 8},
		{"#leading", -1}, // leading-# already filtered upstream
		{"foo#bar", -1},  // no whitespace before #
		{"foo #bar", 4},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := indexInlineComment(tt.s); got != tt.want {
				t.Errorf("indexInlineComment(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}
