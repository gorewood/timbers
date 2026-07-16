package ledger

import (
	"reflect"
	"testing"

	"github.com/gorewood/timbers/internal/git"
)

func TestResolveContributorsFromCommits(t *testing.T) {
	commits := []git.Commit{
		{
			Author:      "Bob",
			AuthorEmail: "bob@example.com",
			CoAuthors: []git.Identity{
				{Name: "Alice", Email: "alice@example.com"},
				{Name: "Bob", Email: "BOB@example.com"},
				{Name: "Missing email"},
			},
		},
		{Author: "Alice", AuthorEmail: "alice@example.com"},
		{Author: "dependabot[bot]", AuthorEmail: "49699333+dependabot[bot]@users.noreply.github.com"},
		{Author: "Invalid", AuthorEmail: "not-an-email"},
	}

	got, err := ResolveContributors(commits, nil)
	if err != nil {
		t.Fatalf("ResolveContributors: %v", err)
	}
	want := []Contributor{
		{Name: "dependabot[bot]", Email: "49699333+dependabot[bot]@users.noreply.github.com", Sources: []string{ContributorSourceGitAuthor}},
		{Name: "Alice", Email: "alice@example.com", Sources: []string{ContributorSourceGitAuthor, ContributorSourceCoAuthoredBy}},
		{Name: "Bob", Email: "bob@example.com", Sources: []string{ContributorSourceGitAuthor, ContributorSourceCoAuthoredBy}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("contributors = %#v, want %#v", got, want)
	}
}

func TestResolveContributorsExplicitReplacesAutomatic(t *testing.T) {
	commits := []git.Commit{{Author: "Git Author", AuthorEmail: "git@example.com"}}
	who := []string{"Pair Two <pair2@example.com>", "Pair One <pair1@example.com>"}

	got, err := ResolveContributors(commits, who)
	if err != nil {
		t.Fatalf("ResolveContributors: %v", err)
	}
	want := []Contributor{
		{Name: "Pair One", Email: "pair1@example.com", Sources: []string{ContributorSourceExplicit}},
		{Name: "Pair Two", Email: "pair2@example.com", Sources: []string{ContributorSourceExplicit}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("contributors = %#v, want %#v", got, want)
	}
}

func TestResolveContributorsRejectsIncompleteExplicitIdentity(t *testing.T) {
	for _, who := range []string{"pair@example.com", "Pair <not-an-email>"} {
		if _, err := ResolveContributors(nil, []string{who}); err == nil {
			t.Errorf("ResolveContributors(%q) expected error", who)
		}
	}
}
