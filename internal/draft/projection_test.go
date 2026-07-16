package draft

import (
	"strings"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
)

func TestProjectEntries(t *testing.T) {
	entry := &ledger.Entry{
		ID:        "tb_test",
		CreatedAt: time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC),
		Workset: ledger.Workset{
			Commits:  []string{"abc", "def"},
			Range:    "abc..def",
			Diffstat: &ledger.Diffstat{Files: 2, Insertions: 10},
		},
		Summary:   ledger.Summary{What: "Stored subject", Why: "Reason", How: "Approach"},
		Notes:     "Trade-off",
		Tags:      []string{"design"},
		WorkItems: []ledger.WorkItem{{System: "beads", ID: "timbers-1"}},
		Contributors: []ledger.Contributor{{
			Name: "Ada Lovelace", Email: "ada@example.com", Sources: []string{"explicit"},
		}},
	}
	subjects := map[string]string{"abc": "Stored subject", "def": "Useful current subject"}

	decision, err := ProjectEntries([]*ledger.Entry{entry}, ProjectionDecision, subjects)
	if err != nil {
		t.Fatalf("ProjectEntries(decision) error = %v", err)
	}
	text := string(decision)
	for _, absent := range []string{`"how"`, `"diffstat"`, `"range"`, `"commits"`} {
		if strings.Contains(text, absent) {
			t.Errorf("decision projection contains %s: %s", absent, text)
		}
	}
	if !strings.Contains(text, "Useful current subject") || strings.Count(text, "Stored subject") != 1 {
		t.Errorf("decision projection subjects = %s", text)
	}
	if !strings.Contains(text, `"contributors"`) || !strings.Contains(text, `"name": "Ada Lovelace"`) {
		t.Errorf("decision projection omitted contributors: %s", text)
	}

	narrative, err := ProjectEntries([]*ledger.Entry{entry}, ProjectionNarrative, subjects)
	if err != nil {
		t.Fatalf("ProjectEntries(narrative) error = %v", err)
	}
	if !strings.Contains(string(narrative), `"how": "Approach"`) {
		t.Errorf("narrative projection omitted how: %s", narrative)
	}
	if !strings.Contains(string(narrative), `"contributors"`) {
		t.Errorf("narrative projection omitted contributors: %s", narrative)
	}
}

func TestRenderEntriesJSONOverride(t *testing.T) {
	result, err := Render(&Template{Content: "{{entries_json}}"}, &RenderContext{
		Entries:     []*ledger.Entry{{ID: "full-entry"}},
		EntriesJSON: []byte(`[{"id":"compact-entry"}]`),
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if result != `[{"id":"compact-entry"}]` {
		t.Fatalf("Render() = %q", result)
	}
}

func TestProjectEntriesOmitsSubjectsContainedInBatchWhat(t *testing.T) {
	entry := &ledger.Entry{
		Summary: ledger.Summary{What: "Subject A; Subject B", Why: "Reason"},
		Workset: ledger.Workset{Commits: []string{"a", "b"}},
	}
	data, err := ProjectEntries([]*ledger.Entry{entry}, ProjectionDecision, map[string]string{
		"a": "Subject A", "b": "subject b",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "git_subjects") {
		t.Fatalf("contained subjects were duplicated: %s", data)
	}
}
