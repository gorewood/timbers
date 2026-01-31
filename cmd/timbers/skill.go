// Package main provides the entry point for the timbers CLI.
package main

import (
	"github.com/rbergman/timbers/internal/output"
	"github.com/spf13/cobra"
)

// skillResult holds the structured skill documentation.
type skillResult struct {
	Concepts skillConcepts  `json:"concepts"`
	Workflow skillWorkflow  `json:"workflow"`
	Commands []skillCommand `json:"commands"`
	Contract skillContract  `json:"contract"`
}

// skillConcepts describes core timbers concepts.
type skillConcepts struct {
	Definition string   `json:"definition"`
	DevLedger  string   `json:"dev_ledger"`
	Entry      string   `json:"entry"`
	Workset    string   `json:"workset"`
	Summary    string   `json:"summary"`
	KeyPoints  []string `json:"key_points"`
}

// skillWorkflow describes the typical workflow.
type skillWorkflow struct {
	Description string      `json:"description"`
	Phases      []workPhase `json:"phases"`
}

// workPhase describes a workflow phase.
type workPhase struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
}

// skillCommand documents a single command.
type skillCommand struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Usage       string        `json:"usage"`
	Flags       []commandFlag `json:"flags,omitempty"`
	Examples    []string      `json:"examples,omitempty"`
}

// commandFlag documents a command flag.
type commandFlag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
}

// skillContract documents the output contract.
type skillContract struct {
	Schema      string     `json:"schema"`
	ExitCodes   []exitCode `json:"exit_codes"`
	ErrorFormat string     `json:"error_format"`
	JSONSupport string     `json:"json_support"`
}

// exitCode documents an exit code.
type exitCode struct {
	Code        int    `json:"code"`
	Meaning     string `json:"meaning"`
	Description string `json:"description"`
}

// newSkillCmd creates the skill command.
func newSkillCmd() *cobra.Command {
	var formatFlag string
	var includeExamples bool

	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Output skill documentation for building agent skills",
		Long: `Skill outputs documentation for building AI agent skills.

This command provides:
  - Core concepts: what is timbers, dev ledger, entries, worksets
  - Workflow patterns: how to use timbers in a session
  - Command reference: all commands with flags
  - Contract: JSON schema, exit codes, error format

Examples:
  timbers skill                     # Output as markdown
  timbers skill --format json       # Output as JSON
  timbers skill --include-examples  # Include usage examples`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSkill(cmd, formatFlag, includeExamples)
		},
	}

	cmd.Flags().StringVar(&formatFlag, "format", "md", "Output format: md or json")
	cmd.Flags().BoolVar(&includeExamples, "include-examples", false, "Include usage examples")

	return cmd
}

// runSkill executes the skill command.
func runSkill(cmd *cobra.Command, formatFlag string, includeExamples bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	if formatFlag != "md" && formatFlag != "json" {
		err := output.NewUserError("--format must be 'md' or 'json'")
		printer.Error(err)
		return err
	}

	result := buildSkillData(includeExamples)

	if jsonFlag || formatFlag == "json" {
		return printer.WriteJSON(result)
	}

	outputSkillMarkdown(printer, result, includeExamples)
	return nil
}

// buildSkillData constructs the skill documentation data.
func buildSkillData(includeExamples bool) *skillResult {
	return &skillResult{
		Concepts: buildConcepts(),
		Workflow: buildWorkflow(),
		Commands: buildCommands(includeExamples),
		Contract: buildContract(),
	}
}

// buildConcepts returns the core concepts section.
func buildConcepts() skillConcepts {
	return skillConcepts{
		Definition: "Timbers is a Git-native development ledger that captures what/why/how as structured records.",
		DevLedger:  "A development ledger is a persistent record of work that pairs objective facts from Git with human-authored rationale.",
		Entry:      "An entry is a single ledger record documenting a unit of work. It contains a workset (git data) and summary (what/why/how).",
		Workset:    "A workset captures the Git evidence: anchor commit, commit list, range, and diffstat.",
		Summary:    "A summary provides the rationale: what was done, why it was done, and how it was accomplished.",
		KeyPoints: []string{
			"Entries are stored in Git notes (refs/notes/timbers) and sync with remotes",
			"Each entry has a unique ID: tb_<timestamp>_<short-sha>",
			"The ledger is append-only; entries document completed work",
			"All commands support --json for structured output",
		},
	}
}

// buildWorkflow returns the workflow patterns section.
func buildWorkflow() skillWorkflow {
	return skillWorkflow{
		Description: "A typical session follows: prime -> work -> pending -> log -> query",
		Phases: []workPhase{
			{Name: "Prime", Command: "timbers prime", Description: "Bootstrap session context."},
			{Name: "Work", Command: "(git commits)", Description: "Do development work."},
			{Name: "Check", Command: "timbers pending", Description: "Review undocumented commits."},
			{Name: "Log", Command: `timbers log "..." --why "..." --how "..."`, Description: "Document work."},
			{Name: "Query", Command: "timbers query --last 5", Description: "Review recent entries."},
			{Name: "Sync", Command: "timbers notes push", Description: "Push notes to remote."},
		},
	}
}

// buildCommands returns the command reference.
func buildCommands(includeExamples bool) []skillCommand {
	commands := getCoreCommands()
	if includeExamples {
		addExamplesToCommands(commands)
	}
	return commands
}

// getCoreCommands returns the base command definitions.
func getCoreCommands() []skillCommand {
	return []skillCommand{
		{Name: "log", Description: "Record work as a ledger entry",
			Usage: "timbers log <what> --why <why> --how <how> [flags]",
			Flags: []commandFlag{
				{Name: "--why", Description: "Why (required unless --minor/--auto)"},
				{Name: "--how", Description: "How (required unless --minor/--auto)"},
				{Name: "--tag", Description: "Add tag (repeatable)"},
				{Name: "--work-item", Description: "Link work item (system:id)"},
				{Name: "--range", Description: "Commit range (A..B)"},
				{Name: "--minor", Description: "Use defaults for trivial changes"},
				{Name: "--auto", Description: "Extract what/why/how from commits"},
				{Name: "--yes", Description: "Skip confirmation in auto mode"},
				{Name: "--batch", Description: "Create entries by work-item/day"},
				{Name: "--dry-run", Description: "Preview without writing"},
				{Name: "--push", Description: "Push notes after logging"},
			}},
		{Name: "pending", Description: "Show undocumented commits",
			Usage: "timbers pending [flags]",
			Flags: []commandFlag{{Name: "--count", Description: "Show only count"}}},
		{Name: "prime", Description: "Session context injection",
			Usage: "timbers prime [flags]",
			Flags: []commandFlag{{Name: "--last", Description: "Recent entries", Default: "3"}}},
		{Name: "status", Description: "Show repository and notes state",
			Usage: "timbers status [flags]"},
		{Name: "show", Description: "Display a single entry",
			Usage: "timbers show [<id>] [flags]",
			Flags: []commandFlag{{Name: "--latest", Description: "Show most recent entry"}}},
		{Name: "query", Description: "Search and retrieve entries",
			Usage: "timbers query [flags]",
			Flags: []commandFlag{
				{Name: "--last", Description: "Show last N entries"},
				{Name: "--since", Description: "Entries since duration (24h, 7d) or date"},
				{Name: "--until", Description: "Entries until duration (24h, 7d) or date"},
				{Name: "--oneline", Description: "Compact output"}}},
		{Name: "export", Description: "Export entries to formats",
			Usage: "timbers export [flags]",
			Flags: []commandFlag{
				{Name: "--last", Description: "Export last N"},
				{Name: "--since", Description: "Entries since duration (24h, 7d) or date"},
				{Name: "--until", Description: "Entries until duration (24h, 7d) or date"},
				{Name: "--range", Description: "Commit range (A..B)"},
				{Name: "--format", Description: "json or md"},
				{Name: "--out", Description: "Output directory"}}},
		{Name: "notes", Description: "Notes management",
			Usage: "timbers notes <subcommand>"},
	}
}

// addExamplesToCommands adds examples to each command.
func addExamplesToCommands(commands []skillCommand) {
	for i := range commands {
		commands[i].Examples = getCommandExamples(commands[i].Name)
	}
}

// getCommandExamples returns examples for a command.
func getCommandExamples(name string) []string {
	examples := map[string][]string{
		"log":     {`timbers log "Added auth" --why "Security" --how "JWT"`, `timbers log "Fix" --why "Bug" --how "Check" --tag bugfix`},
		"pending": {`timbers pending`, `timbers pending --count`},
		"prime":   {`timbers prime`, `timbers prime --last 5`},
		"status":  {`timbers status`, `timbers status --json`},
		"show":    {`timbers show <id>`, `timbers show --last`},
		"query":   {`timbers query --last 5`, `timbers query --last 10 --oneline`},
		"export":  {`timbers export --last 5 --json`, `timbers export --format md --out ./notes/`},
		"notes":   {`timbers notes init`, `timbers notes push`},
	}
	return examples[name]
}

// buildContract returns the contract section.
func buildContract() skillContract {
	return skillContract{
		Schema: "timbers.devlog/v1",
		ExitCodes: []exitCode{
			{Code: 0, Meaning: "Success", Description: "Command completed successfully"},
			{Code: 1, Meaning: "User error", Description: "Bad arguments, missing fields, not found"},
			{Code: 2, Meaning: "System error", Description: "Git failed, I/O error"},
			{Code: 3, Meaning: "Conflict", Description: "Entry exists, state mismatch"},
		},
		ErrorFormat: `{"error": "message", "code": N}`,
		JSONSupport: "All commands support --json for structured output",
	}
}

// outputSkillMarkdown writes the skill data as markdown.
func outputSkillMarkdown(printer *output.Printer, result *skillResult, includeExamples bool) {
	printer.Println("# Timbers Skill Documentation")
	printer.Println()
	outputConceptsMarkdown(printer, &result.Concepts)
	outputWorkflowMarkdown(printer, &result.Workflow)
	outputCommandsMarkdown(printer, result.Commands, includeExamples)
	outputContractMarkdown(printer, &result.Contract)
}

func outputConceptsMarkdown(printer *output.Printer, concepts *skillConcepts) {
	printer.Println("## Core Concepts")
	printer.Println()
	printer.Print("**Timbers**: %s\n\n", concepts.Definition)
	printer.Print("**Development Ledger**: %s\n\n", concepts.DevLedger)
	printer.Print("**Entry**: %s\n\n", concepts.Entry)
	printer.Print("**Workset**: %s\n\n", concepts.Workset)
	printer.Print("**Summary**: %s\n\n", concepts.Summary)
	printer.Println("### Key Points")
	printer.Println()
	for _, point := range concepts.KeyPoints {
		printer.Print("- %s\n", point)
	}
	printer.Println()
}

func outputWorkflowMarkdown(printer *output.Printer, w *skillWorkflow) {
	printer.Println("## Workflow Patterns")
	printer.Println()
	printer.Println(w.Description)
	printer.Println()
	for _, phase := range w.Phases {
		printer.Print("### %s\n**Command**: `%s`\n\n%s\n\n", phase.Name, phase.Command, phase.Description)
	}
}

func outputCommandsMarkdown(printer *output.Printer, commands []skillCommand, includeExamples bool) {
	printer.Println("## Command Reference")
	printer.Println()
	for _, cmd := range commands {
		outputSingleCommandMarkdown(printer, &cmd, includeExamples)
	}
}

func outputSingleCommandMarkdown(printer *output.Printer, cmd *skillCommand, includeExamples bool) {
	printer.Print("### %s\n\n%s\n\n**Usage**: `%s`\n\n", cmd.Name, cmd.Description, cmd.Usage)
	if len(cmd.Flags) > 0 {
		printer.Println("**Flags**:")
		for _, flag := range cmd.Flags {
			if flag.Default != "" {
				printer.Print("- `%s`: %s (default: %s)\n", flag.Name, flag.Description, flag.Default)
			} else {
				printer.Print("- `%s`: %s\n", flag.Name, flag.Description)
			}
		}
		printer.Println()
	}
	if includeExamples && len(cmd.Examples) > 0 {
		printer.Println("**Examples**:")
		for _, example := range cmd.Examples {
			printer.Print("```bash\n%s\n```\n", example)
		}
		printer.Println()
	}
}

func outputContractMarkdown(printer *output.Printer, contract *skillContract) {
	printer.Println("## Contract")
	printer.Println()
	printer.Print("**Schema**: `%s`\n\n", contract.Schema)
	printer.Print("**JSON Support**: %s\n\n", contract.JSONSupport)
	printer.Print("**Error Format**: `%s`\n\n", contract.ErrorFormat)
	printer.Println("### Exit Codes")
	printer.Println()
	printer.Println("| Code | Meaning | Description |")
	printer.Println("|------|---------|-------------|")
	for _, ec := range contract.ExitCodes {
		printer.Print("| %d | %s | %s |\n", ec.Code, ec.Meaning, ec.Description)
	}
}
