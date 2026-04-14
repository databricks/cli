import os

from databricks.bundles.core import Resources


def load_resources() -> Resources:
    unique_name = os.environ["UNIQUE_NAME"]
    spark_version = os.environ.get("DEFAULT_SPARK_VERSION", "13.3.x-scala2.12")
    node_type_id = os.environ.get("NODE_TYPE_ID", "i3.xlarge")

    resources = Resources()
    resources.add_job(
        resource_name="foo",
        job={
            "name": f"test-job-{unique_name}",
            "tasks": [
                {
                    "task_key": f"task_{i:04d}",
                    "notebook_task": {
                        "notebook_path": "/Shared/notebook",
                    },
                    "job_cluster_key": "main_cluster",
                }
                for i in range(1000)
            ],
            "job_clusters": [
                {
                    "job_cluster_key": "main_cluster",
                    "new_cluster": {
                        "spark_version": spark_version,
                        "node_type_id": node_type_id,
                        "num_workers": 1,
                    },
                }
            ],
        },
    )
    return resources
