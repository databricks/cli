from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_volume(
        "my_volume_2",
        {
            "name": "My Volume (2)",
            "catalog_name": "my_catalog_2",
            "schema_name": "my_schema_2",
        },
    )

    return resources
