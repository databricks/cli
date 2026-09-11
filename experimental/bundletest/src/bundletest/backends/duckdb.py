"""Local backend backed by DuckDB, running the bundle's *real* deployed SQL.

Design invariants (from a design review of the earlier sqlite/callable prototype):

1. **Same source of truth.** A job runs the actual ``.sql`` artifact the bundle deploys,
   never a hand-written reimplementation. That is what makes "wrong table name" and
   broken-wiring bugs catchable at all.

2. **Bind names at the environment level, never rewrite the query body.** Seeded tables
   and target namespaces are created under their real ``catalog.schema.table`` names
   (via ATTACH / CREATE SCHEMA) so the unmodified SQL text resolves against them. The
   scan below only *discovers* namespaces to create; it never edits an executed statement.

3. **Fail loud, three ways.** A missing table/column is a real bug -> the run fails (red).
   A Databricks-only function DuckDB lacks, a notebook/Python task, or a reserved catalog
   name cannot be judged locally -> ``LocalUnsupported`` -> the test skips with a reason.
   DuckDB is stricter than sqlite (real typing, no silent coercion), so it does not
   quietly turn a real error into a green.
"""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import tempfile
import time
from pathlib import Path
from typing import Any

import duckdb

from bundletest.backend import LocalUnsupported, RunResult

# DuckDB reader function per file extension, for reading files uploaded to a volume.
_FILE_READERS = {".csv": "read_csv", ".json": "read_json", ".parquet": "read_parquet"}

# DuckDB cannot ATTACH a catalog under these names, so a job referencing e.g. `main.*`
# can't bind at the environment level. Per invariant 2 we route it to cloud, never rewrite.
_RESERVED_CATALOGS = {"main", "temp", "system"}

# DuckDB raises a CatalogException for BOTH a missing table and a missing function. Only
# the missing-function case is a dialect gap (skip); a missing table/column is a real bug
# (red). DuckDB has no distinct exception type, so we key off its message shape.
_MISSING_FUNCTION = re.compile(r"Function with name .* does not exist", re.IGNORECASE)

# A qualified name: schema.table (2 parts) or catalog.schema.table (3 parts). Anchored to
# an identifier start so numeric literals like `10.00` never match.
_QUALIFIED = re.compile(r"\b([A-Za-z_]\w*)\.([A-Za-z_]\w*)(?:\.([A-Za-z_]\w*))?")


def _duckdb_type(values: list[Any]) -> str:
    for v in values:
        if v is None:
            continue
        if isinstance(v, bool):
            return "BOOLEAN"
        if isinstance(v, int):
            return "INTEGER"
        if isinstance(v, float):
            return "DOUBLE"
        return "VARCHAR"
    return "VARCHAR"


def _first_line(err: Exception) -> str:
    return str(err).splitlines()[0] if str(err) else type(err).__name__


# Bundle config is resolved offline by the CLI's own engine (cmd/offline-resolve), which
# reuses the real load/target/variable mutators — no reimplementation here. References only a
# workspace can resolve come back two ways: ${workspace.*}/${resources.*} stay literal ($${...}
# is the escape, so a ${ not preceded by $ is a real reference), and a lookup or unset variable
# comes back as a sentinel marker. Both are rejected loudly at the use site.
_VAR_REF = re.compile(r"(?<!\$)\$\{[^}]+\}")
# Must match SentinelFormat in cmd/offline-resolve/main.go.
_SENTINEL = re.compile(r"__bundletest_unresolved__(lookup|unset)__([A-Za-z_][\w-]*)__")

# Package path of the offline resolver, relative to the repo root (the module containing go.mod).
_OFFLINE_RESOLVE_PKG = "./experimental/bundletest/cmd/offline-resolve"


def _repo_root() -> Path:
    for p in Path(__file__).resolve().parents:
        if (p / "go.mod").exists():
            return p
    raise RuntimeError("could not locate the repo root (go.mod) for the offline-resolve helper")


def _resolve_config(bundle_path: str) -> dict[str, Any]:
    """Load and resolve the bundle at ``bundle_path`` offline via the Go helper, returning the
    resolved config as a dict."""
    result = subprocess.run(
        ["go", "run", _OFFLINE_RESOLVE_PKG, bundle_path],
        cwd=_repo_root(),
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(f"offline bundle resolution failed:\n{result.stderr.strip()}")
    return json.loads(result.stdout)


def _unresolved_in_str(s: str) -> str | None:
    """A message describing why ``s`` can't be resolved locally (a sentinel-marked variable or a
    residual ${...} reference), or None if it is fully resolved."""
    m = _SENTINEL.search(s)
    if m:
        reason, name = m.group(1), m.group(2)
        if reason == "lookup":
            return f"variable {name!r} is a lookup variable that needs a workspace to resolve"
        return f"variable {name!r} has no value offline — set BUNDLE_VAR_{name} or run on the cloud backend"
    m = _VAR_REF.search(s)
    if m:
        return f"{m.group(0)} needs a workspace to resolve"
    return None


def _unresolved(node: Any) -> str | None:
    """First offline-unresolvable reference anywhere in the subtree, as a message, or None."""
    if isinstance(node, dict):
        return next((r for r in map(_unresolved, node.values()) if r), None)
    if isinstance(node, list):
        return next((r for r in map(_unresolved, node) if r), None)
    if isinstance(node, str):
        return _unresolved_in_str(node)
    return None


class _Task:
    """One task of a job, as declared in databricks.yml."""

    def __init__(self, key: str, kind: str, sql_file: str | None):
        self.key = key
        self.kind = kind  # "sql" | "notebook" | "python" | "pipeline" | ...
        self.sql_file = sql_file


class DuckDBBackend:
    """Local ``Backend`` implementation. Runs sql_task artifacts against DuckDB."""

    def __init__(self) -> None:
        self._con = duckdb.connect()
        self._attached: set[str] = set()
        self._bundle_path = ""
        self._config: dict[str, Any] = {}
        self._jobs: dict[str, list[_Task]] = {}
        self._volume_root = tempfile.mkdtemp(prefix="bundletest-vol-")

    # --- lifecycle ---
    def deploy(self, bundle_path: str) -> None:
        self._bundle_path = bundle_path
        self._config = _resolve_config(bundle_path)
        self._jobs = self._extract_jobs(self._config)

    def teardown(self) -> None:
        self._con.close()
        shutil.rmtree(self._volume_root, ignore_errors=True)

    # --- scaffolding ---
    def seed_table(self, fqn: str, rows: list[dict[str, Any]]) -> None:
        if not rows:
            raise ValueError(f"cannot seed {fqn!r} with no rows")
        self._prepare_namespaces(fqn)
        columns = list(rows[0].keys())
        coldefs = ", ".join(f'"{c}" {_duckdb_type([r.get(c) for r in rows])}' for c in columns)
        self._con.execute(f"CREATE OR REPLACE TABLE {fqn} ({coldefs})")
        placeholders = ", ".join("?" for _ in columns)
        self._con.executemany(
            f"INSERT INTO {fqn} ({', '.join(columns)}) VALUES ({placeholders})",
            [tuple(r.get(c) for c in columns) for r in rows],
        )

    # --- execution ---
    def run_job(self, name: str, params: dict[str, Any] | None = None) -> RunResult:
        tasks = self._jobs.get(name)
        if tasks is None:
            raise KeyError(f"no job named {name!r} in the bundle (known: {sorted(self._jobs)})")
        start = time.perf_counter()
        current_task: _Task | None = None
        try:
            for task in tasks:
                current_task = task
                if task.kind != "sql":
                    raise LocalUnsupported(
                        f"job {name!r} task {task.key!r} is a {task.kind} task; the "
                        f"local backend runs sql_task only — run it on the cloud backend"
                    )
                if task.sql_file:
                    reason = _unresolved_in_str(task.sql_file)
                    if reason is not None:
                        raise LocalUnsupported(
                            f"job {name!r} task {task.key!r} sql path can't be resolved locally: {reason}"
                        )
                sql = Path(self._bundle_path, task.sql_file).read_text()
                self._prepare_namespaces(sql)
                for statement in _split_statements(sql):
                    self._con.execute(statement)
            return RunResult(
                "SUCCESS",
                time.perf_counter() - start,
                backend="local",
                resource_name=name,
            )
        except LocalUnsupported:
            raise
        except duckdb.Error as e:
            if _MISSING_FUNCTION.search(str(e)):
                raise LocalUnsupported(f"job {name!r} uses SQL not available locally: {_first_line(e)}") from e
            return RunResult(
                "FAILED",
                time.perf_counter() - start,
                error=_first_line(e),
                backend="local",
                resource_name=name,
                task_key=current_task.key if current_task else "",
                source_path=current_task.sql_file if current_task and current_task.sql_file else "",
            )

    # --- data plane ---
    def execute_sql(self, query: str) -> list[tuple]:
        try:
            return self._con.execute(query).fetchall()
        except duckdb.Error as e:
            if _MISSING_FUNCTION.search(str(e)):
                raise LocalUnsupported(f"assertion uses SQL not available locally: {_first_line(e)}") from e
            raise

    def table_schema(self, fqn: str) -> dict[str, str]:
        rows = self._con.execute(f"DESCRIBE {fqn}").fetchall()
        # DESCRIBE columns: (column_name, column_type, null, key, default, extra)
        return {name: coltype for name, coltype, *_ in rows}

    # --- control plane ---
    def get_resource(self, kind: str, name: str) -> dict[str, Any]:
        cfg = self._config.get("resources", {}).get(kind, {})[name]
        reason = _unresolved(cfg)
        if reason is not None:
            raise LocalUnsupported(
                f"resource {kind}.{name} can't be introspected locally: {reason}; "
                f"use the cloud backend"
            )
        return cfg

    def put_file(self, dst: str, src: str) -> None:
        if not os.path.exists(src):
            raise FileNotFoundError(f"upload source not found: {src}")
        target = Path(self._volume_root) / dst.lstrip("/")
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy(src, target)

    def read_volume_file(self, volume: str, filename: str) -> list[dict[str, Any]]:
        path = Path(self._volume_root) / "Volumes" / volume / filename
        if not path.exists():
            raise FileNotFoundError(f"no file {filename!r} uploaded to volume {volume!r}")
        reader = _FILE_READERS.get(path.suffix)
        if reader is None:
            raise LocalUnsupported(f"cannot read {path.suffix!r} files locally")
        cur = self._con.execute(f"SELECT * FROM {reader}('{path.as_posix()}')")
        columns = [d[0] for d in cur.description]
        return [dict(zip(columns, row, strict=True)) for row in cur.fetchall()]

    # --- internals ---
    @staticmethod
    def _extract_jobs(config: dict[str, Any]) -> dict[str, list[_Task]]:
        jobs: dict[str, list[_Task]] = {}
        for name, job in config.get("resources", {}).get("jobs", {}).items():
            tasks = []
            for t in job.get("tasks", [{}]) or [{}]:
                if "sql_task" in t:
                    tasks.append(_Task(t.get("task_key", ""), "sql", t["sql_task"]["file"]["path"]))
                elif "notebook_task" in t:
                    tasks.append(_Task(t.get("task_key", ""), "notebook", None))
                elif any(k.endswith("_task") for k in t):
                    kind = next(k[:-5] for k in t if k.endswith("_task"))
                    tasks.append(_Task(t.get("task_key", ""), kind, None))
            jobs[name] = tasks
        return jobs

    def _prepare_namespaces(self, sql: str) -> None:
        """Create the catalogs/schemas referenced by ``sql`` so the unmodified statement
        resolves. Over-creation from a misparsed column ref is harmless; the query text is
        never touched."""
        for m in _QUALIFIED.finditer(sql):
            catalog, schema, table = m.group(1), m.group(2), m.group(3)
            if table:  # catalog.schema.table
                self._ensure_catalog(catalog)
                self._ensure_schema(catalog, schema)
            else:  # schema.table in the default catalog
                self._ensure_schema(None, catalog)

    def _ensure_catalog(self, catalog: str) -> None:
        if catalog in self._attached or catalog == "memory":
            return
        if catalog in _RESERVED_CATALOGS:
            raise LocalUnsupported(
                f"catalog {catalog!r} is a DuckDB-reserved name and cannot be hosted "
                f"locally — run this on the cloud backend"
            )
        self._con.execute(f"ATTACH ':memory:' AS {catalog}")
        self._attached.add(catalog)

    def _ensure_schema(self, catalog: str | None, schema: str) -> None:
        target = f"{catalog}.{schema}" if catalog else schema
        try:
            self._con.execute(f"CREATE SCHEMA IF NOT EXISTS {target}")
        except duckdb.Error:
            pass  # best-effort env prep (e.g. a system schema); real errors surface on run


def _split_statements(sql: str) -> list[str]:
    return [s.strip() for s in sql.split(";") if s.strip()]
