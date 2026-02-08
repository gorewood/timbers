package main

import (
	"fmt"

	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

// runSetupClaudeCheck reports the installation status.
func runSetupClaudeCheck(printer *output.Printer, hookPath, scope string) error {
	installed := setup.IsTimbersSectionInstalled(hookPath)

	if jsonFlag {
		return printer.Success(map[string]any{
			"integration": "claude",
			"installed":   installed,
			"location":    hookPath,
			"scope":       scope,
		})
	}

	printer.Section("Claude Integration Status")
	printer.KeyValue("Scope", scope)
	printer.KeyValue("Location", hookPath)
	if installed {
		printer.KeyValue("Status", "installed")
	} else {
		printer.KeyValue("Status", "not installed")
	}
	return nil
}

// runSetupClaudeRemove removes the timbers section from the hook.
func runSetupClaudeRemove(printer *output.Printer, hookPath, scope string, dryRun bool) error {
	installed := setup.IsTimbersSectionInstalled(hookPath)

	if !installed {
		if jsonFlag {
			return printer.Success(map[string]any{
				"status":      "not_installed",
				"integration": "claude",
				"scope":       scope,
			})
		}
		return printer.Success(map[string]any{
			"message": "Claude integration is not installed",
		})
	}

	if dryRun {
		if jsonFlag {
			return printer.Success(map[string]any{
				"status":      "dry_run",
				"integration": "claude",
				"action":      "would remove",
				"location":    hookPath,
				"scope":       scope,
			})
		}
		printer.Section("Dry Run")
		printer.KeyValue("Action", "would remove timbers section")
		printer.KeyValue("Location", hookPath)
		return nil
	}

	if err := setup.RemoveTimbersSectionFromHook(hookPath); err != nil {
		printer.Error(err)
		return err
	}

	if jsonFlag {
		return printer.Success(map[string]any{
			"status":      "removed",
			"integration": "claude",
			"location":    hookPath,
			"scope":       scope,
		})
	}
	return printer.Success(map[string]any{
		"message": "Removed Claude integration from " + hookPath,
	})
}

// runSetupClaudeInstall installs or updates the timbers section in the hook.
func runSetupClaudeInstall(printer *output.Printer, hookPath, scope string, dryRun bool) error {
	installed := setup.IsTimbersSectionInstalled(hookPath)

	if dryRun {
		action := "would install"
		if installed {
			action = "would update (already installed)"
		}

		if jsonFlag {
			return printer.Success(map[string]any{
				"status":            "dry_run",
				"integration":       "claude",
				"action":            action,
				"location":          hookPath,
				"scope":             scope,
				"already_installed": installed,
			})
		}
		printer.Section("Dry Run")
		printer.KeyValue("Action", action)
		printer.KeyValue("Location", hookPath)
		return nil
	}

	if err := setup.InstallTimbersSection(hookPath); err != nil {
		printer.Error(err)
		return err
	}

	msg := "Installed"
	if installed {
		msg = "Updated"
	}

	if jsonFlag {
		return printer.Success(map[string]any{
			"status":      "installed",
			"integration": "claude",
			"location":    hookPath,
			"scope":       scope,
		})
	}
	return printer.Success(map[string]any{
		"message": fmt.Sprintf("%s Claude integration at %s", msg, hookPath),
	})
}
