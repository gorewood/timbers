package output

import (
	"io"
	"os"
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
