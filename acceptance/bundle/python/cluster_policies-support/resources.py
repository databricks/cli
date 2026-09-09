from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    # interface{} field authored as a serialized JSON string
    resources.add_cluster_policy(
        "my_cluster_policy_2",
        {
            "name": "my_cluster_policy_2",
            "description": "My cluster policy (2)",
            "definition": '{"spark_version": {"type": "fixed", "value": "13.3.x-scala2.12"}}',
        },
    )

    # interface{} field authored as an inline dict
    resources.add_cluster_policy(
        "my_cluster_policy_3",
        {
            "name": "my_cluster_policy_3",
            "description": "My cluster policy (3)",
            "definition": {
                "spark_version": {"type": "fixed", "value": "13.3.x-scala2.12"}
            },
        },
    )

    return resources
