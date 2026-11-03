from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_model_serving_endpoint(
        "my_endpoint_2",
        {
            "name": "my_endpoint_2",
            "description": "My endpoint (2)",
        },
    )

    return resources
