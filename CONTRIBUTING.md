# Contributing to Timbers

## Getting Started

```bash
git clone https://github.com/gorewood/timbers.git
cd timbers
just setup
```

## Development

```bash
just check    # Lint + test (required before commit)
just fix      # Auto-fix lint issues
just run      # Run the CLI
just build    # Build binary to bin/timbers
```

`just check` must pass before any commit. No exceptions.

## Testing

- Unit tests alongside source: `foo.go` / `foo_test.go`
- Integration tests in `internal/integration/`
- Table-driven tests for multiple cases
- Run with `just test` or `just test-cover` for coverage

## For AI Agents

See `CLAUDE.md` for agent-specific development instructions, architecture details, and conventions.
