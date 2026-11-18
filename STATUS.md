# STATUS

Phase: 3
Phase name: Stats and export
Gate: G3 not requested
Updated: 2026-11-18

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Scorecards (R-41, R-42, P3-02): store query for completed tasks by `ended_at` range
   (no N+1; 10k tasks under 100 ms), Stats screen scorecards from `stats.Compute`.
2. Then chart, navigation, export, in docs/PLAN.md order.

## Blockers

None.

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|
| P3-01 | `go test ./internal/stats/ -count=1 -v` | 15 PASS, 0 FAIL | artifacts/phase-3/p3-01-stats.txt |
| check | `make check` | pass (lint 0 issues) | — |

## Gate request

None.

## Handoff

- Phase 2 closed 2026-11-18 as v0.2.0; detail in docs/archive/phase-2.md.
- Phase 3 work is on branch `phase-3` (worktree .claude/worktrees/phase-3, from local
  main ea98a8d); fast-forward main to it (`git merge --ff-only phase-3`) per D-03.
- D-15 done: migration 0003 + Go step `fillProjectKeys` (store.go `goSteps`), key from
  `foldKey` (fold.go). Store matches on `name_key`; the old NOCASE UNIQUE stays (harmless,
  stricter key subsumes it).
- Carried over: `s` (stats) on Start and Active is not handled or hinted until Stats
  exists (R-24). The alert must also overlay Stats (R-30): reuse `overlay` in active.go.
- Alarm heard and notification seen on macOS only; Ubuntu untested (Phase 4 E2E).
- internal/alarm implements tui.Alarm; main wires it (bell via tea.RawMsg, Close deferred).
- Lock: cmd/unitock/lock.go. Too small: tui.go `tooSmall` (keys except ctrl+c ignored).
- Visuals: internal/tui/timer.go; goldens in internal/tui/testdata (`make golden`).
- TUI tests: harness in start_test.go (send/press/typeText/run, fakeAlarm); teatest
  flows in flow_test.go.
