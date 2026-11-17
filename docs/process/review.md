# Pre-gate review

An agent grading its own work tends to approve it. Before requesting a gate, have the
phase reviewed by a reviewer that did not see the work being done: a subagent where the
harness supports one, otherwise a new session.

Give the reviewer only the diff and the criteria, not your reasoning.

## Reviewer prompt

```
Review the changes in this repository since <last tag or the commit where the phase
started> against phase <N> of docs/PLAN.md and its criteria in docs/acceptance.json.

For each exit criterion: run its `verify` step yourself and say whether it passes.

Then check:
- every task of the phase is implemented, and nothing outside the phase's scope changed;
- tests exercise the behavior end to end and were not weakened, skipped, or deleted;
- docs/SPEC.md requirements touched by this phase are met, including the edge cases it
  names;
- no decision marked Proposed or Open was built on.

Report only gaps that affect correctness or the stated requirements, each with
file:line and a one-sentence reason. Do not report style preferences. If there are no
such gaps, say so.
```

## Handling findings

- Fix findings that affect correctness or the requirements, re-run the affected checks,
  and update the evidence.
- A reviewer asked to find gaps will usually find some. Treat the rest as optional, and
  do not add abstraction, defensive code, or tests for cases that cannot happen.
- Record the outcome in one line in the gate request. Do not save the review as a file.
