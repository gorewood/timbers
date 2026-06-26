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

// RemoveTimbersSection removes timbers integration from the hook file at
// hookPath. For new-format hooks (with delimiters), removes the delimited
// section. For old-format hooks (timbers owns the entire file), deletes the
// file. If the file becomes empty after section removal, it is deleted.
// Returns nil if the file does not exist or contains no timbers content
// (idempotent). Writes are atomic via temp file + os.Rename.
func RemoveTimbersSection(hookPath string) error {
	existing, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading hook file: %w", err)
	}

	content := string(existing)

	// New format: remove delimited section.
	if hasSectionDelimiters(content) {
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

	// Old format: timbers owns the file. Delete it.
	if hasOldFormatTimbers(content) {
		if removeErr := os.Remove(hookPath); removeErr != nil {
			return fmt.Errorf("removing old-format hook file: %w", removeErr)
		}
		return nil
	}

	return nil
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

// IsOldFormatHook returns true if the hook file at hookPath contains timbers
// integration in the old format (no section delimiters). These hooks were
// written by timbers before the section-delimited format was introduced.
func IsOldFormatHook(hookPath string) bool {
	data, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return hasOldFormatTimbers(string(data))
}

// MigrateOldFormatHook replaces an old-format timbers hook with the
// section-delimited format. The old file is replaced entirely (since timbers
// owned it). The sectionContent should be the hook-type-specific content
// (without delimiters — they are added by AppendTimbersSection).
func MigrateOldFormatHook(hookPath string, sectionContent string) error {
	if !IsOldFormatHook(hookPath) {
		return nil
	}
	// Remove the old file and create a fresh section-delimited one.
	if err := os.Remove(hookPath); err != nil {
		return fmt.Errorf("removing old-format hook: %w", err)
	}
	return AppendTimbersSection(hookPath, sectionContent)
}

// extractSectionContent returns the content between the timbers section
// delimiters (exclusive of the delimiter lines), and whether a delimited
// section was found. Trailing newlines are trimmed so the result can be
// compared against a freshly generated section regardless of how the file
// was terminated.
func extractSectionContent(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	var section []string
	inSection := false
	found := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == sectionStart {
			inSection = true
			found = true
			continue
		}
		if inSection && trimmed == sectionEnd {
			inSection = false
			continue
		}
		if inSection {
			section = append(section, line)
		}
	}

	return strings.TrimRight(strings.Join(section, "\n"), "\n"), found
}

// SectionUpToDate reports whether the hook at hookPath contains a delimited
// timbers section whose content matches sectionContent. Returns false when the
// file is missing, has no delimited section (e.g. old-format hooks), or the
// installed section has drifted from the current generator output. Comparing
// content directly — rather than a stamped version number — means any change to
// the generated hook is detected without anything to keep in sync.
func SectionUpToDate(hookPath string, sectionContent string) bool {
	data, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	installed, found := extractSectionContent(string(data))
	if !found {
		return false
	}
	return installed == strings.TrimRight(sectionContent, "\n")
}

// ReplaceTimbersSection refreshes the timbers section in the hook at hookPath to
// sectionContent, preserving any non-timbers content in the file. Implemented as
// remove-then-append so a drifted section is swapped for the current generator
// output. Safe to call when no section is present (it simply installs one).
func ReplaceTimbersSection(hookPath string, sectionContent string) error {
	if err := RemoveTimbersSection(hookPath); err != nil {
		return err
	}
	return AppendTimbersSection(hookPath, sectionContent)
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
