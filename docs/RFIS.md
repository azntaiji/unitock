# Requests for information

Status: `Open`, `Answered`, `Deferred (to phase N)`.

**Coding agent:** if a task depends on an `Open` item with `Blocking: yes`, do the
independent parts, then stop and ask. Never fill in an answer yourself.

Once an answer has been carried into `docs/SPEC.md` or a decision, move its row to
`docs/archive/` at the next gate close.

| ID | Question | Owner | Blocks | Blocking | Status | Answer |
|---|---|---|---|---|---|---|
| RFI-01 | Which license, and is the repository public or private? | azntaiji | README badges, D-02 | no | Answered | MIT, public (Phase 0 interview, 2026-11-17); carried into D-02. |
| RFI-02 | When will the GitHub repository be renamed from `time-tracker-v2` to `unitock`? The module path, GoReleaser target, `go install` path, and README badges all use `azntaiji/unitock`. | azntaiji | G1 release (release workflow publishes to `azntaiji/unitock`) | yes, for G1 | Open | |
