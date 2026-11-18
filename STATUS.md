# STATUS

Phase: 2
Phase name: Active screen and alert
Gate: G2 not requested
Updated: 2026-11-17

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Rework Active visuals to amended D-09: ANSI palette, four badges, project/name under
   the badge (R-17). Regenerate goldens.
2. Project input with suggestions (R-12, R-13, R-25, R-26, D-15; P2-08).
3. Then Extend (R-21) and the remaining Phase 2 tasks in docs/PLAN.md in order.
3. Before G2: azntaiji to review the Active golden files in internal/tui/testdata
   (`cat` them in a terminal to see colours). P2-01 stays `passes: false` until then.

## Blockers

None. D-09, D-10, D-12, D-13 approved by azntaiji 2026-11-17; D-09 amended and D-15
(projects) approved the same day, with SPEC R-12/R-13/R-17/R-25/R-26, P2-01 reword, P2-08.

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|
| make check (visuals) | `make check` | pass | artifacts/phase-2/check-visuals.txt |
| make check (pause) | `make check` | pass | artifacts/phase-2/check-pause.txt |
| P2-02 pause | `go test ./... -count=1 -run 'Pause' -v` | pass | artifacts/phase-2/p2-02-pause.txt |
| P2-01 view goldens | `go test ./internal/tui/ -count=1 -run 'View' -v` | pass, awaiting golden review | artifacts/phase-2/p2-01-view.txt |

## Gate request

None.

## Handoff

- Phase 1 closed 2026-11-17 as v0.1.0; detail in docs/archive/phase-1.md.
- Start Phase 2 in a new session.
- Carried over: `s` (stats) on the Start button is not handled or hinted until Stats
  exists (Phase 3).
- Visuals: internal/tui/timer.go (bands, block font, bar, badge). Root Model holds a
  `theme` from tea.BackgroundColorMsg and tea.ColorProfileMsg (ASCII or lower = no
  colour). Golden helper `requireGolden` in view_test.go; `make golden` rewrites.
- Store has Create/Active/Get/Complete/Pause/Resume; add extension and cancel operations
  (with tests in internal/store) as their screens need them.
- Active writes go through activeModel.save (one in flight, `busy`); results arrive as
  taskUpdatedMsg / taskCompletedMsg / saveErrMsg. Harness runs commands for `p` too.
- TUI tests: Update-level tests in internal/tui run commands only for enter/e/q/ctrl+c
  (blink and tick commands sleep); teatest tests assert via flow_test.go's `probe`.
- `unitock.db` is created 0644 by SQLite inside the 0700 dir; the spec only requires the dir mode.
