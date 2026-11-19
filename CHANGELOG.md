# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-19

### Added

- README install and usage sections, with animated demos recorded by vhs from
  `docs/demo/*.tape` (`make demo` re-records them).
- Choose the alarm sound: press `ctrl+o` on the Start screen to open the Sound screen
  and pick Chime (default), Bell, Beep, Marimba, or Rising. Each plays once as you
  highlight it; `enter` saves, `esc` cancels. The choice is kept in the database (a new
  `settings` table, migrated on startup) and shown beside the Start task button as
  `Alarm: <name>`.

### Changed

- App-wide commands are now `ctrl` chords that work from any focus, including while
  typing: `ctrl+s` opens Stats, `ctrl+o` opens the Sound screen, and `ctrl+c` quits.
  Plain `q` and `s` no longer quit or open Stats on any screen. Key hints that do not
  fit on one line show the chords on a second line.
- The line under the Active screen's progress bar shows elapsed time and the estimate
  (`Elapsed 12:31 · Estimate 25:00`), and once time has been added, the original
  estimate and the total added (`Estimate 25:00 + 10:00 added`). It no longer repeats
  the remaining time shown in the large digits.

### Fixed

- `--version`, and the `app_version` in exports, name the real version for builds made
  with `go install` (they printed `dev`).
- The Start screen fits the 60×20 minimum with the project suggestions open: they now
  list up to 4 projects instead of 5.
- Every screen fits the 60×20 minimum: long error messages (such as an invalid
  duration) wrap instead of widening the screen, and the time-up alert's key hints use
  two lines.
- Clearer startup errors: an unwritable data directory says so and names it, and an
  unreadable database names the file (which is never changed). A failed export shows
  the cause before the path, so a long path no longer hides why it failed.
- Release notes are published with each GitHub Release (the release body was empty).

## [0.3.0] - 2026-11-18

### Added

- Stats screen: `s` opens it from the Start button and the Active screen; `tab` cycles
  7 days, 30 days, and 3 months; scorecards show completed tasks and the average per
  day against the previous period; `esc` goes back with the timer still running, and
  the time-up alert appears over it.
- Stats chart: estimated and actual time per day, week, or month as vertical bars, with
  a legend and a y-axis in hours and minutes; a message when nothing was completed.
- Export: `e` on the Stats screen writes every task, with its pauses and extensions, to
  `unitock-export-YYYYMMDD-HHMMSS.json` in the working directory (JSON schema version
  1); it never overwrites an existing file and shows the path or the error.

### Fixed

- Project names now match ignoring case for all Unicode letters, so "über" reuses
  "Über". Upgrading merges existing projects that differ only by case into the oldest.

## [0.2.0] - 2026-11-18

### Added

- Active screen shows a status badge (`● RUNNING`, `◐ ENDING`, `▲ OVERTIME`,
  `❚❚ PAUSED`) above the task name, the remaining time in large block digits, and a
  progress bar with the elapsed percentage, coloured by how much time is left (blue at
  half, yellow at a quarter, red at a tenth and in overtime). Colours come from the
  terminal's own palette; under `NO_COLOR` the badge shows in reverse video.
- Optional project for each task. The Start screen's new Project input suggests existing
  projects as you type (tab completes, ↑/↓ choose, esc closes), and a project typed in
  different case reuses the existing one. The Active screen shows the project in bold
  above the task name.
- `d` adds time to the active task, running or paused, using the same duration forms as
  the Start screen.
- `n` cancels the active task after a Yes/No confirmation (No is selected first; `y`
  confirms at once) and returns to the Start screen.
- Time-up alert: when a running task reaches zero, a box floating over the timer offers
  Extend or Mark as completed (`←`/`→`/`tab`, `enter`, or `d`/`e`). Extending from the
  alert makes the entered amount the new remaining time. A task that ran out while the
  app was closed alerts at launch.
- At time up, a system notification (`osascript` on macOS, `notify-send` on Ubuntu) and
  a looping chime (`afplay`; `paplay` or `aplay`) until you choose. With no working
  player, the terminal bell rings every 2 seconds instead. The chime is written to
  `alarm.wav` in the data directory.
- Only one unitock runs per data directory: a second exits with status 1 and names the
  lock file (`unitock.lock`).
- Below 60×20 the app shows "Terminal too small" with the current size instead of the
  screen; only `ctrl+c` works until it is resized. Timers and the alarm keep running.
- `p` pauses and resumes the active task. Paused time does not count, and a task paused
  at quit is still paused at the next launch.

### Changed

- The database moves to schema 2 (projects) automatically on first launch.

## [0.1.0] - 2026-11-17

### Added

- `--data-dir DIR` flag; defaults to `$XDG_DATA_HOME/unitock` or `~/.local/share/unitock`,
  created with mode 0700. Holds `unitock.db` (SQLite) and `unitock.log`.
- `--version` flag printing `unitock <version> (<commit>, <date>)`.
- Start screen: task name and duration inputs and a Start task button, with tab/↑/↓
  focus, inline validation errors, and a key-hint bar.
- Active screen: task name with an `MM:SS` (or `H:MM:SS`) countdown updated every
  second, `+MM:SS` once over the estimate; `e` completes the task and shows a
  motivational message.
- An active task survives quitting: relaunching opens the Active screen with the time
  spent closed counted as elapsed.
- Exit status 1 on startup errors (bad flag, unusable data directory or database).
