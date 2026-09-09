from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_vector_search_endpoint(
        "my_endpoint_2",
        {
            "name": "my_endpoint_2",
            "endpoint_type": "STANDARD",
        },
    )

    return resources
