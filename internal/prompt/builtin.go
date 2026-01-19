package prompt

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed templates/*.md
var builtinFS embed.FS

// loadBuiltin loads a built-in template by name.
func loadBuiltin(name string) (*Template, error) {
	path := "templates/" + name + ".md"
	data, err := builtinFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading builtin template %s: %w", path, err)
	}
	return parseTemplate(string(data))
}

// listBuiltins returns info for all built-in templates.
func listBuiltins() []TemplateInfo {
	dirEntries, err := builtinFS.ReadDir("templates")
	if err != nil {
		return nil
	}

	var templates []TemplateInfo
	for _, entry := range dirEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		data, err := builtinFS.ReadFile("templates/" + entry.Name())
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
			Source:      "built-in",
		})
	}

	return templates
}
