"""Command-line workflow for running and scaffolding bundle tests."""

from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path
from typing import Sequence

from bundletest.project import ProjectError, find_bundle_root
from bundletest.scaffold import create_starter
from bundletest.selection import (
    format_change_selection,
    git_changed_files,
    select_changes,
)
from bundletest.support import format_support_report, inspect_local_support


def main(argv: Sequence[str] | None = None) -> int:
    args = list(argv if argv is not None else sys.argv[1:])
    try:
        if args and args[0] == "init":
            return _init(args[1:])
        return _run(args)
    except (FileExistsError, ProjectError, RuntimeError, ValueError) as err:
        print(f"bundletest: {err}", file=sys.stderr)
        return 2


def _run(argv: Sequence[str]) -> int:
    parser = _run_parser()
    options, pytest_args = parser.parse_known_args(argv)
    _validate_options(options)
    root = find_bundle_root(options.bundle)
    backend = "cloud" if options.cloud else "local"
    _configure_environment(options, backend)

    report = inspect_local_support(root)
    if not options.no_support_report:
        print(format_support_report(root, report))
        print()
    if options.support_only:
        return 1 if any(entry.status == "error" for entry in report.entries) else 0

    selection = None
    if options.changed:
        base = options.base or "HEAD~1"
        changed = git_changed_files(root, base)
        selection = select_changes(root, changed)
        print(format_change_selection(selection, base))
        print()
        if selection.is_empty:
            return 0

    arguments = list(pytest_args)
    if arguments and arguments[0] == "--":
        arguments.pop(0)
    if not _has_test_path(arguments, root):
        tests = root / "tests"
        arguments.append(str(tests if tests.is_dir() else root))
    if selection and not selection.run_all:
        arguments.extend(f"--bundletest-resource={name}" for name in selection.resources)
        arguments.extend(f"--bundletest-test-file={name}" for name in selection.test_files)

    import pytest

    print(f"bundletest: running {backend} tests")
    previous = Path.cwd()
    os.chdir(root)
    try:
        return int(pytest.main(arguments))
    finally:
        os.chdir(previous)


def _init(argv: Sequence[str]) -> int:
    parser = argparse.ArgumentParser(
        prog="bundletest init",
        description="Generate starter pytest files for an existing Databricks bundle",
    )
    parser.add_argument("path", nargs="?", default=".", help="bundle directory or a child path")
    parser.add_argument("--force", action="store_true", help="replace the generated starter files if they exist")
    options = parser.parse_args(argv)
    root = find_bundle_root(options.path)
    generated = create_starter(root, force=options.force)
    print(f"bundletest: generated starter for {generated.resource}")
    for path in generated.files:
        print(f"  {path.relative_to(root)}")
    print("next: bundletest --local")
    return 0


def _run_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="bundletest",
        description="Run pytest against a Databricks bundle locally or in a workspace",
    )
    parser.add_argument("--bundle", default=".", help="bundle directory or a child path (default: current directory)")
    backend = parser.add_mutually_exclusive_group()
    backend.add_argument("--local", action="store_true", help="run with the local DuckDB backend")
    backend.add_argument("--cloud", action="store_true", help="run against a Databricks workspace")
    parser.add_argument("--profile", help="explicit Databricks CLI profile for --cloud")
    parser.add_argument("--warehouse-id", help="SQL warehouse used by cloud tests")
    parser.add_argument("--target", help="bundle target used by cloud tests")
    parser.add_argument("--changed", action="store_true", help="run tests affected by files changed from --base")
    parser.add_argument("--base", help="Git base for --changed (default: HEAD~1)")
    parser.add_argument("--support-only", action="store_true", help="print local support without running pytest")
    parser.add_argument("--no-support-report", action="store_true", help="do not print the local support report")
    return parser


def _validate_options(options: argparse.Namespace) -> None:
    if options.base and not options.changed:
        raise ValueError("--base requires --changed")
    if options.support_only and options.no_support_report:
        raise ValueError("--support-only cannot be used with --no-support-report")


def _configure_environment(options: argparse.Namespace, backend: str) -> None:
    if backend == "local":
        incompatible = [
            flag
            for flag, value in (
                ("--profile", options.profile),
                ("--warehouse-id", options.warehouse_id),
                ("--target", options.target),
            )
            if value
        ]
        if incompatible:
            raise ValueError(f"{', '.join(incompatible)} can only be used with --cloud")
    elif not options.profile:
        raise ValueError("--cloud requires an explicit --profile")

    os.environ["BUNDLETEST_BACKEND"] = backend
    _set_or_clear("BUNDLETEST_PROFILE", options.profile)
    _set_or_clear("BUNDLETEST_WAREHOUSE_ID", options.warehouse_id)
    _set_or_clear("BUNDLETEST_TARGET", options.target)


def _set_or_clear(name: str, value: str | None) -> None:
    if value is None:
        os.environ.pop(name, None)
    else:
        os.environ[name] = value


def _has_test_path(arguments: Sequence[str], root: Path) -> bool:
    for argument in arguments:
        if argument.startswith("-"):
            continue
        path_text = argument.split("::", 1)[0]
        path = Path(path_text)
        if (path if path.is_absolute() else root / path).exists():
            return True
    return False


if __name__ == "__main__":
    raise SystemExit(main())
