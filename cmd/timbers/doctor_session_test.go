package main

import (
	"os"
	"strings"
	"testing"
)

// TestCheckSessionIdentity_EmptyUserEmail asserts the doctor surfaces a
// warning when git config user.email is unset — the load-bearing safe-
// degradation signal for the cross-agent debt classifier. Without this
// warning, an operator with a misconfigured environment sees a passing
// doctor while the gate silently falls back to "all commits in-session,"
// defeating the entire foreign-author detection path.
func TestCheckSessionIdentity_EmptyUserEmail(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// Initialize a git repo without setting user.email.
	runGit(t, dir, "init")
	// Force git to ignore any user-level config that might leak user.email in.
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("GIT_AUTHOR_EMAIL", "")
	t.Setenv("GIT_COMMITTER_EMAIL", "")

	got := checkSessionIdentity()
	if got.Status != checkWarn {
		t.Errorf("Status = %v, want checkWarn (empty user.email must warn)", got.Status)
	}
	if !strings.Contains(got.Message, "unset") {
		t.Errorf("Message = %q, expected to mention 'unset'", got.Message)
	}
	if got.Hint == "" {
		t.Errorf("Hint must guide the operator to fix it; got empty")
	}
}

// TestCheckSessionIdentity_SetUserEmail asserts the doctor passes when
// user.email is set, with the email value in the message for visibility.
func TestCheckSessionIdentity_SetUserEmail(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "configured@example.com")

	got := checkSessionIdentity()
	if got.Status != checkPass {
		t.Errorf("Status = %v, want checkPass (user.email is set)", got.Status)
	}
	if !strings.Contains(got.Message, "configured@example.com") {
		t.Errorf("Message = %q, expected to contain the email", got.Message)
	}
}
