from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()
    resources.add_job(
        resource_name="my_job",
        job={"name": "My Job"},
    )

    return resources
