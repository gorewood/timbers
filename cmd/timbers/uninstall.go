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
	var dryRun, force, removeBinary, keepData bool
	cmd := &cobra.Command{
		Use: "uninstall", Short: "Remove timbers from the current repository",
		Long: `Remove timbers components: .timbers/ directory, hooks, Claude integration.
Use --keep-data to preserve ledger data. Use --binary to remove the binary.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUninstall(cmd, dryRun, force, removeBinary, keepData)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompts")
	cmd.Flags().BoolVar(&removeBinary, "binary", false, "Also remove the binary")
	cmd.Flags().BoolVar(&keepData, "keep-data", false, "Keep ledger data")
	cmd.Flags().Bool("keep-notes", false, "Alias for --keep-data (deprecated)")
	_ = cmd.Flags().MarkHidden("keep-notes")
	return cmd
}

func runUninstall(cmd *cobra.Command, dryRun, force, removeBinary, keepData bool) error {
	// Support deprecated --keep-notes as alias
	if kn, _ := cmd.Flags().GetBool("keep-notes"); kn {
		keepData = true
	}
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))
	info, err := gatherUninstallInfo(removeBinary)
	if err != nil {
		printer.Error(err)
		return err
	}
	if dryRun {
		return outputDryRunUninstall(printer, info, removeBinary, keepData)
	}
	if !force && !printer.IsJSON() && !confirmUninstall(cmd, info, removeBinary, keepData) {
		printer.Println("Uninstall cancelled.")
		return nil
	}
	errs := doUninstall(info, removeBinary, keepData)
	return reportUninstallResult(printer, info, removeBinary, keepData, errs)
}

func gatherUninstallInfo(includeBinary bool) (*setup.UninstallInfo, error) {
	info := &setup.UninstallInfo{}
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
		"status": "dry_run", "in_repo": info.InRepo, "keep_data": keep,
		"timbers_dir_exists": info.TimbersDirExists, "entry_count": info.EntryCount,
		"hooks_installed": info.HooksInstalled, "claude_installed": info.ClaudeInstalled,
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
	return info.TimbersDirExists || info.HooksInstalled || info.ClaudeInstalled || binary
}

func formatEntryCount(count int) string {
	if count == 1 {
		return "1 entry"
	}
	return fmt.Sprintf("%d entries", count)
}

func printComponents(printer *output.Printer, styles uninstallStyleSet, info *setup.UninstallInfo, keep, binary bool, indent string) {
	if info.TimbersDirExists {
		entry := formatEntryCount(info.EntryCount)
		if keep {
			printer.Println(styles.dim.Render(indent + "• .timbers/ directory: " + entry + " (keeping)"))
		} else {
			printer.Println(styles.bullet.Render(indent+"• ") + ".timbers/ directory: " + entry)
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
	if !keep {
		errs = uninstallTimbersDir(info, errs)
	}
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

func uninstallTimbersDir(info *setup.UninstallInfo, errs []string) []string {
	if !info.InRepo || !info.TimbersDirExists {
		return errs
	}
	if err := setup.RemoveTimbersDirContents(info.TimbersDirPath); err != nil {
		return append(errs, ".timbers/: "+err.Error())
	}
	info.TimbersDirRemoved = true
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
		"hooks_restored": info.HooksRestored, "timbers_dir_removed": info.TimbersDirRemoved && !keep,
		"keep_data": keep,
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
		printer.Println(styles.success.Render("  ok ") + "Claude integration removed")
	}
	if info.HooksRemoved {
		msg := "Git hooks removed"
		if info.HooksRestored {
			msg += " (original restored)"
		}
		printer.Println(styles.success.Render("  ok ") + msg)
	}
	if info.TimbersDirRemoved && !keep {
		printer.Println(styles.success.Render("  ok ") + ".timbers/ entries removed")
	}
	if binary && info.BinaryRemoved {
		printer.Println(styles.success.Render("  ok ") + "Binary removed")
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
