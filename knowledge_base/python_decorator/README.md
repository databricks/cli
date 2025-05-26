# Python decorator

This example shows how to package a **Python decorator–based job** so you can
call regular Python functions from a Databricks job with minimal boilerplate.

## How it works

| File                                   | Purpose                                                        |
|----------------------------------------|----------------------------------------------------------------|
| `src/my_project/tasks.py`              | Defines decorated task functions.                              |
| `src/my_project/jobs.py`               | Wires those tasks into a `Job` definition.                     |
| `src/my_project/python_wheel_task.py`  | Implements `@python_wheel_task` decorator.                     |
| `src/my_project/resources.py`          | Defines `load_resources` function to load `Job` definitions.   |

### Creating a task function

```python
# src/my_project/tasks.py
from my_project.python_wheel_task import python_wheel_task


@python_wheel_task
def get_message() -> None:
    from databricks.sdk.runtime import dbutils

    # Makes the value available to downstream tasks as
    # '{{ tasks.<task_key>.values.message }}'
    dbutils.jobs.taskValues.set("message", "Hello World")
```

### Referencing the task in a job

```python
# src/my_project/jobs.py
from my_project.tasks import get_message, print_message
from databricks.sdk.service import Job, Task

my_job = Job(
    name="Python Decorator Example",
    tasks=[
        Task(
            task_key="get_message",
            python_wheel_task=get_message(),
            ...
        ),
        Task(
            task_key="print_message",
            python_wheel_task=print_message(
                message="{{ tasks.get_message.values.message }}"
            ),
            ...
        ),
        ...
    ],
)
```

## Deploying the example

1. Create a virtual environment and install `databricks-bundles` package into it:

```bash
  python3 -m venv .venv
  .venv/bin/pip3 install databricks-bundles
```

2. Update the `host` field under `workspace` in `databricks.yml` to the Databricks workspace you wish to deploy to.

3. Run `databricks bundle deploy` to upload the wheel and deploy the job.

4. Run `databricks bundle run` to run either job.

## Cleaning up
To remove all assets created by this example:

```bash
  databricks bundle destroy
```
