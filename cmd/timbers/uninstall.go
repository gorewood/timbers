package main

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

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
	info, err := gatherUninstallInfo(removeBinary)
	if err != nil {
		printer.Error(err)
		return err
	}
	if dryRun {
		return outputDryRunUninstall(printer, info, removeBinary, keepNotes)
	}
	if !force && !printer.IsJSON() && !confirmUninstall(cmd, info, removeBinary, keepNotes) {
		printer.Println("Uninstall cancelled.")
		return nil
	}
	errs := doUninstall(info, removeBinary, keepNotes)
	return reportUninstallResult(printer, info, removeBinary, keepNotes, errs)
}

func gatherUninstallInfo(includeBinary bool) (*setup.UninstallInfo, error) {
	info := &setup.UninstallInfo{ConfigsRemoved: []string{}}
	if includeBinary {
		path, err := setup.GatherBinaryPath()
		if err != nil {
			return nil, err
		}
		info.BinaryPath = path
	}
	info.InRepo = git.IsRepo()
	if info.InRepo {
		setup.GatherRepoInfo(info)
		setup.GatherHookInfo(info)
	}
	setup.GatherClaudeInfo(info)
	return info, nil
}

func outputDryRunUninstall(printer *output.Printer, info *setup.UninstallInfo, binary, keep bool) error {
	if printer.IsJSON() {
		return outputDryRunJSON(printer, info, binary, keep)
	}
	return outputDryRunHuman(printer, info, binary, keep)
}

func outputDryRunJSON(printer *output.Printer, info *setup.UninstallInfo, binary, keep bool) error {
	data := map[string]any{
		"status": "dry_run", "in_repo": info.InRepo, "keep_notes": keep,
		"notes_ref_exists": info.NotesRefRemoved && !keep, "notes_entry_count": info.NotesEntryCount,
		"configs_to_remove": info.ConfigsRemoved, "hooks_installed": info.HooksInstalled,
		"claude_installed": info.ClaudeInstalled,
	}
	if info.ClaudeInstalled {
		data["claude_scope"] = info.ClaudeScope
	}
	if binary {
		data["binary_path"] = info.BinaryPath
	}
	if info.RepoName != "" {
		data["repo_name"] = info.RepoName
	}
	return printer.Success(data)
}

func outputDryRunHuman(printer *output.Printer, info *setup.UninstallInfo, binary, keep bool) error {
	styles := uninstallStyles(printer.IsTTY())
	msg := "Dry run: Would perform the following actions:"
	if info.RepoName != "" {
		msg = "Dry run: Would remove timbers from " + info.RepoName
	}
	printer.Println(styles.warning.Render(msg))
	printer.Println()
	if !info.InRepo {
		printer.Println(styles.dim.Render("  (Not in a git repository)"))
	} else {
		printComponents(printer, styles, info, keep, binary, "  ")
	}
	if !hasAnyComponents(info, binary) {
		printer.Println(styles.dim.Render("  (No timbers components found)"))
	}
	return nil
}

func hasAnyComponents(info *setup.UninstallInfo, binary bool) bool {
	return info.NotesRefRemoved || len(info.ConfigsRemoved) > 0 || info.HooksInstalled || info.ClaudeInstalled || binary
}

func formatEntryCount(count int) string {
	if count == 1 {
		return "1 entry"
	}
	return fmt.Sprintf("%d entries", count)
}

func printComponents(printer *output.Printer, styles uninstallStyleSet, info *setup.UninstallInfo, keep, binary bool, indent string) {
	if info.NotesRefRemoved {
		entry := formatEntryCount(info.NotesEntryCount)
		if keep {
			printer.Println(styles.dim.Render(indent + "• Git notes: " + entry + " (keeping)"))
		} else {
			printer.Println(styles.bullet.Render(indent+"• ") + "Git notes: " + entry)
		}
	}
	for _, remote := range info.ConfigsRemoved {
		if keep {
			printer.Println(styles.dim.Render(indent + "• Notes config for " + remote + " (keeping)"))
		} else {
			printer.Println(styles.bullet.Render(indent+"• ") + "Notes config for remote: " + remote)
		}
	}
	if info.HooksInstalled {
		printer.Println(styles.bullet.Render(indent+"• ") + "Git hooks: pre-commit")
	}
	if info.ClaudeInstalled {
		printer.Println(styles.bullet.Render(indent+"• ") + "Claude integration: " + info.ClaudeScope)
	}
	if binary {
		printer.Println(styles.bullet.Render(indent+"• ") + "Binary: " + info.BinaryPath)
	}
}

func confirmUninstall(cmd *cobra.Command, info *setup.UninstallInfo, binary, keep bool) bool {
	printer := output.NewPrinter(cmd.OutOrStdout(), false, output.IsTTY(cmd.OutOrStdout()))
	styles := uninstallStyles(printer.IsTTY())
	if info.RepoName != "" {
		printer.Println(styles.warning.Render("Removing timbers from " + info.RepoName + "..."))
	}
	printer.Println()
	printer.Println("  Components found:")
	if !hasAnyComponents(info, binary) {
		printer.Println(styles.dim.Render("    (No components found)"))
		return false
	}
	if info.InRepo {
		printComponents(printer, styles, info, keep, binary, "    ")
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

func doUninstall(info *setup.UninstallInfo, binary, keep bool) []string {
	var errs []string
	errs = uninstallClaude(info, errs)
	errs = uninstallHooks(info, errs)
	if keep {
		info.NotesRefRemoved = false
		info.ConfigsRemoved = []string{}
		return errs
	}
	errs = uninstallNotes(info, errs)
	errs = uninstallConfigs(info, errs)
	errs = uninstallBinary(info, binary, errs)
	return errs
}

func uninstallClaude(info *setup.UninstallInfo, errs []string) []string {
	if !info.ClaudeInstalled {
		return errs
	}
	if err := setup.RemoveClaudeIntegration(info.ClaudeHookPath); err != nil {
		return append(errs, "claude: "+err.Error())
	}
	info.ClaudeRemoved = true
	return errs
}

func uninstallHooks(info *setup.UninstallInfo, errs []string) []string {
	if !info.InRepo || !info.HooksInstalled {
		return errs
	}
	removed, restored, err := setup.RemoveGitHook(info.PreCommitHookPath, info.HooksHasBackup, info.PreCommitBackupPath)
	info.HooksRemoved = removed
	info.HooksRestored = restored
	if err != nil {
		errs = append(errs, "hooks: "+err.Error())
	}
	return errs
}

func uninstallNotes(info *setup.UninstallInfo, errs []string) []string {
	if !info.InRepo || !info.NotesRefRemoved {
		return errs
	}
	if err := setup.RemoveNotesRef(); err != nil {
		info.NotesRefRemoved = false
		return append(errs, "notes ref: "+err.Error())
	}
	return errs
}

func uninstallConfigs(info *setup.UninstallInfo, errs []string) []string {
	removed := make([]string, 0, len(info.ConfigsRemoved))
	for _, remote := range info.ConfigsRemoved {
		if err := setup.RemoveNotesConfig(remote); err != nil {
			errs = append(errs, "config for "+remote+": "+err.Error())
		} else {
			removed = append(removed, remote)
		}
	}
	info.ConfigsRemoved = removed
	return errs
}

func uninstallBinary(info *setup.UninstallInfo, binary bool, errs []string) []string {
	if !binary {
		return errs
	}
	if err := setup.RemoveBinary(info.BinaryPath); err != nil {
		return append(errs, "binary: "+err.Error())
	}
	info.BinaryRemoved = true
	return errs
}

func reportUninstallResult(printer *output.Printer, info *setup.UninstallInfo, binary, keep bool, errs []string) error {
	if printer.IsJSON() {
		return reportUninstallJSON(printer, info, binary, keep, errs)
	}
	return reportUninstallHuman(printer, info, binary, keep, errs)
}

func reportUninstallJSON(printer *output.Printer, info *setup.UninstallInfo, binary, keep bool, errs []string) error {
	status := "ok"
	if len(errs) > 0 {
		status = "partial"
	}
	data := map[string]any{
		"status": status, "claude_removed": info.ClaudeRemoved, "hooks_removed": info.HooksRemoved,
		"hooks_restored": info.HooksRestored, "notes_removed": info.NotesRefRemoved && !keep,
		"configs_removed": info.ConfigsRemoved, "keep_notes": keep,
	}
	if info.ClaudeRemoved && info.ClaudeScope != "" {
		data["claude_scope"] = info.ClaudeScope
	}
	if len(errs) > 0 {
		data["errors"], data["recovery_hint"] = errs, "Check permissions and try again."
	}
	if binary {
		data["binary_removed"] = info.BinaryRemoved
	}
	return printer.Success(data)
}

func reportUninstallHuman(printer *output.Printer, info *setup.UninstallInfo, binary, keep bool, errs []string) error {
	styles := uninstallStyles(printer.IsTTY())
	printer.Println()
	printRemovalSummary(printer, styles, info, binary, keep)
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

func printRemovalSummary(printer *output.Printer, styles uninstallStyleSet, info *setup.UninstallInfo, binary, keep bool) {
	if info.ClaudeRemoved {
		printer.Println(styles.success.Render("  ✓ ") + "Claude integration removed")
	}
	if info.HooksRemoved {
		msg := "Git hooks removed"
		if info.HooksRestored {
			msg += " (original restored)"
		}
		printer.Println(styles.success.Render("  ✓ ") + msg)
	}
	if info.NotesRefRemoved && !keep {
		printer.Println(styles.success.Render("  ✓ ") + "Git notes refs removed")
	}
	if len(info.ConfigsRemoved) > 0 && !keep {
		printer.Println(styles.success.Render("  ✓ ") + "Git config cleaned")
	}
	if binary && info.BinaryRemoved {
		printer.Println(styles.success.Render("  ✓ ") + "Binary removed")
	}
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
