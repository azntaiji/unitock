# STATUS

Phase: 2
Phase name: Active screen and alert
Gate: G2 not requested
Updated: 2026-11-17

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Phase 2 tasks in docs/PLAN.md in order, starting with the Active screen visuals
   (R-17, R-18, D-09) and golden-file view tests per band.

## Blockers

None. D-09, D-10, D-12, D-13 approved by azntaiji 2026-11-17.

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|

## Gate request

None.

## Handoff

- Phase 1 closed 2026-11-17 as v0.1.0; detail in docs/archive/phase-1.md.
- Start Phase 2 in a new session.
- Carried over: `s` (stats) on the Start button is not handled or hinted until Stats
  exists (Phase 3). Overtime shows plain `+MM:SS`; colours come with D-09.
- Store has Create/Active/Get/Complete only; add pause, extension, and cancel operations
  (with tests in internal/store) as their screens need them.
- TUI tests: Update-level tests in internal/tui run commands only for enter/e/q/ctrl+c
  (blink and tick commands sleep); teatest tests assert via flow_test.go's `probe`.
- `unitock.db` is created 0644 by SQLite inside the 0700 dir; the spec only requires the dir mode.
