// Package main provides the entry point for the timbers CLI.
package main

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// dirtyChecker abstracts working-tree dirty detection for testability.
type dirtyChecker func() bool

// newLogCmd creates the log command.
func newLogCmd() *cobra.Command {
	return newLogCmdInternal(nil, nil)
}

// logFlags holds all flag values for the log command.
type logFlags struct {
	why       string
	how       string
	notes     string
	tags      []string
	workItems []string
	who       []string
	rangeStr  string
	anchor    string
	minor     bool
	dryRun    bool
	push      bool
	auto      bool
	yes       bool
	batch     bool
}

// newLogCmdInternal creates the log command with optional storage and dirty checker injection.
// If storage is nil, a real storage is created when the command runs.
// If isDirty is nil, git.HasUncommittedChanges is used.
func newLogCmdInternal(storage *ledger.Storage, isDirty dirtyChecker) *cobra.Command {
	vars := newLogFlagVars()

	cmd := &cobra.Command{
		Use:   "log [<what>]",
		Short: "Record work as a ledger entry",
		Long:  logCmdLongHelp,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLog(cmd, storage, isDirty, args, vars.toLogFlags())
		},
	}

	registerLogFlags(cmd, vars)
	return cmd
}

// logCmdLongHelp is the long help text for the log command.
const logCmdLongHelp = `Record work as a development ledger entry with what/why/how summary.

The log command captures what you did, why you did it, and how you did it.
This creates a structured record attached to your git commits.

Examples:
  timbers log "Fixed auth bug" --why "Users couldn't login" --how "Added null check"
  timbers log "Updated deps" --minor
  timbers log "New feature" --why "User request" --how "New component" --tag feature
  timbers log "Bug fix" --why "Issue #123" --how "Patched" --work-item jira:PROJ-456
  timbers log "New API" --why "Agents need access" --how "MCP server" --notes "Debated HTTP vs exec wrapping"
  timbers log "Paired work" --why "..." --how "..." --who "Name <email>"
  timbers log --auto              # Extract what/why/how from commit messages
  timbers log --auto --yes        # Auto mode without confirmation
  timbers log --batch             # Create entries for each work-item group or day

Each entry is committed separately (not folded into the code commit). This
enables reliable pending detection and keeps captured text independent of later
SHA rewrites. Timbers relinks known one-to-one local rewrites when possible;
squashes may leave anchors stale. To filter entry commits from git log:
git log --invert-grep --grep="^timbers: document"

Contributor attribution is automatic from mailmap-normalized Git authors and
Co-authored-by trailers. Usually omit --who. Repeat --who "Name <email>" for
pairing, shared work, bots, or correction; any use replaces the automatic set,
so provide every intended contributor. Only provide identities intended for repository publication.`

// logContext holds all data needed to create a log entry.
type logContext struct {
	what         string
	flags        logFlags
	commits      []git.Commit
	anchor       string
	diffstat     git.Diffstat
	workItems    []ledger.WorkItem
	contributors []ledger.Contributor
}

// runLog executes the log command.
func runLog(cmd *cobra.Command, storage *ledger.Storage, isDirty dirtyChecker, args []string, flags logFlags) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd)).
		WithWidth(output.TerminalWidth(cmd.OutOrStdout(), 80))

	storage, err := initLogStorage(storage, printer)
	if err != nil {
		return err
	}

	// Refuse if working tree is dirty: the auto-commit pathspec-scopes to the
	// entry file (internal/ledger/filestorage.go: git commit -- <path>), so
	// staged feature changes stay in the index while the entry rides on the
	// old HEAD. Push then ships a phantom: an entry whose prose describes
	// work that isn't in any commit below it. Most often hit when the
	// pre-commit gate aborted the prior `git commit` and the caller chained
	// `timbers log` after a newline (no &&). --dry-run is still allowed
	// because it short-circuits before the auto-commit and only prints what
	// the entry would look like.
	if isDirty == nil {
		isDirty = git.HasUncommittedChanges
	}
	if isDirty() && !flags.dryRun {
		err := output.NewUserError(
			"working tree has uncommitted changes; commit (or stash) them " +
				"first to avoid phantom entries. If the prior `git commit` " +
				"was aborted by the pre-commit gate, your staged changes " +
				"are still in the index — inspect with: git diff --cached. " +
				"For a no-op peek, re-run with --dry-run.")
		printer.Error(err)
		return err
	}

	// Refuse to log during rebase/merge — we can't commit the entry file
	// and the pending commit set is unreliable.
	if git.IsInteractiveGitOp() {
		err := output.NewUserError(
			"git operation in progress (rebase, merge, or cherry-pick); " +
				"complete it first, then run timbers log")
		printer.Error(err)
		return err
	}

	// Dispatch to batch mode if --batch is set
	if flags.batch {
		return runBatchLog(storage, flags, printer)
	}

	ctx, err := prepareLogContext(storage, args, flags, printer)
	if err != nil {
		return err
	}

	entry := buildEntry(ctx)

	if flags.dryRun {
		return outputDryRun(printer, entry)
	}

	return executeLogWrite(storage, entry, printer)
}

// initLogStorage initializes the storage, checking for git repo if needed.
func initLogStorage(storage *ledger.Storage, printer *output.Printer) (*ledger.Storage, error) {
	if storage == nil && !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return nil, err
	}

	if storage == nil {
		var err error
		storage, err = ledger.NewDefaultStorage()
		if err != nil {
			printer.Error(err)
			return nil, err
		}
	}
	return storage, nil
}

// resolveAnchorFlag resolves a symbolic --anchor (HEAD, a branch, a short SHA)
// to a full SHA in place before it flows into range selection or the stored
// anchor. Persisting a symbolic ref like "HEAD" yields entry ids suffixed
// "_HEAD" and an anchor that changes meaning per-commit and per-worktree,
// defeating the since-anchor model. An unresolvable ref errors here rather than
// writing a phantom entry anchored on nothing. No-op when --anchor is unset.
func resolveAnchorFlag(storage *ledger.Storage, flags *logFlags, printer *output.Printer) error {
	if flags.anchor == "" {
		return nil
	}
	resolved, err := storage.ResolveCommit(flags.anchor)
	if err != nil {
		printer.Error(err)
		return err
	}
	flags.anchor = resolved
	return nil
}

// prepareLogContext validates inputs and gathers all data needed for the entry.
func prepareLogContext(
	storage *ledger.Storage,
	args []string,
	flags logFlags,
	printer *output.Printer,
) (*logContext, error) {
	// For auto mode, we need commits first to extract content
	// So we validate basic input first, then get commits, then extract/validate content
	if err := validateBasicInput(args, flags); err != nil {
		printer.Error(err)
		return nil, err
	}

	parsedWorkItems, err := parseWorkItems(flags.workItems)
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	if err = resolveAnchorFlag(storage, &flags, printer); err != nil {
		return nil, err
	}

	commits, fromRef, staleAnchor, err := getLogCommits(storage, flags)
	if err != nil {
		printer.Error(err)
		return nil, err
	}
	contributors, err := resolveLogContributors(commits, flags.who, staleAnchor, printer)
	if err != nil {
		return nil, err
	}

	// Extract or validate what/why/how based on mode
	what, updatedFlags, err := resolveLogContent(args, flags, commits)
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	anchor := determineAnchor(commits, flags.anchor)

	diffstat, err := getDiffstatForRange(storage, fromRef, anchor, commits)
	if err != nil {
		diffstat = git.Diffstat{}
	}

	return &logContext{
		what:         what,
		flags:        updatedFlags,
		commits:      commits,
		anchor:       anchor,
		diffstat:     diffstat,
		workItems:    parsedWorkItems,
		contributors: contributors,
	}, nil
}

// executeLogWrite writes the entry to the .timbers/ directory.
func executeLogWrite(
	storage *ledger.Storage,
	entry *ledger.Entry,
	printer *output.Printer,
) error {
	if err := storage.WriteEntry(entry, false); err != nil {
		printer.Error(err)
		return err
	}

	// Push-before-log race detection: if the commit we just documented is
	// already on the upstream branch, then the user pushed before logging
	// and the entry we just auto-committed is stranded locally. Without a
	// follow-up push, anyone branching off origin sees the content commit
	// pending with no entry.
	if anchor := entry.Workset.AnchorCommit; anchor != "" && git.IsPushedToUpstream(anchor) {
		printer.Warn(
			"documented commit %s is already pushed, but this entry is not — "+
				"run `git push` to sync the entry",
			shortSHA(anchor),
		)
	}

	return outputLogSuccess(printer, entry)
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
		Notes:        ctx.flags.notes,
		Tags:         ctx.flags.tags,
		WorkItems:    ctx.workItems,
		Contributors: ctx.contributors,
	}
}
