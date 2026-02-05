// Package export provides formatting and output for ledger entries.
package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// FormatJSON outputs the entries as a JSON array to the printer.
func FormatJSON(printer *output.Printer, entries []*ledger.Entry) error {
	return printer.WriteJSON(entries)
}

// WriteJSONFiles writes each entry as a separate JSON file to the output directory.
// Files are named <entry-id>.json.
func WriteJSONFiles(entries []*ledger.Entry, dir string) error {
	for _, entry := range entries {
		filename := filepath.Join(dir, entry.ID+".json")

		// Marshal entry to JSON
		data, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			return output.NewSystemError(fmt.Sprintf("failed to marshal entry %s: %v", entry.ID, err))
		}

		// Write to file
		if err := os.WriteFile(filename, data, 0600); err != nil {
			return output.NewSystemError(fmt.Sprintf("failed to write file %s: %v", filename, err))
		}
	}

	return nil
}
