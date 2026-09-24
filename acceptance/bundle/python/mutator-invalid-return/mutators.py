from databricks.bundles.core import job_mutator
from databricks.bundles.jobs import Job


@job_mutator
def invalid_return(job: Job):
    return {"name": "not a Job"}
