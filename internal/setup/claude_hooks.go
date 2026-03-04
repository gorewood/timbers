package setup

// addTimbersHooks adds all timbers hooks, upgrading stale hooks to current versions.
func addTimbersHooks(settings map[string]any) {
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = make(map[string]any)
		settings["hooks"] = hooks
	}

	// Clean up retired events
	for _, event := range retiredEvents {
		removeTimbersHooksFromEvent(hooks, event)
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
