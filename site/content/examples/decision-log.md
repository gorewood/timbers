+++
title = 'Decision Log'
date = '2026-05-27'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Detect push-before-log instead of blocking; codify commit→log→never-push-between ordering

**Status:** Accepted
**Date:** 2026-05-20

**Context:** A push-before-log race in osprey-strike stranded a ledger entry locally — a documented commit reached upstream before `timbers log` ran, and the entry never followed. The protocol said "commit, then log" but never bolded the rule that nothing should be pushed between those steps, and `timbers log` stayed silent even though the upstream state needed to catch the mistake was already in hand.

**Decision:** Surface the race with a warning rather than enforce it with a block. `IsPushedToUpstream` (in `internal/git`) compares the docs anchor against `@{u}` after `WriteEntry`; `printer.Warn` fires when the documented commit is already on upstream but its entry is not. The protocol text was rewritten to lead with explicit commit→log ordering and a "never push between them" callout.

**Consequences:**
- Agents get an immediate signal at the moment the entry is stranded, using data already available — no extra git round-trips.
- A warning informs but does not prevent; an agent can still push between commit and log and proceed past it.
- The "never push between" rule is now explicit in the protocol doc rather than implied by step order.

## ADR-2: Compose protocol text from shared consts in `internal/protocol` over a single-file doc

**Status:** Accepted
**Date:** 2026-05-20

**Context:** The commit that codified commit→log ordering needed its protocol/workflow text consumed by two surfaces — `cmd/timbers` emits the full PRIME document, while `internal/mcp` needs only a subset. The question was where that text should live.

**Decision:** Reject the single-file approach. The workflow text is fundamentally compositional — different consumers need different subsets — so shared protocol and stale-anchor sections moved into an `internal/protocol` package, composed by each consumer. Const+const concatenation gives compile-time composition with no runtime concat overhead.

**Consequences:**
- Both consumers draw from one source, so the full doc and the MCP subset cannot drift apart.
- New shared sections (e.g. `RebaseRelinkGuidance`, added later — see ADR-9) slot in as additional consts wired into each compose site.
- Composition is fixed at compile time; dynamically selecting subsets at runtime is not supported by this shape.

## ADR-3: Provide `timbers ack` for honest skip-with-reason over fabricated entries or `--no-verify`

**Status:** Accepted
**Date:** 2026-05-20

**Context:** The v0.22.0 parallel-agent work (osprey-strike feedback) surfaced commits an agent legitimately won't document — merge SHAs, bot autofixes from the impending q-redshifted pipeline. An agent facing the commit gate had only bad options: fabricate a hollow ledger entry, or bypass the hook with `--no-verify`.

**Decision:** Add `timbers ack` — a structured skip-with-reason recorded as `kind="ack"` under `timbers.devlog/v1` in `.timbers/YYYY/MM/DD/ack_*.json`. `AckedSet` threads through `filterCommits` parallel to `docSet`, within the same single scan per pending check.

**Consequences:**
- A skipped commit leaves an auditable record with a stated reason instead of a fabricated entry or an untracked `--no-verify` bypass.
- The pending gate honors acks, so acked commits stop showing as pending.
- ack becomes a first-class disposition alongside "documented"; later work (ADR-9) reuses it for the rebase-relink case.

## ADR-4: Resolve merge-topology pending friction by fixing `--batch` anchor selection and filtering coverage, not the docSet algorithm

**Status:** Accepted
**Date:** 2026-05-21

**Context:** Laura's v0.22.0 friction read as "pending scrambling" against a side-branch merge. The maintainer's initial plan was a Phase 1 algorithm change — making the latest-entry anchor first-parent-aware. An independent reviewer flagged that plan as a regression: it would lose the anchor entirely for the common "entry on a feature branch, then merge" workflow already exercised by `TestBranchMerge_EntryOnBranch_NoneOnMain`.

**Decision:** Drop the algorithm change — the docSet algorithm was correct; the friction was opacity plus filtering coverage. Ship a diagnostic-only Phase 0 (v0.22.1) — `IsOnFirstParentLine` / `LatestAnchorOffFirstParent`, wired into `pending` and `doctor` — to surface the topology and soak data before changing behavior; the Phase 0 contract test passing on unchanged code was the empirical signal that Phase 1 was unnecessary. The actual root causes, fixed together in v0.22.2, came from two reviewers: `--batch` naively picked `commits[0]`, which could be a side-branch SHA (fixed by `pickBatchAnchor`, which prefers the first commit on HEAD's first-parent line and falls back to `commits[0]` for the pure cross-agent case); and `filterByRules` used `docSet` only for revert detection, never for direct "is this commit documented" membership (fixed by a direct `docSet[commit.SHA]` check, paired with an off-first-parent gate fallback). Both v0.22.2 fixes were required together — the gate fallback alone returned unfiltered reachable commits because direct membership wasn't yet a thing.

**Consequences:**
- Laura's merge-topology friction class reaches zero pending without touching the core algorithm.
- "documented" now surfaces as a distinct classify-reason in the `TIMBERS_DEBUG` trace and the auto-skip count.
- `pickBatchAnchor` resolves HEAD best-effort via `git.HEAD()` and degrades to legacy behavior on git error, so a batch run never fails on anchor resolution.
- The residual case — an off-first-parent anchor at *zero* detected pending — is left unaddressed here and handled later (see ADR-11).

## ADR-5: Extend `.timbersignore` with `author:` and `msg:` identity globs as the single skip-config source

**Status:** Accepted
**Date:** 2026-05-27

**Context:** The q-redshifted autofix pipeline needed a way to skip bot-authored commits, and release changelog commits were polluting pending. The first implementation put skip-authors in a dedicated `.timbers/skip-authors` file; the user pushed back on file sprawl.

**Decision:** Fold identity-based skipping into `.timbersignore` via line prefixes, mirroring the `.gitignore` family for a single source of truth. `author:<glob>` came first (v0.22.0); `msg:<glob>` followed for commit-subject matching. Message-based skipping was chosen over a path-based skip for release commits specifically because `filterByRules` requires *every* file in a commit to match — and release changelog commits also touch `site/layouts/index.html` (the version badge), so a path rule would hide legitimate landing-page edits. `msg:chore: changelog for v*` skips them precisely. `.timbersignore` was allowlisted in `.gitignore` so every clone and CI reads the same config.

**Consequences:**
- One file holds all repo skip config in `.gitignore`-idiomatic form, rather than a proliferation of per-concern files.
- `msg:` rules generalize to other housekeeping subjects (version bumps, release commits) where path rules cannot be precise.
- `filepath.Match` character-class semantics make `author:dependabot[bot]` a silent no-op glob; a `doctor` lint now flags literal-looking `[..]` classes in `author:`/`msg:` globs, and the canonical bot recipe uses a prefix wildcard. Discoverability surfaces (pending hint, `--explain`, `help timbersignore`, onboard blurb) were added so the lever doesn't require source-diving.
- The `author:`/`msg:` prefixes collide with literal paths beginning with those strings — accepted as extremely rare (`:` is forbidden in Windows filenames).

## ADR-6: Migrate beads from server-mode Dolt + JSONL-in-git to embedded Dolt + canonical `refs/dolt/data` sync

**Status:** Accepted
**Date:** 2026-05-27

**Context:** The repo's beads setup was in a broken hybrid state. Server-mode Dolt couldn't persist under the sandbox — it re-imported JSONL on every call because the localhost TCP it needs was blocked. A stale CLI remote and a three-week-old `refs/dolt/data` existed while the steering docs claimed "no remote." JSONL was the only current source of truth (server Dolt last modified Mar 22; `refs/dolt/data` topped out May 7).

**Decision:** Align to the canonical embedded model already validated in osprey-strike. Embedded Dolt at `.beads/embeddeddolt/` (gitignored) replaces the server; sync moves to `refs/dolt/data` on origin via `bd dolt push`/`pull`; `.beads/issues.jsonl` becomes a passive, gitignored export rather than git transport. Because JSONL was the only trustworthy state, rebuild-from-JSONL via `bd bootstrap` was safe and correct despite losing Dolt audit history. The stale `refs/dolt/data` was dropped and re-seeded fresh.

**Consequences:**
- Beads works under the sandbox — embedded mode avoids the TCP the server required.
- Bead state syncs invisibly to PR diffs via `refs/dolt/data`; `metadata.json` (`dolt_mode: embedded`) and `config.yaml` are committed so fresh clones inherit the model via `bd bootstrap`.
- Dolt audit history prior to the migration is lost — the rebuild started from the 60-issue JSONL snapshot.
- Bootstrap detection order matters: `refs/dolt/data` had to be dropped *and* `.beads/backup` set aside so bootstrap fell through to current JSONL instead of cloning stale state.
- Hand-edits to the Sync / Landing-the-Plane sections landed inside bd's managed markers; relocating them is tracked separately (timbers-069).

## ADR-7: `just release` polls the GitHub release endpoint and installs the published artifact as a normal consumer

**Status:** Accepted
**Date:** 2026-05-27

**Context:** After tagging, the locally built binary is byte-identical to what CI publishes, but installing the local build verifies nothing about the release pipeline — CI build, GitHub release creation, `install.sh`. The user's intent was to validate the release "as if a normal consumer."

**Decision:** Append a bounded poll loop to `just release` after `git push` — `gh release view <tag>` until assets appear (~10m deadline, 15s interval) — then run `just install-release` to install the published build and verify the installed `--version` contains the tag. A cron follow-up was considered (the user's initial framing) but rejected: a poll-and-install at the tail of `just release` has no moving parts, and CI can't install on the dev's laptop anyway, so the step has to be local. Relies on `.goreleaser draft:false` so `latest` equals the just-pushed tag once assets publish.

**Consequences:**
- Each release exercises the full publish→install path end to end, and the local binary stays current with no manual follow-up.
- On poll timeout the release is already pushed, so the recipe prints a recovery hint rather than failing.
- `install-release` (not a pinned `VERSION`) remains the real consumer path under test.

## ADR-8: Add a `doctor` Binary Shadowing check instead of debugging stale-hook commit blocks reactively

**Status:** Accepted
**Date:** 2026-05-27

**Context:** Discovered live during the v0.22.3 release — a `dev` go-install binary in mise's GOBIN shadowed the current `~/.local/bin` build on PATH. The git hook runs whatever `timbers` is first on PATH, so the stale binary didn't recognize acks and silently gate-blocked an ack's auto-commit, surfacing only as a confusing "failed to commit ack file."

**Decision:** Surface the divergence proactively. `doctor` enumerates `timbers` binaries on PATH (deduped by resolved path) and warns when the first reports a different version *token* than a shadowed one. Comparison is on the version token, not the full `--version` string, so two builds of the same version with differing commit/date suffixes don't false-warn; `?` tokens are skipped to avoid false positives on broken shims. `WriteAck`'s commit-failure path now explains the staged-but-uncommitted state and points at upgrade + doctor + `git commit` recovery.

**Consequences:**
- A shadowing stale binary is caught by a health check instead of through a misleading commit failure.
- Token-level comparison trades exactness for fewer false alarms — two genuinely different builds of the same version number won't be flagged.
- The ack commit-failure error now guides recovery rather than reporting a bare failure.

## ADR-9: Satisfy rebase-relink by documenting the existing `ack` path, deferring content-matching

**Status:** Accepted
**Date:** 2026-05-27

**Context:** A five-proposal feature request asked for "auto content-match relink" (the requester's preferred option A) so a ledger entry survives a rebase that rewrites its anchor SHA. The key correction: timbers has no content-matching today — `docSet` is literal SHAs — so option A would be built from scratch, and its naive byte-identical-diff/reflog form is unsound (rebases shift context lines; reflog is absent on fresh clones and CI). A sound version needs patch-id stored at log time, i.e. a schema change.

**Decision:** `ack` already produces a structured record that counts as documented, so it satisfies the rebase-relink requirement today. Invert the requester's preference order to D→B→maybe-A per Gall's Law: ship documentation now (a `RebaseRelinkGuidance` protocol section in `internal/protocol`, wired into both prime and MCP compose sites — see ADR-2), graduate to a typed `ack --to <entry>` only if the free-text link bites, and build patch-id gating only with real volume evidence. A copy-pasteable `rebased; content in <entry-id>` ack reason replaces the generic placeholder.

**Consequences:**
- The rebase-relink need is met with documentation and an existing command — no new code paths or schema changes shipped.
- The relink is a free-text convention, not an enforced/typed link; correctness depends on the operator pasting the right entry id.
- Distinct from the existing stale-anchor guidance: there the anchor is GC'd and self-heals, whereas a rebased anchor stays reachable and does not.
- A protocol test locks the new section in place.

## ADR-10: Make `just release` site-example regeneration non-fatal so an LLM hiccup can't block a tag

**Status:** Accepted
**Date:** 2026-05-27

**Context:** A version release was aborted entirely when the ancillary site-content step (`just examples`, which fans out to parallel `claude -p` calls) flaked on the decision-log example. A transient LLM hiccup in non-critical content generation was blocking the shipping of a tag.

**Decision:** Wrap `just examples` in the release recipe with `|| echo WARNING` so a failure warns and the release proceeds to commit/tag/push. Examples regenerate separately.

**Consequences:**
- The critical release path (commit/tag/push) is decoupled from best-effort content generation and can't be held hostage by it.
- Site examples may briefly lag the tag when generation flakes, and must be regenerated out of band.

## ADR-11: Honor `timbers log --anchor` at zero pending and name the off-first-parent situation

**Status:** Accepted
**Date:** 2026-05-28

**Context:** The v0.22.2 fixes (ADR-4) got the normal merge-topology case to zero pending but left a residual osprey hit: when the anchor sits off the first-parent line and pending is 0, `timbers log --anchor` still refused — contradicting the flag's name, which promises "use this anchor" — and `timbers pending` printed a bare "No pending commits" that conflated a genuinely clean tree with one computed from an off-line anchor.

**Decision:** Honor the flag. At zero detected pending with `--anchor` set, `getLogCommits` falls back to `LogRange(anchor^, anchor)` to document the single commit, and the refusal path now names `--range` as the explicit escape hatch. Separately, `timbers pending`'s count:0 branch prints a conditional note keyed on `AnchorOffFirstParentLine` that names the situation and points at `--explain`/`--range`.

**Consequences:**
- `--anchor` does what its name promises even at zero pending, closing the off-first-parent/zero-pending gap.
- "Clean" and "computed from an off-line anchor" are now distinguishable in pending output rather than collapsed into one message.
- Commit-resolution helpers were extracted to `log_resolve.go` to stay under the file-length limit; a ref-aware mock and `TestLogAnchorBypassesZeroPending` lock the behavior in.
