"""This file configures pytest.

This file is in the root since it can be used for tests in any place in this
project, including tests under resources/.
"""

import os, sys, pathlib
from contextlib import contextmanager


try:
    from databricks.connect import DatabricksSession
    from databricks.sdk import WorkspaceClient
    from pyspark.sql import SparkSession
    import pytest
except ImportError:
    raise ImportError("Test dependencies not found.\n\nRun tests using 'uv run pytest'. See http://docs.astral.sh/uv to learn more about uv.")


def add_all_resources_to_sys_path():
    """Add all resources/* directories to sys.path for module discovery."""
    resources = pathlib.Path(__file__).with_name("resources")
    resource_dirs = filter(pathlib.Path.is_dir, resources.iterdir())
    seen: dict[str, pathlib.Path] = {}
    for resource in resource_dirs:
        sys.path.append(str(resource.resolve()))
        for py in resource.rglob("*.py"):
            mod = ".".join(py.relative_to(resource).with_suffix("").parts)
            if mod in seen:
                raise ImportError(f"Duplicate module '{mod}' found:\n  {seen[mod]}\n  {py}")
            seen[mod] = py


def enable_fallback_compute():
    """Enable serverless compute if no compute is specified."""
    conf = WorkspaceClient().config
    if conf.serverless_compute_id or conf.cluster_id or os.environ.get("SPARK_REMOTE"):
        return

    url = "https://docs.databricks.com/dev-tools/databricks-connect/cluster-config"
    print("☁️ no compute specified, falling back to serverless compute", file=sys.stderr)
    print(f"  see {url} for manual configuration", file=sys.stdout)

    os.environ["DATABRICKS_SERVERLESS_COMPUTE_ID"] = "auto"


@contextmanager
def allow_stderr_output(config: pytest.Config):
    """Temporarily disable pytest output capture."""
    capman = config.pluginmanager.get_plugin("capturemanager")
    if capman:
        with capman.global_and_fixture_disabled():
            yield
    else:
        yield


def pytest_configure(config: pytest.Config):
    """Configure pytest session."""
    with allow_stderr_output(config):
        add_all_resources_to_sys_path()
        enable_fallback_compute()

        # Initialize Spark session eagerly, so it is available even when
        # SparkSession.builder.getOrCreate() is used. For DB Connect 15+,
        # we validate version compatibility with the remote cluster.
        if hasattr(DatabricksSession.builder, "validateSession"):
            DatabricksSession.builder.validateSession().getOrCreate()
        else:
            DatabricksSession.builder.getOrCreate()


@pytest.fixture(scope="session")
def spark() -> SparkSession:
    """Provide a SparkSession fixture for tests."""
    return DatabricksSession.builder.getOrCreate()
