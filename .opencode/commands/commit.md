---
description: Generate a conventional commit message from staged changes
model: deepseek/deepseek-v4-flash
subtask: true
---

Generate a conventional commit message for these staged changes:

!`git diff --staged`

Format: `<type>(<scope>): <subject>`. Types: feat, fix, refactor, test, chore, docs.
Keep the subject under 72 characters. If there are multiple logical changes, suggest splitting the commit.
