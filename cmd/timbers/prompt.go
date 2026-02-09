package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/llm"
	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/prompt"
)

// newPromptCmd creates the prompt command.
func newPromptCmd() *cobra.Command {
	var lastFlag string
	var sinceFlag string
	var untilFlag string
	var rangeFlag string
	var appendFlag string
	var listFlag bool
	var showFlag bool
	var modelFlag string
	var providerFlag string
	var withFrontmatterFlag bool

	cmd := &cobra.Command{
		Use:   "prompt <template>",
		Short: "Render a template with entries for LLM piping",
		Long: `Render a template with ledger entries for piping to an LLM.

Templates resolve: project (.timbers/templates/) → global → built-in.
Use --model to execute via built-in LLM client directly.

Examples:
  timbers prompt changelog --since 7d                  # For piping
  timbers prompt changelog --since 7d --model haiku    # Direct LLM
  timbers prompt --list                                # List templates
  timbers prompt devblog --since 7d --model haiku --with-frontmatter`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := promptFlags{
				last: lastFlag, since: sinceFlag, until: untilFlag, rng: rangeFlag,
				appendText: appendFlag, list: listFlag, show: showFlag,
				model: modelFlag, provider: providerFlag, withFrontmatter: withFrontmatterFlag,
			}
			return runPrompt(cmd, args, flags)
		},
	}

	cmd.Flags().StringVar(&lastFlag, "last", "", "Use last N entries")
	cmd.Flags().StringVar(&sinceFlag, "since", "", "Use entries since duration (24h, 7d) or date")
	cmd.Flags().StringVar(&untilFlag, "until", "", "Use entries until duration (24h, 7d) or date")
	cmd.Flags().StringVar(&rangeFlag, "range", "", "Use entries in commit range (A..B)")
	cmd.Flags().StringVar(&appendFlag, "append", "", "Append extra instructions to the prompt")
	cmd.Flags().BoolVar(&listFlag, "list", false, "List available templates")
	cmd.Flags().BoolVar(&showFlag, "show", false, "Show template content without rendering")
	cmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Model name for built-in LLM execution (e.g., haiku, sonnet, gemini-flash)")
	cmd.Flags().StringVarP(&providerFlag, "provider", "p", "", "Provider (anthropic, openai, google, local) - inferred if omitted")
	cmd.Flags().BoolVar(&withFrontmatterFlag, "with-frontmatter", false, "Include generation metadata as TOML frontmatter (requires --model)")

	return cmd
}

// runPrompt executes the prompt command.
func runPrompt(cmd *cobra.Command, args []string, flags promptFlags) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

	// Handle --list
	if flags.list {
		return runPromptList(printer)
	}

	// Template name required for other operations
	if len(args) == 0 {
		err := output.NewUserError("template name required. Use 'timbers prompt --list'")
		printer.Error(err)
		return err
	}
	templateName := args[0]

	// Load template
	tmpl, err := prompt.LoadTemplate(templateName)
	if err != nil {
		userErr := output.NewUserError(fmt.Sprintf("template %q not found", templateName))
		printer.Error(userErr)
		return userErr
	}

	// Handle --show
	if flags.show {
		return runPromptShow(printer, tmpl)
	}

	return runPromptRender(cmd, printer, tmpl, templateName, flags)
}

// runPromptRender renders the template with entries and outputs the result.
func runPromptRender(
	_ *cobra.Command, printer *output.Printer,
	tmpl *prompt.Template, templateName string, flags promptFlags,
) error {
	// Validate entry selection flags
	if flags.last == "" && flags.since == "" && flags.until == "" && flags.rng == "" {
		err := output.NewUserError("specify --last, --since, --until, or --range")
		printer.Error(err)
		return err
	}

	// Get entries
	entries, err := getPromptEntries(printer, flags.last, flags.since, flags.until, flags.rng)
	if err != nil {
		return err
	}

	// Build render context
	renderCtx := buildRenderContext(entries, flags.appendText)

	// Render template
	rendered, err := prompt.Render(tmpl, renderCtx)
	if err != nil {
		sysErr := output.NewSystemError(fmt.Sprintf("failed to render: %v", err))
		printer.Error(sysErr)
		return sysErr
	}

	// If --model is specified, pipe through LLM client
	if flags.model != "" {
		selFlags := promptSelectionFlags{
			last: flags.last, since: flags.since, until: flags.until, rng: flags.rng,
		}
		return runPromptWithLLM(
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

	printer.Print("%s\n", rendered)
	return nil
}

// runPromptWithLLM sends the rendered prompt to an LLM and outputs the response.
func runPromptWithLLM(
	printer *output.Printer, rendered, templateName string,
	tmpl *prompt.Template, entries []*ledger.Entry,
	modelFlag, providerFlag string,
	withFrontmatter bool, selFlags promptSelectionFlags,
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
			"response":       resp.Content,
			"generated_with": metadata,
		}
		return printer.Success(result)
	}

	// With frontmatter: output TOML frontmatter before content
	if withFrontmatter {
		printer.Print("%s\n", formatTOMLFrontmatter(metadata))
	}

	printer.Print("%s\n", resp.Content)
	return nil
}

// runPromptList lists available templates.
func runPromptList(printer *output.Printer) error {
	templates, err := prompt.ListTemplates()
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
	bySource := make(map[string][]prompt.TemplateInfo)
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

// runPromptShow shows template content without rendering.
func runPromptShow(printer *output.Printer, tmpl *prompt.Template) error {
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

// getPromptEntries retrieves entries based on flags.
func getPromptEntries(
	printer *output.Printer, lastFlag, sinceFlag, untilFlag, rangeFlag string,
) ([]*ledger.Entry, error) {
	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return nil, err
	}

	storage := ledger.NewStorage(nil)
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

// parseTimeCutoffs parses --since and --until flags into time cutoffs.
func parseTimeCutoffs(printer *output.Printer, sinceFlag, untilFlag string) (time.Time, time.Time, error) {
	var sinceCutoff, untilCutoff time.Time
	var err error
	if sinceFlag != "" {
		if sinceCutoff, err = parseSinceValue(sinceFlag); err != nil {
			userErr := output.NewUserError(err.Error())
			printer.Error(userErr)
			return time.Time{}, time.Time{}, userErr
		}
	}
	if untilFlag != "" {
		if untilCutoff, err = parseUntilValue(untilFlag); err != nil {
			userErr := output.NewUserError(err.Error())
			printer.Error(userErr)
			return time.Time{}, time.Time{}, userErr
		}
	}
	return sinceCutoff, untilCutoff, nil
}

// getEntriesByRangeWithFilters retrieves entries by commit range with time filters.
func getEntriesByRangeWithFilters(
	printer *output.Printer, storage *ledger.Storage,
	rangeFlag string, sinceCutoff, untilCutoff time.Time,
) ([]*ledger.Entry, error) {
	entries, err := getEntriesByRange(printer, storage, rangeFlag)
	if err != nil {
		return nil, err
	}
	if !sinceCutoff.IsZero() {
		entries = filterEntriesSince(entries, sinceCutoff)
	}
	if !untilCutoff.IsZero() {
		entries = filterEntriesUntil(entries, untilCutoff)
	}
	return entries, nil
}
