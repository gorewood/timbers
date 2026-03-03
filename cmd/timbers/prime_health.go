// Package main provides the entry point for the timbers CLI.
package main

import (
	"path/filepath"

	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

// primeHealthItem represents a single health check issue found during prime.
type primeHealthItem struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// runQuickHealthCheck performs lightweight checks for issues that affect
// logging reliability. Returns nil if everything looks good.
func runQuickHealthCheck() []primeHealthItem {
	var issues []primeHealthItem

	// Check post-commit hook
	if hooksDir, err := setup.GetHooksDir(); err == nil {
		hookPath := filepath.Join(hooksDir, "post-commit")
		if !setup.CheckPostCommitHookStatus(hookPath).Installed {
			issues = append(issues, primeHealthItem{
				Name:    "post_commit_hook",
				Message: "no post-commit hook — agents may forget to log",
			})
		}
	}

	// Check agent env integration
	for _, env := range setup.AllAgentEnvs() {
		if _, _, installed := env.Detect(); !installed {
			issues = append(issues, primeHealthItem{
				Name:    env.Name() + "_integration",
				Message: env.DisplayName() + " hooks not configured",
			})
		}
	}

	return issues
}

// outputPrimeHealth prints a health tip if issues were found.
func outputPrimeHealth(printer *output.Printer, health []primeHealthItem) {
	if len(health) == 0 {
		return
	}
	printer.Println("Health")
	printer.Println("------")
	for _, item := range health {
		printer.Print("  !!  %s\n", item.Message)
	}
	printer.Print("  ->  Run 'timbers doctor --fix' to resolve\n")
	printer.Println()
}
