package ledger

import (
	"fmt"
	"net/mail"
	"slices"
	"strings"

	"github.com/gorewood/timbers/internal/git"
)

// Contributor source values are stable JSON provenance identifiers.
const (
	ContributorSourceGitAuthor    = "git-author"
	ContributorSourceCoAuthoredBy = "co-authored-by"
	ContributorSourceExplicit     = "explicit"
)

// ResolveContributors returns explicit identities when who is non-empty;
// otherwise it derives identities from commit authors and co-author trailers.
func ResolveContributors(commits []git.Commit, who []string) ([]Contributor, error) {
	if len(who) > 0 {
		contributors := make([]Contributor, 0, len(who))
		for _, value := range who {
			name, email, valid := parseExplicitContributor(value)
			if !valid {
				return nil, fmt.Errorf("invalid contributor %q: expected Name <email>", value)
			}
			contributors = append(contributors, Contributor{
				Name: name, Email: email,
				Sources: []string{ContributorSourceExplicit},
			})
		}
		return dedupeContributors(contributors), nil
	}

	var contributors []Contributor
	for _, commit := range commits {
		if validIdentity(commit.Author, commit.AuthorEmail) {
			contributors = append(contributors, Contributor{
				Name: commit.Author, Email: commit.AuthorEmail,
				Sources: []string{ContributorSourceGitAuthor},
			})
		}
		for _, identity := range commit.CoAuthors {
			if validIdentity(identity.Name, identity.Email) {
				contributors = append(contributors, Contributor{
					Name: identity.Name, Email: identity.Email,
					Sources: []string{ContributorSourceCoAuthoredBy},
				})
			}
		}
	}
	return dedupeContributors(contributors), nil
}

func validIdentity(name, email string) bool {
	return strings.TrimSpace(name) != "" && validEmail(email)
}

func validEmail(email string) bool {
	email = strings.TrimSpace(email)
	local, domain, found := strings.Cut(email, "@")
	return found && strings.Count(email, "@") == 1 && local != "" && domain != "" &&
		!strings.ContainsAny(email, " \t\r\n<>")
}

func parseExplicitContributor(value string) (string, string, bool) {
	if address, err := mail.ParseAddress(strings.TrimSpace(value)); err == nil &&
		strings.TrimSpace(address.Name) != "" && validEmail(address.Address) {
		return strings.TrimSpace(address.Name), strings.TrimSpace(address.Address), true
	}
	value = strings.TrimSpace(value)
	open := strings.LastIndex(value, "<")
	if open <= 0 || !strings.HasSuffix(value, ">") {
		return "", "", false
	}
	name := strings.Trim(strings.TrimSpace(value[:open]), `"`)
	email := strings.TrimSpace(value[open+1 : len(value)-1])
	return name, email, name != "" && validEmail(email)
}

func dedupeContributors(input []Contributor) []Contributor {
	byEmail := make(map[string]Contributor, len(input))
	for _, contributor := range input {
		key := strings.ToLower(contributor.Email)
		if existing, ok := byEmail[key]; ok {
			for _, source := range contributor.Sources {
				if !slices.Contains(existing.Sources, source) {
					existing.Sources = append(existing.Sources, source)
				}
			}
			byEmail[key] = existing
			continue
		}
		byEmail[key] = contributor
	}

	contributors := make([]Contributor, 0, len(byEmail))
	for _, contributor := range byEmail {
		slices.SortFunc(contributor.Sources, func(left, right string) int {
			return contributorSourceRank(left) - contributorSourceRank(right)
		})
		contributors = append(contributors, contributor)
	}
	slices.SortFunc(contributors, func(left, right Contributor) int {
		if compared := strings.Compare(strings.ToLower(left.Email), strings.ToLower(right.Email)); compared != 0 {
			return compared
		}
		return strings.Compare(left.Name, right.Name)
	})
	return contributors
}

func contributorSourceRank(source string) int {
	switch source {
	case ContributorSourceGitAuthor:
		return 0
	case ContributorSourceCoAuthoredBy:
		return 1
	default:
		return 2
	}
}
