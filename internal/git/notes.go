// Package git provides Git operations via exec for the timbers CLI.
package git

import (
	"errors"
	"strings"

	"github.com/gorewood/timbers/internal/output"
)

const notesRefName = "refs/notes/timbers"

// NotesRefExists checks if the notes ref exists.
// Returns true if refs/notes/timbers exists in the repository.
func NotesRefExists() bool {
	_, err := Run("show-ref", "--verify", notesRefName)
	return err == nil
}

// NotesConfigured checks if notes fetch is configured for the given remote.
// Returns true if the remote has notes fetch configuration.
func NotesConfigured(remote string) bool {
	out, err := Run("config", "--get-all", "remote."+remote+".fetch")
	if err != nil {
		return false
	}
	// Check if any of the fetch specs includes notes
	for spec := range strings.SplitSeq(out, "\n") {
		if strings.Contains(spec, notesRefName) {
			return true
		}
	}
	return false
}

// ReadNote reads the note content for a given commit.
// Returns the note content as bytes.
// Returns a user error (exit code 1) if the note is not found.
// Returns a system error (exit code 2) for other git errors.
func ReadNote(commit string) ([]byte, error) {
	out, err := Run("notes", "--ref="+notesRefName, "show", commit)
	if err != nil {
		// Check if error is "no note found" vs other git errors
		var exitErr *output.ExitError
		if errors.As(err, &exitErr) {
			// Parse stderr to check for "no note found"
			errMsg := exitErr.Message
			if strings.Contains(errMsg, "no note found") || strings.Contains(errMsg, "no such object") {
				return nil, output.NewUserError("note not found for commit: " + commit)
			}
		}
		return nil, err
	}
	return []byte(out), nil
}

// WriteNote writes a note to a commit.
// If force is true, overwrites an existing note.
// If force is false, returns an error if the note already exists.
func WriteNote(commit string, content string, force bool) error {
	args := []string{"notes", "--ref=" + notesRefName, "add"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, "-m", content, commit)
	_, err := Run(args...)
	return err
}

// ListNotedCommits returns a list of all commits that have notes.
// Returns an empty slice if no notes exist.
func ListNotedCommits() ([]string, error) {
	out, err := Run("notes", "--ref="+notesRefName, "list")
	if err != nil {
		var exitErr *output.ExitError
		if errors.As(err, &exitErr) {
			// If notes ref doesn't exist, return empty list
			if strings.Contains(exitErr.Message, "no such object") || strings.Contains(exitErr.Message, "no notes") {
				return []string{}, nil
			}
		}
		return nil, err
	}

	if out == "" {
		return []string{}, nil
	}

	// Parse output: each line is "note-sha commit-sha"
	// We need the commit SHA (second part)
	lines := strings.Split(out, "\n")
	var commits []string
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			// Format is "note-sha commit-sha", split on whitespace
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				commits = append(commits, parts[1])
			}
		}
	}
	return commits, nil
}

// ConfigureNotesFetch adds the notes fetch configuration for a remote.
// If already configured, this is a no-op.
func ConfigureNotesFetch(remote string) error {
	// Check if already configured
	if NotesConfigured(remote) {
		return nil
	}

	// Add the fetch config
	fetchSpec := "+" + notesRefName + ":" + notesRefName
	_, err := Run("config", "--add", "remote."+remote+".fetch", fetchSpec)
	return err
}

// PushNotes pushes the notes ref to the given remote.
// Returns an error if the push fails.
func PushNotes(remote string) error {
	_, err := Run("push", remote, notesRefName)
	return err
}

// FetchNotes fetches the notes ref from the given remote.
// Returns an error if the fetch fails.
func FetchNotes(remote string) error {
	refSpec := notesRefName + ":" + notesRefName
	_, err := Run("fetch", remote, refSpec)
	return err
}
