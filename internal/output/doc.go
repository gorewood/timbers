// Package output provides structured output handling for the timbers CLI.
//
// This package handles both human-readable and JSON output formats, supporting
// the agent-friendly design principle that all commands should work well for
// both human users and automated agents.
//
// # Printer
//
// The Printer is the primary interface for command output. It automatically
// handles format switching based on the --json flag and TTY detection:
//
//	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))
//
//	// For success output
//	printer.Success(map[string]any{"message": "Entry created", "id": entry.ID})
//
//	// For error output
//	printer.Error(err)
//
//	// For raw output
//	printer.Println("Some text")
//	printer.Print("Formatted: %s\n", value)
//
// # JSON Mode
//
// When JSON mode is enabled (via --json flag), all output is structured:
//
//	// Success: {"message": "...", "id": "...", ...}
//	// Error: {"error": "message", "code": N}
//
// # Styling
//
// For human-readable output, the package provides lipgloss-based styling
// that automatically disables when output is piped:
//
//	printer.styles.Error   // Red, bold
//	printer.styles.Success // Green
//	printer.styles.Warning // Yellow
//	printer.styles.Bold    // Bold
//	printer.styles.Dim     // Gray
//
// # Exit Codes
//
// The package defines standard exit codes and error types:
//
//	output.ExitSuccess     // 0: Success
//	output.ExitUserError   // 1: User error (bad args, missing fields)
//	output.ExitSystemError // 2: System error (git failed, I/O error)
//	output.ExitConflict    // 3: Conflict (entry exists, state mismatch)
//
// # Error Types
//
// Use the error constructors to create properly-coded errors:
//
//	output.NewUserError("missing required flag: --why")
//	output.NewSystemError("git command failed")
//	output.NewConflictError("entry already exists")
//
// These errors carry exit codes that are used for both JSON error output
// and process exit codes.
package output
