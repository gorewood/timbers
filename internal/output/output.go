package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Printer handles formatted output to a writer.
// It supports both JSON and human-readable output modes.
type Printer struct {
	w      io.Writer
	errW   io.Writer
	json   bool
	isTTY  bool
	styles *Styles
}

// Styles holds lipgloss styles for human-readable output.
type Styles struct {
	Error   lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Bold    lipgloss.Style
	Dim     lipgloss.Style
	Title   lipgloss.Style
	Muted   lipgloss.Style
	Key     lipgloss.Style
	Value   lipgloss.Style
	Border  lipgloss.Color
	Accent  lipgloss.Style
}

// NewPrinter creates a new Printer.
// If json is true, output will be JSON formatted.
// If isTTY is true, colors will be enabled for human output.
func NewPrinter(writer io.Writer, jsonMode bool, isTTY bool) *Printer {
	styles := &Styles{
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true), // Red
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color("10")),           // Green
		Warning: lipgloss.NewStyle().Foreground(lipgloss.Color("11")),           // Yellow
		Bold:    lipgloss.NewStyle().Bold(true),
		Dim:     lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		Title:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")), // Blue
		Muted:   lipgloss.NewStyle().Faint(true),
		Key:     lipgloss.NewStyle().Foreground(lipgloss.Color("14")), // Cyan
		Value:   lipgloss.NewStyle(),
		Border:  lipgloss.Color("8"),                                  // Gray
		Accent:  lipgloss.NewStyle().Foreground(lipgloss.Color("13")), // Magenta
	}

	// Disable colors if not a TTY
	if !isTTY {
		styles.Error = lipgloss.NewStyle()
		styles.Success = lipgloss.NewStyle()
		styles.Warning = lipgloss.NewStyle()
		styles.Bold = lipgloss.NewStyle()
		styles.Dim = lipgloss.NewStyle()
		styles.Title = lipgloss.NewStyle()
		styles.Muted = lipgloss.NewStyle()
		styles.Key = lipgloss.NewStyle()
		styles.Value = lipgloss.NewStyle()
		styles.Border = lipgloss.Color("")
		styles.Accent = lipgloss.NewStyle()
	}

	return &Printer{
		w:      writer,
		errW:   writer,
		json:   jsonMode,
		isTTY:  isTTY,
		styles: styles,
	}
}

// WithStderr sets a separate writer for errors and warnings in human mode.
// In JSON mode, errors still go to the main writer (structured protocol).
// Returns the printer for chaining.
func (p *Printer) WithStderr(w io.Writer) *Printer {
	p.errW = w
	return p
}

// IsJSON returns true if the printer is in JSON mode.
func (p *Printer) IsJSON() bool {
	return p.json
}

// IsTTY returns true if the printer output is a TTY.
func (p *Printer) IsTTY() bool {
	return p.isTTY
}

// Success outputs a success result.
// For JSON mode, outputs the data as JSON.
// For human mode, looks for a "message" key or pretty-prints the data.
func (p *Printer) Success(data map[string]any) error {
	if p.json {
		return p.writeJSON(data)
	}

	// Human-readable output
	if msg, ok := data["message"].(string); ok {
		mustWrite(fmt.Fprintln(p.w, p.styles.Success.Render(msg)))
		return nil
	}

	// Pretty-print the data
	for key, val := range data {
		mustWrite(fmt.Fprintf(p.w, "%s: %v\n", p.styles.Bold.Render(key), val))
	}
	return nil
}

// Error outputs an error.
// For JSON mode, outputs {"error": "...", "code": N} to stdout.
// For human mode, outputs a styled error message to stderr (if set).
func (p *Printer) Error(err error) {
	exitErr := &ExitError{}
	ok := errors.As(err, &exitErr)
	if !ok {
		exitErr = &ExitError{
			Code:    ExitUserError,
			Message: err.Error(),
		}
	}

	if p.json {
		mustWrite(p.w.Write(ErrorJSON(exitErr.Message, exitErr.Code)))
		mustWrite(fmt.Fprintln(p.w))
		return
	}

	// Human-readable error goes to errW (stderr when set)
	mustWrite(fmt.Fprintf(p.errW, "%s: %s\n", p.styles.Error.Render("Error"), exitErr.Message))
}

// Warn outputs a warning message.
// For JSON mode, outputs {"warning": "..."} to stdout.
// For human mode, outputs a styled warning to stderr (if set).
func (p *Printer) Warn(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if p.json {
		data := map[string]any{"warning": msg}
		_ = p.writeJSON(data)
		return
	}
	mustWrite(fmt.Fprintf(p.errW, "%s: %s\n", p.styles.Warning.Render("Warning"), msg))
}

// Stderr writes a message to the error writer (for status hints when piped).
// No-op in JSON mode (structured protocol handles metadata).
func (p *Printer) Stderr(format string, args ...any) {
	if p.json {
		return
	}
	mustWrite(fmt.Fprintf(p.errW, format, args...))
}

// Print formats and writes to the output without a newline.
func (p *Printer) Print(format string, args ...any) {
	mustWrite(fmt.Fprintf(p.w, format, args...))
}

// Println writes a line to the output.
func (p *Printer) Println(args ...any) {
	mustWrite(fmt.Fprintln(p.w, args...))
}

// writeJSON encodes data as JSON and writes it.
func (p *Printer) writeJSON(data any) error {
	enc := json.NewEncoder(p.w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}

// WriteJSON encodes any data as JSON and writes it.
// Use this for outputting structs or other types that aren't maps.
func (p *Printer) WriteJSON(data any) error {
	return p.writeJSON(data)
}

// ErrorJSON returns JSON-formatted error bytes.
// Format: {"error": "message", "code": N}
func ErrorJSON(message string, code int) []byte {
	data := map[string]any{
		"error": message,
		"code":  code,
	}
	result, _ := json.Marshal(data)
	return result
}

// mustWrite panics if a write operation fails.
// Use this to wrap write operations that should never fail
// (e.g., writing to stdout/stderr or buffers).
func mustWrite(_ int, err error) {
	if err != nil {
		panic(fmt.Sprintf("write failed: %v", err))
	}
}

// Table renders a simple table with column alignment.
// Headers are rendered in Bold style. Column widths are auto-calculated.
// For non-TTY output, renders plain text with space padding.
func (p *Printer) Table(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	widths := calcColumnWidths(headers, rows)
	p.printTableHeaders(headers, widths)
	p.printTableRows(rows, widths)
}

// calcColumnWidths computes the max width for each column.
func calcColumnWidths(headers []string, rows [][]string) []int {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	return widths
}

// printTableHeaders renders the table header row.
func (p *Printer) printTableHeaders(headers []string, widths []int) {
	for i, h := range headers {
		padded := padRight(h, widths[i])
		if i > 0 {
			mustWrite(fmt.Fprint(p.w, "  "))
		}
		mustWrite(fmt.Fprint(p.w, p.styles.Bold.Render(padded)))
	}
	mustWrite(fmt.Fprintln(p.w))
}

// printTableRows renders all data rows.
func (p *Printer) printTableRows(rows [][]string, widths []int) {
	for _, row := range rows {
		p.printTableRow(row, widths)
	}
}

// printTableRow renders a single data row.
func (p *Printer) printTableRow(row []string, widths []int) {
	for i, cell := range row {
		if i >= len(widths) {
			break
		}
		if i > 0 {
			mustWrite(fmt.Fprint(p.w, "  "))
		}
		mustWrite(fmt.Fprint(p.w, padRight(cell, widths[i])))
	}
	mustWrite(fmt.Fprintln(p.w))
}

// Box renders content in a bordered box with an optional title.
// For TTY output, uses lipgloss.RoundedBorder.
// For non-TTY output, renders plain text without borders.
func (p *Printer) Box(title string, content string) {
	if !p.isTTY {
		// Non-TTY: plain text without borders
		if title != "" {
			mustWrite(fmt.Fprintln(p.w, title))
			mustWrite(fmt.Fprintln(p.w))
		}
		mustWrite(fmt.Fprintln(p.w, content))
		return
	}

	// TTY: use lipgloss border
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.styles.Border).
		Padding(0, 1)

	boxContent := content
	if title != "" {
		boxContent = p.styles.Title.Render(title) + "\n\n" + content
	}

	mustWrite(fmt.Fprintln(p.w, style.Render(boxContent)))
}

// Section renders a section header with underline.
// Adds a blank line before the header.
func (p *Printer) Section(title string) {
	mustWrite(fmt.Fprintln(p.w))
	mustWrite(fmt.Fprintln(p.w, p.styles.Title.Render(title)))
	// Create underline matching title length
	underline := strings.Repeat("─", len(title))
	mustWrite(fmt.Fprintln(p.w, p.styles.Muted.Render(underline)))
}

// KeyValue renders a key-value pair with styles applied.
// Format: "Key: Value"
func (p *Printer) KeyValue(key string, value string) {
	styledKey := p.styles.Key.Render(key + ":")
	styledValue := p.styles.Value.Render(value)
	mustWrite(fmt.Fprintf(p.w, "%s %s\n", styledKey, styledValue))
}

// padRight pads a string with spaces to reach the target width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
