from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_synced_database_table(
        "my_synced_table_2",
        {
            "name": "main.default.my_synced_table_2",
        },
    )

    return resources
