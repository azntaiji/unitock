# STATUS

Phase: 1
Phase name: Walking skeleton
Gate: G1 REQUESTED
Updated: 2026-11-17

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Wait for the G1 decision from azntaiji. On approval, run the gate close routine in
   docs/process/gates.md (apply the version, README, archive, release commit, tag, yeet).

## Blockers

None. D-02 to D-08 Approved 2026-11-17; D-09 to D-14 Proposed (Phase 2+).

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|
| P1-01 | `go test ./internal/duration/ -count=1` + `-fuzz '^FuzzParse$' -fuzztime 30s` | PASS (8.3M execs) | artifacts/phase-1/p1-01-duration.txt |
| P1-02 | `go test ./internal/task/ -count=1 -v` | PASS | artifacts/phase-1/p1-02-task.txt |
| P1-03 | `go test ./internal/store/ -count=1 -v` | PASS | artifacts/phase-1/p1-03-store.txt |
| P1-04 | `go test ./internal/tui/ -count=1 -run 'Start' -v` | PASS (11) | artifacts/phase-1/p1-04-start.txt |
| P1-05 | `go test ./internal/tui/ -count=1 -run 'Flow' -v` | PASS (after review fix) | artifacts/phase-1/p1-05-flow.txt |
| P1-06 | `go test ./... -count=1 -run 'Resume' -v` + by hand | PASS (4 tests, after review fix); by hand 1:38 | artifacts/phase-1/p1-06-resume.txt |
| P1-07 | `gh run view 44201069710` + `./bin/unitock --version` | success ubuntu + macos on 2ed8753 | artifacts/phase-1/p1-07-ci.txt |
| setup | `go mod download` | PASS | artifacts/phase-1/gate/setup.txt |
| fmt | `make fmt` | PASS, no code changes | artifacts/phase-1/gate/fmt.txt |
| lint | `make lint` | PASS, 0 issues | artifacts/phase-1/gate/lint.txt |
| vet | `make vet` | PASS | artifacts/phase-1/gate/vet.txt |
| test | `make test` | PASS | artifacts/phase-1/gate/test.txt |
| build | `make build` | PASS | artifacts/phase-1/gate/build.txt |
| check | `make check` (after review fix) | PASS | artifacts/phase-1/gate/check.txt |
| scaffold | `python3 scripts/check_scaffold.py` | 0 errors, 0 warnings | artifacts/phase-1/gate/scaffold.txt |
| release | `make release-check` | PASS, snapshot built | artifacts/phase-1/gate/release-check.txt |

## Gate request

Gate: G1, phase 1 (Walking skeleton)
Approvers: azntaiji
Exit criteria: P1-01 to P1-07 all passing
Review: one gap fixed (countdown tick drifted past 1 s and could skip a second, R-19;
  now `tea.Every`, 4f7d50a). The fix is after the CI-tested commit 2ed8753: verified
  locally with `make check`, and it reaches CI on the gate-close push. No other gaps.
Proposed version: 0.1.0 (minor: `feat` commits; first release)
Look at: CHANGELOG.md [Unreleased]; internal/tui/flow_test.go; artifacts/phase-1/
Open decisions or RFIs the next phase depends on: D-09 (timer visuals), D-10 (extend in
  overtime), D-13 (alarm sound and notifications), all Proposed

## Handoff

- Packages: `internal/duration` (D-08 parser), `internal/task` (timer maths, Clock with
  System/Fake), `internal/store` (modernc SQLite, embedded migrations via user_version),
  `internal/tui` (Start, Active, Done screens), `cmd/unitock` (flags, log, wiring).
- Deps: Bubble Tea v2.1.1, Bubbles v2.2.1, Lip Gloss v2.1.1, modernc sqlite v1.62.0,
  teatest v2 pinned at v2.0.0-20261101151554-27ecf3e4d454 (no tags upstream).
- Deferred to Phase 2+: `s` (stats) on the Start button is not handled or hinted until
  Stats exists (R-07); overtime is plain `+MM:SS` until D-09; no lock until D-12.
- On the Done screen `q` quits (R-06 wins over R-23's "any key").
- Tests: Update-level tests run commands only for enter/e/q/ctrl+c (blink and tick
  commands sleep). teatest tests use flow_test.go's `probe`, which records each rendered
  View, because the v2 renderer writes only changed cells.
- `unitock.db` is created 0644 by SQLite inside the 0700 dir; the spec only requires the dir mode.
