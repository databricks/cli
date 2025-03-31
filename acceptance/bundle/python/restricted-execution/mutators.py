from dataclasses import replace
from databricks.bundles.jobs import Job, Task
from databricks.bundles.core import job_mutator, Bundle
import os


@job_mutator
def read_envs(bundle: Bundle, job: Job) -> Job:
    value = os.getenv("SOME_ENV_VAR", "default")
    with open("envs.txt", "w") as file:
        file.write(value)

    tasks = bundle.resolve_variable(job.tasks)
    task_2 = Task(task_key="task_2")

    return replace(job, tasks=[*tasks, task_2])
