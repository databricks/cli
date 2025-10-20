from databricks.bundles.jobs import Job

"""
The main job for pydabs_dlt.
This job runs pydabs_dlt_pipeline on a schedule.
"""

pydabs_dlt_job = Job.from_dict(
    {
        "name": "pydabs_dlt_job",
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
                    "pipeline_id": "${resources.pipelines.pydabs_dlt_pipeline.id}",
                },
            },
        ],
        "job_clusters": [
            {
                "job_cluster_key": "job_cluster",
                "new_cluster": {
                    "spark_version": "15.4.x-scala2.12",
                    "node_type_id": "[NODE_TYPE_ID]",
                    "data_security_mode": "SINGLE_USER",
                    "autoscale": {
                        "min_workers": 1,
                        "max_workers": 4,
                    },
                },
            }
        ],
    }
)
