"""End-to-end validation of the cloud backend against a real workspace.

Exercises the seam methods that can only be verified on cloud: deploy, run_job on the
deployed jobs, execute_sql/table_schema round-trips, get_resource off `bundle summary`
(which inlines a file_path dashboard's serialized form), and volume upload/read.
"""

import pytest


@pytest.mark.bundle_resource("jobs.transform_orders", "jobs.aggregate_orders")
def test_bronze_to_silver_to_gold(env, schema):
    env.seed(
        f"{schema}.raw_orders",
        [
            {"order_id": 1, "total_price": 10.0},
            {"order_id": 1, "total_price": 10.0},  # duplicate
            {"order_id": 2, "total_price": 5.0},
            {"order_id": None, "total_price": 1.0},  # null id -> dropped
        ],
    )

    assert env.run_job("transform_orders").succeeded  # bronze -> silver
    silver = env.table(f"{schema}.orders")
    assert silver.row_count() == 2
    assert silver.has_no_nulls("order_id")
    assert silver.column("order_id").is_unique()

    assert env.run_job("aggregate_orders").succeeded  # silver -> gold
    summary = env.table(f"{schema}.order_summary")
    assert summary.row_count() == 1
    assert summary.column("order_count").min() == 2
    assert summary.column("total_revenue").min() == 15.0


@pytest.mark.cloud_only
@pytest.mark.bundle_resource("jobs.transform_orders")
def test_price_type_is_databricks_decimal(env, schema):
    env.seed(f"{schema}.raw_orders", [{"order_id": 1, "total_price": 10.0}])
    env.run_job("transform_orders")
    assert env.table(f"{schema}.orders").schema["total_price"] == "decimal(10,2)"


@pytest.mark.bundle_resource("jobs.transform_orders")
def test_job_is_wired_to_its_sql(env):
    job = env.backend.get_resource("jobs", "transform_orders")
    assert job["tasks"][0]["sql_task"]["file"]["path"].endswith("transform_orders.sql")


@pytest.mark.bundle_resource("dashboards.orders_overview")
def test_dashboard_source_tables_from_file_path(env, schema):
    # The dashboard is defined by file_path, not inline, yet source_tables() still resolves:
    # `bundle summary` inlines the file's serialized form at config-load, so get_resource has it.
    dashboard = env.dashboard("orders_overview")
    assert dashboard.exists()
    assert dashboard.source_tables() == [f"{schema}.order_summary"]


@pytest.mark.bundle_resource("volumes.raw_data")
def test_uploaded_csv_is_readable(env, tmp_path):
    csv = tmp_path / "orders.csv"
    csv.write_text("order_id,total_price\n1,10.0\n2,5.0\n")

    env.volume("raw_data").upload(str(csv))

    orders = env.volume("raw_data").file("orders.csv")
    assert orders.exists()
    assert orders.row_count() == 2
    assert "order_id" in orders.columns


@pytest.mark.cloud_only
@pytest.mark.bundle_resource("jobs.transform_orders")
def test_deployed_job_carries_server_filled_fields(env):
    # get_deployed reads the workspace's stored object, so it carries values the server filled
    # in or normalized that our databricks.yml never declared — what get_resource (the declared
    # config) cannot show. This is the point of validating against real deployment.
    deployed = env.backend.get_deployed("jobs", "transform_orders")
    assert deployed["settings"]["name"] == "transform_orders"
    assert deployed["settings"]["format"] == "MULTI_TASK"  # server-normalized
    assert deployed["settings"]["max_concurrent_runs"] == 1  # server default
    assert deployed["run_as_user_name"]  # server-assigned
