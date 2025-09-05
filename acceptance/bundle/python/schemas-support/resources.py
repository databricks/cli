from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_schema(
        "my_schema_2",
        {
            "name": "My Schema (2)",
            "catalog_name": "my_catalog_2",
        },
    )

    return resources
