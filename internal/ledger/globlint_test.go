package ledger

import "testing"

func TestLooksLikeLiteralBracket(t *testing.T) {
	tests := []struct {
		name string
		glob string
		want bool
	}{
		{"dependabot bot suffix (the footgun)", "dependabot[bot]", true},
		{"renovate bot suffix", "renovate[bot]", true},
		{"plain prefix wildcard is fine", "dependabot*", false},
		{"substring wildcard is fine", "*dependabot*", false},
		{"no brackets at all", "alice@example.com", false},
		{"deliberate range class", "[a-z]*", false},
		{"deliberate negation class", "[!x]bar", false},
		{"caret negation class", "[^x]bar", false},
		{"single-char class is not flagged", "foo[a]", false},
		{"two-char word class flagged", "foo[ab]", true},
		{"unterminated bracket not flagged", "foo[bar", false},
		{"literal bracket mid-pattern", "build(deps)[bot] *", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LooksLikeLiteralBracket(tt.glob); got != tt.want {
				t.Errorf("LooksLikeLiteralBracket(%q) = %v, want %v", tt.glob, got, tt.want)
			}
		})
	}
}
