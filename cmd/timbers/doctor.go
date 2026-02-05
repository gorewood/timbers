// Package main provides the entry point for the timbers CLI.
package main

import (
	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
	"github.com/spf13/cobra"
)

// checkStatus represents the result of a health check.
type checkStatus string

const (
	checkPass checkStatus = "pass"
	checkWarn checkStatus = "warn"
	checkFail checkStatus = "fail"
)

// checkResult holds the result of a single health check.
type checkResult struct {
	Name    string      `json:"name"`
	Status  checkStatus `json:"status"`
	Message string      `json:"message"`
	Hint    string      `json:"hint,omitempty"`
}

// doctorResult holds all check results organized by category.
type doctorResult struct {
	Version     string         `json:"version"`
	Core        []checkResult  `json:"core"`
	Workflow    []checkResult  `json:"workflow"`
	Integration []checkResult  `json:"integration"`
	Summary     *doctorSummary `json:"summary"`
}

// doctorSummary holds the counts of check results.
type doctorSummary struct {
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Failed   int `json:"failed"`
}

// doctorFlags holds the command-line flags for the doctor command.
type doctorFlags struct {
	fix   bool
	quiet bool
}

// newDoctorCmd creates the doctor command.
func newDoctorCmd() *cobra.Command {
	flags := &doctorFlags{}

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check installation health and suggest fixes",
		Long: `Check timbers installation health and suggest fixes.

Runs a series of health checks across three categories:
  CORE        - Git notes configuration and binary availability
  WORKFLOW    - Pending commits and recent entries
  INTEGRATION - Git hooks and Claude Code integration

Each check reports:
  Pass    - Check passed successfully
  Warning - Non-critical issue found
  Fail    - Critical issue that needs attention

Examples:
  timbers doctor              # Run all health checks
  timbers doctor --fix        # Auto-fix what can be fixed
  timbers doctor --quiet      # Only show failures and warnings
  timbers doctor --json       # Output results as JSON`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDoctor(cmd, flags)
		},
	}

	cmd.Flags().BoolVar(&flags.fix, "fix", false, "Auto-fix what can be fixed (notes init, etc.)")
	cmd.Flags().BoolVar(&flags.quiet, "quiet", false, "Only show failures and warnings")

	return cmd
}

// runDoctor executes the doctor command.
func runDoctor(cmd *cobra.Command, flags *doctorFlags) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	// Check if we're in a git repo
	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	// Run all checks
	result := gatherDoctorChecks(flags)

	// Output based on mode
	if jsonFlag {
		return outputDoctorJSON(printer, result)
	}

	outputDoctorHuman(printer, result, flags.quiet)
	return nil
}

// gatherDoctorChecks runs all health checks and returns results.
func gatherDoctorChecks(flags *doctorFlags) *doctorResult {
	result := &doctorResult{
		Version:     version,
		Core:        runCoreChecks(flags),
		Workflow:    runWorkflowChecks(),
		Integration: runIntegrationChecks(),
		Summary:     &doctorSummary{},
	}

	// Calculate summary
	allChecks := append(append(result.Core, result.Workflow...), result.Integration...)
	for _, check := range allChecks {
		switch check.Status {
		case checkPass:
			result.Summary.Passed++
		case checkWarn:
			result.Summary.Warnings++
		case checkFail:
			result.Summary.Failed++
		}
	}

	return result
}

// outputDoctorJSON outputs the doctor result as JSON.
func outputDoctorJSON(printer *output.Printer, result *doctorResult) error {
	data := map[string]any{
		"version":     result.Version,
		"core":        result.Core,
		"workflow":    result.Workflow,
		"integration": result.Integration,
		"summary": map[string]any{
			"passed":   result.Summary.Passed,
			"warnings": result.Summary.Warnings,
			"failed":   result.Summary.Failed,
		},
	}
	return printer.WriteJSON(data)
}

// outputDoctorHuman outputs the doctor result in human-readable format.
func outputDoctorHuman(printer *output.Printer, result *doctorResult, quiet bool) {
	// Header
	printer.Println()
	printer.Print("timbers doctor v%s\n", result.Version)

	// Core checks
	printCheckSection(printer, "CORE", result.Core, quiet)

	// Workflow checks
	printCheckSection(printer, "WORKFLOW", result.Workflow, quiet)

	// Integration checks
	printCheckSection(printer, "INTEGRATION", result.Integration, quiet)

	// Summary
	printer.Println()
	printer.Print("%s %d passed  %s %d warnings  %s %d failed\n",
		statusIcon(checkPass), result.Summary.Passed,
		statusIcon(checkWarn), result.Summary.Warnings,
		statusIcon(checkFail), result.Summary.Failed,
	)
}

// printCheckSection prints a section of checks.
func printCheckSection(printer *output.Printer, title string, checks []checkResult, quiet bool) {
	// In quiet mode, skip sections with only passing checks
	if quiet {
		hasNonPass := false
		for _, check := range checks {
			if check.Status != checkPass {
				hasNonPass = true
				break
			}
		}
		if !hasNonPass {
			return
		}
	}

	printer.Println()
	printer.Println(title)

	for _, check := range checks {
		// In quiet mode, skip passing checks
		if quiet && check.Status == checkPass {
			continue
		}

		printer.Print("  %s  %s %s\n", statusIcon(check.Status), check.Name, check.Message)
		if check.Hint != "" {
			printer.Print("     %s %s\n", hintPrefix(), check.Hint)
		}
	}
}

// statusIcon returns the icon for a check status.
func statusIcon(status checkStatus) string {
	switch status {
	case checkPass:
		return "ok"
	case checkWarn:
		return "!!"
	case checkFail:
		return "XX"
	default:
		return "??"
	}
}

// hintPrefix returns the prefix for hint lines.
func hintPrefix() string {
	return "->"
}
