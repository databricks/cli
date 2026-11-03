from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_alert(
        "my_alert_2",
        {
            "display_name": "My Alert (2)",
            "query_text": "SELECT 2",
            "warehouse_id": "my_warehouse_2",
            "evaluation": {
                "comparison_operator": "LESS_THAN",
                "source": {"name": "column_2"},
            },
            "schedule": {
                "quartz_cron_schedule": "0 0 12 * * ?",
                "timezone_id": "UTC",
            },
        },
    )

    return resources
