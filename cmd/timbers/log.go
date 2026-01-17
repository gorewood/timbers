// Package main provides the entry point for the timbers CLI.
package main

import (
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/steveyegge/timbers/internal/git"
	"github.com/steveyegge/timbers/internal/ledger"
	"github.com/steveyegge/timbers/internal/output"
)

// newLogCmd creates the log command.
func newLogCmd() *cobra.Command {
	return newLogCmdInternal(nil)
}

// logFlags holds all flag values for the log command.
type logFlags struct {
	why       string
	how       string
	tags      []string
	workItems []string
	rangeStr  string
	anchor    string
	minor     bool
	dryRun    bool
	push      bool
}

// newLogCmdInternal creates the log command with optional storage injection.
// If storage is nil, a real storage is created when the command runs.
func newLogCmdInternal(storage *ledger.Storage) *cobra.Command {
	var (
		whyFlag    string
		howFlag    string
		tags       []string
		workItems  []string
		rangeFlag  string
		anchorFlag string
		minorFlag  bool
		dryRunFlag bool
		pushFlag   bool
	)

	cmd := &cobra.Command{
		Use:   "log <what>",
		Short: "Record work as a ledger entry",
		Long: `Record work as a development ledger entry with what/why/how summary.

The log command captures what you did, why you did it, and how you did it.
This creates a structured record attached to your git commits.

Examples:
  timbers log "Fixed auth bug" --why "Users couldn't login" --how "Added null check"
  timbers log "Updated deps" --minor
  timbers log "New feature" --why "User request" --how "New component" --tag feature
  timbers log "Bug fix" --why "Issue #123" --how "Patched" --work-item jira:PROJ-456`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLog(cmd, storage, args, logFlags{
				why:       whyFlag,
				how:       howFlag,
				tags:      tags,
				workItems: workItems,
				rangeStr:  rangeFlag,
				anchor:    anchorFlag,
				minor:     minorFlag,
				dryRun:    dryRunFlag,
				push:      pushFlag,
			})
		},
	}

	cmd.Flags().StringVar(&whyFlag, "why", "", "Why this change was made (required unless --minor)")
	cmd.Flags().StringVar(&howFlag, "how", "", "How this change was implemented (required unless --minor)")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tags for categorization (repeatable)")
	cmd.Flags().StringArrayVar(&workItems, "work-item", nil, "Work item reference as system:id (repeatable)")
	cmd.Flags().StringVar(&rangeFlag, "range", "", "Explicit commit range (e.g., abc123..def456)")
	cmd.Flags().StringVar(&anchorFlag, "anchor", "", "Override anchor commit (default: HEAD)")
	cmd.Flags().BoolVar(&minorFlag, "minor", false, "Trivial change - makes why/how optional")
	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be written without writing")
	cmd.Flags().BoolVar(&pushFlag, "push", false, "Push notes after writing")

	return cmd
}

// logContext holds all data needed to create a log entry.
type logContext struct {
	what      string
	flags     logFlags
	commits   []git.Commit
	anchor    string
	diffstat  git.Diffstat
	workItems []ledger.WorkItem
}

// runLog executes the log command.
func runLog(cmd *cobra.Command, storage *ledger.Storage, args []string, flags logFlags) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	storage, err := initLogStorage(storage, printer)
	if err != nil {
		return err
	}

	ctx, err := prepareLogContext(storage, args, flags, printer)
	if err != nil {
		return err
	}

	entry := buildEntry(ctx)

	if flags.dryRun {
		return outputDryRun(printer, entry)
	}

	return executeLogWrite(storage, entry, flags, printer)
}

// initLogStorage initializes the storage, checking for git repo if needed.
func initLogStorage(storage *ledger.Storage, printer *output.Printer) (*ledger.Storage, error) {
	if storage == nil && !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return nil, err
	}

	if storage == nil {
		storage = ledger.NewStorage(nil)
	}
	return storage, nil
}

// prepareLogContext validates inputs and gathers all data needed for the entry.
func prepareLogContext(
	storage *ledger.Storage,
	args []string,
	flags logFlags,
	printer *output.Printer,
) (*logContext, error) {
	what, err := validateLogInput(args, flags)
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	parsedWorkItems, err := parseWorkItems(flags.workItems)
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	commits, fromRef, err := getLogCommits(storage, flags)
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	if len(commits) == 0 {
		err := output.NewUserError("no pending commits to document; run 'timbers pending' to check status")
		printer.Error(err)
		return nil, err
	}

	anchor := determineAnchor(commits, flags.anchor)

	diffstat, err := getDiffstatForRange(storage, fromRef, anchor, commits)
	if err != nil {
		diffstat = git.Diffstat{}
	}

	return &logContext{
		what:      what,
		flags:     flags,
		commits:   commits,
		anchor:    anchor,
		diffstat:  diffstat,
		workItems: parsedWorkItems,
	}, nil
}

// executeLogWrite writes the entry and optionally pushes.
func executeLogWrite(
	storage *ledger.Storage,
	entry *ledger.Entry,
	flags logFlags,
	printer *output.Printer,
) error {
	if err := storage.WriteEntry(entry, false); err != nil {
		printer.Error(err)
		return err
	}

	pushedMsg := ""
	if flags.push {
		if pushErr := storage.PushNotes("origin"); pushErr != nil {
			pushedMsg = " (push failed: " + pushErr.Error() + ")"
		} else {
			pushedMsg = " (Pushed to origin)"
		}
	}

	return outputLogSuccess(printer, entry, pushedMsg)
}

// getLogCommits retrieves the commits to include in the entry.
func getLogCommits(storage *ledger.Storage, flags logFlags) ([]git.Commit, string, error) {
	if flags.rangeStr != "" {
		parts := strings.SplitN(flags.rangeStr, "..", 2)
		fromRef := parts[0]
		toRef := parts[1]
		commits, err := storage.LogRange(fromRef, toRef)
		if err != nil {
			return nil, "", err
		}
		return commits, fromRef, nil
	}

	commits, _, err := storage.GetPendingCommits()
	if err != nil {
		return nil, "", err
	}

	fromRef := ""
	if len(commits) > 0 {
		fromRef = commits[len(commits)-1].SHA + "^"
	}

	return commits, fromRef, nil
}

// determineAnchor determines the anchor commit for the entry.
func determineAnchor(commits []git.Commit, anchorOverride string) string {
	if anchorOverride != "" {
		return anchorOverride
	}
	if len(commits) > 0 {
		return commits[0].SHA
	}
	return ""
}

// getDiffstatForRange gets the diffstat for a commit range.
func getDiffstatForRange(
	storage *ledger.Storage,
	fromRef, toRef string,
	commits []git.Commit,
) (git.Diffstat, error) {
	if fromRef == "" && len(commits) > 0 {
		fromRef = commits[len(commits)-1].SHA + "^"
	}
	if fromRef == "" {
		return git.Diffstat{}, nil
	}
	return storage.GetDiffstat(fromRef, toRef)
}

// buildEntry constructs the ledger entry from the context.
func buildEntry(ctx *logContext) *ledger.Entry {
	now := time.Now().UTC()

	why := ctx.flags.why
	how := ctx.flags.how
	if ctx.flags.minor {
		if why == "" {
			why = "Minor change"
		}
		if how == "" {
			how = "Minor change"
		}
	}

	commitSHAs := make([]string, len(ctx.commits))
	for i, commit := range ctx.commits {
		commitSHAs[i] = commit.SHA
	}

	rangeStr := ""
	if len(ctx.commits) > 1 {
		rangeStr = ctx.commits[len(ctx.commits)-1].Short + ".." + ctx.commits[0].Short
	}

	return &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        ledger.GenerateID(ctx.anchor, now),
		CreatedAt: now,
		UpdatedAt: now,
		Workset: ledger.Workset{
			AnchorCommit: ctx.anchor,
			Commits:      commitSHAs,
			Range:        rangeStr,
			Diffstat: &ledger.Diffstat{
				Files:      ctx.diffstat.Files,
				Insertions: ctx.diffstat.Insertions,
				Deletions:  ctx.diffstat.Deletions,
			},
		},
		Summary: ledger.Summary{
			What: ctx.what,
			Why:  why,
			How:  how,
		},
		Tags:      ctx.flags.tags,
		WorkItems: ctx.workItems,
	}
}
