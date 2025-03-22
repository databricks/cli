from databricks.bundles.core import Resources


def load_resources_2() -> Resources:
    resources = Resources()
    resources.add_job(
        resource_name="my_job_2",
        job={"name": "Job 2"},
    )

    return resources


def load_resources_1() -> Resources:
    resources = Resources()
    resources.add_job(
        resource_name="my_job_1",
        job={"name": "Job 1"},
    )

    return resources
