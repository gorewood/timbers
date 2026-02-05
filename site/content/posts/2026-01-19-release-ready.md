+++
title = 'Building a Better Release Experience'
date = '2026-01-19'
tags = ['release', 'infrastructure', 'robustness']
+++

# Building a Better Release Experience: This Week in Timbers

Last week, we shipped something we've been planning for a while: **automated releases and graceful handling of non-timbers git notes**. These might sound like small infrastructure improvements, but they represent real quality-of-life wins for both users and contributors. Let me walk you through what we built and why we made these choices.

## The Release Pipeline: From Commit to curl|bash

Getting software into users' hands shouldn't require them to be Go developers. We wanted timbers to be as easy to install as popular tools like `ripgrep` or `fd`, which meant setting up a proper release pipeline.

Here's what we built:

**GoReleaser integration** creates cross-platform binaries (Linux, macOS, Windows) automatically when we push a git tag. This is handled by a GitHub Actions workflow that triggers on version tags. We added a `.goreleaser.yaml` config that specifies which platforms we support and how binaries should be named—boring stuff, but essential infrastructure.

**The install script** (`install.sh`) lets users do the classic `curl | bash` installation, but with built-in checksum verification. This isn't just convenience; it's a security practice we inherited from tools like Rustup. The script downloads the appropriate binary for your platform and verifies its SHA256 hash against what GitHub has on record.

**Local testing** was important here. We added a `just install-local` command so contributors and maintainers can test the install script before pushing a release. This caught several edge cases early—small things like path handling differences across shells.

### A transparent tradeoff

One decision we deliberated: should we distribute via package managers (Homebrew, apt, etc.) right away? We decided to start with GitHub Releases and the install script for now. Here's why: package manager distributions add maintenance burden, and we wanted to validate the release process with real users first. We can expand to other package managers when demand is clear. If you're packaging timbers for your distribution and want to coordinate with us, we'd love to hear about it.

## Playing Nice with Other Tools

This brings us to our second improvement: **robust handling of non-timbers git notes**.

Git notes are a lesser-known feature that lets you attach metadata to commits without modifying the commit itself. It's perfect for development logs—we store each entry as a git note. But here's the reality: if someone else is also using git notes in your repository (for CI status, code review metadata, etc.), timbers was throwing errors when it encountered notes it didn't recognize.

We fixed this by:

1. **Adding schema validation** in our entry parsing. Now we explicitly check if a note is actually a timbers entry before trying to deserialize it as one.

2. **Introducing `ErrNotTimbersNote`** so callers can distinguish between "this isn't a timbers note" (fine, skip it) and "this is a timbers note but it's corrupted" (problem we should know about).

3. **Adding `ListEntriesWithStats()`** to improve the `status --verbose` command. Users can now see at a glance how many entries exist, how many are timbers entries, and whether there are any skipped or corrupted notes.

This is a small thing, but it means timbers can coexist peacefully in repositories where development happens with other tools and workflows. That's important for adoption.

### Why this matters

These aren't flashy features. They're the kind of "plumbing" work that's easy to overlook but makes a huge difference in real-world usage. We're trying to be the kind of tool that doesn't assume it owns your entire development environment.

## How to Help

We've got several areas where we'd love contributor input:

**Testing on your platform**: If you try the new install script on macOS, Linux, or Windows and run into friction, please open an issue. Edge cases in shell scripts are real.

**Package manager coordination**: Maintaining timbers in Homebrew, or another package manager? Reach out—let's make sure we're aligned on versioning and update cadence.

**Documentation**: Our README now has an Installation section, but if you think setup instructions could be clearer, send a PR. New contributor perspectives are invaluable here.

**Robustness improvements**: Have you hit edge cases with git notes? Found a scenario where timbers doesn't handle mixed note sources gracefully? We want to hear about it.

## What's Next

We're settling into a good rhythm on the fundamentals. The next phases will likely focus on expanding timbers' capabilities for larger projects and teams. But for now, we're confident in the release infrastructure and coexistence model.

Thanks to everyone who's contributed, reported issues, or just tried timbers out. Building tools in the open is collaborative, and we really mean that—your feedback shapes where we go.
