from my_project.python_wheel_task import python_wheel_task


@python_wheel_task
def get_message() -> None:
    from databricks.sdk.runtime import dbutils

    # Makes the value available to downstream tasks as
    # '{{ tasks.<task_key>.values.message }}'
    dbutils.jobs.taskValues.set("message", "Hello World")


@python_wheel_task
def print_message(message: str) -> None:
    print(f"Message: {message}")
