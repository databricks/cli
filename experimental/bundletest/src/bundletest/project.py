"""Bundle project discovery and configuration loading."""

from __future__ import annotations

from pathlib import Path
from typing import Any

import yaml


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
    """Load the root bundle file for static CLI inspection."""
    path = root / "databricks.yml"
    try:
        raw = yaml.safe_load(path.read_text())
    except yaml.YAMLError as err:
        raise ProjectError(f"failed to parse {path}: {err}") from err
    if raw is None:
        return {}
    if not isinstance(raw, dict):
        raise ProjectError(f"{path} must contain a YAML mapping")
    return raw
