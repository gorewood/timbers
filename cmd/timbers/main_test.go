package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRootCommand_Version(t *testing.T) {
	// Set version for testing
	version = "1.2.3"

	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1.2.3") {
		t.Errorf("--version output should contain version: %q", output)
	}
	if !strings.Contains(output, "timbers") {
		t.Errorf("--version output should contain 'timbers': %q", output)
	}
}

func TestRootCommand_Help(t *testing.T) {
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Check for expected help content
	expectations := []string{
		"timbers",
		"Usage:",
		"--json",
		"--help",
	}

	for _, expected := range expectations {
		if !strings.Contains(output, expected) {
			t.Errorf("--help output should contain %q: %q", expected, output)
		}
	}
}

func TestRootCommand_JSONFlag_NoSubcommand(t *testing.T) {
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	// Should error because no subcommand is provided
	if err == nil {
		t.Fatal("Expected error when running with --json but no subcommand")
	}

	output := buf.String()

	// Should output JSON error
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Output should be valid JSON: %v\nOutput: %s", err, output)
	}

	if _, ok := result["error"]; !ok {
		t.Errorf("JSON output should contain 'error' field: %s", output)
	}
	if _, ok := result["code"]; !ok {
		t.Errorf("JSON output should contain 'code' field: %s", output)
	}
}

func TestRootCommand_JSONFlag_Persistence(t *testing.T) {
	// Verify --json flag is persistent and accessible to subcommands
	cmd := newRootCmd()

	// The flag should be persistent
	flag := cmd.PersistentFlags().Lookup("json")
	if flag == nil {
		t.Fatal("--json flag should be a persistent flag")
	}
}

func TestRootCommand_ColorFlag_Persistence(t *testing.T) {
	cmd := newRootCmd()

	flag := cmd.PersistentFlags().Lookup("color")
	if flag == nil {
		t.Fatal("--color flag should be a persistent flag")
	}
	if flag.DefValue != "auto" {
		t.Errorf("--color default = %q, want %q", flag.DefValue, "auto")
	}
}

func TestRootCommand_ColorFlag_InHelp(t *testing.T) {
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "--color") {
		t.Errorf("--help output should contain --color: %q", output)
	}
}

func TestGetColorMode(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "default is auto", args: []string{"--help"}, want: "auto"},
		{name: "never", args: []string{"--color", "never", "--help"}, want: "never"},
		{name: "always", args: []string{"--color", "always", "--help"}, want: "always"},
		{name: "auto explicit", args: []string{"--color", "auto", "--help"}, want: "auto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newRootCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			_ = cmd.Execute()

			got := getColorMode(cmd)
			if got != tt.want {
				t.Errorf("getColorMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUseColor_Never(t *testing.T) {
	// --color never should produce unstyled output even if IsTTY would be true.
	// Because we write to a buffer (non-TTY), both auto and never give false.
	// The key test: even if the underlying writer were a TTY, "never" overrides.
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--color", "never", "--help"})
	_ = cmd.Execute()

	// useColor resolves through ResolveColorMode which is tested in output_test.go.
	// Here we verify the plumbing works end-to-end.
	got := useColor(cmd)
	if got {
		t.Error("useColor() with --color never should return false")
	}
}

func TestUseColor_Always(t *testing.T) {
	// --color always should enable colors even on a non-TTY writer (buffer).
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--color", "always", "--help"})
	_ = cmd.Execute()

	got := useColor(cmd)
	if !got {
		t.Error("useColor() with --color always should return true")
	}
}

func TestExecute_WithError(t *testing.T) {
	// Test that Execute() properly returns exit codes
	// This tests the run() function behavior
	version = "test"

	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--json"}) // No subcommand = error

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for --json with no subcommand")
	}
}
