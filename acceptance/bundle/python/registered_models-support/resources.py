from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_registered_model(
        "my_registered_model_2",
        {
            "name": "my_registered_model_2",
            "comment": "My model (2)",
        },
    )

    return resources
