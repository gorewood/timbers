package git

import "testing"

func TestParseCoAuthorsKeepsBotAndDropsMalformedIdentities(t *testing.T) {
	values := "dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\x1eMissing Email"
	got := parseCoAuthors(values)
	if len(got) != 1 || got[0].Name != "dependabot[bot]" ||
		got[0].Email != "49699333+dependabot[bot]@users.noreply.github.com" {
		t.Fatalf("parseCoAuthors = %#v, want one bot identity", got)
	}
}
