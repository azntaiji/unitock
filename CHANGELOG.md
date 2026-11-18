# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
- `p` pauses and resumes the active task. Paused time does not count, and a task paused
  at quit is still paused at the next launch.

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
