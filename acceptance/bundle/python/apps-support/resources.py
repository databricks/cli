from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_app(
        "my_app_2",
        {
            "name": "my_app_2",
            "description": "My app (2)",
            "source_code_path": "./app2",
        },
    )

    return resources
