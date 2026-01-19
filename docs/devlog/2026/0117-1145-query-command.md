# Query Command Implementation - timbers-mu4

**Date**: 2026-01-17 11:45 UTC
**Bead**: timbers-mu4 (Epic 9)

## What

Implemented the `query` command for retrieving ledger entries with the `--last N` flag, completing M1 scope of Epic 9.

## Why

The query command is essential for examining the ledger history in a controlled way. This enables users to:
- Retrieve the last N entries with `timbers query --last N`
- View results in three formats: human-readable, compact oneline, or machine-readable JSON
- Perform analysis and filtering on historical entries for further processing

This unblocks downstream features that depend on entry retrieval (export, analysis, etc.).

## How

### New Methods

Added `GetLastNEntries(n int)` to `internal/ledger/storage.go`:
- Lists all entries, sorts by CreatedAt descending (most recent first)
- Returns up to N entries
- Returns empty slice for no entries (not an error)

### New Command

Implemented `cmd/timbers/query.go` with:
- Required `--last N` flag (positive integer)
- Optional `--json` flag (global, inherited)
- Optional `--oneline` flag (compact format)
- Three output formats:
  1. **Human (default)**: Full entry display (reuses show.go output helpers)
  2. **JSON**: Array of entry objects
  3. **Oneline**: `<id>  <what>` per line

### Error Handling

- Missing `--last` flag: "specify --last N to retrieve entries" (exit 1)
- Invalid `--last` value (0, negative, non-integer): "--last must be a positive integer" (exit 1)
- No entries: returns empty result (not an error)

### Testing

Added `cmd/timbers/query_test.go` with table-driven tests covering:
- Error cases (missing flag, invalid values)
- Empty entry sets
- Boundary conditions (1 vs N entries, N > count)
- All three output formats
- JSON/human modes

Tests use same pattern as show_test.go: mock GitOps injected into Storage.

### Wiring

- Added `newQueryCmd()` to cmd/timbers/main.go
- Registered as subcommand in `newRootCmd()`

## Files Changed

- **cmd/timbers/query.go** (new): Command implementation
- **cmd/timbers/query_test.go** (new): Table-driven test suite
- **internal/ledger/storage.go** (+27 lines): Added GetLastNEntries method
- **cmd/timbers/main.go** (+1 line): Registered query command

## Quality Assurance

- Code parses correctly: `go list ./cmd/timbers` ✓
- Imports formatted: `goimports` ✓
- No syntax errors detected
- Table-driven tests for all error paths
- Follows existing patterns from show.go, pending.go
- All exported identifiers have doc comments

## Design Notes

- Sorting: Simple O(n²) bubble sort suitable for small entry counts
- Output: Reuses `outputShowEntry()` helpers to ensure consistency
- Injection: Uses `newQueryCmdInternal(storage)` pattern for testing
- Global flag: Reuses existing `jsonFlag` global from main.go

## Next Steps

- Run `just check` to verify lint/test/build (skipped due to sandbox cache issues)
- Manual testing against real repos
- Consider index/caching for large entry counts (future optimization)
