// Package main provides the entry point for the timbers CLI.
package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
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
	Config      []checkResult  `json:"config"`
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

Runs a series of health checks across four categories:
  CORE        - Storage directory, binary, and version update check
  CONFIG      - Config directory, env files, API keys, templates
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

	cmd.Flags().BoolVar(&flags.fix, "fix", false, "Auto-fix what can be fixed")
	cmd.Flags().BoolVar(&flags.quiet, "quiet", false, "Only show failures and warnings")

	return cmd
}

// runDoctor executes the doctor command.
func runDoctor(cmd *cobra.Command, flags *doctorFlags) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

	// Check if we're in a git repo
	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	// Run all checks
	result := gatherDoctorChecks(flags)

	// Output based on mode
	if printer.IsJSON() {
		return outputDoctorJSON(printer, result)
	}

	outputDoctorHuman(printer, result, flags.quiet)
	return nil
}

// gatherDoctorChecks runs all health checks and returns results.
func gatherDoctorChecks(flags *doctorFlags) *doctorResult {
	result := &doctorResult{
		Version:     version,
		Core:        runCoreChecks(),
		Config:      runConfigChecks(flags),
		Workflow:    runWorkflowChecks(),
		Integration: runIntegrationChecks(flags),
		Summary:     &doctorSummary{},
	}

	// Calculate summary
	allChecks := append(append(append(result.Core, result.Config...), result.Workflow...), result.Integration...)
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
		"config":      result.Config,
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

// doctorStyleSet holds lipgloss styles for doctor output.
type doctorStyleSet struct {
	heading lipgloss.Style
	section lipgloss.Style
	pass    lipgloss.Style
	warn    lipgloss.Style
	fail    lipgloss.Style
	hint    lipgloss.Style
	dim     lipgloss.Style
}

// doctorStyles returns a TTY-aware style set.
func doctorStyles(isTTY bool) doctorStyleSet {
	if !isTTY {
		return doctorStyleSet{}
	}
	return doctorStyleSet{
		heading: lipgloss.NewStyle().Bold(true),
		section: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		pass:    lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		warn:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		fail:    lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		hint:    lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		dim:     lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	}
}

// outputDoctorHuman outputs the doctor result in human-readable format.
func outputDoctorHuman(printer *output.Printer, result *doctorResult, quiet bool) {
	styles := doctorStyles(printer.IsTTY())

	// Header
	printer.Println()
	ver := result.Version
	if ver != "" && ver[0] != 'v' {
		ver = "v" + ver
	}
	printer.Println(styles.heading.Render("timbers doctor") + " " + styles.dim.Render(ver))

	// Sections
	printCheckSection(printer, styles, "CORE", result.Core, quiet)
	printCheckSection(printer, styles, "CONFIG", result.Config, quiet)
	printCheckSection(printer, styles, "WORKFLOW", result.Workflow, quiet)
	printCheckSection(printer, styles, "INTEGRATION", result.Integration, quiet)

	// Summary
	printer.Println()
	printer.Print("%s %d passed  %s %d warnings  %s %d failed\n",
		styles.pass.Render("ok"), result.Summary.Passed,
		styles.warn.Render("!!"), result.Summary.Warnings,
		styles.fail.Render("XX"), result.Summary.Failed,
	)
}

// printCheckSection prints a section of checks.
func printCheckSection(printer *output.Printer, styles doctorStyleSet, title string, checks []checkResult, quiet bool) {
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
	printer.Println(styles.section.Render(title))

	for _, check := range checks {
		// In quiet mode, skip passing checks
		if quiet && check.Status == checkPass {
			continue
		}

		icon := styledIcon(styles, check.Status)
		printer.Print("  %s  %s %s\n", icon, check.Name, styles.dim.Render(check.Message))
		if check.Hint != "" {
			printer.Print("     %s %s\n", styles.hint.Render("->"), check.Hint)
		}
	}
}

// styledIcon returns the styled icon for a check status.
func styledIcon(styles doctorStyleSet, status checkStatus) string {
	switch status {
	case checkPass:
		return styles.pass.Render("ok")
	case checkWarn:
		return styles.warn.Render("!!")
	case checkFail:
		return styles.fail.Render("XX")
	default:
		return "??"
	}
}
