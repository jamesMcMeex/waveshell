---
description: Generate a conventional commit message from staged changes
model: deepseek/deepseek-v4-flash
subtask: true
---

Generate a conventional commit message for these staged changes:

!`git diff --staged`

Format: `<type>(<scope>): <subject>`. Types: feat, fix, refactor, test, chore, docs.
Keep the subject under 72 characters. If there are multiple logical changes, suggest splitting the commit.

Then ask the user to approve the message. If they approve, run:
`!git add -A` and `!git commit -m "<message>"` with a multi-line body if needed.
