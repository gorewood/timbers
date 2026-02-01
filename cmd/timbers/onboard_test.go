package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestOnboardCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantInOutput   []string
		wantJSONFields map[string]any
	}{
		{
			name: "default output is markdown for claude",
			args: []string{"onboard"},
			wantInOutput: []string{
				"## Development Ledger",
				"timbers prime",
				"timbers pending",
				"timbers log",
				"timbers notes push",
			},
		},
		{
			name: "target agents",
			args: []string{"onboard", "--target", "agents"},
			wantInOutput: []string{
				"## Development Ledger",
				"timbers prime",
			},
		},
		{
			name: "JSON output with --json flag",
			args: []string{"onboard", "--json"},
			wantJSONFields: map[string]any{
				"target": "claude",
				"format": "md",
			},
		},
		{
			name: "JSON output with --format json",
			args: []string{"onboard", "--format", "json"},
			wantJSONFields: map[string]any{
				"target": "claude",
				"format": "json",
			},
		},
		{
			name: "JSON output with target agents",
			args: []string{"onboard", "--json", "--target", "agents"},
			wantJSONFields: map[string]any{
				"target": "agents",
				"format": "md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			cmd := newRootCmd()
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("command failed: %v", err)
			}

			output := buf.String()

			// Check plain text output
			for _, want := range tt.wantInOutput {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nOutput: %s", want, output)
				}
			}

			// Check JSON output
			if len(tt.wantJSONFields) > 0 {
				var result map[string]any
				if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
					t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
				}

				for key, want := range tt.wantJSONFields {
					got, ok := result[key]
					if !ok {
						t.Errorf("missing field %q in output", key)
						continue
					}
					if got != want {
						t.Errorf("field %q = %v, want %v", key, got, want)
					}
				}

				// Verify snippet field exists and is non-empty
				snippet, ok := result["snippet"].(string)
				switch {
				case !ok:
					t.Error("missing or invalid 'snippet' field in JSON output")
				case snippet == "":
					t.Error("snippet field is empty")
				case !strings.Contains(snippet, "Development Ledger"):
					t.Errorf("snippet doesn't contain expected content: %s", snippet)
				}
			}
		})
	}
}

func TestOnboardInvalidTarget(t *testing.T) {
	var buf bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"onboard", "--target", "invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid target")
	}

	output := buf.String()
	if !strings.Contains(output, "invalid target") {
		t.Errorf("expected 'invalid target' in error message, got: %s", output)
	}
}

func TestOnboardInvalidFormat(t *testing.T) {
	var buf bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"onboard", "--format", "invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid format")
	}

	output := buf.String()
	if !strings.Contains(output, "invalid format") {
		t.Errorf("expected 'invalid format' in error message, got: %s", output)
	}
}

func TestOnboardJSONErrorOutput(t *testing.T) {
	var buf bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"onboard", "--json", "--target", "invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid target")
	}

	// Verify JSON error output
	var result map[string]any
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Fatalf("failed to parse JSON error output: %v\nOutput: %s", jsonErr, buf.String())
	}

	// Check error code is 1 (user error)
	code, ok := result["code"].(float64)
	if !ok {
		t.Fatalf("missing or invalid 'code' in error output: %v", result)
	}
	if code != 1 {
		t.Errorf("error code = %v, want 1 (user error)", code)
	}

	// Check error message
	errMsg, ok := result["error"].(string)
	if !ok {
		t.Fatalf("missing or invalid 'error' in error output: %v", result)
	}
	if !strings.Contains(errMsg, "invalid target") {
		t.Errorf("error message = %q, want to contain 'invalid target'", errMsg)
	}
}
