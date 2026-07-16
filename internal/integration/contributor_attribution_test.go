//go:build integration

package integration

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gorewood/timbers/internal/ledger"
)

func TestContributorAttributionSurvivesSquashAndPrune(t *testing.T) {
	repo := newTestRepo(t)
	repo.git("config", "user.name", "Alice Alias")
	repo.git("config", "user.email", "alice-alt@example.com")
	repo.createFile(".mailmap", "Alice <alice@example.com> Alice Alias <alice-alt@example.com>\nBob <bob@example.com> Bob Alias <bob-alt@example.com>\n")
	repo.commit("add mailmap")
	repo.createFile("alice.txt", "alice\n")
	repo.commitWithBody("alice work", "Co-authored-by: Bob Alias <bob-alt@example.com>")
	repo.git("config", "user.name", "Bob")
	repo.git("config", "user.email", "bob@example.com")
	repo.createFile("bob.txt", "bob\n")
	repo.commit("bob work")

	repo.timbersOK("log", "shared work", "--why", "reason", "--how", "method", "--json")
	before := queryLatestEntry(t, repo)
	want := []ledger.Contributor{
		{Name: "Alice", Email: "alice@example.com", Sources: []string{ledger.ContributorSourceGitAuthor}},
		{Name: "Bob", Email: "bob@example.com", Sources: []string{ledger.ContributorSourceGitAuthor, ledger.ContributorSourceCoAuthoredBy}},
	}
	if !reflect.DeepEqual(before.Contributors, want) {
		t.Fatalf("contributors before rewrite = %#v, want %#v", before.Contributors, want)
	}

	anchor := before.Workset.AnchorCommit
	root := repo.git("rev-list", "--max-parents=0", "HEAD")
	repo.git("reset", "--soft", root)
	repo.git("commit", "--amend", "-m", "squashed history")
	repo.git("reflog", "expire", "--expire=now", "--all")
	repo.git("gc", "--prune=now")
	if _, err := repo.gitMayFail("cat-file", "-e", anchor+"^{commit}"); err == nil {
		t.Fatalf("original workset anchor %s still exists after prune", anchor)
	}

	after := queryLatestEntry(t, repo)
	if !reflect.DeepEqual(after.Contributors, want) {
		t.Fatalf("contributors after rewrite = %#v, want %#v", after.Contributors, want)
	}
	var draft struct {
		Entries []ledger.Entry `json:"entries"`
	}
	if err := json.Unmarshal([]byte(repo.timbersOK("draft", "release-notes", "--last", "1", "--json")), &draft); err != nil {
		t.Fatalf("decode draft JSON: %v", err)
	}
	if len(draft.Entries) != 1 || !reflect.DeepEqual(draft.Entries[0].Contributors, want) {
		t.Fatalf("draft contributors = %#v, want persisted attribution", draft.Entries)
	}
}

func queryLatestEntry(t *testing.T, repo *testRepo) ledger.Entry {
	t.Helper()
	var entries []ledger.Entry
	if err := json.Unmarshal([]byte(repo.timbersOK("query", "--last", "1", "--json")), &entries); err != nil {
		t.Fatalf("decode query JSON: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("query returned %d entries, want 1", len(entries))
	}
	return entries[0]
}
