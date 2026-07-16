// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"

	"github.com/gorewood/timbers/internal/output"
)

// outputPrimeCompactHuman outputs compact session context for agent injection.
func outputPrimeCompactHuman(printer *output.Printer, result *primeResult) {
	printer.Print("Timbers Prime: %s\n", primeCompactMode)
	printer.Print("Repo: %s | Branch: %s\n", result.Repo, result.Branch)
	printer.Print("Ledger: %d entries | Pending: %s\n", result.EntryCount, compactPendingStatus(result))
	printer.Println()

	outputPrimeCompactRecent(printer, result.RecentEntries)
	outputPrimeCompactState(printer, result)
	outputPrimeCompactHealth(printer, result.Health)

	printer.Println("Rules:")
	printer.Println(`- After each git commit: timbers log "what" --why "why" --how "how"`)
	printer.Println("- Order: commit → timbers log → push (never push before logging — it strands the entry)")
	printer.Println("- Before handoff: timbers pending must be 0")
	printer.Println("- Contributor attribution is automatic; usually omit --who.")
	printer.Println(`- Pairing/shared/correction: --who "Name <email>" is repeatable and replaces the automatic set.`)
	printer.Println("- Only provide contributor identities intended for repository publication.")
	printer.Println("- Do not log secrets, customer data, private URLs, or credentials.")
	printer.Println()
	printer.Println("Commands:")
	printer.Println("- timbers pending")
	printer.Println(`- timbers log "..." --why "..." --how "..."`)
	printer.Println("- timbers query --last 5")
	printer.Println("- timbers draft pr-description --range <base>..HEAD")
	if result.CustomWorkflow {
		printer.Println()
		printer.Println("Custom workflow: .timbers/PRIME.md present — run timbers prime --full to view.")
	}
}

func outputPrimeCompactRecent(printer *output.Printer, entries []primeEntry) {
	printer.Println("Recent:")
	if len(entries) == 0 {
		printer.Println("- none")
		printer.Println()
		return
	}
	for _, entry := range entries {
		printer.Print("- %s %s\n", entry.ID, truncateText(entry.What, 96))
		if entry.Why != "" {
			printer.Print("  Why: %s\n", truncateText(entry.Why, 96))
		}
		if entry.How != "" {
			printer.Print("  How: %s\n", truncateText(entry.How, 96))
		}
		if entry.Notes != "" {
			printer.Print("  Notes: %s\n", truncateText(entry.Notes, 96))
		}
	}
	printer.Println()
}

func outputPrimeCompactState(printer *output.Printer, result *primeResult) {
	if result.StaleAnchor {
		printer.Println("State:")
		printer.Println("- Stale anchor: likely squash merge or rebase.")
		printer.Println("- No action needed; do not re-document old commits.")
		printer.Println("- Anchor self-heals on the next timbers log.")
		printer.Println()
		return
	}
	if result.Pending.Count == 0 {
		return
	}

	printer.Println("Pending commits:")
	limit := min(result.Pending.Count, 5)
	for _, commit := range result.Pending.Commits[:limit] {
		printer.Print("- %s %s\n", commit.Short, truncateText(commit.Subject, 90))
	}
	if result.Pending.Count > limit {
		printer.Print("- ... %d more; run timbers pending\n", result.Pending.Count-limit)
	}
	printer.Println(`Next: timbers log "what" --why "why" --how "how"`)
	printer.Println()
}

func outputPrimeCompactHealth(printer *output.Printer, health []primeHealthItem) {
	if len(health) == 0 {
		return
	}
	printer.Println("Health:")
	limit := min(len(health), 3)
	for _, item := range health[:limit] {
		printer.Print("- %s\n", truncateText(item.Message, 96))
	}
	if len(health) > limit {
		printer.Print("- ... %d more; run timbers doctor\n", len(health)-limit)
	}
	printer.Println("Fix: timbers doctor --fix")
	printer.Println()
}

func outputPrimeUnavailable(printer *output.Printer, status string, command string) error {
	if printer.IsJSON() {
		return printer.WriteJSON(map[string]any{
			"mode":    primeCompactMode,
			"status":  status,
			"command": command,
		})
	}
	printer.Print("Timbers Prime: %s\n", primeCompactMode)
	printer.Print("Timbers: %s. Run: %s\n", status, command)
	return nil
}

func compactPendingStatus(result *primeResult) string {
	if result.StaleAnchor {
		return "0 actionable (stale anchor)"
	}
	if result.Pending.Count == 0 {
		return "0 (clear)"
	}
	if result.Pending.Count == 1 {
		return "1 (action required)"
	}
	return fmt.Sprintf("%d (action required)", result.Pending.Count)
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}
