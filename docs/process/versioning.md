# Versioning and releases

The project follows [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).

## Source of truth

- `VERSION` holds the current version as `MAJOR.MINOR.PATCH`, with an optional
  pre-release such as `1.0.0-rc.1`. It starts at `0.0.0`, meaning nothing is released.
- Language files that also carry the version are mirrors. They are listed under
  `version_mirrors` in `.scaffold.json` and must agree with `VERSION`. The profile says
  which files and in which notation.
- Git tags are annotated, `v` + the version, for example `v1.2.3`. The agent creates
  and pushes them in the gate close routine, only for a version approved in `GATES.md`.
  A pushed tag is never moved or deleted.
- The public interface that SemVer protects is the one defined in `docs/SPEC.md`.

## Choosing the bump

The agent proposes a bump at each gate from the Conventional Commits since the last
release (`git log <last tag>..HEAD --pretty=%s%n%b`). The largest change wins.

| Commits since last release | At 1.0.0 or later | While 0.y.z |
|---|---|---|
| Any `!` or `BREAKING CHANGE:` | MAJOR | MINOR |
| Any `feat` | MINOR | MINOR |
| Only `fix`, `perf` | PATCH | PATCH |
| Only `docs`, `test`, `refactor`, `build`, `ci`, `chore` | none | none |

Notes:

- SemVer treats 0.y.z as initial development where anything may change. Bumping MINOR
  for breaking changes before 1.0.0 is this project's convention, not a spec rule.
- The first release is `0.1.0`.
- Moving to `1.0.0` declares the public interface stable. It is a human decision made at
  a gate, never an automatic result of the table.
- Reset lower numbers on a bump: `1.4.2` → `1.5.0` → `2.0.0`.
- A released version is never changed. Fix mistakes with a new version.
- Commits without a Conventional Commits type, such as a hand-run `/yeet`, do not
  affect the bump. Read their diff if you suspect one carries a feature or a fix.

## Pre-releases

Use `-alpha.N`, `-beta.N`, and `-rc.N` only, for example `1.0.0-rc.1`. This form is
valid SemVer and maps cleanly onto every profile's notation. Precedence:
`1.0.0-alpha.1 < 1.0.0-beta.1 < 1.0.0-rc.1 < 1.0.0`.

## Changelog

`CHANGELOG.md` follows [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/).

- Add a bullet under `[Unreleased]` when a user-visible change lands, grouped as
  `Added`, `Changed`, `Deprecated`, `Removed`, `Fixed`, or `Security`.
- At gate close, rename `[Unreleased]` to `[X.Y.Z] - YYYY-MM-DD` (ISO 8601 date) and add
  a new empty `[Unreleased]` above it.
- Write for users of the project. Do not paste commit logs.

## Release sequence

1. Agent: proposes the version in the gate request.
2. Human: approves with `Version: X.Y.Z` or `Version: none`, by editing `GATES.md` or
   by replying `approved` and confirming the line the agent records.
3. Agent: runs the gate close routine in `gates.md`, which applies the version, commits
   the release, tags it, and invokes the `yeet` skill.
4. Human: confirms the `yeet` push.
5. Agent: pushes the tag once the branch push has succeeded.

For a fix between gates, the human adds an `R: APPROVED ... Version: X.Y.Z` line to
`GATES.md`, or replies `release X.Y.Z` and confirms the line the agent records (see
`gates.md`). The agent then runs steps 1 and 5 through 9 of the gate close routine, using
the commit message `chore(release): vX.Y.Z`.

## What the scaffold check verifies

`VERSION` is valid SemVer and equals the last approved version in `GATES.md`; approved
versions only increase; the newest `CHANGELOG.md` release heading equals `VERSION`; the
README version badge and every mirror agree with it. Every `vX.Y.Z` tag names an
approved version and sits on a commit whose `VERSION` is `X.Y.Z`; an approved version
without a tag is a warning.
