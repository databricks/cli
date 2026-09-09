from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    # interface{} field authored as a serialized JSON string
    resources.add_genie_space(
        "my_genie_space_2",
        {
            "title": "my_genie_space_2",
            "description": "My genie space (2)",
            "warehouse_id": "abc123",
            "parent_path": "/Workspace/Users/me",
            "serialized_space": '{"instructions": ["Answer questions about sales data"]}',
        },
    )

    # interface{} field authored as an inline dict
    resources.add_genie_space(
        "my_genie_space_3",
        {
            "title": "my_genie_space_3",
            "description": "My genie space (3)",
            "warehouse_id": "abc123",
            "parent_path": "/Workspace/Users/me",
            "serialized_space": {"instructions": ["Answer questions about sales data"]},
        },
    )

    return resources
