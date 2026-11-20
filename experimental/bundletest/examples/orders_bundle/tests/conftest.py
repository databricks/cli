"""Shared fixture for the example gallery: deploy the bundle once per test."""

from pathlib import Path

import pytest
from bundletest import bundle_env
from bundletest.env import current_backend_kind

BUNDLE = str(Path(__file__).resolve().parent.parent)

# This gallery declares one of every resource kind for the local backend to READ; several
# aren't deployable (an external location, an alert query_id, models), so it is never actually
# deployed. The cloud backend really runs `bundle deploy`, so skip the gallery there — the
# cloud fixture is examples/cloud_orders.
if current_backend_kind() == "cloud":
    collect_ignore_glob = ["test_*.py"]


@pytest.fixture
def env():
    with bundle_env(BUNDLE) as e:
        yield e
