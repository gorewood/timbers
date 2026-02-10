package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestFile creates a file in tests. Hook files need 0o755 for realistic testing.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	// #nosec G306 -- test hook files need execute permission
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveTimbersSectionFromContent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string // substring that must be absent
		wantHas string // substring that must be present
	}{
		{
			name:    "empty file",
			input:   "",
			want:    TimbersHookMarkerBegin,
			wantHas: "",
		},
		{
			name:    "no timbers section",
			input:   "#!/bin/bash\necho hello\n",
			wantHas: "echo hello",
		},
		{
			name: "removes timbers section preserves rest",
			input: "#!/bin/bash\necho before\n" +
				TimbersHookMarkerBegin + "\ntimbers prime\n" + TimbersHookMarkerEnd + "\n" +
				"echo after\n",
			want:    TimbersHookMarkerBegin,
			wantHas: "echo after",
		},
		{
			name: "removes only timbers section",
			input: "#!/bin/bash\n" +
				TimbersHookMarkerBegin + "\ntimbers prime\n" + TimbersHookMarkerEnd + "\n",
			want: "timbers prime",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveTimbersSectionFromContent(tt.input)
			if tt.want != "" && strings.Contains(got, tt.want) {
				t.Errorf("result should not contain %q, got:\n%s", tt.want, got)
			}
			if tt.wantHas != "" && !strings.Contains(got, tt.wantHas) {
				t.Errorf("result should contain %q, got:\n%s", tt.wantHas, got)
			}
		})
	}
}

func TestRemoveTimbersSectionFromContent_CollapsesBlanks(t *testing.T) {
	input := "#!/bin/bash\necho before\n\n\n" +
		TimbersHookMarkerBegin + "\ntimbers prime\n" + TimbersHookMarkerEnd + "\n\n\n" +
		"echo after\n"
	got := RemoveTimbersSectionFromContent(input)
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("should collapse triple newlines, got:\n%q", got)
	}
}

func TestIsTimbersSectionInstalled(t *testing.T) {
	dir := t.TempDir()

	t.Run("file does not exist", func(t *testing.T) {
		if IsTimbersSectionInstalled(filepath.Join(dir, "nonexistent")) {
			t.Error("expected false for nonexistent file")
		}
	})

	t.Run("file without timbers section", func(t *testing.T) {
		path := filepath.Join(dir, "no-timbers.sh")
		writeTestFile(t, path, "#!/bin/bash\necho hello\n")
		if IsTimbersSectionInstalled(path) {
			t.Error("expected false for file without timbers section")
		}
	})

	t.Run("file with timbers section", func(t *testing.T) {
		path := filepath.Join(dir, "has-timbers.sh")
		writeTestFile(t, path, "#!/bin/bash\n"+ClaudeHookContent+"\n")
		if !IsTimbersSectionInstalled(path) {
			t.Error("expected true for file with timbers section")
		}
	})
}

func TestInstallTimbersSection(t *testing.T) {
	t.Run("creates new file with shebang", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "hooks", "hook.sh")
		if err := InstallTimbersSection(path); err != nil {
			t.Fatalf("InstallTimbersSection() error: %v", err)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read: %v", err)
		}
		got := string(content)
		if !strings.HasPrefix(got, "#!/bin/bash") {
			t.Error("expected shebang prefix")
		}
		if !strings.Contains(got, TimbersHookMarkerBegin) {
			t.Error("expected timbers section")
		}
		if !strings.Contains(got, "timbers prime") {
			t.Error("expected timbers prime command")
		}
	})

	t.Run("appends to existing file preserving content", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "hook.sh")
		writeTestFile(t, path, "#!/bin/bash\necho existing\n")
		if err := InstallTimbersSection(path); err != nil {
			t.Fatalf("InstallTimbersSection() error: %v", err)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got := string(content)
		if !strings.Contains(got, "echo existing") {
			t.Error("expected existing content preserved")
		}
		if !strings.Contains(got, TimbersHookMarkerBegin) {
			t.Error("expected timbers section added")
		}
	})

	t.Run("replaces existing timbers section", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "hook.sh")
		writeTestFile(t, path, "#!/bin/bash\n"+TimbersHookMarkerBegin+"\nold content\n"+TimbersHookMarkerEnd+"\n")
		if err := InstallTimbersSection(path); err != nil {
			t.Fatalf("InstallTimbersSection() error: %v", err)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got := string(content)
		if strings.Contains(got, "old content") {
			t.Error("old section content should be replaced")
		}
		if strings.Count(got, TimbersHookMarkerBegin) != 1 {
			t.Error("expected exactly one timbers section")
		}
	})

	t.Run("file is executable", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "hook.sh")
		if err := InstallTimbersSection(path); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0o111 == 0 {
			t.Errorf("expected executable, got mode %o", info.Mode())
		}
	})
}

func TestRemoveTimbersSectionFromHook(t *testing.T) {
	t.Run("nonexistent file is no-op", func(t *testing.T) {
		dir := t.TempDir()
		if err := RemoveTimbersSectionFromHook(filepath.Join(dir, "nonexistent")); err != nil {
			t.Errorf("expected no error for nonexistent file, got: %v", err)
		}
	})

	t.Run("removes section from file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "hook.sh")
		writeTestFile(t, path, "#!/bin/bash\necho keep\n"+ClaudeHookContent+"\n")
		if err := RemoveTimbersSectionFromHook(path); err != nil {
			t.Fatalf("RemoveTimbersSectionFromHook() error: %v", err)
		}
		result, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got := string(result)
		if strings.Contains(got, TimbersHookMarkerBegin) {
			t.Error("timbers section should be removed")
		}
		if !strings.Contains(got, "echo keep") {
			t.Error("non-timbers content should be preserved")
		}
	})

	t.Run("only-timbers file becomes empty shell", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "hook.sh")
		writeTestFile(t, path, "#!/bin/bash\n"+ClaudeHookContent+"\n")
		if err := RemoveTimbersSectionFromHook(path); err != nil {
			t.Fatal(err)
		}
		result, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got := string(result)
		if !strings.HasPrefix(got, "#!/bin/bash") {
			t.Error("should preserve shebang")
		}
		cleaned := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(got), "#!/bin/bash"))
		if cleaned != "" {
			t.Errorf("expected empty shell, got: %q", cleaned)
		}
	})
}

func TestResolveClaudeHookPath(t *testing.T) {
	t.Run("global path structure", func(t *testing.T) {
		path, scope, err := ResolveClaudeHookPath(false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scope != "global" {
			t.Errorf("scope = %q, want %q", scope, "global")
		}
		if !strings.HasSuffix(path, filepath.Join(".claude", "hooks", "user_prompt_submit.sh")) {
			t.Errorf("path should end with .claude/hooks/user_prompt_submit.sh, got: %s", path)
		}
	})

	t.Run("project path structure", func(t *testing.T) {
		path, scope, err := ResolveClaudeHookPath(true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scope != "project" {
			t.Errorf("scope = %q, want %q", scope, "project")
		}
		if !strings.HasSuffix(path, filepath.Join(".claude", "hooks", "user_prompt_submit.sh")) {
			t.Errorf("path should end with .claude/hooks/user_prompt_submit.sh, got: %s", path)
		}
	})
}
