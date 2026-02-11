// Package git provides Git operations via exec for the timbers CLI.
//
// This package wraps git commands by shelling out to the git executable,
// capturing stdout/stderr and translating exit codes to appropriate errors.
// It provides general git operations used by the timbers CLI.
//
// # General Operations
//
// The package provides common git operations through simple function calls:
//
//	git.IsRepo()       // Check if current directory is a git repository
//	git.RepoRoot()     // Get the root directory of the repository
//	git.CurrentBranch() // Get the current branch name
//	git.HEAD()         // Get the current HEAD commit SHA
//
// # Running Git Commands
//
// For custom git commands, use Run or RunContext:
//
//	output, err := git.Run("status", "--short")
//	output, err := git.RunContext(ctx, "log", "--oneline", "-5")
//
// # Commit Operations
//
// For working with commits and commit history:
//
//	commits, err := git.LogRange(from, to)  // Get commits in a range
//	diffstat, err := git.GetDiffstat(a, b)  // Get file change statistics
//
// # Error Handling
//
// All functions return errors wrapped with appropriate exit codes:
//   - ExitUserError (1) for user errors like bad arguments
//   - ExitSystemError (2) for system errors like git not found
//
// Example:
//
//	if !git.IsRepo() {
//	    return output.NewSystemError("not in a git repository")
//	}
//	sha, err := git.HEAD()
//	if err != nil {
//	    return err // Error already wrapped with appropriate code
//	}
package git
