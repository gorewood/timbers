package draft

import (
	"strings"
	"testing"
)

func TestParseReportProfile(t *testing.T) {
	tmpl, err := parseTemplate(`---
name: decision-digest
report:
  scope:
    last: 20
  projection: decision
  format: markdown
  quiet_output: _No decisions._
---
{{entries_json}}`)
	if err != nil {
		t.Fatalf("parseTemplate() error = %v", err)
	}
	if tmpl.Report == nil || tmpl.Report.Scope.Last != "20" {
		t.Fatalf("report scope = %#v, want last 20", tmpl.Report)
	}
	if tmpl.Report.Projection != ProjectionDecision || tmpl.Report.Format != "markdown" {
		t.Errorf("report = %#v", tmpl.Report)
	}
}

func TestRejectInvalidReportProfiles(t *testing.T) {
	tests := []struct {
		name   string
		report string
		want   string
	}{
		{"two scopes", "scope: {last: 20, since: 7d}\n  projection: decision\n  format: markdown", "exactly one"},
		{"no scope", "projection: decision\n  format: markdown", "exactly one"},
		{"projection", "scope: {last: 20}\n  projection: custom\n  format: markdown", "projection"},
		{"format", "scope: {last: 20}\n  projection: decision\n  format: html", "format"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTemplate("---\nname: test\nreport:\n  " + tt.report + "\n---\nbody")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseTemplate() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestTemplateWithoutReportRemainsValid(t *testing.T) {
	tmpl, err := parseTemplate("---\nname: draft-only\n---\nbody")
	if err != nil {
		t.Fatalf("parseTemplate() error = %v", err)
	}
	if tmpl.Report != nil {
		t.Fatalf("Report = %#v, want nil", tmpl.Report)
	}
}
