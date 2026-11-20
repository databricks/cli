"""pytest plugin: the ``cloud_only`` marker.

Assertions that depend on cloud-only behavior (SQL dialect/schema, SLA timing,
permissions) are marked ``cloud_only``. On any non-cloud backend they are **skipped
with a visible reason** rather than silently passing, so a green local run never
implies false confidence.
"""

from __future__ import annotations

from pathlib import Path

import pytest
from _pytest.outcomes import Skipped

from bundletest.backend import LocalUnsupported
from bundletest.env import current_backend_kind


def pytest_addoption(parser: pytest.Parser) -> None:
    group = parser.getgroup("bundletest")
    group.addoption(
        "--bundletest-resource",
        action="append",
        default=[],
        help="collect tests marked for this bundle resource (repeatable)",
    )
    group.addoption(
        "--bundletest-test-file",
        action="append",
        default=[],
        help="collect this changed test file even when it has no resource marker",
    )


def pytest_configure(config: pytest.Config) -> None:
    config.addinivalue_line(
        "markers",
        "cloud_only: assertion depends on cloud-only behavior; skipped unless BUNDLETEST_BACKEND=cloud",
    )
    config.addinivalue_line(
        "markers",
        "bundle_resource(name): resource exercised by this test, such as jobs.transform_orders",
    )


@pytest.hookimpl(hookwrapper=True)
def pytest_runtest_call(item: pytest.Item):
    """Turn a LocalUnsupported escaping a test into a skip-with-reason, never a failure.

    This is the third routing arm: something the local backend can't judge (notebook task,
    Databricks-only function, reserved catalog) skips loudly instead of false-red-ing."""
    outcome = yield
    excinfo = outcome.excinfo
    if excinfo is not None and issubclass(excinfo[0], LocalUnsupported):
        outcome.force_exception(Skipped(msg=str(excinfo[1])))


def pytest_collection_modifyitems(config: pytest.Config, items: list[pytest.Item]) -> None:
    _select_changed_items(config, items)
    backend = current_backend_kind()
    if backend == "cloud":
        return
    skip = pytest.mark.skip(reason=f"cloud_only: needs cloud backend (BUNDLETEST_BACKEND={backend})")
    for item in items:
        if "cloud_only" in item.keywords:
            item.add_marker(skip)


def _select_changed_items(config: pytest.Config, items: list[pytest.Item]) -> None:
    resources = set(config.getoption("--bundletest-resource"))
    test_files = {Path(path).resolve() for path in config.getoption("--bundletest-test-file")}
    if not resources and not test_files:
        return

    selected: list[pytest.Item] = []
    deselected: list[pytest.Item] = []
    for item in items:
        marked = {str(argument) for marker in item.iter_markers("bundle_resource") for argument in marker.args}
        if marked.intersection(resources) or Path(str(item.path)).resolve() in test_files:
            selected.append(item)
        else:
            deselected.append(item)
    items[:] = selected
    if deselected:
        config.hook.pytest_deselected(items=deselected)
