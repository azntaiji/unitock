# STATUS

Phase: 3
Phase name: Stats and export
Gate: G3 REQUESTED
Updated: 2026-11-18

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Wait for the G3 decision. On approval, run the gate close routine (docs/process/gates.md).

## Blockers

None. Branch `phase-3` is not yet fast-forwarded into main (human to run
`git merge --ff-only phase-3` from the main checkout before gate close).

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|
| P3-01 | `go test ./internal/stats/ -count=1 -v` | 15 PASS, 0 FAIL | artifacts/phase-3/p3-01-stats.txt |
| P3-02 | `go test ./... -count=1 -run 'Scorecard' -v` | 4 PASS, 0 FAIL | artifacts/phase-3/p3-02-scorecard.txt |
| P3-03 | `go test ./internal/tui/ -count=1 -run 'Chart' -v` | 12 PASS, 0 FAIL; goldens approved by azntaiji | artifacts/phase-3/p3-03-chart.txt |
| P3-05 | `go test ./... -count=1 -run 'Export' -v` | 9 PASS, 0 FAIL | artifacts/phase-3/p3-05-export.txt |
| P3-04 | `go test ./internal/tui/ -count=1 -run 'Stats' -v` | 11 PASS, 0 FAIL | artifacts/phase-3/p3-04-stats-nav.txt |
| gate cmds | Commands table + `make release-check` + scaffold | all rc=0 (lint 0 issues) | artifacts/phase-3/gate-*.txt |
| e2e | real binary in pty, scratch dirs: esc, s, tab, e, q | Stats shown, export written and path shown, exit 0 | — |
| review | fresh-context subagent, docs/process/review.md | no gaps; P3-04 count fixed | — |
| perf | one-off test, 10k tasks (deleted) | Completed 44 ms, 3× Compute 1 ms | — |
| check | `make check` | pass (lint 0 issues) | — |

## Gate request

Gate: G3, phase 3 (Stats and export)
Approvers: azntaiji
Exit criteria: P3-01, P3-02, P3-03, P3-04, P3-05 all passing
Review: fresh-context subagent found no gaps; one stale evidence count (P3-04) fixed
Proposed version: 0.3.0 (minor: feat commits for Stats screen, chart, export; fix for D-15)
Look at: artifacts/phase-3/, internal/tui/testdata/stats_*.golden,
  internal/export/testdata/export_v1.golden.json, CHANGELOG.md [Unreleased]
Open decisions or RFIs the next phase depends on: none

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
- Chart: tui/chart.go. Plot rows = terminal height − 14 (4..16); slot per bucket from
  width (4..16); 7-day labels split onto two lines when they do not fit. Estimated `▒`
  faint, actual `█` blue, so NO_COLOR still reads.
- Export: internal/export (`Build` pure, `Write` O_EXCL, mode 0600), golden
  internal/export/testdata/export_v1.golden.json (task 1 = SPEC example). `e` on Stats
  via tui/stats.go `runExport`; Deps.Version (main passes ldflags version) and
  Deps.ExportDir (empty = cwd at export). Result shows on the line under the title.
- Alarm heard and notification seen on macOS only; Ubuntu untested (Phase 4 E2E).
- internal/alarm implements tui.Alarm; main wires it (bell via tea.RawMsg, Close deferred).
- Lock: cmd/unitock/lock.go. Too small: tui.go `tooSmall` (keys except ctrl+c ignored).
- Visuals: internal/tui/timer.go; goldens in internal/tui/testdata (`make golden`).
- TUI tests: harness in start_test.go (send/press/typeText/run, fakeAlarm); teatest
  flows in flow_test.go. Stats helper `pressS` in stats_nav_test.go.
