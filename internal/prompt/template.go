package prompt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Template represents a prompt template with metadata and content.
type Template struct {
	// Metadata from frontmatter
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     int    `yaml:"version,omitempty"`

	// Template content (after frontmatter)
	Content string `yaml:"-"`

	// Source location for display
	Source string `yaml:"-"`
}

// TemplateInfo provides template metadata for listing.
type TemplateInfo struct {
	Name        string
	Description string
	Source      string // "built-in", "global", "project", or path
	Overrides   string // empty or name of what it overrides
}

// LoadTemplate finds and loads a template by name.
// Resolution order: project-local → user global → built-in
func LoadTemplate(name string) (*Template, error) {
	// 1. Project-local
	if tmpl, err := loadFromPath(projectTemplatesDir(), name); err == nil {
		tmpl.Source = "project"
		return tmpl, nil
	}

	// 2. User global
	if tmpl, err := loadFromPath(globalTemplatesDir(), name); err == nil {
		tmpl.Source = "global"
		return tmpl, nil
	}

	// 3. Built-in
	if tmpl, err := loadBuiltin(name); err == nil {
		tmpl.Source = "built-in"
		return tmpl, nil
	}

	return nil, fmt.Errorf("template %q not found", name)
}

// ListTemplates returns all available templates grouped by source.
func ListTemplates() ([]TemplateInfo, error) {
	seen := make(map[string]string) // name -> first source
	var templates []TemplateInfo

	// Collect from each source, tracking what overrides what
	sources := []struct {
		name string
		dir  string
	}{
		{"project", projectTemplatesDir()},
		{"global", globalTemplatesDir()},
	}

	for _, src := range sources {
		infos, err := listFromPath(src.dir, src.name)
		if err != nil {
			continue // directory might not exist
		}
		for _, info := range infos {
			if _, exists := seen[info.Name]; !exists {
				seen[info.Name] = src.name
				templates = append(templates, info)
			}
		}
	}

	// Add built-ins, marking overrides
	for _, info := range listBuiltins() {
		if overrideSource, exists := seen[info.Name]; exists {
			info.Overrides = overrideSource
		} else {
			templates = append(templates, info)
		}
	}

	return templates, nil
}

// projectTemplatesDir returns the project-local templates directory.
func projectTemplatesDir() string {
	return ".timbers/templates"
}

// globalTemplatesDir returns the user's global templates directory.
func globalTemplatesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "timbers", "templates")
}

// loadFromPath attempts to load a template from a directory.
func loadFromPath(dir, name string) (*Template, error) {
	if dir == "" {
		return nil, errors.New("no directory")
	}

	path := filepath.Join(dir, name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading template %s: %w", path, err)
	}

	return parseTemplate(string(data))
}

// listFromPath lists templates in a directory.
func listFromPath(dir, source string) ([]TemplateInfo, error) {
	if dir == "" {
		return nil, errors.New("no directory")
	}

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var templates []TemplateInfo
	for _, entry := range dirEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		tmpl, err := parseTemplate(string(data))
		if err != nil {
			continue
		}

		templates = append(templates, TemplateInfo{
			Name:        name,
			Description: tmpl.Description,
			Source:      source,
		})
	}

	return templates, nil
}

// parseTemplate parses a template from raw content with YAML frontmatter.
func parseTemplate(raw string) (*Template, error) {
	// Split frontmatter from content
	frontmatter, content := splitFrontmatter(raw)

	var tmpl Template
	if frontmatter != "" {
		if err := yaml.Unmarshal([]byte(frontmatter), &tmpl); err != nil {
			return nil, fmt.Errorf("invalid frontmatter: %w", err)
		}
	}

	tmpl.Content = strings.TrimSpace(content)
	return &tmpl, nil
}

// splitFrontmatter separates YAML frontmatter from content.
// Frontmatter is delimited by --- at the start and end.
func splitFrontmatter(raw string) (frontmatter, content string) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "---") {
		return "", raw
	}

	// Find end of frontmatter
	rest := raw[3:] // skip opening ---
	before, after, ok := strings.Cut(rest, "\n---")
	if !ok {
		return "", raw
	}

	return strings.TrimSpace(before), strings.TrimSpace(after)
}
