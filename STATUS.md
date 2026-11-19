# STATUS

Phase: 4
Phase name: Hardening and 1.0.0
Gate: G4 not requested
Updated: 2026-11-18

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Start Phase 4 in a new session: read the Phase 4 section of docs/PLAN.md and P4-01
   to P4-04 in docs/acceptance.json.
2. Performance check with 10,000 seeded tasks: a one-off measurement in Phase 3 gave
   44 ms to load stats and 1 ms to compute; make it a repeatable check, and measure
   startup to first frame (Constraints).
3. Error paths, including an export directory that is not writable (export already
   shows the error; add a test that proves it).

## Blockers

None.

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|

## Gate request

None.

## Handoff

- Phase 3 closed 2026-11-18 as v0.3.0; detail in docs/archive/phase-3.md.
- Start Phase 4 in a new session.
- Phase 3 was built on branch `phase-3` in worktree .claude/worktrees/phase-3 and
  fast-forwarded into main; the worktree and branch can be removed once main has the
  release commit and tag (`git worktree remove .claude/worktrees/phase-3`,
  `git branch -d phase-3`).
- Alarm heard and notification seen on macOS only; Ubuntu untested (Phase 4 E2E).
- internal/alarm implements tui.Alarm; main wires it (bell via tea.RawMsg, Close deferred).
- Lock: cmd/unitock/lock.go. Too small: tui.go `tooSmall` (keys except ctrl+c ignored).
- Stats: internal/stats (pure); store.Completed/All share `load`; tui/stats.go, chart.go;
  navigation in tui.go `updateStats`. Export: internal/export; `e` on Stats.
- Visuals: internal/tui/timer.go, chart.go; goldens in internal/tui/testdata and
  internal/export/testdata (`make golden`).
- TUI tests: harness in start_test.go (send/press/typeText/run, fakeAlarm); teatest
  flows in flow_test.go; `pressS` in stats_nav_test.go.
