+++
title = 'Building the Foundation'
date = '2026-01-15'
tags = ['development', 'architecture']
+++

# Timbers: Building a Development Log That Works for Humans and AI

We're excited to share progress on **Timbers**, an open-source development logging tool that bridges the gap between how developers actually work and how teams need to document that work. Over the past few weeks, we've shipped the core functionality, and we want to pull back the curtain on what we built, why we made certain tradeoffs, and where we need community help.

## The Problem We're Solving

Development work is invisible until it's documented. Developers live in their git history, their terminals, their pull requests—but team updates, reports, and knowledge sharing require manually extracting that context. Tools that try to automate this often feel like surveillance. Timbers takes a different approach: **store development entries as git notes** (your repository, your data), make logging frictionless (auto-extract from commits), and make the output useful (for reports, for LLMs, for handoff documents).

## What We Built

### Foundation: Core Git Abstraction and CLI Workflow

The first major piece was getting the fundamentals right. We built a clean abstraction layer over git operations and created a ledger schema with built-in validation. This became the backbone for four essential commands:

- **`status`** — See pending work
- **`log` / `show`** — Review recorded entries
- **`pending`** — Surface unlogged work

These commands form the core recording workflow. We're intentionally keeping this simple: the barrier to logging work should be lower than the friction of *not* logging it.

**Why this architecture?** Git notes are stored in your repository, versioned, and mergeable—they're not a black box service. By wrapping git operations carefully, we ensure Timbers stays compatible with existing workflows without adding proprietary dependencies.

### Reducing Friction: Auto-Logging and Templates

Here's where it gets interesting. We added an **auto mode** that parses commit messages to generate log entries automatically. Most developers are already writing commits; why force them to write again?

We also built **8 built-in prompt templates** for common documentation tasks: daily standups, retrospectives, status reports, and more. Combined with a resolution chain, these templates adapt to your project's context. The `prompt` command can generate LLM-ready output—useful for developers who want to draft updates with AI assistance, then refine them.

**The tradeoff here:** Auto-logging is convenient but imperfect. Commit messages aren't always detailed enough. We're leaning on the community to tell us which templates miss the mark and which ones are goldmines. If you're using Timbers, [we'd love to hear which templates save you time](https://github.com/timbers-devlog/timbers).

### Integration and Intelligence: Query, Export, Agent Support

We added three command families for different use cases:

**Query** filters entries by tags and time ranges, giving you a lightweight search interface without a database.

**Export** outputs to JSON or Markdown, enabling data pipelines—feed your logs into a team wiki, a metrics dashboard, or an analysis script.

**Agent integration** (`prime`, `skill`, `notes`) prepares Timbers for AI agents. The `prime` command injects workflow context into an agent session, so AI tools understand what you're trying to do. The `skill` command makes Timbers self-documenting—agents can introspect available commands. The `notes` subcommand syncs entries bidirectionally.

**Why prioritize agent support?** We believe the future of development tooling is collaborative—humans and AI working together. Timbers should be agent-friendly by default, not as an afterthought.

### UX and Documentation

Finally, we rewrote the README with a **problem/solution framework**: "Development work is invisible. Here's how Timbers makes it visible." We also added agent onboarding instructions and clean uninstall support.

The `prime` command now outputs workflow instructions with override support, so agents (or humans picking up the tool) understand the intended workflow immediately.

## Architectural Decisions: Tradeoffs We Made

**Why git notes instead of a separate database?**
Decentralization and data ownership. Your logs stay in your repository, versioned and mergeable. The tradeoff: notes aren't as queryable as a database, so we keep query features focused on common patterns (tag/time filters) rather than complex joins.

**Why built-in templates instead of infinite customization?**
We wanted Timbers to *ship with value*. The 8 templates cover 80% of logging needs; custom templates can come later if the community needs them. This also keeps the surface area small for new users.

**Why auto-logging instead of requiring manual entry?**
Friction is the enemy of adoption. But we know auto-logging is imperfect, which is why we made it a mode, not the only path. You can mix auto-generated entries with manual ones.

## How to Help

We need community input on several fronts:

1. **Test the templates.** Which ones are useful? Which miss the mark? [Open an issue](https://github.com/timbers-devlog/timbers) with your feedback or a PR with a new template.

2. **Share your workflow.** How are you logging work? Are the four core commands (`status`, `log`, `pending`, `show`) enough, or do you need something else? The best features come from real usage.

3. **Agent integration ideas.** If you're working with LLMs or agents, try the `prime` and `skill` commands. What context is missing? What's too verbose?

4. **Documentation.** Newcomers often see things that feel obvious to us. If something's confusing, please file an issue—we'll clarify it.

## What's Next

We're tracking a few priorities: **bead tracer support** (grouping entries by work units), **richer query syntax** for power users, and **interop with common tools** (linear, github issues, etc.). But we want to build this with the community, not for an imagined userbase.

If you're interested in development logging, collaborative AI, or just want to try a tool that respects your data, [clone the repo](https://github.com/timbers-devlog/timbers) and kick the tires. Your feedback shapes what we build next.

—The Timbers team
