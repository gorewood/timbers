package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/fang"
)

// TestErrorHandlerNonTTYIsSinglePlainLine is the regression guard: when stderr
// is not a terminal, the error must render as exactly one plain line equal to
// the error text — no "ERROR" badge, no padding/blank lines, no "Try --help".
// A trailing blank line is what made blocked commits vanish under
// `2>&1 | tail -1` and output compression.
func TestErrorHandlerNonTTYIsSinglePlainLine(t *testing.T) {
	var buf bytes.Buffer
	handler := newErrorHandler(false)
	handler(&buf, fang.Styles{}, errors.New("undocumented commit(s) exist; run 'timbers log' first"))

	got := buf.String()
	if got != "undocumented commit(s) exist; run 'timbers log' first\n" {
		t.Fatalf("non-TTY error not a single plain line: %q", got)
	}

	// The actionable text must be the last non-empty line — i.e. it survives
	// `tail -1`.
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if last := lines[len(lines)-1]; !strings.Contains(last, "timbers log") {
		t.Errorf("last line lost the actionable message: %q", last)
	}
	if strings.Contains(got, "ERROR") || strings.Contains(got, "Try --help") {
		t.Errorf("non-TTY error should carry no decorative chrome: %q", got)
	}
}
