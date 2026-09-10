"""The backend seam.

Everything that differs between running against a local simulator and a real cloud
workspace lives behind the ``Backend`` protocol. The test API above it (``BundleEnv``,
``TableHandle``, ``ColumnHandle``) is written once against these methods and never
branches on which backend it holds.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Protocol, runtime_checkable


class LocalUnsupported(Exception):
    """Raised when the local backend cannot faithfully run something.

    Not a failure — a signal that the check can only be answered on the cloud backend
    (a notebook/Python task, a Databricks-only SQL function, a reserved catalog name).
    The pytest plugin turns it into a *skip with a reason*, never a red test, so the
    local tier neither false-greens nor false-reds on things it can't judge.
    """


@dataclass
class RunResult:
    """Outcome of running a single bundle resource (e.g. a job)."""

    result_state: str  # "SUCCESS" | "FAILED"
    duration_seconds: float = 0.0  # wall-clock; only meaningful on the cloud backend
    run_id: str = ""
    error: str = ""  # failure detail when result_state == "FAILED"

    @property
    def succeeded(self) -> bool:
        return self.result_state == "SUCCESS"


@runtime_checkable
class Backend(Protocol):
    """The only abstraction with more than one implementation.

    Assertions are derived above this seam from ``execute_sql`` and ``table_schema``;
    ``seed_table`` stands in for an upstream resource so a single component can be run
    in isolation.
    """

    def deploy(self, bundle_path: str) -> None:
        """Make the bundle's resources available to run."""

    def teardown(self) -> None:
        """Release everything created for this environment."""

    def seed_table(self, fqn: str, rows: list[dict[str, Any]]) -> None:
        """Populate a table with fixture rows, standing in for an upstream resource."""

    def run_job(self, name: str, params: dict[str, Any] | None = None) -> RunResult:
        """Run one resource by name (not the whole DAG)."""

    def execute_sql(self, query: str) -> list[tuple]:
        """Run a query and return its rows."""

    def table_schema(self, fqn: str) -> dict[str, str]:
        """Return column name -> type. Its own primitive because DESCRIBE syntax and
        type names differ across engines."""

    def get_resource(self, kind: str, name: str) -> dict[str, Any]:
        """Read back a deployed resource's config."""

    def put_file(self, dst: str, src: str) -> None:
        """Upload a local file to a volume path."""

    def read_volume_file(self, volume: str, filename: str) -> list[dict[str, Any]]:
        """Read a file previously uploaded to a volume, as a list of row dicts."""
