package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// mockPendingChecker implements pendingChecker for testing.
type mockPendingChecker struct {
	pending bool
	err     error
}

func (m *mockPendingChecker) HasPendingCommits() (bool, error) {
	return m.pending, m.err
}

func TestRunClaudePreToolUse(t *testing.T) {
	tests := []struct {
		name       string
		input      hookInput
		checker    *mockPendingChecker
		wantDeny   bool
		wantSilent bool // expect no output (allow)
	}{
		{
			name: "allows non-git-commit commands",
			input: hookInput{
				ToolName:  "Bash",
				ToolInput: hookToolInput{Command: "ls -la"},
			},
			checker:    &mockPendingChecker{pending: true},
			wantSilent: true,
		},
		{
			name: "allows git commit when no pending",
			input: hookInput{
				ToolName:  "Bash",
				ToolInput: hookToolInput{Command: "git commit -m 'test'"},
			},
			checker:    &mockPendingChecker{pending: false},
			wantSilent: true,
		},
		{
			name: "denies git commit when pending",
			input: hookInput{
				ToolName:  "Bash",
				ToolInput: hookToolInput{Command: "git commit -m 'test'"},
			},
			checker:  &mockPendingChecker{pending: true},
			wantDeny: true,
		},
		{
			name: "denies git commit with flags when pending",
			input: hookInput{
				ToolName:  "Bash",
				ToolInput: hookToolInput{Command: "git add . && git commit -m 'feat: new thing'"},
			},
			checker:  &mockPendingChecker{pending: true},
			wantDeny: true,
		},
		{
			name: "allows timbers own commits",
			input: hookInput{
				ToolName:  "Bash",
				ToolInput: hookToolInput{Command: "timbers log 'test' && git commit -m 'timbers entry'"},
			},
			checker:    &mockPendingChecker{pending: true},
			wantSilent: true,
		},
		{
			name: "allows on checker error",
			input: hookInput{
				ToolName:  "Bash",
				ToolInput: hookToolInput{Command: "git commit -m 'test'"},
			},
			checker:    &mockPendingChecker{err: errors.New("storage error")},
			wantSilent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJSON, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("marshal input: %v", err)
			}

			stdin := bytes.NewReader(inputJSON)
			var stdout bytes.Buffer

			err = runClaudePreToolUseWith(stdin, &stdout, tt.checker)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := stdout.String()

			if tt.wantSilent {
				if output != "" {
					t.Errorf("expected no output, got: %s", output)
				}
				return
			}

			if tt.wantDeny {
				var resp preToolUseOutput
				if err := json.Unmarshal([]byte(output), &resp); err != nil {
					t.Fatalf("failed to parse output: %v\nraw: %s", err, output)
				}
				if resp.HookSpecificOutput.Decision != "deny" {
					t.Errorf("decision = %q, want %q", resp.HookSpecificOutput.Decision, "deny")
				}
				if resp.HookSpecificOutput.Reason == "" {
					t.Error("deny reason should not be empty")
				}
			}
		})
	}
}

func TestRunClaudePreToolUse_MalformedInput(t *testing.T) {
	stdin := strings.NewReader("not json")
	var stdout bytes.Buffer

	err := runClaudePreToolUseWith(stdin, &stdout, &mockPendingChecker{pending: true})
	if err != nil {
		t.Fatalf("malformed input should not error: %v", err)
	}
	if stdout.String() != "" {
		t.Errorf("malformed input should produce no output, got: %s", stdout.String())
	}
}

func TestRunClaudeStop(t *testing.T) {
	tests := []struct {
		name       string
		input      hookInput
		checker    *mockPendingChecker
		wantBlock  bool
		wantSilent bool
	}{
		{
			name: "blocks when pending commits exist",
			input: hookInput{
				StopHookActive: false,
			},
			checker:   &mockPendingChecker{pending: true},
			wantBlock: true,
		},
		{
			name: "allows when no pending commits",
			input: hookInput{
				StopHookActive: false,
			},
			checker:    &mockPendingChecker{pending: false},
			wantSilent: true,
		},
		{
			name: "allows when stop_hook_active to prevent loops",
			input: hookInput{
				StopHookActive: true,
			},
			checker:    &mockPendingChecker{pending: true},
			wantSilent: true,
		},
		{
			name: "allows on checker error",
			input: hookInput{
				StopHookActive: false,
			},
			checker:    &mockPendingChecker{err: errors.New("storage error")},
			wantSilent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJSON, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("marshal input: %v", err)
			}

			stdin := bytes.NewReader(inputJSON)
			var stdout bytes.Buffer

			err = runClaudeStopWith(stdin, &stdout, tt.checker)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := stdout.String()

			if tt.wantSilent {
				if output != "" {
					t.Errorf("expected no output, got: %s", output)
				}
				return
			}

			if tt.wantBlock {
				var resp stopOutput
				if err := json.Unmarshal([]byte(output), &resp); err != nil {
					t.Fatalf("failed to parse output: %v\nraw: %s", err, output)
				}
				if resp.Decision != "block" {
					t.Errorf("decision = %q, want %q", resp.Decision, "block")
				}
				if resp.Reason == "" {
					t.Error("block reason should not be empty")
				}
			}
		})
	}
}

func TestRunClaudeStop_MalformedInput(t *testing.T) {
	stdin := strings.NewReader("not json")
	var stdout bytes.Buffer

	err := runClaudeStopWith(stdin, &stdout, &mockPendingChecker{pending: true})
	if err != nil {
		t.Fatalf("malformed input should not error: %v", err)
	}
	if stdout.String() != "" {
		t.Errorf("malformed input should produce no output, got: %s", stdout.String())
	}
}

func TestIsGitCommitCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"git commit -m 'test'", true},
		{"git commit --amend", true},
		{"git add . && git commit -m 'feat'", true},
		{"git status", false},
		{"git push", false},
		{"ls -la", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			if got := isGitCommitCommand(tt.cmd); got != tt.want {
				t.Errorf("isGitCommitCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}
