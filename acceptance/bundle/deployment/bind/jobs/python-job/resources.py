from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_job(
        resource_name="job_1",
        job={"name": "Updated Job"},
    )

    return resources
