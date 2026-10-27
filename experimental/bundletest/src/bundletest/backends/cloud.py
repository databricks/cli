"""Cloud backend: runs the same tests against a real Databricks workspace.

The fidelity tier behind the ``Backend`` seam. Where the local DuckDB backend simulates,
this deploys the bundle for real and drives the workspace, so the ``cloud_only`` assertions
(Databricks type names, SLA timing, notebook/Python jobs, permissions) actually run instead
of skipping.

Design invariants — the cloud analogues of the DuckDB backend's:

1. **Same source of truth.** ``run_job`` runs the *deployed* job (``jobs.run_now`` on the
   real cluster/warehouse), not a reimplementation — the whole point of the fidelity tier.

2. **Real names, no rewriting.** Seeded tables and job targets use their real
   ``catalog.schema.table`` names. Missing schemas are created (``CREATE SCHEMA IF NOT
   EXISTS``) — the UC equivalent of the local backend's ATTACH; there is no reserved-catalog
   restriction here (``main`` is a fine UC catalog), so this backend never raises
   ``LocalUnsupported``.

3. **Fail loud.** A failed statement raises; a failed job run comes back as a FAILED
   ``RunResult``. Nothing is silently coerced to green.

Config comes from the environment: ``BUNDLETEST_PROFILE`` (CLI/SDK auth profile),
``BUNDLETEST_WAREHOUSE_ID`` (the SQL warehouse for seeding/queries; required for any SQL),
``BUNDLETEST_TARGET`` (bundle target). Bundle variables are supplied the normal DABs way
(``BUNDLE_VAR_<name>`` env vars or the target), so ``deploy`` stays bundle-agnostic.
"""

from __future__ import annotations

import json
import logging
import os
import re
import shutil
import subprocess
import tempfile
import time
from pathlib import Path
from typing import Any

import duckdb
import yaml

from bundletest.backend import RunResult

log = logging.getLogger(__name__)

# DuckDB reader per file extension, reused to parse a volume file downloaded from the
# workspace (duckdb is already a base dependency, so no extra parser is pulled in).
_FILE_READERS = {".csv": "read_csv", ".json": "read_json", ".parquet": "read_parquet"}

# A three-part UC name catalog.schema.table. Anchored to an identifier start so numeric
# literals never match. Only used to discover which schemas a job's SQL writes to, so
# over-matching (e.g. a column ref) is harmless — CREATE SCHEMA IF NOT EXISTS is idempotent.
_QUALIFIED = re.compile(r"\b([A-Za-z_]\w*)\.([A-Za-z_]\w*)\.[A-Za-z_]\w*")

# Statement Execution API column type_name -> value caster. Every value arrives as a string
# in data_array; the assertion layer needs native types (row_count() == 2, not "2").
_INT_TYPES = {"BYTE", "SHORT", "INT", "LONG"}
_FLOAT_TYPES = {"FLOAT", "DOUBLE", "DECIMAL"}


def _sql_type(values: list[Any]) -> str:
    for v in values:
        if v is None:
            continue
        if isinstance(v, bool):
            return "BOOLEAN"
        if isinstance(v, int):
            return "BIGINT"
        if isinstance(v, float):
            return "DOUBLE"
        return "STRING"
    return "STRING"


def _sql_literal(v: Any) -> str:
    if v is None:
        return "NULL"
    if isinstance(v, bool):
        return "true" if v else "false"
    if isinstance(v, (int, float)):
        return repr(v)
    # Databricks/Spark SQL escapes string literals with a backslash, and (unlike ANSI SQL)
    # a doubled quote '' is NOT an escaped quote — it drops the quote. So escape the
    # backslash first, then the single quote, both with a backslash.
    return "'" + str(v).replace("\\", "\\\\").replace("'", "\\'") + "'"


def _cast_value(raw: str | None, type_name: str) -> Any:
    """Cast one string cell from data_array to the native type its column declares."""
    if raw is None:
        return None
    if type_name == "BOOLEAN":
        return raw.lower() == "true"
    if type_name in _INT_TYPES:
        return int(raw)
    if type_name in _FLOAT_TYPES:
        return float(raw)
    return raw


def _schemas_in(sql: str) -> list[str]:
    """The distinct ``catalog.schema`` prefixes of the three-part names in ``sql``."""
    out: list[str] = []
    for m in _QUALIFIED.finditer(sql):
        ns = f"{m.group(1)}.{m.group(2)}"
        if ns not in out:
            out.append(ns)
    return out


class CloudBackend:
    """Cloud ``Backend`` implementation. Deploys the bundle and drives a real workspace."""

    def __init__(
        self,
        profile: str | None = None,
        warehouse_id: str | None = None,
        target: str | None = None,
    ) -> None:
        self._profile = profile or os.environ.get("BUNDLETEST_PROFILE")
        self._warehouse_id = warehouse_id or os.environ.get("BUNDLETEST_WAREHOUSE_ID")
        self._target = target or os.environ.get("BUNDLETEST_TARGET")
        self._client: Any = None  # lazy WorkspaceClient; constructing it resolves auth
        self._bundle_path = ""
        self._config: dict[str, Any] = {}
        self._summary: dict[str, Any] | None = None
        self._seeded: set[str] = set()

    # --- lifecycle ---
    def deploy(self, bundle_path: str) -> None:
        self._bundle_path = bundle_path
        self._summary = None
        path = Path(bundle_path, "databricks.yml")
        self._config = yaml.safe_load(path.read_text()) if path.exists() else {}
        self._bundle("deploy")

    def teardown(self) -> None:
        # Drop what we seeded, then destroy the bundle. Best-effort (must not raise), but a
        # failed cleanup leaks real resources — log it rather than swallow it silently.
        for fqn in self._seeded:
            try:
                self.execute_sql(f"DROP TABLE IF EXISTS {fqn}")
            except Exception as e:
                log.warning("teardown: failed to drop seeded table %s (may be leaked): %s", fqn, e)
        try:
            self._bundle("destroy", "--auto-approve")
        except Exception as e:
            log.warning("teardown: `bundle destroy` failed (resources may be leaked): %s", e)

    # --- scaffolding ---
    def seed_table(self, fqn: str, rows: list[dict[str, Any]]) -> None:
        if not rows:
            raise ValueError(f"cannot seed {fqn!r} with no rows")
        self._ensure_schema(fqn)
        columns = list(rows[0].keys())
        # Backtick-quote column identifiers so a reserved word (order, end) or special char
        # works, matching the local backend's quoting.
        coldefs = ", ".join(f"`{c}` {_sql_type([r.get(c) for r in rows])}" for c in columns)
        self.execute_sql(f"CREATE OR REPLACE TABLE {fqn} ({coldefs})")
        values = ", ".join("(" + ", ".join(_sql_literal(r.get(c)) for c in columns) + ")" for r in rows)
        collist = ", ".join(f"`{c}`" for c in columns)
        self.execute_sql(f"INSERT INTO {fqn} ({collist}) VALUES {values}")
        self._seeded.add(fqn)

    # --- execution ---
    def run_job(self, name: str, params: dict[str, Any] | None = None) -> RunResult:
        from databricks.sdk.service.jobs import RunResultState

        job_id = int(self.get_resource("jobs", name)["id"])
        self._ensure_job_schemas(name)
        job_params = {k: str(v) for k, v in (params or {}).items()}
        run = self._ws().jobs.run_now(job_id, job_parameters=job_params or None).result()
        succeeded = run.state.result_state == RunResultState.SUCCESS
        return RunResult(
            "SUCCESS" if succeeded else "FAILED",
            (run.run_duration or 0) / 1000,
            str(run.run_id),
            error="" if succeeded else (run.state.state_message or ""),
            backend="cloud",
            resource_name=name,
        )

    # --- data plane ---
    def execute_sql(self, query: str) -> list[tuple]:
        from databricks.sdk.service.sql import Disposition, Format, StatementState

        warehouse_id = self._require_warehouse()
        se = self._ws().statement_execution
        resp = se.execute_statement(
            statement=query,
            warehouse_id=warehouse_id,
            wait_timeout="30s",
            disposition=Disposition.INLINE,
            format=Format.JSON_ARRAY,
        )
        while resp.status.state in (StatementState.PENDING, StatementState.RUNNING):
            time.sleep(1)
            resp = se.get_statement(resp.statement_id)
        if resp.status.state != StatementState.SUCCEEDED:
            err = resp.status.error
            detail = err.message if err else resp.status.state.value
            raise RuntimeError(f"statement failed: {detail}")

        # A successful DDL/DML statement (CREATE/INSERT/DROP, and the schema-prep here)
        # returns no result set, so there is nothing to read or cast.
        if resp.result is None:
            return []
        columns = (resp.manifest.schema.columns if resp.manifest and resp.manifest.schema else None) or []
        types = [c.type_name.value for c in columns]
        rows = list(resp.result.data_array or [])
        # Inline results past the first chunk are fetched by index.
        nxt = resp.result.next_chunk_index
        while nxt is not None:
            chunk = se.get_statement_result_chunk_n(resp.statement_id, nxt)
            rows += chunk.data_array or []
            nxt = chunk.next_chunk_index
        return [tuple(_cast_value(v, types[i]) for i, v in enumerate(row)) for row in rows]

    def table_schema(self, fqn: str) -> dict[str, str]:
        # DESCRIBE emits (col_name, data_type, comment); trailing partition/detail rows have a
        # blank or '#'-prefixed col_name. data_type is already the Databricks spelling
        # (e.g. "decimal(10,2)"), which is exactly what the cloud_only type test wants.
        schema: dict[str, str] = {}
        for name, dtype, *_ in self.execute_sql(f"DESCRIBE TABLE {fqn}"):
            if not name or name.startswith("#"):
                break
            schema[name] = dtype
        return schema

    # --- control plane ---
    def get_resource(self, kind: str, name: str) -> dict[str, Any]:
        # `...[kind][name]` raises KeyError for an undeclared resource, which exists() expects.
        # `bundle summary` gives the rendered config, and its config mutators inline
        # serialized_dashboard/serialized_space from a file_path at load time, so the
        # serialized form source_tables() needs is already here — no workspace read needed.
        return self._resources()[kind][name]

    def get_deployed(self, kind: str, name: str) -> dict[str, Any]:
        """Read the resource back from the workspace as the server stored it — server shape,
        with the values the server filled in or normalized. This differs from get_resource,
        which returns the *declared* config; use this to validate what deployment actually did.

        Cloud-only by nature (there is no server locally), so it lives only on this backend and
        tests that call it must be ``@cloud_only``. Returns the SDK object as a dict."""
        cfg = self.get_resource(kind, name)
        if kind == "jobs":
            return self._ws().jobs.get(int(cfg["id"])).as_dict()
        if kind == "dashboards":
            return self._ws().lakeview.get(cfg["id"]).as_dict()
        if kind == "volumes":
            fqn = f"{cfg['catalog_name']}.{cfg['schema_name']}.{cfg['name']}"
            return self._ws().volumes.read(fqn).as_dict()
        raise ValueError(f"get_deployed is not implemented for kind {kind!r} (have: jobs, dashboards, volumes)")

    def put_file(self, dst: str, src: str) -> None:
        if not os.path.exists(src):
            raise FileNotFoundError(f"upload source not found: {src}")
        with open(src, "rb") as f:
            self._ws().files.upload(self._volume_path(dst), f, overwrite=True)

    def read_volume_file(self, volume: str, filename: str) -> list[dict[str, Any]]:
        path = self._volume_path(f"/Volumes/{volume}/{filename}")
        suffix = Path(filename).suffix
        reader = _FILE_READERS.get(suffix)
        if reader is None:
            raise ValueError(f"cannot read {suffix!r} files")
        from databricks.sdk.errors import NotFound

        try:
            resp = self._ws().files.download(path)
        except NotFound as e:
            # Only a genuine missing file becomes FileNotFoundError (FileHandle.exists() keys on
            # it); auth/permission errors must surface, not masquerade as "not found".
            raise FileNotFoundError(f"no file {filename!r} in volume {volume!r}: {e}") from e
        # DuckDB re-opens the file by path, so write it into a temp dir and pass a forward-slash
        # path: a NamedTemporaryFile can't be reopened while open on Windows, and its backslash
        # path would break the SQL string literal.
        tmp_dir = tempfile.mkdtemp(prefix="bundletest-vol-")
        try:
            local = Path(tmp_dir) / f"data{suffix}"
            local.write_bytes(resp.contents.read())
            con = duckdb.connect()
            try:
                cur = con.execute(f"SELECT * FROM {reader}('{local.as_posix()}')")
                cols = [d[0] for d in cur.description]
                return [dict(zip(cols, row, strict=True)) for row in cur.fetchall()]
            finally:
                con.close()
        finally:
            shutil.rmtree(tmp_dir, ignore_errors=True)

    # --- internals ---
    def _ws(self) -> Any:
        if self._client is None:
            from databricks.sdk import WorkspaceClient

            self._client = WorkspaceClient(profile=self._profile)
        return self._client

    def _require_warehouse(self) -> str:
        if not self._warehouse_id:
            raise RuntimeError("cloud SQL needs a warehouse — set BUNDLETEST_WAREHOUSE_ID")
        return self._warehouse_id

    def _bundle(self, *args: str) -> str:
        cmd = ["databricks", "bundle", *args]
        if self._target:
            cmd += ["-t", self._target]
        env = dict(os.environ)
        if self._profile:
            env["DATABRICKS_CONFIG_PROFILE"] = self._profile
        out = subprocess.run(cmd, cwd=self._bundle_path, env=env, capture_output=True, text=True)
        if out.returncode != 0:
            raise RuntimeError(f"`{' '.join(cmd)}` failed: {out.stderr.strip()}")
        return out.stdout

    def _resources(self) -> dict[str, Any]:
        if self._summary is None:
            self._summary = json.loads(self._bundle("summary", "-o", "json"))
        return self._summary.get("resources", {})

    def _ensure_schema(self, fqn: str) -> None:
        parts = fqn.split(".")
        if len(parts) == 3:
            self.execute_sql(f"CREATE SCHEMA IF NOT EXISTS {parts[0]}.{parts[1]}")

    def _ensure_job_schemas(self, name: str) -> None:
        """Create the schemas a job's SQL writes to, so its unmodified statements resolve —
        the cloud analogue of the local backend's namespace preparation. Best-effort: a
        non-file sql_task (query/alert) is skipped, and a misparsed reference (a struct field
        read as catalog.schema) just fails to create — the real error, if any, surfaces on run."""
        for task in self._config.get("resources", {}).get("jobs", {}).get(name, {}).get("tasks", []):
            file = (task.get("sql_task") or {}).get("file")
            if not file:
                continue
            sql = Path(self._bundle_path, file["path"]).read_text()
            for ns in _schemas_in(sql):
                try:
                    self.execute_sql(f"CREATE SCHEMA IF NOT EXISTS {ns}")
                except RuntimeError as e:
                    # A misparsed reference (e.g. a struct field read as catalog.schema) fails
                    # here; ignore it and let a real error surface on run. Narrow to the
                    # statement failure execute_sql raises so auth/permission errors propagate.
                    log.debug("skipping schema prep for %s: %s", ns, e)

    def _volume_path(self, dst: str) -> str:
        """Resolve ``/Volumes/<resource_name>/<path...>`` to a real UC volume path.

        The first segment after ``/Volumes/`` is the volume *resource* name — the same
        contract the local backend uses on both put_file and read_volume_file — which we
        expand to ``/Volumes/<catalog>/<schema>/<volume>/<path...>`` via the deployed config."""
        _, resource, *rest = dst.strip("/").split("/")
        vol = self.get_resource("volumes", resource)
        return "/".join(["/Volumes", vol["catalog_name"], vol["schema_name"], vol["name"], *rest])
