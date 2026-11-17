# Logging (Python profile)

Read on demand, when adding or changing logging. The short rules are in `PROFILE.md`.

The project uses the standard library `logging` module with a rotating file handler.
Configuration lives in one place, `src/<pkg>/logging_config.py`, copied from
`profiles/python/starter/logging_config.py` at setup. Modules never configure logging.

## Setup

`setup_logging()` attaches two handlers to the root logger:

- a rotating file at `out/logs/app.log`: DEBUG and up, 5 MB per file, 5 backups,
  format `time | logger | LEVEL | message`;
- the console (stderr): INFO and up, format `LEVEL | message`.

Calling it again replaces those two handlers instead of adding duplicates, so tests may
call it freely. `out/` is gitignored; log files are never committed.

## Initialization

Call it once, first thing in the CLI entrypoint, before other work runs:

```python
# src/<pkg>/cli.py
import logging

from <pkg>.logging_config import setup_logging

logger = logging.getLogger(__name__)


def main() -> None:
    setup_logging()
    logger.info("Starting")
    ...
```

## Usage in modules

Each module gets its own logger named after the module, so output shows the full path
(`pkg.sub.module`):

```python
import logging

logger = logging.getLogger(__name__)


def do_work(item: str) -> None:
    logger.info("Starting work")
    logger.debug("Processing %s", item)
```

## Rules

- **One setup call.** Only the entrypoint calls `setup_logging()`. Modules do not call
  it, add handlers, set levels, or use the root logger.
- **`__name__` loggers only.** Always `logging.getLogger(__name__)`.
- **Lazy formatting.** `logger.info("x=%s", x)`, not f-strings or `%` pre-formatting,
  so filtered messages cost nothing.
- **Exceptions.** Inside `except`, use `logger.exception("...")` to keep the traceback.
- **`print()` is for CLI output only.** Results, tables, or JSON the user asked for go
  to stdout with `print()`. Everything diagnostic goes through a logger.

## Levels

| Level | Use for |
|---|---|
| `DEBUG` | Detailed diagnostics: variable values, flow tracing |
| `INFO` | Routine events: startup, shutdown, task completion |
| `WARNING` | Unexpected but recoverable situations |
| `ERROR` | Failures that stop one operation |
| `CRITICAL` | Failures that may force the application to stop |
