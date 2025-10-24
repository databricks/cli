from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_app(
        "my_app_2",
        {
            "name": "My App (2)",
            "description": "My second app",
        },
    )

    return resources
