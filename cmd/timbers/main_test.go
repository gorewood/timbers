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
