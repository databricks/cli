from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_instance_pool(
        "my_pool_2",
        {
            "instance_pool_name": "my_pool_2",
            "node_type_id": "i3.xlarge",
        },
    )

    return resources
