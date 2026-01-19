package ledger

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name      string
		anchor    string
		timestamp time.Time
		want      string
	}{
		{
			name:      "standard input",
			anchor:    "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f",
			timestamp: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
			want:      "tb_2026-01-15T15:04:05Z_8f2c1a",
		},
		{
			name:      "different anchor",
			anchor:    "abc123def456789012345678901234567890abcd",
			timestamp: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
			want:      "tb_2026-01-15T15:04:05Z_abc123",
		},
		{
			name:      "different timestamp",
			anchor:    "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f",
			timestamp: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
			want:      "tb_2025-12-31T23:59:59Z_8f2c1a",
		},
		{
			name:      "short anchor preserved",
			anchor:    "abcdef",
			timestamp: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
			want:      "tb_2026-01-15T15:04:05Z_abcdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateID(tt.anchor, tt.timestamp)
			if got != tt.want {
				t.Errorf("GenerateID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateID_Determinism(t *testing.T) {
	anchor := "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f"
	timestamp := time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC)

	id1 := GenerateID(anchor, timestamp)
	id2 := GenerateID(anchor, timestamp)

	if id1 != id2 {
		t.Errorf("GenerateID not deterministic: got %q and %q", id1, id2)
	}
}

func TestEntry_Validate(t *testing.T) {
	validEntry := func() *Entry {
		return &Entry{
			Schema:    SchemaVersion,
			Kind:      KindEntry,
			ID:        "tb_2026-01-15T15:04:05Z_8f2c1a",
			CreatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
			Workset: Workset{
				AnchorCommit: "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f",
				Commits:      []string{"8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f"},
			},
			Summary: Summary{
				What: "Fixed authentication bypass",
				Why:  "Input not sanitized",
				How:  "Added validation middleware",
			},
		}
	}

	tests := []struct {
		name       string
		modify     func(*Entry)
		wantErr    bool
		wantFields []string
	}{
		{
			name:    "valid entry",
			modify:  func(e *Entry) {},
			wantErr: false,
		},
		{
			name:       "missing schema",
			modify:     func(e *Entry) { e.Schema = "" },
			wantErr:    true,
			wantFields: []string{"schema"},
		},
		{
			name:       "missing kind",
			modify:     func(e *Entry) { e.Kind = "" },
			wantErr:    true,
			wantFields: []string{"kind"},
		},
		{
			name:       "missing id",
			modify:     func(e *Entry) { e.ID = "" },
			wantErr:    true,
			wantFields: []string{"id"},
		},
		{
			name:       "missing created_at",
			modify:     func(e *Entry) { e.CreatedAt = time.Time{} },
			wantErr:    true,
			wantFields: []string{"created_at"},
		},
		{
			name:       "missing updated_at",
			modify:     func(e *Entry) { e.UpdatedAt = time.Time{} },
			wantErr:    true,
			wantFields: []string{"updated_at"},
		},
		{
			name:       "missing anchor_commit",
			modify:     func(e *Entry) { e.Workset.AnchorCommit = "" },
			wantErr:    true,
			wantFields: []string{"workset.anchor_commit"},
		},
		{
			name:       "missing commits",
			modify:     func(e *Entry) { e.Workset.Commits = nil },
			wantErr:    true,
			wantFields: []string{"workset.commits"},
		},
		{
			name:       "empty commits",
			modify:     func(e *Entry) { e.Workset.Commits = []string{} },
			wantErr:    true,
			wantFields: []string{"workset.commits"},
		},
		{
			name:       "missing what",
			modify:     func(e *Entry) { e.Summary.What = "" },
			wantErr:    true,
			wantFields: []string{"summary.what"},
		},
		{
			name:       "missing why",
			modify:     func(e *Entry) { e.Summary.Why = "" },
			wantErr:    true,
			wantFields: []string{"summary.why"},
		},
		{
			name:       "missing how",
			modify:     func(e *Entry) { e.Summary.How = "" },
			wantErr:    true,
			wantFields: []string{"summary.how"},
		},
		{
			name: "multiple missing fields",
			modify: func(e *Entry) {
				e.Schema = ""
				e.Summary.What = ""
				e.Summary.Why = ""
			},
			wantErr:    true,
			wantFields: []string{"schema", "summary.what", "summary.why"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := validEntry()
			tt.modify(entry)

			err := entry.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				var valErr *ValidationError
				if !AsValidationError(err, &valErr) {
					t.Errorf("expected ValidationError, got %T", err)
					return
				}

				// Check that all expected fields are present
				for _, field := range tt.wantFields {
					found := slices.Contains(valErr.Fields, field)
					if !found {
						t.Errorf("expected field %q in error, got fields: %v", field, valErr.Fields)
					}
				}
			}
		})
	}
}

func TestEntry_ToJSON(t *testing.T) {
	entry := &Entry{
		Schema:    SchemaVersion,
		Kind:      KindEntry,
		ID:        "tb_2026-01-15T15:04:05Z_8f2c1a",
		CreatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		Workset: Workset{
			AnchorCommit: "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f",
			Commits:      []string{"8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f"},
			Range:        "abc123..8f2c1a",
			Diffstat: &Diffstat{
				Files:      3,
				Insertions: 45,
				Deletions:  12,
			},
		},
		Summary: Summary{
			What: "Fixed authentication bypass",
			Why:  "Input not sanitized",
			How:  "Added validation middleware",
		},
		Tags: []string{"security", "auth"},
		WorkItems: []WorkItem{
			{System: "beads", ID: "bd-a1b2c3"},
		},
	}

	data, err := entry.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("ToJSON() returned empty data")
	}

	// Verify it contains expected fields
	json := string(data)
	expectedFields := []string{
		`"schema":"timbers.devlog/v1"`,
		`"kind":"entry"`,
		`"id":"tb_2026-01-15T15:04:05Z_8f2c1a"`,
		`"anchor_commit":"8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f"`,
		`"what":"Fixed authentication bypass"`,
		`"why":"Input not sanitized"`,
		`"how":"Added validation middleware"`,
	}

	for _, field := range expectedFields {
		if !containsString(json, field) {
			t.Errorf("ToJSON() missing expected field: %s", field)
		}
	}
}

func TestEntry_FromJSON(t *testing.T) {
	jsonData := []byte(`{
		"schema": "timbers.devlog/v1",
		"kind": "entry",
		"id": "tb_2026-01-15T15:04:05Z_8f2c1a",
		"created_at": "2026-01-15T15:04:05Z",
		"updated_at": "2026-01-15T15:04:05Z",
		"workset": {
			"anchor_commit": "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f",
			"commits": ["8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f"],
			"range": "abc123..8f2c1a",
			"diffstat": {"files": 3, "insertions": 45, "deletions": 12}
		},
		"summary": {
			"what": "Fixed authentication bypass",
			"why": "Input not sanitized",
			"how": "Added validation middleware"
		},
		"tags": ["security", "auth"],
		"work_items": [{"system": "beads", "id": "bd-a1b2c3"}]
	}`)

	entry, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if entry.Schema != SchemaVersion {
		t.Errorf("Schema = %q, want %q", entry.Schema, SchemaVersion)
	}
	if entry.Kind != KindEntry {
		t.Errorf("Kind = %q, want %q", entry.Kind, KindEntry)
	}
	if entry.ID != "tb_2026-01-15T15:04:05Z_8f2c1a" {
		t.Errorf("ID = %q, want %q", entry.ID, "tb_2026-01-15T15:04:05Z_8f2c1a")
	}
	if entry.Workset.AnchorCommit != "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f" {
		t.Errorf("AnchorCommit = %q, want %q", entry.Workset.AnchorCommit, "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f")
	}
	if len(entry.Workset.Commits) != 1 {
		t.Errorf("Commits length = %d, want 1", len(entry.Workset.Commits))
	}
	if entry.Summary.What != "Fixed authentication bypass" {
		t.Errorf("What = %q, want %q", entry.Summary.What, "Fixed authentication bypass")
	}
	if entry.Summary.Why != "Input not sanitized" {
		t.Errorf("Why = %q, want %q", entry.Summary.Why, "Input not sanitized")
	}
	if entry.Summary.How != "Added validation middleware" {
		t.Errorf("How = %q, want %q", entry.Summary.How, "Added validation middleware")
	}
	if entry.Workset.Diffstat == nil {
		t.Error("Diffstat is nil")
	} else if entry.Workset.Diffstat.Files != 3 {
		t.Errorf("Diffstat.Files = %d, want 3", entry.Workset.Diffstat.Files)
	}
	if len(entry.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(entry.Tags))
	}
	if len(entry.WorkItems) != 1 {
		t.Errorf("WorkItems length = %d, want 1", len(entry.WorkItems))
	}
}

func TestEntry_RoundTrip(t *testing.T) {
	original := &Entry{
		Schema:    SchemaVersion,
		Kind:      KindEntry,
		ID:        "tb_2026-01-15T15:04:05Z_8f2c1a",
		CreatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 15, 15, 4, 5, 0, time.UTC),
		Workset: Workset{
			AnchorCommit: "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f",
			Commits:      []string{"8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f", "abc123"},
			Range:        "abc123..8f2c1a",
			Diffstat: &Diffstat{
				Files:      3,
				Insertions: 45,
				Deletions:  12,
			},
		},
		Summary: Summary{
			What: "Fixed authentication bypass",
			Why:  "Input not sanitized",
			How:  "Added validation middleware",
		},
		Tags: []string{"security", "auth"},
		WorkItems: []WorkItem{
			{System: "beads", ID: "bd-a1b2c3"},
		},
	}

	// Serialize
	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Deserialize
	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	// Compare key fields
	if restored.Schema != original.Schema {
		t.Errorf("Schema: got %q, want %q", restored.Schema, original.Schema)
	}
	if restored.Kind != original.Kind {
		t.Errorf("Kind: got %q, want %q", restored.Kind, original.Kind)
	}
	if restored.ID != original.ID {
		t.Errorf("ID: got %q, want %q", restored.ID, original.ID)
	}
	if !restored.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", restored.CreatedAt, original.CreatedAt)
	}
	if !restored.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf("UpdatedAt: got %v, want %v", restored.UpdatedAt, original.UpdatedAt)
	}
	if restored.Workset.AnchorCommit != original.Workset.AnchorCommit {
		t.Errorf("AnchorCommit: got %q, want %q", restored.Workset.AnchorCommit, original.Workset.AnchorCommit)
	}
	if len(restored.Workset.Commits) != len(original.Workset.Commits) {
		t.Errorf("Commits length: got %d, want %d", len(restored.Workset.Commits), len(original.Workset.Commits))
	}
	if restored.Workset.Range != original.Workset.Range {
		t.Errorf("Range: got %q, want %q", restored.Workset.Range, original.Workset.Range)
	}
	if restored.Workset.Diffstat == nil {
		t.Error("Diffstat is nil after round-trip")
	} else {
		if restored.Workset.Diffstat.Files != original.Workset.Diffstat.Files {
			t.Errorf("Diffstat.Files: got %d, want %d", restored.Workset.Diffstat.Files, original.Workset.Diffstat.Files)
		}
		if restored.Workset.Diffstat.Insertions != original.Workset.Diffstat.Insertions {
			t.Errorf("Diffstat.Insertions: got %d, want %d", restored.Workset.Diffstat.Insertions, original.Workset.Diffstat.Insertions)
		}
		if restored.Workset.Diffstat.Deletions != original.Workset.Diffstat.Deletions {
			t.Errorf("Diffstat.Deletions: got %d, want %d", restored.Workset.Diffstat.Deletions, original.Workset.Diffstat.Deletions)
		}
	}
	if restored.Summary.What != original.Summary.What {
		t.Errorf("What: got %q, want %q", restored.Summary.What, original.Summary.What)
	}
	if restored.Summary.Why != original.Summary.Why {
		t.Errorf("Why: got %q, want %q", restored.Summary.Why, original.Summary.Why)
	}
	if restored.Summary.How != original.Summary.How {
		t.Errorf("How: got %q, want %q", restored.Summary.How, original.Summary.How)
	}
	if len(restored.Tags) != len(original.Tags) {
		t.Errorf("Tags length: got %d, want %d", len(restored.Tags), len(original.Tags))
	}
	if len(restored.WorkItems) != len(original.WorkItems) {
		t.Errorf("WorkItems length: got %d, want %d", len(restored.WorkItems), len(original.WorkItems))
	}
}

func TestFromJSON_InvalidJSON(t *testing.T) {
	_, err := FromJSON([]byte("not valid json"))
	if err == nil {
		t.Error("FromJSON() expected error for invalid JSON")
	}
}

func TestFromJSON_EmptyInput(t *testing.T) {
	_, err := FromJSON([]byte{})
	if err == nil {
		t.Error("FromJSON() expected error for empty input")
	}
}

func TestFromJSON_NotTimbersSchema(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr error
	}{
		{
			name:    "different schema prefix",
			json:    `{"schema": "other.schema/v1", "kind": "entry", "id": "test"}`,
			wantErr: ErrNotTimbersNote,
		},
		{
			name:    "empty schema",
			json:    `{"schema": "", "kind": "entry", "id": "test"}`,
			wantErr: ErrNotTimbersNote,
		},
		{
			name:    "missing schema field",
			json:    `{"kind": "entry", "id": "test"}`,
			wantErr: ErrNotTimbersNote,
		},
		{
			name:    "valid timbers schema",
			json:    `{"schema": "timbers.devlog/v1", "kind": "entry", "id": "test"}`,
			wantErr: nil,
		},
		{
			name:    "future timbers schema version",
			json:    `{"schema": "timbers.devlog/v2", "kind": "entry", "id": "test"}`,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromJSON([]byte(tt.json))
			if tt.wantErr != nil {
				if err == nil {
					t.Error("FromJSON() expected error, got nil")
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("FromJSON() error = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("FromJSON() unexpected error = %v", err)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Fields:  []string{"schema", "summary.what"},
		Message: "missing required fields",
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("ValidationError.Error() returned empty string")
	}
	// Should mention the missing fields
	if !containsString(errStr, "schema") || !containsString(errStr, "summary.what") {
		t.Errorf("ValidationError.Error() = %q, expected to contain field names", errStr)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
