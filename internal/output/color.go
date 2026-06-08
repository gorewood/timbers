package output

import (
	"io"
	"os"

	xterm "github.com/charmbracelet/x/term"
)

// ResolveColorMode determines the effective isTTY value based on the --color
// flag and actual TTY detection. The colorMode parameter accepts "never",
// "always", or "auto":
//   - "never":  always disable colors (returns false)
//   - "always": always enable colors (returns true)
//   - "auto":   use the detected isTTY value (default behavior)
func ResolveColorMode(colorMode string, isTTY bool) bool {
	switch colorMode {
	case "never":
		return false
	case "always":
		return true
	default:
		return isTTY
	}
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

// TerminalWidth returns the column width of the terminal backing writer, or
// fallback when writer is not a terminal or the size cannot be determined
// (piped output, tests writing to a buffer). This keeps panel wrapping
// deterministic off a TTY while fitting the real terminal when present.
func TerminalWidth(writer io.Writer, fallback int) int {
	file, ok := writer.(*os.File)
	if !ok {
		return fallback
	}
	width, _, err := xterm.GetSize(file.Fd())
	if err != nil || width <= 0 {
		return fallback
	}
	return width
}
