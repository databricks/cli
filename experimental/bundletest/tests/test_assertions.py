"""Framework tests: exercise the assertion layer against the in-memory backend."""

import pytest

from bundletest import BundleEnv, InMemoryBackend, bundle_env


@pytest.fixture
def env():
    e = BundleEnv("(none)", InMemoryBackend())
    yield e
    e.teardown()


def test_row_count_and_nulls(env):
    env.seed("db.t", [{"id": 1}, {"id": 2}, {"id": None}])
    t = env.table("db.t")
    assert t.row_count() == 3
    assert not t.has_no_nulls("id")


def test_column_stats(env):
    env.seed("db.t", [{"id": 1, "v": 10.0}, {"id": 2, "v": 5.0}, {"id": 2, "v": 1.0}])
    assert env.table("db.t").column("v").min() == 1.0
    assert env.table("db.t").column("v").max() == 10.0
    assert env.table("db.t").column("id").is_unique() is False
    assert env.table("db.t").column("v").is_unique() is True


def test_seeded_schema(env):
    env.seed("db.t", [{"id": 1, "name": "a", "price": 2.5}])
    assert env.table("db.t").schema == {"id": "INTEGER", "name": "TEXT", "price": "REAL"}


def test_table_exists(env):
    env.seed("db.t", [{"id": 1}])
    assert env.table("db.t").exists()
    assert not env.table("db.missing").exists()


def test_numeric_literals_not_rewritten(env):
    # 10.00 looks like a dotted table ref but must be left alone.
    env.seed("db.t", [{"v": 1}])
    assert env.backend.execute_sql("SELECT 10.00 FROM db.t")[0][0] == 10.0


def test_unknown_job_raises(env):
    with pytest.raises(KeyError):
        env.run_job("does_not_exist")


def test_failing_job_reports_failed(tmp_path):
    (tmp_path / "transforms.py").write_text(
        "JOBS = {'boom': lambda sql: sql('SELECT * FROM missing_table_xyz')}\n"
    )
    with bundle_env(str(tmp_path)) as env:
        result = env.run_job("boom")
        assert not result.succeeded
        assert result.error
