"""End-to-end: chain two SQL jobs through the shared local environment.

bronze -> (transform_orders) -> silver -> (aggregate_orders) -> gold. Both jobs run their
real .sql; the first's output persists in the env's DuckDB connection for the second to read.

CAN test locally (DuckDB backend):
- an all-SQL chain, hop by hop, asserting the final table
- intermediate tables between hops (silver here)

CANNOT test locally — needs the cloud backend:
- a chain containing any notebook / Python / pipeline step -> skips at that hop
- automatic dependency ordering (here the test sequences run_job calls itself)
"""


def test_bronze_to_silver_to_gold(env):
    env.seed(
        "shop.bronze.raw_orders",
        [
            {"order_id": 1, "total_price": 10.0},
            {"order_id": 1, "total_price": 10.0},  # duplicate
            {"order_id": 2, "total_price": 5.0},
            {"order_id": None, "total_price": 1.0},  # null id -> dropped
        ],
    )

    assert env.run_job("transform_orders").succeeded  # bronze -> silver
    assert env.table("shop.silver.orders").row_count() == 2  # intermediate hop

    assert env.run_job("aggregate_orders").succeeded  # silver -> gold

    summary = env.table("shop.gold.order_summary")
    assert summary.row_count() == 1
    # single summary row: min() over the column returns its only value
    assert summary.column("order_count").min() == 2
    assert summary.column("total_revenue").min() == 15.0
