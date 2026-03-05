package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// sectionStart is the opening delimiter for timbers hook sections.
	sectionStart = "# --- timbers section (do not edit) ---"
	// sectionEnd is the closing delimiter for timbers hook sections.
	sectionEnd = "# --- end timbers section ---"
)

// AppendTimbersSection appends a delimited timbers section to the hook file at
// hookPath. If the file does not exist, it is created with a shebang line.
// The operation is idempotent: if a timbers section already exists, no changes
// are made. The sectionContent is automatically wrapped in section delimiters.
// Writes are atomic via temp file + os.Rename.
func AppendTimbersSection(hookPath string, sectionContent string) error {
	var content string

	existing, err := os.ReadFile(hookPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading hook file: %w", err)
		}
		// File doesn't exist — start with shebang.
		content = "#!/bin/sh\n"
	} else {
		content = string(existing)
		// Idempotent: if section already present, do nothing.
		if hasSectionDelimiters(content) {
			return nil
		}
	}

	// Ensure trailing newline before appending section.
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	content += sectionStart + "\n"
	content += sectionContent
	if !strings.HasSuffix(sectionContent, "\n") {
		content += "\n"
	}
	content += sectionEnd + "\n"

	return atomicWrite(hookPath, content)
}

// RemoveTimbersSection removes the delimited timbers section from the hook file
// at hookPath. If the file becomes empty (only shebang + whitespace) after
// removal, the file is deleted. Returns nil if the file does not exist or
// contains no timbers section (idempotent). Writes are atomic via temp file +
// os.Rename.
func RemoveTimbersSection(hookPath string) error {
	existing, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading hook file: %w", err)
	}

	content := string(existing)
	if !hasSectionDelimiters(content) {
		return nil
	}

	remaining := removeSectionLines(content)

	// If only shebang + whitespace remains, delete the file.
	stripped := strings.TrimSpace(remaining)
	if stripped == "" || stripped == "#!/bin/sh" {
		if removeErr := os.Remove(hookPath); removeErr != nil {
			return fmt.Errorf("removing empty hook file: %w", removeErr)
		}
		return nil
	}

	return atomicWrite(hookPath, remaining)
}

// removeSectionLines strips the timbers section (delimiters inclusive) from content.
func removeSectionLines(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == sectionStart {
			inSection = true
			continue
		}
		if inSection && trimmed == sectionEnd {
			inSection = false
			continue
		}
		if !inSection {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// HasTimbersSection returns true if the hook file at hookPath contains a
// timbers section. Detects both the new delimited format and the old format
// (containing "timbers hook run" without delimiters) for backward compatibility.
func HasTimbersSection(hookPath string) bool {
	data, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	content := string(data)
	return hasSectionDelimiters(content) || hasOldFormatTimbers(content)
}

// hasSectionDelimiters returns true if content contains the timbers section
// start delimiter.
func hasSectionDelimiters(content string) bool {
	return strings.Contains(content, sectionStart)
}

// hasOldFormatTimbers returns true if content contains "timbers hook run"
// without the new section delimiters. This detects the old hook format where
// timbers owned the entire file.
func hasOldFormatTimbers(content string) bool {
	return strings.Contains(content, "timbers hook run") && !hasSectionDelimiters(content)
}

// cleanupTempFile is a best-effort removal of temporary files during error paths.
func cleanupTempFile(name string) {
	_ = os.Remove(name)
}

// atomicWrite writes content to path atomically by writing to a temp file in
// the same directory and renaming it.
func atomicWrite(path string, content string) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".timbers-hook-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		cleanupTempFile(tmpName)
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		cleanupTempFile(tmpName)
		return fmt.Errorf("closing temp file: %w", err)
	}

	// #nosec G302 -- hook files need execute permission
	if err := os.Chmod(tmpName, 0o755); err != nil {
		cleanupTempFile(tmpName)
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		cleanupTempFile(tmpName)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}
