from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_job_run(
        "py_run",
        {
            "job_id": "${resources.jobs.my_job.id}",
            "lifecycle": {"triggers": [{"on_value_change": "${var.x}"}]},
        },
    )

    return resources
