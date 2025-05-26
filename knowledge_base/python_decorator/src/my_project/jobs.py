from databricks.bundles.jobs import (
    Environment,
    Job,
    JobEnvironment,
    Task,
    TaskDependency,
)

from my_project.tasks import get_message, print_message

my_job = Job(
    name="Python Decorator Example",
    tasks=[
        Task(
            task_key="get_message",
            python_wheel_task=get_message(),
            # use serverless, alternatively, you can use 'libraries' with 'job_cluster_key'
            environment_key="Default",
        ),
        Task(
            task_key="print_message",
            environment_key="Default",
            python_wheel_task=print_message(
                message="{{ tasks.get_message.values.message }}"
            ),
            depends_on=[
                TaskDependency(task_key="get_message"),
            ],
        ),
    ],
    environments=[
        JobEnvironment(
            environment_key="Default",
            spec=Environment(
                client="2",
                dependencies=[
                    "dist/*.whl",
                ],
            ),
        )
    ],
)
