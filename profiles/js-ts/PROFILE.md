# Profile: JavaScript / TypeScript

## Toolchain

- Node.js, current LTS. Record it in `.nvmrc` and in `engines` in `package.json`.
- `npm` with a committed `package-lock.json`. If the project uses `pnpm` or `yarn`,
  change every command below and record the choice as a decision.
- TypeScript in strict mode, `eslint` for linting, `prettier` for formatting, `vitest`
  for tests.
- The commands below assume these `package.json` scripts: `format`, `format:check`,
  `lint`, `typecheck`, `test`, `build`.

## Commands

Copy into the Commands table in `AGENTS.md`.

| Task | Command |
|---|---|
| Setup | `npm ci` |
| Format | `npm run format` |
| Lint | `npm run lint` |
| Typecheck | `npm run typecheck` |
| Test (all) | `npm test` |
| Test (single) | `npx vitest run path/to/file.test.ts -t "test name"` |
| Build | `npm run build` |

Suggested scripts: `"format": "prettier --write ."`,
`"format:check": "prettier --check ."`, `"lint": "eslint ."`,
`"typecheck": "tsc --noEmit"`, `"test": "vitest run"`.

## Layout

`src/` for code, tests beside the code as `*.test.ts` or under `tests/`, build output in
`dist/`.

## Versioning

`package.json` `version` mirrors `VERSION` exactly; npm uses SemVer.

```json
"profiles": ["js-ts"],
"version_mirrors": [
  {"path": "package.json", "kind": "package_json"}
]
```

Set the mirror with `npm version X.Y.Z --no-git-tag-version`, which also updates the
lockfile. Never run `npm version` without that flag: it commits and tags on its own, and
tagging happens only in the gate close routine, after the release commit.

## Conventions

- ES modules (`"type": "module"`), unless a decision says otherwise.
- No `any` without a comment explaining why. No default exports in library code.
- Set `"private": true` unless the package is published.

## Gotchas

- `npm install` rewrites the lockfile; use `npm ci` for a reproducible setup and
  `npm install <pkg>` only when adding a dependency.
- `vitest` without `run` starts watch mode and never exits. Agents must use
  `vitest run`.
- Type checking and tests are separate: a passing test run does not mean `tsc` passes.

## .gitignore

```
node_modules/
dist/
coverage/
*.tsbuildinfo
.env
.env.*
!.env.example
```
