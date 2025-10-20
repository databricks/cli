from databricks.bundles.jobs import Job

"""
The main job for pydabs_python_serverless.
"""

pydabs_python_serverless_job = Job.from_dict(
    {
        "name": "pydabs_python_serverless_job",
        "trigger": {
            # Run this job every day, exactly one day from the last run; see https://docs.databricks.com/api/workspace/jobs/create#trigger
            "periodic": {
                "interval": 1,
                "unit": "DAYS",
            },
        },
        # "email_notifications": {
        #     "on_failure": [
        #         "[USERNAME]",
        #     ],
        # },
        "tasks": [
            {
                "task_key": "main_task",
                "environment_key": "default",
                "python_wheel_task": {
                    "package_name": "pydabs_python_serverless",
                    "entry_point": "main",
                },
            },
        ],
        # A list of task execution environment specifications that can be referenced by tasks of this job.
        "environments": [
            {
                "environment_key": "default",
                # Full documentation of this spec can be found at:
                # https://docs.databricks.com/api/workspace/jobs/create#environments-spec
                "spec": {
                    "environment_version": "2",
                    "dependencies": [
                        "dist/*.whl",
                    ],
                },
            }
        ],
    }
)
