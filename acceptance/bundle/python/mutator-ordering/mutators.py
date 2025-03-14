from dataclasses import replace
from databricks.bundles.jobs import Job, Task
from databricks.bundles.core import job_mutator, Bundle

# databricks.yml defines the order of mutators


@job_mutator
def add_task_2(bundle: Bundle, job: Job) -> Job:
    tasks = bundle.resolve_variable(job.tasks)
    task_2 = Task(task_key="task_2")

    return replace(job, tasks=[*tasks, task_2])


@job_mutator
def add_task_1(bundle: Bundle, job: Job) -> Job:
    tasks = bundle.resolve_variable(job.tasks)
    task_1 = Task(task_key="task_1")

    return replace(job, tasks=[*tasks, task_1])
