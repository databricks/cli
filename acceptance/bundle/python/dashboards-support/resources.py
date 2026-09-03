from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_dashboard(
        "my_dashboard_2",
        {
            "display_name": "My Dashboard (2)",
            "serialized_dashboard": {"pages": [{"name": "python_page"}]},
            "warehouse_id": "my_warehouse_2",
        },
    )

    return resources
