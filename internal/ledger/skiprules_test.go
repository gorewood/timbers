package ledger

import "testing"

func TestParseSkipRule(t *testing.T) {
	tests := []struct {
		pattern  string
		wantKind skipKind
		wantPat  string
	}{
		{".timbers/", skipPrefix, ".timbers/"},
		{".beads/", skipPrefix, ".beads/"},
		{".gitignore", skipExact, ".gitignore"},
		{".github/CODEOWNERS", skipExact, ".github/CODEOWNERS"},
		{"*.lock", skipSuffix, ".lock"},
		{"*.md", skipSuffix, ".md"},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := parseSkipRule(tt.pattern)
			if got.kind != tt.wantKind {
				t.Errorf("kind = %v, want %v", got.kind, tt.wantKind)
			}
			if got.pattern != tt.wantPat {
				t.Errorf("pattern = %q, want %q", got.pattern, tt.wantPat)
			}
		})
	}
}

func TestSkipRule_Match(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		// Directory prefix
		{"prefix matches under dir", ".timbers/", ".timbers/2026/foo.json", true},
		{"prefix matches dir itself with slash", ".timbers/", ".timbers/", true},
		{"prefix does not match outside dir", ".timbers/", "src/timbers/foo.go", false},
		{"prefix does not match adjacent prefix", ".beads/", ".beads", false},

		// Exact path — the bug fix: must NOT match longer files
		{"exact matches itself", ".gitignore", ".gitignore", true},
		{"exact does NOT match .gitignores", ".gitignore", ".gitignores", false},
		{"exact does NOT match prefix overlap", ".gitignore", ".gitignore.bak", false},
		{"exact does NOT match suffix overlap", ".gitignore", "foo.gitignore", false},
		{"exact matches deep path", ".github/CODEOWNERS", ".github/CODEOWNERS", true},
		{"exact does not match wrong dir", ".github/CODEOWNERS", "docs/CODEOWNERS", false},

		// Suffix
		{"suffix matches extension", "*.lock", "go.lock", true},
		{"suffix matches deep path", "*.lock", "vendor/foo.lock", true},
		{"suffix does not match different ext", "*.lock", "go.sum", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := parseSkipRule(tt.pattern)
			if got := r.match(tt.path); got != tt.want {
				t.Errorf("match(%q) on pattern %q = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestDefaultSkipRules_Coverage(t *testing.T) {
	// Spot-check that the expanded defaults match the intended files
	// and do NOT accidentally over-skip near-miss filenames.
	type tc struct {
		path string
		skip bool
	}
	cases := []tc{
		// Should be skipped
		{".timbers/2026/01/15/tb_x.json", true},
		{".beads/issues.jsonl", true},
		{".gitignore", true},
		{".gitattributes", true},
		{".editorconfig", true},
		{".github/dependabot.yml", true},
		{".github/CODEOWNERS", true},
		{".github/FUNDING.yml", true},
		{".github/pull_request_template.md", true},
		{"renovate.json", true},
		{"dependabot.yml", true},

		// Should NOT be skipped (the latent-bug regression cases)
		{".gitignores", false},
		{".gitattributes.bak", false},
		{".editorconfigs", false},
		{"renovate.jsonc", false},
		{"dependabot.yml.example", false},

		// Should NOT be skipped (substantive .github/ files)
		{".github/workflows/ci.yml", false},
		{".github/workflows/release.yml", false},
		{".github/actions/setup/action.yml", false},

		// Generic source / docs are not skipped
		{"cmd/main.go", false},
		{"README.md", false},
		{"LICENSE", false},
		{"AGENTS.md", false},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			if got := matchAny(compiledDefaultSkipRules, c.path); got != c.skip {
				t.Errorf("matchAny(defaults, %q) = %v, want %v", c.path, got, c.skip)
			}
		})
	}
}
