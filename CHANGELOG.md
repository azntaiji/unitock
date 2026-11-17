# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `--data-dir DIR` flag; defaults to `$XDG_DATA_HOME/unitock` or `~/.local/share/unitock`,
  created with mode 0700. Holds `unitock.db` (SQLite) and `unitock.log`.
- `--version` flag printing `unitock <version> (<commit>, <date>)`.
- Exit status 1 on startup errors (bad flag, unusable data directory or database).
