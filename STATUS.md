# STATUS

Phase: 1
Phase name: Walking skeleton
Gate: G1 not requested
Updated: 2026-11-17

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. P1-06 by-hand check (human): `./bin/unitock --data-dir <scratch>`, start `2m`, q, wait
   20 s, relaunch, read remaining (expect ~1:40). Then set P1-06 passes.
3. Then the gate routine in docs/process/gates.md (fresh-context review, G1 request).
2. P1-07: human runs `/yeet`; agent checks `gh run list --workflow ci.yml --limit 1`.

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
| active | `go test ./internal/tui/ -count=1 -v` | PASS | artifacts/phase-1/p1-active.txt |
| P1-06 (test part) | `go test ./... -count=1 -run 'Resume' -v` | PASS (4 incl. teatest relaunch); by-hand pending | artifacts/phase-1/p1-06-resume.txt |
| P1-05 | `go test ./internal/tui/ -count=1 -run 'Flow' -v` | PASS | artifacts/phase-1/p1-05-flow.txt |
| check | `make check` | PASS, 0 lint issues | artifacts/phase-1/p1-05-make-check.txt |

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
- Active task done: countdown from the clock on each `tickMsg` (1 s, generation-tagged so
  old loops die); remaining rounds up, overtime shows plain `+MM:SS` (colours wait for
  D-09). `e` stores completion, then the Done screen shows `messages[Deps.Rand(n)]` (22
  messages); any key opens Start. Big digits, p/d/n/s: Phase 2.
- Resume done: `tui.New` now returns `(Model, error)` and opens Active if the store has an
  active task. On the Done screen `q` quits (R-06 wins over R-23's "any key").
- teatest v2 pinned at v2.0.0-20261101151554-27ecf3e4d454. The renderer emits only changed
  cells, so flow_test.go's `probe` wrapper records each rendered View for assertions.
- Note: `unitock.db` is created 0644 by SQLite (inside the 0700 dir); spec only requires dir mode.
- Commit pushes from agents happen only through `yeet` at gate close. P1-07 needs CI green
  on a pushed commit, so ask the human to `/yeet` before requesting G1.
