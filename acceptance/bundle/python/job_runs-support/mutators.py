from dataclasses import replace
from typing import Any, Dict

from databricks.bundles.core import job_run_mutator
from databricks.bundles.job_runs import JobRun


@job_run_mutator
def update_job_run(run: JobRun) -> JobRun:
    # Update job_parameters dict with " (updated)" suffix to description
    params = run.job_parameters or {}
    updated_params: Dict[str, Any] = {}
    for key, value in params.items():
        if key == "description" and isinstance(value, str):
            updated_params[key] = f"{value} (updated)"
        else:
            updated_params[key] = value

    return replace(run, job_parameters=updated_params)
