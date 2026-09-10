"""Isolation test for one component (the transform_orders job).

Upstream (the ingest job that fills bronze) is stood in for with env.seed(...). The job's
*real* deployed SQL (src/transform_orders.sql) then runs against the seeded table, and the
assertions check its real output.
"""

import os

import pytest

from bundletest import bundle_env

BUNDLE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


@pytest.fixture
def env():
    with bundle_env(BUNDLE) as e:
        yield e


def test_transform_dedupes(env):
    env.seed(
        "shop.bronze.raw_orders",
        [
            {"order_id": 1, "total_price": 10.0},
            {"order_id": 1, "total_price": 10.0},  # duplicate
            {"order_id": 2, "total_price": 5.0},
            {"order_id": None, "total_price": 1.0},  # null id -> filtered out
        ],
    )

    result = env.run_job("transform_orders")
    assert result.succeeded

    silver = env.table("shop.silver.orders")
    assert silver.row_count() == 2
    assert silver.has_no_nulls("order_id")
    assert silver.column("order_id").is_unique()


@pytest.mark.cloud_only
def test_price_is_decimal(env):
    # A schema/type assertion tied to Databricks type naming (decimal(10,2) vs DuckDB's
    # DECIMAL(10,2)): faithful only on a real warehouse, so it skips on the local backend.
    env.seed("shop.bronze.raw_orders", [{"order_id": 1, "total_price": 10.0}])
    env.run_job("transform_orders")
    assert env.table("shop.silver.orders").schema["total_price"] == "decimal(10,2)"
