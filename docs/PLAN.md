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

## Phase 0 — Foundations

- **Goal:** an approved spec, plan, and working toolchain, with no product code yet.
- **Depends on:** nothing.
- **Tasks:**
  - [x] Interview the human about goals, users, constraints, edge cases, and what is
        out of scope. Keep asking until the hard parts are covered. Write
        `docs/SPEC.md`.
  - [x] Choose the language profile. Set `profiles` and `version_mirrors` in
        `.scaffold.json`. Delete unused profile directories.
  - [x] Fill the Commands table in `AGENTS.md` and run every command to prove it works.
  - [x] Copy the profile's `ci.yml` to `.github/workflows/ci.yml` and its `starter/`
        files into place; create the layout the profile describes.
  - [x] Draft the remaining phases below and their criteria in `docs/acceptance.json`.
  - [x] Record design choices in `docs/decisions/DECISIONS.md` as `Proposed` and open
        questions in `docs/RFIS.md`.
  - [x] Replace the placeholders in `README.md` and the Project section of `AGENTS.md`.
- **Exit criteria:** P0-01 to P0-05.
- **Gate G0:** azntaiji approves the spec, the plan, and every decision the next
  phase depends on.

## Phase 1 — Walking skeleton

- **Goal:** start a named, timed task, watch it count down, complete it, and have it
  survive a quit and relaunch, all stored in SQLite and green in CI on macOS and Ubuntu.
- **Depends on:** D-03, D-04, D-05, D-07, D-08; RFI-02 before the G1 release.
- **Tasks:**
  - [ ] `internal/duration`: parser per D-08 with table tests and a fuzz test (R-10, R-11).
  - [ ] `internal/task`: pure timer maths (estimate, elapsed, remaining, overtime) from
        stored timestamps and an injected clock (R-04, D-07).
  - [ ] `internal/store`: SQLite store with embedded migrations, one-active-task index,
        create/get-active/complete operations; tests against `t.TempDir()` (R-02, R-03).
  - [ ] `cmd/unitock`: `--data-dir` (XDG default) and `--version` flags, exit codes,
        `unitock.log` (Interfaces).
  - [ ] Start screen: inputs, button, focus rules, validation (R-12 to R-16).
  - [ ] Minimal Active screen: name and `MM:SS` countdown ticking each second; `e`
        completes with a motivational message, then Start (R-19, R-23 with the full list).
  - [ ] Resume an active task on launch (R-05, without the alert); `q`/`ctrl+c` (R-06);
        key-hint bar (R-07).
  - [ ] A `teatest` test that types a task, starts it, advances a fake clock, completes
        it, and checks the stored row; and one that quits and relaunches mid-task.
  - [ ] CI green on the default branch on both OSes.
- **Exit criteria:** P1-01 to P1-07.
- **Gate G1:** azntaiji. Expected version 0.1.0.

## Phase 2 — Active screen and alert

- **Goal:** the full Active screen: visuals, pause, extend, cancel, and the time-up alert
  with notification and looping sound.
- **Depends on:** D-09, D-10, D-12, D-13.
- **Tasks:**
  - [ ] Block-digit timer, progress bar, colour bands, PAUSED and overtime display,
        `NO_COLOR` (R-17, R-18, D-09); golden-file view tests per band.
  - [ ] Pause/resume stored as pause rows (R-20).
  - [ ] Extend with the duration input (R-21).
  - [ ] Cancel confirmation (R-22).
  - [ ] Time-up alert modal with Extend / Mark as completed and the D-10 rule (R-30 to
        R-32); fires immediately when relaunched in overtime (R-05).
  - [ ] Notifier and alarm player behind interfaces, with the OS implementations and
        fallbacks (R-33, R-34, D-13).
  - [ ] Single-instance lock (R-09, D-12) and minimum terminal size (R-08).
- **Exit criteria:** P2-01 to P2-07.
- **Gate G2:** azntaiji. Expected version 0.2.0.

## Phase 3 — Stats and export

- **Goal:** the Stats screen with periods, scorecards, and the actual-vs-estimated chart,
  plus JSON export.
- **Depends on:** D-11, D-14.
- **Tasks:**
  - [ ] `internal/stats`: periods, buckets, previous-period comparison, per D-11, as
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
- **Depends on:** RFI-02 answered; the human's decision to declare 1.0.0.
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
