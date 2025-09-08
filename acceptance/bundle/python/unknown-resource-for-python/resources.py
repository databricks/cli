from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()
    resources.add_job(
        resource_name="my_job_1",
        job={"name": "Job 1"},
    )

    return resources
