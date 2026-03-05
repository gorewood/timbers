package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePreCommitHook(t *testing.T) {
	t.Run("without chain", func(t *testing.T) {
		got := GeneratePreCommitHook(false, "")
		if !strings.HasPrefix(got, "#!/bin/sh") {
			t.Error("expected shebang")
		}
		if !strings.Contains(got, "timbers hook run pre-commit") {
			t.Error("expected timbers hook command")
		}
		if strings.Contains(got, ".backup") {
			t.Error("should not contain backup chain")
		}
		if !strings.Contains(got, "exit $rc") {
			t.Error("expected exit code propagation")
		}
	})

	t.Run("with chain default dir", func(t *testing.T) {
		got := GeneratePreCommitHook(true, "")
		if !strings.Contains(got, "timbers hook run pre-commit") {
			t.Error("expected timbers hook command")
		}
		if !strings.Contains(got, ".git/hooks/pre-commit.backup") {
			t.Error("expected default backup path")
		}
	})

	t.Run("with chain custom dir", func(t *testing.T) {
		got := GeneratePreCommitHook(true, ".beads/hooks")
		if !strings.Contains(got, ".beads/hooks/pre-commit.backup") {
			t.Error("expected custom backup path, got: " + got)
		}
	})
}

func TestDescribeInstallAction(t *testing.T) {
	tests := []struct {
		name         string
		existingHook bool
		chain        bool
		force        bool
		want         string
	}{
		{"no existing hook", false, false, false, "would install"},
		{"existing with force", true, false, true, "would overwrite"},
		{"existing with chain", true, true, false, "would backup and chain"},
		{"existing no flags", true, false, false, "would fail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DescribeInstallAction(tt.existingHook, tt.chain, tt.force)
			if !strings.Contains(got, tt.want) {
				t.Errorf("DescribeInstallAction(%v,%v,%v) = %q, want to contain %q",
					tt.existingHook, tt.chain, tt.force, got, tt.want)
			}
		})
	}
}

func TestDescribeUninstallAction(t *testing.T) {
	tests := []struct {
		name      string
		installed bool
		hasBackup bool
		want      string
	}{
		{"not installed", false, false, "no timbers hook installed"},
		{"installed with backup", true, true, "would remove and restore backup"},
		{"installed no backup", true, false, "would remove"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DescribeUninstallAction(tt.installed, tt.hasBackup)
			if !strings.Contains(got, tt.want) {
				t.Errorf("DescribeUninstallAction(%v,%v) = %q, want to contain %q",
					tt.installed, tt.hasBackup, got, tt.want)
			}
		})
	}
}

func TestHookExists(t *testing.T) {
	dir := t.TempDir()

	t.Run("nonexistent", func(t *testing.T) {
		if HookExists(filepath.Join(dir, "nope")) {
			t.Error("expected false for nonexistent file")
		}
	})

	t.Run("exists", func(t *testing.T) {
		path := filepath.Join(dir, "hook")
		writeTestFile(t, path, "#!/bin/sh\n")
		if !HookExists(path) {
			t.Error("expected true for existing file")
		}
	})
}

func TestCheckHookStatus(t *testing.T) {
	dir := t.TempDir()

	t.Run("no file returns empty status", func(t *testing.T) {
		status := CheckHookStatus(filepath.Join(dir, "nonexistent"))
		if status.Installed || status.Chained {
			t.Error("expected empty status for nonexistent file")
		}
	})

	t.Run("non-timbers hook", func(t *testing.T) {
		path := filepath.Join(dir, "other-hook")
		writeTestFile(t, path, "#!/bin/sh\necho hello\n")
		status := CheckHookStatus(path)
		if status.Installed {
			t.Error("expected not installed for non-timbers hook")
		}
	})

	t.Run("timbers hook without chain", func(t *testing.T) {
		path := filepath.Join(dir, "timbers-hook")
		writeTestFile(t, path, "#!/bin/sh\ntimbers hook run pre-commit\n")
		status := CheckHookStatus(path)
		if !status.Installed {
			t.Error("expected installed")
		}
		if status.Chained {
			t.Error("expected not chained")
		}
	})

	t.Run("timbers hook with chain", func(t *testing.T) {
		path := filepath.Join(dir, "chained-hook")
		writeTestFile(t, path, "#!/bin/sh\ntimbers hook run pre-commit\nexec .git/hooks/pre-commit.backup\n")
		status := CheckHookStatus(path)
		if !status.Installed {
			t.Error("expected installed")
		}
		if !status.Chained {
			t.Error("expected chained")
		}
	})
}

func TestGeneratePostCommitHook(t *testing.T) {
	got := GeneratePostCommitHook()
	if !strings.HasPrefix(got, "#!/bin/sh") {
		t.Error("expected shebang")
	}
	if !strings.Contains(got, "timbers hook run post-commit") {
		t.Error("expected timbers hook command")
	}
}

func TestCheckPostCommitHookStatus(t *testing.T) {
	dir := t.TempDir()

	t.Run("no file", func(t *testing.T) {
		status := CheckPostCommitHookStatus(filepath.Join(dir, "missing"))
		if status.Installed {
			t.Error("expected not installed for nonexistent file")
		}
	})

	t.Run("non-timbers hook", func(t *testing.T) {
		path := filepath.Join(dir, "other-post-commit")
		writeTestFile(t, path, "#!/bin/sh\necho done\n")
		status := CheckPostCommitHookStatus(path)
		if status.Installed {
			t.Error("expected not installed for non-timbers hook")
		}
	})

	t.Run("timbers hook", func(t *testing.T) {
		path := filepath.Join(dir, "timbers-post-commit")
		writeTestFile(t, path, "#!/bin/sh\ntimbers hook run post-commit\n")
		status := CheckPostCommitHookStatus(path)
		if !status.Installed {
			t.Error("expected installed")
		}
	})
}

func TestInstallPostCommitHook(t *testing.T) {
	t.Run("fresh install", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "post-commit")

		if err := InstallPostCommitHook(hookPath); err != nil {
			t.Fatalf("InstallPostCommitHook() error: %v", err)
		}

		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("failed to read hook: %v", err)
		}
		if !strings.HasPrefix(string(content), "#!/bin/sh") {
			t.Error("expected shebang")
		}
		if !strings.Contains(string(content), "timbers hook run post-commit") {
			t.Error("expected timbers hook command")
		}
	})

	t.Run("idempotent when already installed", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "post-commit")
		writeTestFile(t, hookPath, GeneratePostCommitHook())

		if err := InstallPostCommitHook(hookPath); err != nil {
			t.Fatalf("InstallPostCommitHook() error: %v", err)
		}

		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatal(err)
		}
		// Should not duplicate the timbers section
		count := strings.Count(string(content), "timbers hook run post-commit")
		if count != 1 {
			t.Errorf("expected 1 timbers section, got %d", count)
		}
	})

	t.Run("appends to existing non-timbers hook", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "post-commit")
		existing := "#!/bin/sh\necho 'existing hook'\n"
		writeTestFile(t, hookPath, existing)

		if err := InstallPostCommitHook(hookPath); err != nil {
			t.Fatalf("InstallPostCommitHook() error: %v", err)
		}

		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatal(err)
		}
		contentStr := string(content)
		if !strings.Contains(contentStr, "existing hook") {
			t.Error("existing hook content was lost")
		}
		if !strings.Contains(contentStr, "timbers hook run post-commit") {
			t.Error("timbers section not appended")
		}
	})

	t.Run("appends newline if missing", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "post-commit")
		writeTestFile(t, hookPath, "#!/bin/sh\necho done") // no trailing newline

		if err := InstallPostCommitHook(hookPath); err != nil {
			t.Fatalf("InstallPostCommitHook() error: %v", err)
		}

		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatal(err)
		}
		// Should have a newline before the timbers section
		if strings.Contains(string(content), "echo done\n# timbers") {
			// Good — newline was added
		} else if strings.Contains(string(content), "echo done# timbers") {
			t.Error("missing newline before timbers section")
		}
	})
}

func TestBackupExistingHook(t *testing.T) {
	dir := t.TempDir()
	hookPath := filepath.Join(dir, "pre-commit")
	backupPath := hookPath + ".backup"

	content := "#!/bin/sh\noriginal hook\n"
	writeTestFile(t, hookPath, content)

	if err := BackupExistingHook(hookPath); err != nil {
		t.Fatalf("BackupExistingHook() error: %v", err)
	}

	if HookExists(hookPath) {
		t.Error("original hook should be moved")
	}
	if !HookExists(backupPath) {
		t.Error("backup should exist")
	}

	backed, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(backed) != content {
		t.Errorf("backup content = %q, want %q", string(backed), content)
	}
}

// --- Phase 1: Environment Classification Tests ---

func TestClassifyHookEnvFrom(t *testing.T) {
	tests := []struct {
		name          string
		coreHooksPath string
		hooksDir      string
		hookExists    bool
		hookContent   string
		wantTier      HookEnvTier
		wantOwner     string
		wantHasHook   bool
		wantHasTimb   bool
	}{
		{
			name:        "tier 1: uncontested, no hook",
			hooksDir:    "/repo/.git/hooks",
			wantTier:    HookEnvUncontested,
			wantHasHook: false,
		},
		{
			name:        "tier 2a: existing hook with timbers section delimiters",
			hooksDir:    "/repo/.git/hooks",
			hookExists:  true,
			hookContent: "#!/bin/sh\n# --- timbers section (do not edit) ---\ntimbers hook run pre-commit\n# --- end timbers section ---\n",
			wantTier:    HookEnvExistingHook,
			wantHasHook: true,
			wantHasTimb: true,
		},
		{
			name:        "tier 2a: existing hook with old-format timbers",
			hooksDir:    "/repo/.git/hooks",
			hookExists:  true,
			hookContent: "#!/bin/sh\ntimbers hook run pre-commit \"$@\"\n",
			wantTier:    HookEnvExistingHook,
			wantHasHook: true,
			wantHasTimb: true,
		},
		{
			name:        "tier 2b: existing hook without timbers",
			hooksDir:    "/repo/.git/hooks",
			hookExists:  true,
			hookContent: "#!/bin/sh\necho hello\n",
			wantTier:    HookEnvExistingHook,
			wantHasHook: true,
			wantHasTimb: false,
		},
		{
			name:          "tier 3a: beads owner with timbers",
			coreHooksPath: ".beads/hooks",
			hooksDir:      "/repo/.beads/hooks",
			hookExists:    true,
			hookContent:   "#!/bin/sh\n# --- timbers section (do not edit) ---\ntimbers hook run pre-commit\n# --- end timbers section ---\n",
			wantTier:      HookEnvKnownOverride,
			wantOwner:     "beads",
			wantHasHook:   true,
			wantHasTimb:   true,
		},
		{
			name:          "tier 3b: beads owner without timbers",
			coreHooksPath: ".beads/hooks",
			hooksDir:      "/repo/.beads/hooks",
			hookExists:    true,
			hookContent:   "#!/bin/sh\nbeads stuff\n",
			wantTier:      HookEnvKnownOverride,
			wantOwner:     "beads",
			wantHasHook:   true,
			wantHasTimb:   false,
		},
		{
			name:          "tier 3: husky owner (.husky)",
			coreHooksPath: ".husky",
			hooksDir:      "/repo/.husky",
			hookExists:    true,
			hookContent:   "#!/bin/sh\nhusky stuff\n",
			wantTier:      HookEnvKnownOverride,
			wantOwner:     "husky",
			wantHasHook:   true,
			wantHasTimb:   false,
		},
		{
			name:          "tier 3: husky owner (.husky/_)",
			coreHooksPath: ".husky/_",
			hooksDir:      "/repo/.husky/_",
			hookExists:    false,
			wantTier:      HookEnvKnownOverride,
			wantOwner:     "husky",
			wantHasHook:   false,
		},
		{
			name:          "tier 4: unknown override",
			coreHooksPath: "/custom/hooks/path",
			hooksDir:      "/custom/hooks/path",
			hookExists:    false,
			wantTier:      HookEnvUnknownOverride,
			wantHasHook:   false,
		},
		{
			name:          "tier 4: unknown override with hook",
			coreHooksPath: "/opt/company/git-hooks",
			hooksDir:      "/opt/company/git-hooks",
			hookExists:    true,
			hookContent:   "#!/bin/sh\ncustom stuff\n",
			wantTier:      HookEnvUnknownOverride,
			wantHasHook:   true,
			wantHasTimb:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := classifyHookEnvFrom(tt.coreHooksPath, tt.hooksDir, tt.hookExists, tt.hookContent)

			if info.Tier != tt.wantTier {
				t.Errorf("Tier = %d, want %d", info.Tier, tt.wantTier)
			}
			if info.Owner != tt.wantOwner {
				t.Errorf("Owner = %q, want %q", info.Owner, tt.wantOwner)
			}
			if info.HasHook != tt.wantHasHook {
				t.Errorf("HasHook = %v, want %v", info.HasHook, tt.wantHasHook)
			}
			if info.HasTimbers != tt.wantHasTimb {
				t.Errorf("HasTimbers = %v, want %v", info.HasTimbers, tt.wantHasTimb)
			}
			if info.HooksDir != tt.hooksDir {
				t.Errorf("HooksDir = %q, want %q", info.HooksDir, tt.hooksDir)
			}
		})
	}
}

func TestMatchKnownOwner(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{".beads/hooks", "beads"},
		{"/repo/.beads/hooks", "beads"},
		{".husky", "husky"},
		{".husky/_", "husky"},
		{"/custom/path", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := matchKnownOwner(tt.path)
			if got != tt.want {
				t.Errorf("matchKnownOwner(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsAppendable(t *testing.T) {
	t.Run("regular text file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "hook")
		writeTestFile(t, path, "#!/bin/sh\necho hello\n")

		ok, reason := IsAppendable(path)
		if !ok {
			t.Errorf("expected appendable, got reason: %q", reason)
		}
		if reason != "" {
			t.Errorf("expected empty reason, got %q", reason)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target")
		writeTestFile(t, target, "#!/bin/sh\n")
		link := filepath.Join(dir, "link")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}

		ok, reason := IsAppendable(link)
		if ok {
			t.Error("expected not appendable for symlink")
		}
		if reason != "symlink" {
			t.Errorf("reason = %q, want %q", reason, "symlink")
		}
	})

	t.Run("binary file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "binary")
		// Write content with null bytes.
		data := []byte("ELF\x00\x00\x00binary content")
		// #nosec G306 -- test binary file needs execute permission for realistic testing
		if err := os.WriteFile(path, data, 0o755); err != nil { //nolint:gosec // test file
			t.Fatal(err)
		}

		ok, reason := IsAppendable(path)
		if ok {
			t.Error("expected not appendable for binary")
		}
		if reason != "binary" {
			t.Errorf("reason = %q, want %q", reason, "binary")
		}
	})

	t.Run("nonexistent", func(t *testing.T) {
		ok, reason := IsAppendable("/nonexistent/path/hook")
		if ok {
			t.Error("expected not appendable for nonexistent")
		}
		if reason != "not found" {
			t.Errorf("reason = %q, want %q", reason, "not found")
		}
	})
}

// --- Phase 2: Section Management Tests ---

func TestAppendTimbersSection(t *testing.T) {
	sectionContent := `if command -v timbers >/dev/null 2>&1; then
  timbers hook run pre-commit "$@"
  rc=$?
  if [ $rc -ne 0 ]; then exit $rc; fi
fi
`

	t.Run("creates file with shebang when nonexistent", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "pre-commit")

		if err := AppendTimbersSection(hookPath, sectionContent); err != nil {
			t.Fatalf("AppendTimbersSection() error: %v", err)
		}

		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatal(err)
		}
		hookContent := string(content)
		if !strings.HasPrefix(hookContent, "#!/bin/sh\n") {
			t.Error("expected shebang")
		}
		if !strings.Contains(hookContent, "# --- timbers section (do not edit) ---") {
			t.Error("expected section start delimiter")
		}
		if !strings.Contains(hookContent, "# --- end timbers section ---") {
			t.Error("expected section end delimiter")
		}
		if !strings.Contains(hookContent, "timbers hook run pre-commit") {
			t.Error("expected timbers hook command in section")
		}

		// Check file is executable.
		fi, err := os.Stat(hookPath)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode()&0o111 == 0 {
			t.Error("expected executable permission")
		}
	})

	t.Run("appends to existing script", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "pre-commit")
		writeTestFile(t, hookPath, "#!/bin/sh\necho 'existing logic'\n")

		if err := AppendTimbersSection(hookPath, sectionContent); err != nil {
			t.Fatalf("AppendTimbersSection() error: %v", err)
		}

		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatal(err)
		}
		hookContent := string(content)
		if !strings.Contains(hookContent, "existing logic") {
			t.Error("existing content was lost")
		}
		if !strings.Contains(hookContent, "# --- timbers section (do not edit) ---") {
			t.Error("expected section start delimiter")
		}
		if !strings.Contains(hookContent, "timbers hook run pre-commit") {
			t.Error("expected timbers hook command")
		}
	})

	t.Run("idempotent no duplicate section", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "pre-commit")
		writeTestFile(t, hookPath, "#!/bin/sh\necho hello\n")

		// Append twice.
		if err := AppendTimbersSection(hookPath, sectionContent); err != nil {
			t.Fatal(err)
		}
		if err := AppendTimbersSection(hookPath, sectionContent); err != nil {
			t.Fatal(err)
		}

		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatal(err)
		}
		count := strings.Count(string(content), "# --- timbers section (do not edit) ---")
		if count != 1 {
			t.Errorf("expected 1 section start delimiter, got %d", count)
		}
	})

	t.Run("handles missing trailing newline in existing file", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "pre-commit")
		writeTestFile(t, hookPath, "#!/bin/sh\necho done") // no trailing newline

		if err := AppendTimbersSection(hookPath, sectionContent); err != nil {
			t.Fatal(err)
		}

		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatal(err)
		}
		// The section start should be on its own line, not concatenated.
		if strings.Contains(string(content), "echo done#") {
			t.Error("section delimiter should be on its own line")
		}
	})
}

func TestRemoveTimbersSection(t *testing.T) {
	t.Run("removes section preserves other content", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "pre-commit")
		hookBody := "#!/bin/sh\necho 'before'\n" +
			"# --- timbers section (do not edit) ---\n" +
			"timbers hook run pre-commit\n" +
			"# --- end timbers section ---\n" +
			"echo 'after'\n"
		writeTestFile(t, hookPath, hookBody)

		if err := RemoveTimbersSection(hookPath); err != nil {
			t.Fatalf("RemoveTimbersSection() error: %v", err)
		}

		result, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatal(err)
		}
		remaining := string(result)
		if strings.Contains(remaining, "timbers section") {
			t.Error("section should be removed")
		}
		if !strings.Contains(remaining, "echo 'before'") {
			t.Error("content before section was lost")
		}
		if !strings.Contains(remaining, "echo 'after'") {
			t.Error("content after section was lost")
		}
	})

	t.Run("deletes file when only shebang remains", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "pre-commit")
		hookBody := "#!/bin/sh\n" +
			"# --- timbers section (do not edit) ---\n" +
			"timbers hook run pre-commit\n" +
			"# --- end timbers section ---\n"
		writeTestFile(t, hookPath, hookBody)

		if err := RemoveTimbersSection(hookPath); err != nil {
			t.Fatalf("RemoveTimbersSection() error: %v", err)
		}

		if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
			t.Error("expected file to be deleted when only shebang remains")
		}
	})

	t.Run("no-op when file does not exist", func(t *testing.T) {
		if err := RemoveTimbersSection("/nonexistent/path/hook"); err != nil {
			t.Errorf("expected nil error for nonexistent file, got: %v", err)
		}
	})

	t.Run("no-op when file has no section", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "pre-commit")
		original := "#!/bin/sh\necho hello\n"
		writeTestFile(t, hookPath, original)

		if err := RemoveTimbersSection(hookPath); err != nil {
			t.Fatalf("RemoveTimbersSection() error: %v", err)
		}

		result, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(result) != original {
			t.Errorf("file content changed: got %q, want %q", string(result), original)
		}
	})
}

func TestHasTimbersSection(t *testing.T) {
	t.Run("new format with delimiters", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "pre-commit")
		delimited := "#!/bin/sh\n" +
			"# --- timbers section (do not edit) ---\n" +
			"timbers hook run pre-commit\n" +
			"# --- end timbers section ---\n"
		writeTestFile(t, hookPath, delimited)

		if !HasTimbersSection(hookPath) {
			t.Error("expected true for new-format delimiters")
		}
	})

	t.Run("old format without delimiters", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "pre-commit")
		writeTestFile(t, hookPath, "#!/bin/sh\ntimbers hook run pre-commit \"$@\"\n")

		if !HasTimbersSection(hookPath) {
			t.Error("expected true for old-format timbers hook")
		}
	})

	t.Run("no timbers content", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "pre-commit")
		writeTestFile(t, hookPath, "#!/bin/sh\necho hello\n")

		if HasTimbersSection(hookPath) {
			t.Error("expected false for non-timbers hook")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		if HasTimbersSection("/nonexistent/path/hook") {
			t.Error("expected false for nonexistent file")
		}
	})
}
