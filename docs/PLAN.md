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

## Phase 3 — Stats and export

- **Goal:** the Stats screen with periods, scorecards, and the actual-vs-estimated chart,
  plus JSON export.
- **Depends on:** D-11, D-14.
- **Tasks:**
  - [x] Unicode case-insensitive project matching per amended D-15 (R-25).
  - [x] `internal/stats`: periods, buckets, previous-period comparison, per D-11, as
        pure functions with table tests including DST and month-end dates (R-40 to R-44).
  - [ ] Scorecards (R-41, R-42).
  - [ ] Vertical bar chart with legend, y-axis, and per-period labels (R-43, R-45).
  - [ ] Navigation: `s` from Start and Active, `esc` back, alert overlay on Stats
        (R-24, R-30, R-46).
  - [ ] Export to JSON schema v1 (R-47, R-50, D-14) with a golden-file test.
- **Exit criteria:** P3-01 to P3-05.
- **Gate G3:** azntaiji. Expected version 0.3.0.

## Phase 4 — Hardening and 1.0.0

- **Goal:** prove the whole app end to end on both platforms and release 1.0.0.
- **Depends on:** the human's decision to declare 1.0.0.
- **Tasks:**
  - [ ] Run the SPEC end-to-end verification on macOS and Ubuntu; record results.
  - [ ] Performance check with 10,000 seeded tasks (Constraints).
  - [ ] Error paths: unwritable data dir, corrupt database, missing audio and notify
        tools, export directory not writable.
  - [ ] README usage section, install instructions, and a screenshot or recording.
  - [ ] `make release-check` passes; release notes complete.
- **Exit criteria:** P4-01 to P4-04.
- **Gate G4:** azntaiji. Proposed version 1.0.0.

## Out of scope

Everything in the SPEC Non-goals: multiple or concurrent tasks, task lists, projects,
tags, editing or deleting history, Pomodoro cycles, sync or network access, Windows, a
GUI, a configuration file, localisation, and data import.
