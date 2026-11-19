# STATUS

Phase: 5
Phase name: Not yet planned
Gate: G5 not requested
Updated: 2026-11-19

Working state for the current phase only. Rewrite in place; keep under 80 lines.

## Next tasks

1. Check that the v1.0.0 GitHub Release has its notes in the body (first real run of
   the `changelog: disable: false` fix in .goreleaser.yaml). If it is empty, set it with
   `make release-notes VERSION=1.0.0` and `gh release edit v1.0.0 --notes-file
   bin/release-notes.md`.
2. Wait for azntaiji to set Phase 5's goal, tasks, and exit criteria in docs/PLAN.md.
   No Phase 5 work starts before then.

## Blockers

None. Phase 5 scope is the human's decision.

## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|

## Gate request

None.

## Handoff

- 1.0.0 released at G4 (2026-11-19); phase 4 detail in docs/archive/phase-4.md.
- Release: tag vX.Y.Z triggers .github/workflows/release.yml (Go 1.27.3 pinned for
  GoReleaser v2.19.0); `gh workflow run release.yml -f tag=vX.Y.Z` republishes a tag
  that has no release. Notes come from `make release-notes` (CHANGELOG section).
- Deferred past 1.0.0: mouse input (docs/PLAN.md Out of scope).
- Code map: see AGENTS.md Packages; TUI test harness in internal/tui/start_test.go,
  goldens via `make golden`, demos via `make demo` (needs vhs).
