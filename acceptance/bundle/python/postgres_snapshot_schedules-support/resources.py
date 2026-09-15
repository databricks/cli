from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_postgres_snapshot_schedule(
        "my_snapshot_schedule_2",
        {
            "branch": "projects/other-project/branches/production",
            "schedule": [
                {
                    "retention": "604800s",
                    "daily_schedule": {"hour": 5},
                },
            ],
        },
    )

    return resources
