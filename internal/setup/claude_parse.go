package setup

// getEventGroups parses hook groups from settings for a specific event.
func getEventGroups(settings map[string]any, event string) []hookGroup {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return nil
	}

	groups, ok := hooks[event].([]any)
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

// getSessionStartGroups parses SessionStart hook groups from settings.
func getSessionStartGroups(settings map[string]any) []hookGroup {
	return getEventGroups(settings, "SessionStart")
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
