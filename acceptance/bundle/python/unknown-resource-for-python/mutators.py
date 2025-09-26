from dataclasses import replace
from databricks.bundles.jobs import Job
from databricks.bundles.core import job_mutator


@job_mutator
def update_job(job: Job) -> Job:
    return replace(job, name="Updated Job Name")
