from databricks.bundles.core import Resources, Variable
from databricks.bundles.job_runs import JobRun, JobRunLifecycle, JobRunTrigger
from databricks.bundles.jobs import Job


def load_resources() -> Resources:
    resources = Resources()

    resources.add_job(
        "my_job",
        Job(name="My Job"),
    )
    resources.add_job_run(
        "my_run",
        JobRun(
            job_id=Variable(path="resources.jobs.my_job.id", type=int),
            job_parameters={"environment": "test"},
            lifecycle=JobRunLifecycle(triggers=[JobRunTrigger(on_bundle_deploy=True)]),
        ),
    )

    return resources
