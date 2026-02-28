package draft

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantFrontmatter string
		wantContent     string
	}{
		{
			name:            "no frontmatter",
			input:           "Just some content",
			wantFrontmatter: "",
			wantContent:     "Just some content",
		},
		{
			name: "with frontmatter",
			input: `---
name: test
description: A test template
---
Template content here`,
			wantFrontmatter: "name: test\ndescription: A test template",
			wantContent:     "Template content here",
		},
		{
			name: "frontmatter only opening",
			input: `---
name: test
No closing delimiter`,
			wantFrontmatter: "",
			wantContent:     "---\nname: test\nNo closing delimiter",
		},
		{
			name: "empty frontmatter",
			input: `---
---
Content after empty frontmatter`,
			wantFrontmatter: "",
			wantContent:     "Content after empty frontmatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrontmatter, gotContent := splitFrontmatter(tt.input)
			if gotFrontmatter != tt.wantFrontmatter {
				t.Errorf("splitFrontmatter() frontmatter = %q, want %q", gotFrontmatter, tt.wantFrontmatter)
			}
			if gotContent != tt.wantContent {
				t.Errorf("splitFrontmatter() content = %q, want %q", gotContent, tt.wantContent)
			}
		})
	}
}

func TestParseTemplate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantName    string
		wantDesc    string
		wantContent string
		wantErr     bool
	}{
		{
			name: "valid template",
			input: `---
name: changelog
description: Generate a changelog
version: 1
---
Create a changelog from {{entries_json}}`,
			wantName:    "changelog",
			wantDesc:    "Generate a changelog",
			wantContent: "Create a changelog from {{entries_json}}",
			wantErr:     false,
		},
		{
			name:        "no frontmatter",
			input:       "Just content, no metadata",
			wantName:    "",
			wantDesc:    "",
			wantContent: "Just content, no metadata",
			wantErr:     false,
		},
		{
			name: "invalid yaml",
			input: `---
name: [invalid yaml
---
Content`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := parseTemplate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if tmpl.Name != tt.wantName {
				t.Errorf("parseTemplate() Name = %q, want %q", tmpl.Name, tt.wantName)
			}
			if tmpl.Description != tt.wantDesc {
				t.Errorf("parseTemplate() Description = %q, want %q", tmpl.Description, tt.wantDesc)
			}
			if tmpl.Content != tt.wantContent {
				t.Errorf("parseTemplate() Content = %q, want %q", tmpl.Content, tt.wantContent)
			}
		})
	}
}

func TestLoadBuiltinTemplate(t *testing.T) {
	// Test loading a known built-in template
	tmpl, err := loadBuiltin("changelog")
	if err != nil {
		t.Fatalf("loadBuiltin(changelog) error = %v", err)
	}

	if tmpl.Name != "changelog" {
		t.Errorf("loadBuiltin(changelog) Name = %q, want %q", tmpl.Name, "changelog")
	}

	if tmpl.Description == "" {
		t.Error("loadBuiltin(changelog) Description is empty")
	}

	if tmpl.Content == "" {
		t.Error("loadBuiltin(changelog) Content is empty")
	}

	// Test loading non-existent template
	_, err = loadBuiltin("nonexistent-template")
	if err == nil {
		t.Error("loadBuiltin(nonexistent) expected error, got nil")
	}
}

func TestLoadTemplateResolution(t *testing.T) {
	// Create a temporary directory for project templates
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	chdirErr := os.Chdir(tmpDir)
	if chdirErr != nil {
		t.Fatalf("failed to chdir to temp dir: %v", chdirErr)
	}

	// Test loading built-in template (no project override)
	tmpl, err := LoadTemplate("changelog")
	if err != nil {
		t.Fatalf("LoadTemplate(changelog) error = %v", err)
	}
	if tmpl.Source != "built-in" {
		t.Errorf("LoadTemplate(changelog) Source = %q, want %q", tmpl.Source, "built-in")
	}

	// Create project override
	mkdirErr := os.MkdirAll(".timbers/templates", 0755)
	if mkdirErr != nil {
		t.Fatalf("failed to create templates dir: %v", mkdirErr)
	}
	overrideContent := `---
name: changelog
description: Project-specific changelog
---
Custom changelog template`
	writeErr := os.WriteFile(filepath.Join(".timbers/templates", "changelog.md"), []byte(overrideContent), 0600)
	if writeErr != nil {
		t.Fatalf("failed to write override template: %v", writeErr)
	}

	// Test that project override takes precedence
	tmpl, err = LoadTemplate("changelog")
	if err != nil {
		t.Fatalf("LoadTemplate(changelog) with override error = %v", err)
	}
	if tmpl.Source != "project" {
		t.Errorf("LoadTemplate(changelog) with override Source = %q, want %q", tmpl.Source, "project")
	}
	if tmpl.Description != "Project-specific changelog" {
		t.Errorf("LoadTemplate(changelog) Description = %q, want %q", tmpl.Description, "Project-specific changelog")
	}

	// Test loading non-existent template
	_, err = LoadTemplate("nonexistent")
	if err == nil {
		t.Error("LoadTemplate(nonexistent) expected error, got nil")
	}
}

func TestListBuiltins(t *testing.T) {
	templates := listBuiltins()

	if len(templates) == 0 {
		t.Fatal("listBuiltins() returned empty list")
	}

	// Check that expected templates are present
	expectedNames := []string{"changelog", "standup", "sprint-report", "pr-description", "release-notes"}
	found := make(map[string]bool)
	for _, tmpl := range templates {
		found[tmpl.Name] = true
		if tmpl.Source != "built-in" {
			t.Errorf("listBuiltins() template %q Source = %q, want %q", tmpl.Name, tmpl.Source, "built-in")
		}
	}

	for _, name := range expectedNames {
		if !found[name] {
			t.Errorf("listBuiltins() missing expected template %q", name)
		}
	}
}
