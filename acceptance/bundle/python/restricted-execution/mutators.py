from databricks.bundles.jobs import Job
from databricks.bundles.core import job_mutator, Bundle
import os


@job_mutator
def read_envs(bundle: Bundle, job: Job) -> Job:
    # This test ensures environment variables are not readable by PyDABs in restriction execution mode
    # because PyDABs mutators are not executed in the first place
    value = os.getenv("SOME_ENV_VAR", "default")
    with open("envs.txt", "w") as file:
        file.write(value)
    return job
