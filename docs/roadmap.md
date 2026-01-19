# Timbers Development Roadmap

Development plan following the philosophy: **grow complexity from a simple system that already works**.

Each epic builds on proven foundations. No epic begins until prerequisites are working and tested.

---

## Epic 0: Foundation Infrastructure

**Goal**: Minimal CLI that builds, tests, lints, and handles output correctly.

### 0.1 CLI Skeleton
- [ ] Set up Cobra root command with version/help
- [ ] Add `--json` global flag infrastructure
- [ ] Add `--help` that shows command groups
- [ ] Verify: `just run --version` outputs version

### 0.2 Output Infrastructure
- [ ] Create `internal/output` package
- [ ] Implement JSON output helper (structured success/error)
- [ ] Implement human output helper with lipgloss styling
- [ ] Add TTY detection (disable colors when piped)
- [ ] Implement structured error type with code + message + hint
- [ ] Test: JSON errors have correct structure

### 0.3 Exit Code Infrastructure
- [ ] Define exit code constants (0=success, 1=user, 2=system, 3=conflict)
- [ ] Wire exit codes through Cobra's error handling
- [ ] Test: commands return correct exit codes

**Acceptance**: `timbers --version` and `timbers --help` work. `--json` on any command outputs structured JSON or error. Exit codes are correct.

---

## Epic 1: Git Operations Layer

**Goal**: Reliable git operations via exec with proper error handling.

### 1.1 Git Exec Wrapper
- [ ] Create `internal/git` package
- [ ] Implement `Run(args ...string) (string, error)` wrapper
- [ ] Handle git not found, not a repo, command failures
- [ ] Map git errors to appropriate exit codes
- [ ] Test: wrapper captures stdout/stderr correctly

### 1.2 Repository Detection
- [ ] Implement `IsRepo() bool`
- [ ] Implement `RepoRoot() (string, error)`
- [ ] Implement `CurrentBranch() (string, error)`
- [ ] Implement `HEAD() (string, error)` - full SHA
- [ ] Test: works in repo, errors outside repo

### 1.3 Commit Operations
- [ ] Implement `Log(from, to string) ([]Commit, error)`
- [ ] Implement `Commit` struct (SHA, short, subject, body, author, date)
- [ ] Implement `CommitsReachableFrom(sha string) ([]Commit, error)`
- [ ] Implement `Diffstat(from, to string) (Diffstat, error)`
- [ ] Test: commit parsing handles edge cases (empty body, special chars)

**Acceptance**: Can query commits, parse output, get diffstats. All operations tested.

---

## Epic 2: Git Notes Layer

**Goal**: Read/write JSON entries to `refs/notes/timbers`.

### 2.1 Notes Detection
- [ ] Implement `NotesRefExists() bool`
- [ ] Implement `NotesConfigured() bool` (checks remote fetch config)
- [ ] Test: detects presence/absence of notes ref

### 2.2 Notes Reading
- [ ] Implement `ReadNote(commit string) ([]byte, error)`
- [ ] Implement `ListNotedCommits() ([]string, error)`
- [ ] Handle note not found vs other errors
- [ ] Test: reads notes, handles missing notes

### 2.3 Notes Writing
- [ ] Implement `WriteNote(commit, content string) error`
- [ ] Implement `--force` behavior for overwrites
- [ ] Test: writes notes, detects conflicts

### 2.4 Notes Sync Helpers
- [ ] Implement `ConfigureNotesFetch(remote string) error`
- [ ] Implement `PushNotes(remote string) error`
- [ ] Implement `FetchNotes(remote string) error`
- [ ] Test: configures fetch spec correctly

**Acceptance**: Can read/write JSON to git notes. Notes sync operations work.

---

## Epic 3: Entry Schema

**Goal**: Type-safe entry struct with validation and ID generation.

### 3.1 Entry Struct
- [ ] Create `internal/ledger` package
- [ ] Define `Entry` struct matching schema
- [ ] Define `Workset` struct (anchor, commits, range, diffstat)
- [ ] Define `Summary` struct (what, why, how)
- [ ] Define `WorkItem` struct (system, id)
- [ ] Add JSON tags for serialization

### 3.2 Entry Validation
- [ ] Implement `Validate() error` on Entry
- [ ] Validate required fields (schema, kind, id, timestamps, workset.anchor, workset.commits, summary.*)
- [ ] Return structured errors with field names
- [ ] Test: validation catches missing/invalid fields

### 3.3 Entry ID Generation
- [ ] Implement `GenerateID(anchor string, timestamp time.Time) string`
- [ ] Format: `tb_<ISO8601>_<short-sha>`
- [ ] Test: same inputs = same output (determinism)

### 3.4 Entry Serialization
- [ ] Implement `ToJSON() ([]byte, error)`
- [ ] Implement `FromJSON(data []byte) (*Entry, error)`
- [ ] Test: round-trip serialization preserves data

**Acceptance**: Entry struct serializes/deserializes correctly. Validation catches errors. IDs are deterministic.

---

## Epic 4: Status Command

**Goal**: First working command - shows repo and notes state.

### 4.1 Status Logic
- [ ] Create `cmd/timbers/status.go`
- [ ] Add `status` command to Cobra
- [ ] Gather: repo name, branch, HEAD, notes ref, notes configured, entry count
- [ ] Wire up `--json` flag

### 4.2 Status Output
- [ ] Human output: formatted status display
- [ ] JSON output: `{repo, branch, head, notes_ref, notes_configured, entry_count}`
- [ ] Handle not-a-repo error with exit code 2
- [ ] Test: outputs correct structure

**Acceptance**: `timbers status` and `timbers status --json` work. Correct exit codes.

---

## Epic 5: Ledger Storage

**Goal**: Read/write entries to notes, find latest entry.

### 5.1 Storage Read
- [ ] Create `internal/ledger/storage.go`
- [ ] Implement `ReadEntry(anchor string) (*Entry, error)`
- [ ] Implement `ListEntries() ([]*Entry, error)`
- [ ] Implement `GetLatestEntry() (*Entry, error)`
- [ ] Test: reads entries, handles empty ledger

### 5.2 Storage Write
- [ ] Implement `WriteEntry(entry *Entry) error`
- [ ] Validate entry before writing
- [ ] Handle conflict (entry exists for anchor)
- [ ] Support `--force` overwrite
- [ ] Test: writes entries, detects conflicts

### 5.3 Since-Last-Entry Algorithm
- [ ] Implement `GetPendingCommits() ([]Commit, *Entry, error)`
- [ ] Find latest entry by `created_at`
- [ ] Return commits from anchor (exclusive) to HEAD (inclusive)
- [ ] Handle no entries (return all commits from HEAD)
- [ ] Test: algorithm handles various scenarios

**Acceptance**: Can read/write entries. Since-last-entry algorithm works correctly.

---

## Epic 6: Pending Command

**Goal**: Show undocumented commits - the "clear next action" command.

### 6.1 Pending Logic
- [ ] Create `cmd/timbers/pending.go`
- [ ] Add `pending` command to Cobra
- [ ] Use `GetPendingCommits()` from storage
- [ ] Support `--count` flag (count only)
- [ ] Wire up `--json` flag

### 6.2 Pending Output
- [ ] Human output: list commits with subjects, show suggested command
- [ ] JSON output: `{count, last_entry, commits[]}`
- [ ] Handle empty pending (0 commits)
- [ ] Test: correct output formats

**Acceptance**: `timbers pending` shows undocumented commits with next action. JSON output works.

---

## Epic 7: Log Command

**Goal**: The primary command - record work as ledger entry.

### 7.1 Log Flags
- [ ] Create `cmd/timbers/log.go`
- [ ] Add `log` command with positional `<what>` argument
- [ ] Add `--why` flag (required unless --minor)
- [ ] Add `--how` flag (required unless --minor)
- [ ] Add `--tag` flag (repeatable)
- [ ] Add `--work-item` flag (repeatable, format: `system:id`)
- [ ] Add `--range` flag (explicit commit range)
- [ ] Add `--anchor` flag (override anchor)
- [ ] Add `--minor` flag (trivial change defaults)
- [ ] Add `--dry-run` flag
- [ ] Add `--push` flag

### 7.2 Log Validation
- [ ] Validate required fields based on flags
- [ ] Parse `--work-item` format
- [ ] Validate `--range` format
- [ ] Return structured errors with hints

### 7.3 Log Execution
- [ ] Build Entry from flags and git data
- [ ] Gather commits (from range or since-last-entry)
- [ ] Gather diffstat
- [ ] Generate ID
- [ ] Write to notes (respecting --dry-run)
- [ ] Optionally push notes (--push)

### 7.4 Log Output
- [ ] Human output: confirmation with ID
- [ ] JSON output: `{status, id, anchor, commits}`
- [ ] Dry-run output: what would be written
- [ ] Test: full workflow creates valid entry

**Acceptance**: `timbers log "what" --why "why" --how "how"` creates entry. All flags work. Dry-run works.

---

## Epic 8: Show Command

**Goal**: Display a single entry by ID or `--last`.

### 8.1 Show Logic
- [ ] Create `cmd/timbers/show.go`
- [ ] Add `show` command with optional `<id>` argument
- [ ] Add `--last` flag (most recent entry)
- [ ] Wire up `--json` flag
- [ ] Handle entry not found

### 8.2 Show Output
- [ ] Human output: formatted entry display
- [ ] JSON output: full entry object
- [ ] Test: shows entries correctly

**Acceptance**: `timbers show <id>` and `timbers show --last` work.

---

## Epic 9: Query Command

**Goal**: Search and retrieve entries (M1 scope: `--last N` only).

### 9.1 Query Logic
- [ ] Create `cmd/timbers/query.go`
- [ ] Add `query` command
- [ ] Add `--last` flag (required for M1)
- [ ] Add `--oneline` flag (compact format)
- [ ] Wire up `--json` flag

### 9.2 Query Output
- [ ] Human output: list of entries
- [ ] Oneline output: compact format
- [ ] JSON output: array of entries
- [ ] Test: query returns correct entries

**Acceptance**: `timbers query --last 5` works with all output formats.

---

## Epic 10: Export Command

**Goal**: Export entries for pipeline consumption.

### 10.1 Export Logic
- [ ] Create `cmd/timbers/export.go`
- [ ] Create `internal/export` package
- [ ] Add `export` command
- [ ] Add `--last` flag
- [ ] Add `--range` flag
- [ ] Add `--format` flag (json|md)
- [ ] Add `--out` flag (directory output)

### 10.2 JSON Export
- [ ] Implement JSON array export to stdout
- [ ] Implement JSON files export to directory
- [ ] Test: JSON is valid and complete

### 10.3 Markdown Export
- [ ] Implement markdown with YAML frontmatter
- [ ] Include all entry fields in frontmatter
- [ ] Format body with what/why/how sections
- [ ] Include evidence section (commits, diffstat)
- [ ] Test: markdown is well-formed

**Acceptance**: `timbers export --last 5 --json | jq` works. Markdown export creates valid files.

---

## Epic 11: Prime Command

**Goal**: Context injection for session bootstrapping.

### 11.1 Prime Logic
- [ ] Create `cmd/timbers/prime.go`
- [ ] Add `prime` command
- [ ] Gather: repo info, last N entries, pending count
- [ ] Wire up `--json` flag

### 11.2 Prime Output
- [ ] Human output: workflow context with suggested commands
- [ ] JSON output: structured context object
- [ ] Test: output is useful for session start

**Acceptance**: `timbers prime` provides useful session context.

---

## Epic 12: Skill Command

**Goal**: Self-documentation for building agent skills.

### 12.1 Skill Content
- [ ] Create `cmd/timbers/skill.go`
- [ ] Embed skill content (core concepts, workflows, commands)
- [ ] Add `--format` flag (md|json)
- [ ] Add `--include-examples` flag

### 12.2 Skill Output
- [ ] Markdown output: documentation for skill creation
- [ ] JSON output: structured skill data
- [ ] Include: concepts, workflow patterns, command reference, contract
- [ ] Test: output is complete and accurate

**Acceptance**: `timbers skill` outputs useful skill-building content.

---

## Epic 13: Notes Subcommands

**Goal**: Notes management for syncing.

### 13.1 Notes Init
- [ ] Create `cmd/timbers/notes.go`
- [ ] Add `notes` command with subcommands
- [ ] Implement `notes init [--remote]`
- [ ] Configure fetch refspec

### 13.2 Notes Push/Fetch
- [ ] Implement `notes push`
- [ ] Implement `notes fetch`
- [ ] Handle remote errors

### 13.3 Notes Status
- [ ] Implement `notes status`
- [ ] Show sync state, entry counts
- [ ] Wire up `--json` flag

**Acceptance**: `timbers notes init/push/fetch/status` all work.

---

## Epic 14: Auto and Batch Modes

**Goal**: Efficiency features for high-volume documentation.

### 14.1 Auto Mode
- [ ] Add `--auto` flag to `log` command
- [ ] Extract what/why/how from commit messages
- [ ] Combine commit subjects for "what"
- [ ] Extract body content for "why/how"
- [ ] Support `--yes` for non-interactive auto

### 14.2 Batch Mode
- [ ] Add `--batch` flag to `log` command
- [ ] Group pending commits by work-item trailer or by day
- [ ] Process each group as separate entry
- [ ] Test: batch creates multiple entries

**Acceptance**: `timbers log --auto` and `timbers log --batch` work.

---

## Epic 15: Polish and Integration

**Goal**: Production readiness.

### 15.1 Help Text
- [ ] Write comprehensive help for all commands
- [ ] Group commands in root help (Core, Query, Sync)
- [ ] Add examples to command help

### 15.2 Integration Tests
- [ ] Create `internal/integration` package
- [ ] Test full log → pending → query → export cycle
- [ ] Test notes init → push → fetch cycle
- [ ] Test error scenarios

### 15.3 Documentation
- [ ] Update README with final examples
- [ ] Verify CLAUDE.md workflow instructions
- [ ] Add CHANGELOG

**Acceptance**: All acceptance criteria from spec section 8 pass. Documentation is complete.

---

## Milestone Summary

| Milestone | Epics | Outcome |
|-----------|-------|---------|
| M0 | 0-3 | Infrastructure + data model |
| M1-core | 4-7 | status, pending, log (MVP) |
| M1-query | 8-10 | show, query, export |
| M1-full | 11-13 | prime, skill, notes |
| M1-complete | 14-15 | auto/batch, polish |

---

## Development Principles

1. **Each epic is independently testable** - Don't move to next epic until current one has tests passing

2. **Read-only before write** - Implement query operations before mutations

3. **Human output before JSON** - Get the UX right, then add structured output

4. **Errors are features** - Structured errors with recovery hints from the start

5. **Dogfood early** - Use timbers to document timbers development once `log` works
