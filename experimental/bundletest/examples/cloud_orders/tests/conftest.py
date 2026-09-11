"""Fixture for the cloud-backend end-to-end validation.

These tests really deploy + run against a workspace, so they are collected only when
BUNDLETEST_BACKEND=cloud; on the local backend there is nothing to run. The env is
module-scoped (deploy once — deploys take minutes and cost compute, unlike the local
backend's fresh per-test DuckDB). The target schema is created up front and dropped
CASCADE afterwards, so the run leaves nothing behind even if `bundle destroy` half-fails.
"""

from pathlib import Path

import pytest
from bundletest import BundleEnv
from bundletest.env import current_backend_kind, make_backend

BUNDLE = str(Path(__file__).resolve().parent.parent)
SCHEMA = "main.bundletest_cloud"

# Nothing here runs on the local backend — skip collection entirely so CI stays local-only.
if current_backend_kind() != "cloud":
    collect_ignore_glob = ["test_*.py"]


@pytest.fixture(scope="module")
def env():
    backend = make_backend("cloud")
    backend.execute_sql(f"CREATE SCHEMA IF NOT EXISTS {SCHEMA}")
    e = BundleEnv(BUNDLE, backend)
    e.deploy()
    try:
        yield e
    finally:
        e.teardown()
        try:
            backend.execute_sql(f"DROP SCHEMA IF EXISTS {SCHEMA} CASCADE")
        except Exception:
            pass
