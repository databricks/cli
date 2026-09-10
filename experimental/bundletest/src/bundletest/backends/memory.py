"""In-memory backend backed by stdlib sqlite3.

Runs today with zero infra. It is a *real* engine, not a mock: seeded rows are stored,
jobs run their registered logic against them, and assertions query the real output. It
is deliberately low-fidelity — sqlite's SQL dialect and type system differ from
Databricks SQL, which is why dialect/type assertions belong behind ``@cloud_only``.

A job's logic is supplied by a ``transforms.py`` next to the bundle that exposes a
``JOBS`` dict mapping job name -> ``callable(sql)``, where ``sql`` runs one statement.
"""

from __future__ import annotations

import importlib.util
import os
import re
import sqlite3
import time
from typing import Any, Callable

from bundletest.backend import RunResult

# Match a dotted table reference only where a table is expected (after FROM/JOIN/INTO/
# UPDATE/TABLE), so numeric literals like ``10.00`` are never rewritten.
_TABLE_REF = re.compile(r"(?i)\b(from|join|into|update|table)\s+([A-Za-z_]\w*(?:\.\w+)+)")


def _flat(fqn: str) -> str:
    return fqn.replace(".", "__")


def _sqlite_type(values: list[Any]) -> str:
    for v in values:
        if v is None:
            continue
        if isinstance(v, bool):
            return "INTEGER"
        if isinstance(v, int):
            return "INTEGER"
        if isinstance(v, float):
            return "REAL"
        return "TEXT"
    return "TEXT"


class InMemoryBackend:
    """sqlite-backed implementation of the ``Backend`` protocol."""

    def __init__(self) -> None:
        self._conn = sqlite3.connect(":memory:")
        self._known: dict[str, str] = {}  # fqn -> flat sqlite name
        self._jobs: dict[str, Callable[[Callable[[str], list[tuple]]], None]] = {}
        self._uploads: list[tuple[str, str]] = []

    # --- lifecycle ---
    def deploy(self, bundle_path: str) -> None:
        self._jobs = {}
        transforms = os.path.join(bundle_path, "transforms.py")
        if os.path.exists(transforms):
            spec = importlib.util.spec_from_file_location(
                "_bundletest_transforms", transforms
            )
            assert spec and spec.loader  # spec_from_file_location on a real path
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)
            self._jobs = dict(getattr(module, "JOBS", {}))

    def teardown(self) -> None:
        self._conn.close()

    # --- scaffolding ---
    def seed_table(self, fqn: str, rows: list[dict[str, Any]]) -> None:
        if not rows:
            raise ValueError(f"cannot seed {fqn!r} with no rows")
        flat = _flat(fqn)
        self._known[fqn] = flat
        columns = list(rows[0].keys())
        coldefs = ", ".join(
            f"{c} {_sqlite_type([r.get(c) for r in rows])}" for c in columns
        )
        cur = self._conn.cursor()
        cur.execute(f"DROP TABLE IF EXISTS {flat}")
        cur.execute(f"CREATE TABLE {flat} ({coldefs})")
        placeholders = ", ".join("?" for _ in columns)
        cur.executemany(
            f"INSERT INTO {flat} ({', '.join(columns)}) VALUES ({placeholders})",
            [tuple(r.get(c) for c in columns) for r in rows],
        )
        self._conn.commit()

    # --- execution ---
    def run_job(self, name: str, params: dict[str, Any] | None = None) -> RunResult:
        fn = self._jobs.get(name)
        if fn is None:
            raise KeyError(
                f"no job named {name!r} registered in the bundle's transforms.py "
                f"(known: {sorted(self._jobs)})"
            )
        start = time.perf_counter()
        try:
            fn(self.execute_sql)
            state, error = "SUCCESS", ""
        except Exception as e:  # noqa: BLE001 - surface as a failed run, not a crash
            state, error = "FAILED", f"{type(e).__name__}: {e}"
        return RunResult(
            result_state=state,
            duration_seconds=time.perf_counter() - start,
            error=error,
        )

    # --- data plane ---
    def execute_sql(self, query: str) -> list[tuple]:
        cur = self._conn.cursor()
        cur.execute(self._rewrite(query))
        rows = cur.fetchall()
        self._conn.commit()
        return rows

    def table_schema(self, fqn: str) -> dict[str, str]:
        flat = self._known.get(fqn, _flat(fqn))
        cur = self._conn.cursor()
        cur.execute(f"PRAGMA table_info({flat})")
        # PRAGMA columns: (cid, name, type, notnull, dflt_value, pk)
        return {name: coltype for _, name, coltype, *_ in cur.fetchall()}

    # --- control plane ---
    def get_resource(self, kind: str, name: str) -> dict[str, Any]:
        raise NotImplementedError(
            "the in-memory backend has no deployed resource config; "
            "read-back assertions are cloud-only"
        )

    def put_file(self, dst: str, src: str) -> None:
        self._uploads.append((dst, src))

    # --- internals ---
    def _rewrite(self, query: str) -> str:
        def repl(m: re.Match) -> str:
            keyword, fqn = m.group(1), m.group(2)
            flat = _flat(fqn)
            self._known[fqn] = flat
            return f"{keyword} {flat}"

        return _TABLE_REF.sub(repl, query)
