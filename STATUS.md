# STATUS

Phase: 1
Phase name: Walking skeleton
Gate: G1 not requested
Updated: 2026-11-17

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Minimal Active screen: `MM:SS` countdown ticking each second; `e` completes with a
   motivational message (≥20, R-23), then Start. Replaces `internal/tui/active.go` stub.
2. Then the remaining Phase 1 tasks in docs/PLAN.md, in order.

## Blockers

None. D-02 to D-08 Approved 2026-11-17; D-09 to D-14 Proposed (later phases).

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|
| P1-01 | `go test ./internal/duration/ -count=1` + `-fuzz '^FuzzParse$' -fuzztime 30s` | PASS (8.3M execs) | artifacts/phase-1/p1-01-duration.txt |
| P1-02 | `go test ./internal/task/ -count=1 -v` | PASS | artifacts/phase-1/p1-02-task.txt |
| P1-03 | `go test ./internal/store/ -count=1 -v` | PASS | artifacts/phase-1/p1-03-store.txt |
| cmd | `go test ./cmd/unitock/ -count=1 -v` | PASS | artifacts/phase-1/p1-cmd.txt |
| P1-04 | `go test ./internal/tui/ -count=1 -run 'Start' -v` | PASS | artifacts/phase-1/p1-04-start.txt |
| check | `make check` | PASS, 0 lint issues | artifacts/phase-1/p1-04-make-check.txt |

## Gate request

None.

## Handoff

- Phase 0 closed 2026-11-17 with no release; detail in docs/archive/phase-0.md.
- P1-01 done: `internal/duration.Parse` returns ErrSyntax/ErrRange; both messages name
  the accepted forms, so the Start screen can show `err.Error()` inline (R-11).
- P1-02 done: `internal/task` has `Task` (Estimate/Elapsed/Remaining/Overtime/TimeUp/Paused
  taking `now`) and `Clock` with `SystemClock` and a concurrency-safe `FakeClock` for teatest.
- P1-03 done: `store.Open(ctx, dir)` makes dir (0700) + `unitock.db`, migrates via embedded
  `migrations/NNNN_*.sql` and `user_version`; Create/Active/Get/Complete. modernc v1.62.0;
  `CGO_ENABLED=0 go build ./...` OK. Cancel/pause/extend ops come with their screens.
- cmd task done: `run()` parses flags, makes the data dir, opens `unitock.log` (slog) and
  the store, then `runUI` runs the Bubble Tea program. Lock (R-09) waits for D-12 (Proposed).
- P1-04 done: Bubble Tea v2.1.1, Bubbles v2.2.1, Lip Gloss v2.1.1. `s` (stats) on the Start
  button is not handled or hinted until the Stats screen exists (R-07). Tests drive
  `Model.Update` directly and run commands only for enter/q/ctrl+c, because blink ticks sleep.
  teatest v2 has no tags: pin `v2.0.0-20261101151554-27ecf3e4d454` in P1-05.
- Note: `unitock.db` is created 0644 by SQLite (inside the 0700 dir); spec only requires dir mode.
- Commit pushes from agents happen only through `yeet` at gate close. P1-07 needs CI green
  on a pushed commit, so ask the human to `/yeet` before requesting G1.
