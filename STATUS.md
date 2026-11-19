# STATUS

Phase: 4
Phase name: Hardening and 1.0.0
Gate: G4 REQUESTED
Updated: 2026-11-19

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Wait for azntaiji's G4 decision. At gate close: VERSION 1.0.0, CHANGELOG [1.0.0],
   README badges, roadmap, and the Install snippet's `VERSION=`; then check the
   v1.0.0 GitHub Release body is not empty (first real run of the notes fix).

## Blockers

None.

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|
| G4 commands table | `make fmt lint vet test build check release-check`; scaffold | all exit 0 | artifacts/phase-4/gate-*.log |
| P4-01 E2E by hand | SPEC end-to-end, macOS + Ubuntu | 15/15 rows pass on both | artifacts/phase-4/e2e-results.md |
| P4-02 perf | `go test ./... -count=1 -run 'Perf' -v` | macOS 1.7/36 ms; Ubuntu 1.7/50 ms | gate-perf.log, perf-ubuntu.log |
| P4-03 errors | `go test ./... -count=1 -run 'Error' -v` | pass, 0 skips | artifacts/phase-4/gate-error.log |
| P4-05 row | `go test ./internal/tui/ -run 'View\|Extend\|Pause'` | pass | artifacts/phase-4/gate-p405.log |
| P4-06 sounds | `go test ./... -count=1 -run 'Sound' -v` + by hand | pass | gate-sound.log, e2e-results.md |

## Gate request

Gate: G4, phase 4 (Hardening and 1.0.0)
Approvers: azntaiji
Exit criteria: P4-01 to P4-06 all passing
Review: fresh-context subagent, no gaps; stale PLAN checkboxes ticked
Proposed version: 1.0.0 (major: SPEC complete and verified on both OSes per PLAN;
  also q/s no longer quit or open Stats, a breaking key change)
Look at: artifacts/phase-4/e2e-results.md, perf-ubuntu.log, gate-*.log; CHANGELOG
  [Unreleased]; README.md
Open decisions or RFIs the next phase depends on: none (mouse input deferred past 1.0)

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
- Version: cmd/unitock/version.go falls back to debug.ReadBuildInfo when -ldflags did not
  set it (go install gives the module version; local builds a pseudo-version + vcs).
- Demos: docs/demo/*.tape (settings + launch shared; alert seeds --running 9),
  scripts/demoseed (6 months, refuses an existing db), `make demo` needs vhs.
  Recorded with PATH=/var/empty so no sound or notification fires.
- Keys (R-06 amended 2026-11-19): ctrl+s stats, ctrl+o sound, ctrl+c quit; no plain
  q/s/a commands. Hints: screen keys, then `chordHints(...)` on a 2nd line (Start,
  Active). Verified in a real pty via tmux (ctrl+s is not eaten as XOFF).
- Sounds (D-16): alarm/sound.go synthesises the five (Chime byte-identical to D-13,
  hash-tested; others normalised to 0.5 peak); Alarm.Preview plays once, the alarm
  always wins over a preview. store/settings.go + migration 0005. tui/sound.go is the
  Sound screen (ctrl+o from Start); tui.New loads the sound and calls SetSound.
- Perf: internal/store/storetest.Seed bulk-inserts history in one transaction;
  cmd/unitock/perf_test.go (startup via run() with runUI stubbed, to first View) and
  internal/tui/perf_test.go (s to rendered Stats, then tab). Both skip under -race
  (storetest.Race), so CI's make check does not time them; the P4-02 verify command does.
  Startup excludes process exec and the terminal write (no TTY in tests).
- internal/alarm implements tui.Alarm; main wires it (bell via tea.RawMsg, Close deferred).
- Lock: cmd/unitock/lock.go. Too small: tui.go `tooSmall` (keys except ctrl+c ignored).
- Stats: internal/stats (pure); store.Completed/All share `load`; tui/stats.go, chart.go;
  navigation in tui.go `updateStats`. Export: internal/export; `e` on Stats.
- Visuals: internal/tui/timer.go, chart.go; goldens in internal/tui/testdata and
  internal/export/testdata (`make golden`).
- TUI tests: harness in start_test.go (send/press/typeText/run, fakeAlarm); teatest
  flows in flow_test.go; `pressStats` in stats_nav_test.go; `key("ctrl+x")` chords.
