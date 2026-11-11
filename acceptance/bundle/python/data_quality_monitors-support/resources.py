from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_data_quality_monitor(
        "my_data_quality_monitor_2",
        {
            "object_type": "table",
            "object_id": "66666666-7777-8888-9999-000000000000",
            "data_profiling_config": {
                "assets_dir": "/Workspace/Users/user@example.com/assets",
                "output_schema_id": "11111111-2222-3333-4444-555555555555",
                "snapshot": {},
            },
        },
    )

    return resources
