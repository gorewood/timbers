// Package main provides the entry point for the timbers CLI.
package main

import (
	"strings"

	"github.com/gorewood/timbers/internal/setup"
)

// checkAgentIntegrations checks all registered agent environments.
func checkAgentIntegrations(flags *doctorFlags) []checkResult {
	envs := setup.AllAgentEnvs()
	results := make([]checkResult, 0, len(envs))
	for _, env := range envs {
		results = append(results, checkAgentEnv(env, flags))
	}
	return results
}

// checkAgentEnv checks a single agent environment's integration status.
// Detects both missing hooks and stale (outdated) hook configurations.
func checkAgentEnv(env setup.AgentEnv, flags *doctorFlags) checkResult {
	name := env.DisplayName() + " Integration"

	path, _, installed := env.Detect()
	if !installed {
		return fixOrWarnAgentEnv(env, name, flags)
	}

	// Hooks are installed — check if they're outdated
	if result, stale := checkAgentEnvStaleness(env, name, path, flags); stale {
		return result
	}

	return checkResult{
		Name:    name,
		Status:  checkPass,
		Message: "timbers configured in " + env.DisplayName() + " hooks",
	}
}

// fixOrWarnAgentEnv attempts auto-fix or returns a warning for missing hooks.
func fixOrWarnAgentEnv(env setup.AgentEnv, name string, flags *doctorFlags) checkResult {
	if flags.fix {
		if _, err := env.Install(true); err == nil {
			return checkResult{
				Name:    name,
				Status:  checkPass,
				Message: "timbers configured in " + env.DisplayName() + " hooks (auto-fixed)",
			}
		}
	}
	return checkResult{
		Name:    name,
		Status:  checkWarn,
		Message: env.DisplayName() + " hooks not configured for timbers",
		Hint:    "Run 'timbers setup " + env.Name() + "' or 'timbers doctor --fix'",
	}
}

// checkAgentEnvStaleness checks if installed hooks are outdated.
// Returns the result and whether staleness was detected.
func checkAgentEnvStaleness(env setup.AgentEnv, name, path string, flags *doctorFlags) (checkResult, bool) {
	if env.Name() != "claude" {
		return checkResult{}, false
	}

	stale, details := setup.CheckHookStaleness(path)
	if !stale {
		return checkResult{}, false
	}

	if flags.fix {
		if _, err := env.Install(true); err == nil {
			return checkResult{
				Name:    name,
				Status:  checkPass,
				Message: "hooks upgraded (auto-fixed)",
			}, true
		}
	}
	return checkResult{
		Name:    name,
		Status:  checkWarn,
		Message: "hooks installed but outdated: " + strings.Join(details, "; "),
		Hint:    "Run 'timbers setup " + env.Name() + "' or 'timbers doctor --fix'",
	}, true
}
