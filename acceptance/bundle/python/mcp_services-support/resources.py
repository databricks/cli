from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_mcp_service(
        "my_mcp_service_2",
        {
            "parent": "schemas/main.default",
            "mcp_service_id": "my_mcp_service_2",
            "comment": "My MCP service (2)",
        },
    )

    return resources
