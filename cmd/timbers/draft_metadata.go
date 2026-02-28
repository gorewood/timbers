package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gorewood/timbers/internal/draft"
	"github.com/gorewood/timbers/internal/ledger"
)

// draftFlags holds all draft command flags.
type draftFlags struct {
	last            string
	since           string
	until           string
	rng             string // "range" is a keyword
	appendText      string
	list            bool
	show            bool
	models          bool
	model           string
	provider        string
	withFrontmatter bool
}

// draftSelectionFlags holds the entry selection flags for metadata.
type draftSelectionFlags struct {
	last  string
	since string
	until string
	rng   string // "range" is a keyword
}

// generationMetadata holds information about how content was generated.
type generationMetadata struct {
	Template        string   `json:"template"`
	TemplateVersion int      `json:"template_version"`
	Model           string   `json:"model"`
	Timestamp       string   `json:"timestamp"`
	EntryIDs        []string `json:"entry_ids"`
	Selection       string   `json:"selection"`
}

// buildGenerationMetadata creates metadata about the generation.
func buildGenerationMetadata(
	templateName string, tmpl *draft.Template,
	entries []*ledger.Entry, model string,
	selFlags draftSelectionFlags,
) generationMetadata {
	// Collect entry IDs
	entryIDs := make([]string, len(entries))
	for idx, entry := range entries {
		entryIDs[idx] = entry.ID
	}

	// Build selection string
	var selParts []string
	if selFlags.last != "" {
		selParts = append(selParts, "--last "+selFlags.last)
	}
	if selFlags.since != "" {
		selParts = append(selParts, "--since "+selFlags.since)
	}
	if selFlags.until != "" {
		selParts = append(selParts, "--until "+selFlags.until)
	}
	if selFlags.rng != "" {
		selParts = append(selParts, "--range "+selFlags.rng)
	}
	selection := strings.Join(selParts, " ")

	return generationMetadata{
		Template:        templateName,
		TemplateVersion: tmpl.Version,
		Model:           model,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		EntryIDs:        entryIDs,
		Selection:       selection,
	}
}

// formatTOMLFrontmatter formats metadata as TOML frontmatter.
func formatTOMLFrontmatter(meta generationMetadata) string {
	var builder strings.Builder
	builder.WriteString("+++\n")
	builder.WriteString(fmt.Sprintf("generated_template = \"%s\"\n", meta.Template))
	builder.WriteString(fmt.Sprintf("generated_template_version = %d\n", meta.TemplateVersion))
	builder.WriteString(fmt.Sprintf("generated_model = \"%s\"\n", meta.Model))
	builder.WriteString(fmt.Sprintf("generated_at = \"%s\"\n", meta.Timestamp))
	builder.WriteString(fmt.Sprintf("generated_selection = \"%s\"\n", meta.Selection))
	builder.WriteString(fmt.Sprintf("generated_entry_count = %d\n", len(meta.EntryIDs)))
	builder.WriteString("+++\n")
	return builder.String()
}
