# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
