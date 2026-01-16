package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Printer handles formatted output to a writer.
// It supports both JSON and human-readable output modes.
type Printer struct {
	w      io.Writer
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
	}

	// Disable colors if not a TTY
	if !isTTY {
		styles.Error = lipgloss.NewStyle()
		styles.Success = lipgloss.NewStyle()
		styles.Warning = lipgloss.NewStyle()
		styles.Bold = lipgloss.NewStyle()
		styles.Dim = lipgloss.NewStyle()
	}

	return &Printer{
		w:      writer,
		json:   jsonMode,
		isTTY:  isTTY,
		styles: styles,
	}
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
		_, _ = fmt.Fprintln(p.w, p.styles.Success.Render(msg))
		return nil
	}

	// Pretty-print the data
	for key, val := range data {
		_, _ = fmt.Fprintf(p.w, "%s: %v\n", p.styles.Bold.Render(key), val)
	}
	return nil
}

// Error outputs an error.
// For JSON mode, outputs {"error": "...", "code": N}.
// For human mode, outputs a styled error message.
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
		_, _ = p.w.Write(ErrorJSON(exitErr.Message, exitErr.Code))
		_, _ = fmt.Fprintln(p.w)
		return
	}

	// Human-readable error
	_, _ = fmt.Fprintf(p.w, "%s: %s\n", p.styles.Error.Render("Error"), exitErr.Message)
}

// Print formats and writes to the output without a newline.
func (p *Printer) Print(format string, args ...any) {
	_, _ = fmt.Fprintf(p.w, format, args...)
}

// Println writes a line to the output.
func (p *Printer) Println(args ...any) {
	_, _ = fmt.Fprintln(p.w, args...)
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

// IsTTY checks if a writer is a terminal.
// Returns true only for os.File that is a terminal.
func IsTTY(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
