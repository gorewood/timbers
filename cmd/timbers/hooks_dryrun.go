package main

import (
	"path/filepath"

	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

// handleInstallDryRun handles dry-run output for install.
func handleInstallDryRun(
	printer *output.Printer, env setup.HookEnvInfo, force bool,
) error {
	hookTypes := []string{"pre-commit", "post-commit", "post-rewrite"}
	actions := make(map[string]string)

	for _, hookType := range hookTypes {
		actions[hookType] = describeInstallDryRunAction(
			env, hookType, force,
		)
	}

	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":       "dry_run",
			"tier":         tierString(env.Tier),
			"tier_desc":    tierDescription(env.Tier, env.Owner),
			"hooks_dir":    env.HooksDir,
			"owner":        env.Owner,
			"pre_commit":   actions["pre-commit"],
			"post_commit":  actions["post-commit"],
			"post_rewrite": actions["post-rewrite"],
		})
	}

	printer.Section("Dry Run")
	printer.KeyValue("Tier", tierDescription(env.Tier, env.Owner))
	printer.KeyValue("Hooks dir", env.HooksDir)
	if env.Owner != "" {
		printer.KeyValue("Owner", env.Owner)
	}
	printer.Println()
	for _, hookType := range hookTypes {
		printer.KeyValue("  "+hookType, actions[hookType])
	}

	return nil
}

// describeInstallDryRunAction returns the action description for a dry-run.
func describeInstallDryRunAction(
	env setup.HookEnvInfo, hookType string, force bool,
) string {
	hookPath := filepath.Join(env.HooksDir, hookType)

	switch {
	case setup.HasTimbersSection(hookPath):
		return "already installed (no-op)"
	case env.Tier == setup.HookEnvUnknownOverride && !force:
		return "would skip (unknown hook environment; use --force)"
	case setup.HookExists(hookPath):
		appendable, reason := setup.IsAppendable(hookPath)
		if !appendable {
			return "would skip (hook is a " + reason + ")"
		}
		return "would append timbers section"
	default:
		return "would create with timbers section"
	}
}
