# Profile: Python

## Toolchain

- Python 3.11 or later. State the floor in `pyproject.toml` `requires-python`.
- Virtual environment at `.venv`. Always invoke tools as `.venv/bin/python -m <tool>` so
  the right interpreter is used whatever the shell's state.
- `ruff` for formatting and linting, `pytest` for tests, `mypy` for type checking.
- Dev tools are declared under `[project.optional-dependencies] dev` in `pyproject.toml`.
- Optional, for humans: `direnv` with an `.envrc` of `source .venv/bin/activate`, then
  `direnv allow`. Agent commands never rely on an activated shell.

## Commands

Copy into the Commands table in `AGENTS.md`.

| Task | Command |
|---|---|
| Setup | `python3 -m venv .venv && .venv/bin/python -m pip install --upgrade pip && .venv/bin/python -m pip install -e ".[dev]"` |
| Format | `.venv/bin/python -m ruff format .` |
| Lint | `.venv/bin/python -m ruff check .` |
| Typecheck | `.venv/bin/python -m mypy src` |
| Test (all) | `.venv/bin/python -m pytest -q` |
| Test (single) | `.venv/bin/python -m pytest path/to/test_file.py::test_name -q` |
| Build | `.venv/bin/python -m build` |

## Layout

```
src/<pkg>/          __init__.py, cli.py (main), config.py, logging_config.py
tests/              pytest tests
data/               input files
out/                generated output, gitignored; rotating logs in out/logs/
scripts/            thin wrappers for IDE run configurations, beside check_scaffold.py
.env.example        every setting the app reads, no secrets; .env stays ignored
pyproject.toml      all tool configuration
```

Use the `src` layout so tests run against the installed package. At setup, copy
`starter/logging_config.py` into `src/<pkg>/`. Add to `pyproject.toml`:

```toml
[project.scripts]
<pkg> = "<pkg>.cli:main"

[tool.pytest.ini_options]
testpaths = ["tests"]  # keeps pytest out of scripts/test_check_scaffold.py
```

## Logging

Full guide and rationale: `profiles/python/logging.md`. Read it before adding logging.

- Call `setup_logging()` once, at the top of `cli.main()`. Nothing else configures
  logging.
- Every module uses `logger = logging.getLogger(__name__)`. Modules never add handlers,
  set levels, or touch the root logger.
- Lazy formatting: `logger.info("x=%s", x)`, never f-strings in log calls.
- No `print()` for diagnostics. `print()` is only for a CLI's actual output: the results,
  tables, or JSON the user asked for.

## Versioning

`VERSION` holds SemVer. Python packaging requires PEP 440, which writes pre-releases
differently, so the mirrors use the PEP 440 form and the scaffold check converts.

| `VERSION` (SemVer) | `pyproject.toml` and `__version__` (PEP 440) |
|---|---|
| `1.2.0` | `1.2.0` |
| `1.2.0-alpha.1` | `1.2.0a1` |
| `1.2.0-beta.1` | `1.2.0b1` |
| `1.2.0-rc.1` | `1.2.0rc1` |

Other pre-release forms and build metadata have no agreed PEP 440 equivalent; do not use
them. Writing the SemVer form into `pyproject.toml` is a mistake that build tools hide
by silently normalizing it. Versions change only at gate close, never per commit.

```json
"profiles": ["python"],
"version_mirrors": [
  {"path": "pyproject.toml", "kind": "pyproject"},
  {"path": "src/<package_name>/__init__.py", "kind": "python_dunder"}
]
```

Drop the second mirror if the package reads its version from metadata with
`importlib.metadata.version`.

What counts as breaking in Python, for the bump table in `docs/process/versioning.md`:

- Breaking: removing or renaming a public module, class, or function; removing or
  making required a parameter; changing a return type; dropping a Python version;
  changing default behavior users depend on; removing a CLI command or flag.
- Feature: a new public name, a new optional parameter, a new CLI command or flag, a
  new configuration option, a deprecation without removal.
- Fix: a bug fix that keeps the public interface, including behavior brought in line
  with what was already documented.

## Conventions

- Google-style docstrings on public modules, classes, and functions.
- Type hints on all public functions.
- `pathlib` over `os.path`.
- `README.md` documents every CLI argument and flag; update it with the CLI.
- Configuration comes from the environment through `config.py`; document each setting
  in `.env.example`.

## Gotchas

- `python` may not exist on the machine; use `python3` to create the venv and
  `.venv/bin/python` afterwards.
- `pip install -e .` must be re-run after changing dependencies or entry points.
- Run one test file while iterating; run the whole suite before committing.
- Log paths such as `out/logs/` are relative to the working directory; run the CLI from
  the repository root.

## .gitignore

```
.venv/
venv/
.Python
.envrc
__pycache__/
*.py[cod]
*.egg-info/
dist/
build/
out/
.pytest_cache/
.mypy_cache/
.ruff_cache/
.coverage
htmlcov/
*.code-workspace
```
