package main

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/ledger"
	"github.com/rbergman/timbers/internal/llm"
	"github.com/rbergman/timbers/internal/output"
	"github.com/spf13/cobra"
)

type catchupFlags struct {
	model     string
	provider  string
	batchSize int
	rangeStr  string
	parallel  int
	dryRun    bool
	push      bool
	tags      []string
}

type catchupResult struct {
	Status  string            `json:"status"`
	Count   int               `json:"count"`
	Entries []catchupEntryRef `json:"entries"`
}

type catchupEntryRef struct {
	ID       string `json:"id"`
	Anchor   string `json:"anchor"`
	GroupKey string `json:"group_key"`
	What     string `json:"what"`
	Why      string `json:"why"`
	How      string `json:"how"`
}

func newCatchupCmd() *cobra.Command {
	var flags catchupFlags

	cmd := &cobra.Command{
		Use:   "catchup",
		Short: "Generate ledger entries for historical commits using LLM",
		Long: `Generate ledger entries for undocumented commits using an LLM.

This command groups pending commits (by work-item or day) and uses an LLM
to generate meaningful what/why/how summaries from commit messages and diffs.

Examples:
  timbers catchup --model haiku              # Catch up with default model
  timbers catchup --model haiku --dry-run    # Preview without writing
  timbers catchup --model haiku --parallel 10

Environment variables:
  ANTHROPIC_API_KEY  Required for Anthropic models (haiku, sonnet, opus)
  OPENAI_API_KEY     Required for OpenAI models (gpt-4o, etc.)`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCatchup(cmd, flags)
		},
	}

	cmd.Flags().StringVarP(&flags.model, "model", "m", "local", "Model name (default: local)")
	cmd.Flags().StringVarP(&flags.provider, "provider", "p", "", "Provider (anthropic, openai, google, local)")
	cmd.Flags().IntVar(&flags.batchSize, "batch-size", 20, "Max commits per LLM call")
	cmd.Flags().StringVar(&flags.rangeStr, "range", "", "Specific commit range (A..B)")
	cmd.Flags().IntVar(&flags.parallel, "parallel", 5, "Concurrent LLM calls")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Preview entries without writing")
	cmd.Flags().BoolVar(&flags.push, "push", false, "Push notes after creating entries")
	cmd.Flags().StringSliceVar(&flags.tags, "tag", nil, "Tags to add to all entries")

	return cmd
}

// validateCatchupFlags validates the LLM-related flags.
func validateCatchupFlags(flags catchupFlags) error {
	if flags.parallel <= 0 {
		return output.NewUserError("parallel must be positive, got " + strconv.Itoa(flags.parallel))
	}
	if flags.batchSize <= 0 {
		return output.NewUserError("batch-size must be positive, got " + strconv.Itoa(flags.batchSize))
	}
	return nil
}

func runCatchup(cmd *cobra.Command, flags catchupFlags) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	// Validate flags before any other work
	if err := validateCatchupFlags(flags); err != nil {
		printer.Error(err)
		return err
	}

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	storage := ledger.NewStorage(nil)
	commits, err := getCatchupCommits(storage, flags.rangeStr)
	if err != nil {
		printer.Error(err)
		return err
	}
	if len(commits) == 0 {
		err := output.NewUserError("no pending commits; run 'timbers pending'")
		printer.Error(err)
		return err
	}

	groups := groupCommits(commits)
	if len(groups) == 0 {
		err := output.NewUserError("no groups found for processing")
		printer.Error(err)
		return err
	}

	client, err := llm.New(flags.model, llm.Provider(flags.provider))
	if err != nil {
		userErr := output.NewUserError(err.Error())
		printer.Error(userErr)
		return userErr
	}

	entries, err := processCatchupGroups(cmd.Context(), storage, client, groups, flags, printer)
	if err != nil {
		return err
	}

	return outputCatchupResult(printer, entries, flags)
}

func getCatchupCommits(storage *ledger.Storage, rangeStr string) ([]git.Commit, error) {
	if rangeStr != "" {
		parts := strings.SplitN(rangeStr, "..", 2)
		if len(parts) != 2 {
			return nil, output.NewUserError("invalid range format; use A..B")
		}
		return storage.LogRange(parts[0], parts[1])
	}
	commits, _, err := storage.GetPendingCommits()
	return commits, err
}

type catchupWorkerResult struct {
	entry *catchupEntryRef
	err   error
}

func processCatchupGroups(
	ctx context.Context, storage *ledger.Storage, client *llm.Client,
	groups []commitGroup, flags catchupFlags, printer *output.Printer,
) ([]catchupEntryRef, error) {
	results := make(chan catchupWorkerResult, len(groups))
	semaphore := make(chan struct{}, flags.parallel)
	var waitGroup sync.WaitGroup

	for _, grp := range groups {
		waitGroup.Add(1)
		go catchupWorker(ctx, storage, client, grp, flags, printer, semaphore, results, &waitGroup)
	}

	go func() {
		waitGroup.Wait()
		close(results)
	}()

	return collectCatchupResults(results)
}

func catchupWorker(
	ctx context.Context, storage *ledger.Storage, client *llm.Client,
	group commitGroup, flags catchupFlags, printer *output.Printer,
	sem chan struct{}, results chan<- catchupWorkerResult, wg *sync.WaitGroup,
) {
	defer wg.Done()
	sem <- struct{}{}
	defer func() { <-sem }()

	if ctx.Err() != nil {
		return
	}

	entry, err := processSingleCatchupGroup(ctx, storage, client, group, flags, printer)
	results <- catchupWorkerResult{entry: entry, err: err}
}

func collectCatchupResults(results <-chan catchupWorkerResult) ([]catchupEntryRef, error) {
	var entries []catchupEntryRef
	var firstErr error
	for result := range results {
		if result.err != nil && firstErr == nil {
			firstErr = result.err
		}
		if result.entry != nil {
			entries = append(entries, *result.entry)
		}
	}
	return entries, firstErr
}

func processSingleCatchupGroup(
	ctx context.Context, storage *ledger.Storage, client *llm.Client,
	group commitGroup, flags catchupFlags, printer *output.Printer,
) (*catchupEntryRef, error) {
	req := llm.Request{System: catchupSystemPrompt, Prompt: buildCatchupPrompt(group)}
	resp, err := client.Complete(ctx, req)
	if err != nil {
		sysErr := output.NewSystemErrorWithCause("LLM generation failed for group "+group.key, err)
		printer.Error(sysErr)
		return nil, sysErr
	}

	what, why, how := parseCatchupResponse(resp.Content)
	entry := buildCatchupEntry(storage, group, what, why, how, flags.tags)

	if !flags.dryRun {
		if err := storage.WriteEntry(entry, false); err != nil {
			printer.Error(err)
			return nil, err
		}
	}

	return &catchupEntryRef{
		ID: entry.ID, Anchor: entry.Workset.AnchorCommit, GroupKey: group.key,
		What: what, Why: why, How: how,
	}, nil
}

const catchupSystemPrompt = `You are a development documentation assistant. ` +
	`Given git commits, generate a concise what/why/how summary.

Output EXACTLY in this format (3 lines, no extra text):
WHAT: <one sentence describing what was done>
WHY: <one sentence explaining the motivation>
HOW: <one sentence describing the approach>

Rules: Be concise (<100 chars each). Use active voice. Infer reason if unclear.`

func buildCatchupPrompt(group commitGroup) string {
	var b strings.Builder
	b.WriteString("Group: ")
	b.WriteString(group.key)
	b.WriteString("\nCommits: ")
	b.WriteString(strconv.Itoa(len(group.commits)))
	b.WriteString("\n\n")

	for idx, commit := range group.commits {
		b.WriteString("--- Commit ")
		b.WriteString(strconv.Itoa(idx + 1))
		b.WriteString(" ---\nSHA: ")
		b.WriteString(commit.Short)
		b.WriteString("\nDate: ")
		b.WriteString(commit.Date.Format("2006-01-02 15:04"))
		b.WriteString("\nSubject: ")
		b.WriteString(commit.Subject)
		b.WriteString("\n")
		if commit.Body != "" {
			b.WriteString("Body:\n")
			b.WriteString(commit.Body)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func parseCatchupResponse(response string) (what, why, how string) {
	for line := range strings.SplitSeq(response, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "WHAT:"):
			what = strings.TrimSpace(strings.TrimPrefix(line, "WHAT:"))
		case strings.HasPrefix(line, "WHY:"):
			why = strings.TrimSpace(strings.TrimPrefix(line, "WHY:"))
		case strings.HasPrefix(line, "HOW:"):
			how = strings.TrimSpace(strings.TrimPrefix(line, "HOW:"))
		}
	}
	if what == "" {
		what = "Auto-documented commits"
	}
	if why == "" {
		why = "Historical documentation"
	}
	if how == "" {
		how = "See commit messages for details"
	}
	return what, why, how
}

func buildCatchupEntry(
	storage *ledger.Storage, group commitGroup, what, why, how string, tags []string,
) *ledger.Entry {
	anchor := group.commits[0].SHA
	now := time.Now().UTC()
	return &ledger.Entry{
		Schema: ledger.SchemaVersion, Kind: ledger.KindEntry,
		ID: ledger.GenerateID(anchor, now), CreatedAt: now, UpdatedAt: now,
		Workset: ledger.Workset{
			AnchorCommit: anchor, Commits: extractCommitSHAs(group.commits),
			Range: buildCommitRange(group.commits),
			Diffstat: func() *ledger.Diffstat {
				d := getBatchDiffstat(storage, group.commits, anchor)
				return &ledger.Diffstat{Files: d.Files, Insertions: d.Insertions, Deletions: d.Deletions}
			}(),
		},
		Summary:   ledger.Summary{What: what, Why: why, How: how},
		Tags:      tags,
		WorkItems: extractWorkItemsFromKey(group.key),
	}
}

func outputCatchupResult(printer *output.Printer, entries []catchupEntryRef, flags catchupFlags) error {
	status := "created"
	if flags.dryRun {
		status = "dry_run"
	}
	if jsonFlag {
		return printer.WriteJSON(catchupResult{Status: status, Count: len(entries), Entries: entries})
	}
	if flags.dryRun {
		printer.Print("Dry run - would create %d entries:\n\n", len(entries))
	} else {
		printer.Print("Created %d entries:\n\n", len(entries))
	}
	for _, e := range entries {
		printer.Print("  %s [%s]\n", e.ID, e.GroupKey)
		printer.Print("    What: %s\n    Why:  %s\n    How:  %s\n\n",
			truncateString(e.What, 70), truncateString(e.Why, 70), truncateString(e.How, 70))
	}
	return nil
}
