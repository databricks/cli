"""Isolation test for one component (the transform_orders job).

Upstream (the ingest job that fills bronze) is stood in for with env.seed(...); the
real transform runs; assertions check its real output.
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
        "bronze.raw_orders",
        [
            {"order_id": 1, "total_price": 10.0},
            {"order_id": 1, "total_price": 10.0},  # duplicate
            {"order_id": 2, "total_price": 5.0},
            {"order_id": None, "total_price": 1.0},  # null id -> filtered out
        ],
    )

    result = env.run_job("transform_orders")
    assert result.succeeded

    silver = env.table("silver.orders")
    assert silver.row_count() == 2
    assert silver.has_no_nulls("order_id")
    assert silver.column("order_id").is_unique()


@pytest.mark.cloud_only
def test_price_is_decimal(env):
    # A schema/type assertion: faithful only on a real warehouse, so it must skip on
    # the in-memory backend rather than pass misleadingly.
    env.seed("bronze.raw_orders", [{"order_id": 1, "total_price": 10.0}])
    env.run_job("transform_orders")
    assert env.table("silver.orders").schema["total_price"] == "decimal(10,2)"
