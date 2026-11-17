# Profile: Go

## Toolchain

- Go, a currently supported release. The `go` directive in `go.mod` is the floor.
- `make` is the single command surface; copy `starter/Makefile` at setup.
- golangci-lint v2 (and GoReleaser, with the release kit) pinned in the Makefile and run
  with `go run tool@version`: nothing to install, and CI uses the same versions because
  it calls `make`. Not `go tool`, which would add the linters' dependencies to `go.mod`.

## Commands

Copy into the Commands table in `AGENTS.md`.

| Task | Command |
|---|---|
| Setup | `go mod download` |
| Format | `make fmt` |
| Lint | `make lint` (also fails on unformatted files) |
| Typecheck | `make vet` |
| Test (all) | `make test` |
| Test (single) | `go test ./path/to/pkg -run '^TestName$'` |
| Build | `make build` |
| Check (all of it) | `make check`: build, vet, lint, race tests |

`make check` must pass before a task counts as done.

## Layout

```
cmd/<name>/          main package; thin: flags, wiring, exit codes
internal/<pkg>/      everything else; not importable by other modules
internal/<pkg>/testdata/   fixtures and *.golden files
bin/, dist/          build output, gitignored
```

A public library puts packages at the root or under `pkg/` instead. Tests sit beside the
code as `*_test.go`. At setup copy `starter/Makefile` and `starter/.golangci.yml` and fill
their placeholders. In Phase 0 add a package map to the Project section of `AGENTS.md`,
one line per package saying what it owns, and keep it current as packages change.

## Versioning

Go modules are versioned by git tags of the form `vX.Y.Z`, which is the same tag this
scaffold already uses. `VERSION` stays the source of truth for the changelog and README,
and the tag the agent creates at release must match it. There is no mirror file by default.

```json
"profiles": ["go"],
"version_mirrors": []
```

If a binary reports its version, inject it at build time: declare
`var version, commit, date = "dev", "none", "unknown"` in `main` and set them with
`-ldflags "-X main.version=<X.Y.Z>"` (the release kit does this). Otherwise keep a
constant and mirror it:

```json
"version_mirrors": [
  {"path": "internal/version/version.go", "kind": "regex",
   "pattern": "Version\\s*=\\s*\"([^\"]+)\""}
]
```

A MAJOR version of 2 or higher changes the module path: `module example.com/x/v2` in
`go.mod`, and every import of the module's own packages gains `/v2`. A breaking release
is therefore a code change as well as a version change. Say so in the gate request.

### Release kit (optional)

For projects that ship binaries. Adopting it is a decision in `DECISIONS.md`. Copy
`starter/.goreleaser.yaml` to the root and `release.yml` to `.github/workflows/`. Every
`vX.Y.Z` tag the agent pushes then publishes a public GitHub Release built by GoReleaser
(darwin and linux, amd64 and arm64), with notes taken from that version's
`CHANGELOG.md` section. Because the tag push publishes:

- Gate request: `make release-check` (config check and snapshot build) is a required
  check, with its evidence line.
- Gate close: after applying the version and before tagging, `make release-notes` must
  succeed.

## Conventions

- Return errors; wrap with `fmt.Errorf("doing x: %w", err)`. No panics in library code.
- Accept `context.Context` as the first parameter of anything that blocks or does I/O.
- Table-driven tests with `t.Run`.
- Keep core logic pure: domain and parsing packages import no UI, storage, or network
  packages, and take the current time (or a clock) and any randomness as parameters
  instead of calling `time.Now()`. Tests then need no sleeps or real clocks.
- Fuzz-test parsers and decoders (`func FuzzX(f *testing.F)`).
- When an interface has two or more implementations (a real store and an in-memory
  one), one shared contract test suite runs against each. Add new behaviour there first.
- Golden files: compare rendered or serialized output with `testdata/*.golden`; rewrite
  them when `UPDATE_GOLDEN=1` is set (`make golden`), only after an intended change,
  and review the diff before committing.
- A binary that keeps state takes a data-directory flag. Agents always run it against
  `t.TempDir()` or a scratch directory, never the user's real data.
- With a database: each schema change is a new numbered migration file; never edit a
  migration that has shipped. Schema changes are recorded decisions.
- Never weaken `.golangci.yml` to get a pass. A `//nolint` must name the linter and give
  a reason (`nolintlint` enforces this).

## Gotchas

- `go test` caches results. Use `-count=1` when you need a real re-run.
- `-race` needs cgo and a C toolchain; macOS and the Ubuntu CI runners have one.
- The first `go run tool@version` downloads and builds the tool; later runs are cached.
- A tag pushed to a public module is cached by the module proxy and cannot be changed.
  Fix a bad release with a new version.

## .gitignore

```
/bin/
/dist/
*.test
*.out
coverage.*
*.code-workspace
```
