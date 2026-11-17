# README upkeep

`README.md` tells a newcomer what exists today and how to use it. It is updated at gate
close, and immediately when setup or usage commands change. It never describes planned
work as if it were done.

## Sections and their sources

| Section | Source | Rule |
|---|---|---|
| Badges | `VERSION`, `STATUS.md`, CI, license | See below |
| Overview paragraph | `docs/SPEC.md` | What it does, for whom, current maturity |
| Key features | `docs/acceptance.json` criteria that pass | Shipped features only |
| Quick start | AGENTS.md Commands table | Commands must be copy-paste correct; run them |
| Roadmap | `docs/PLAN.md` phases | One row per phase with a status |
| Contributing | `CONTRIBUTING.md` | Link, do not duplicate |
| License | `LICENSE` | Name and link |

Edit inside the `<!-- BEGIN:x -->` / `<!-- END:x -->` markers and leave the markers in
place.

Roadmap status values: `Planned`, `In progress`, `Awaiting gate`, `Done (vX.Y.Z)`.

## Badges

Badges come from [Shields.io](https://shields.io/). Static badges have the form
`https://img.shields.io/badge/<label>-<message>-<color>`. In the label and message, write
a literal dash as `--`, a space as `%20` or `_`, and a literal underscore as `__`.

| Badge | URL | Updated |
|---|---|---|
| version | `https://img.shields.io/badge/version-<VERSION>-blue` | Gate close, with `VERSION` |
| phase | `https://img.shields.io/badge/phase-<N>%20<name>-orange` | Gate close, with STATUS.md |
| CI | `https://img.shields.io/github/actions/workflow/status/<OWNER>/<REPO>/ci.yml?branch=main&label=CI` | Never; it is live |
| SemVer | `https://img.shields.io/badge/semver-2.0.0-blue` | Never |
| Conventional Commits | `https://img.shields.io/badge/Conventional%20Commits-1.0.0-fe5196` | Never |
| license | `https://img.shields.io/badge/license-<LICENSE>-green` | When the license changes |

Examples: version `1.0.0-rc.1` is written `version-1.0.0--rc.1-blue`.

The version and phase badges are static on purpose. They work for private repositories,
and the scaffold check verifies them against `VERSION` and STATUS.md.

Shields cannot read private repositories. For a private repository, replace the CI badge
with GitHub's own:
`https://github.com/<OWNER>/<REPO>/actions/workflows/ci.yml/badge.svg`. For a public
repository you may swap the static version badge for the live
`https://img.shields.io/github/v/tag/<OWNER>/<REPO>?sort=semver&label=version` and set
`readme_version_badge` to `false` in `.scaffold.json`.

Add at most two or three more badges, and only ones a reader would act on, such as
coverage or the package registry version.
