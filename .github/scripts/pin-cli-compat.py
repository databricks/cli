#!/usr/bin/env python3
"""Resolve floating skills versions for go:embed without changing the checkout.

GoReleaser and the local build tasks pass the resulting overlay to go build.
The GitHub CLI authenticates using GITHUB_TOKEN in release CI, or its local
credentials for developer builds. If the lookup fails, reuse the skills pin
from the newest concrete compatibility entry.
"""

import json
import re
import subprocess
import sys
from pathlib import Path


def fetch_latest_skills_release() -> str:
    return subprocess.check_output(
        [
            "gh",
            "api",
            "repos/databricks/databricks-agent-skills/releases/latest",
            "--jq",
            ".tag_name",
        ],
        text=True,
    ).strip()


def cli_version_key(version: str) -> tuple:
    """Order manifest floors by semver, including numeric prerelease identifiers.

    >>> versions = ["1.9.0", "1.10.0", "1.10.0-rc.10", "1.10.0-rc.2"]
    >>> sorted(versions, key=cli_version_key)
    ['1.9.0', '1.10.0-rc.2', '1.10.0-rc.10', '1.10.0']
    >>> cli_version_key("1.10.0+build") == cli_version_key("1.10.0")
    True
    """
    core, _, prerelease = version.split("+", 1)[0].partition("-")
    parts = tuple(map(int, core.split(".")))
    identifiers = tuple((0, int(part)) if part.isdecimal() else (1, part) for part in prerelease.split("."))
    return parts + (0,) * (3 - len(parts)), not prerelease, identifiers


def pin_latest_skills(manifest: dict, fetch_latest=fetch_latest_skills_release) -> None:
    """Resolve once for every floating range, preserving pins and other fields.

    >>> manifest = {
    ...     "1.22.0": {"appkit": "0.84.0", "skills": "latest"},
    ...     "1.23.0": {"skills": "latest"},
    ...     "1.20.0": {"skills": "0.2.30"},
    ... }
    >>> from unittest.mock import Mock
    >>> fetch = Mock(return_value="v0.2.32")
    >>> pin_latest_skills(manifest, fetch)
    >>> manifest
    {'1.22.0': {'appkit': '0.84.0', 'skills': '0.2.32', 'skills_resolved_from_latest': True}, '1.23.0': {'skills': '0.2.32', 'skills_resolved_from_latest': True}, '1.20.0': {'skills': '0.2.30'}}
    >>> fetch.call_count
    1
    >>> pin_latest_skills(manifest, fetch)
    >>> fetch.call_count
    1
    >>> pin_latest_skills({"1.22.0": {"skills": "latest"}}, lambda: "0.2.33")
    >>> for tag in ("latest", "main", "", "v1.2", "v01.2.3"):
    ...     try:
    ...         pin_latest_skills({"1.22.0": {"skills": "latest"}}, lambda: tag)
    ...     except ValueError:
    ...         pass
    ...     else:
    ...         raise AssertionError(tag)
    >>> pinned = {
    ...     "1.9.0": {"skills": "0.3.0"},
    ...     "1.22.0": {"appkit": "0.84.0", "skills": "latest"},
    ...     "1.20.0": {"skills": "0.2.30"},
    ... }
    >>> import contextlib, io
    >>> with contextlib.redirect_stderr(io.StringIO()) as warning:
    ...     pin_latest_skills(pinned, Mock(side_effect=FileNotFoundError("gh")))
    >>> pinned["1.22.0"]
    {'appkit': '0.84.0', 'skills': '0.2.30', 'skills_resolved_from_latest': True}
    >>> "1.20.0" in warning.getvalue()
    True
    >>> for fetch in (Mock(side_effect=subprocess.CalledProcessError(1, "gh")), lambda: "latest"):
    ...     manifest = {"1.20.0": {"skills": "0.2.30"}, "1.22.0": {"skills": "latest"}}
    ...     with contextlib.redirect_stderr(io.StringIO()):
    ...         pin_latest_skills(manifest, fetch)
    ...     assert manifest["1.22.0"]["skills"] == "0.2.30"
    >>> pin_latest_skills({"1.22.0": {"skills": "latest"}}, Mock(side_effect=FileNotFoundError("gh")))
    Traceback (most recent call last):
        ...
    ValueError: cannot resolve latest skills release and cli-compat.json has no concrete skills version
    """
    floating = [entry for entry in manifest.values() if entry.get("skills") == "latest"]
    if not floating:
        return

    try:
        version = fetch_latest().removeprefix("v")
        if not re.fullmatch(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", version):
            raise ValueError(f"latest skills release must be a concrete stable version, got {version!r}")
    except (OSError, subprocess.CalledProcessError, ValueError) as err:
        concrete = {floor: entry["skills"] for floor, entry in manifest.items() if entry["skills"] != "latest"}
        if not concrete:
            raise ValueError(
                "cannot resolve latest skills release and cli-compat.json has no concrete skills version"
            ) from err
        floor = max(concrete, key=cli_version_key)
        version = concrete[floor]
        print(f"Could not resolve latest skills release ({err}); embedding {version} from CLI {floor}", file=sys.stderr)
    for entry in floating:
        entry["skills"] = version
        entry["skills_resolved_from_latest"] = True


def build_overlay(root: Path, fetch_latest=fetch_latest_skills_release) -> None:
    """Write resolved build inputs while leaving the checked-in manifest intact.

    >>> from tempfile import TemporaryDirectory
    >>> from unittest.mock import Mock
    >>> with TemporaryDirectory() as tmp:
    ...     root = Path(tmp).resolve()
    ...     source = root / "internal/build/cli-compat.json"
    ...     source.parent.mkdir(parents=True)
    ...     original = '{"1.22.0": {"appkit": "0.84.0", "skills": "latest"}}'
    ...     _ = source.write_text(original)
    ...     build_overlay(root, lambda: "v0.2.32")
    ...     overlay = json.loads((root / ".databricks/cli-compat/overlay.json").read_text())
    ...     resolved = Path(overlay["Replace"][str(source)])
    ...     assert json.loads(resolved.read_text())["1.22.0"]["skills"] == "0.2.32"
    ...     assert source.read_text() == original

    Release CI can build tags from before cli-compat.json was introduced.

    >>> with TemporaryDirectory() as tmp:
    ...     root = Path(tmp).resolve()
    ...     build_overlay(root, Mock(side_effect=AssertionError("must not fetch")))
    ...     assert json.loads((root / ".databricks/cli-compat/overlay.json").read_text()) == {"Replace": {}}

    An all-pinned manifest must build without GitHub credentials or even gh.

    >>> with TemporaryDirectory() as tmp:
    ...     root = Path(tmp).resolve()
    ...     source = root / "internal/build/cli-compat.json"
    ...     source.parent.mkdir(parents=True)
    ...     _ = source.write_text('{"1.22.0": {"skills": "0.2.30"}}')
    ...     build_overlay(root, Mock(side_effect=AssertionError("must not fetch")))
    ...     resolved = json.loads((root / ".databricks/cli-compat/cli-compat.json").read_text())
    ...     assert resolved["1.22.0"]["skills"] == "0.2.30"
    """
    source = root / "internal/build/cli-compat.json"
    output = root / ".databricks/cli-compat"
    output.mkdir(parents=True, exist_ok=True)
    replacements = {}
    if source.exists():
        manifest = json.loads(source.read_text(encoding="utf-8"))
        pin_latest_skills(manifest, fetch_latest)
        resolved = output / "cli-compat.json"
        resolved.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
        replacements[str(source)] = str(resolved)
    (output / "overlay.json").write_text(json.dumps({"Replace": replacements}, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    build_overlay(Path.cwd().resolve())
