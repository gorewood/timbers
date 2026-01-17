// Package git provides Git operations via exec for the timbers CLI.
package git

import (
	"errors"
	"os"
	"testing"

	"github.com/steveyegge/timbers/internal/output"
)

// Helper to set up a test git repo with notes
func setupTestRepoWithNotes(t *testing.T, tmpDir string) {
	t.Helper()
	origDir, getWdErr := os.Getwd()
	if getWdErr != nil {
		t.Fatalf("failed to get current dir: %v", getWdErr)
	}

	if chdirErr := os.Chdir(tmpDir); chdirErr != nil {
		t.Fatalf("failed to change to temp dir: %v", chdirErr)
	}

	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Initialize git repo
	if _, initErr := Run("init"); initErr != nil {
		t.Fatalf("failed to init repo: %v", initErr)
	}

	// Configure git for commits
	if _, cfgErr := Run("config", "user.email", "test@example.com"); cfgErr != nil {
		t.Fatalf("failed to config email: %v", cfgErr)
	}
	if _, cfgErr := Run("config", "user.name", "Test User"); cfgErr != nil {
		t.Fatalf("failed to config name: %v", cfgErr)
	}

	// Create initial commit
	if writeErr := os.WriteFile("test.txt", []byte("test content"), 0600); writeErr != nil {
		t.Fatalf("failed to write test file: %v", writeErr)
	}
	if _, addErr := Run("add", "test.txt"); addErr != nil {
		t.Fatalf("failed to stage file: %v", addErr)
	}
	if _, commitErr := Run("commit", "-m", "initial commit"); commitErr != nil {
		t.Fatalf("failed to commit: %v", commitErr)
	}
}

func TestNotesRefExists(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T, tmpDir string)
		want      bool
	}{
		{
			name: "notes ref does not exist",
			setupFunc: func(t *testing.T, tmpDir string) {
				setupTestRepoWithNotes(t, tmpDir)
				// No notes written yet
			},
			want: false,
		},
		{
			name: "notes ref exists after write",
			setupFunc: func(t *testing.T, tmpDir string) {
				setupTestRepoWithNotes(t, tmpDir)
				// Write a note to create the ref
				head, err := HEAD()
				if err != nil {
					t.Fatalf("failed to get HEAD: %v", err)
				}
				if _, err := Run("notes", "--ref="+notesRefName, "add", "-m", "test note", head); err != nil {
					t.Fatalf("failed to write note: %v", err)
				}
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupFunc(t, tmpDir)

			got := NotesRefExists()
			if got != tt.want {
				t.Errorf("NotesRefExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotesConfigured(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T, tmpDir string, remote string)
		remote    string
		want      bool
	}{
		{
			name: "notes not configured for remote",
			setupFunc: func(t *testing.T, tmpDir string, remote string) {
				setupTestRepoWithNotes(t, tmpDir)
				// Create a bare repo to act as remote
				if _, err := Run("remote", "add", remote, "/tmp/"+remote); err != nil {
					t.Fatalf("failed to add remote: %v", err)
				}
			},
			remote: "origin",
			want:   false,
		},
		{
			name: "notes configured for remote",
			setupFunc: func(t *testing.T, tmpDir string, remote string) {
				setupTestRepoWithNotes(t, tmpDir)
				if _, err := Run("remote", "add", remote, "/tmp/"+remote); err != nil {
					t.Fatalf("failed to add remote: %v", err)
				}
				// Configure notes fetch
				if _, err := Run("config", "--add", "remote."+remote+".fetch", "+refs/notes/timbers:refs/notes/timbers"); err != nil {
					t.Fatalf("failed to config notes fetch: %v", err)
				}
			},
			remote: "origin",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupFunc(t, tmpDir, tt.remote)

			got := NotesConfigured(tt.remote)
			if got != tt.want {
				t.Errorf("NotesConfigured(%q) = %v, want %v", tt.remote, got, tt.want)
			}
		})
	}
}

func TestReadNote(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T, tmpDir string) string
		wantErr   bool
		wantMsg   string
	}{
		{
			name: "read existing note",
			setupFunc: func(t *testing.T, tmpDir string) string {
				setupTestRepoWithNotes(t, tmpDir)
				head, err := HEAD()
				if err != nil {
					t.Fatalf("failed to get HEAD: %v", err)
				}
				noteContent := "This is a test note"
				if _, err := Run("notes", "--ref="+notesRefName, "add", "-m", noteContent, head); err != nil {
					t.Fatalf("failed to write note: %v", err)
				}
				return head
			},
			wantErr: false,
			wantMsg: "This is a test note",
		},
		{
			name: "note not found",
			setupFunc: func(t *testing.T, tmpDir string) string {
				setupTestRepoWithNotes(t, tmpDir)
				head, err := HEAD()
				if err != nil {
					t.Fatalf("failed to get HEAD: %v", err)
				}
				return head
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			commit := tt.setupFunc(t, tmpDir)

			got, err := ReadNote(commit)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadNote() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && string(got) != tt.wantMsg {
				t.Errorf("ReadNote() = %q, want %q", string(got), tt.wantMsg)
			}

			if tt.wantErr {
				var exitErr *output.ExitError
				if !errors.As(err, &exitErr) {
					t.Errorf("ReadNote() error should be *output.ExitError, got %T", err)
				}
			}
		})
	}
}

func TestWriteNote(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(t *testing.T, tmpDir string) string
		content    string
		force      bool
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "write new note",
			setupFunc: func(t *testing.T, tmpDir string) string {
				setupTestRepoWithNotes(t, tmpDir)
				head, err := HEAD()
				if err != nil {
					t.Fatalf("failed to get HEAD: %v", err)
				}
				return head
			},
			content: "New note content",
			force:   false,
			wantErr: false,
		},
		{
			name: "overwrite note without force fails",
			setupFunc: func(t *testing.T, tmpDir string) string {
				setupTestRepoWithNotes(t, tmpDir)
				head, err := HEAD()
				if err != nil {
					t.Fatalf("failed to get HEAD: %v", err)
				}
				// Write initial note
				if _, err := Run("notes", "--ref="+notesRefName, "add", "-m", "original", head); err != nil {
					t.Fatalf("failed to write initial note: %v", err)
				}
				return head
			},
			content:    "Overwrite attempt",
			force:      false,
			wantErr:    true,
			wantErrMsg: "git command failed",
		},
		{
			name: "overwrite note with force succeeds",
			setupFunc: func(t *testing.T, tmpDir string) string {
				setupTestRepoWithNotes(t, tmpDir)
				head, err := HEAD()
				if err != nil {
					t.Fatalf("failed to get HEAD: %v", err)
				}
				// Write initial note
				if _, err := Run("notes", "--ref="+notesRefName, "add", "-m", "original", head); err != nil {
					t.Fatalf("failed to write initial note: %v", err)
				}
				return head
			},
			content: "Forced overwrite",
			force:   true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			commit := tt.setupFunc(t, tmpDir)

			err := WriteNote(commit, tt.content, tt.force)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteNote() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify the note was written
				got, readErr := ReadNote(commit)
				if readErr != nil {
					t.Errorf("failed to read back written note: %v", readErr)
					return
				}
				if string(got) != tt.content {
					t.Errorf("wrote note %q, but read back %q", tt.content, string(got))
				}
			}

			if tt.wantErr && tt.wantErrMsg != "" {
				if !errors.Is(err, errors.New(tt.wantErrMsg)) && err.Error() != tt.wantErrMsg {
					// Check if error message contains expected substring
					if !contains(err.Error(), tt.wantErrMsg) {
						t.Errorf("WriteNote() error message %q, want to contain %q", err.Error(), tt.wantErrMsg)
					}
				}
			}
		})
	}
}

func TestListNotedCommits(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(t *testing.T, tmpDir string)
		wantCount  int
		wantCommit bool
	}{
		{
			name: "no notes returns empty",
			setupFunc: func(t *testing.T, tmpDir string) {
				setupTestRepoWithNotes(t, tmpDir)
			},
			wantCount: 0,
		},
		{
			name: "lists commits with notes",
			setupFunc: func(t *testing.T, tmpDir string) {
				setupTestRepoWithNotes(t, tmpDir)
				head, err := HEAD()
				if err != nil {
					t.Fatalf("failed to get HEAD: %v", err)
				}
				if _, err := Run("notes", "--ref="+notesRefName, "add", "-m", "test note", head); err != nil {
					t.Fatalf("failed to write note: %v", err)
				}
			},
			wantCount:  1,
			wantCommit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupFunc(t, tmpDir)

			got, err := ListNotedCommits()
			if err != nil {
				t.Errorf("ListNotedCommits() error = %v", err)
				return
			}

			if len(got) != tt.wantCount {
				t.Errorf("ListNotedCommits() returned %d commits, want %d", len(got), tt.wantCount)
			}

			if tt.wantCommit && len(got) > 0 {
				if got[0] == "" {
					t.Error("ListNotedCommits() returned empty commit SHA")
				}
			}
		})
	}
}

func TestConfigureNotesFetch(t *testing.T) {
	tests := []struct {
		name    string
		remote  string
		wantErr bool
	}{
		{
			name:    "configure notes for origin",
			remote:  "origin",
			wantErr: false,
		},
		{
			name:    "configure notes for other remote",
			remote:  "upstream",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			setupTestRepoWithNotes(t, tmpDir)

			// Add remote
			if _, err := Run("remote", "add", tt.remote, "/tmp/test-repo"); err != nil {
				t.Fatalf("failed to add remote: %v", err)
			}

			err := ConfigureNotesFetch(tt.remote)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigureNotesFetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify config was set
				if !NotesConfigured(tt.remote) {
					t.Errorf("ConfigureNotesFetch() did not configure notes for %q", tt.remote)
				}
			}
		})
	}
}

func TestPushNotes(t *testing.T) {
	t.Run("push to non-existent remote fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupTestRepoWithNotes(t, tmpDir)

		// Write a note so there's something to push
		head, headErr := HEAD()
		if headErr != nil {
			t.Fatalf("failed to get HEAD: %v", headErr)
		}
		if writeErr := WriteNote(head, "test note", false); writeErr != nil {
			t.Fatalf("failed to write note: %v", writeErr)
		}

		// Try to push without a remote configured
		pushErr := PushNotes("nonexistent")
		if pushErr == nil {
			t.Error("PushNotes() expected error for non-existent remote")
		}
	})

	t.Run("push to valid remote", func(t *testing.T) {
		// Create a bare repo to act as remote
		remoteDir := t.TempDir()
		origDir, getWdErr := os.Getwd()
		if getWdErr != nil {
			t.Fatalf("failed to get current dir: %v", getWdErr)
		}

		if chdirErr := os.Chdir(remoteDir); chdirErr != nil {
			t.Fatalf("failed to change to remote dir: %v", chdirErr)
		}
		if _, initErr := Run("init", "--bare"); initErr != nil {
			_ = os.Chdir(origDir)
			t.Fatalf("failed to init bare repo: %v", initErr)
		}
		_ = os.Chdir(origDir)

		// Create local repo
		tmpDir := t.TempDir()
		setupTestRepoWithNotes(t, tmpDir)

		// Add the bare repo as remote
		if _, remoteErr := Run("remote", "add", "origin", remoteDir); remoteErr != nil {
			t.Fatalf("failed to add remote: %v", remoteErr)
		}

		// Push main branch first so we have a ref on remote
		if _, pushBranchErr := Run("push", "-u", "origin", "master"); pushBranchErr != nil {
			// Try main if master fails
			if _, pushMainErr := Run("push", "-u", "origin", "main"); pushMainErr != nil {
				t.Logf("push branch failed (expected in some setups): %v / %v", pushBranchErr, pushMainErr)
			}
		}

		// Write a note
		head, headErr := HEAD()
		if headErr != nil {
			t.Fatalf("failed to get HEAD: %v", headErr)
		}
		if writeErr := WriteNote(head, "test note for push", false); writeErr != nil {
			t.Fatalf("failed to write note: %v", writeErr)
		}

		// Push notes
		pushErr := PushNotes("origin")
		if pushErr != nil {
			t.Errorf("PushNotes() error = %v, want nil", pushErr)
		}
	})
}

func TestFetchNotes(t *testing.T) {
	t.Run("fetch from non-existent remote fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupTestRepoWithNotes(t, tmpDir)

		fetchErr := FetchNotes("nonexistent")
		if fetchErr == nil {
			t.Error("FetchNotes() expected error for non-existent remote")
		}
	})

	t.Run("fetch from remote with notes", func(t *testing.T) {
		// Create a bare repo to act as remote
		remoteDir := t.TempDir()
		origDir, getWdErr := os.Getwd()
		if getWdErr != nil {
			t.Fatalf("failed to get current dir: %v", getWdErr)
		}

		if chdirErr := os.Chdir(remoteDir); chdirErr != nil {
			t.Fatalf("failed to change to remote dir: %v", chdirErr)
		}
		if _, initErr := Run("init", "--bare"); initErr != nil {
			_ = os.Chdir(origDir)
			t.Fatalf("failed to init bare repo: %v", initErr)
		}
		_ = os.Chdir(origDir)

		// Create first local repo and push notes
		tmpDir1 := t.TempDir()
		setupTestRepoWithNotes(t, tmpDir1)
		if _, remoteErr := Run("remote", "add", "origin", remoteDir); remoteErr != nil {
			t.Fatalf("failed to add remote: %v", remoteErr)
		}
		// Push branch
		if _, pushBranchErr := Run("push", "-u", "origin", "master"); pushBranchErr != nil {
			if _, pushMainErr := Run("push", "-u", "origin", "main"); pushMainErr != nil {
				t.Logf("push branch failed: %v / %v", pushBranchErr, pushMainErr)
			}
		}
		// Write and push a note
		head, headErr := HEAD()
		if headErr != nil {
			t.Fatalf("failed to get HEAD: %v", headErr)
		}
		if writeErr := WriteNote(head, "note to fetch", false); writeErr != nil {
			t.Fatalf("failed to write note: %v", writeErr)
		}
		if pushErr := PushNotes("origin"); pushErr != nil {
			t.Fatalf("failed to push notes: %v", pushErr)
		}

		// Create second local repo and fetch notes
		tmpDir2 := t.TempDir()
		if chdirErr := os.Chdir(tmpDir2); chdirErr != nil {
			t.Fatalf("failed to change to second repo: %v", chdirErr)
		}
		t.Cleanup(func() { _ = os.Chdir(origDir) })

		if _, cloneErr := Run("clone", remoteDir, "."); cloneErr != nil {
			t.Fatalf("failed to clone: %v", cloneErr)
		}

		// Fetch notes
		fetchErr := FetchNotes("origin")
		if fetchErr != nil {
			t.Errorf("FetchNotes() error = %v, want nil", fetchErr)
			return
		}

		// Verify notes were fetched
		if !NotesRefExists() {
			t.Error("FetchNotes() did not fetch notes ref")
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
