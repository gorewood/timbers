package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePreCommitHook(t *testing.T) {
	t.Run("without chain", func(t *testing.T) {
		got := GeneratePreCommitHook(false)
		if !strings.HasPrefix(got, "#!/bin/sh") {
			t.Error("expected shebang")
		}
		if !strings.Contains(got, "timbers hook run pre-commit") {
			t.Error("expected timbers hook command")
		}
		if strings.Contains(got, ".backup") {
			t.Error("should not contain backup chain")
		}
	})

	t.Run("with chain", func(t *testing.T) {
		got := GeneratePreCommitHook(true)
		if !strings.Contains(got, "timbers hook run pre-commit") {
			t.Error("expected timbers hook command")
		}
		if !strings.Contains(got, "pre-commit.backup") {
			t.Error("expected backup chain section")
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
