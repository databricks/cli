from databricks.bundles.jobs import Job

"""
The main job for my_jobs_as_code.
"""


my_jobs_as_code_job = Job.from_dict(
    {
        "name": "my_jobs_as_code_job",
        "trigger": {
            # Run this job every day, exactly one day from the last run; see https://docs.databricks.com/api/workspace/jobs/create#trigger
            "periodic": {
                "interval": 1,
                "unit": "DAYS",
            },
        },
        "email_notifications": {
            "on_failure": [
                "[USERNAME]",
            ],
        },
        "tasks": [
            {
                "task_key": "notebook_task",
                "job_cluster_key": "job_cluster",
                "notebook_task": {
                    "notebook_path": "src/notebook.ipynb",
                },
            },
            {
                "task_key": "main_task",
                "depends_on": [
                    {
                        "task_key": "notebook_task",
                    },
                ],
                "job_cluster_key": "job_cluster",
                "python_wheel_task": {
                    "package_name": "my_jobs_as_code",
                    "entry_point": "main",
                },
                "libraries": [
                    # By default we just include the .whl file generated for the my_jobs_as_code package.
                    # See https://docs.databricks.com/dev-tools/bundles/library-dependencies.html
                    # for more information on how to add other libraries.
                    {
                        "whl": "dist/*.whl",
                    },
                ],
            },
        ],
        "job_clusters": [
            {
                "job_cluster_key": "job_cluster",
                "new_cluster": {
                    "spark_version": "15.4.x-scala2.12",
                    "node_type_id": "i3.xlarge",
                    "autoscale": {
                        "min_workers": 1,
                        "max_workers": 4,
                    },
                },
            },
        ],
    }
)
