from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_mlflow_experiment(
        "my_experiment_2",
        {
            "name": "/my_experiment_2",
        },
    )

    return resources
