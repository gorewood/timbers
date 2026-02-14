package output

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestResolveColorMode(t *testing.T) {
	tests := []struct {
		name      string
		colorMode string
		isTTY     bool
		want      bool
	}{
		{name: "never disables on TTY", colorMode: "never", isTTY: true, want: false},
		{name: "never disables on non-TTY", colorMode: "never", isTTY: false, want: false},
		{name: "always enables on TTY", colorMode: "always", isTTY: true, want: true},
		{name: "always enables on non-TTY", colorMode: "always", isTTY: false, want: true},
		{name: "auto uses TTY true", colorMode: "auto", isTTY: true, want: true},
		{name: "auto uses TTY false", colorMode: "auto", isTTY: false, want: false},
		{name: "empty string defaults to auto", colorMode: "", isTTY: true, want: true},
		{name: "unknown value defaults to auto", colorMode: "bogus", isTTY: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveColorMode(tt.colorMode, tt.isTTY)
			if got != tt.want {
				t.Errorf("ResolveColorMode(%q, %v) = %v, want %v", tt.colorMode, tt.isTTY, got, tt.want)
			}
		})
	}
}

func TestIsTTY_Buffer(t *testing.T) {
	var buf bytes.Buffer
	if IsTTY(&buf) {
		t.Error("IsTTY(buffer) should return false")
	}
}

func TestResolveColorMode_NeverClearsStyles(t *testing.T) {
	var buf bytes.Buffer
	isTTY := ResolveColorMode("never", true) // force off despite "TTY"
	printer := NewPrinter(&buf, false, isTTY)

	// Printer should report non-TTY and use empty styles
	if printer.IsTTY() {
		t.Error("printer should report non-TTY when color=never")
	}

	// Error style should be empty (no foreground color set)
	empty := lipgloss.NewStyle()
	if printer.styles.Error.GetForeground() != empty.GetForeground() {
		t.Error("Error style should have no foreground color when color=never")
	}
}

func TestResolveColorMode_AlwaysKeepsStyles(t *testing.T) {
	var buf bytes.Buffer
	isTTY := ResolveColorMode("always", false) // force on despite non-TTY
	printer := NewPrinter(&buf, false, isTTY)

	// Printer should report TTY and retain colored styles
	if !printer.IsTTY() {
		t.Error("printer should report TTY when color=always")
	}

	// Error style should have a foreground color set (not empty)
	empty := lipgloss.NewStyle()
	if printer.styles.Error.GetForeground() == empty.GetForeground() {
		t.Error("Error style should have foreground color when color=always")
	}
}

func TestResolveColorMode_NeverNoANSI(t *testing.T) {
	var buf bytes.Buffer
	isTTY := ResolveColorMode("never", true)
	printer := NewPrinter(&buf, false, isTTY)

	printer.Error(NewUserError("test error"))

	// With styles cleared, output must not contain ANSI escape codes
	out := buf.String()
	if containsANSI(out) {
		t.Errorf("--color never should produce no ANSI codes, got: %q", out)
	}
}

// containsANSI checks if a string contains ANSI escape sequences.
func containsANSI(s string) bool {
	for i := range len(s) - 1 {
		if s[i] == '\033' && s[i+1] == '[' {
			return true
		}
	}
	return false
}
