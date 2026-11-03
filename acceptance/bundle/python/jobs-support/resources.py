from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_job("my_job_2", {"name": "My Job 2"})

    return resources
