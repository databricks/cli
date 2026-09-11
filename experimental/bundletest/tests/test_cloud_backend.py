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
    # Spark escapes with a backslash; a doubled quote would drop the quote (verified live).
    assert _sql_literal("a'b") == "'a\\'b'"  # single quote -> \'
    assert _sql_literal("a\\b") == "'a\\\\b'"  # backslash -> \\


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


def test_run_job_success_result_mapping():
    from databricks.sdk.service.jobs import RunResultState

    # Successful job run: succeeded=True, run_duration ms -> seconds conversion
    be = CloudBackend(warehouse_id="w")
    be._summary = {"resources": {"jobs": {"myjob": {"id": "123"}}}}
    be._config = {"resources": {"jobs": {"myjob": {"tasks": []}}}}
    # Mock the workspace client
    be._client = SimpleNamespace(
        jobs=SimpleNamespace(
            run_now=lambda job_id, job_parameters=None: SimpleNamespace(
                result=lambda: SimpleNamespace(
                    state=SimpleNamespace(result_state=RunResultState.SUCCESS),
                    run_duration=2000,
                    run_id=456,
                )
            )
        ),
        statement_execution=_FakeStatements(_resp(StatementState.SUCCEEDED)),
    )

    result = be.run_job("myjob")
    assert result.result_state == "SUCCESS"
    assert result.succeeded is True
    assert result.duration_seconds == 2.0
    assert result.run_id == "456"
    assert result.error == ""


def test_run_job_failed_result_mapping():
    from databricks.sdk.service.jobs import RunResultState

    # Failed job run: succeeded=False, error populated, run_duration converted
    be = CloudBackend(warehouse_id="w")
    be._summary = {"resources": {"jobs": {"failing_job": {"id": "789"}}}}
    be._config = {"resources": {"jobs": {"failing_job": {"tasks": []}}}}
    be._client = SimpleNamespace(
        jobs=SimpleNamespace(
            run_now=lambda job_id, job_parameters=None: SimpleNamespace(
                result=lambda: SimpleNamespace(
                    state=SimpleNamespace(
                        result_state=RunResultState.FAILED,
                        state_message="Task failed: invalid syntax",
                    ),
                    run_duration=5000,
                    run_id=999,
                )
            )
        ),
        statement_execution=_FakeStatements(_resp(StatementState.SUCCEEDED)),
    )

    result = be.run_job("failing_job")
    assert result.result_state == "FAILED"
    assert result.succeeded is False
    assert result.duration_seconds == 5.0
    assert result.run_id == "999"
    assert result.error == "Task failed: invalid syntax"


def test_run_job_failed_with_no_message():
    from databricks.sdk.service.jobs import RunResultState

    # Failed run with no state_message uses empty string
    be = CloudBackend(warehouse_id="w")
    be._summary = {"resources": {"jobs": {"job": {"id": "1"}}}}
    be._config = {"resources": {"jobs": {"job": {"tasks": []}}}}
    be._client = SimpleNamespace(
        jobs=SimpleNamespace(
            run_now=lambda job_id, job_parameters=None: SimpleNamespace(
                result=lambda: SimpleNamespace(
                    state=SimpleNamespace(
                        result_state=RunResultState.FAILED,
                        state_message=None,
                    ),
                    run_duration=0,
                    run_id=111,
                )
            )
        ),
        statement_execution=_FakeStatements(_resp(StatementState.SUCCEEDED)),
    )

    result = be.run_job("job")
    assert result.succeeded is False
    assert result.error == ""


def test_table_schema_filters_special_rows():
    # DESCRIBE TABLE returns real columns, then blank name or '#'-prefixed rows;
    # only return actual columns in the schema dict.
    cols = [
        _column("col_name", T.STRING),
        _column("data_type", T.STRING),
        _column("comment", T.STRING),
    ]
    first = _resp(
        StatementState.SUCCEEDED,
        columns=cols,
        data=[
            ["id", "LONG", ""],
            ["name", "STRING", ""],
            ["created_at", "STRING", ""],
            ["", "", ""],  # blank col_name marks end of real columns
            ["# Partition Information", "STRING", ""],
        ],
    )
    be = _backend_with(_FakeStatements(first))

    schema = be.table_schema("main.default.users")
    assert schema == {"id": "LONG", "name": "STRING", "created_at": "STRING"}


def test_table_schema_stops_at_hash_prefix():
    # DESCRIBE can also have '#'-prefixed row as the break indicator.
    cols = [
        _column("col_name", T.STRING),
        _column("data_type", T.STRING),
        _column("comment", T.STRING),
    ]
    first = _resp(
        StatementState.SUCCEEDED,
        columns=cols,
        data=[
            ["x", "INT", ""],
            ["#Partition", "STRING", ""],  # starts with '#', marks end
            ["y", "INT", ""],
        ],
    )
    be = _backend_with(_FakeStatements(first))

    schema = be.table_schema("main.default.t")
    assert schema == {"x": "INT"}


def test_seed_table_creates_with_inferred_types():
    # seed_table creates CREATE OR REPLACE with inferred types from rows.
    submitted = []

    def capture_execute(query):
        submitted.append(query)
        return []

    be = CloudBackend(warehouse_id="w")
    be.execute_sql = capture_execute
    be._ensure_schema = lambda fqn: None

    rows = [
        {"id": 1, "name": "Alice", "score": 95.5},
        {"id": 2, "name": "Bob", "score": 87.3},
    ]
    be.seed_table("main.default.scores", rows)

    assert len(submitted) == 2
    create_sql = submitted[0]
    assert "CREATE OR REPLACE TABLE main.default.scores" in create_sql
    assert "id BIGINT" in create_sql
    assert "name STRING" in create_sql
    assert "score DOUBLE" in create_sql


def test_seed_table_inserts_with_escaped_literals():
    # seed_table inserts with properly escaped string literals.
    submitted = []

    def capture_execute(query):
        submitted.append(query)
        return []

    be = CloudBackend(warehouse_id="w")
    be.execute_sql = capture_execute
    be._ensure_schema = lambda fqn: None

    rows = [
        {"name": "Alice"},
        {"name": "O'Brien"},  # single quote needs escaping
        {"name": "Path\\to\\file"},  # backslash needs escaping
    ]
    be.seed_table("main.default.names", rows)

    insert_sql = submitted[1]
    assert "INSERT INTO main.default.names" in insert_sql
    # Spark escapes with backslash: single quote -> \', backslash -> \\
    assert "'Alice'" in insert_sql
    assert "'O\\'Brien'" in insert_sql
    assert "'Path\\\\to\\\\file'" in insert_sql


def test_seed_table_tracks_fqn_for_teardown():
    # seed_table adds fqn to be._seeded so teardown will drop it.
    submitted = []

    def capture_execute(query):
        submitted.append(query)
        return []

    be = CloudBackend(warehouse_id="w")
    be.execute_sql = capture_execute
    be._ensure_schema = lambda fqn: None

    rows = [{"x": 1}]
    be.seed_table("catalog.schema.table_one", rows)
    be.seed_table("catalog.schema.table_two", rows)

    assert "catalog.schema.table_one" in be._seeded
    assert "catalog.schema.table_two" in be._seeded


def test_teardown_does_not_raise_on_execute_sql_failure():
    # teardown is best-effort: does not raise when execute_sql fails.
    be = CloudBackend()
    be._seeded = {"main.default.t1", "main.default.t2"}

    def fail_execute(query):
        raise RuntimeError("warehouse down")

    be.execute_sql = fail_execute

    # Should not raise, even with both tables failing to drop
    be.teardown()


def test_teardown_does_not_raise_on_bundle_destroy_failure():
    # teardown is best-effort: does not raise when bundle destroy fails.
    be = CloudBackend()
    be._seeded = set()  # no seeded tables

    def fail_bundle(*args):
        raise RuntimeError("bundle destroy failed")

    be._bundle = fail_bundle

    # Should not raise
    be.teardown()


def test_teardown_cleans_up_both_seeded_and_bundle():
    # teardown calls DROP TABLE on each seeded table, then bundle destroy.
    dropped = []
    destroyed = []

    def capture_execute(query):
        dropped.append(query)
        return []

    def capture_bundle(*args):
        destroyed.append(args)
        return ""

    be = CloudBackend()
    be._seeded = {"main.default.t1", "main.default.t2"}
    be.execute_sql = capture_execute
    be._bundle = capture_bundle

    be.teardown()

    # All seeded tables should be dropped
    assert len(dropped) == 2
    assert all("DROP TABLE IF EXISTS" in q for q in dropped)
    # Bundle destroy should be called
    assert ("destroy", "--auto-approve") in destroyed
