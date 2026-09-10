from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_quality_monitor(
        "my_monitor_2",
        {
            "assets_dir": "/Workspace/monitoring",
            "output_schema_name": "default.monitoring",
            "table_name": "default.test_table",
        },
    )

    return resources
