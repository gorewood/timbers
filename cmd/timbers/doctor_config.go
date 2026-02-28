package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorewood/timbers/internal/config"
	"github.com/gorewood/timbers/internal/draft"
	"github.com/gorewood/timbers/internal/llm"
)

// checkVersion compares installed version against latest GitHub release.
func checkVersion() checkResult {
	// Skip for dev builds
	if version == "dev" || version == "" {
		return checkResult{
			Name:    "Version",
			Status:  checkPass,
			Message: "dev build (skipping update check)",
		}
	}

	latest, err := fetchLatestVersion()
	if err != nil {
		return checkResult{
			Name:    "Version",
			Status:  checkPass,
			Message: version + " (update check failed: " + err.Error() + ")",
		}
	}

	installed := strings.TrimPrefix(version, "v")
	latestClean := strings.TrimPrefix(latest, "v")

	if installed == latestClean {
		return checkResult{
			Name:    "Version",
			Status:  checkPass,
			Message: version + " (latest)",
		}
	}

	return checkResult{
		Name:    "Version",
		Status:  checkWarn,
		Message: fmt.Sprintf("%s (latest: %s)", version, latest),
		Hint:    "Update: curl -fsSL https://raw.githubusercontent.com/gorewood/timbers/main/install.sh | bash",
	}
}

// fetchLatestVersion queries GitHub for the latest release tag.
func fetchLatestVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.github.com/repos/gorewood/timbers/releases/latest", nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	return release.TagName, nil
}

// runConfigChecks performs configuration-related checks.
func runConfigChecks(flags *doctorFlags) []checkResult {
	checks := make([]checkResult, 0, 4)
	checks = append(checks, checkConfigDir(flags))
	checks = append(checks, checkEnvFiles())
	checks = append(checks, checkTemplates())
	checks = append(checks, checkGeneration())
	return checks
}

// pipeCLIs are LLM CLIs that accept stdin for pipe-based generation.
var pipeCLIs = []string{"claude", "codex", "gemini"}

// findCLIs returns the names of LLM CLIs found in PATH.
func findCLIs() []string {
	var found []string
	for _, name := range pipeCLIs {
		if _, err := exec.LookPath(name); err == nil {
			found = append(found, name)
		}
	}
	return found
}

// findAPIKeys returns the env var names of set API keys.
func findAPIKeys() []string {
	var found []string
	for _, envVar := range llm.APIKeyEnvVars() {
		if os.Getenv(envVar) != "" {
			found = append(found, envVar)
		}
	}
	return found
}

// generationHint builds the hint for when no generation method is available.
func generationHint() string {
	var b strings.Builder
	b.WriteString("Set an API key for 'timbers draft --model':")
	for _, envVar := range llm.APIKeyEnvVars() {
		b.WriteString("\n      ")
		b.WriteString(envVar)
	}
	if os.Getenv("CI") == "" {
		b.WriteString("\n      Or install an LLM CLI: ")
		b.WriteString(strings.Join(pipeCLIs, ", "))
	}
	return b.String()
}

// checkGeneration reports whether draft generation methods are available.
func checkGeneration() checkResult {
	foundCLIs := findCLIs()
	foundKeys := findAPIKeys()

	var parts []string
	if len(foundCLIs) > 0 {
		parts = append(parts, "pipe: "+strings.Join(foundCLIs, ", "))
	} else {
		parts = append(parts, "pipe: no CLI found")
	}
	if len(foundKeys) > 0 {
		parts = append(parts, "direct: "+strings.Join(foundKeys, ", "))
	} else {
		parts = append(parts, "direct: no API key")
	}

	msg := strings.Join(parts, " | ")

	if len(foundCLIs) == 0 && len(foundKeys) == 0 {
		return checkResult{
			Name:    "Generation",
			Status:  checkWarn,
			Message: msg,
			Hint:    generationHint(),
		}
	}

	return checkResult{
		Name:    "Generation",
		Status:  checkPass,
		Message: msg,
	}
}

// checkConfigDir reports the resolved configuration directory.
func checkConfigDir(flags *doctorFlags) checkResult {
	dir := config.Dir()
	if dir == "" {
		return checkResult{
			Name:    "Config Dir",
			Status:  checkWarn,
			Message: "could not determine config directory",
		}
	}

	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return checkResult{
			Name:    "Config Dir",
			Status:  checkPass,
			Message: dir,
		}
	}

	if flags.fix {
		if err := os.MkdirAll(dir, 0o755); err == nil {
			return checkResult{
				Name:    "Config Dir",
				Status:  checkPass,
				Message: dir + " (created)",
			}
		}
	}

	return checkResult{
		Name:    "Config Dir",
		Status:  checkPass,
		Message: dir + " (not created yet)",
		Hint:    "mkdir -p " + dir + " or 'timbers doctor --fix'",
	}
}

// checkEnvFiles reports which env files are active and which API keys are configured.
func checkEnvFiles() checkResult {
	var found []string
	var keys []string

	// Check each env file location in resolution order
	type envCandidate struct {
		label string
		path  string
	}
	candidates := []envCandidate{
		{"local", ".env.local"},
		{"repo", ".env"},
	}
	if dir := config.Dir(); dir != "" {
		candidates = append(candidates, envCandidate{"global", filepath.Join(dir, "env")})
	}

	for _, c := range candidates {
		if _, err := os.Stat(c.path); err == nil {
			found = append(found, c.label+" ("+c.path+")")
		}
	}

	// Check which API keys are available (from any source)
	for _, envVar := range llm.APIKeyEnvVars() {
		if os.Getenv(envVar) != "" {
			// Derive display label: "ANTHROPIC_API_KEY" -> "anthropic"
			keys = append(keys, strings.ToLower(strings.TrimSuffix(envVar, "_API_KEY")))
		}
	}

	msg := "no env files found"
	if len(found) > 0 {
		msg = strings.Join(found, ", ")
	}
	if len(keys) > 0 {
		msg += " | keys: " + strings.Join(keys, ", ")
	}

	if len(found) == 0 && len(keys) == 0 {
		return checkResult{
			Name:    "Env Files",
			Status:  checkPass,
			Message: msg + " (not needed for local models)",
			Hint:    "For cloud models: mkdir -p " + config.Dir() + " && cp .env.example " + filepath.Join(config.Dir(), "env"),
		}
	}

	return checkResult{
		Name:    "Env Files",
		Status:  checkPass,
		Message: msg,
	}
}

// checkTemplates reports project-local and global custom templates.
func checkTemplates() checkResult {
	var parts []string

	// Project-local templates
	projectCount := countTemplates(".timbers/templates")
	if projectCount > 0 {
		parts = append(parts, fmt.Sprintf("%d project-local", projectCount))
	}

	// Global templates
	if dir := config.Dir(); dir != "" {
		globalDir := filepath.Join(dir, "templates")
		globalCount := countTemplates(globalDir)
		if globalCount > 0 {
			parts = append(parts, fmt.Sprintf("%d global", globalCount))
		}
	}

	if len(parts) == 0 {
		return checkResult{
			Name:    "Custom Templates",
			Status:  checkPass,
			Message: fmt.Sprintf("none (%d built-in available)", draft.BuiltinCount()),
			Hint:    "Run 'timbers draft --list' to see built-in templates",
		}
	}

	return checkResult{
		Name:    "Custom Templates",
		Status:  checkPass,
		Message: fmt.Sprintf("%s + %d built-in", strings.Join(parts, ", "), draft.BuiltinCount()),
	}
}

// countTemplates counts .md files in a directory.
func countTemplates(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			count++
		}
	}
	return count
}
