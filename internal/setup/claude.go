package setup

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gorewood/timbers/internal/output"
)

// hookEntry represents a single hook action.
type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// hookGroup represents a hook event group with matcher and hooks.
type hookGroup struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

// timbersHookConfig describes a single hook event to install.
type timbersHookConfig struct {
	Event   string
	Matcher string
	Command string
}

// timbersHookCommand is the resilient hook command that degrades gracefully
// when timbers is not installed: prints a helpful message instead of erroring.
//
//nolint:lll // shell one-liner, splitting would reduce readability
const timbersHookCommand = `command -v timbers >/dev/null 2>&1 && timbers prime || echo "timbers: not installed (https://github.com/gorewood/timbers)"`

// legacyHookCommand is the old non-resilient format, kept for backward-compat detection and removal.
const legacyHookCommand = "timbers prime"

// stopCommand checks for undocumented commits at session end.
//
//nolint:lll // shell one-liner
const stopCommand = `command -v timbers >/dev/null 2>&1 && timbers pending --json 2>/dev/null | grep -q '"count":[1-9][0-9]*' && echo "timbers: undocumented commits - run 'timbers pending' to review" || true`

// legacyPostToolUseBashCommand is the old format that used $TOOL_INPUT (always empty).
// Claude Code hooks receive input via stdin, not env vars. Kept for upgrade detection.
//
//nolint:lll // shell one-liner
const legacyPostToolUseBashCommand = `printf '%s\n' "$TOOL_INPUT" | grep -q 'git commit' && command -v timbers >/dev/null 2>&1 && echo "timbers: remember to run 'timbers log' to document this commit" || true`

// postToolUseBashCommand nudges after git commits to document work.
// Reads tool input from stdin (Claude Code hook protocol) — hooks receive
// JSON on stdin, not via environment variables.
//
//nolint:lll // shell one-liner
const postToolUseBashCommand = `grep -q 'git commit' && command -v timbers >/dev/null 2>&1 && echo "timbers: remember to run 'timbers log' to document this commit" || true`

// timbersHooks defines all hook events timbers installs into Claude Code settings.
var timbersHooks = []timbersHookConfig{
	{Event: "SessionStart", Matcher: "", Command: timbersHookCommand},
	{Event: "PreCompact", Matcher: "", Command: timbersHookCommand},
	{Event: "Stop", Matcher: "", Command: stopCommand},
	{Event: "PostToolUse", Matcher: "Bash", Command: postToolUseBashCommand},
}

// ResolveClaudeSettingsPath determines the settings file path based on scope.
// If project is true, returns the project-local settings path; otherwise the global path.
func ResolveClaudeSettingsPath(project bool) (string, string, error) {
	if project {
		cwd, err := os.Getwd()
		if err != nil {
			return "", "", output.NewSystemErrorWithCause("failed to get working directory", err)
		}
		return filepath.Join(cwd, ".claude", "settings.local.json"), "project", nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", output.NewSystemErrorWithCause("failed to get home directory", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), "global", nil
}

// IsTimbersSectionInstalled checks if any timbers hooks are configured in a Claude settings file.
func IsTimbersSectionInstalled(settingsPath string) bool {
	settings, err := readSettings(settingsPath)
	if err != nil {
		return false
	}
	return hasTimbersHooks(settings)
}

// InstallTimbersSection adds timbers hooks to a Claude settings file.
// On upgrade, adds missing event hooks without duplicating existing ones.
func InstallTimbersSection(settingsPath string) error {
	settingsDir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		return output.NewSystemErrorWithCause("failed to create settings directory", err)
	}

	settings, err := readSettings(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return output.NewSystemErrorWithCause("failed to read settings file", err)
	}
	if settings == nil {
		settings = make(map[string]any)
	}

	addTimbersHooks(settings)

	return writeSettings(settingsPath, settings)
}

// RemoveTimbersSectionFromHook removes all timbers hooks from a Claude settings file.
func RemoveTimbersSectionFromHook(settingsPath string) error {
	settings, err := readSettings(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return output.NewSystemErrorWithCause("failed to read settings file", err)
	}

	removeTimbersHooks(settings)

	return writeSettings(settingsPath, settings)
}

// readSettings reads and parses a JSON settings file.
// Returns the raw os error on read failure so callers can check os.IsNotExist.
func readSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err //nolint:wrapcheck // callers need os.IsNotExist to work
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, output.NewSystemErrorWithCause("failed to parse settings file", err)
	}
	return settings, nil
}

// writeSettings writes a settings map as formatted JSON.
func writeSettings(path string, settings map[string]any) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return output.NewSystemErrorWithCause("failed to marshal settings", err)
	}
	data = append(data, '\n')

	// #nosec G306 -- settings files are not secrets; 0o644 matches Claude Code's own convention
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return output.NewSystemErrorWithCause("failed to write settings file", err)
	}
	return nil
}

// isTimbersCommand checks if a command string is any timbers hook command.
func isTimbersCommand(cmd string) bool {
	for _, cfg := range timbersHooks {
		if cmd == cfg.Command {
			return true
		}
	}
	return cmd == legacyHookCommand || cmd == legacyPostToolUseBashCommand
}

// hasExactHookCommand checks if a specific command exists in an event's hooks.
func hasExactHookCommand(settings map[string]any, event, command string) bool {
	groups := getEventGroups(settings, event)
	for _, group := range groups {
		for _, hook := range group.Hooks {
			if hook.Command == command {
				return true
			}
		}
	}
	return false
}

// hasTimbersHooks checks if any timbers hooks are installed across all events.
func hasTimbersHooks(settings map[string]any) bool {
	for _, cfg := range timbersHooks {
		if hasHookForEvent(settings, cfg.Event) {
			return true
		}
	}
	return false
}

// hasHookForEvent checks if a timbers hook exists for a specific event.
func hasHookForEvent(settings map[string]any, event string) bool {
	groups := getEventGroups(settings, event)
	for _, group := range groups {
		for _, hook := range group.Hooks {
			if isTimbersCommand(hook.Command) {
				return true
			}
		}
	}
	return false
}

// addTimbersHooks adds all timbers hooks, upgrading stale hooks to current versions.
func addTimbersHooks(settings map[string]any) {
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = make(map[string]any)
		settings["hooks"] = hooks
	}

	for _, cfg := range timbersHooks {
		if hasExactHookCommand(settings, cfg.Event, cfg.Command) {
			continue // Already up to date
		}

		// Remove stale timbers hooks for this event before adding current version
		removeTimbersHooksFromEvent(hooks, cfg.Event)

		newGroup := map[string]any{
			"matcher": cfg.Matcher,
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": cfg.Command,
				},
			},
		}

		existing, _ := hooks[cfg.Event].([]any)
		hooks[cfg.Event] = append(existing, newGroup)
	}
}

// removeTimbersHooksFromEvent removes timbers hooks from a single event,
// preserving non-timbers hooks.
func removeTimbersHooksFromEvent(hooks map[string]any, event string) {
	groups, ok := hooks[event].([]any)
	if !ok {
		return
	}
	filtered := filterGroups(groups)
	if len(filtered) > 0 {
		hooks[event] = filtered
	} else {
		delete(hooks, event)
	}
}

// removeTimbersHooks removes all timbers hooks from all events.
func removeTimbersHooks(settings map[string]any) {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return
	}

	for _, cfg := range timbersHooks {
		removeTimbersHooksFromEvent(hooks, cfg.Event)
	}

	if len(hooks) == 0 {
		delete(settings, "hooks")
	}
}

// filterGroups removes timbers hooks from a list of hook groups,
// dropping groups that become empty.
func filterGroups(groups []any) []any {
	var filtered []any
	for _, rawGroup := range groups {
		group, ok := rawGroup.(map[string]any)
		if !ok {
			filtered = append(filtered, rawGroup)
			continue
		}

		rawHooks, _ := group["hooks"].([]any)
		filteredHooks := filterHooks(rawHooks)

		if len(filteredHooks) > 0 {
			group["hooks"] = filteredHooks
			filtered = append(filtered, group)
		}
	}
	return filtered
}

// filterHooks removes timbers entries from a list of hook entries.
func filterHooks(rawHooks []any) []any {
	var filtered []any
	for _, rawHook := range rawHooks {
		hook, ok := rawHook.(map[string]any)
		if !ok {
			filtered = append(filtered, rawHook)
			continue
		}
		if cmd, _ := hook["command"].(string); isTimbersCommand(cmd) {
			continue
		}
		filtered = append(filtered, rawHook)
	}
	return filtered
}
