"""Fixture for the cloud-backend end-to-end validation.

These tests really deploy + run against a workspace, so they are collected only when
BUNDLETEST_BACKEND=cloud. The env is module-scoped (deploy once — deploys take minutes and
cost compute). Each run gets a UNIQUE schema (and bundle name) so two concurrent runs, or any
shared use of the workspace, never collide; the schema is created up front and dropped CASCADE
afterwards, so cleanup is unambiguous.
"""

import shutil
import tempfile
import uuid
import warnings
from pathlib import Path

import pytest
from bundletest import BundleEnv
from bundletest.env import current_backend_kind, make_backend

BUNDLE = str(Path(__file__).resolve().parent.parent)
RUN_ID = uuid.uuid4().hex[:8]
SCHEMA = f"main.bundletest_cloud_{RUN_ID}"

# Nothing here runs on the local backend — skip collection entirely so CI stays local-only.
if current_backend_kind() != "cloud":
    collect_ignore_glob = ["test_*.py"]


@pytest.fixture(scope="module")
def schema() -> str:
    """The unique target schema for this run (main.bundletest_cloud_<run>)."""
    return SCHEMA


@pytest.fixture(scope="module")
def env():
    backend = make_backend("cloud")
    # Deploy from a per-run copy with the `bundletest_cloud` schema and `bundletest-cloud` bundle
    # name suffixed by RUN_ID. The .sql/.lvdash.json artifacts hardcode the schema (DABs doesn't
    # interpolate file contents), so substituting in a copy is how each run gets its own tables,
    # volume, dashboard, and deploy path — the committed fixture keeps the readable placeholder.
    tmp = Path(tempfile.mkdtemp(prefix="bundletest-cloud-"))
    bundle_dir = tmp / "bundle"
    shutil.copytree(BUNDLE, bundle_dir)
    for path in bundle_dir.rglob("*"):
        if path.suffix in (".yml", ".sql", ".json"):
            text = path.read_text()
            text = text.replace("bundletest_cloud", f"bundletest_cloud_{RUN_ID}")
            text = text.replace("bundletest-cloud", f"bundletest-cloud-{RUN_ID}")
            path.write_text(text)

    backend.execute_sql(f"CREATE SCHEMA IF NOT EXISTS {SCHEMA}")
    e = BundleEnv(str(bundle_dir), backend)
    e.deploy()
    try:
        yield e
    finally:
        e.teardown()
        try:
            backend.execute_sql(f"DROP SCHEMA IF EXISTS {SCHEMA} CASCADE")
        except Exception as exc:
            warnings.warn(f"failed to drop {SCHEMA} (may be leaked): {exc}", stacklevel=1)
        shutil.rmtree(tmp, ignore_errors=True)
