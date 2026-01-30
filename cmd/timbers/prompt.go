package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/ledger"
	"github.com/rbergman/timbers/internal/llm"
	"github.com/rbergman/timbers/internal/output"
	"github.com/rbergman/timbers/internal/prompt"
	"github.com/spf13/cobra"
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

	cmd := &cobra.Command{
		Use:   "prompt <template>",
		Short: "Render a template with entries for LLM piping",
		Long: `Render a template with ledger entries for piping to an LLM.

Templates are resolved in order:
  1. .timbers/templates/<name>.md (project-local)
  2. ~/.config/timbers/templates/<name>.md (user global)
  3. Built-in templates

By default, outputs the rendered prompt for piping to external tools.
Use --model to execute via the built-in LLM client directly.

Examples:
  timbers prompt changelog --since 7d                    # Output for piping
  timbers prompt changelog --since 7d | claude -p        # Pipe to Claude CLI
  timbers prompt changelog --since 7d --model haiku      # Built-in LLM execution
  timbers prompt exec-summary --last 5 --model sonnet    # Use specific model
  timbers prompt pr-description --range main..HEAD --model gemini-flash
  timbers prompt changelog --since 2026-01-01 --until 2026-01-15  # Date range

  timbers prompt --list                    # List available templates
  timbers prompt changelog --show          # Show template content`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrompt(cmd, args, lastFlag, sinceFlag, untilFlag, rangeFlag, appendFlag, listFlag, showFlag, modelFlag, providerFlag)
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

	return cmd
}

// runPrompt executes the prompt command.
func runPrompt(
	cmd *cobra.Command, args []string,
	lastFlag, sinceFlag, untilFlag, rangeFlag, appendFlag string,
	listFlag, showFlag bool,
	modelFlag, providerFlag string,
) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	// Handle --list
	if listFlag {
		return runPromptList(printer)
	}

	// Template name required for other operations
	if len(args) == 0 {
		err := output.NewUserError("template name required. Run 'timbers prompt --list' to see available templates")
		printer.Error(err)
		return err
	}
	templateName := args[0]

	// Load template
	tmpl, err := prompt.LoadTemplate(templateName)
	if err != nil {
		userErr := output.NewUserError(fmt.Sprintf("template %q not found. Run 'timbers prompt --list' to see available templates", templateName))
		printer.Error(userErr)
		return userErr
	}

	// Handle --show
	if showFlag {
		return runPromptShow(printer, tmpl)
	}

	return runPromptRender(printer, tmpl, templateName, lastFlag, sinceFlag, untilFlag, rangeFlag, appendFlag, modelFlag, providerFlag)
}

// runPromptRender renders the template with entries and outputs the result.
func runPromptRender(
	printer *output.Printer, tmpl *prompt.Template, templateName, lastFlag, sinceFlag, untilFlag, rangeFlag, appendFlag, modelFlag, providerFlag string,
) error {
	// Validate entry selection flags
	if lastFlag == "" && sinceFlag == "" && untilFlag == "" && rangeFlag == "" {
		err := output.NewUserError("specify --last N, --since <duration|date>, --until <duration|date>, or --range A..B to select entries")
		printer.Error(err)
		return err
	}

	// Get entries
	entries, err := getPromptEntries(printer, lastFlag, sinceFlag, untilFlag, rangeFlag)
	if err != nil {
		return err
	}

	// Build render context
	renderCtx := buildRenderContext(entries, appendFlag)

	// Render template
	rendered, err := prompt.Render(tmpl, renderCtx)
	if err != nil {
		sysErr := output.NewSystemError(fmt.Sprintf("failed to render template: %v", err))
		printer.Error(sysErr)
		return sysErr
	}

	// If --model is specified, pipe through LLM client
	if modelFlag != "" {
		return runPromptWithLLM(printer, rendered, templateName, tmpl, entries, modelFlag, providerFlag)
	}

	// Default: output rendered prompt
	if jsonFlag {
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

	// Output result
	if jsonFlag {
		return printer.Success(map[string]any{
			"template":      templateName,
			"template_path": tmpl.Source,
			"prompt":        rendered,
			"entry_count":   len(entries),
			"model":         resp.Model,
			"response":      resp.Content,
		})
	}

	printer.Print("%s\n", resp.Content)
	return nil
}

// buildRenderContext creates a RenderContext from entries and flags.
func buildRenderContext(entries []*ledger.Entry, appendFlag string) *prompt.RenderContext {
	repoName := ""
	if root, rootErr := git.RepoRoot(); rootErr == nil {
		repoName = filepath.Base(root)
	}
	branch, _ := git.CurrentBranch()

	return &prompt.RenderContext{
		Entries:    entries,
		RepoName:   repoName,
		Branch:     branch,
		AppendText: appendFlag,
	}
}

// runPromptList lists available templates.
func runPromptList(printer *output.Printer) error {
	templates, err := prompt.ListTemplates()
	if err != nil {
		sysErr := output.NewSystemError(fmt.Sprintf("failed to list templates: %v", err))
		printer.Error(sysErr)
		return sysErr
	}

	if jsonFlag {
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
	if jsonFlag {
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
func getPromptEntries(printer *output.Printer, lastFlag, sinceFlag, untilFlag, rangeFlag string) ([]*ledger.Entry, error) {
	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return nil, err
	}

	storage := ledger.NewStorage(nil)

	// Parse --since if provided
	var sinceCutoff time.Time
	if sinceFlag != "" {
		var parseErr error
		sinceCutoff, parseErr = parseSinceValue(sinceFlag)
		if parseErr != nil {
			err := output.NewUserError(parseErr.Error())
			printer.Error(err)
			return nil, err
		}
	}

	// Parse --until if provided
	var untilCutoff time.Time
	if untilFlag != "" {
		var parseErr error
		untilCutoff, parseErr = parseUntilValue(untilFlag)
		if parseErr != nil {
			err := output.NewUserError(parseErr.Error())
			printer.Error(err)
			return nil, err
		}
	}

	// If --range is specified, use commit-based filtering
	if rangeFlag != "" {
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

	// If --since or --until is specified, filter by time
	if !sinceCutoff.IsZero() || !untilCutoff.IsZero() {
		return getEntriesByTimeRange(printer, storage, sinceCutoff, untilCutoff, lastFlag)
	}

	// Otherwise use --last
	return getEntriesByLast(printer, storage, lastFlag)
}
