from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_mlflow_model(
        "my_model_2",
        {
            "name": "my_model_2",
            "description": "My model (2)",
        },
    )

    return resources
