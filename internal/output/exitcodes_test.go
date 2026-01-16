// Package output provides structured output and error handling for the timbers CLI.
package output

import (
	"errors"
	"testing"
)

func TestExitCodeConstants(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"ExitSuccess", ExitSuccess, 0},
		{"ExitUserError", ExitUserError, 1},
		{"ExitSystemError", ExitSystemError, 2},
		{"ExitConflict", ExitConflict, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.expected)
			}
		})
	}
}

func TestExitError(t *testing.T) {
	tests := []struct {
		name         string
		err          *ExitError
		wantCode     int
		wantMessage  string
		wantErrorStr string
	}{
		{
			name:         "user error",
			err:          NewUserError("missing required flag: --why"),
			wantCode:     ExitUserError,
			wantMessage:  "missing required flag: --why",
			wantErrorStr: "missing required flag: --why",
		},
		{
			name:         "system error",
			err:          NewSystemError("git operation failed"),
			wantCode:     ExitSystemError,
			wantMessage:  "git operation failed",
			wantErrorStr: "git operation failed",
		},
		{
			name:         "conflict error",
			err:          NewConflictError("entry already exists"),
			wantCode:     ExitConflict,
			wantMessage:  "entry already exists",
			wantErrorStr: "entry already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %d, want %d", tt.err.Code, tt.wantCode)
			}
			if tt.err.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", tt.err.Message, tt.wantMessage)
			}
			if tt.err.Error() != tt.wantErrorStr {
				t.Errorf("Error() = %q, want %q", tt.err.Error(), tt.wantErrorStr)
			}
		})
	}
}

func TestExitErrorWrapping(t *testing.T) {
	underlying := errors.New("connection refused")
	err := NewSystemErrorWithCause("git fetch failed", underlying)

	if err.Code != ExitSystemError {
		t.Errorf("Code = %d, want %d", err.Code, ExitSystemError)
	}

	// Test Unwrap
	if !errors.Is(err, underlying) {
		t.Error("errors.Is should find underlying error")
	}

	// Test that Error() includes the message
	if err.Error() != "git fetch failed" {
		t.Errorf("Error() = %q, want %q", err.Error(), "git fetch failed")
	}
}

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ExitSuccess,
		},
		{
			name:     "ExitError user",
			err:      NewUserError("bad input"),
			expected: ExitUserError,
		},
		{
			name:     "ExitError system",
			err:      NewSystemError("git failed"),
			expected: ExitSystemError,
		},
		{
			name:     "ExitError conflict",
			err:      NewConflictError("duplicate"),
			expected: ExitConflict,
		},
		{
			name:     "regular error defaults to user error",
			err:      errors.New("some error"),
			expected: ExitUserError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetExitCode(tt.err)
			if got != tt.expected {
				t.Errorf("GetExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}
