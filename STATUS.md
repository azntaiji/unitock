# STATUS

Phase: 2
Phase name: Active screen and alert
Gate: G2 REQUESTED
Updated: 2026-11-18

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Wait for azntaiji's G2 decision; on approval run the gate close routine.

## Blockers

None.

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|
| make check (visuals) | `make check` | pass | artifacts/phase-2/check-visuals.txt |
| Gate commands (all 9) | AGENTS.md table + `make release-check` + scaffold | pass | artifacts/phase-2/gate/ |
| Gate verifies P2-01..08 | each `verify` in acceptance.json | pass | artifacts/phase-2/gate/P2-0*.txt |
| Fresh-context review | subagent per review.md | no correctness gaps; 1 left (see request) | n/a |
| make check (lock, size) | `make check` | pass | artifacts/phase-2/check-lock-size.txt |
| P2-07 lock, size | `go test ./... -count=1 -run 'Lock\|TooSmall' -v` | pass (5); two instances by hand, azntaiji 2026-11-18 | artifacts/phase-2/p2-07-lock-size.txt |
| make check (alarm) | `make check` | pass | artifacts/phase-2/check-alarm.txt |
| P2-06 alarm | `go test ./internal/alarm/ -count=1 -v` | pass (13) | artifacts/phase-2/p2-06-alarm.txt |
| make check (overlay) | `make check` | pass | artifacts/phase-2/check-overlay.txt |
| make check (alert) | `make check` | pass | artifacts/phase-2/check-alert.txt |
| P2-05 alert | `go test ./... -count=1 -run 'Alert' -v` + by hand (macOS) | pass; heard and seen by azntaiji 2026-11-18 | artifacts/phase-2/p2-05-alert.txt |
| make check (cancel) | `make check` | pass | artifacts/phase-2/check-cancel.txt |
| P2-04 cancel | `go test ./internal/tui/ -count=1 -run 'Cancel' -v` | pass | artifacts/phase-2/p2-04-cancel.txt |
| make check (extend) | `make check` | pass | artifacts/phase-2/check-extend.txt |
| P2-03 extend | `go test ./... -count=1 -run 'Extend' -v` | pass | artifacts/phase-2/p2-03-extend.txt |
| make check (green badge) | `make check` | pass | artifacts/phase-2/check-green.txt |
| make check (projects) | `make check` | pass | artifacts/phase-2/check-projects.txt |
| P2-08 projects | `go test ./... -count=1 -run 'Project' -v` | pass (11) | artifacts/phase-2/p2-08-project.txt |
| P1-04 regression | `go test ./internal/tui/ -count=1 -run 'Start' -v` | pass | artifacts/phase-2/p1-04-regression.txt |
| make check (badges) | `make check` | pass | artifacts/phase-2/check-badges.txt |
| make check (pause) | `make check` | pass | artifacts/phase-2/check-pause.txt |
| P2-02 pause | `go test ./... -count=1 -run 'Pause' -v` | pass | artifacts/phase-2/p2-02-pause.txt |
| P2-01 view goldens | `go test ./internal/tui/ -count=1 -run 'View' -v` | pass; goldens approved by azntaiji 2026-11-18 | artifacts/phase-2/p2-01-view.txt |

## Gate request

Gate: G2, phase 2 (Active screen and alert)
Approvers: azntaiji
Exit criteria: P2-01 to P2-08 all passing (artifacts/phase-2/gate/P2-0*.txt)
Review: no correctness gaps; left: SQLite NOCASE folds ASCII only, so "über" vs "Über"
  makes two projects (R-25); fixing needs a D-15 amendment, for azntaiji to decide.
Proposed version: 0.2.0 (minor: feat commits while 0.y.z)
Look at: artifacts/phase-2/gate/ (all commands and verifies), CHANGELOG [Unreleased]
Open decisions or RFIs the next phase depends on: D-11, D-14 (Proposed)

## Handoff

- Phase 1 closed 2026-11-17 as v0.1.0; detail in docs/archive/phase-1.md.
- Start Phase 2 in a new session.
- Carried over: `s` (stats) on the Start button is not handled or hinted until Stats
  exists (Phase 3).
- Visuals: internal/tui/timer.go (bands, block font, bar, badge, heading). ANSI palette
  only; root Model's `theme.noColor` comes from tea.ColorProfileMsg (ASCII or lower). Golden helper `requireGolden` in view_test.go; `make golden` rewrites.
- Schema 2 (projects, D-15). Store.Create takes a project (may be ""); Store.Projects
  lists names most recent first; Start screen filters/ranks in Go (start.go).
- Start focus begins on Project; tests press tab first (harness.startTask does).
- Extend: activeModel.ext (extendInput); task.ExtensionFor does D-10 maths.
- Alert: alert.go (timeUpAlert, checkTimeUp on tick/update/open); drawn over the
  screen with `overlay` (lipgloss Compositor) in active.go, spanning its width.
- Lock: cmd/unitock/lock.go (flock, taken after the log opens, before the store).
  Too small: tui.go `tooSmall` (only once a size is known); keys except ctrl+c ignored.
- internal/alarm implements tui.Alarm; main wires it with the bell sent as tea.RawMsg
  and defers Close (waits for tools). Heard and seen on macOS 2026-11-18; Ubuntu untested.
- Store has Create/Active/Get/Complete/Cancel/Pause/Resume/Projects/AddExtension; add extension and cancel operations
  (with tests in internal/store) as their screens need them.
- Active writes go through activeModel.save (one in flight, `busy`); results arrive as
  taskUpdatedMsg / taskCompletedMsg / saveErrMsg. Harness runs commands for `p` too.
- TUI tests: Update-level tests in internal/tui run commands only for enter/e/q/ctrl+c
  (blink and tick commands sleep); teatest tests assert via flow_test.go's `probe`.
- `unitock.db` is created 0644 by SQLite inside the 0700 dir; the spec only requires the dir mode.
