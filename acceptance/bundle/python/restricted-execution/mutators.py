from dataclasses import replace
from databricks.bundles.jobs import Job
from databricks.bundles.core import job_mutator, Bundle
import os


@job_mutator
def read_envs(bundle: Bundle, job: Job) -> Job:
    # Reading from envs to ensure that the envs are accessible for this mutator
    # (which this test case is verifying that they are not)
    value = os.getenv("SOME_ENV_VAR", "default")
    with open("envs.txt", "w") as file:
        file.write(value)
    return job
