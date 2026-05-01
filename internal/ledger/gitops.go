package ledger

import "github.com/gorewood/timbers/internal/git"

// realGitOps implements GitOps using the actual git package functions.
type realGitOps struct{}

func (realGitOps) HEAD() (string, error) {
	return git.HEAD()
}

func (realGitOps) Log(fromRef, toRef string) ([]git.Commit, error) {
	return git.Log(fromRef, toRef)
}

func (realGitOps) CommitsReachableFrom(sha string) ([]git.Commit, error) {
	return git.CommitsReachableFrom(sha)
}

func (realGitOps) IsAncestorOf(ancestor, descendant string) bool {
	return git.IsAncestorOf(ancestor, descendant)
}

func (realGitOps) GetDiffstat(fromRef, toRef string) (git.Diffstat, error) {
	return git.GetDiffstat(fromRef, toRef)
}

func (realGitOps) CommitFiles(sha string) ([]string, error) {
	return git.CommitFiles(sha)
}

func (realGitOps) CommitFilesMulti(shas []string) (map[string][]string, error) {
	return git.CommitFilesMulti(shas)
}

func (realGitOps) DiffNameOnly(fromRef, toRef, pathPrefix string) ([]string, error) {
	return git.DiffNameOnly(fromRef, toRef, pathPrefix)
}
