from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_catalog(
        "my_catalog_2",
        {
            "name": "my_catalog_2",
            "comment": "My catalog (2)",
        },
    )

    return resources
