from dataclasses import replace

from databricks.bundles.core import job_run_mutator
from databricks.bundles.job_runs import JobRun


@job_run_mutator
def update_job_run(job_run: JobRun) -> JobRun:
    assert isinstance(job_run.job_parameters, dict)

    return replace(
        job_run,
        job_parameters={**job_run.job_parameters, "mutated": "true"},
    )
