package ledger

import (
	"testing"

	"github.com/gorewood/timbers/internal/git"
)

func TestParseRevertedSHAs(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"empty", "", nil},
		{"no trailer", "Some unrelated body text.", nil},
		{
			name: "single full SHA",
			body: "This reverts commit a4e80a72c11d2a9d8f2c1a3b4d5e6f7081234567.",
			want: []string{"a4e80a72c11d2a9d8f2c1a3b4d5e6f7081234567"},
		},
		{
			name: "leading prose then trailer",
			body: "Reverting because the migration broke prod.\n\nThis reverts commit a4e80a72c11d2a9d8f2c1a3b4d5e6f7081234567.",
			want: []string{"a4e80a72c11d2a9d8f2c1a3b4d5e6f7081234567"},
		},
		{
			name: "short SHA still tolerated",
			body: "This reverts commit a4e80a7.",
			want: []string{"a4e80a7"},
		},
		{
			name: "multiple reverts in one commit",
			body: "This reverts commit a4e80a72c11d2a9d8f2c1a3b4d5e6f7081234567.\nThis reverts commit b60c773d2e4f5a6b7c8d9e0f1a2b3c4d5e6f7080.",
			want: []string{
				"a4e80a72c11d2a9d8f2c1a3b4d5e6f7081234567",
				"b60c773d2e4f5a6b7c8d9e0f1a2b3c4d5e6f7080",
			},
		},
		{
			name: "trailer must be at line start",
			body: "  This reverts commit a4e80a72c11d2a9d8f2c1a3b4d5e6f7081234567.",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRevertedSHAs(tt.body)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d shas, want %d (got=%v)", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("sha[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsRevertCommit(t *testing.T) {
	tests := []struct {
		name string
		c    git.Commit
		want bool
	}{
		{
			name: "subject + trailer = revert",
			c: git.Commit{
				Subject: `Revert "feat: add thing"`,
				Body:    "This reverts commit a4e80a72c11d2a9d8f2c1a3b4d5e6f7081234567.",
			},
			want: true,
		},
		{
			name: "subject only without trailer = NOT revert",
			c: git.Commit{
				Subject: `Revert "feat: hand-written"`,
				Body:    "User decided to undo this.",
			},
			want: false,
		},
		{
			name: "trailer only without revert subject = NOT revert",
			c: git.Commit{
				Subject: "fix: cherry-picked",
				Body:    "This reverts commit a4e80a72c11d2a9d8f2c1a3b4d5e6f7081234567.",
			},
			want: false,
		},
		{
			name: "ordinary commit",
			c:    git.Commit{Subject: "feat: add X", Body: ""},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRevertCommit(tt.c); got != tt.want {
				t.Errorf("isRevertCommit = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDocumentedRevert(t *testing.T) {
	full := "a4e80a72c11d2a9d8f2c1a3b4d5e6f7081234567"
	other := "b60c773d2e4f5a6b7c8d9e0f1a2b3c4d5e6f7080"
	docSet := map[string]bool{full: true}

	tests := []struct {
		name string
		c    git.Commit
		want bool
	}{
		{
			name: "documented revert via full SHA",
			c: git.Commit{
				Subject: `Revert "feat: x"`,
				Body:    "This reverts commit " + full + ".",
			},
			want: true,
		},
		{
			name: "documented revert via short SHA",
			c: git.Commit{
				Subject: `Revert "feat: x"`,
				Body:    "This reverts commit a4e80a7.",
			},
			want: true,
		},
		{
			name: "undocumented revert (sha not in entries)",
			c: git.Commit{
				Subject: `Revert "feat: x"`,
				Body:    "This reverts commit " + other + ".",
			},
			want: false,
		},
		{
			name: "multi-revert: any undocumented sha keeps it pending",
			c: git.Commit{
				Subject: `Revert "feat: x"`,
				Body:    "This reverts commit " + full + ".\nThis reverts commit " + other + ".",
			},
			want: false,
		},
		{
			name: "non-revert commit",
			c:    git.Commit{Subject: "feat: x", Body: ""},
			want: false,
		},
		{
			name: "empty docSet (no entries) — never auto-skip",
			c: git.Commit{
				Subject: `Revert "feat: x"`,
				Body:    "This reverts commit " + full + ".",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := docSet
			if tt.name == "empty docSet (no entries) — never auto-skip" {
				set = nil
			}
			if got := isDocumentedRevert(tt.c, set); got != tt.want {
				t.Errorf("isDocumentedRevert = %v, want %v", got, tt.want)
			}
		})
	}
}
