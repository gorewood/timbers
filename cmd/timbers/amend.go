// Package main provides the entry point for the timbers CLI.
package main

import (
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// newAmendCmd creates the amend command.
func newAmendCmd() *cobra.Command {
	return newAmendCmdInternal(nil)
}

// amendFlags holds all flag values for the amend command.
type amendFlags struct {
	what   string
	why    string
	how    string
	tags   []string
	dryRun bool
}

// newAmendCmdInternal creates the amend command with optional storage injection.
// If storage is nil, a real storage is created when the command runs.
func newAmendCmdInternal(storage *ledger.Storage) *cobra.Command {
	var flags amendFlags

	cmd := &cobra.Command{
		Use:   "amend <entry-id>",
		Short: "Modify an existing ledger entry",
		Long: `Modify an existing ledger entry's summary fields or tags.

The amend command allows you to update what/why/how fields and tags on existing entries.
Only the fields you specify will be updated; unspecified fields retain their current values.
The updated_at timestamp will be set to the current time when amending.

Examples:
  timbers amend tb_2026-01-15T15:04:05Z_8f2c1a --what "Fixed critical auth bug"
  timbers amend tb_2026-01-15T15:04:05Z_8f2c1a --why "Updated reasoning" --how "Better approach"
  timbers amend tb_2026-01-15T15:04:05Z_8f2c1a --tag security --tag auth
  timbers amend tb_2026-01-15T15:04:05Z_8f2c1a --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAmend(cmd, storage, args[0], flags)
		},
	}

	cmd.Flags().StringVar(&flags.what, "what", "", "Update the 'what' summary field")
	cmd.Flags().StringVar(&flags.why, "why", "", "Update the 'why' summary field")
	cmd.Flags().StringVar(&flags.how, "how", "", "Update the 'how' summary field")
	cmd.Flags().StringSliceVar(&flags.tags, "tag", nil, "Replace tags (repeatable)")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Preview changes without writing")

	return cmd
}

// runAmend executes the amend command.
func runAmend(cmd *cobra.Command, storage *ledger.Storage, entryID string, flags amendFlags) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

	if err := validateAmendFlags(flags, printer); err != nil {
		return err
	}

	storage, err := initAmendStorage(storage, printer)
	if err != nil {
		return err
	}

	entry, err := storage.GetEntryByID(entryID)
	if err != nil {
		printer.Error(err)
		return err
	}

	amended := amendEntry(entry, flags)

	if flags.dryRun {
		return outputAmendDryRun(printer, entry, amended, flags)
	}

	if err := storage.WriteEntry(amended, true); err != nil {
		printer.Error(err)
		return err
	}

	return outputAmendSuccess(printer, amended)
}

// validateAmendFlags checks that at least one field is being updated.
func validateAmendFlags(flags amendFlags, printer *output.Printer) error {
	if flags.what == "" && flags.why == "" && flags.how == "" && len(flags.tags) == 0 {
		err := output.NewUserError("at least one field must be specified for amendment (--what, --why, --how, or --tag)")
		printer.Error(err)
		return err
	}
	return nil
}

// initAmendStorage initializes the storage, checking for git repo if needed.
func initAmendStorage(storage *ledger.Storage, printer *output.Printer) (*ledger.Storage, error) {
	if storage == nil && !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return nil, err
	}

	if storage == nil {
		storage = ledger.NewStorage(nil)
	}
	return storage, nil
}

// amendEntry applies the amendments to the entry and returns the modified entry.
func amendEntry(entry *ledger.Entry, flags amendFlags) *ledger.Entry {
	// Create a copy to avoid modifying the original
	amended := *entry

	// Update summary fields if specified
	if flags.what != "" {
		amended.Summary.What = flags.what
	}
	if flags.why != "" {
		amended.Summary.Why = flags.why
	}
	if flags.how != "" {
		amended.Summary.How = flags.how
	}

	// Replace tags if specified (empty slice means clear tags)
	if flags.tags != nil {
		amended.Tags = flags.tags
	}

	// Update timestamp
	amended.UpdatedAt = time.Now().UTC()

	return &amended
}

// outputAmendDryRun outputs a preview of the changes.
func outputAmendDryRun(printer *output.Printer, original, amended *ledger.Entry, flags amendFlags) error {
	if printer.IsJSON() {
		return printer.WriteJSON(map[string]any{
			"dry_run": true,
			"entry":   amended,
			"changes": buildChangesMap(original, amended, flags),
		})
	}

	printer.Println("Dry run - changes that would be made:")
	printer.Println()
	printer.KeyValue("Entry ID", amended.ID)

	if flags.what != "" {
		printer.Println()
		printer.Section("What")
		printer.Println("  Before: " + original.Summary.What)
		printer.Println("  After:  " + amended.Summary.What)
	}

	if flags.why != "" {
		printer.Println()
		printer.Section("Why")
		printer.Println("  Before: " + original.Summary.Why)
		printer.Println("  After:  " + amended.Summary.Why)
	}

	if flags.how != "" {
		printer.Println()
		printer.Section("How")
		printer.Println("  Before: " + original.Summary.How)
		printer.Println("  After:  " + amended.Summary.How)
	}

	if flags.tags != nil {
		printer.Println()
		printer.Section("Tags")
		printer.Println("  Before: " + formatTags(original.Tags))
		printer.Println("  After:  " + formatTags(amended.Tags))
	}

	return nil
}

// buildChangesMap builds a map of changes for JSON output.
func buildChangesMap(original, amended *ledger.Entry, flags amendFlags) map[string]any {
	changes := make(map[string]any)

	if flags.what != "" {
		changes["what"] = map[string]string{
			"before": original.Summary.What,
			"after":  amended.Summary.What,
		}
	}

	if flags.why != "" {
		changes["why"] = map[string]string{
			"before": original.Summary.Why,
			"after":  amended.Summary.Why,
		}
	}

	if flags.how != "" {
		changes["how"] = map[string]string{
			"before": original.Summary.How,
			"after":  amended.Summary.How,
		}
	}

	if flags.tags != nil {
		changes["tags"] = map[string][]string{
			"before": original.Tags,
			"after":  amended.Tags,
		}
	}

	return changes
}

// formatTags formats a slice of tags as a comma-separated string.
func formatTags(tags []string) string {
	if len(tags) == 0 {
		return "(none)"
	}
	return strings.Join(tags, ", ")
}

// outputAmendSuccess outputs the success message after amending.
func outputAmendSuccess(printer *output.Printer, entry *ledger.Entry) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":     "amended",
			"id":         entry.ID,
			"updated_at": entry.UpdatedAt.Format("2006-01-02 15:04:05 UTC"),
		})
	}

	printer.Println("Entry amended successfully")
	printer.Println()
	printer.KeyValue("Entry ID", entry.ID)
	printer.KeyValue("Updated", entry.UpdatedAt.Format("2006-01-02 15:04:05 UTC"))

	return nil
}
