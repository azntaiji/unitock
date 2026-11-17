"""Central logging configuration.

Copy into ``src/<pkg>/logging_config.py`` at setup. Call :func:`setup_logging` once, at
the top of the CLI entrypoint. Modules only call ``logging.getLogger(__name__)``.
"""

from __future__ import annotations

import logging
from logging.handlers import RotatingFileHandler
from pathlib import Path

LOG_DIR = Path("out/logs")
LOG_FILE = "app.log"
MAX_BYTES = 5 * 1024 * 1024
BACKUP_COUNT = 5

_FILE_HANDLER = "app-file"
_CONSOLE_HANDLER = "app-console"


def setup_logging(
    level: int = logging.DEBUG,
    console_level: int = logging.INFO,
    log_dir: Path = LOG_DIR,
) -> None:
    """Configure the root logger with a rotating file handler and a console handler.

    Safe to call more than once: handlers added by an earlier call are replaced, not
    duplicated.

    Args:
        level: Root logger level and file handler level.
        console_level: Minimum level written to the console (stderr).
        log_dir: Directory for the rotating log file. Created if missing.
    """
    log_dir.mkdir(parents=True, exist_ok=True)

    root = logging.getLogger()
    root.setLevel(level)
    for handler in list(root.handlers):
        if handler.get_name() in (_FILE_HANDLER, _CONSOLE_HANDLER):
            root.removeHandler(handler)
            handler.close()

    file_handler = RotatingFileHandler(
        log_dir / LOG_FILE,
        maxBytes=MAX_BYTES,
        backupCount=BACKUP_COUNT,
        encoding="utf-8",
    )
    file_handler.set_name(_FILE_HANDLER)
    file_handler.setLevel(level)
    file_handler.setFormatter(
        logging.Formatter("%(asctime)s | %(name)s | %(levelname)-8s | %(message)s")
    )

    console_handler = logging.StreamHandler()
    console_handler.set_name(_CONSOLE_HANDLER)
    console_handler.setLevel(console_level)
    console_handler.setFormatter(logging.Formatter("%(levelname)-8s | %(message)s"))

    root.addHandler(file_handler)
    root.addHandler(console_handler)
