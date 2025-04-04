from databricks.bundles.jobs import Job
from databricks.bundles.core import job_mutator, Bundle
import os


@job_mutator
def read_envs(bundle: Bundle, job: Job) -> Job:
    # This test ensures environment variables are not readable by PyDABs.
    # In restriction execution mode PyDABs mutators are not executed
    # and therefore code in this function won't run and envs.txt file won't exist
    value = os.getenv("SOME_ENV_VAR", "default")
    with open("envs.txt", "w") as file:
        file.write(value)
    return job
