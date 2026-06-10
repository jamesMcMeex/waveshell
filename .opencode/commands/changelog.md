---
description: Draft a CHANGELOG entry from recent commits
model: deepseek/deepseek-v4-flash
subtask: true
---

Draft a CHANGELOG entry for the following commits:

!`git log --oneline $(git describe --tags --abbrev=0 2>/dev/null || git rev-list --max-parents=0 HEAD)..HEAD`

Group by: Added, Changed, Fixed. Omit chore/refactor commits unless user-visible.
