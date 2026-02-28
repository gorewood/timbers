package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/draft"
	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/llm"
	"github.com/gorewood/timbers/internal/output"
)

// newDraftCmd creates the draft command.
func newDraftCmd() *cobra.Command {
	var lastFlag string
	var sinceFlag string
	var untilFlag string
	var rangeFlag string
	var appendFlag string
	var listFlag bool
	var showFlag bool
	var modelsFlag bool
	var modelFlag string
	var providerFlag string
	var withFrontmatterFlag bool

	cmd := &cobra.Command{
		Use:   "draft <template>",
		Short: "Generate documents from entries (release notes, changelog, blog posts)",
		Long: `Generate documents from ledger entries using prompt templates.

Templates resolve: project (.timbers/templates/) → global → built-in.
Use --model to generate directly, or pipe output to your preferred LLM.

Examples:
  timbers draft release-notes --since 7d               # Render prompt for piping
  timbers draft changelog --last 10 --model opus       # Generate with built-in LLM
  timbers draft devblog --since 7d --model opus --with-frontmatter
  timbers draft decision-log --last 20                 # ADR-style decision log
  timbers draft --list                                 # List available templates
  timbers draft release-notes --last 5 --append "Focus on security changes"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := draftFlags{
				last: lastFlag, since: sinceFlag, until: untilFlag, rng: rangeFlag,
				appendText: appendFlag, list: listFlag, show: showFlag, models: modelsFlag,
				model: modelFlag, provider: providerFlag, withFrontmatter: withFrontmatterFlag,
			}
			return runDraft(cmd, args, flags)
		},
	}

	cmd.Flags().StringVar(&lastFlag, "last", "", "Use last N entries")
	cmd.Flags().StringVar(&sinceFlag, "since", "", "Use entries since duration (24h, 7d) or date")
	cmd.Flags().StringVar(&untilFlag, "until", "", "Use entries until duration (24h, 7d) or date")
	cmd.Flags().StringVar(&rangeFlag, "range", "", "Use entries in commit range (A..B)")
	cmd.Flags().StringVar(&appendFlag, "append", "", "Append extra instructions to the prompt")
	cmd.Flags().BoolVar(&listFlag, "list", false, "List available templates")
	cmd.Flags().BoolVar(&modelsFlag, "models", false, "List providers, model aliases, and required API keys")
	cmd.Flags().BoolVar(&showFlag, "show", false, "Show template content without rendering")
	cmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Model name for built-in LLM execution (e.g., haiku, sonnet, gemini-flash)")
	cmd.Flags().StringVarP(&providerFlag, "provider", "p", "", "Provider (anthropic, openai, google, local) - inferred if omitted")
	cmd.Flags().BoolVar(&withFrontmatterFlag, "with-frontmatter", false, "Include generation metadata as TOML frontmatter (requires --model)")

	return cmd
}

// runDraft executes the draft command.
func runDraft(cmd *cobra.Command, args []string, flags draftFlags) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd)).
		WithStderr(cmd.ErrOrStderr())

	// Handle --list
	if flags.list {
		return runDraftList(printer)
	}

	// Handle --models
	if flags.models {
		return runDraftModels(printer)
	}

	// Template name required for other operations
	if len(args) == 0 {
		err := output.NewUserError("template name required. Use 'timbers draft --list'")
		printer.Error(err)
		return err
	}
	templateName := args[0]

	// Load template
	tmpl, err := draft.LoadTemplate(templateName)
	if err != nil {
		userErr := output.NewUserError(fmt.Sprintf("template %q not found", templateName))
		printer.Error(userErr)
		return userErr
	}

	// Handle --show
	if flags.show {
		return runDraftShow(printer, tmpl)
	}

	return runDraftRender(cmd, printer, tmpl, templateName, flags)
}

// runDraftRender renders the template with entries and outputs the result.
func runDraftRender(
	_ *cobra.Command, printer *output.Printer,
	tmpl *draft.Template, templateName string, flags draftFlags,
) error {
	// Validate entry selection flags
	if flags.last == "" && flags.since == "" && flags.until == "" && flags.rng == "" {
		err := output.NewUserError("specify --last, --since, --until, or --range")
		printer.Error(err)
		return err
	}

	// Get entries
	entries, err := getDraftEntries(printer, flags.last, flags.since, flags.until, flags.rng)
	if err != nil {
		return err
	}

	// Build render context
	renderCtx := buildRenderContext(entries, flags.appendText)

	// Render template
	rendered, err := draft.Render(tmpl, renderCtx)
	if err != nil {
		sysErr := output.NewSystemError(fmt.Sprintf("failed to render: %v", err))
		printer.Error(sysErr)
		return sysErr
	}

	// If --model is specified, pipe through LLM client
	if flags.model != "" {
		selFlags := draftSelectionFlags{
			last: flags.last, since: flags.since, until: flags.until, rng: flags.rng,
		}
		return runDraftWithLLM(
			printer, rendered, templateName, tmpl, entries,
			flags.model, flags.provider, flags.withFrontmatter, selFlags,
		)
	}

	// Default: output rendered prompt
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"template":      templateName,
			"template_path": tmpl.Source,
			"prompt":        rendered,
			"entry_count":   len(entries),
			"entries":       entries,
		})
	}

	// When piped, emit a status hint to stderr so the user gets feedback
	if !printer.IsTTY() {
		printer.Stderr("timbers: rendered %q with %d entries\n", templateName, len(entries))
	}

	printer.Print("%s\n", rendered)
	return nil
}

// runDraftWithLLM sends the rendered prompt to an LLM and outputs the response.
func runDraftWithLLM(
	printer *output.Printer, rendered, templateName string,
	tmpl *draft.Template, entries []*ledger.Entry,
	modelFlag, providerFlag string,
	withFrontmatter bool, selFlags draftSelectionFlags,
) error {
	// Create LLM client
	client, err := llm.New(modelFlag, llm.Provider(providerFlag))
	if err != nil {
		userErr := output.NewUserError(err.Error())
		printer.Error(userErr)
		return userErr
	}

	// Build request
	req := llm.Request{
		Prompt: rendered,
	}

	// Execute with timeout (2 minutes default, same as generate command)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := client.Complete(ctx, req)
	if err != nil {
		sysErr := output.NewSystemErrorWithCause("LLM request failed", err)
		printer.Error(sysErr)
		return sysErr
	}

	// Build metadata
	metadata := buildGenerationMetadata(templateName, tmpl, entries, resp.Model, selFlags)

	// Output result
	if printer.IsJSON() {
		result := map[string]any{
			"template":       templateName,
			"template_path":  tmpl.Source,
			"prompt":         rendered,
			"entry_count":    len(entries),
			"model":          resp.Model,
			"response":       draft.SanitizeLLMOutput(resp.Content),
			"generated_with": metadata,
		}
		return printer.Success(result)
	}

	// Sanitize LLM output to strip preamble/signoff leakage
	content := draft.SanitizeLLMOutput(resp.Content)

	// With frontmatter: output TOML frontmatter before content
	if withFrontmatter {
		printer.Print("%s\n", formatTOMLFrontmatter(metadata))
	}

	printer.Print("%s\n", content)
	return nil
}

// runDraftList lists available templates.
func runDraftList(printer *output.Printer) error {
	templates, err := draft.ListTemplates()
	if err != nil {
		sysErr := output.NewSystemError(fmt.Sprintf("failed to list templates: %v", err))
		printer.Error(sysErr)
		return sysErr
	}

	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"templates": templates,
		})
	}

	// Group by source for human output
	bySource := make(map[string][]draft.TemplateInfo)
	for _, t := range templates {
		bySource[t.Source] = append(bySource[t.Source], t)
	}

	// Print in order: built-in, global, project
	sources := []struct {
		key   string
		label string
	}{
		{"built-in", "Built-in:"},
		{"global", "Global (~/.config/timbers/templates/):"},
		{"project", "Project (.timbers/templates/):"},
	}

	for _, src := range sources {
		if infos, ok := bySource[src.key]; ok && len(infos) > 0 {
			printer.Print("%s\n", src.label)
			for _, info := range infos {
				override := ""
				if info.Overrides != "" {
					override = fmt.Sprintf(" [overrides %s]", info.Overrides)
				}
				printer.Print("  %-20s %s%s\n", info.Name, info.Description, override)
			}
			printer.Print("\n")
		}
	}

	return nil
}

// runDraftShow shows template content without rendering.
func runDraftShow(printer *output.Printer, tmpl *draft.Template) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"name":        tmpl.Name,
			"description": tmpl.Description,
			"source":      tmpl.Source,
			"content":     tmpl.Content,
		})
	}

	printer.Print("# %s\n", tmpl.Name)
	printer.Print("Source: %s\n", tmpl.Source)
	printer.Print("Description: %s\n\n", tmpl.Description)
	printer.Print("---\n%s\n", tmpl.Content)
	return nil
}

// getDraftEntries retrieves entries based on flags.
func getDraftEntries(
	printer *output.Printer, lastFlag, sinceFlag, untilFlag, rangeFlag string,
) ([]*ledger.Entry, error) {
	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return nil, err
	}

	storage, err := ledger.NewDefaultStorage()
	if err != nil {
		printer.Error(err)
		return nil, err
	}
	sinceCutoff, untilCutoff, err := parseTimeCutoffs(printer, sinceFlag, untilFlag)
	if err != nil {
		return nil, err
	}

	if rangeFlag != "" {
		return getEntriesByRangeWithFilters(printer, storage, rangeFlag, sinceCutoff, untilCutoff)
	}
	if !sinceCutoff.IsZero() || !untilCutoff.IsZero() {
		return getEntriesByTimeRange(printer, storage, sinceCutoff, untilCutoff, lastFlag, nil)
	}
	return getEntriesByLast(printer, storage, lastFlag, nil)
}
