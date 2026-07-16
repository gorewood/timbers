# Contributor attribution

Introduced in Timbers v0.25.0, entries may contain a top-level
`contributors` array:

```json
[
  {
    "name": "Ada Lovelace",
    "email": "ada@example.com",
    "sources": ["git-author", "co-authored-by"]
  }
]
```

At `timbers log` time, Timbers snapshots commit authors and valid
`Co-authored-by` trailers. The repository `.mailmap` canonicalizes both names
and emails. Contributors are deduplicated by canonical email and sorted for
stable JSON.

`--who "Name <email>"` is repeatable and explicit: if any `--who` value is
present, those values replace all Git-derived contributors. The same flag on
`timbers amend` repairs older entries without accessing their workset commits.
Malformed automatic identities are omitted; malformed explicit values fail.
Valid bot identities are retained because Timbers records identities and does
not guess whether an identity is human.

Contributors describe who performed the logged work. They are intentionally
separate from the person or process that records and commits the Timbers entry.

The fixed source vocabulary is:

- `git-author`
- `co-authored-by`
- `explicit`

Old entries remain valid and omit `contributors`. Absence means attribution is
unknown, not that nobody contributed. Query, show, export, draft, and MCP query
serialize the stored field; none performs a post-hoc Git join.

## Downstream contract

Use only `entry.contributors` for contributor attribution. If it is absent or
empty, show no person-level credit. Do not inspect `workset.commits`, Git
history, or prose, and do not ask an LLM to infer names. Applications that need
person-only views should match stored identities against their own known-human
roster; unmatched identities, including bots, remain uncredited as people.
