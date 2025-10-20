from databricks.bundles.jobs import Job

"""
The main job for pydabs_notebook_dlt.
"""

pydabs_notebook_dlt_job = Job.from_dict(
    {
        "name": "pydabs_notebook_dlt_job",
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
                "job_cluster_key": "job_cluster",
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
                    "pipeline_id": "${resources.pipelines.pydabs_notebook_dlt_pipeline.id}",
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
