// Package export provides formatting and file output for ledger entries.
//
// This package handles exporting timbers ledger entries to various formats
// for use in documentation, reporting, and integration with other tools.
//
// # Supported Formats
//
// The package supports two output formats:
//
//   - JSON: Machine-readable format preserving the full entry schema
//   - Markdown: Human-readable format with YAML frontmatter
//
// # JSON Export
//
// JSON export preserves the complete entry structure:
//
//	export.FormatJSON(printer, entries)           // Write to printer
//	export.WriteJSONFiles(entries, "/path/to/dir") // Write individual files
//
// Each entry is written with the full timbers.devlog/v1 schema, suitable
// for consumption by other tools or data pipelines.
//
// # Markdown Export
//
// Markdown export creates documentation-ready files:
//
//	markdown := export.FormatMarkdown(entry)       // Get markdown string
//	export.WriteMarkdownFiles(entries, "/path/to") // Write individual files
//
// The markdown format includes:
//   - YAML frontmatter with schema, id, date, anchor commit, and tags
//   - Title from the "what" summary
//   - What/Why/How sections
//   - Evidence section with commit count and diffstat
//
// Example markdown output:
//
//	---
//	schema: timbers.export/v1
//	id: tb_2026-01-15T15:04:05Z_8f2c1a
//	date: 2026-01-15
//	anchor_commit: 8f2c1a9d1234
//	commit_count: 3
//	tags: [feature, auth]
//	---
//
//	# Added user authentication
//
//	**What:** Added user authentication
//	**Why:** Security requirement for protected routes
//	**How:** JWT-based auth with refresh tokens
//
//	## Evidence
//
//	- Commits: 3 (abc1234..def5678)
//	- Files changed: 8 (+245/-12)
//
// # File Naming
//
// When writing to files, entries are named by their ID:
//   - JSON: <entry-id>.json
//   - Markdown: <entry-id>.md
package export
