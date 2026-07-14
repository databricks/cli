from databricks.bundles.jobs import Job
from databricks.bundles.core import job_mutator, Bundle
import os


@job_mutator
def capture_profile_env(bundle: Bundle, job: Job) -> Job:
    # The CLI must propagate DATABRICKS_CONFIG_PROFILE to the python subprocess
    # so the Databricks SDK can disambiguate when multiple profiles share a host.
    value = os.getenv("DATABRICKS_CONFIG_PROFILE", "<unset>")
    with open("captured_env.txt", "w") as f:
        f.write(value)
    return job
