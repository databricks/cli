"""pytest plugin: the ``cloud_only`` marker.

Assertions that depend on cloud-only behavior (SQL dialect/schema, SLA timing,
permissions) are marked ``cloud_only``. On any non-cloud backend they are **skipped
with a visible reason** rather than silently passing, so a green local run never
implies false confidence.
"""

from __future__ import annotations

import pytest

from bundletest.env import current_backend_kind


def pytest_configure(config: pytest.Config) -> None:
    config.addinivalue_line(
        "markers",
        "cloud_only: assertion depends on cloud-only behavior; skipped unless "
        "BUNDLETEST_BACKEND=cloud",
    )


def pytest_collection_modifyitems(
    config: pytest.Config, items: list[pytest.Item]
) -> None:
    backend = current_backend_kind()
    if backend == "cloud":
        return
    skip = pytest.mark.skip(
        reason=f"cloud_only: needs cloud backend (BUNDLETEST_BACKEND={backend})"
    )
    for item in items:
        if "cloud_only" in item.keywords:
            item.add_marker(skip)
