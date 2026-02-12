// Package main provides the entry point for the timbers CLI.
package main

import "github.com/spf13/cobra"

// logFlagVars holds the flag variable pointers for the log command.
type logFlagVars struct {
	why       *string
	how       *string
	notes     *string
	tags      *[]string
	workItems *[]string
	rangeStr  *string
	anchor    *string
	minor     *bool
	dryRun    *bool
	push      *bool
	auto      *bool
	yes       *bool
	batch     *bool
}

// toLogFlags converts flag vars to a logFlags struct.
func (vars *logFlagVars) toLogFlags() logFlags {
	return logFlags{
		why:       *vars.why,
		how:       *vars.how,
		notes:     *vars.notes,
		tags:      *vars.tags,
		workItems: *vars.workItems,
		rangeStr:  *vars.rangeStr,
		anchor:    *vars.anchor,
		minor:     *vars.minor,
		dryRun:    *vars.dryRun,
		push:      *vars.push,
		auto:      *vars.auto,
		yes:       *vars.yes,
		batch:     *vars.batch,
	}
}

// newLogFlagVars creates initialized flag variable pointers.
func newLogFlagVars() *logFlagVars {
	return &logFlagVars{
		why:       new(string),
		how:       new(string),
		notes:     new(string),
		tags:      new([]string),
		workItems: new([]string),
		rangeStr:  new(string),
		anchor:    new(string),
		minor:     new(bool),
		dryRun:    new(bool),
		push:      new(bool),
		auto:      new(bool),
		yes:       new(bool),
		batch:     new(bool),
	}
}

// registerLogFlags registers all flags on the log command.
func registerLogFlags(cmd *cobra.Command, flagVars *logFlagVars) {
	cmd.Flags().StringVar(flagVars.why, "why", "", "Why this change was made (required unless --minor or --auto)")
	cmd.Flags().StringVar(flagVars.how, "how", "", "How this change was implemented (required unless --minor or --auto)")
	cmd.Flags().StringArrayVar(flagVars.tags, "tag", nil, "Tags for categorization (repeatable)")
	cmd.Flags().StringArrayVar(flagVars.workItems, "work-item", nil, "Work item reference as system:id (repeatable)")
	cmd.Flags().StringVar(flagVars.rangeStr, "range", "", "Explicit commit range (e.g., abc123..def456)")
	cmd.Flags().StringVar(flagVars.anchor, "anchor", "", "Override anchor commit (default: HEAD)")
	cmd.Flags().BoolVar(flagVars.minor, "minor", false, "Trivial change - makes why/how optional")
	cmd.Flags().BoolVar(flagVars.dryRun, "dry-run", false, "Show what would be written without writing")
	cmd.Flags().BoolVar(flagVars.push, "push", false, "Push to remote after writing")
	cmd.Flags().BoolVar(flagVars.auto, "auto", false, "Extract what/why/how from commit messages")
	cmd.Flags().BoolVar(flagVars.yes, "yes", false, "Skip confirmation in auto mode")
	cmd.Flags().StringVar(flagVars.notes, "notes", "", "Deliberation notes capturing the journey to a decision")
	cmd.Flags().BoolVar(flagVars.batch, "batch", false, "Create entries grouped by work-item trailer or day")
}
