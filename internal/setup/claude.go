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

// legacyPreToolUseCommand was the first structured-JSON approach for blocking git commit.
// Replaced by git pre-commit hook blocking, which works for all git clients.
//
//nolint:lll // shell one-liner
const legacyPreToolUseCommand = `command -v timbers >/dev/null 2>&1 && timbers hook run claude-pre-tool-use || true`

// stopHookCommand blocks session end when pending commits exist.
// Uses structured JSON responses that Claude Code actually enforces.
//
//nolint:lll // shell one-liner
const stopHookCommand = `command -v timbers >/dev/null 2>&1 && timbers hook run claude-stop || true`

// legacyStopCommand is the old plain-text echo format that didn't actually block.
// Kept for backward-compat detection and removal.
//
//nolint:lll // shell one-liner
const legacyStopCommand = `command -v timbers >/dev/null 2>&1 && timbers pending --json 2>/dev/null | grep -q '"count":[1-9][0-9]*' && echo "timbers: undocumented commits - run 'timbers pending' to review" || true`

// legacyPostToolUseBashCommand is the old format that used $TOOL_INPUT (always empty).
// Kept for upgrade detection so reinstall removes stale hooks.
//
//nolint:lll // shell one-liner
const legacyPostToolUseBashCommand = `printf '%s\n' "$TOOL_INPUT" | grep -q 'git commit' && command -v timbers >/dev/null 2>&1 && echo "timbers: remember to run 'timbers log' to document this commit" || true`

// legacyPostToolUseStdinCommand was the fixed version reading stdin.
// Removed because Claude Code doesn't surface PostToolUse hook stdout —
// the Stop hook covers the same case by checking timbers pending at session end.
//
//nolint:lll // shell one-liner
const legacyPostToolUseStdinCommand = `grep -q 'git commit' && command -v timbers >/dev/null 2>&1 && echo "timbers: remember to run 'timbers log' to document this commit" || true`

// timbersHooks defines all hook events timbers installs into Claude Code settings.
var timbersHooks = []timbersHookConfig{
	{Event: "SessionStart", Matcher: "", Command: timbersHookCommand},
	{Event: "PreCompact", Matcher: "", Command: timbersHookCommand},
	{Event: "Stop", Matcher: "", Command: stopHookCommand},
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

// CheckHookStaleness checks if installed timbers hooks are outdated.
// Returns whether any hooks are stale and descriptions of what's outdated.
// A hook is stale if a timbers hook exists for the event but doesn't match the current command.
func CheckHookStaleness(settingsPath string) (stale bool, details []string) {
	settings, err := readSettings(settingsPath)
	if err != nil {
		return false, nil
	}

	for _, cfg := range timbersHooks {
		if hasExactHookCommand(settings, cfg.Event, cfg.Command) {
			continue // This event's hook is current
		}
		if hasHookForEvent(settings, cfg.Event) {
			// Has a timbers hook but it's not the current version
			stale = true
			details = append(details, cfg.Event+": outdated hook command")
			continue
		}
		// Missing entirely — also stale (added in newer version)
		stale = true
		details = append(details, cfg.Event+": missing hook")
	}

	// Check for retired events that still have timbers hooks
	for _, event := range retiredEvents {
		if hasHookForEvent(settings, event) {
			stale = true
			details = append(details, event+": retired hook (will be removed)")
		}
	}

	return stale, details
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
	return cmd == legacyHookCommand || cmd == legacyStopCommand ||
		cmd == legacyPreToolUseCommand ||
		cmd == legacyPostToolUseBashCommand || cmd == legacyPostToolUseStdinCommand
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

// retiredEvents lists hook events that timbers previously installed but no longer uses.
// On upgrade, these are cleaned up to avoid dead hooks lingering in settings.
var retiredEvents = []string{"PostToolUse", "PreToolUse"}
