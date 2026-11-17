# AGENTS.md

Instructions for any coding agent working in this repository. This file loads into every
session, so it holds only rules and pointers. Procedures live in `docs/process/` and are
read when the task calls for them.

## Project

<PROJECT_SUMMARY: two or three sentences on what this is, who uses it, and what "done"
means. Written in Phase 0.>

## Commands

Copied from `profiles/<lang>/PROFILE.md` in Phase 0 and verified by running each one.

| Task | Command |
|---|---|
| Setup | `<fill>` |
| Format | `<fill>` |
| Lint | `<fill>` |
| Typecheck | `<fill>` |
| Test (all) | `<fill>` |
| Test (single) | `<fill>` |
| Build | `<fill>` |
| Scaffold check | `python3 scripts/check_scaffold.py` |

## Start of every session

1. Read `STATUS.md`: current phase, next tasks, blockers, handoff notes.
2. Read the current phase's section of `docs/PLAN.md`, plus only the decisions and RFIs
   that STATUS.md or that section cite.
3. Run `git status` and `git log --oneline -15`.
4. Run the test command. If the tree is broken, fix that before starting new work.

Do not read `docs/archive/`, `artifacts/`, or `CHANGELOG.md` history unless the task names
them or you need the reasoning behind a past decision.

## How work proceeds

- Work is divided into phases in `docs/PLAN.md`. Each phase ends at a gate that a named
  human approves.
- Take one task at a time: explore, plan, implement, verify. If you could describe the
  diff in one sentence, skip the plan.
- A task is done when a check you ran passed. Report the command and its result; do not
  report success you did not observe.
- Fix root causes. Do not weaken, skip, or delete a test or an acceptance criterion to
  get a pass.
- Stay inside the task. Note unrelated problems in STATUS.md instead of fixing them.
- End each task with the tree clean: checks pass, STATUS.md is current, and the work is
  committed locally.
- If the same fix has failed twice, stop. Write what you tried and learned in the
  STATUS.md handoff and recommend a fresh session.

## Reserved for humans

These rules exist so that a human, not the agent, is accountable for scope and releases.

- Gate and release decisions are the human's. Append a line to `GATES.md` only as
  "Recording a decision" in `docs/process/gates.md` directs: an explicit decision in the
  human's own message, then their confirmation of the exact line. Never edit or delete
  an existing line.
- Do not start work that belongs to phase N+1 until `GATES.md` shows GN as APPROVED.
- Build only on decisions marked `Settled` or `Approved` in
  `docs/decisions/DECISIONS.md`. For `Proposed` or `Open`, stop and ask.
- Never answer an RFI in `docs/RFIS.md` yourself. If it is blocking, finish the
  independent work, then stop and ask.
- In `docs/acceptance.json`, change only `passes` and `evidence`. New or reworded
  criteria need human approval.
- Never publish packages or change `VERSION` without an approved version in `GATES.md`.
- Create and push a git tag only as the release routine in `docs/process/gates.md`
  directs: `vX.Y.Z` for a version APPROVED in `GATES.md`, on its release commit. Never
  move, delete, or re-push an existing tag.

A checkpoint is a smaller stop inside a phase, for a decision or an RFI. Ask, then carry
on with unrelated tasks in the same phase while you wait.

## Context hygiene

Context is the scarce resource. Keep what the next session needs and nothing else.
Full rules: `docs/process/context.md`.

- `STATUS.md` is working state, not a log. Rewrite it in place and keep it under 80 lines.
- Write raw command and test output to `artifacts/<phase>/`. Record one line of evidence
  in STATUS.md: check, command, result, path.
- Do not create new summary, report, progress, or notes files. Update the existing file
  that owns that information.
- Use a subagent, or a separate session where the harness has none, for broad codebase
  exploration, log triage, and the pre-gate review. Bring back conclusions only.
- Start a new session for each phase and after a gate closes.
- Before ending a session, update the Handoff section of STATUS.md.

## Gates

At the end of a phase follow `docs/process/gates.md`. In short: run every check, record
evidence, get a fresh-context review, propose the version bump, write the gate request
in STATUS.md, commit, and end your turn with:

> Phase N complete. Gate GN awaiting approval from <names>. Needed from you: <list>.

When the human replies with a decision, record it as that document describes. After an
approval, run the gate close routine in the same turn. It applies the approved version,
updates the README, archives the phase, commits, tags the release, and pushes.

## Versioning and commits

- Semantic Versioning 2.0.0. `VERSION` is the source of truth. Rules and the bump table:
  `docs/process/versioning.md`.
- Commit locally at the end of each task: `git add -A`, then `git commit` with a
  Conventional Commits 1.0.0 message. No co-author or attribution trailers.
- Never push, force-push, or rebase yourself. Pushing happens only at gate close,
  through the `yeet` skill, after the gate is APPROVED in `GATES.md`. Invoke `yeet` as
  the last skill of your turn. One exception: after `yeet` has pushed a release, push
  its tag with `git push origin vX.Y.Z`.
- Keep `CHANGELOG.md` `[Unreleased]` current as you complete user-visible changes.

## README

`README.md` describes what exists today. Update it at gate close, and whenever setup or
usage commands change. Rules and badge formats: `docs/process/readme.md`.

## Where things live

| Need | File |
|---|---|
| Current state | `STATUS.md` |
| Phases, tasks, exit criteria | `docs/PLAN.md` |
| Requirements | `docs/SPEC.md` |
| Acceptance criteria | `docs/acceptance.json` |
| Decisions | `docs/decisions/DECISIONS.md` |
| Open questions | `docs/RFIS.md` |
| Gate approvals | `GATES.md` |
| Language conventions | `profiles/<lang>/PROFILE.md` |
| Finished phases | `docs/archive/` |
