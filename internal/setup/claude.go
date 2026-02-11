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

// timbersHookCommand is the resilient hook command that degrades gracefully
// when timbers is not installed: prints a helpful message instead of erroring.
//
//nolint:lll // shell one-liner, splitting would reduce readability
const timbersHookCommand = `command -v timbers >/dev/null 2>&1 && timbers prime || echo "timbers: not installed (https://github.com/gorewood/timbers)"`

// legacyHookCommand is the old non-resilient format, kept for backward-compat detection and removal.
const legacyHookCommand = "timbers prime"

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

// IsTimbersSectionInstalled checks if timbers prime is configured in a Claude settings file.
func IsTimbersSectionInstalled(settingsPath string) bool {
	settings, err := readSettings(settingsPath)
	if err != nil {
		return false
	}
	return hasTimbersPrime(settings)
}

// InstallTimbersSection adds timbers prime to the SessionStart hooks in a Claude settings file.
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

	addTimbersPrime(settings)

	return writeSettings(settingsPath, settings)
}

// RemoveTimbersSectionFromHook removes timbers prime from a Claude settings file.
func RemoveTimbersSectionFromHook(settingsPath string) error {
	settings, err := readSettings(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return output.NewSystemErrorWithCause("failed to read settings file", err)
	}

	removeTimbersPrime(settings)

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

// isTimbersPrimeCommand checks if a command string is a timbers prime hook
// (either the current resilient format or the legacy bare command).
func isTimbersPrimeCommand(cmd string) bool {
	return cmd == timbersHookCommand || cmd == legacyHookCommand
}

// hasTimbersPrime checks if the SessionStart hooks contain timbers prime.
func hasTimbersPrime(settings map[string]any) bool {
	groups := getSessionStartGroups(settings)
	for _, group := range groups {
		for _, hook := range group.Hooks {
			if isTimbersPrimeCommand(hook.Command) {
				return true
			}
		}
	}
	return false
}

// addTimbersPrime adds timbers prime to SessionStart hooks if not already present.
func addTimbersPrime(settings map[string]any) {
	if hasTimbersPrime(settings) {
		return
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = make(map[string]any)
		settings["hooks"] = hooks
	}

	newGroup := map[string]any{
		"matcher": "",
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": timbersHookCommand,
			},
		},
	}

	existing, _ := hooks["SessionStart"].([]any)
	hooks["SessionStart"] = append(existing, newGroup)
}

// removeTimbersPrime removes timbers prime from SessionStart hooks.
func removeTimbersPrime(settings map[string]any) {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return
	}

	groups, ok := hooks["SessionStart"].([]any)
	if !ok {
		return
	}

	filtered := filterGroups(groups)

	if len(filtered) > 0 {
		hooks["SessionStart"] = filtered
	} else {
		delete(hooks, "SessionStart")
	}

	if len(hooks) == 0 {
		delete(settings, "hooks")
	}
}

// filterGroups removes timbers prime hooks from a list of hook groups,
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

// filterHooks removes timbers prime entries from a list of hook entries.
func filterHooks(rawHooks []any) []any {
	var filtered []any
	for _, rawHook := range rawHooks {
		hook, ok := rawHook.(map[string]any)
		if !ok {
			filtered = append(filtered, rawHook)
			continue
		}
		if cmd, _ := hook["command"].(string); isTimbersPrimeCommand(cmd) {
			continue
		}
		filtered = append(filtered, rawHook)
	}
	return filtered
}

// getSessionStartGroups parses SessionStart hook groups from settings.
func getSessionStartGroups(settings map[string]any) []hookGroup {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return nil
	}

	groups, ok := hooks["SessionStart"].([]any)
	if !ok {
		return nil
	}

	var result []hookGroup
	for _, rawGroup := range groups {
		if parsed, ok := parseHookGroup(rawGroup); ok {
			result = append(result, parsed)
		}
	}
	return result
}

// parseHookGroup converts a raw JSON group into a typed hookGroup.
func parseHookGroup(rawGroup any) (hookGroup, bool) {
	group, ok := rawGroup.(map[string]any)
	if !ok {
		return hookGroup{}, false
	}

	parsed := hookGroup{}
	if matcher, ok := group["matcher"].(string); ok {
		parsed.Matcher = matcher
	}

	rawHooks, _ := group["hooks"].([]any)
	for _, rawHook := range rawHooks {
		if entry, ok := parseHookEntry(rawHook); ok {
			parsed.Hooks = append(parsed.Hooks, entry)
		}
	}
	return parsed, true
}

// parseHookEntry converts a raw JSON hook into a typed hookEntry.
func parseHookEntry(rawHook any) (hookEntry, bool) {
	hook, ok := rawHook.(map[string]any)
	if !ok {
		return hookEntry{}, false
	}
	entry := hookEntry{}
	if hookType, ok := hook["type"].(string); ok {
		entry.Type = hookType
	}
	if command, ok := hook["command"].(string); ok {
		entry.Command = command
	}
	return entry, true
}
