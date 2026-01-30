---
title: "Release Automation"
date: 2026-01-20T03:13:00Z
---

I hit one of those infuriating edge cases where your tool works perfectly in your pristine dev repo, then someone tries it in the wild and it explodes. Turns out people use git notes for *other things*. Who knew?

Timbers stores its structured entries as git notes under a specific ref. But if you've got notes from another tool—or just random experiments—the parser would choke trying to deserialize them as timbers entries. The fix was straightforward: added `ErrNotTimbersNote` and schema validation during parsing. Now `timbers status --verbose` will skip foreign notes gracefully instead of vomiting errors. **The tool should fit into existing workflows, not demand a sterile environment.**

While I was in there, I wired up `--limit` and `--group-by` flags for the `catchup` command. The original version would just batch everything by day, which is fine until you've got a week of uncommitted work and don't want to generate a novella. Now you can cap how many entries you process in one go, and choose whether to group by day or by logical work item. Small tweak, but it makes the command actually usable when you're behind.

The bigger change: **built-in LLM execution**. Before, you'd pipe `timbers prompt` output to `llm` or whatever your preferred wrapper was. It worked, but felt clunky. Now you can pass `--model` and `--provider` directly to the `prompt` command and get output inline. The plumbing hooks into the same `llm` package I use for catchup generation, so there's no extra dependency sprawl. Just less friction between intent and result.

Also stood up goreleaser and a GitHub Actions workflow. The goal is dead-simple distribution: `curl | bash` for the impatient, pre-built binaries for everyone else. Config was straightforward—goreleaser does the heavy lifting, and I threw in checksum verification in `install.sh` because security nihilism is boring. Added a `just install-local` target to smoke-test the install script before actually cutting a release, which already saved me from shipping something embarrassing.

All of this felt like yak-shaving, but the good kind. The core tool works; now it's about removing papercuts and making it easy to get your hands on.

---
*Written with AI assistance as a editing tool.*
