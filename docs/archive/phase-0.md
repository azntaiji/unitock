# Phase 0 — Foundations (archived)

Gate G0 approved by azntaiji, 2026-11-17. Version: none (no release).

## Plan section as completed

- **Goal:** an approved spec, plan, and working toolchain, with no product code yet.
- **Depends on:** nothing.
- **Tasks:**
  - [x] Interview the human about goals, users, constraints, edge cases, and what is
        out of scope. Keep asking until the hard parts are covered. Write
        `docs/SPEC.md`.
  - [x] Choose the language profile. Set `profiles` and `version_mirrors` in
        `.scaffold.json`. Delete unused profile directories.
  - [x] Fill the Commands table in `AGENTS.md` and run every command to prove it works.
  - [x] Copy the profile's `ci.yml` to `.github/workflows/ci.yml` and its `starter/`
        files into place; create the layout the profile describes.
  - [x] Draft the remaining phases below and their criteria in `docs/acceptance.json`.
  - [x] Record design choices in `docs/decisions/DECISIONS.md` as `Proposed` and open
        questions in `docs/RFIS.md`.
  - [x] Replace the placeholders in `README.md` and the Project section of `AGENTS.md`.
- **Exit criteria:** P0-01 to P0-05.
- **Gate G0:** azntaiji approves the spec, the plan, and every decision the next
  phase depends on.


## Gate request (as submitted)

Gate: G0, phase 0 (Foundations)
Approvers: azntaiji
Exit criteria: P0-01 to P0-05 all passing
Review: fresh-context review found no correctness defects; fixed 5 doc gaps (spec cites
  decisions, R-07/R-23 criteria coverage, P1-07 CI wording, D-02/D-06 to Proposed, STATUS)
Proposed version: none (only docs/build/ci/chore commits; first release is 0.1.0 at G1)
Look at: docs/SPEC.md, docs/PLAN.md, docs/decisions/DECISIONS.md (D-02 to D-14),
  docs/acceptance.json (P1-01 to P4-04 are new and need approval), docs/RFIS.md
Open decisions or RFIs the next phase depends on: D-03, D-04, D-05, D-07, D-08
  (RFI-02 answered: repo renamed to azntaiji/unitock)


## Evidence

| Check | Command | Result | Raw output |
|---|---|---|---|
| Setup | `go mod download` | exit 0 | artifacts/phase-0/go-mod-download.log |
| Format | `make fmt` | exit 0 | artifacts/phase-0/make-fmt.log |
| Lint | `make lint` | exit 0, 0 issues | artifacts/phase-0/make-lint.log |
| Typecheck | `make vet` | exit 0 | artifacts/phase-0/make-vet.log |
| Test (all) | `make test` | exit 0 (no test files yet) | artifacts/phase-0/make-test.log |
| Build | `make build` | exit 0; `--version` prints line | artifacts/phase-0/make-build.log |
| Check | `make check` | exit 0 | artifacts/phase-0/make-check.log |
| Release preflight | `make release-check` | exit 0, 4 targets | artifacts/phase-0/make-release-check.log |
| Scaffold | `python3 scripts/check_scaffold.py` | 0 errors, 0 warnings | artifacts/phase-0/python3-scripts-check-scaffold-py.log |
| CI | `gh run view 44189751710` | success on ubuntu + macos; scaffold-check 44189751706 success | artifacts/phase-0/ci-run.log |


## Answered RFIs

| ID | Question | Owner | Blocks | Blocking | Status | Answer |
|---|---|---|---|---|---|---|
| RFI-01 | Which license, and is the repository public or private? | azntaiji | README badges, D-02 | no | Answered | MIT, public (Phase 0 interview, 2026-11-17); carried into D-02. |
| RFI-02 | When will the GitHub repository be renamed from `time-tracker-v2` to `unitock`? The module path, GoReleaser target, `go install` path, and README badges all use `azntaiji/unitock`. | azntaiji | G1 release (release workflow publishes to `azntaiji/unitock`) | yes, for G1 | Answered | Renamed to `azntaiji/unitock` (public) by the human, 2026-11-17; local `origin` updated. |

## Notes

- Interview answers (2026-11-17) are carried into docs/SPEC.md and D-01, D-02, D-06,
  D-07, D-10, D-11, D-13.
- Name `unitock` chosen from candidates with no GitHub repo-name matches; repo and local
  directory renamed by the human (RFI-02).
- Fresh-context review: no correctness defects; 5 doc gaps fixed before the request.
- docs/APP-IDEA.md is gitignored (internal); SPEC carries everything from it.
- The timer visual (D-09) was shown to the human as an ASCII mockup mid-session.
