from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_model_provider_service(
        "my_model_provider_service_2",
        {
            "parent": "schemas/main.default",
            "model_provider_service_id": "my_model_provider_service_2",
            "comment": "My model provider service (2)",
        },
    )

    return resources
