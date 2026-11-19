# STATUS

Phase: 4
Phase name: Hardening and 1.0.0
Gate: G4 not requested
Updated: 2026-11-18

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Error paths (P4-03), tests named `Error`: unwritable data dir, corrupt database,
   missing audio and notify tools, export directory not writable (export already shows
   the error; add a test that proves it). Some exist already under other names
   (cmd/unitock TestUnusableDataDirExitsOne, TestUnusableDatabaseExitsOne); check
   `-run 'Error'` picks up every case.
2. README usage section, install instructions, and a screenshot or recording.
3. SPEC end-to-end verification on macOS and Ubuntu, by hand (P4-01); needs the human
   for Ubuntu. Rerun the Perf tests on Ubuntu at the same time.
4. `make release-check`, CHANGELOG release notes, gate request (P4-04).

## Blockers

None. Open for the human: tags v0.1.0 to v0.3.0 have no GitHub Release (every release
run failed, see Handoff); whether to publish them is a release decision.

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|
| P4-02 perf (macOS arm64) | `go test ./... -count=1 -run 'Perf' -v` | pass: startup 1.6 ms, stats open 27 ms, tab 0.7 ms | artifacts/phase-4/perf.log |
| make check after perf task | `make check` | exit 0 | artifacts/phase-4/check-perf.log |
| release kit on Go 1.27.3 (CI fix) | `make release-check` | exit 0 | artifacts/phase-4/release-check-ci-fix.log |

## Gate request

None.

## Handoff

- Release workflow failed for v0.1.0 to v0.3.0: setup-go took go.mod's 1.26.0 with
  GOTOOLCHAIN=local; GoReleaser v2.19.0 needs >= 1.27.3. release.yml now pins Go 1.27.3
  (committed locally; reaches GitHub at the G4 push). Rerunning old runs would reuse
  the old workflow file at each tag.
- Phase 3 closed 2026-11-18 as v0.3.0; detail in docs/archive/phase-3.md.
- Perf: internal/store/storetest.Seed bulk-inserts history in one transaction;
  cmd/unitock/perf_test.go (startup via run() with runUI stubbed, to first View) and
  internal/tui/perf_test.go (s to rendered Stats, then tab). Both skip under -race
  (storetest.Race), so CI's make check does not time them; the P4-02 verify command does.
  Startup excludes process exec and the terminal write (no TTY in tests).
- Alarm heard and notification seen on macOS only; Ubuntu untested (Phase 4 E2E).
- internal/alarm implements tui.Alarm; main wires it (bell via tea.RawMsg, Close deferred).
- Lock: cmd/unitock/lock.go. Too small: tui.go `tooSmall` (keys except ctrl+c ignored).
- Stats: internal/stats (pure); store.Completed/All share `load`; tui/stats.go, chart.go;
  navigation in tui.go `updateStats`. Export: internal/export; `e` on Stats.
- Visuals: internal/tui/timer.go, chart.go; goldens in internal/tui/testdata and
  internal/export/testdata (`make golden`).
- TUI tests: harness in start_test.go (send/press/typeText/run, fakeAlarm); teatest
  flows in flow_test.go; `pressS` in stats_nav_test.go.
