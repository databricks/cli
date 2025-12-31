from dataclasses import replace
from databricks.bundles.jobs import Job
from databricks.bundles.core import job_mutator, Bundle
from databricks.sdk import WorkspaceClient


@job_mutator
def test_workspace_client(bundle: Bundle, job: Job) -> Job:
    w = WorkspaceClient()
    user = w.current_user.me()
    return replace(job, description=f"Validated by user: {user.user_name}")
