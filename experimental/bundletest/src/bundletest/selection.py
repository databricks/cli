"""Map changed bundle files to pytest resource markers."""

from __future__ import annotations

import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable

from bundletest.project import load_bundle_config

_PATH_KEYS = {"file_path", "notebook_path", "path", "source_code_path"}


@dataclass(frozen=True)
class ChangeSelection:
    changed_files: tuple[str, ...]
    resources: tuple[str, ...]
    test_files: tuple[str, ...]
    run_all: bool = False

    @property
    def is_empty(self) -> bool:
        return not self.run_all and not self.resources and not self.test_files


def git_changed_files(root: Path, base: str) -> list[str]:
    """Return committed, staged, unstaged, and untracked paths relative to the bundle."""
    repo = _git(root, "rev-parse", "--show-toplevel").strip()
    repo_root = Path(repo)
    paths: set[str] = set()
    commands = (
        ("diff", "--name-only", "--diff-filter=ACMR", f"{base}...HEAD"),
        ("diff", "--name-only", "--diff-filter=ACMR"),
        ("diff", "--cached", "--name-only", "--diff-filter=ACMR"),
        ("ls-files", "--others", "--exclude-standard"),
    )
    for args in commands:
        for name in _git(root, *args).splitlines():
            absolute = (repo_root / name).resolve()
            if absolute == root or root in absolute.parents:
                paths.add(absolute.relative_to(root).as_posix())
    return sorted(paths)


def select_changes(root: Path, changed_files: Iterable[str]) -> ChangeSelection:
    changed = tuple(sorted(set(changed_files)))
    config = load_bundle_config(root)
    resources = config.get("resources", {}) or {}
    all_resources = tuple(
        f"{kind}.{name}" for kind, declared in resources.items() if isinstance(declared, dict) for name in declared
    )

    if any(Path(name).suffix in {".yml", ".yaml"} for name in changed):
        return ChangeSelection(changed, all_resources, (), run_all=True)

    references = _resource_references(root, resources)
    affected: set[str] = set()
    tests: set[str] = set()
    for name in changed:
        path = (root / name).resolve()
        if _is_test_file(root, path):
            tests.add(path.as_posix())
        for resource, targets in references.items():
            if any(path == target or target in path.parents for target in targets):
                affected.add(resource)
    return ChangeSelection(changed, tuple(sorted(affected)), tuple(sorted(tests)))


def format_change_selection(selection: ChangeSelection, base: str) -> str:
    lines = [f"bundletest changed selection: {base}"]
    if selection.run_all:
        lines.append("bundle configuration changed; running the complete suite")
        return "\n".join(lines)
    for resource in selection.resources:
        lines.append(f"[RESOURCE] {resource}")
    for test in selection.test_files:
        lines.append(f"[TEST    ] {test}")
    if selection.is_empty:
        lines.append("no bundle resources or tests were affected")
    return "\n".join(lines)


def _git(root: Path, *args: str) -> str:
    result = subprocess.run(["git", *args], cwd=root, capture_output=True, text=True, check=False)
    if result.returncode:
        detail = result.stderr.strip() or result.stdout.strip()
        raise RuntimeError(f"git {' '.join(args)} failed: {detail}")
    return result.stdout


def _resource_references(root: Path, resources: dict[str, Any]) -> dict[str, tuple[Path, ...]]:
    references: dict[str, tuple[Path, ...]] = {}
    for kind, declared in resources.items():
        if not isinstance(declared, dict):
            continue
        for name, config in declared.items():
            paths = tuple((root / value).resolve() for value in _path_values(config))
            references[f"{kind}.{name}"] = paths
    return references


def _path_values(node: Any, key: str = "") -> Iterable[str]:
    if isinstance(node, dict):
        for child_key, value in node.items():
            yield from _path_values(value, child_key)
    elif isinstance(node, list):
        for value in node:
            yield from _path_values(value, key)
    elif key in _PATH_KEYS and isinstance(node, str) and "://" not in node:
        yield node


def _is_test_file(root: Path, path: Path) -> bool:
    tests = root / "tests"
    return path.suffix == ".py" and (path == tests or tests in path.parents)
