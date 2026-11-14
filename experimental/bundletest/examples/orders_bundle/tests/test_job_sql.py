"""transform_orders — a SQL job (sql_task).

The local backend runs its real artifact (src/transform_orders.sql) against DuckDB, with
shop.bronze.raw_orders seeded to stand in for the upstream ingest job.

CAN test locally (DuckDB backend):
- job runs its real .sql without error (result.succeeded)
- output table row count / emptiness
- no-nulls, uniqueness, min / max on columns
- output column names (schema keys)

CANNOT test locally — needs the cloud backend:
- exact Databricks type names (decimal(10,2) vs DuckDB's DECIMAL(10,2))
- Databricks-only SQL functions (e.g. from_utc_timestamp) -> skips loudly
- run duration / SLA (local wall-clock is not the cluster's)
- real warehouse / cluster behavior
"""

import pytest


def test_transform_dedupes(env):
    env.seed(
        "shop.bronze.raw_orders",
        [
            {"order_id": 1, "total_price": 10.0},
            {"order_id": 1, "total_price": 10.0},  # duplicate
            {"order_id": 2, "total_price": 5.0},
            {"order_id": None, "total_price": 1.0},  # null id -> dropped
        ],
    )

    result = env.run_job("transform_orders")
    assert result.succeeded

    silver = env.table("shop.silver.orders")
    assert silver.row_count() == 2
    assert silver.has_no_nulls("order_id")
    assert silver.column("order_id").is_unique()
    assert silver.column("total_price").min() == 5.0
    assert silver.column("total_price").max() == 10.0
    assert set(silver.schema) == {"order_id", "total_price"}


@pytest.mark.cloud_only
def test_price_type_is_databricks_decimal(env):
    env.seed("shop.bronze.raw_orders", [{"order_id": 1, "total_price": 10.0}])
    env.run_job("transform_orders")
    assert env.table("shop.silver.orders").schema["total_price"] == "decimal(10,2)"
