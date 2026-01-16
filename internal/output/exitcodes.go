// Package output provides structured output and error handling for the timbers CLI.
package output

import "errors"

// Exit codes following the specification:
// 0 = Success
// 1 = User error (bad args, missing fields, not found)
// 2 = System error (git failed, I/O error)
// 3 = Conflict (entry exists, state mismatch)
const (
	ExitSuccess     = 0
	ExitUserError   = 1
	ExitSystemError = 2
	ExitConflict    = 3
)

// ExitError is an error that carries an exit code for the CLI.
type ExitError struct {
	Code    int
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *ExitError) Error() string {
	return e.Message
}

// Unwrap returns the underlying cause for errors.Is/errors.As support.
func (e *ExitError) Unwrap() error {
	return e.Cause
}

// NewUserError creates an error for user-caused issues (exit code 1).
// Use for: bad arguments, missing required fields, entry not found.
func NewUserError(message string) *ExitError {
	return &ExitError{
		Code:    ExitUserError,
		Message: message,
	}
}

// NewSystemError creates an error for system failures (exit code 2).
// Use for: git operation failures, I/O errors.
func NewSystemError(message string) *ExitError {
	return &ExitError{
		Code:    ExitSystemError,
		Message: message,
	}
}

// NewSystemErrorWithCause creates a system error wrapping an underlying cause.
func NewSystemErrorWithCause(message string, cause error) *ExitError {
	return &ExitError{
		Code:    ExitSystemError,
		Message: message,
		Cause:   cause,
	}
}

// NewConflictError creates an error for conflict situations (exit code 3).
// Use for: entry already exists, state mismatches.
func NewConflictError(message string) *ExitError {
	return &ExitError{
		Code:    ExitConflict,
		Message: message,
	}
}

// GetExitCode extracts the exit code from an error.
// Returns ExitSuccess for nil, ExitUserError for non-ExitError errors.
func GetExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}

	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}

	// Default to user error for untyped errors
	return ExitUserError
}
