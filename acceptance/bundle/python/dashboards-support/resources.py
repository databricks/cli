from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    # interface{} field authored as a serialized JSON string
    resources.add_dashboard(
        "my_dashboard_2",
        {
            "display_name": "my_dashboard_2",
            "warehouse_id": "abc123",
            "parent_path": "/Workspace/Users/me",
            "serialized_dashboard": '{"pages": [{"name": "main", "displayName": "Main"}]}',
        },
    )

    return resources
