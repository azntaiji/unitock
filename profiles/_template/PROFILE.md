# Profile: <language>

## Toolchain

<Runtime and minimum version, package manager, formatter, linter, type checker, test
runner. One line each.>

## Commands

Copy into the Commands table in `AGENTS.md`.

| Task | Command |
|---|---|
| Setup | `<command>` |
| Format | `<command>` |
| Lint | `<command>` |
| Typecheck | `<command>` |
| Test (all) | `<command>` |
| Test (single) | `<command>` |
| Build | `<command>` |

## Layout

<Where source, tests, and configuration live.>

## Versioning

<Which files mirror `VERSION`, in which notation, and the `.scaffold.json` snippet.>

```json
"version_mirrors": [
  {"path": "<file>", "kind": "regex", "pattern": "<regex with one capture group>"}
]
```

Mirror kinds: `pyproject`, `python_dunder`, `package_json`, `regex`.

## Conventions

<Only rules that differ from the language's defaults. Put a longer guide in its own file
beside this one and link it. Ready-to-copy source files go in `starter/`.>

## Gotchas

<Non-obvious behavior that has caused or would cause mistakes.>

## .gitignore

```
<lines>
```
