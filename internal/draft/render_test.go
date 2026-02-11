package draft

import (
	"strings"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
)

func TestRender(t *testing.T) {
	tmpl := &Template{
		Name:        "test",
		Description: "Test template",
		Content:     "Entries: {{entry_count}}\nRepo: {{repo_name}}\nBranch: {{branch}}\n\n{{entries_summary}}",
	}

	entries := []*ledger.Entry{
		{
			ID:        "tb_2026-01-18_abc123",
			CreatedAt: time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC),
			Summary: ledger.Summary{
				What: "Added feature X",
				Why:  "User requested it",
				How:  "Implemented via API",
			},
		},
	}

	ctx := &RenderContext{
		Entries:  entries,
		RepoName: "test-repo",
		Branch:   "main",
	}

	result, err := Render(tmpl, ctx)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(result, "Entries: 1") {
		t.Errorf("Render() result missing entry_count, got: %s", result)
	}

	if !strings.Contains(result, "Repo: test-repo") {
		t.Errorf("Render() result missing repo_name, got: %s", result)
	}

	if !strings.Contains(result, "Branch: main") {
		t.Errorf("Render() result missing branch, got: %s", result)
	}

	if !strings.Contains(result, "Added feature X") {
		t.Errorf("Render() result missing entry summary, got: %s", result)
	}
}

func TestRenderWithAppend(t *testing.T) {
	tmpl := &Template{
		Name:    "test",
		Content: "Base content",
	}

	ctx := &RenderContext{
		Entries:    []*ledger.Entry{},
		AppendText: "Extra instructions here",
	}

	result, err := Render(tmpl, ctx)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(result, "Base content") {
		t.Errorf("Render() result missing base content, got: %s", result)
	}

	if !strings.Contains(result, "Additional Instructions") {
		t.Errorf("Render() result missing append header, got: %s", result)
	}

	if !strings.Contains(result, "Extra instructions here") {
		t.Errorf("Render() result missing append text, got: %s", result)
	}
}

func TestBuildEntriesSummary(t *testing.T) {
	entries := []*ledger.Entry{
		{
			ID:        "tb_2026-01-18_abc123",
			CreatedAt: time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC),
			Summary: ledger.Summary{
				What: "First entry",
				Why:  "Reason one",
			},
		},
		{
			ID:        "tb_2026-01-17_def456",
			CreatedAt: time.Date(2026, 1, 17, 15, 0, 0, 0, time.UTC),
			Summary: ledger.Summary{
				What: "Second entry",
				Why:  "Reason two",
			},
		},
	}

	result := buildEntriesSummary(entries)

	if !strings.Contains(result, "2026-01-18") {
		t.Errorf("buildEntriesSummary() missing date, got: %s", result)
	}

	if !strings.Contains(result, "First entry") {
		t.Errorf("buildEntriesSummary() missing first entry what, got: %s", result)
	}

	if !strings.Contains(result, "Reason one") {
		t.Errorf("buildEntriesSummary() missing first entry why, got: %s", result)
	}

	lines := strings.Split(result, "\n")
	if len(lines) != 2 {
		t.Errorf("buildEntriesSummary() expected 2 lines, got %d", len(lines))
	}
}

func TestBuildDateRange(t *testing.T) {
	tests := []struct {
		name    string
		entries []*ledger.Entry
		want    string
	}{
		{
			name:    "no entries",
			entries: []*ledger.Entry{},
			want:    "no entries",
		},
		{
			name: "single entry",
			entries: []*ledger.Entry{
				{CreatedAt: time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)},
			},
			want: "2026-01-18",
		},
		{
			name: "same day",
			entries: []*ledger.Entry{
				{CreatedAt: time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)},
				{CreatedAt: time.Date(2026, 1, 18, 15, 0, 0, 0, time.UTC)},
			},
			want: "2026-01-18",
		},
		{
			name: "different days",
			entries: []*ledger.Entry{
				{CreatedAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)},
				{CreatedAt: time.Date(2026, 1, 18, 15, 0, 0, 0, time.UTC)},
			},
			want: "2026-01-15 to 2026-01-18",
		},
		{
			name: "zero times",
			entries: []*ledger.Entry{
				{CreatedAt: time.Time{}},
			},
			want: "unknown date range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildDateRange(tt.entries)
			if got != tt.want {
				t.Errorf("buildDateRange() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderWithEntriesJSON(t *testing.T) {
	tmpl := &Template{
		Name:    "test",
		Content: "JSON: {{entries_json}}",
	}

	entries := []*ledger.Entry{
		{
			ID: "tb_test_123",
			Summary: ledger.Summary{
				What: "Test entry",
				Why:  "Testing",
				How:  "Via test",
			},
		},
	}

	ctx := &RenderContext{
		Entries: entries,
	}

	result, err := Render(tmpl, ctx)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(result, `"id": "tb_test_123"`) {
		t.Errorf("Render() result missing JSON id, got: %s", result)
	}

	if !strings.Contains(result, `"what": "Test entry"`) {
		t.Errorf("Render() result missing JSON what, got: %s", result)
	}
}
