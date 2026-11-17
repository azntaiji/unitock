@AGENTS.md

## Claude Code

- When compacting, preserve: the current phase and task, the list of modified files,
  failing checks with the commands that produce them, open blocker IDs, and any pending
  gate request.
- Use plan mode for the explore and plan steps of multi-file tasks.
- Use subagents for codebase investigation, test and log triage, and the pre-gate review
  in `docs/process/review.md`.
- Do not add imports of STATUS.md, GATES.md, CHANGELOG.md, `docs/archive/`, or
  `artifacts/` to this file. Imports load into every session.
