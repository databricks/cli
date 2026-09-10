"""Cloud backend unit tests that need no workspace.

The workspace round-trips (deploy, run, statement submit) can only be exercised against a
real cloud, but the logic most likely to be wrong is pure and testable here: casting the
Statement Execution API's all-string results to native types (the base-compat trap),
literal/type rendering for seeding, namespace discovery, and volume-path resolution.
"""

from types import SimpleNamespace

import pytest
from bundletest.backends.cloud import (
    CloudBackend,
    _cast_value,
    _schemas_in,
    _sql_literal,
    _sql_type,
)
from databricks.sdk.service.sql import ColumnInfoTypeName as T
from databricks.sdk.service.sql import StatementState


def test_sql_type_inference():
    assert _sql_type([True, False]) == "BOOLEAN"
    assert _sql_type([1, 2]) == "BIGINT"
    assert _sql_type([1.5]) == "DOUBLE"
    assert _sql_type(["x"]) == "STRING"
    assert _sql_type([None, None]) == "STRING"  # all-null -> default
    assert _sql_type([None, 3]) == "BIGINT"  # first non-null wins


def test_sql_literal_escaping():
    assert _sql_literal(None) == "NULL"
    assert _sql_literal(True) == "true"
    assert _sql_literal(5) == "5"
    assert _sql_literal(2.5) == "2.5"
    assert _sql_literal("a'b") == "'a''b'"  # single quotes doubled
    assert _sql_literal("a\\b") == "'a\\\\b'"  # backslash escaped for Spark SQL


def test_cast_value_by_type():
    assert _cast_value(None, "INT") is None
    assert _cast_value("2", "LONG") == 2
    assert _cast_value("2.5", "DOUBLE") == 2.5
    assert _cast_value("15.00", "DECIMAL") == 15.0
    assert _cast_value("true", "BOOLEAN") is True
    assert _cast_value("false", "BOOLEAN") is False
    assert _cast_value("hi", "STRING") == "hi"


def test_schemas_in_scans_three_part_names_only():
    sql = "CREATE TABLE shop.silver.orders AS SELECT * FROM shop.bronze.raw WHERE x = 10.00"
    # 10.00 is a literal, not a namespace; each catalog.schema appears once.
    assert _schemas_in(sql) == ["shop.silver", "shop.bronze"]


def _column(name, type_name):
    return SimpleNamespace(name=name, type_name=type_name)


def _resp(state, columns=None, data=None, next_chunk_index=None, error=None):
    return SimpleNamespace(
        statement_id="s1",
        status=SimpleNamespace(state=state, error=error),
        manifest=SimpleNamespace(schema=SimpleNamespace(columns=columns or [])),
        result=SimpleNamespace(data_array=data, next_chunk_index=next_chunk_index),
    )


class _FakeStatements:
    """Stand-in for w.statement_execution: canned submit + chunk fetch, no network."""

    def __init__(self, first, chunks=None, poll=None):
        self._first = first
        self._chunks = chunks or {}
        self._poll = list(poll or [])

    def execute_statement(self, **kwargs):
        return self._first

    def get_statement(self, statement_id):
        return self._poll.pop(0)

    def get_statement_result_chunk_n(self, statement_id, chunk_index):
        return self._chunks[chunk_index]


def _backend_with(statements):
    be = CloudBackend(warehouse_id="w")
    be._client = SimpleNamespace(statement_execution=statements)
    return be


def test_execute_sql_casts_and_paginates_chunks():
    cols = [_column("i", T.INT), _column("d", T.DOUBLE), _column("s", T.STRING)]
    first = _resp(
        StatementState.SUCCEEDED,
        columns=cols,
        data=[["1", "1.5", "a"]],
        next_chunk_index=1,
    )
    chunk1 = SimpleNamespace(data_array=[["2", "2.5", "b"]], next_chunk_index=None)
    be = _backend_with(_FakeStatements(first, chunks={1: chunk1}))

    assert be.execute_sql("SELECT ...") == [(1, 1.5, "a"), (2, 2.5, "b")]


def test_execute_sql_returns_empty_for_ddl_with_no_result_set():
    # A successful CREATE/INSERT/DROP comes back SUCCEEDED with result=None (and no manifest);
    # touching result.data_array would crash. seed/teardown/schema-prep all rely on this.
    ddl = SimpleNamespace(
        statement_id="s1",
        status=SimpleNamespace(state=StatementState.SUCCEEDED, error=None),
        manifest=None,
        result=None,
    )
    be = _backend_with(_FakeStatements(ddl))
    assert be.execute_sql("CREATE OR REPLACE TABLE t (a INT)") == []


def test_execute_sql_polls_until_terminal():
    cols = [_column("n", T.LONG)]
    pending = _resp(StatementState.PENDING)
    done = _resp(StatementState.SUCCEEDED, columns=cols, data=[["42"]])
    be = _backend_with(_FakeStatements(pending, poll=[done]))

    assert be.execute_sql("SELECT 42") == [(42,)]


def test_execute_sql_raises_on_failure():
    failed = _resp(
        StatementState.FAILED,
        error=SimpleNamespace(message="Table not found: nope"),
    )
    be = _backend_with(_FakeStatements(failed))
    with pytest.raises(RuntimeError, match="Table not found"):
        be.execute_sql("SELECT * FROM nope")


def test_execute_sql_requires_warehouse():
    be = CloudBackend()  # no warehouse configured
    be._warehouse_id = None
    with pytest.raises(RuntimeError, match="BUNDLETEST_WAREHOUSE_ID"):
        be.execute_sql("SELECT 1")


def test_volume_path_resolves_resource_name():
    be = CloudBackend()
    # Pretend the bundle summary is already loaded, so no subprocess/client is touched.
    be._summary = {
        "resources": {"volumes": {"raw_data": {"catalog_name": "shop", "schema_name": "bronze", "name": "raw_data"}}}
    }
    assert be._volume_path("/Volumes/raw_data/orders.csv") == "/Volumes/shop/bronze/raw_data/orders.csv"
    # A nested path keeps its tail.
    assert be._volume_path("/Volumes/raw_data/sub/f.csv") == "/Volumes/shop/bronze/raw_data/sub/f.csv"


def test_get_resource_keeps_inline_serialized_dashboard():
    be = CloudBackend()
    inline = {"serialized_dashboard": {"datasets": []}, "id": "abc"}
    be._summary = {"resources": {"dashboards": {"d": inline}}}
    # Already inline -> returned as-is, no workspace call (client stays None).
    assert be.get_resource("dashboards", "d") is inline
    assert be._client is None
