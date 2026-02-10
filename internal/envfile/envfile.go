// Package envfile loads environment variables from .env files.
// Variables already set in the environment take precedence.
package envfile

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Load reads a .env file and sets any variables not already in the environment.
// Returns nil if the file doesn't exist. Returns an error only for read failures.
func Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("opening env file %s: %w", path, err)
	}
	defer file.Close() //nolint:errcheck // best-effort close on read-only file

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := parseEnvLine(line)
		if !ok {
			continue
		}

		// Only set if not already in the environment
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading env file %s: %w", path, err)
	}
	return nil
}

// parseEnvLine extracts KEY=VALUE from a line.
// Handles optional quoting (single or double quotes) around the value.
func parseEnvLine(line string) (key, value string, ok bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	key = strings.TrimSpace(parts[0])
	value = strings.TrimSpace(parts[1])

	if key == "" {
		return "", "", false
	}

	// Strip optional export prefix
	key = strings.TrimPrefix(key, "export ")
	key = strings.TrimSpace(key)

	// Strip matching quotes from value
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	return key, value, true
}
