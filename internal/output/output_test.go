package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrinter_JSON_Success(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf, true, false) // json=true, tty=false

	data := map[string]any{
		"status": "created",
		"id":     "tb_2026-01-15_abc123",
	}

	err := printer.Success(data)
	if err != nil {
		t.Fatalf("Success() error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["status"] != "created" {
		t.Errorf("status = %v, want %q", result["status"], "created")
	}
	if result["id"] != "tb_2026-01-15_abc123" {
		t.Errorf("id = %v, want %q", result["id"], "tb_2026-01-15_abc123")
	}
}

func TestPrinter_JSON_Error(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf, true, false) // json=true, tty=false

	exitErr := NewUserError("missing required flag: --why")
	printer.Error(exitErr)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["error"] != "missing required flag: --why" {
		t.Errorf("error = %v, want %q", result["error"], "missing required flag: --why")
	}
	if code, ok := result["code"].(float64); !ok || int(code) != ExitUserError {
		t.Errorf("code = %v, want %d", result["code"], ExitUserError)
	}
}

func TestPrinter_Human_Success(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf, false, false) // json=false, tty=false (no colors)

	data := map[string]any{
		"message": "Entry created successfully",
	}

	err := printer.Success(data)
	if err != nil {
		t.Fatalf("Success() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Entry created successfully") {
		t.Errorf("output = %q, want to contain 'Entry created successfully'", output)
	}
}

func TestPrinter_Human_Error(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf, false, false) // json=false, tty=false

	exitErr := NewUserError("missing required flag: --why")
	printer.Error(exitErr)

	output := buf.String()
	if !strings.Contains(output, "Error") {
		t.Errorf("output should contain 'Error': %q", output)
	}
	if !strings.Contains(output, "missing required flag: --why") {
		t.Errorf("output should contain error message: %q", output)
	}
}

func TestPrinter_Print(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf, false, false)

	printer.Print("Hello, %s!", "world")

	if buf.String() != "Hello, world!" {
		t.Errorf("output = %q, want %q", buf.String(), "Hello, world!")
	}
}

func TestPrinter_Println(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf, false, false)

	printer.Println("Hello")

	if buf.String() != "Hello\n" {
		t.Errorf("output = %q, want %q", buf.String(), "Hello\n")
	}
}

func TestIsTTY(t *testing.T) {
	// IsTTY on a buffer should return false
	var buf bytes.Buffer
	if IsTTY(&buf) {
		t.Error("IsTTY(buffer) should return false")
	}
}

func TestPrinter_IsJSON(t *testing.T) {
	var buf bytes.Buffer

	jsonPrinter := NewPrinter(&buf, true, false)
	if !jsonPrinter.IsJSON() {
		t.Error("IsJSON() should return true for JSON printer")
	}

	humanPrinter := NewPrinter(&buf, false, false)
	if humanPrinter.IsJSON() {
		t.Error("IsJSON() should return false for human printer")
	}
}

func TestPrinter_Warn_Human(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf, false, false)

	printer.Warn("working tree has %s", "uncommitted changes")

	output := buf.String()
	if !strings.Contains(output, "Warning") {
		t.Errorf("output should contain 'Warning': %q", output)
	}
	if !strings.Contains(output, "uncommitted changes") {
		t.Errorf("output should contain message: %q", output)
	}
}

func TestPrinter_Warn_JSON(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf, true, false)

	printer.Warn("dirty tree")

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, buf.String())
	}
	if result["warning"] != "dirty tree" {
		t.Errorf("warning = %v, want %q", result["warning"], "dirty tree")
	}
}

func TestErrorJSON_Format(t *testing.T) {
	// Verify ErrorJSON produces exact format from spec
	result := ErrorJSON("test error", ExitUserError)

	var parsed struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse ErrorJSON output: %v", err)
	}

	if parsed.Error != "test error" {
		t.Errorf("error = %q, want %q", parsed.Error, "test error")
	}
	if parsed.Code != ExitUserError {
		t.Errorf("code = %d, want %d", parsed.Code, ExitUserError)
	}
}
