package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/draft"
	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/llm"
	"github.com/gorewood/timbers/internal/output"
)

type reportSubjectLookup func(string) (string, error)

// newReportCmd creates the report command.
func newReportCmd() *cobra.Command {
	var flags draftFlags
	cmd := &cobra.Command{
		Use:   "report <profile>",
		Short: "Render or generate a configured report",
		Long: `Render a report profile using its default entry scope.

Report profiles are ordinary Timbers templates with a report frontmatter block.
Explicit selection flags replace the profile default. Without --model, the
resolved prompt is printed for piping; with --model, the report is generated.

Examples:
  timbers report decision-digest
  timbers report decision-digest --model opus
  timbers report decision-digest --since 30d --model opus
  timbers report decision-digest --range main..HEAD --model opus`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReport(cmd, args[0], flags)
		},
	}
	cmd.Flags().StringVar(&flags.last, "last", "", "Use last N entries")
	cmd.Flags().StringVar(&flags.since, "since", "", "Use entries since duration (24h, 7d) or date")
	cmd.Flags().StringVar(&flags.until, "until", "", "Use entries until duration (24h, 7d) or date")
	cmd.Flags().StringVar(&flags.rng, "range", "", "Use entries in commit range (A..B)")
	cmd.Flags().StringVar(&flags.appendText, "append", "", "Append extra instructions to the prompt")
	cmd.Flags().StringVarP(&flags.model, "model", "m", "", "Model name for built-in LLM execution")
	cmd.Flags().StringVarP(&flags.provider, "provider", "p", "", "Provider (anthropic, openai, google, local)")
	cmd.Flags().BoolVar(
		&flags.withFrontmatter, "with-frontmatter", false,
		"Include generation metadata as TOML frontmatter (requires --model)",
	)
	cmd.Flags().StringArrayVar(&flags.vars, "var", nil, "Template variable as key=value (repeatable)")
	return cmd
}

func runReport(cmd *cobra.Command, profileName string, flags draftFlags) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd)).
		WithStderr(cmd.ErrOrStderr())
	tmpl, err := draft.LoadTemplate(profileName)
	if err != nil {
		return reportUserError(printer, err.Error())
	}
	if tmpl.Report == nil {
		return reportUserError(printer, fmt.Sprintf(
			"template %q is not a report profile; use 'timbers draft %s' or add report frontmatter",
			profileName, profileName,
		))
	}
	flags, err = resolveReportSelection(tmpl.Report, flags)
	if err != nil {
		return reportUserError(printer, err.Error())
	}
	entries, renderCtx, err := prepareRender(printer, flags)
	if err != nil {
		return err
	}
	metadata := reportMetadata(profileName, tmpl, entries, flags, 0, 0, "")
	if len(entries) == 0 {
		return outputQuietReport(printer, profileName, "no_entries", metadata)
	}

	subjects, resolved, unresolved := resolveReportSubjects(entries, lookupGitSubject)
	metadata = reportMetadata(profileName, tmpl, entries, flags, resolved, unresolved, "")
	renderCtx.EntriesJSON, err = draft.ProjectEntries(entries, tmpl.Report.Projection, subjects)
	if err != nil {
		sysErr := output.NewSystemErrorWithCause("failed to project report entries", err)
		printer.Error(sysErr)
		return sysErr
	}
	rendered, err := draft.Render(tmpl, renderCtx)
	if err != nil {
		sysErr := output.NewSystemErrorWithCause("failed to render report", err)
		printer.Error(sysErr)
		return sysErr
	}
	if flags.model == "" {
		return outputRenderedReport(printer, profileName, tmpl, rendered, entries, metadata)
	}
	return runGeneratedReport(printer, profileName, tmpl, rendered, entries, flags, metadata)
}

func resolveReportSelection(profile *draft.ReportProfile, flags draftFlags) (draftFlags, error) {
	primary := 0
	for _, value := range []string{flags.last, flags.since, flags.rng} {
		if value != "" {
			primary++
		}
	}
	if primary > 1 {
		return flags, errors.New("use only one of --last, --since, or --range")
	}
	if primary == 0 {
		flags.last = profile.Scope.Last
		flags.since = profile.Scope.Since
	}
	if flags.until != "" && flags.since == "" {
		return flags, errors.New("--until requires a --since scope for reports")
	}
	return flags, nil
}

func resolveReportSubjects(
	entries []*ledger.Entry, lookup reportSubjectLookup,
) (map[string]string, int, int) {
	seen := make(map[string]bool)
	subjects := make(map[string]string)
	resolved, unresolved := 0, 0
	for _, entry := range entries {
		for _, sha := range entry.Workset.Commits {
			if sha == "" || seen[sha] {
				continue
			}
			seen[sha] = true
			subject, err := lookup(sha)
			if err != nil || strings.TrimSpace(subject) == "" {
				unresolved++
				continue
			}
			subjects[sha] = strings.TrimSpace(subject)
			resolved++
		}
	}
	return subjects, resolved, unresolved
}

func lookupGitSubject(sha string) (string, error) {
	return git.Run("show", "-s", "--format=%s", sha)
}

func runGeneratedReport(
	printer *output.Printer, profileName string, tmpl *draft.Template, rendered string,
	entries []*ledger.Entry, flags draftFlags, metadata generationMetadata,
) error {
	client, err := llm.New(flags.model, llm.Provider(flags.provider))
	if err != nil {
		return reportUserError(printer, err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	resp, err := client.Complete(ctx, llm.Request{Prompt: rendered})
	if err != nil {
		sysErr := output.NewSystemErrorWithCause("LLM request failed", err)
		printer.Error(sysErr)
		return sysErr
	}
	content := draft.SanitizeLLMOutput(resp.Content)
	metadata.Model = resp.Model
	metadata.Timestamp = time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(content) == strings.TrimSpace(tmpl.Report.QuietOutput) && tmpl.Report.QuietOutput != "" {
		return outputQuietReport(printer, profileName, "no_reportable_content", metadata)
	}
	return outputGeneratedReport(printer, profileName, tmpl, rendered, content, entries, metadata, flags.withFrontmatter)
}

func reportUserError(printer *output.Printer, message string) error {
	err := output.NewUserError(message)
	printer.Error(err)
	return err
}
