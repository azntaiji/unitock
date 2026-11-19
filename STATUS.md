# STATUS

Phase: 4
Phase name: Hardening and 1.0.0
Gate: G4 not requested
Updated: 2026-11-18

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. vhs (charmbracelet/vhs) GIFs for the README, with README usage and install sections.
2. Error paths (P4-03), tests named `Error`: unwritable data dir, corrupt database,
   missing audio and notify tools, export directory not writable (export already shows
   the error; add a test that proves it). Some exist already under other names
   (cmd/unitock TestUnusableDataDirExitsOne, TestUnusableDatabaseExitsOne); check
   `-run 'Error'` picks up every case.
3. SPEC end-to-end verification on macOS and Ubuntu, by hand (P4-01), now 11 steps;
   step 11 also completes P4-06 (hear every sound). Needs the human for Ubuntu. Rerun
   the Perf tests on Ubuntu at the same time.
4. `make release-check`, CHANGELOG release notes, gate request (P4-04).

## Blockers

None. Found, not fixed (pre-existing, outside the task): with the Project dropdown open
(5 suggestions) the Start screen is 21 rows, over the 60×20 minimum (R-08); asked the
human how to resolve it.

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|
| P4-02 perf (macOS arm64) | `go test ./... -count=1 -run 'Perf' -v` | pass: startup 1.6 ms, stats open 27 ms, tab 0.7 ms | artifacts/phase-4/perf.log |
| make check after perf task | `make check` | exit 0 | artifacts/phase-4/check-perf.log |
| old releases published | `gh workflow run release.yml -f tag=v0.X.0` | 3/3 success | gh runs 44429925530, 44430921813, 44431766718 |
| P4-05 Active row | `go test ./internal/tui/ -count=1 -run 'View\|Extend\|Pause' -v` | pass; make check exit 0 | artifacts/phase-4/p4-05.log |
| P4-06 sounds, automated part | `go test ./... -count=1 -run 'Sound' -v` | pass (21 tests); by hand pending | artifacts/phase-4/p4-06.log |
| release kit on Go 1.27.3 (CI fix) | `make release-check` | exit 0 | artifacts/phase-4/release-check-ci-fix.log |

## Gate request

None.

## Handoff

- Release workflow pins Go 1.27.3 (GoReleaser v2.19.0 needs it) and has a manual
  trigger to publish an existing tag: `gh workflow run release.yml -f tag=vX.Y.Z`.
- 2026-11-18: azntaiji approved a one-off push of main before G4, to prove the release
  fix by publishing v0.1.0 to v0.3.0. Not a precedent; the gate-close push rule stands.
- v0.1.0 to v0.3.0 published via manual runs (all green; 4 archives + checksums each;
  v0.3.0 darwin_arm64 checksum OK, `--version` runs). Their bodies came out empty:
  `changelog: disable: true` skips the GoReleaser step that loads --release-notes. Notes
  set after with `gh release edit --notes-file` from each tag's CHANGELOG section.
  .goreleaser.yaml now enables it (local commit, goes out at G4); not yet proven on
  GitHub, so check the v1.0.0 release body after the G4 release run.
- Keys (R-06 amended 2026-11-19): ctrl+s stats, ctrl+o sound, ctrl+c quit; no plain
  q/s/a commands. Hints: screen keys, then `chordHints(...)` on a 2nd line (Start,
  Active). Verified in a real pty via tmux (ctrl+s is not eaten as XOFF).
- Sounds (D-16): alarm/sound.go synthesises the five (Chime byte-identical to D-13,
  hash-tested; others normalised to 0.5 peak); Alarm.Preview plays once, the alarm
  always wins over a preview. store/settings.go + migration 0005. tui/sound.go is the
  Sound screen (ctrl+o from Start); tui.New loads the sound and calls SetSound.
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
  flows in flow_test.go; `pressStats` in stats_nav_test.go; `key("ctrl+x")` chords.
