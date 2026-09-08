from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_secret_scope(
        "my_scope_2",
        {
            "name": "my_scope_2",
            "backend_type": "DATABRICKS",
        },
    )

    return resources
