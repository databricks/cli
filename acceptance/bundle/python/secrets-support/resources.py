from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_secret(
        "my_secret_2",
        {
            "catalog_name": "main",
            "schema_name": "default",
            "name": "my_secret_2",
            "value": "${var.secret_value}",
            "expire_time": "2030-01-01T00:00:00Z",
        },
    )

    return resources
