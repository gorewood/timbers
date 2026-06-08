package output

import (
	"strings"
)

// defaultPanelWidth is the fallback wrap width used when no terminal width
// is known (e.g. piped output or tests writing to a buffer).
const defaultPanelWidth = 80

// Field is a single labeled row in an entry panel.
// A Field with an empty Key and Value renders as a blank separator line,
// used to visually divide substance (what/why/how) from bookkeeping.
type Field struct {
	Key   string
	Value string
	// Emphasis renders the value with bold styling at a TTY. It is a no-op
	// for piped output, where all styles are neutral.
	Emphasis bool
}

// Separator returns a Field that renders as a blank line within a panel.
func Separator() Field { return Field{} }

// FieldsBox renders fields as an aligned key/value panel inside a Box.
//
// Keys are left-aligned to a common column width; long values wrap under the
// value column with a hanging indent so the eye scans straight down the keys.
// At a TTY the panel gets a rounded border (via Box); piped output is
// borderless and plain so it stays parseable.
//
// Field keys and values must be plain text, not pre-styled strings: wrapping
// measures raw bytes, so embedded ANSI escapes (or wide runes) would throw off
// the wrap points and column alignment. Styling is applied here, after wrapping.
func (p *Printer) FieldsBox(title string, fields []Field) {
	p.Box(title, p.renderFields(fields, p.boxContentWidth()))
}

// boxContentWidth returns the wrap width available for field values,
// accounting for the box border and horizontal padding when at a TTY.
func (p *Printer) boxContentWidth() int {
	width := p.width
	if width <= 0 {
		width = defaultPanelWidth
	}
	if p.isTTY {
		// rounded border (2 cols) + Padding(0, 1) (2 cols)
		width -= 4
	}
	return max(width, 1)
}

// renderFields builds the aligned, wrapped panel body as a single string.
func (p *Printer) renderFields(fields []Field, width int) string {
	keyWidth := maxKeyWidth(fields)
	const gap = "  "
	indent := keyWidth + len(gap)
	valueWidth := max(width-indent, 1)

	var lines []string
	for _, f := range fields {
		if f.Key == "" && f.Value == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, p.renderField(f, keyWidth, gap, indent, valueWidth)...)
	}
	return strings.Join(lines, "\n")
}

// renderField renders one field to one or more lines, wrapping the value and
// hanging-indenting continuation lines to the value column.
func (p *Printer) renderField(field Field, keyWidth int, gap string, indent, valueWidth int) []string {
	wrapped := wrapText(field.Value, valueWidth)

	valueStyle := p.styles.Value
	if field.Emphasis {
		valueStyle = p.styles.Bold
	}

	keyCell := p.styles.Key.Render(padRight(field.Key, keyWidth))
	out := make([]string, 0, len(wrapped))
	for i, line := range wrapped {
		styledVal := valueStyle.Render(line)
		if i == 0 {
			out = append(out, keyCell+gap+styledVal)
			continue
		}
		out = append(out, strings.Repeat(" ", indent)+styledVal)
	}
	return out
}

// maxKeyWidth returns the widest key among the fields (separators excluded,
// since they have empty keys).
func maxKeyWidth(fields []Field) int {
	w := 0
	for _, f := range fields {
		if len(f.Key) > w {
			w = len(f.Key)
		}
	}
	return w
}

// wrapText wraps s to width columns on word boundaries, preserving any
// existing newlines as hard breaks. Always returns at least one line.
func wrapText(s string, width int) []string {
	width = max(width, 1)
	lines := make([]string, 0, strings.Count(s, "\n")+1)
	for para := range strings.SplitSeq(s, "\n") {
		lines = append(lines, wrapParagraph(para, width)...)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// wrapParagraph greedily wraps a single paragraph (no embedded newlines).
// A word longer than width is placed on its own line rather than split.
func wrapParagraph(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := ""
	for _, word := range words {
		switch {
		case cur == "":
			cur = word
		case len(cur)+1+len(word) <= width:
			cur += " " + word
		default:
			lines = append(lines, cur)
			cur = word
		}
	}
	lines = append(lines, cur)
	return lines
}
