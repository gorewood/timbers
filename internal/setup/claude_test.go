package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestIsTimbersSectionInstalled(t *testing.T) {
	dir := t.TempDir()

	t.Run("file does not exist", func(t *testing.T) {
		if IsTimbersSectionInstalled(filepath.Join(dir, "nonexistent.json")) {
			t.Error("expected false for nonexistent file")
		}
	})

	t.Run("file without timbers hook", func(t *testing.T) {
		path := filepath.Join(dir, "no-timbers.json")
		writeJSON(t, path, map[string]any{
			"permissions": map[string]any{"allow": []any{}},
		})
		if IsTimbersSectionInstalled(path) {
			t.Error("expected false for file without timbers hook")
		}
	})

	t.Run("file with timbers hook", func(t *testing.T) {
		path := filepath.Join(dir, "has-timbers.json")
		writeJSON(t, path, map[string]any{
			"hooks": map[string]any{
				"SessionStart": []any{
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": timbersHookCommand},
						},
					},
				},
			},
		})
		if !IsTimbersSectionInstalled(path) {
			t.Error("expected true for file with timbers hook")
		}
	})

	t.Run("detects legacy hook format", func(t *testing.T) {
		path := filepath.Join(dir, "has-legacy.json")
		writeJSON(t, path, map[string]any{
			"hooks": map[string]any{
				"SessionStart": []any{
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": "timbers prime"},
						},
					},
				},
			},
		})
		if !IsTimbersSectionInstalled(path) {
			t.Error("expected true for file with legacy timbers hook")
		}
	})

	t.Run("file with other hooks but not timbers", func(t *testing.T) {
		path := filepath.Join(dir, "other-hooks.json")
		writeJSON(t, path, map[string]any{
			"hooks": map[string]any{
				"SessionStart": []any{
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": "bd prime"},
						},
					},
				},
			},
		})
		if IsTimbersSectionInstalled(path) {
			t.Error("expected false for file with only other hooks")
		}
	})
}

func TestInstallTimbersSection(t *testing.T) {
	t.Run("creates new file with hook", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".claude", "settings.local.json")
		if err := InstallTimbersSection(path); err != nil {
			t.Fatalf("InstallTimbersSection() error: %v", err)
		}
		settings := readJSON(t, path)
		if !hasTimbersPrime(settings) {
			t.Error("expected timbers prime hook to be present")
		}
	})

	t.Run("preserves existing settings", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		writeJSON(t, path, map[string]any{
			"permissions": map[string]any{"allow": []any{"Bash(ls:*)"}},
		})
		if err := InstallTimbersSection(path); err != nil {
			t.Fatalf("InstallTimbersSection() error: %v", err)
		}
		settings := readJSON(t, path)
		if !hasTimbersPrime(settings) {
			t.Error("expected timbers prime hook")
		}
		perms, ok := settings["permissions"].(map[string]any)
		if !ok {
			t.Fatal("permissions should be preserved")
		}
		allow, ok := perms["allow"].([]any)
		if !ok || len(allow) == 0 {
			t.Error("existing permissions should be preserved")
		}
	})

	t.Run("preserves existing hooks", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		writeJSON(t, path, map[string]any{
			"hooks": map[string]any{
				"SessionStart": []any{
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": "bd prime"},
						},
					},
				},
			},
		})
		if err := InstallTimbersSection(path); err != nil {
			t.Fatalf("InstallTimbersSection() error: %v", err)
		}
		settings := readJSON(t, path)
		if !hasTimbersPrime(settings) {
			t.Error("expected timbers prime hook")
		}
		// Verify bd prime is still there
		groups := getSessionStartGroups(settings)
		foundBd := false
		for _, g := range groups {
			for _, h := range g.Hooks {
				if h.Command == "bd prime" {
					foundBd = true
				}
			}
		}
		if !foundBd {
			t.Error("existing bd prime hook should be preserved")
		}
	})

	t.Run("idempotent - does not duplicate", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		if err := InstallTimbersSection(path); err != nil {
			t.Fatal(err)
		}
		if err := InstallTimbersSection(path); err != nil {
			t.Fatal(err)
		}
		settings := readJSON(t, path)
		count := 0
		groups := getSessionStartGroups(settings)
		for _, g := range groups {
			for _, h := range g.Hooks {
				if isTimbersPrimeCommand(h.Command) {
					count++
				}
			}
		}
		if count != 1 {
			t.Errorf("expected exactly 1 timbers prime hook, got %d", count)
		}
	})

	t.Run("detects legacy format as already installed", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		// Write legacy format
		writeJSON(t, path, map[string]any{
			"hooks": map[string]any{
				"SessionStart": []any{
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": "timbers prime"},
						},
					},
				},
			},
		})
		// Install should be a no-op since legacy is detected
		if err := InstallTimbersSection(path); err != nil {
			t.Fatal(err)
		}
		settings := readJSON(t, path)
		count := 0
		groups := getSessionStartGroups(settings)
		for _, g := range groups {
			for _, h := range g.Hooks {
				if isTimbersPrimeCommand(h.Command) {
					count++
				}
			}
		}
		if count != 1 {
			t.Errorf("expected exactly 1 timbers hook, got %d", count)
		}
	})
}

func TestRemoveTimbersSectionFromHook(t *testing.T) {
	t.Run("nonexistent file is no-op", func(t *testing.T) {
		dir := t.TempDir()
		if err := RemoveTimbersSectionFromHook(filepath.Join(dir, "nonexistent.json")); err != nil {
			t.Errorf("expected no error for nonexistent file, got: %v", err)
		}
	})

	t.Run("removes timbers hook preserving others", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		writeJSON(t, path, map[string]any{
			"hooks": map[string]any{
				"SessionStart": []any{
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": "bd prime"},
						},
					},
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": "timbers prime"},
						},
					},
				},
			},
		})
		if err := RemoveTimbersSectionFromHook(path); err != nil {
			t.Fatalf("RemoveTimbersSectionFromHook() error: %v", err)
		}
		settings := readJSON(t, path)
		if hasTimbersPrime(settings) {
			t.Error("timbers prime should be removed")
		}
		// bd prime should still be there
		groups := getSessionStartGroups(settings)
		if len(groups) != 1 {
			t.Fatalf("expected 1 remaining group, got %d", len(groups))
		}
		if groups[0].Hooks[0].Command != "bd prime" {
			t.Error("bd prime should be preserved")
		}
	})

	t.Run("removes legacy hook format", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		writeJSON(t, path, map[string]any{
			"hooks": map[string]any{
				"SessionStart": []any{
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": "timbers prime"},
						},
					},
				},
			},
		})
		if err := RemoveTimbersSectionFromHook(path); err != nil {
			t.Fatal(err)
		}
		settings := readJSON(t, path)
		if hasTimbersPrime(settings) {
			t.Error("legacy timbers prime should be removed")
		}
	})

	t.Run("cleans up empty hooks section", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		writeJSON(t, path, map[string]any{
			"permissions": map[string]any{"allow": []any{}},
			"hooks": map[string]any{
				"SessionStart": []any{
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": "timbers prime"},
						},
					},
				},
			},
		})
		if err := RemoveTimbersSectionFromHook(path); err != nil {
			t.Fatal(err)
		}
		settings := readJSON(t, path)
		if _, ok := settings["hooks"]; ok {
			t.Error("empty hooks section should be removed")
		}
		if _, ok := settings["permissions"]; !ok {
			t.Error("other settings should be preserved")
		}
	})
}

func TestResolveClaudeSettingsPath(t *testing.T) {
	t.Run("global path structure", func(t *testing.T) {
		path, scope, err := ResolveClaudeSettingsPath(false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scope != "global" {
			t.Errorf("scope = %q, want %q", scope, "global")
		}
		if !filepath.IsAbs(path) {
			t.Errorf("path should be absolute, got: %s", path)
		}
		if filepath.Base(path) != "settings.json" {
			t.Errorf("global path should end with settings.json, got: %s", path)
		}
	})

	t.Run("project path structure", func(t *testing.T) {
		path, scope, err := ResolveClaudeSettingsPath(true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scope != "project" {
			t.Errorf("scope = %q, want %q", scope, "project")
		}
		if filepath.Base(path) != "settings.local.json" {
			t.Errorf("project path should end with settings.local.json, got: %s", path)
		}
	})
}

// writeTestFile creates a file in tests. Hook files need 0o755 for realistic testing.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	// #nosec G306 -- test hook files need execute permission
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

// writeJSON is a test helper that writes a map as formatted JSON.
func writeJSON(t *testing.T, path string, data map[string]any) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	// #nosec G306 -- test settings files, not secrets
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// readJSON is a test helper that reads a JSON file into a map.
func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}
