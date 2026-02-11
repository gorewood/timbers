package main

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/llm"
	"github.com/gorewood/timbers/internal/output"
)

type catchupFlags struct {
	model     string
	provider  string
	batchSize int
	rangeStr  string
	parallel  int
	limit     int
	dryRun    bool
	push      bool
	tags      []string
	groupBy   string
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
  timbers catchup --model haiku --limit 5    # Generate at most 5 entries
  timbers catchup --model haiku --group-by day        # Group by day only
  timbers catchup --model haiku --group-by work-item  # Group by work-item trailer

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
	cmd.Flags().IntVarP(&flags.limit, "limit", "l", 0, "Maximum entries to generate (0 = unlimited)")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Preview entries without writing")
	cmd.Flags().BoolVar(&flags.push, "push", false, "Push notes after creating entries")
	cmd.Flags().StringSliceVar(&flags.tags, "tag", nil, "Tags to add to all entries")
	cmd.Flags().StringVarP(&flags.groupBy, "group-by", "g", "auto", "Grouping strategy: auto, day, work-item")

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
	if flags.limit < 0 {
		return output.NewUserError("limit must be non-negative, got " + strconv.Itoa(flags.limit))
	}
	switch flags.groupBy {
	case "auto", "day", "work-item":
		// valid
	default:
		return output.NewUserError("group-by must be auto, day, or work-item; got " + flags.groupBy)
	}
	return nil
}

func runCatchup(cmd *cobra.Command, flags catchupFlags) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

	storage, groups, err := setupCatchup(printer, flags)
	if err != nil {
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

// setupCatchup validates flags, resolves storage, and gathers commit groups.
func setupCatchup(printer *output.Printer, flags catchupFlags) (*ledger.Storage, []commitGroup, error) {
	if err := validateCatchupFlags(flags); err != nil {
		printer.Error(err)
		return nil, nil, err
	}

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return nil, nil, err
	}

	storage, err := ledger.NewDefaultStorage()
	if err != nil {
		printer.Error(err)
		return nil, nil, err
	}
	commits, err := getCatchupCommits(storage, flags.rangeStr)
	if err != nil {
		printer.Error(err)
		return nil, nil, err
	}
	if len(commits) == 0 {
		err := output.NewUserError("no pending commits; run 'timbers pending'")
		printer.Error(err)
		return nil, nil, err
	}

	groups := groupCommitsByStrategy(commits, GroupStrategy(flags.groupBy))
	if len(groups) == 0 {
		err := output.NewUserError("no groups found for processing")
		printer.Error(err)
		return nil, nil, err
	}

	// Apply limit if specified (limits number of entries/groups, not commits)
	if flags.limit > 0 && len(groups) > flags.limit {
		groups = groups[:flags.limit]
	}

	return storage, groups, nil
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
