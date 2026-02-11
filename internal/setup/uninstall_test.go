package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveGitHook(t *testing.T) {
	t.Run("removes hook without backup", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "pre-commit")
		writeTestFile(t, hookPath, "#!/bin/sh\n")

		removed, restored, err := RemoveGitHook(hookPath, false, "")
		if err != nil {
			t.Fatalf("RemoveGitHook() error: %v", err)
		}
		if !removed {
			t.Error("expected removed=true")
		}
		if restored {
			t.Error("expected restored=false")
		}
		if HookExists(hookPath) {
			t.Error("hook file should be removed")
		}
	})

	t.Run("removes hook and restores backup", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "pre-commit")
		backupPath := hookPath + ".backup"

		writeTestFile(t, hookPath, "#!/bin/sh\ntimbers\n")
		writeTestFile(t, backupPath, "#!/bin/sh\noriginal\n")

		removed, restored, err := RemoveGitHook(hookPath, true, backupPath)
		if err != nil {
			t.Fatalf("RemoveGitHook() error: %v", err)
		}
		if !removed {
			t.Error("expected removed=true")
		}
		if !restored {
			t.Error("expected restored=true")
		}

		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "#!/bin/sh\noriginal\n" {
			t.Errorf("expected restored content, got: %q", string(content))
		}
		if HookExists(backupPath) {
			t.Error("backup should be renamed away")
		}
	})

	t.Run("nonexistent hook without backup", func(t *testing.T) {
		dir := t.TempDir()
		hookPath := filepath.Join(dir, "nonexistent")

		removed, restored, err := RemoveGitHook(hookPath, false, "")
		if err != nil {
			t.Fatalf("RemoveGitHook() error: %v", err)
		}
		if !removed {
			t.Error("expected removed=true (no-op remove)")
		}
		if restored {
			t.Error("expected restored=false")
		}
	})
}

func TestRemoveBinary(t *testing.T) {
	t.Run("removes existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "timbers")
		writeTestFile(t, path, "binary")
		if err := RemoveBinary(path); err != nil {
			t.Fatalf("RemoveBinary() error: %v", err)
		}
		if HookExists(path) {
			t.Error("binary should be removed")
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "nonexistent")
		if err := RemoveBinary(path); err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestRemoveTimbersDirContents_Recursive(t *testing.T) {
	dir := t.TempDir()

	// Create nested date directory structure
	subdir := filepath.Join(dir, "2026", "01", "15")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(subdir, "entry1.json"), "{}")
	writeTestFile(t, filepath.Join(subdir, "entry2.json"), "{}")

	// Also a file at root level
	writeTestFile(t, filepath.Join(dir, "root.json"), "{}")

	if err := RemoveTimbersDirContents(dir); err != nil {
		t.Fatalf("RemoveTimbersDirContents() error: %v", err)
	}

	// All JSON files should be removed
	var jsonFiles []string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(d.Name()) == ".json" {
			jsonFiles = append(jsonFiles, path)
		}
		return nil
	})
	if len(jsonFiles) > 0 {
		t.Errorf("expected all JSON files removed, found: %v", jsonFiles)
	}

	// Empty date directories should be cleaned up
	if _, err := os.Stat(subdir); !os.IsNotExist(err) {
		t.Error("expected empty subdirectories to be removed")
	}
}

func TestGatherBinaryPath(t *testing.T) {
	path, err := GatherBinaryPath()
	if err != nil {
		t.Fatalf("GatherBinaryPath() error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}
