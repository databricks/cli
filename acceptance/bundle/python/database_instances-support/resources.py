from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_database_instance(
        "my_database_instance_2",
        {
            "name": "my_database_instance_2",
            "capacity": "CU_1",
        },
    )

    return resources
