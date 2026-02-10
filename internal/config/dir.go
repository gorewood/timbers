// Package config provides the global configuration directory for timbers.
package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// Dir returns the timbers configuration directory.
//
// Resolution:
//   - $TIMBERS_CONFIG_HOME if set (explicit override)
//   - $XDG_CONFIG_HOME/timbers if set (respects XDG on any platform)
//   - %AppData%/timbers on Windows
//   - ~/.config/timbers on macOS and Linux
func Dir() string {
	// Explicit override
	if dir := os.Getenv("TIMBERS_CONFIG_HOME"); dir != "" {
		return dir
	}

	// XDG override (works on any platform)
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "timbers")
	}

	// Windows: use AppData
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "timbers")
		}
	}

	// macOS and Linux: ~/.config/timbers
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "timbers")
}
