package draft

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
)

type projectedEntry struct {
	ID          string            `json:"id"`
	CreatedAt   time.Time         `json:"created_at"`
	What        string            `json:"what"`
	Why         string            `json:"why"`
	How         string            `json:"how,omitempty"`
	Notes       string            `json:"notes,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	WorkItems   []ledger.WorkItem `json:"work_items,omitempty"`
	GitSubjects []string          `json:"git_subjects,omitempty"`
}

// ProjectEntries returns the compact JSON input selected by a report profile.
func ProjectEntries(
	entries []*ledger.Entry, projection string, subjects map[string]string,
) ([]byte, error) {
	if projection != ProjectionNarrative && projection != ProjectionDecision {
		return nil, fmt.Errorf("unsupported projection %q", projection)
	}
	projected := make([]projectedEntry, 0, len(entries))
	for _, entry := range entries {
		item := projectedEntry{
			ID: entry.ID, CreatedAt: entry.CreatedAt,
			What: entry.Summary.What, Why: entry.Summary.Why,
			Notes: entry.Notes, Tags: entry.Tags, WorkItems: entry.WorkItems,
		}
		if projection == ProjectionNarrative {
			item.How = entry.Summary.How
		}
		seen := make(map[string]bool)
		for _, sha := range entry.Workset.Commits {
			subject := strings.TrimSpace(subjects[sha])
			if subject == "" || subjectRepresented(subject, item.What) || seen[subject] {
				continue
			}
			seen[subject] = true
			item.GitSubjects = append(item.GitSubjects, subject)
		}
		projected = append(projected, item)
	}
	data, err := json.MarshalIndent(projected, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal report entries: %w", err)
	}
	return data, nil
}

func subjectRepresented(subject, what string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(what)), strings.ToLower(strings.TrimSpace(subject)))
}
