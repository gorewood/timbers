package main

import (
	"strings"
	"testing"

	"github.com/gorewood/timbers/internal/draft"
	"github.com/gorewood/timbers/internal/ledger"
)

func TestParseVars(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "empty input",
			input: nil,
			want:  map[string]string{},
		},
		{
			name:  "single pair",
			input: []string{"audience=leadership"},
			want:  map[string]string{"audience": "leadership"},
		},
		{
			name:  "multiple pairs",
			input: []string{"a=1", "b=two"},
			want:  map[string]string{"a": "1", "b": "two"},
		},
		{
			name:  "empty value is valid",
			input: []string{"k="},
			want:  map[string]string{"k": ""},
		},
		{
			name:  "value with equals signs preserved",
			input: []string{"cmd=foo=bar"},
			want:  map[string]string{"cmd": "foo=bar"},
		},
		{
			name:    "missing equals",
			input:   []string{"bare"},
			wantErr: true,
		},
		{
			name:    "empty key",
			input:   []string{"=value"},
			wantErr: true,
		},
		{
			name:    "duplicate key",
			input:   []string{"k=1", "k=2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVars(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseVars() err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseVars() len = %d, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseVars()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestGenerationMetadataFrontmatterIncludesSourceAndEntries(t *testing.T) {
	tmpl := &draft.Template{Name: "decision-digest", Version: 2, Source: "project"}
	entries := []*ledger.Entry{{ID: "tb_one"}, {ID: "tb_two"}}
	metadata := buildGenerationMetadata(
		"decision-digest", tmpl, entries, "local", draftSelectionFlags{last: "2"},
	)
	if metadata.TemplateSource != "project" {
		t.Fatalf("TemplateSource = %q", metadata.TemplateSource)
	}
	frontmatter := formatTOMLFrontmatter(metadata)
	for _, want := range []string{
		`generated_template_source = "project"`,
		`generated_entry_ids = ["tb_one", "tb_two"]`,
	} {
		if !strings.Contains(frontmatter, want) {
			t.Errorf("frontmatter missing %q:\n%s", want, frontmatter)
		}
	}
}
