from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_database_catalog(
        "my_database_catalog_2",
        {
            "database_instance_name": "test_instance",
            "database_name": "test_db_2",
            "name": "my_database_catalog_2",
        },
    )

    return resources
