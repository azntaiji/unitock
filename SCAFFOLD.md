# Agentic coding scaffold

Scaffold version: 1.6.0

A template repository for building software with coding agents in phases, with human
approval between phases and deliberate control over what the agent has to read. It is
plain files, so it works with any agent that reads `AGENTS.md` (Codex and others) or
`CLAUDE.md` (Claude Code), and with any programming language.

This file is for the human setting up a project. Agents do not need to read it.

## Quick start

1. Create a repository from this template, or copy the files into an empty one.
2. Start your agent in the repository and send the Phase 0 prompt below.
3. Answer its questions. It writes the spec, picks a language profile, verifies the
   toolchain, and drafts the phases.
4. When it requests gate G0, review the evidence and reply `approved`. The agent shows
   the exact `GATES.md` line; confirm it, and the agent records it and closes the gate.
   (You can still write the line by hand instead.)
5. Start each later phase in a new session.

### Prompts

Phase 0, first session:

```
Read AGENTS.md and the Phase 0 section of docs/PLAN.md. I want to build: <brief
description>. Interview me in detail about goals, users, interfaces, edge cases,
constraints, and what is out of scope. Do not ask obvious questions; dig into the hard
parts. Then carry out the Phase 0 tasks.
```

Any later session:

```
Start the session as AGENTS.md describes and continue the current phase.
```

Answering a gate request (the agent confirms the exact line with you first):

```
approved
approved, version 0.4.0
approved, no release
rejected: <what must change>
release 0.3.1
```

If you wrote the approval line in GATES.md by hand:

```
G<N> is approved in GATES.md. Run the gate close routine.
```

## How a phase runs

```
session start          read STATUS.md and the current phase; run the tests
  task loop            explore, plan, implement, verify, update STATUS.md, commit locally
gate request           all checks, evidence, fresh-context review, proposed version
human approval         reply `approved` and confirm, or write the line by hand
gate close             apply version, update README, archive the phase, commit and
                       tag vX.Y.Z, invoke yeet
human publishes        confirm the yeet push; the agent then pushes the tag
new session            next phase
```

## File map

| Path | Purpose | Written by |
|---|---|---|
| `AGENTS.md` | Always-loaded rules and pointers | Human, with care |
| `CLAUDE.md` | Imports `AGENTS.md`; Claude Code specifics | Human |
| `STATUS.md` | Current working state, capped at 80 lines | Agent |
| `GATES.md` | Gate approvals and approved versions | Human, or agent after human confirms the line |
| `VERSION`, `CHANGELOG.md` | SemVer source of truth and release notes | Agent, at gate close |
| `README.md` | Project README with badges, features, roadmap | Agent, at gate close |
| `docs/PLAN.md` | Phases, tasks, exit criteria, approvers | Agent drafts, human approves |
| `docs/SPEC.md` | Requirements | Agent drafts from the interview |
| `docs/acceptance.json` | Pass/fail criteria per phase | Agent flips `passes` only |
| `docs/decisions/`, `docs/RFIS.md` | Decisions and open questions | Both |
| `docs/process/` | Procedures, read on demand | Human |
| `docs/archive/` | Finished phases, cold storage | Agent, at gate close |
| `profiles/` | Per-language commands, versioning, CI, guides, starter files | Human |
| `scripts/check_scaffold.py` | Enforcement, standard-library Python 3.9+ | Human |
| `.scaffold.json` | Limits, allowlists, version mirrors | Human |

## What is enforced, and what is convention

`scripts/check_scaffold.py` runs locally and in CI (`.github/workflows/scaffold-check.yml`)
and fails when:

- STATUS.md is at a phase whose previous gates are not approved in GATES.md;
- an approved gate has acceptance criteria that do not pass, or a passing criterion has
  no evidence;
- `VERSION` is not valid SemVer 2.0.0, differs from the last approved version, or
  disagrees with the changelog, the README badge, or a language mirror;
- `AGENTS.md`, `CLAUDE.md`, or `STATUS.md` exceeds its size budget, or `CLAUDE.md`
  imports state or archive files;
- an unlisted Markdown file appears in the repository root, or a report-style file
  appears under `docs/`;
- a `vX.Y.Z` tag names a version that is not approved in GATES.md, or sits on a commit
  whose `VERSION` differs (turn off with `check_tags` in `.scaffold.json`).

Everything else in `AGENTS.md` is convention that the agent is asked to follow. In
particular, the check cannot tell who edited `GATES.md`, so an agent-recorded approval
rests on the confirmation rule in `docs/process/gates.md`. If approvals must be
hand-signed, drop the "By reply" route, turn on branch protection, and fill in
`.github/CODEOWNERS`.

## Customizing

- **Limits and allowlists:** `.scaffold.json`.
- **A different publish skill:** replace `yeet` in `AGENTS.md`,
  `docs/process/gates.md`, and `docs/process/versioning.md`. Whatever you use must be
  invocable by the agent, accept a commit summary, ask the human to confirm, then
  commit the working tree and push. When there is nothing to commit it must still offer
  to push unpushed commits, because at a release the agent commits and tags first and
  pushes the tag only after the branch push.
- **Keeping tagging human-only:** restore the old rule in `AGENTS.md`, S-01 in
  `docs/decisions/DECISIONS.md`, and steps 6–9 of the gate close routine.
- **Phase branches instead of trunk:** decide D-03 in `docs/decisions/DECISIONS.md`.
- **Keeping raw output as audit evidence:** remove the `artifacts/` lines from
  `.gitignore`.
- **Another language:** copy `profiles/_template/`.

Change `AGENTS.md` the way you would change code: when the agent keeps making a mistake,
add or sharpen one line; when a line is not changing behavior, delete it.

## Design sources

- Anthropic, [Best practices for Claude Code](https://code.claude.com/docs/en/best-practices):
  context as the main constraint, explore-plan-implement-commit, runnable verification,
  short instruction files, fresh-context review.
- Anthropic, [Effective harnesses for long-running agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents):
  session start routine, one task at a time, clean state at the end of a session,
  machine-readable pass/fail criteria that agents may not rewrite.
- Anthropic, [Harness design for long-running application development](https://www.anthropic.com/engineering/harness-design-long-running-apps):
  separating the evaluator from the generator.
- Anthropic, [Effective context engineering for AI agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents):
  structured notes outside the context window, compaction, subagents.
- [Claude Code memory](https://code.claude.com/docs/en/memory) and
  [Codex AGENTS.md](https://developers.openai.com/codex/guides/agents-md): how the
  instruction files load and their size limits.
- [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html),
  [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/),
  [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/),
  [Shields.io](https://shields.io/).
