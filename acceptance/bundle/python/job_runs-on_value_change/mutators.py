from dataclasses import replace

from databricks.bundles.core import job_run_mutator
from databricks.bundles.job_runs import JobRun, JobRunLifecycle, JobRunTrigger


@job_run_mutator
def watch_y(run: JobRun) -> JobRun:
    lifecycle = run.lifecycle or JobRunLifecycle()
    triggers = [*lifecycle.triggers, JobRunTrigger(on_value_change="${var.y}")]

    return replace(run, lifecycle=replace(lifecycle, triggers=triggers))
