from dataclasses import replace

from databricks.bundles.core import postgres_snapshot_schedule_mutator
from databricks.bundles.postgres_snapshot_schedules import PostgresSnapshotSchedule


@postgres_snapshot_schedule_mutator
def update_postgres_snapshot_schedule(
    schedule: PostgresSnapshotSchedule,
) -> PostgresSnapshotSchedule:
    assert isinstance(schedule.branch, str)

    return replace(schedule, branch=f"{schedule.branch} (updated)")
