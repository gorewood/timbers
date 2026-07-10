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

// TestRunClaudeStop_EmptyStdin pins the parse-bailout: an empty stdin fails to
// decode as hook JSON, so runClaudeStopWith returns nil BEFORE the pending
// check. This is why a manual `timbers hook run claude-stop` (no stdin) exits 0
// regardless of ledger state — it never runs the check. Documented so nobody
// "fixes" the bailout into a false-clean verdict (the block only fires when the
// harness supplies real stdin JSON).
func TestRunClaudeStop_EmptyStdin(t *testing.T) {
	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	err := runClaudeStopWith(stdin, &stdout, &mockPendingChecker{pending: true})
	if err != nil {
		t.Fatalf("empty stdin should not error: %v", err)
	}
	if stdout.String() != "" {
		t.Errorf("empty stdin should produce no output (parse bailout, no check), got: %s", stdout.String())
	}
}

// TestRunClaudeStop_SkipCrossAgentDebt verifies the Stop hook honors the same
// TIMBERS_SKIP_CROSS_AGENT_DEBT escape hatch as the pre-commit and post-commit
// hooks. Without this, an operator who set the env var to tame the gate in a
// parallel-agent repo still gets blocked at session end — the divergence the
// bug report hit.
func TestRunClaudeStop_SkipCrossAgentDebt(t *testing.T) {
	validInput := func(t *testing.T) *bytes.Reader {
		t.Helper()
		data, err := json.Marshal(hookInput{StopHookActive: false})
		if err != nil {
			t.Fatalf("marshal input: %v", err)
		}
		return bytes.NewReader(data)
	}

	t.Run("truthy env var bypasses the block despite pending", func(t *testing.T) {
		for _, val := range []string{"1", "true", "YES", "On"} {
			t.Run(val, func(t *testing.T) {
				t.Setenv(envSkipCrossAgentDebt, val)
				var stdout bytes.Buffer
				err := runClaudeStopWith(validInput(t), &stdout, &mockPendingChecker{pending: true})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if stdout.String() != "" {
					t.Errorf("expected no block with env var %q; got: %s", val, stdout.String())
				}
			})
		}
	})

	t.Run("falsy env var still blocks on pending", func(t *testing.T) {
		t.Setenv(envSkipCrossAgentDebt, "0")
		var stdout bytes.Buffer
		err := runClaudeStopWith(validInput(t), &stdout, &mockPendingChecker{pending: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var resp stopOutput
		if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
			t.Fatalf("expected block JSON; parse failed: %v\nraw: %s", err, stdout.String())
		}
		if resp.Decision != "block" {
			t.Errorf("decision = %q, want block", resp.Decision)
		}
	})
}
