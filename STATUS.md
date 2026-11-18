# STATUS

Phase: 3
Phase name: Stats and export
Gate: G3 not requested
Updated: 2026-11-18

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Ask azntaiji to decide D-11 (stats periods and buckets) and D-14 (export format),
   both Proposed; Phase 3 builds on them.
2. Ask azntaiji: fix or accept non-ASCII case matching of project names (SQLite NOCASE
   is ASCII only; "über" vs "Über" makes two projects; R-25). A fix amends D-15.
3. Then Phase 3 tasks in docs/PLAN.md, starting with `internal/stats`.

## Blockers

None.

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|

## Gate request

None.

## Handoff

- Phase 2 closed 2026-11-18 as v0.2.0; detail in docs/archive/phase-2.md.
- Start Phase 3 in a new session.
- Carried over: `s` (stats) on Start and Active is not handled or hinted until Stats
  exists (R-24). The alert must also overlay Stats (R-30): reuse `overlay` in active.go.
- Alarm heard and notification seen on macOS only; Ubuntu untested (Phase 4 E2E).
- internal/alarm implements tui.Alarm; main wires it (bell via tea.RawMsg, Close deferred).
- Lock: cmd/unitock/lock.go. Too small: tui.go `tooSmall` (keys except ctrl+c ignored).
- Visuals: internal/tui/timer.go; goldens in internal/tui/testdata (`make golden`).
- TUI tests: harness in start_test.go (send/press/typeText/run, fakeAlarm); teatest
  flows in flow_test.go.
