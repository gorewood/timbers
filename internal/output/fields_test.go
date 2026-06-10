package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWrapText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  []string
	}{
		{"empty", "", 10, []string{""}},
		{"fits on one line", "alpha beta", 20, []string{"alpha beta"}},
		{"wraps on word boundary", "alpha beta gamma", 11, []string{"alpha beta", "gamma"}},
		{"long word on its own line", "supercalifragilistic ok", 8, []string{"supercalifragilistic", "ok"}},
		{"hard newline preserved", "one\ntwo", 20, []string{"one", "two"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapText(tt.input, tt.width)
			if len(got) != len(tt.want) {
				t.Fatalf("wrapText(%q, %d) = %v, want %v", tt.input, tt.width, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// newTestPanel renders fields through a non-TTY printer (no border, no styles)
// at a fixed width, returning the raw output for assertions.
func newTestPanel(t *testing.T, width int, fields []Field) string {
	t.Helper()
	var buf bytes.Buffer
	NewPrinter(&buf, false, false).WithWidth(width).FieldsBox("Title", fields)
	return buf.String()
}

func TestFieldsBoxAlignsKeysToCommonWidth(t *testing.T) {
	out := newTestPanel(t, 80, []Field{
		{Key: "What", Value: "did a thing"},
		{Key: "Commits", Value: "1"},
	})
	// "What" is padded to the width of the widest key ("Commits" = 7) plus a
	// two-space gap, so the value column starts at the same offset for both.
	if !strings.Contains(out, "What     did a thing") {
		t.Errorf("keys not aligned to common column:\n%s", out)
	}
	if !strings.Contains(out, "Commits  1") {
		t.Errorf("widest key has wrong gap:\n%s", out)
	}
}

func TestFieldsBoxWrapsWithHangingIndent(t *testing.T) {
	out := newTestPanel(t, 40, []Field{
		{Key: "Why", Value: "alpha beta gamma delta epsilon zeta eta"},
	})
	// keyWidth=3, gap=2 -> value column / continuation indent = 5.
	if !strings.Contains(out, "Why  alpha") {
		t.Errorf("first value line missing:\n%s", out)
	}
	if !strings.Contains(out, "\n     ") {
		t.Errorf("continuation line not hanging-indented to value column:\n%s", out)
	}
}

func TestFieldsBoxSeparatorIsBlankLine(t *testing.T) {
	out := newTestPanel(t, 80, []Field{
		{Key: "What", Value: "x"},
		Separator(),
		{Key: "ID", Value: "y"},
	})
	// The separator must produce an empty line between the two rows.
	if !strings.Contains(out, "What  x\n\nID") {
		t.Errorf("separator did not render a blank line:\n%q", out)
	}
}

func TestFieldsBoxNonTTYHasNoBorder(t *testing.T) {
	out := newTestPanel(t, 80, []Field{{Key: "What", Value: "x"}})
	for _, ch := range []string{"╭", "╮", "╰", "╯", "│", "─"} {
		if strings.Contains(out, ch) {
			t.Errorf("piped output should not contain border rune %q:\n%s", ch, out)
		}
	}
	// Title still present, content retained.
	if !strings.Contains(out, "Title") || !strings.Contains(out, "What  x") {
		t.Errorf("piped panel missing title or content:\n%s", out)
	}
}
