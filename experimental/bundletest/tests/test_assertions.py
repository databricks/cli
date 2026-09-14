"""Framework tests against the DuckDB local backend.

Covers the assertion layer, the environment-level name binding, and the three-way run
routing: real bug -> red, dialect gap / notebook task / reserved catalog -> LocalUnsupported.
"""

import pytest
from bundletest import BundleEnv, DuckDBBackend, JobRunFailed, LocalUnsupported, bundle_env


@pytest.fixture
def env():
    e = BundleEnv("(none)", DuckDBBackend())
    yield e
    e.teardown()


def test_row_count_and_nulls(env):
    env.seed("app.data.t", [{"id": 1}, {"id": 2}, {"id": None}])
    t = env.table("app.data.t")
    assert t.row_count() == 3
    assert not t.has_no_nulls("id")


def test_column_stats(env):
    env.seed("app.data.t", [{"id": 1, "v": 10.0}, {"id": 2, "v": 5.0}, {"id": 2, "v": 1.0}])
    assert env.table("app.data.t").column("v").min() == 1.0
    assert env.table("app.data.t").column("v").max() == 10.0
    assert env.table("app.data.t").column("id").is_unique() is False
    assert env.table("app.data.t").column("v").is_unique() is True


def test_seeded_schema(env):
    env.seed("app.data.t", [{"id": 1, "name": "a", "price": 2.5}])
    assert env.table("app.data.t").schema == {
        "id": "INTEGER",
        "name": "VARCHAR",
        "price": "DOUBLE",
    }


def test_two_part_name(env):
    # schema.table in the default catalog also resolves.
    env.seed("staging.events", [{"id": 1}])
    assert env.table("staging.events").row_count() == 1


def test_numeric_literals_not_treated_as_namespaces(env):
    # 10.00 must not be parsed as a schema.table reference.
    env.seed("app.data.t", [{"v": 1}])
    assert env.backend.execute_sql("SELECT 10.00 FROM app.data.t")[0][0] == 10.0


def test_unknown_job_raises(env):
    with pytest.raises(KeyError):
        env.run_job("does_not_exist")


def _write_bundle(tmp_path, yml: str, sql: str | None = None):
    (tmp_path / "databricks.yml").write_text(yml)
    if sql is not None:
        (tmp_path / "job.sql").write_text(sql)


def test_wrong_table_name_fails_red(tmp_path):
    # The headline catch: the artifact references a table that was never created.
    _write_bundle(
        tmp_path,
        "resources:\n  jobs:\n    j:\n      tasks:\n        - task_key: t\n"
        "          sql_task:\n            file:\n              path: job.sql\n",
        "CREATE OR REPLACE TABLE app.gold.out AS SELECT * FROM app.bronze.does_not_exist;",
    )
    with bundle_env(str(tmp_path), backend="local") as env:
        # This job intentionally fails, so opt out of the raise-by-default and inspect it.
        result = env.run_job("j", check=False)
        assert not result.succeeded
        assert "does_not_exist" in result.error


def test_failing_job_raises_by_default(tmp_path):
    # Without check=False, a failed run raises JobRunFailed carrying the RunResult.
    _write_bundle(
        tmp_path,
        "resources:\n  jobs:\n    j:\n      tasks:\n        - task_key: t\n"
        "          sql_task:\n            file:\n              path: job.sql\n",
        "CREATE OR REPLACE TABLE app.gold.out AS SELECT * FROM app.bronze.does_not_exist;",
    )
    with bundle_env(str(tmp_path)) as env:
        with pytest.raises(JobRunFailed) as excinfo:
            env.run_job("j")
        assert not excinfo.value.result.succeeded


def test_notebook_task_skips(tmp_path):
    _write_bundle(
        tmp_path,
        "resources:\n  jobs:\n    j:\n      tasks:\n        - task_key: t\n"
        "          notebook_task:\n            notebook_path: /nb\n",
    )
    with bundle_env(str(tmp_path), backend="local") as env:
        with pytest.raises(LocalUnsupported):
            env.run_job("j")


def test_databricks_only_function_skips(tmp_path):
    _write_bundle(
        tmp_path,
        "resources:\n  jobs:\n    j:\n      tasks:\n        - task_key: t\n"
        "          sql_task:\n            file:\n              path: job.sql\n",
        "SELECT from_utc_timestamp(now(), 'UTC');",
    )
    with bundle_env(str(tmp_path), backend="local") as env:
        with pytest.raises(LocalUnsupported):
            env.run_job("j")


def test_reserved_catalog_skips(env):
    with pytest.raises(LocalUnsupported):
        env.seed("main.bronze.raw", [{"id": 1}])


def test_volume_upload_and_read(env, tmp_path):
    csv = tmp_path / "in.csv"
    csv.write_text("a,b\n1,x\n2,y\n")
    env.volume("raw").upload(str(csv))

    f = env.volume("raw").file("in.csv")
    assert f.exists()
    assert f.row_count() == 2
    assert f.columns == ["a", "b"]
    assert not env.volume("raw").file("missing.csv").exists()


def test_upload_missing_source_raises(env):
    with pytest.raises(FileNotFoundError):
        env.volume("raw").upload("/does/not/exist.csv")
