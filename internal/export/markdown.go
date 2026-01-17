// Package export provides formatting and output for ledger entries.
package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rbergman/timbers/internal/ledger"
	"github.com/rbergman/timbers/internal/output"
)

// FormatMarkdown formats a single entry as a markdown document.
// Returns the formatted markdown string.
func FormatMarkdown(entry *ledger.Entry) string {
	var builder strings.Builder

	writeFrontmatter(&builder, entry)
	writeSummary(&builder, entry)
	writeEvidence(&builder, entry)

	return builder.String()
}

// writeFrontmatter writes the YAML frontmatter section.
func writeFrontmatter(builder *strings.Builder, entry *ledger.Entry) {
	builder.WriteString("---\n")
	builder.WriteString("schema: timbers.export/v1\n")
	fmt.Fprintf(builder, "id: %s\n", entry.ID)

	// Format date as YYYY-MM-DD
	dateStr := entry.CreatedAt.Format("2006-01-02")
	fmt.Fprintf(builder, "date: %s\n", dateStr)

	// Anchor commit short SHA
	shortSHA := entry.Workset.AnchorCommit
	if len(shortSHA) > 12 {
		shortSHA = shortSHA[:12]
	}
	fmt.Fprintf(builder, "anchor_commit: %s\n", shortSHA)

	fmt.Fprintf(builder, "commit_count: %d\n", len(entry.Workset.Commits))

	// Tags
	if len(entry.Tags) > 0 {
		fmt.Fprintf(builder, "tags: [%s]\n", strings.Join(entry.Tags, ", "))
	}

	builder.WriteString("---\n\n")
}

// writeSummary writes the title and What/Why/How sections.
func writeSummary(builder *strings.Builder, entry *ledger.Entry) {
	fmt.Fprintf(builder, "# %s\n\n", entry.Summary.What)
	fmt.Fprintf(builder, "**What:** %s\n\n", entry.Summary.What)
	fmt.Fprintf(builder, "**Why:** %s\n\n", entry.Summary.Why)
	fmt.Fprintf(builder, "**How:** %s\n\n", entry.Summary.How)
}

// writeEvidence writes the Evidence section with commits and diffstat.
func writeEvidence(builder *strings.Builder, entry *ledger.Entry) {
	builder.WriteString("## Evidence\n\n")

	commitCount := len(entry.Workset.Commits)
	commitRange := computeCommitRange(entry)

	fmt.Fprintf(builder, "- Commits: %d", commitCount)
	if commitRange != "" {
		fmt.Fprintf(builder, " (%s)", commitRange)
	}
	builder.WriteString("\n")

	if entry.Workset.Diffstat != nil {
		fmt.Fprintf(builder, "- Files changed: %d (+%d/-%d)\n",
			entry.Workset.Diffstat.Files,
			entry.Workset.Diffstat.Insertions,
			entry.Workset.Diffstat.Deletions)
	}
}

// computeCommitRange returns the commit range string for the entry.
func computeCommitRange(entry *ledger.Entry) string {
	if entry.Workset.Range != "" {
		return entry.Workset.Range
	}

	if len(entry.Workset.Commits) == 0 {
		return ""
	}

	// Generate range from first to last commit
	first := entry.Workset.Commits[0]
	last := entry.Workset.Commits[len(entry.Workset.Commits)-1]

	// Use short SHAs
	if len(first) > 7 {
		first = first[:7]
	}
	if len(last) > 7 {
		last = last[:7]
	}

	return first + ".." + last
}

// WriteMarkdownFiles writes each entry as a separate markdown file to the output directory.
// Files are named <entry-id>.md.
func WriteMarkdownFiles(entries []*ledger.Entry, dir string) error {
	for _, entry := range entries {
		filename := filepath.Join(dir, entry.ID+".md")

		// Format entry
		content := FormatMarkdown(entry)

		// Write to file
		if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
			return output.NewSystemError(fmt.Sprintf("failed to write file %s: %v", filename, err))
		}
	}

	return nil
}
