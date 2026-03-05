package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

// postCommitSectionContent is the timbers section content for the post-commit hook.
// Does not include delimiters — AppendTimbersSection adds those.
const postCommitSectionContent = `if command -v timbers >/dev/null 2>&1; then
  timbers hook run post-commit "$@"
fi
`

// newHooksInstallCmd creates the hooks install subcommand.
func newHooksInstallCmd() *cobra.Command {
	var chain bool
	var force bool
	var skip bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install timbers git hooks",
		Long: `Install timbers git hooks. Respects core.hooksPath if configured.

Installs pre-commit, post-commit, and post-rewrite hooks as delimited
sections appended to existing hook files (or creates new files).

The pre-commit hook blocks commits when undocumented commits exist,
requiring 'timbers log' before continuing. Use --no-verify to bypass.

Use --force to install even when core.hooksPath points to an unknown location.
Use --skip to exit 0 on any conflict (for automation).`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHooksInstall(cmd, force, skip, dryRun)
		},
	}

	cmd.Flags().BoolVar(&chain, "chain", false, "Deprecated: append is now the default behavior")
	cmd.Flags().BoolVar(&force, "force", false, "Install even in unknown hook environments (Tier 4)")
	cmd.Flags().BoolVar(&skip, "skip", false, "Exit 0 on conflict (for automation)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without doing it")

	// Hide --chain: it's a deprecated alias that maps to default behavior.
	_ = cmd.Flags().MarkHidden("chain")

	return cmd
}

// runHooksInstall executes the hooks install command.
func runHooksInstall(cmd *cobra.Command, force, skip, dryRun bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd))

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	env, err := setup.ClassifyHookEnv()
	if err != nil {
		printer.Error(err)
		return err
	}

	if dryRun {
		return handleInstallDryRun(printer, env, force)
	}

	return performInstall(printer, env, force, skip)
}

// performInstall does the actual hook installation using tier-based logic.
func performInstall(printer *output.Printer, env setup.HookEnvInfo, force, skip bool) error {
	// Tier 4: unknown override — error unless --force or --skip.
	if env.Tier == setup.HookEnvUnknownOverride && !force {
		if skip {
			return outputInstallSkipped(printer, env)
		}
		err := output.NewUserError(
			"core.hooksPath is set to " + env.HooksDir +
				"; timbers won't modify hooks it doesn't recognize." +
				" Use --force to override, or add timbers manually." +
				" See `timbers hooks status` for details.")
		printer.Error(err)
		return err
	}

	// Install all three hook types.
	installed := make(map[string]string) // hookType -> action description
	var errors []string

	hookSpecs := []struct {
		hookType string
		content  string
	}{
		{"pre-commit", preCommitSectionContent},
		{"post-commit", postCommitSectionContent},
		{"post-rewrite", postRewriteTimbersSection()},
	}

	for _, spec := range hookSpecs {
		action, installErr := installHookSection(env, spec.hookType, spec.content)
		if installErr != nil {
			errors = append(errors, spec.hookType+": "+installErr.Error())
		} else {
			installed[spec.hookType] = action
		}
	}

	if len(errors) > 0 {
		sysErr := output.NewSystemError("some hooks failed to install: " + fmt.Sprintf("%v", errors))
		printer.Error(sysErr)
		return sysErr
	}

	return outputInstallSuccess(printer, env, installed)
}

// installHookSection installs a single hook type using AppendTimbersSection.
// Returns (action description, error).
func installHookSection(
	env setup.HookEnvInfo, hookType, sectionContent string,
) (string, error) {
	hookPath := filepath.Join(env.HooksDir, hookType)

	// Check if already installed.
	if setup.HasTimbersSection(hookPath) {
		return "already installed", nil
	}

	// If file exists, check if appendable.
	if setup.HookExists(hookPath) {
		appendable, reason := setup.IsAppendable(hookPath)
		if !appendable {
			return "", fmt.Errorf(
				"hook is a %s; add `timbers hook run %s \"$@\"` manually",
				reason, hookType,
			)
		}
	}

	if err := setup.AppendTimbersSection(hookPath, sectionContent); err != nil {
		return "", err
	}

	if env.Owner != "" {
		return "installed (alongside " + env.Owner + " hook)", nil
	}
	if env.HasHook {
		return "installed (appended to existing hook)", nil
	}
	return "installed", nil
}

// outputInstallSuccess outputs the success message for install.
func outputInstallSuccess(
	printer *output.Printer, env setup.HookEnvInfo, installed map[string]string,
) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":       "ok",
			"tier":         tierString(env.Tier),
			"hooks_dir":    env.HooksDir,
			"owner":        env.Owner,
			"pre_commit":   installed["pre-commit"],
			"post_commit":  installed["post-commit"],
			"post_rewrite": installed["post-rewrite"],
		})
	}

	for hookType, action := range installed {
		printer.Println(hookType + ": " + action)
	}
	return printer.Success(map[string]any{"message": "Hooks installed"})
}

// outputInstallSkipped outputs the message when install is skipped via --skip.
func outputInstallSkipped(printer *output.Printer, env setup.HookEnvInfo) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":    "skipped",
			"tier":      tierString(env.Tier),
			"hooks_dir": env.HooksDir,
			"reason":    "unknown hook environment; use --force to override",
		})
	}
	return printer.Success(map[string]any{
		"message": "Skipped: core.hooksPath set to " + env.HooksDir +
			" (unknown environment)",
	})
}
