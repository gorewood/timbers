package git

import (
	"context"
	"net/mail"
	"os/exec"
	"strings"
)

// Identity is a name and email recorded in Git metadata.
type Identity struct {
	Name  string
	Email string
}

func parseCoAuthors(values string) []Identity {
	var identities []Identity
	for value := range strings.SplitSeq(values, "\x1e") {
		if identity, valid := parseIdentity(value); valid {
			identities = append(identities, identity)
		}
	}
	return identities
}

func parseIdentity(value string) (Identity, bool) {
	address, err := mail.ParseAddress(strings.TrimSpace(value))
	if err == nil && strings.TrimSpace(address.Name) != "" && validIdentityEmail(address.Address) {
		return Identity{Name: strings.TrimSpace(address.Name), Email: strings.TrimSpace(address.Address)}, true
	}
	value = strings.TrimSpace(value)
	open := strings.LastIndex(value, "<")
	if open <= 0 || !strings.HasSuffix(value, ">") {
		return Identity{}, false
	}
	identity := Identity{
		Name:  strings.Trim(strings.TrimSpace(value[:open]), `"`),
		Email: strings.TrimSpace(value[open+1 : len(value)-1]),
	}
	return identity, identity.Name != "" && validIdentityEmail(identity.Email)
}

func validIdentityEmail(email string) bool {
	email = strings.TrimSpace(email)
	local, domain, found := strings.Cut(email, "@")
	return found && strings.Count(email, "@") == 1 && local != "" && domain != "" &&
		!strings.ContainsAny(email, " \t\r\n<>")
}

// normalizeCoAuthors applies Git's repository mailmap to trailer identities.
// Failure leaves the already parsed identities unchanged.
func normalizeCoAuthors(commits []Commit) {
	input := make([]string, 0, len(commits))
	positions := make([][2]int, 0, len(commits))
	for commitIdx := range commits {
		for authorIdx, identity := range commits[commitIdx].CoAuthors {
			input = append(input, identity.Name+" <"+identity.Email+">")
			positions = append(positions, [2]int{commitIdx, authorIdx})
		}
	}
	if len(input) == 0 {
		return
	}

	cmd := exec.CommandContext(context.Background(), "git", "check-mailmap", "--stdin")
	cmd.Stdin = strings.NewReader(strings.Join(input, "\n") + "\n")
	out, err := cmd.Output()
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != len(positions) {
		return
	}
	for idx, line := range lines {
		if identity, valid := parseIdentity(line); valid {
			position := positions[idx]
			commits[position[0]].CoAuthors[position[1]] = identity
		}
	}
}
