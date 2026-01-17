// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestSkillCmd_DefaultMarkdown(t *testing.T) {
	// Reset global flag
	jsonFlag = false

	cmd := newSkillCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Should contain core sections
	requiredSections := []string{
		"# Timbers",
		"## Core Concepts",
		"## Workflow Patterns",
		"## Command Reference",
		"## Contract",
	}

	for _, section := range requiredSections {
		if !strings.Contains(output, section) {
			t.Errorf("expected output to contain %q", section)
		}
	}
}

func TestSkillCmd_JSONFormat(t *testing.T) {
	// Reset global flag
	jsonFlag = false

	cmd := newSkillCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--format", "json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be valid JSON
	var result skillResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	// Check required fields
	if result.Concepts.Definition == "" {
		t.Error("expected concepts.definition to be non-empty")
	}
	if len(result.Workflow.Phases) == 0 {
		t.Error("expected workflow.phases to be non-empty")
	}
	if len(result.Commands) == 0 {
		t.Error("expected commands to be non-empty")
	}
	if result.Contract.Schema == "" {
		t.Error("expected contract.schema to be non-empty")
	}
}

func TestSkillCmd_GlobalJSONFlag(t *testing.T) {
	// Test that --json global flag also outputs JSON
	jsonFlag = true
	defer func() { jsonFlag = false }()

	cmd := newSkillCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be valid JSON
	var result skillResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestSkillCmd_IncludeExamples(t *testing.T) {
	// Reset global flag
	jsonFlag = false

	cmd := newSkillCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--include-examples"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Should contain examples section (markdown format: **Examples**:)
	if !strings.Contains(output, "**Examples**:") {
		t.Error("expected output to contain examples when --include-examples is set")
	}
}

func TestSkillCmd_IncludeExamplesJSON(t *testing.T) {
	// Reset global flag
	jsonFlag = false

	cmd := newSkillCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--format", "json", "--include-examples"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result skillResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Check that commands have examples
	hasExamples := false
	for _, cmd := range result.Commands {
		if len(cmd.Examples) > 0 {
			hasExamples = true
			break
		}
	}
	if !hasExamples {
		t.Error("expected at least one command to have examples when --include-examples is set")
	}
}

func TestSkillCmd_InvalidFormat(t *testing.T) {
	// Reset global flag
	jsonFlag = false

	cmd := newSkillCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--format", "xml"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid format")
	}

	if !strings.Contains(err.Error(), "md") || !strings.Contains(err.Error(), "json") {
		t.Errorf("expected error to mention valid formats, got: %v", err)
	}
}

func TestSkillCmd_ContainsCommands(t *testing.T) {
	// Reset global flag
	jsonFlag = false

	cmd := newSkillCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--format", "json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result skillResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Check that essential commands are documented
	expectedCmds := []string{"log", "pending", "prime", "query", "status", "show", "export"}
	cmdMap := make(map[string]bool)
	for _, c := range result.Commands {
		cmdMap[c.Name] = true
	}

	for _, expected := range expectedCmds {
		if !cmdMap[expected] {
			t.Errorf("expected command %q to be documented", expected)
		}
	}
}

func TestSkillCmd_ContractExitCodes(t *testing.T) {
	// Reset global flag
	jsonFlag = false

	cmd := newSkillCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--format", "json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result skillResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Check that exit codes are documented
	if len(result.Contract.ExitCodes) == 0 {
		t.Error("expected contract.exit_codes to be non-empty")
	}

	// Check for standard exit codes
	exitCodeMap := make(map[int]bool)
	for _, ec := range result.Contract.ExitCodes {
		exitCodeMap[ec.Code] = true
	}

	expectedCodes := []int{0, 1, 2, 3}
	for _, code := range expectedCodes {
		if !exitCodeMap[code] {
			t.Errorf("expected exit code %d to be documented", code)
		}
	}
}
