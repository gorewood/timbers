package main

import (
	"github.com/gorewood/timbers/internal/draft"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

func reportMetadata(
	profileName string, tmpl *draft.Template, entries []*ledger.Entry, flags draftFlags,
	resolved, unresolved int, model string,
) generationMetadata {
	metadata := buildGenerationMetadata(profileName, tmpl, entries, model, draftSelectionFlags{
		last: flags.last, since: flags.since, until: flags.until, rng: flags.rng,
	})
	metadata.Profile = profileName
	metadata.TemplateSource = tmpl.Source
	metadata.Projection = tmpl.Report.Projection
	metadata.Format = tmpl.Report.Format
	metadata.GitResolved = resolved
	metadata.GitUnresolved = unresolved
	if model == "" {
		metadata.Timestamp = ""
	}
	return metadata
}

func outputQuietReport(
	printer *output.Printer, profileName, reason string, metadata generationMetadata,
) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status": "quiet", "profile": profileName, "reason": reason,
			"entry_count": len(metadata.EntryIDs), "provenance": metadata,
		})
	}
	if printer.IsTTY() {
		printer.Stderr("No report content for %s.\n", profileName)
	}
	return nil
}

func outputRenderedReport(
	printer *output.Printer, profileName string, tmpl *draft.Template, rendered string,
	entries []*ledger.Entry, metadata generationMetadata,
) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status": "rendered", "profile": profileName,
			"template_path": tmpl.Source, "prompt": rendered,
			"entry_count": len(entries), "provenance": metadata,
		})
	}
	if !printer.IsTTY() {
		printer.Stderr("timbers: rendered report %q with %d entries\n", profileName, len(entries))
	}
	printer.Print("%s\n", rendered)
	return nil
}

func outputGeneratedReport(
	printer *output.Printer, profileName string, tmpl *draft.Template, rendered, content string,
	entries []*ledger.Entry, metadata generationMetadata, withFrontmatter bool,
) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status": "generated", "profile": profileName,
			"template_path": tmpl.Source, "prompt": rendered,
			"entry_count": len(entries), "model": metadata.Model,
			"response": content, "provenance": metadata,
		})
	}
	if withFrontmatter {
		printer.Print("%s\n", formatTOMLFrontmatter(metadata))
	}
	printer.Print("%s\n", content)
	return nil
}
