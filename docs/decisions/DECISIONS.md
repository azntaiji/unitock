# Decisions register

Status values: `Settled` (fixed by the scaffold or the spec), `Proposed` (recommended;
needs written approval), `Open` (needs information first), `Approved`, `Rejected`,
`Superseded`.

**Coding agent:** build only on `Settled` or `Approved`. Stop and ask on anything else.

To approve, a human changes the status line to `Approved — <name>, <YYYY-MM-DD>`.

`Rejected` and `Superseded` entries move to `archive.md` at gate close, leaving a
one-line pointer here.

Entry shape: **Status**, **Decision**, **Why**, **Alternatives**, **Consequences**.

---

## S-01 — Gate and version approval are human-only; agents tag approved releases
**Status:** Settled
Agents request gates and propose versions. Humans approve in `GATES.md`, by hand or by
confirming the exact line the agent proposes after an explicit reply. At gate close
the agent commits the release, creates the annotated tag `vX.Y.Z` for the approved
version, and pushes it after `yeet` pushes the branch. Tags are never moved or deleted.

## S-02 — Semantic Versioning 2.0.0 with `VERSION` as the source of truth
**Status:** Settled
See `docs/process/versioning.md`.

## S-03 — Agents commit locally with Conventional Commits 1.0.0; pushes go through `yeet`
**Status:** Settled
Agents commit at each task boundary and never push directly. At gate close the agent
invokes the `yeet` skill, which asks the human to confirm before it commits and pushes.
The one direct push an agent makes is the release tag (S-01), and only after `yeet` has
pushed the release commit. The human can also run `/yeet` at any time for a remote
backup.

---

## D-01 — Language profile
**Status:** Open
**Decision:** <python | js-ts | go | other>, with the toolchain in its PROFILE.md.

## D-02 — License and repository visibility
**Status:** Open
**Why it matters:** sets the license badge and whether Shields can read CI status.

## D-03 — Branching model
**Status:** Proposed
**Decision:** trunk-based. Agents commit locally to the default branch at task
boundaries; the remote receives a phase when `yeet` runs at gate close.
**Alternatives:** one branch per phase, merged by pull request at gate close.
**Consequences:** work in progress exists only on the working machine until gate close. Run `/yeet` mid-phase for a backup; releases are still marked only by tags.
