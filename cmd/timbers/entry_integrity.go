package main

import (
	"fmt"
	"strings"

	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// corruptEntriesError formats ledger scan diagnostics for CLI recovery.
func corruptEntriesError(stats *ledger.ListStats) error {
	if stats == nil || stats.ParseErrors == 0 {
		return nil
	}
	return output.NewUserError(fmt.Sprintf(
		"ledger contains %d malformed entry file(s): %s; run 'timbers doctor' for details",
		stats.ParseErrors, strings.Join(stats.CorruptFiles, ", "),
	))
}
