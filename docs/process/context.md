# Context management

An agent's context window fills quickly and its performance drops as it fills. Each
session also starts with no memory of the last one. So the project keeps its memory in
files, and keeps those files small enough that reading them leaves room to work.

## Tiers

| Tier | Files | Read when | Budget |
|---|---|---|---|
| Always loaded | `AGENTS.md`, `CLAUDE.md` | Every session, automatically | AGENTS.md ≤ 150 lines |
| Session start | `STATUS.md`, current phase section of `docs/PLAN.md` | Every session, by rule | STATUS.md ≤ 80 lines |
| On demand | `docs/process/`, `profiles/<lang>/PROFILE.md`, `docs/SPEC.md`, `docs/decisions/DECISIONS.md`, `docs/RFIS.md`, `docs/acceptance.json` | The task touches them | Read the relevant section |
| Cold | `docs/archive/`, `docs/decisions/archive.md`, `artifacts/`, `CHANGELOG.md` history, git history | A task names them, or you need past reasoning | Search first, then read a slice |

`scripts/check_scaffold.py` enforces the budgets and fails CI when they are exceeded.

## Where information goes

| Information | Home | Form |
|---|---|---|
| What is happening now | `STATUS.md` | Rewritten in place |
| What changed in the code | Commit messages | Conventional Commits |
| Raw command, test, or validation output | `artifacts/<phase>/<name>.log` | Unedited; gitignored by default |
| Proof a check passed | `STATUS.md` Evidence table | One line: check, command, result, path |
| A design choice | `docs/decisions/DECISIONS.md` | One entry |
| An unanswered question | `docs/RFIS.md` | One row |
| User-visible change | `CHANGELOG.md` `[Unreleased]` | One bullet |
| A finished phase | `docs/archive/phase-N.md` | Moved at gate close |

If information has no home in this table, it probably belongs in a commit message or
does not need to be kept. Do not create a new file for it. The scaffold check rejects
unlisted Markdown files in the repository root and report-style files under `docs/`.

## Working habits

- Search before reading, and read the section you need instead of the whole file.
- Do not print large outputs into the conversation. Redirect to `artifacts/` and look at
  the failing part.
- Do not re-read a file you have not changed since you last read it.
- Delegate wide reads. Use a subagent, or a separate session if the harness has none,
  for codebase exploration, log triage, and reviews. Ask it for conclusions with
  `file:line` references, not file contents.
- Keep `AGENTS.md` lean. Before adding a line, ask whether removing it would cause a
  mistake. Language detail belongs in the profile; procedures belong in this directory.

## Session boundaries

Start a new session:

- for each phase, and after a gate closes;
- when switching to an unrelated task;
- after two failed attempts at the same fix, with what you learned written into the
  STATUS.md handoff;
- when the context is mostly old tool output.

Before ending any session, leave the tree clean and update the Handoff section of
STATUS.md. Compaction by the harness does not replace this: a summary of a long session
passes on less than a short, deliberate handoff does.

## Compaction at gate close

A closed gate is the point where phase detail stops being useful to the next session.
The gate close routine in `gates.md` does the following:

1. Move the phase's gate request, evidence table, completed tasks, and notes to
   `docs/archive/phase-N.md`.
2. Replace that phase's section in `docs/PLAN.md` with three to five lines: what
   shipped, the version, and a pointer to the archive file.
3. Move `Rejected` and `Superseded` decisions to `docs/decisions/archive.md`.
4. Move RFIs whose answers now live in the spec or a decision to the archive file.
5. Rewrite `STATUS.md` for the next phase.

Nothing is deleted. Approvals stay in `GATES.md`, and everything else stays in git and
the archive.

## Harness notes

- **Claude Code:** files imported into `CLAUDE.md` with `@path` load in every session,
  so import only `AGENTS.md`. Use `/clear` between unrelated tasks and
  `/compact <what to keep>` when you must continue a long session.
- **Codex:** the combined `AGENTS.md` chain is capped (32 KiB by default) and extra
  content is dropped, which is one more reason to keep it short.
