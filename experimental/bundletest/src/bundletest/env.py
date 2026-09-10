"""The test-facing environment: BundleEnv, resource handles, and the fixture helper."""

from __future__ import annotations

import json
import os
import re
from contextlib import contextmanager
from typing import TYPE_CHECKING, Any, Iterator

from bundletest.backend import Backend, LocalUnsupported, RunResult
from bundletest.table import FileHandle, TableHandle

if TYPE_CHECKING:
    pass

DEFAULT_BACKEND = "local"


class ResourceHandle:
    """A single declared bundle resource, keyed by (kind, name).

    Reads the resource's config through the backend seam (``get_resource``), so every handle
    works unchanged on the local and cloud backends. ``kind`` is the plural key under
    ``resources:`` in databricks.yml (e.g. ``jobs``, ``pipelines``, ``quality_monitors``).
    Subclasses add resource-specific accessors on top of this base.
    """

    def __init__(self, backend: Backend, kind: str, name: str):
        self._backend = backend
        self.kind = kind
        self.name = name

    @property
    def config(self) -> dict[str, Any]:
        return self._backend.get_resource(self.kind, self.name)

    def exists(self) -> bool:
        # get_resource does `...[name]`, which raises KeyError for an undeclared resource.
        try:
            self._backend.get_resource(self.kind, self.name)
            return True
        except KeyError:
            return False

    def permissions(self) -> list:
        return self.config.get("permissions", [])

    def grants(self) -> list:
        return self.config.get("grants", [])


class JobHandle(ResourceHandle):
    """A single job resource. Runs one job by name (not the whole DAG)."""

    def __init__(self, backend: Backend, name: str):
        super().__init__(backend, "jobs", name)
        self._last: RunResult | None = None

    def run(self, params: dict[str, Any] | None = None) -> RunResult:
        self._last = self._backend.run_job(self.name, params)
        return self._last

    def last_run(self) -> RunResult:
        if self._last is None:
            raise RuntimeError(f"job {self.name!r} has not been run yet")
        return self._last


class _Jobs:
    """`env.jobs["name"]` accessor that reuses handles so `last_run()` persists."""

    def __init__(self, backend: Backend):
        self._backend = backend
        self._cache: dict[str, JobHandle] = {}

    def __getitem__(self, name: str) -> JobHandle:
        return self._cache.setdefault(name, JobHandle(self._backend, name))


class VolumeHandle(ResourceHandle):
    """A single volume resource."""

    def __init__(self, backend: Backend, name: str):
        super().__init__(backend, "volumes", name)

    def upload(self, src: str, dst: str | None = None) -> None:
        self._backend.put_file(dst or f"/Volumes/{self.name}/{os.path.basename(src)}", src)

    def file(self, filename: str) -> FileHandle:
        return FileHandle(self._backend, self.name, filename)


class PipelineHandle(ResourceHandle):
    """A pipeline (Lakeflow) resource — wiring only; running it is cloud-only."""

    def __init__(self, backend: Backend, name: str):
        super().__init__(backend, "pipelines", name)

    @property
    def catalog(self) -> str | None:
        return self.config.get("catalog")

    @property
    def schema(self) -> str | None:
        # New pipelines set `schema`; `target` was the legacy (DLT) field for the same idea.
        return self.config.get("schema")

    def libraries(self) -> list[str]:
        """Notebook/file paths the pipeline runs, in declaration order."""
        paths = []
        for lib in self.config.get("libraries", []):
            if "notebook" in lib:
                paths.append(lib["notebook"]["path"])
            elif "file" in lib:
                paths.append(lib["file"]["path"])
        return paths


class DashboardHandle(ResourceHandle):
    """A Lakeview dashboard resource."""

    def __init__(self, backend: Backend, name: str):
        super().__init__(backend, "dashboards", name)

    def source_tables(self) -> list[str]:
        return _serialized_source_tables(self.name, self.config.get("serialized_dashboard"))


class GenieSpaceHandle(ResourceHandle):
    """A Genie Space resource."""

    def __init__(self, backend: Backend, name: str):
        super().__init__(backend, "genie_spaces", name)

    def source_tables(self) -> list[str]:
        return _serialized_source_tables(self.name, self.config.get("serialized_space"))


class QualityMonitorHandle(ResourceHandle):
    """A quality monitor resource. Keyed on the table it monitors."""

    def __init__(self, backend: Backend, name: str):
        super().__init__(backend, "quality_monitors", name)

    def monitored_table(self) -> str:
        return self.config["table_name"]


class VectorSearchIndexHandle(ResourceHandle):
    """A vector search index resource."""

    def __init__(self, backend: Backend, name: str):
        super().__init__(backend, "vector_search_indexes", name)

    @property
    def endpoint_name(self) -> str | None:
        return self.config.get("endpoint_name")

    def source_table(self) -> str | None:
        return self.config.get("delta_sync_index_spec", {}).get("source_table")


class ModelServingEndpointHandle(ResourceHandle):
    """A model serving endpoint resource."""

    def __init__(self, backend: Backend, name: str):
        super().__init__(backend, "model_serving_endpoints", name)

    def served_models(self) -> list[str]:
        """Model names served by the endpoint."""
        cfg = self.config.get("config", {})
        # `served_entities` is the current shape; `served_models` is the deprecated one.
        entities = cfg.get("served_entities") or cfg.get("served_models", [])
        return [e.get("entity_name") or e.get("model_name") for e in entities]


class AppHandle(ResourceHandle):
    """A Databricks App resource."""

    def __init__(self, backend: Backend, name: str):
        super().__init__(backend, "apps", name)

    def command(self) -> list[str]:
        return self.config.get("config", {}).get("command", [])

    @property
    def source_code_path(self) -> str | None:
        return self.config.get("source_code_path")


# Tables a query reads: the qualified (dotted) identifier right after FROM / JOIN. Matching
# only dotted names skips CTE names and aliases (which are unqualified); an outer backtick
# pair is unwrapped. Good enough for wiring, not a SQL parser — it won't unwrap per-segment
# backticks (`a`.`b`) and would match a name inside a `-- FROM ...` comment.
_FROM_JOIN = re.compile(r"\b(?:FROM|JOIN)\s+`?([A-Za-z_]\w*(?:\.\w+)+)`?", re.IGNORECASE)


def _serialized_source_tables(name: str, serialized: Any) -> list[str]:
    """Qualified source tables read by a dashboard/genie-space's dataset queries.

    ``serialized`` is the inline definition: a dict when inlined as YAML, or a JSON string.
    A file_path-only resource carries no inline queries in databricks.yml, so it can't be
    introspected locally — that's a loud skip, not a failure."""
    if serialized is None:
        raise LocalUnsupported(f"{name!r} defined by file_path — no inline queries to parse locally")
    spec = serialized if isinstance(serialized, dict) else json.loads(serialized)
    tables: list[str] = []
    for dataset in spec.get("datasets", []):
        # Lakeview stores a query as queryLines (current) or a single query string (older).
        query = "".join(dataset.get("queryLines", [])) or dataset.get("query", "")
        for m in _FROM_JOIN.finditer(query):
            if m.group(1) not in tables:
                tables.append(m.group(1))
    return tables


class BundleEnv:
    """A deployed bundle under test, backed by a single ``Backend``."""

    def __init__(self, bundle_path: str, backend: Backend):
        self.bundle_path = bundle_path
        self.backend = backend
        self.jobs = _Jobs(backend)

    # --- lifecycle ---
    def deploy(self) -> None:
        self.backend.deploy(self.bundle_path)

    def teardown(self) -> None:
        self.backend.teardown()

    # --- scaffolding: stand in for an upstream resource ---
    def seed(self, table: str, rows: list[dict[str, Any]]) -> None:
        self.backend.seed_table(table, rows)

    # --- resource handles ---
    def resource(self, kind: str, name: str) -> ResourceHandle:
        """Generic handle for any resource kind (the plural `resources:` key)."""
        return ResourceHandle(self.backend, kind, name)

    def table(self, fqn: str) -> TableHandle:
        return TableHandle(self.backend, fqn)

    def volume(self, name: str) -> VolumeHandle:
        return VolumeHandle(self.backend, name)

    def pipeline(self, name: str) -> PipelineHandle:
        return PipelineHandle(self.backend, name)

    def dashboard(self, name: str) -> DashboardHandle:
        return DashboardHandle(self.backend, name)

    def genie_space(self, name: str) -> GenieSpaceHandle:
        return GenieSpaceHandle(self.backend, name)

    def quality_monitor(self, name: str) -> QualityMonitorHandle:
        return QualityMonitorHandle(self.backend, name)

    def vector_search_index(self, name: str) -> VectorSearchIndexHandle:
        return VectorSearchIndexHandle(self.backend, name)

    def model_serving_endpoint(self, name: str) -> ModelServingEndpointHandle:
        return ModelServingEndpointHandle(self.backend, name)

    def app(self, name: str) -> AppHandle:
        return AppHandle(self.backend, name)

    def run_job(self, name: str, params: dict[str, Any] | None = None) -> RunResult:
        return self.jobs[name].run(params)


def make_backend(kind: str, **kwargs: Any) -> Backend:
    """Construct a backend by name. Backends are imported lazily so selecting one
    never pulls in the others' dependencies."""
    if kind == "local":
        from bundletest.backends.duckdb import DuckDBBackend

        return DuckDBBackend(**kwargs)
    if kind == "cloud":
        from bundletest.backends.cloud import CloudBackend

        return CloudBackend(**kwargs)
    raise ValueError(f"unknown backend {kind!r} (expected 'local' or 'cloud')")


def current_backend_kind() -> str:
    """Backend selected for this run. Lets the same test run against either tier
    without editing test code."""
    return os.environ.get("BUNDLETEST_BACKEND", DEFAULT_BACKEND)


@contextmanager
def bundle_env(
    bundle_path: str,
    backend: str | None = None,
    deploy: bool = True,
    **kwargs: Any,
) -> Iterator[BundleEnv]:
    """Build a backend, deploy the bundle, yield the env, and tear it down."""
    be = make_backend(backend or current_backend_kind(), **kwargs)
    env = BundleEnv(bundle_path, be)
    if deploy:
        env.deploy()
    try:
        yield env
    finally:
        env.teardown()
