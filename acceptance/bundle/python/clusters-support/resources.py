from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_cluster(
        "my_cluster_2",
        {
            "cluster_name": "my_cluster_2",
            "spark_version": "13.3.x-scala2.12",
            "num_workers": 1,
        },
    )

    return resources
