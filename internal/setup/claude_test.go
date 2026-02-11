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

	t.Run("detects stop hook as installed", func(t *testing.T) {
		path := filepath.Join(dir, "has-stop.json")
		writeJSON(t, path, map[string]any{
			"hooks": map[string]any{
				"Stop": []any{
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": stopCommand},
						},
					},
				},
			},
		})
		if !IsTimbersSectionInstalled(path) {
			t.Error("expected true for file with stop hook")
		}
	})
}

func TestInstallTimbersSection(t *testing.T) {
	t.Run("creates new file with all hooks", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".claude", "settings.local.json")
		if err := InstallTimbersSection(path); err != nil {
			t.Fatalf("InstallTimbersSection() error: %v", err)
		}
		settings := readJSON(t, path)
		assertAllTimbersHooksPresent(t, settings)
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
		assertAllTimbersHooksPresent(t, settings)
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
		assertAllTimbersHooksPresent(t, settings)
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
		// Count timbers hooks per event — should be exactly 1 each
		for _, cfg := range timbersHooks {
			count := countTimbersHooksForEvent(settings, cfg.Event)
			if count != 1 {
				t.Errorf("event %s: expected 1 timbers hook, got %d", cfg.Event, count)
			}
		}
	})

	t.Run("detects legacy format as already installed", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		// Write legacy format for SessionStart only
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
		// Install should skip SessionStart (legacy detected) and add the 3 new events
		if err := InstallTimbersSection(path); err != nil {
			t.Fatal(err)
		}
		settings := readJSON(t, path)
		// SessionStart should have exactly 1 hook (the legacy one, not duplicated)
		count := countTimbersHooksForEvent(settings, "SessionStart")
		if count != 1 {
			t.Errorf("SessionStart: expected exactly 1 timbers hook, got %d", count)
		}
		// Other events should be added
		for _, event := range []string{"PreCompact", "Stop", "PostToolUse"} {
			if !hasHookForEvent(settings, event) {
				t.Errorf("event %s should be installed during upgrade", event)
			}
		}
	})
}

func TestAddTimbersHooks_AllEvents(t *testing.T) {
	settings := make(map[string]any)
	addTimbersHooks(settings)
	assertAllTimbersHooksPresent(t, settings)
}

func TestAddTimbersHooks_Idempotent(t *testing.T) {
	settings := make(map[string]any)
	addTimbersHooks(settings)
	addTimbersHooks(settings)
	for _, cfg := range timbersHooks {
		count := countTimbersHooksForEvent(settings, cfg.Event)
		if count != 1 {
			t.Errorf("event %s: expected 1 timbers hook after double add, got %d", cfg.Event, count)
		}
	}
}

func TestAddTimbersHooks_UpgradeFromSingle(t *testing.T) {
	settings := map[string]any{
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
	}
	addTimbersHooks(settings)
	// SessionStart preserved (not duplicated)
	if count := countTimbersHooksForEvent(settings, "SessionStart"); count != 1 {
		t.Errorf("SessionStart: expected 1, got %d", count)
	}
	// 3 new events added
	for _, event := range []string{"PreCompact", "Stop", "PostToolUse"} {
		if !hasHookForEvent(settings, event) {
			t.Errorf("event %s should be added during upgrade", event)
		}
	}
}

func TestRemoveTimbersHooks_AllEvents(t *testing.T) {
	// Install all hooks plus some non-timbers hooks
	settings := map[string]any{
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
	}
	addTimbersHooks(settings)
	removeTimbersHooks(settings)

	// All timbers hooks should be gone
	for _, cfg := range timbersHooks {
		if hasHookForEvent(settings, cfg.Event) {
			t.Errorf("event %s should be removed", cfg.Event)
		}
	}
	// bd prime should remain
	groups := getSessionStartGroups(settings)
	if len(groups) != 1 || groups[0].Hooks[0].Command != "bd prime" {
		t.Error("non-timbers hooks should be preserved")
	}
}

func TestRemoveTimbersSectionFromHook(t *testing.T) {
	t.Run("nonexistent file is no-op", func(t *testing.T) {
		dir := t.TempDir()
		if err := RemoveTimbersSectionFromHook(filepath.Join(dir, "nonexistent.json")); err != nil {
			t.Errorf("expected no error for nonexistent file, got: %v", err)
		}
	})

	t.Run("removes all timbers hooks preserving others", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		// Create settings with timbers (all events) and other hooks
		settings := map[string]any{
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
		}
		addTimbersHooks(settings)
		writeJSON(t, path, settings)

		if err := RemoveTimbersSectionFromHook(path); err != nil {
			t.Fatalf("RemoveTimbersSectionFromHook() error: %v", err)
		}
		result := readJSON(t, path)
		for _, cfg := range timbersHooks {
			if hasHookForEvent(result, cfg.Event) {
				t.Errorf("event %s: timbers hook should be removed", cfg.Event)
			}
		}
		// bd prime should still be there
		groups := getSessionStartGroups(result)
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
		result := readJSON(t, path)
		if hasTimbersHooks(result) {
			t.Error("legacy timbers prime should be removed")
		}
	})

	t.Run("cleans up empty hooks section", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		settings := map[string]any{
			"permissions": map[string]any{"allow": []any{}},
		}
		addTimbersHooks(settings)
		writeJSON(t, path, settings)

		if err := RemoveTimbersSectionFromHook(path); err != nil {
			t.Fatal(err)
		}
		result := readJSON(t, path)
		if _, ok := result["hooks"]; ok {
			t.Error("empty hooks section should be removed")
		}
		if _, ok := result["permissions"]; !ok {
			t.Error("other settings should be preserved")
		}
	})
}

func TestIsTimbersCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{"resilient prime command", timbersHookCommand, true},
		{"legacy prime command", legacyHookCommand, true},
		{"stop command", stopCommand, true},
		{"post-tool-use command", postToolUseBashCommand, true},
		{"unrelated command", "bd prime", false},
		{"empty string", "", false},
		{"partial match", "timbers", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTimbersCommand(tc.cmd); got != tc.want {
				t.Errorf("isTimbersCommand(%q) = %v, want %v", tc.cmd, got, tc.want)
			}
		})
	}
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

func TestPostToolUseHasBashMatcher(t *testing.T) {
	settings := make(map[string]any)
	addTimbersHooks(settings)

	hooks, _ := settings["hooks"].(map[string]any)
	groups, ok := hooks["PostToolUse"].([]any)
	if !ok || len(groups) == 0 {
		t.Fatal("PostToolUse hook group should exist")
	}
	group, _ := groups[0].(map[string]any)
	matcher, _ := group["matcher"].(string)
	if matcher != "Bash" {
		t.Errorf("PostToolUse matcher = %q, want %q", matcher, "Bash")
	}
}

// --- test helpers ---

// assertAllTimbersHooksPresent checks that all 4 hook events are installed.
func assertAllTimbersHooksPresent(t *testing.T, settings map[string]any) {
	t.Helper()
	for _, cfg := range timbersHooks {
		if !hasHookForEvent(settings, cfg.Event) {
			t.Errorf("event %s: timbers hook should be present", cfg.Event)
		}
	}
}

// countTimbersHooksForEvent counts timbers hook entries in a specific event.
func countTimbersHooksForEvent(settings map[string]any, event string) int {
	count := 0
	groups := getEventGroups(settings, event)
	for _, g := range groups {
		for _, h := range g.Hooks {
			if isTimbersCommand(h.Command) {
				count++
			}
		}
	}
	return count
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
