"""One test, two tiers.

These assertions run UNCHANGED on both backends, selected by BUNDLETEST_BACKEND:

    pytest                              # local (DuckDB): runs the real sql_task artifact
    BUNDLETEST_BACKEND=cloud pytest     # cloud: deploys + runs the same job on a workspace

There is no per-backend branching in the test itself — the point is to demonstrate the
"same test, two tiers" claim, not just assert it. Everything here is portable SQL semantics
(dedup, null filter, row count, uniqueness, column names); anything Databricks-specific
would be fenced with @pytest.mark.cloud_only instead. The cloud tier needs per-run
namespace isolation that lands with the cloud backend PR, so it is scoped out here for now
(see conftest.py).
"""


def test_active_users_dedupe(env):
    env.seed(
        "demo.bronze.events",
        [
            {"user_id": 1, "event": "login"},
            {"user_id": 1, "event": "login"},  # duplicate -> collapsed
            {"user_id": 2, "event": "login"},
            {"user_id": 3, "event": "logout"},  # not a login -> filtered
            {"user_id": None, "event": "login"},  # null id -> dropped
        ],
    )

    env.run_job("count_active")  # runs src/count_active.sql for real

    users = env.table("demo.gold.active_users")
    assert users.row_count() == 2
    assert users.has_no_nulls("user_id")
    assert users.column("user_id").is_unique()
    assert set(users.schema) == {"user_id"}
