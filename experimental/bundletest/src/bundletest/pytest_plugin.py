"""pytest plugin: the ``cloud_only`` marker.

Assertions that depend on cloud-only behavior (SQL dialect/schema, SLA timing,
permissions) are marked ``cloud_only``. On any non-cloud backend they are **skipped
with a visible reason** rather than silently passing, so a green local run never
implies false confidence.
"""

from __future__ import annotations

import pytest
from _pytest.outcomes import Skipped

from bundletest.backend import LocalUnsupported
from bundletest.env import current_backend_kind


def pytest_configure(config: pytest.Config) -> None:
    config.addinivalue_line(
        "markers",
        "cloud_only: assertion depends on cloud-only behavior; skipped unless BUNDLETEST_BACKEND=cloud",
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
    backend = current_backend_kind()
    if backend == "cloud":
        return
    skip = pytest.mark.skip(reason=f"cloud_only: needs cloud backend (BUNDLETEST_BACKEND={backend})")
    for item in items:
        if "cloud_only" in item.keywords:
            item.add_marker(skip)
