from databricks.bundles.jobs import Job

"""
The main job for pydabs_notebook_dlt_serverless.
"""

pydabs_notebook_dlt_serverless_job = Job.from_dict(
    {
        "name": "pydabs_notebook_dlt_serverless_job",
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
                "task_key": "notebook_task",
                "notebook_task": {
                    "notebook_path": "src/notebook.ipynb",
                },
            },
            {
                "task_key": "refresh_pipeline",
                "depends_on": [
                    {"task_key": "notebook_task"},
                ],
                "pipeline_task": {
                    "pipeline_id": "${resources.pipelines.pydabs_notebook_dlt_serverless_pipeline.id}",
                },
            },
        ],
    }
)
