"""Bundle project discovery and configuration loading."""

from __future__ import annotations

from pathlib import Path
from typing import Any

from bundletest.backends.duckdb import _resolve_config


class ProjectError(ValueError):
    """The requested path is not a usable bundle project."""


def find_bundle_root(start: str | Path = ".") -> Path:
    """Return the nearest parent containing ``databricks.yml``."""
    path = Path(start).resolve()
    if path.is_file():
        path = path.parent
    for candidate in (path, *path.parents):
        if (candidate / "databricks.yml").is_file():
            return candidate
    raise ProjectError(f"no databricks.yml found at or above {path}")


def load_bundle_config(root: Path) -> dict[str, Any]:
    """Resolve bundle configuration offline with the CLI's bundle engine."""
    return _resolve_config(str(root))
