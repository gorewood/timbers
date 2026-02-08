package setup

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gorewood/timbers/internal/output"
)

const (
	// TimbersHookMarkerBegin marks the start of timbers-managed content.
	TimbersHookMarkerBegin = "# BEGIN timbers"
	// TimbersHookMarkerEnd marks the end of timbers-managed content.
	TimbersHookMarkerEnd = "# END timbers"
)

// ClaudeHookContent is the hook script content that runs timbers prime.
var ClaudeHookContent = TimbersHookMarkerBegin + `
# Timbers session context injection
if command -v timbers >/dev/null 2>&1 && [ -d ".git" ]; then
  timbers prime 2>/dev/null
fi
` + TimbersHookMarkerEnd

// ResolveClaudeHookPath determines the hook path based on scope.
// If project is true, returns a project-local path; otherwise returns the global path.
func ResolveClaudeHookPath(project bool) (string, string, error) {
	if project {
		cwd, err := os.Getwd()
		if err != nil {
			return "", "", output.NewSystemErrorWithCause("failed to get working directory", err)
		}
		return filepath.Join(cwd, ".claude", "hooks", "user_prompt_submit.sh"), "project", nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", output.NewSystemErrorWithCause("failed to get home directory", err)
	}
	return filepath.Join(home, ".claude", "hooks", "user_prompt_submit.sh"), "global", nil
}

// IsTimbersSectionInstalled checks if the timbers section exists in a hook file.
func IsTimbersSectionInstalled(hookPath string) bool {
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), TimbersHookMarkerBegin)
}

// InstallTimbersSection adds or updates the timbers section in a hook file.
func InstallTimbersSection(hookPath string) error {
	hookDir := filepath.Dir(hookPath)
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		return output.NewSystemErrorWithCause("failed to create hook directory", err)
	}

	var content string
	existingContent, err := os.ReadFile(hookPath)
	if err == nil {
		content = string(existingContent)
		content = RemoveTimbersSectionFromContent(content)
	} else if !os.IsNotExist(err) {
		return output.NewSystemErrorWithCause("failed to read hook file", err)
	}

	if !strings.HasPrefix(content, "#!") {
		content = "#!/bin/bash\n" + content
	}

	content = strings.TrimRight(content, "\n") + "\n\n" + ClaudeHookContent + "\n"

	// #nosec G306 -- hook needs execute permission
	if err := os.WriteFile(hookPath, []byte(content), 0o755); err != nil {
		return output.NewSystemErrorWithCause("failed to write hook file", err)
	}

	return nil
}

// RemoveTimbersSectionFromHook removes the timbers section from a hook file.
func RemoveTimbersSectionFromHook(hookPath string) error {
	content, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return output.NewSystemErrorWithCause("failed to read hook file", err)
	}

	newContent := RemoveTimbersSectionFromContent(string(content))

	cleaned := strings.TrimSpace(strings.TrimPrefix(newContent, "#!/bin/bash"))
	if cleaned == "" {
		newContent = "#!/bin/bash\n"
	}

	// #nosec G306 -- hook needs execute permission
	if err := os.WriteFile(hookPath, []byte(newContent), 0o755); err != nil {
		return output.NewSystemErrorWithCause("failed to write hook file", err)
	}

	return nil
}

// RemoveTimbersSectionFromContent removes the timbers section from a content string.
func RemoveTimbersSectionFromContent(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inTimbers := false

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), TimbersHookMarkerBegin) {
			inTimbers = true
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), TimbersHookMarkerEnd) {
			inTimbers = false
			continue
		}
		if !inTimbers {
			result = append(result, line)
		}
	}

	finalContent := strings.Join(result, "\n")
	for strings.Contains(finalContent, "\n\n\n") {
		finalContent = strings.ReplaceAll(finalContent, "\n\n\n", "\n\n")
	}

	return strings.TrimRight(finalContent, "\n") + "\n"
}
