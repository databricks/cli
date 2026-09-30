"""Resource config / wiring — read a resource's definition without running it.

The local backend reads it straight from databricks.yml; the cloud backend would read the
deployed config from the workspace API.

CAN test locally:
- a resource exists and is named correctly
- a job's task wiring (task_key, task type, sql file path)
- any declared field (cluster size, schedule, parameters, ...)

CANNOT test locally — needs the cloud backend:
- that the workspace actually accepted / deployed the config
- server-defaulted or normalized values

NOTE: env.resource(kind, name) wraps this same get_resource call for any resource kind, and
the typed handles (env.pipeline, env.dashboard, ...) add resource-specific accessors on top.
"""

import pytest


@pytest.mark.bundle_resource("jobs.transform_orders")
def test_job_is_wired_to_its_sql(env):
    job = env.backend.get_resource("jobs", "transform_orders")
    assert job["name"] == "transform_orders"

    task = job["tasks"][0]
    assert task["task_key"] == "transform"
    assert task["sql_task"]["file"]["path"] == "src/transform_orders.sql"
