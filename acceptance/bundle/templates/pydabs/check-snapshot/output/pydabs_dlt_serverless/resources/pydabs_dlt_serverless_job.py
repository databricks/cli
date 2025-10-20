from databricks.bundles.jobs import Job

"""
The main job for pydabs_dlt_serverless.
This job runs pydabs_dlt_serverless_pipeline on a schedule.
"""

pydabs_dlt_serverless_job = Job.from_dict(
    {
        "name": "pydabs_dlt_serverless_job",
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
                "task_key": "refresh_pipeline",
                "pipeline_task": {
                    "pipeline_id": "${resources.pipelines.pydabs_dlt_serverless_pipeline.id}",
                },
            },
        ],
    }
)
