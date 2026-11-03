from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_sql_warehouse(
        "my_sql_warehouse_2",
        {
            "name": "my_sql_warehouse_2",
            "cluster_size": "2X-Small",
            "auto_stop_mins": 10,
            "max_num_clusters": 1,
            "min_num_clusters": 1,
            "warehouse_type": "CLASSIC",
        },
    )

    return resources
