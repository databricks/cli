#!/usr/bin/env python3
"""Resolve floating skills versions for go:embed without changing the checkout.

GoReleaser and the local build tasks pass the resulting overlay to go build.
The GitHub CLI authenticates using GITHUB_TOKEN in release CI, or its local
credentials for developer builds. If the lookup fails, "latest" stays in the
manifest and the CLI falls back to the newest pin below that range.
"""

import json
import os
import re
import subprocess
import sys
from pathlib import Path

SOURCE = Path("internal/build/cli-compat.json")
OUTPUT = Path(".databricks/cli-compat")


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
        timeout=60,
    ).strip()


def pin_latest_skills(manifest: dict, fetch_latest=fetch_latest_skills_release):
    """Resolve once for every floating range, preserving pins and other fields.

    Returns a warning if the lookup fails.

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

    A failed lookup or an invalid tag leaves "latest" in place.

    >>> failures = [
    ...     Mock(side_effect=FileNotFoundError("gh")),
    ...     Mock(side_effect=subprocess.CalledProcessError(1, "gh")),
    ...     Mock(side_effect=subprocess.TimeoutExpired("gh", 60)),
    ...     *(Mock(return_value=tag) for tag in ("latest", "main", "", "v1.2", "v01.2.3")),
    ... ]
    >>> for fetch in failures:
    ...     manifest = {"1.20.0": {"skills": "0.2.30"}, "1.22.0": {"appkit": "0.84.0", "skills": "latest"}}
    ...     assert "Could not resolve the latest skills release" in pin_latest_skills(manifest, fetch), fetch
    ...     assert manifest["1.22.0"] == {"appkit": "0.84.0", "skills": "latest"}, fetch
    """
    floating = [entry for entry in manifest.values() if entry.get("skills") == "latest"]
    if not floating:
        return

    try:
        version = fetch_latest().removeprefix("v")
        if not re.fullmatch(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", version):
            raise ValueError(f"latest skills release must be a concrete stable version, got {version!r}")
    except (OSError, subprocess.SubprocessError, ValueError) as err:
        return f"Could not resolve the latest skills release ({err}). The CLI falls back to the newest pinned skills version."
    for entry in floating:
        entry["skills"] = version
        entry["skills_resolved_from_latest"] = True


def build_overlay(root: Path, fetch_latest=fetch_latest_skills_release):
    """Write resolved build inputs while leaving the checked-in manifest intact.

    Returns the warning from pin_latest_skills, if any.

    >>> from tempfile import TemporaryDirectory
    >>> from unittest.mock import Mock
    >>> with TemporaryDirectory() as tmp:
    ...     root = Path(tmp)
    ...     source = root / SOURCE
    ...     source.parent.mkdir(parents=True)
    ...     original = '{"1.22.0": {"appkit": "0.84.0", "skills": "latest"}}'
    ...     _ = source.write_text(original)
    ...     build_overlay(root, lambda: "v0.2.32")
    ...     overlay = json.loads((root / OUTPUT / "overlay.json").read_text())
    ...     assert overlay == {"Replace": {"internal/build/cli-compat.json": ".databricks/cli-compat/cli-compat.json"}}
    ...     assert json.loads((root / OUTPUT / "cli-compat.json").read_text())["1.22.0"]["skills"] == "0.2.32"
    ...     assert source.read_text() == original

    Release CI can build tags from before cli-compat.json was introduced.

    >>> with TemporaryDirectory() as tmp:
    ...     root = Path(tmp)
    ...     build_overlay(root, Mock(side_effect=AssertionError("must not fetch")))
    ...     assert json.loads((root / OUTPUT / "overlay.json").read_text()) == {"Replace": {}}

    An all-pinned manifest must build without GitHub credentials or even gh.

    >>> with TemporaryDirectory() as tmp:
    ...     root = Path(tmp)
    ...     source = root / SOURCE
    ...     source.parent.mkdir(parents=True)
    ...     _ = source.write_text('{"1.22.0": {"skills": "0.2.30"}}')
    ...     build_overlay(root, Mock(side_effect=AssertionError("must not fetch")))
    ...     assert json.loads((root / OUTPUT / "cli-compat.json").read_text())["1.22.0"]["skills"] == "0.2.30"
    """
    source = root / SOURCE
    output = root / OUTPUT
    output.mkdir(parents=True, exist_ok=True)
    replacements = {}
    warning = None
    if source.exists():
        manifest = json.loads(source.read_text(encoding="utf-8"))
        warning = pin_latest_skills(manifest, fetch_latest)
        (output / "cli-compat.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
        # Relative paths, because go build matches overlay entries against its
        # own working directory, which may be a symlink that resolve() expands.
        replacements[SOURCE.as_posix()] = (OUTPUT / "cli-compat.json").as_posix()
    (output / "overlay.json").write_text(json.dumps({"Replace": replacements}, indent=2) + "\n", encoding="utf-8")
    return warning


def warn(message: str) -> None:
    print(f"WARNING: {message}", file=sys.stderr)
    # GoReleaser shows hook output only with --verbose, so release builds
    # surface the warning in the GitHub Actions job summary instead.
    if summary := os.environ.get("GITHUB_STEP_SUMMARY"):
        with open(summary, "a", encoding="utf-8") as f:
            f.write(f"> [!WARNING]\n> {message}\n")


if __name__ == "__main__":
    if warning := build_overlay(Path.cwd()):
        warn(warning)
