from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_model_service(
        "my_model_service_2",
        {
            "parent": "schemas/main.default",
            "model_service_id": "my_model_service_2",
            "comment": "My model service (2)",
        },
    )

    return resources
