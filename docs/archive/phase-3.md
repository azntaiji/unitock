# Phase 3 — Stats and export (archived)

Gate G3 approved by azntaiji, 2026-11-18. Version: 0.3.0.

## Plan section as completed

- **Goal:** the Stats screen with periods, scorecards, and the actual-vs-estimated chart,
  plus JSON export.
- **Depends on:** D-11, D-14.
- **Tasks:**
  - [x] Unicode case-insensitive project matching per amended D-15 (R-25).
  - [x] `internal/stats`: periods, buckets, previous-period comparison, per D-11, as
        pure functions with table tests including DST and month-end dates (R-40 to R-44).
  - [x] Scorecards (R-41, R-42).
  - [x] Vertical bar chart with legend, y-axis, and per-period labels (R-43, R-45).
  - [x] Navigation: `s` from Start and Active, `esc` back, alert overlay on Stats
        (R-24, R-30, R-46).
  - [x] Export to JSON schema v1 (R-47, R-50, D-14) with a golden-file test.
- **Exit criteria:** P3-01 to P3-05.
- **Gate G3:** azntaiji. Expected version 0.3.0.

## Gate request

Gate: G3, phase 3 (Stats and export)
Approvers: azntaiji
Exit criteria: P3-01, P3-02, P3-03, P3-04, P3-05 all passing
Review: fresh-context subagent found no gaps; one stale evidence count (P3-04) fixed
Proposed version: 0.3.0 (minor: feat commits for Stats screen, chart, export; fix for D-15)
Look at: artifacts/phase-3/, internal/tui/testdata/stats_*.golden,
  internal/export/testdata/export_v1.golden.json, CHANGELOG.md [Unreleased]
Open decisions or RFIs the next phase depends on: none

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|
| P3-01 | `go test ./internal/stats/ -count=1 -v` | 15 PASS, 0 FAIL | artifacts/phase-3/p3-01-stats.txt |
| P3-02 | `go test ./... -count=1 -run 'Scorecard' -v` | 4 PASS, 0 FAIL | artifacts/phase-3/p3-02-scorecard.txt |
| P3-03 | `go test ./internal/tui/ -count=1 -run 'Chart' -v` | 12 PASS, 0 FAIL; goldens approved by azntaiji | artifacts/phase-3/p3-03-chart.txt |
| P3-04 | `go test ./internal/tui/ -count=1 -run 'Stats' -v` | 11 PASS, 0 FAIL | artifacts/phase-3/p3-04-stats-nav.txt |
| P3-05 | `go test ./... -count=1 -run 'Export' -v` | 9 PASS, 0 FAIL | artifacts/phase-3/p3-05-export.txt |
| gate cmds | Commands table + `make release-check` + scaffold | all rc=0 (lint 0 issues) | artifacts/phase-3/gate-*.txt |
| e2e | real binary in pty, scratch dirs: esc, s, tab, e, q | Stats shown, export written and path shown, exit 0 | — |
| review | fresh-context subagent, docs/process/review.md | no gaps; P3-04 count fixed | — |
| perf | one-off test, 10k tasks (deleted) | Completed 44 ms, 3× Compute 1 ms | — |

## Notes (handoff at gate)

- Phase 3 was built in worktree .claude/worktrees/phase-3 on branch `phase-3` (a
  background-job harness requirement) and fast-forwarded into main by azntaiji.
- D-15 amended: migration 0003 + Go step `fillProjectKeys` (store.go `goSteps`), key from
  `foldKey` (fold.go, equals strings.EqualFold). Store matches on `name_key`; the old
  NOCASE UNIQUE stays (the stricter key subsumes it).
- Stats: `stats.Compute` (pure); `store.Completed` and `store.All` share the 3-query
  loader `load` (migration 0004 indexes `ended_at`); tui/stats.go loads the widest range
  once (3 months' previous period through today) and computes per render.
- Navigation in tui.go `updateStats`: ticks and save results still reach activeModel
  while Stats is open over Active; the alert overlays Stats and takes keys. Choosing
  Extend there switches to Active (where the input is) — a judgement call, R-32 silent.
- Chart: tui/chart.go. Plot rows = terminal height − 14 (4..16); slot per bucket from
  width (4..16); 7-day labels split onto two lines when they do not fit. Estimated `▒`
  faint, actual `█` blue, so NO_COLOR still reads.
- Export: internal/export (`Build` pure, `Write` O_EXCL, mode 0600), golden
  internal/export/testdata/export_v1.golden.json (task 1 = SPEC example). `e` on Stats
  via tui/stats.go `runExport`; Deps.Version and Deps.ExportDir (empty = cwd).
