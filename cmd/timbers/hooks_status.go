package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

// hooksStatusResult holds the data for hooks status output.
type hooksStatusResult struct {
	Environment hooksStatusEnv      `json:"environment"`
	Hooks       hooksStatusHooks    `json:"hooks"`
	Steering    hooksStatusSteering `json:"steering"`
}

// hooksStatusEnv describes the hook environment classification.
type hooksStatusEnv struct {
	Tier     string `json:"tier"`
	HooksDir string `json:"hooks_dir"`
	Owner    string `json:"owner,omitempty"`
}

// hooksStatusHookInfo describes the status of a single hook type.
type hooksStatusHookInfo struct {
	Installed bool   `json:"installed"`
	Format    string `json:"format,omitempty"`
}

// hooksStatusHooks holds per-hook-type status.
type hooksStatusHooks struct {
	PreCommit   hooksStatusHookInfo `json:"pre_commit"`
	PostCommit  hooksStatusHookInfo `json:"post_commit"`
	PostRewrite hooksStatusHookInfo `json:"post_rewrite"`
}

// hooksStatusSteering describes Claude Code steering status.
type hooksStatusSteering struct {
	ClaudeCode bool `json:"claude_code"`
}

// newHooksStatusCmd creates the hooks status subcommand.
func newHooksStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show hook environment and integration status",
		Long: `Show the hook environment classification, per-hook integration state,
and Claude Code steering status.

Useful for understanding why hooks were or weren't installed, and for
debugging hook integration issues.`,
		RunE: runHooksStatus,
	}
}

// runHooksStatus executes the hooks status command.
func runHooksStatus(cmd *cobra.Command, _ []string) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd))

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	result, err := gatherHooksStatusInfo()
	if err != nil {
		printer.Error(err)
		return err
	}

	if printer.IsJSON() {
		return printer.WriteJSON(result)
	}

	printHumanHooksStatus(printer, result)
	return nil
}

// gatherHooksStatusInfo collects comprehensive hook status information.
func gatherHooksStatusInfo() (*hooksStatusResult, error) {
	env, err := setup.ClassifyHookEnv()
	if err != nil {
		return nil, err
	}

	result := &hooksStatusResult{
		Environment: hooksStatusEnv{
			Tier:     tierString(env.Tier),
			HooksDir: env.HooksDir,
			Owner:    env.Owner,
		},
	}

	// Check each hook type.
	hookTypes := []struct {
		name string
		info *hooksStatusHookInfo
	}{
		{"pre-commit", &result.Hooks.PreCommit},
		{"post-commit", &result.Hooks.PostCommit},
		{"post-rewrite", &result.Hooks.PostRewrite},
	}

	for _, ht := range hookTypes {
		hookPath := filepath.Join(env.HooksDir, ht.name)
		if setup.HasTimbersSection(hookPath) {
			ht.info.Installed = true
			ht.info.Format = detectHookFormat(hookPath)
		}
	}

	// Check Claude Code steering.
	result.Steering.ClaudeCode = len(setup.DetectedAgentEnvs()) > 0

	return result, nil
}

// tierString returns a human-readable string for a HookEnvTier.
func tierString(tier setup.HookEnvTier) string {
	switch tier {
	case setup.HookEnvUncontested:
		return "uncontested"
	case setup.HookEnvExistingHook:
		return "existing_hook"
	case setup.HookEnvKnownOverride:
		return "known_override"
	case setup.HookEnvUnknownOverride:
		return "unknown_override"
	default:
		return "unknown"
	}
}

// tierDescription returns a descriptive label for a HookEnvTier with optional owner.
func tierDescription(tier setup.HookEnvTier, owner string) string {
	switch tier {
	case setup.HookEnvUncontested:
		return "Uncontested (no existing hooks)"
	case setup.HookEnvExistingHook:
		return "Existing hook (standard path)"
	case setup.HookEnvKnownOverride:
		if owner != "" {
			return "Known override (" + owner + ")"
		}
		return "Known override"
	case setup.HookEnvUnknownOverride:
		return "Unknown override"
	default:
		return "Unknown"
	}
}

// detectHookFormat returns "section" for new delimited format, "legacy" for old format.
func detectHookFormat(hookPath string) string {
	data, err := os.ReadFile(hookPath)
	if err != nil {
		return "unknown"
	}
	content := string(data)
	if strings.Contains(content, "# --- timbers section (do not edit) ---") {
		return "section"
	}
	return "legacy"
}

// printHumanHooksStatus outputs hooks status in human-readable format.
func printHumanHooksStatus(printer *output.Printer, result *hooksStatusResult) {
	printer.Section("Hook Environment")
	printer.KeyValue("  Tier", tierDescription(tierFromString(result.Environment.Tier), result.Environment.Owner))
	printer.KeyValue("  Hooks dir", result.Environment.HooksDir)
	if result.Environment.Owner != "" {
		printer.KeyValue("  Owner", result.Environment.Owner)
	}

	printer.Section("Hook Integration")
	printHookLine(printer, "  Pre-commit", result.Hooks.PreCommit)
	printHookLine(printer, "  Post-commit", result.Hooks.PostCommit)
	printHookLine(printer, "  Post-rewrite", result.Hooks.PostRewrite)

	printer.Section("Steering")
	if result.Steering.ClaudeCode {
		printer.KeyValue("  Claude Code", "active (Stop hook configured)")
	} else {
		printer.KeyValue("  Claude Code", "not configured")
	}
}

// printHookLine prints a single hook status line.
func printHookLine(printer *output.Printer, label string, info hooksStatusHookInfo) {
	if info.Installed {
		detail := "timbers " + info.Format + " present"
		printer.KeyValue(label, detail)
	} else {
		printer.KeyValue(label, "not installed")
	}
}

// tierFromString converts a tier string back to a HookEnvTier.
func tierFromString(s string) setup.HookEnvTier {
	switch s {
	case "uncontested":
		return setup.HookEnvUncontested
	case "existing_hook":
		return setup.HookEnvExistingHook
	case "known_override":
		return setup.HookEnvKnownOverride
	case "unknown_override":
		return setup.HookEnvUnknownOverride
	default:
		return setup.HookEnvUncontested
	}
}
