# Contributing

This applies to people and coding agents alike. Agents also follow
[AGENTS.md](AGENTS.md).

## Workflow

1. Check [STATUS.md](STATUS.md) for the current phase and [docs/PLAN.md](docs/PLAN.md)
   for what that phase covers. Work outside the current phase waits for its gate.
2. Take one task. Explore, plan, implement, verify.
3. Run the commands in the AGENTS.md Commands table, including
   `python3 scripts/check_scaffold.py`. All must pass.
4. Add user-visible changes to `[Unreleased]` in [CHANGELOG.md](CHANGELOG.md).
5. Commit with a Conventional Commits message and open a pull request, or push to the
   trunk if that is the project's branching decision.

## Commit messages

[Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/):

```
<type>[optional scope][!]: <description>

[optional body]

[optional footer, e.g. BREAKING CHANGE: <what breaks>]
```

Types used here: `feat`, `fix`, `perf`, `refactor`, `docs`, `test`, `build`, `ci`,
`chore`. A `!` or a `BREAKING CHANGE:` footer marks an incompatible change.

## Versions and releases

- The project follows [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).
  `VERSION` is the source of truth.
- Versions change only at a gate or an approved between-gate release. The agent proposes
  the bump; a human approves it in [GATES.md](GATES.md); the agent then commits the
  release and creates and pushes the git tag.
- Details: [docs/process/versioning.md](docs/process/versioning.md).

## Gates

Each phase ends with a gate request in STATUS.md. A named approver reviews the evidence
and either adds a line to GATES.md or replies `approved` and confirms the line the agent
proposes. Only humans make the decision. Details:
[docs/process/gates.md](docs/process/gates.md).

## Decisions and questions

- Propose a design decision as a `Proposed` entry in
  [docs/decisions/DECISIONS.md](docs/decisions/DECISIONS.md). Nothing is built on it
  until a human marks it `Approved`.
- Record an open question as a row in [docs/RFIS.md](docs/RFIS.md) with an owner.

## Documentation

Update the file that already owns the information. Do not add summary, report, or
progress files; the scaffold check rejects them. Finished-phase detail goes to
`docs/archive/`.
