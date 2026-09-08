from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_job_run(
        "my_run_2",
        {
            "job_id": 1,
            "job_parameters": {"description": "test"},
        },
    )

    return resources
