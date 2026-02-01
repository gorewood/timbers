// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rbergman/timbers/internal/output"
)

// resolveClaudeHookPath determines the hook path based on scope.
func resolveClaudeHookPath(project bool) (string, string, error) {
	if project {
		// Project-local installation
		cwd, err := os.Getwd()
		if err != nil {
			return "", "", output.NewSystemErrorWithCause("failed to get working directory", err)
		}
		return filepath.Join(cwd, ".claude", "hooks", "user_prompt_submit.sh"), "project", nil
	}

	// Global installation
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", output.NewSystemErrorWithCause("failed to get home directory", err)
	}
	return filepath.Join(home, ".claude", "hooks", "user_prompt_submit.sh"), "global", nil
}

// runSetupClaudeCheck reports the installation status.
func runSetupClaudeCheck(printer *output.Printer, hookPath, scope string) error {
	installed := isTimbersSectionInstalled(hookPath)

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
	installed := isTimbersSectionInstalled(hookPath)

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

	// Remove the timbers section
	if err := removeTimbersSectionFromHook(hookPath); err != nil {
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
	installed := isTimbersSectionInstalled(hookPath)

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

	// Install/update the hook
	if err := installTimbersSection(hookPath); err != nil {
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

// isTimbersSectionInstalled checks if the timbers section exists in a hook file.
func isTimbersSectionInstalled(hookPath string) bool {
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), timbersHookMarkerBegin)
}

// installTimbersSection adds or updates the timbers section in a hook file.
func installTimbersSection(hookPath string) error {
	// Ensure directory exists
	hookDir := filepath.Dir(hookPath)
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		return output.NewSystemErrorWithCause("failed to create hook directory", err)
	}

	// Read existing content or create new
	var content string
	existingContent, err := os.ReadFile(hookPath)
	if err == nil {
		content = string(existingContent)
		// Remove existing timbers section for replacement
		content = removeTimbersSectionFromContent(content)
	} else if !os.IsNotExist(err) {
		return output.NewSystemErrorWithCause("failed to read hook file", err)
	}

	// Add shebang if not present
	if !strings.HasPrefix(content, "#!") {
		content = "#!/bin/bash\n" + content
	}

	// Append timbers section
	content = strings.TrimRight(content, "\n") + "\n\n" + claudeHookContent + "\n"

	// Write the file
	// #nosec G306 -- hook needs execute permission
	if err := os.WriteFile(hookPath, []byte(content), 0o755); err != nil {
		return output.NewSystemErrorWithCause("failed to write hook file", err)
	}

	return nil
}

// removeTimbersSectionFromHook removes the timbers section from a hook file.
func removeTimbersSectionFromHook(hookPath string) error {
	content, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return output.NewSystemErrorWithCause("failed to read hook file", err)
	}

	newContent := removeTimbersSectionFromContent(string(content))

	// Write back (or delete if empty except shebang)
	cleaned := strings.TrimSpace(strings.TrimPrefix(newContent, "#!/bin/bash"))
	if cleaned == "" {
		// File would be empty, could delete or leave shebang only
		newContent = "#!/bin/bash\n"
	}

	// #nosec G306 -- hook needs execute permission
	if err := os.WriteFile(hookPath, []byte(newContent), 0o755); err != nil {
		return output.NewSystemErrorWithCause("failed to write hook file", err)
	}

	return nil
}

// removeTimbersSectionFromContent removes the timbers section from content string.
func removeTimbersSectionFromContent(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inTimbers := false

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), timbersHookMarkerBegin) {
			inTimbers = true
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), timbersHookMarkerEnd) {
			inTimbers = false
			continue
		}
		if !inTimbers {
			result = append(result, line)
		}
	}

	// Clean up double blank lines that might result from removal
	finalContent := strings.Join(result, "\n")
	for strings.Contains(finalContent, "\n\n\n") {
		finalContent = strings.ReplaceAll(finalContent, "\n\n\n", "\n\n")
	}

	return strings.TrimRight(finalContent, "\n") + "\n"
}
