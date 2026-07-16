# Persisted contributor attribution

**Bead:** `timbers-ddt`
**Target release:** v0.25.0

## Evidence

Timbers v0.24.1 persists workset SHAs, summary prose, notes, tags, and work
items, but no contributor identity. Git authors are loaded into `git.Commit`
for pending/provenance decisions and discarded when `ledger.Entry` is built.
`query --json`, `draft --json`, and `{{entries_json}}` serialize stored entries
directly; none currently recovers attribution.

A local reproduction logged an entry, squashed the repository to a new root,
expired reflogs, and pruned unreachable objects. The entry remained readable,
while its stored anchor changed from a valid commit object to a missing object.
Therefore workset SHAs cannot be the downstream attribution contract.

## Public contract

Add one optional top-level field to `timbers.devlog/v1`:

```json
"contributors": [
  {
    "name": "Ada Lovelace",
    "email": "ada@example.com",
    "sources": ["git-author", "co-authored-by"]
  }
]
```

- At log time, snapshot valid Git authors and `Co-authored-by` trailer
  identities from the selected commits.
- Apply `.mailmap` through Git before persistence.
- Deduplicate case-insensitively by canonical email and sort deterministically.
- Preserve source values from the fixed vocabulary `git-author`,
  `co-authored-by`, and `explicit`.
- `--who "Name <email>"` is repeatable. If present, it **replaces** all
  automatically derived contributors; it never merges implicitly.
- `timbers amend <entry> --who ...` provides the same replacement for older
  or retroactive entries and needs no commit objects.
- Missing or malformed automatic identities are omitted without guessing.
  Malformed explicit identities are user errors.
- Timbers records identities, not organizational roles. A syntactically valid
  bot identity is retained; downstream person views must match contributors
  against a known human roster rather than infer humanity from names.
- The person committing the Timbers entry is not a contributor by default.
- Query, draft, show, export, and MCP query expose the stored field through
  ordinary entry serialization. They never join against Git.

## Compatibility

This is an additive optional v1 field. Existing entries remain valid and omit
`contributors`; no rewrite or migration is required. Readers already ignore
unknown JSON fields, and current Timbers readers deserialize an absent slice
as empty. A v2 schema would add migration cost without changing required-field
semantics, so it is not justified.

## Implementation slice

1. Parse and mailmap-normalize Co-authored-by identities with commit metadata.
2. Add the entry field and a small shared resolver for automatic/explicit
   contributors.
3. Wire log, batch log, MCP log, and amend.
4. Lock the contract with unit and integration tests, then document CLI and
   downstream consumption.

## Conduit contract

Consume only `entry.contributors`. Treat absence as unknown attribution and
show no person-level credit. Use `name`, `email`, and `sources` exactly as
stored; do not inspect `workset.commits`, Git history, entry prose, or an LLM.
Map identities to people only through Conduit's deterministic known-person
roster. This remains valid after rebase, squash, shallow clone, or pruning.
