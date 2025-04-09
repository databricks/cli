from databricks.bundles.core import Bundle, Resources
from databricks.bundles.jobs import Job


def load_resources(bundle: Bundle) -> Resources:
    resources = Resources()

    my_job = Job.from_dict(
        {
            "name": "My Job",
            "tasks": [
                {
                    "task_key": "my_notebook",
                    "notebook_task": {
                        "notebook_path": "my_notebook.py",
                    },
                },
            ],
        }
    )

    resources.add_job("job1", my_job)

    return resources
