package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/ledger"
	"github.com/rbergman/timbers/internal/output"
	"github.com/rbergman/timbers/internal/prompt"
	"github.com/spf13/cobra"
)

// newPromptCmd creates the prompt command.
func newPromptCmd() *cobra.Command {
	var lastFlag string
	var sinceFlag string
	var rangeFlag string
	var appendFlag string
	var listFlag bool
	var showFlag bool

	cmd := &cobra.Command{
		Use:   "prompt <template>",
		Short: "Render a template with entries for LLM piping",
		Long: `Render a template with ledger entries for piping to an LLM.

Templates are resolved in order:
  1. .timbers/templates/<name>.md (project-local)
  2. ~/.config/timbers/templates/<name>.md (user global)
  3. Built-in templates

The -p flag makes Claude print to stdout and exit (vs interactive mode).

Examples:
  timbers prompt changelog --since 7d | claude -p
  timbers prompt exec-summary --last 5 | claude -p --model haiku
  timbers prompt pr-description --range main..HEAD | claude -p
  timbers prompt devblog-gamedev --last 10 --append "Focus on physics" | claude -p

  timbers prompt --list                    # List available templates
  timbers prompt changelog --show          # Show template content`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrompt(cmd, args, lastFlag, sinceFlag, rangeFlag, appendFlag, listFlag, showFlag)
		},
	}

	cmd.Flags().StringVar(&lastFlag, "last", "", "Use last N entries")
	cmd.Flags().StringVar(&sinceFlag, "since", "", "Use entries since duration (24h, 7d) or date")
	cmd.Flags().StringVar(&rangeFlag, "range", "", "Use entries in commit range (A..B)")
	cmd.Flags().StringVar(&appendFlag, "append", "", "Append extra instructions to the prompt")
	cmd.Flags().BoolVar(&listFlag, "list", false, "List available templates")
	cmd.Flags().BoolVar(&showFlag, "show", false, "Show template content without rendering")

	return cmd
}

// runPrompt executes the prompt command.
func runPrompt(cmd *cobra.Command, args []string, lastFlag, sinceFlag, rangeFlag, appendFlag string, listFlag, showFlag bool) error {
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

	return runPromptRender(printer, tmpl, templateName, lastFlag, sinceFlag, rangeFlag, appendFlag)
}

// runPromptRender renders the template with entries and outputs the result.
func runPromptRender(
	printer *output.Printer, tmpl *prompt.Template, templateName, lastFlag, sinceFlag, rangeFlag, appendFlag string,
) error {
	// Validate entry selection flags
	if lastFlag == "" && sinceFlag == "" && rangeFlag == "" {
		err := output.NewUserError("specify --last N, --since <duration|date>, or --range A..B to select entries")
		printer.Error(err)
		return err
	}

	// Get entries
	entries, err := getPromptEntries(printer, lastFlag, sinceFlag, rangeFlag)
	if err != nil {
		return err
	}

	// Build render context
	ctx := buildRenderContext(entries, appendFlag)

	// Render template
	rendered, err := prompt.Render(tmpl, ctx)
	if err != nil {
		sysErr := output.NewSystemError(fmt.Sprintf("failed to render template: %v", err))
		printer.Error(sysErr)
		return sysErr
	}

	// Output
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
func getPromptEntries(printer *output.Printer, lastFlag, sinceFlag, rangeFlag string) ([]*ledger.Entry, error) {
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

	// If --range is specified, use commit-based filtering
	if rangeFlag != "" {
		entries, err := getEntriesByRange(printer, storage, rangeFlag)
		if err != nil {
			return nil, err
		}
		if !sinceCutoff.IsZero() {
			entries = filterEntriesSince(entries, sinceCutoff)
		}
		return entries, nil
	}

	// If --since is specified, filter by time
	if !sinceCutoff.IsZero() {
		return getEntriesBySince(printer, storage, sinceCutoff, lastFlag)
	}

	// Otherwise use --last
	return getEntriesByLast(printer, storage, lastFlag)
}
