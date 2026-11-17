# Plan

Phases, tasks, exit criteria, and gates. Agents read only the current phase's section.
When a gate closes, that phase's section is replaced by a short summary and the detail
moves to `docs/archive/phase-N.md`.

Each phase section uses this shape:

- **Goal:** one sentence.
- **Depends on:** decision and RFI IDs that must be resolved first.
- **Tasks:** a checklist of small, independently verifiable tasks.
- **Exit criteria:** IDs from `docs/acceptance.json`.
- **Gate:** who approves.

## Phase 0 — Foundations

- **Goal:** an approved spec, plan, and working toolchain, with no product code yet.
- **Depends on:** nothing.
- **Tasks:**
  - [ ] Interview the human about goals, users, constraints, edge cases, and what is
        out of scope. Keep asking until the hard parts are covered. Write
        `docs/SPEC.md`.
  - [ ] Choose the language profile. Set `profiles` and `version_mirrors` in
        `.scaffold.json`. Delete unused profile directories.
  - [ ] Fill the Commands table in `AGENTS.md` and run every command to prove it works.
  - [ ] Copy the profile's `ci.yml` to `.github/workflows/ci.yml` and its `starter/`
        files into place; create the layout the profile describes.
  - [ ] Draft the remaining phases below and their criteria in `docs/acceptance.json`.
  - [ ] Record design choices in `docs/decisions/DECISIONS.md` as `Proposed` and open
        questions in `docs/RFIS.md`.
  - [ ] Replace the placeholders in `README.md` and the Project section of `AGENTS.md`.
- **Exit criteria:** P0-01 to P0-05.
- **Gate G0:** <approver name> approves the spec, the plan, and every decision the next
  phase depends on.

## Phase 1 — Walking skeleton

- **Goal:** the thinnest end-to-end slice that runs, is tested, and passes CI.
- **Depends on:** <decision and RFI IDs>.
- **Tasks:**
  - [ ] <Project layout per the profile>
  - [ ] <One real path through the system, end to end>
  - [ ] <A test that exercises that path the way a user would>
  - [ ] <CI green on the default branch>
- **Exit criteria:** <P1-xx>.
- **Gate G1:** <approver name>.

## Phase 2 — <name>

- **Goal:** <one sentence>
- **Depends on:** <IDs>
- **Tasks:**
  - [ ] <task>
- **Exit criteria:** <P2-xx>
- **Gate G2:** <approver name>.

<!-- Add phases as needed. A final hardening phase usually precedes 1.0.0. -->

## Out of scope

<What this project will not do, so that agents do not drift into it.>
