# Plan

Phases, tasks, exit criteria, and gates. Agents read only the current phase's section.
When a gate closes, that phase's section is replaced by a short summary and the detail
moves to `docs/archive/phase-N.md`.

Each phase section uses this shape:

- **Goal:** one sentence.
- **Depends on:** decision and RFI IDs that must be resolved first.
- **Tasks:** a checklist of small, independently verifiable tasks.
- **Exit criteria:** IDs from `docs/acceptance.json`.
- **Gate:** who approves.

Requirement IDs (R-xx) refer to `docs/SPEC.md`.

## Phase 0 — Foundations (done, G0 approved 2026-11-17, no release)

Shipped the spec, phases 1–4 with acceptance criteria, decisions D-01 to D-14, and the
Go toolchain (Makefile, lint, GoReleaser, CI on macOS and Ubuntu). Detail:
`docs/archive/phase-0.md`.

## Phase 1 — Walking skeleton (done, G1 approved 2026-11-17, v0.1.0)

Shipped the duration parser, timer maths, SQLite store with migrations, CLI flags, and
the Start, Active (countdown), and Done screens with resume on relaunch; teatest flows
and CI green on macOS and Ubuntu. Detail: `docs/archive/phase-1.md`.

## Phase 2 — Active screen and alert (done, G2 approved 2026-11-18, v0.2.0)

Shipped the full Active screen (block-digit timer, progress bar, ANSI colour bands and
badges, `NO_COLOR`), optional projects with suggestions (D-15), pause, extend, cancel,
the time-up alert with notification and looping chime, the single-instance lock, and the
minimum terminal size. Detail: `docs/archive/phase-2.md`.

## Phase 3 — Stats and export (done, G3 approved 2026-11-18, v0.3.0)

Shipped the Stats screen (7 days, 30 days, 3 months per D-11; scorecards against the
previous period; estimated-vs-actual bar chart; alert overlay), JSON export schema v1
(D-14), and Unicode case-insensitive project names (amended D-15). Detail:
`docs/archive/phase-3.md`.

## Phase 4 — Hardening and 1.0.0

- **Goal:** prove the whole app end to end on both platforms and release 1.0.0.
- **Depends on:** the human's decision to declare 1.0.0.
- **Tasks:**
  - [ ] Run the SPEC end-to-end verification on macOS and Ubuntu; record results.
  - [x] Performance check with 10,000 seeded tasks (Constraints).
  - [x] Error paths: unwritable data dir, corrupt database, missing audio and notify
        tools, export directory not writable.
  - [ ] README usage section, install instructions, and a screenshot or recording.
  - [x] Active screen row: elapsed and estimate (+ added) instead of remaining (R-17).
  - [x] Choosable alarm sounds with a Sound screen and preview (R-51 to R-53, D-16).
  - [ ] `make release-check` passes; release notes complete.
- **Exit criteria:** P4-01 to P4-06.
- **Gate G4:** azntaiji. Proposed version 1.0.0.

## Out of scope

Everything in the SPEC Non-goals: multiple or concurrent tasks, task lists, projects,
tags, editing or deleting history, Pomodoro cycles, sync or network access, Windows, a
GUI, a configuration file, localisation, and data import.

Deferred until after 1.0.0 (azntaiji, 2026-11-19): mouse input. Every action has a key;
mouse capture would also take over the terminal's click-and-drag text selection.
