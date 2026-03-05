+++
title = 'Decision Log'
date = '2026-03-04'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

No pending commits — everything is documented. The stop hook likely fired because the working tree has uncommitted modifications (the staged/unstaged changelog files), but those aren't commits. The hook's `HasPendingCommits` check may be picking up dirty tree state rather than actual undocumented commits.
