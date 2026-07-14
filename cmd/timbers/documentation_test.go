package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActiveDocumentationAvoidsRetiredSyntax(t *testing.T) {
	root := filepath.Join("..", "..")
	files := []string{
		"README.md",
		"AGENTS.md",
		"docs/agent-dx-guide.md",
		"docs/agent-reference.md",
		"docs/llm-commands.md",
		"docs/publishing-artifacts.md",
		"docs/timbers-integration-guide.md",
		"docs/tutorial.md",
	}
	banned := []string{
		"timbers draft decision-log",
		"{{.",
		"{{range",
		"timbers query --tags",
		"--format markdown",
		"if jsonFlag",
		"refs/notes/timbers",
		"| Git notes |",
		"survive rebases and squash merges cleanly",
	}

	for _, name := range files {
		path := filepath.Join(root, name)
		file, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		scanner := bufio.NewScanner(file)
		for line := 1; scanner.Scan(); line++ {
			text := scanner.Text()
			for _, stale := range banned {
				if strings.Contains(text, stale) {
					t.Errorf("%s:%d contains retired syntax %q", name, line, stale)
				}
			}
			if strings.Contains(text, "timbers catchup") && !strings.Contains(text, "now-retired") {
				t.Errorf("%s:%d presents retired catchup without historical context", name, line)
			}
		}
		if err := scanner.Err(); err != nil {
			t.Errorf("scan %s: %v", name, err)
		}
		if err := file.Close(); err != nil {
			t.Errorf("close %s: %v", name, err)
		}
	}
}

func TestReadmeDocumentsCurrentReportCommand(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, required := range []string{"| `report` |", "timbers report decision-digest", "{{entries_json}}"} {
		if !strings.Contains(text, required) {
			t.Errorf("README missing current report documentation %q", required)
		}
	}
}
