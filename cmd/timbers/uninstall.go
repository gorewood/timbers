// Package main provides the entry point for the timbers CLI.
//
//nolint:revive // file length unavoidable for comprehensive uninstall
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
)

type uninstallResult struct {
	BinaryPath, ClaudeScope, ClaudeHookPath          string
	RepoName, PreCommitHookPath, PreCommitBackupPath string
	ConfigsRemoved                                   []string
	NotesEntryCount                                  int
	BinaryRemoved, NotesRefRemoved, HooksRemoved     bool
	HooksRestored, ClaudeRemoved, InRepo             bool
	HooksInstalled, ClaudeInstalled, HooksHasBackup  bool
}

func newUninstallCmd() *cobra.Command {
	var dryRun, force, removeBinary, keepNotes bool
	cmd := &cobra.Command{
		Use: "uninstall", Short: "Remove timbers from the current repository",
		Long: `Remove timbers components: notes refs, config, hooks, Claude integration.
Use --keep-notes to preserve ledger data. Use --binary to remove the binary.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUninstall(cmd, dryRun, force, removeBinary, keepNotes)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompts")
	cmd.Flags().BoolVar(&removeBinary, "binary", false, "Also remove the binary")
	cmd.Flags().BoolVar(&keepNotes, "keep-notes", false, "Keep ledger data")
	return cmd
}

func runUninstall(cmd *cobra.Command, dryRun, force, removeBinary, keepNotes bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))
	result, err := gatherUninstallInfo(removeBinary)
	if err != nil {
		printer.Error(err)
		return err
	}
	if dryRun {
		return outputDryRunUninstall(printer, result, removeBinary, keepNotes)
	}
	if !force && !printer.IsJSON() && !confirmUninstall(cmd, result, removeBinary, keepNotes) {
		printer.Println("Uninstall cancelled.")
		return nil
	}
	return reportUninstallResult(printer, result, removeBinary, keepNotes, doUninstall(result, removeBinary, keepNotes))
}

//nolint:nestif // gather function collects multiple info sources
func gatherUninstallInfo(includeBinary bool) (*uninstallResult, error) {
	result := &uninstallResult{ConfigsRemoved: []string{}}
	if includeBinary {
		execPath, err := os.Executable()
		if err != nil {
			return nil, output.NewSystemErrorWithCause("failed to determine binary location", err)
		}
		result.BinaryPath = execPath
	}
	result.InRepo = git.IsRepo()
	if result.InRepo {
		if root, err := git.RepoRoot(); err == nil {
			result.RepoName = filepath.Base(root)
		}
		if git.NotesRefExists() {
			result.NotesRefRemoved = true
			if commits, err := git.ListNotedCommits(); err == nil {
				result.NotesEntryCount = len(commits)
			}
		}
		result.ConfigsRemoved = findNotesConfigs()
		if hooksDir, err := getHooksDir(); err == nil {
			p := filepath.Join(hooksDir, "pre-commit")
			result.HooksInstalled, result.HooksHasBackup = checkHookStatus(p).Installed, hookExists(p+".backup")
			result.PreCommitHookPath, result.PreCommitBackupPath = p, p+".backup"
		}
	}
	globalPath, _, _ := resolveClaudeHookPath(false)
	projectPath, _, _ := resolveClaudeHookPath(true)
	if isTimbersSectionInstalled(projectPath) {
		result.ClaudeInstalled, result.ClaudeScope, result.ClaudeHookPath = true, "project", projectPath
	} else if isTimbersSectionInstalled(globalPath) {
		result.ClaudeInstalled, result.ClaudeScope, result.ClaudeHookPath = true, "global", globalPath
	}
	return result, nil
}

func findNotesConfigs() []string {
	var configs []string
	remotesOut, err := git.Run("remote")
	if err != nil {
		return configs
	}
	for remote := range strings.SplitSeq(strings.TrimSpace(remotesOut), "\n") {
		if remote = strings.TrimSpace(remote); remote != "" && git.NotesConfigured(remote) {
			configs = append(configs, remote)
		}
	}
	return configs
}

func outputDryRunUninstall(printer *output.Printer, result *uninstallResult, binary, keep bool) error {
	if printer.IsJSON() {
		data := map[string]any{
			"status": "dry_run", "in_repo": result.InRepo, "keep_notes": keep,
			"notes_ref_exists": result.NotesRefRemoved && !keep, "notes_entry_count": result.NotesEntryCount,
			"configs_to_remove": result.ConfigsRemoved, "hooks_installed": result.HooksInstalled,
			"claude_installed": result.ClaudeInstalled,
		}
		if result.ClaudeInstalled {
			data["claude_scope"] = result.ClaudeScope
		}
		if binary {
			data["binary_path"] = result.BinaryPath
		}
		if result.RepoName != "" {
			data["repo_name"] = result.RepoName
		}
		return printer.Success(data)
	}
	styles := uninstallStyles(printer.IsTTY())
	msg := "Dry run: Would perform the following actions:"
	if result.RepoName != "" {
		msg = "Dry run: Would remove timbers from " + result.RepoName
	}
	printer.Println(styles.warning.Render(msg))
	printer.Println()
	if !result.InRepo {
		printer.Println(styles.dim.Render("  (Not in a git repository)"))
	} else {
		printComponents(printer, styles, result, keep, binary, "  ")
	}
	if !hasAnyComponents(result, binary) {
		printer.Println(styles.dim.Render("  (No timbers components found)"))
	}
	return nil
}

func hasAnyComponents(result *uninstallResult, binary bool) bool {
	return result.NotesRefRemoved || len(result.ConfigsRemoved) > 0 || result.HooksInstalled || result.ClaudeInstalled || binary
}

func formatEntryCount(count int) string {
	if count == 1 {
		return "1 entry"
	}
	return fmt.Sprintf("%d entries", count)
}

func printComponents(printer *output.Printer, styles uninstallStyleSet, result *uninstallResult, keep, binary bool, indent string) {
	if result.NotesRefRemoved {
		entry := formatEntryCount(result.NotesEntryCount)
		if keep {
			printer.Println(styles.dim.Render(indent + "• Git notes: " + entry + " (keeping)"))
		} else {
			printer.Println(styles.bullet.Render(indent+"• ") + "Git notes: " + entry)
		}
	}
	for _, remote := range result.ConfigsRemoved {
		if keep {
			printer.Println(styles.dim.Render(indent + "• Notes config for " + remote + " (keeping)"))
		} else {
			printer.Println(styles.bullet.Render(indent+"• ") + "Notes config for remote: " + remote)
		}
	}
	if result.HooksInstalled {
		printer.Println(styles.bullet.Render(indent+"• ") + "Git hooks: pre-commit")
	}
	if result.ClaudeInstalled {
		printer.Println(styles.bullet.Render(indent+"• ") + "Claude integration: " + result.ClaudeScope)
	}
	if binary {
		printer.Println(styles.bullet.Render(indent+"• ") + "Binary: " + result.BinaryPath)
	}
}

func confirmUninstall(cmd *cobra.Command, result *uninstallResult, binary, keep bool) bool {
	printer := output.NewPrinter(cmd.OutOrStdout(), false, output.IsTTY(cmd.OutOrStdout()))
	styles := uninstallStyles(printer.IsTTY())
	if result.RepoName != "" {
		printer.Println(styles.warning.Render("Removing timbers from " + result.RepoName + "..."))
	}
	printer.Println()
	printer.Println("  Components found:")
	if !hasAnyComponents(result, binary) {
		printer.Println(styles.dim.Render("    (No components found)"))
		return false
	}
	if result.InRepo {
		printComponents(printer, styles, result, keep, binary, "    ")
	}
	printer.Println()
	printer.Print("%s", "  ? Remove all components? [y/N] ")
	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

//nolint:gocognit,revive,cyclop,nestif // orchestration function has inherent complexity
func doUninstall(result *uninstallResult, binary, keep bool) []string {
	var errs []string
	if result.ClaudeInstalled {
		if err := removeTimbersSectionFromHook(result.ClaudeHookPath); err != nil {
			errs = append(errs, "claude: "+err.Error())
		} else {
			result.ClaudeRemoved = true
		}
	}
	if result.InRepo && result.HooksInstalled {
		if err := os.Remove(result.PreCommitHookPath); err != nil && !os.IsNotExist(err) {
			errs = append(errs, "hooks: "+err.Error())
		} else {
			result.HooksRemoved = true
			if result.HooksHasBackup {
				if err := os.Rename(result.PreCommitBackupPath, result.PreCommitHookPath); err != nil {
					errs = append(errs, "hooks restore: "+err.Error())
				} else {
					result.HooksRestored = true
				}
			}
		}
	}
	if keep {
		result.NotesRefRemoved, result.ConfigsRemoved = false, []string{}
		return errs
	}
	if result.InRepo && result.NotesRefRemoved {
		if _, err := git.Run("update-ref", "-d", "refs/notes/timbers"); err != nil {
			result.NotesRefRemoved = false
			errs = append(errs, "notes ref: "+err.Error())
		}
	}
	removed := make([]string, 0, len(result.ConfigsRemoved))
	for _, remote := range result.ConfigsRemoved {
		if err := removeNotesConfig(remote); err != nil {
			errs = append(errs, "config for "+remote+": "+err.Error())
		} else {
			removed = append(removed, remote)
		}
	}
	result.ConfigsRemoved = removed
	if binary {
		if err := os.Remove(result.BinaryPath); err != nil {
			errs = append(errs, "binary: "+err.Error())
		} else {
			result.BinaryRemoved = true
		}
	}
	return errs
}

func removeNotesConfig(remote string) error {
	configKey := "remote." + remote + ".fetch"
	out, err := git.Run("config", "--get-all", configKey)
	if err != nil {
		return nil //nolint:nilerr // expected when no config exists
	}
	for line := range strings.SplitSeq(out, "\n") {
		if line = strings.TrimSpace(line); strings.Contains(line, "refs/notes/timbers") {
			if _, err := git.Run("config", "--unset", configKey, line); err != nil {
				return err
			}
		}
	}
	return nil
}

//nolint:gocognit,cyclop // output function with multiple conditional sections
func reportUninstallResult(printer *output.Printer, result *uninstallResult, binary, keep bool, errs []string) error {
	if printer.IsJSON() {
		status := "ok"
		if len(errs) > 0 {
			status = "partial"
		}
		data := map[string]any{
			"status": status, "claude_removed": result.ClaudeRemoved, "hooks_removed": result.HooksRemoved,
			"hooks_restored": result.HooksRestored, "notes_removed": result.NotesRefRemoved && !keep,
			"configs_removed": result.ConfigsRemoved, "keep_notes": keep,
		}
		if result.ClaudeRemoved && result.ClaudeScope != "" {
			data["claude_scope"] = result.ClaudeScope
		}
		if len(errs) > 0 {
			data["errors"], data["recovery_hint"] = errs, "Check permissions and try again."
		}
		if binary {
			data["binary_removed"] = result.BinaryRemoved
		}
		return printer.Success(data)
	}
	styles := uninstallStyles(printer.IsTTY())
	printer.Println()
	if result.ClaudeRemoved {
		printer.Println(styles.success.Render("  ✓ ") + "Claude integration removed")
	}
	if result.HooksRemoved {
		msg := "Git hooks removed"
		if result.HooksRestored {
			msg += " (original restored)"
		}
		printer.Println(styles.success.Render("  ✓ ") + msg)
	}
	if result.NotesRefRemoved && !keep {
		printer.Println(styles.success.Render("  ✓ ") + "Git notes refs removed")
	}
	if len(result.ConfigsRemoved) > 0 && !keep {
		printer.Println(styles.success.Render("  ✓ ") + "Git config cleaned")
	}
	if binary && result.BinaryRemoved {
		printer.Println(styles.success.Render("  ✓ ") + "Binary removed")
	}
	printer.Println()
	if len(errs) > 0 {
		printer.Println(styles.warning.Render("Completed with errors: " + strings.Join(errs, "; ")))
		return nil
	}
	msg := "Timbers removed. Your git history is unchanged."
	if keep {
		msg = "Timbers tooling removed. Ledger data preserved."
	}
	printer.Println(styles.dim.Render("  " + msg))
	return nil
}

type uninstallStyleSet struct{ warning, success, dim, bullet lipgloss.Style }

func uninstallStyles(isTTY bool) uninstallStyleSet {
	if !isTTY {
		return uninstallStyleSet{}
	}
	return uninstallStyleSet{
		warning: lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		success: lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		dim:     lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		bullet:  lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
	}
}
