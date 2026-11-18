# STATUS

Phase: 3
Phase name: Stats and export
Gate: G3 not requested
Updated: 2026-11-18

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Chart (R-43, R-45, P3-03): vertical estimated/actual bars per bucket, legend, y-axis
   in hours and minutes, R-43 labels (`Period.Label`), empty state; goldens per period
   in internal/tui/testdata, reviewed by azntaiji before P3-03 is marked.
2. Export (R-47, R-50, D-14, P3-05): `e` on Stats; schema v1 golden; never overwrite;
   add `e export` to Stats hints then.

## Blockers

None.

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|
| P3-01 | `go test ./internal/stats/ -count=1 -v` | 15 PASS, 0 FAIL | artifacts/phase-3/p3-01-stats.txt |
| P3-02 | `go test ./... -count=1 -run 'Scorecard' -v` | 4 PASS, 0 FAIL | artifacts/phase-3/p3-02-scorecard.txt |
| P3-04 | `go test ./internal/tui/ -count=1 -run 'Stats' -v` | 7 PASS, 0 FAIL | artifacts/phase-3/p3-04-stats-nav.txt |
| perf | one-off test, 10k tasks (deleted) | Completed 44 ms, 3× Compute 1 ms | — |
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
- Stats: `stats.Compute` (pure); `store.Completed` (3 queries, migration 0004 index on
  `ended_at`); tui/stats.go loads the widest range once (3 months' previous period
  through today) and computes per render, so tab is instant.
- Navigation in tui.go `updateStats`: ticks and save results still reach activeModel
  while Stats is open over Active; the alert overlays Stats and takes keys. Choosing
  Extend there switches to Active (where the input is) — a judgement call, R-32 silent.
- Alarm heard and notification seen on macOS only; Ubuntu untested (Phase 4 E2E).
- internal/alarm implements tui.Alarm; main wires it (bell via tea.RawMsg, Close deferred).
- Lock: cmd/unitock/lock.go. Too small: tui.go `tooSmall` (keys except ctrl+c ignored).
- Visuals: internal/tui/timer.go; goldens in internal/tui/testdata (`make golden`).
- TUI tests: harness in start_test.go (send/press/typeText/run, fakeAlarm); teatest
  flows in flow_test.go. Stats helper `pressS` in stats_nav_test.go.
