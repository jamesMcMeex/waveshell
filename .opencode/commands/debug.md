---
description: Investigate a failure without making changes
agent: debug
subtask: true
---

Investigate this issue: $ARGUMENTS

Current build state:
!`go build ./... 2>&1`

Read relevant files, run targeted tests, and form a hypothesis. Report findings and suggested fixes — do not make changes.
