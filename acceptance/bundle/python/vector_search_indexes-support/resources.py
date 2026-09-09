from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_vector_search_index(
        "my_index_2",
        {
            "name": "my_index_2",
            "endpoint_name": "my_endpoint",
            "primary_key": "id",
            "index_type": "DELTA_SYNC",
        },
    )

    return resources
