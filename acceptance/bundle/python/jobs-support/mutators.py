from dataclasses import replace

from databricks.bundles.core import job_mutator
from databricks.bundles.jobs import Job


@job_mutator
def update_job(job: Job) -> Job:
    assert isinstance(job.name, str)

    return replace(job, name=f"{job.name} (updated)")
