from databricks.bundles.jobs import Job, Task, NotebookTask, JobCluster
from databricks.bundles.variables import Bundle

my_job = Job(
    name="Test Job",
    resource_name="my_job",
    job_clusters=[
        JobCluster(
            job_cluster_key="my_cluster",
            new_cluster=Bundle.variables.default_cluster_spec,
        ),
    ],
    tasks=[
        Task(
            task_key="my_notebook_task",
            job_cluster_key="my_cluster",
            notebook_task=NotebookTask(
                notebook_path="notebooks/my_notebook.py",
            ),
        ),
    ],
)
