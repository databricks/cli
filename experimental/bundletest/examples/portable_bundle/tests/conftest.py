"""Deploy the portable bundle once per test, on whichever backend BUNDLETEST_BACKEND picks."""

from pathlib import Path

import pytest
from bundletest import bundle_env
from bundletest.env import current_backend_kind

BUNDLE = str(Path(__file__).resolve().parent.parent)


@pytest.fixture
def env():
    # The assertions are backend-agnostic, but a shared workspace needs per-run namespace
    # isolation the fixed `demo` catalog can't give. That isolation fixture lands with the
    # cloud backend PR; until then scope the cloud run out rather than collide on a shared
    # workspace. The local (DuckDB) path is per-process and runs in CI.
    if current_backend_kind() == "cloud":
        pytest.skip("portable_bundle cloud fixture (per-run schema isolation) lands with the cloud backend PR")
    with bundle_env(BUNDLE) as e:
        yield e
