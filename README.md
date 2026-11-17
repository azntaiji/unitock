# unitock

[![version](https://img.shields.io/badge/version-0.0.0-blue)](CHANGELOG.md)
[![phase](https://img.shields.io/badge/phase-0%20foundations-orange)](docs/PLAN.md)
[![CI](https://img.shields.io/github/actions/workflow/status/azntaiji/unitock/ci.yml?branch=main&label=CI)](https://github.com/azntaiji/unitock/actions)
[![SemVer](https://img.shields.io/badge/semver-2.0.0-blue)](https://semver.org/spec/v2.0.0.html)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-fe5196)](https://www.conventionalcommits.org/en/v1.0.0/)
[![license](https://img.shields.io/badge/license-MIT-green)](LICENSE)

unitock is a terminal time tracker for one task at a time, for macOS and Ubuntu. Name a
task, give it a duration in plain language (`45m`, `1.5 hours`), and watch a colour-shifting
countdown; when time is up it alerts you on screen, with a system notification, and with a
looping chime until you extend or complete the task. A stats screen compares actual with
estimated time over 7 days, 30 days, or 3 months. Status: in planning (Phase 0); nothing is
usable yet.

## Key features

<!-- BEGIN:features (shipped features only; updated at gate close) -->
Nothing shipped yet. See the roadmap.
<!-- END:features -->

## Quick start

<!-- BEGIN:quickstart (must match the Commands table in AGENTS.md) -->
Requires Go 1.26 or newer.

```sh
go mod download
make check          # build, vet, lint, race tests
./bin/unitock --version
```
<!-- END:quickstart -->

## Roadmap

<!-- BEGIN:roadmap (mirrors docs/PLAN.md; updated at gate close) -->
| Phase | Goal | Status |
|---|---|---|
| 0 | Foundations: spec, plan, tooling | In progress |
| 1 | Walking skeleton: start, countdown, complete, resume | Planned |
| 2 | Active screen and time-up alert | Planned |
| 3 | Stats and export | Planned |
| 4 | Hardening and 1.0.0 | Planned |
<!-- END:roadmap -->

Status values: Planned, In progress, Awaiting gate, Done (vX.Y.Z).

## How this project is built

Work proceeds in phases. Each phase ends at a gate that a named person approves, and
every release follows [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).
See [docs/PLAN.md](docs/PLAN.md) for phases and [CHANGELOG.md](CHANGELOG.md) for
releases.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the workflow, commit format, and how gates
and versions work. Coding agents start from [AGENTS.md](AGENTS.md).

## License

MIT. See [LICENSE](LICENSE).
