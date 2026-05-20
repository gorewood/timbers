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

// newAckCmd creates the ack command.
func newAckCmd() *cobra.Command {
	return newAckCmdInternal(nil)
}

// newAckCmdInternal creates the ack command with optional storage injection.
// If storage is nil, a real storage is created when the command runs.
func newAckCmdInternal(storage *ledger.Storage) *cobra.Command {
	var reason string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "ack <sha>",
		Short: "Record a decision to skip a commit without writing a content entry",
		Long: `Record a "decision to skip" for a commit. The acked SHA counts as documented
for pending detection — a third bypass path alongside infrastructure rules and
revert auto-skip — with an audit trail (date, acker, one-line reason).

Use cases:
  - A teammate's commit landed on main via merge; you reviewed it and it doesn't
    merit a content entry, but you want to clear it from your pending list with
    an explanation rather than fabricating an entry or using --no-verify.
  - A GitHub Action runs on PR merge and acks the merge SHA server-side, so the
    merge self-clears from everyone's pending list without client discipline.
  - A bot-authored commit slips through skip-authors; ack it once with a note.

Examples:
  timbers ack abc1234 --reason "GitHub merge of PR #274; content in entry tb_..."
  timbers ack abc1234 --reason "Upstream sync from main; no design decision here"
  timbers ack abc1234 --reason "..." --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAck(cmd, storage, args[0], reason, dryRun)
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "One-line explanation of why this commit doesn't need an entry (required)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be written without writing")
	_ = cmd.MarkFlagRequired("reason")

	return cmd
}

// runAck executes the ack command.
func runAck(cmd *cobra.Command, storage *ledger.Storage, shaArg, reason string, dryRun bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd))

	if storage == nil && !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	if storage == nil {
		var err error
		storage, err = ledger.NewDefaultStorage()
		if err != nil {
			printer.Error(err)
			return err
		}
	}

	// Resolve the SHA to its full canonical form. This catches typos
	// before we write the ack — a SHA that doesn't exist in the repo
	// shouldn't be acked.
	fullSHA, err := resolveCommitSHA(shaArg)
	if err != nil {
		printer.Error(err)
		return err
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		err := output.NewUserError("--reason must not be empty")
		printer.Error(err)
		return err
	}

	now := time.Now().UTC()
	ack := &ledger.Ack{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindAck,
		ID:        ledger.GenerateAckID(fullSHA, now),
		AckedAt:   now,
		Acker:     resolveAcker(),
		TargetSHA: fullSHA,
		Reason:    reason,
	}

	if dryRun {
		return outputAckDryRun(printer, ack)
	}

	if err := storage.WriteAck(ack); err != nil {
		printer.Error(err)
		return err
	}

	return outputAckSuccess(printer, ack)
}

// resolveCommitSHA expands a short SHA or other revision into the full
// 40-character SHA, validating that it exists in the current repo.
func resolveCommitSHA(rev string) (string, error) {
	rev = strings.TrimSpace(rev)
	if rev == "" {
		return "", output.NewUserError("commit SHA must not be empty")
	}
	out, err := git.Run("rev-parse", "--verify", rev+"^{commit}")
	if err != nil {
		return "", output.NewUserError("not a commit: " + rev)
	}
	return strings.TrimSpace(out), nil
}

// resolveAcker derives the acker identity from git config. Best-effort:
// missing fields fall back to empty strings so the ack still records
// what we know.
func resolveAcker() ledger.Acker {
	name, _ := git.Run("config", "user.name")
	email, _ := git.Run("config", "user.email")
	return ledger.Acker{
		Name:  strings.TrimSpace(name),
		Email: strings.TrimSpace(email),
	}
}

// outputAckDryRun reports what would be written without writing.
func outputAckDryRun(printer *output.Printer, ack *ledger.Ack) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":     "dry_run",
			"ack_id":     ack.ID,
			"target_sha": ack.TargetSHA,
			"reason":     ack.Reason,
			"acker":      ack.Acker,
		})
	}
	printer.Println("Would write ack " + ack.ID)
	printer.KeyValue("Target", ack.TargetSHA)
	printer.KeyValue("Reason", ack.Reason)
	if ack.Acker.Name != "" || ack.Acker.Email != "" {
		printer.KeyValue("Acker", ack.Acker.Name+" <"+ack.Acker.Email+">")
	}
	return nil
}

// outputAckSuccess prints the success summary after the ack is committed.
func outputAckSuccess(printer *output.Printer, ack *ledger.Ack) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":     "ok",
			"ack_id":     ack.ID,
			"target_sha": ack.TargetSHA,
			"reason":     ack.Reason,
		})
	}
	printer.Println("Recorded ack " + ack.ID)
	printer.Println("  " + ack.TargetSHA[:7] + " — " + ack.Reason)
	return nil
}
