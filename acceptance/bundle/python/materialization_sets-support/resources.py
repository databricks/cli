from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_materialization_set(
        "my_materialization_set_2",
        {
            "materialized_features": [
                {
                    "feature_name": "main.default.my_feature_2",
                    "offline_store_config": {
                        "catalog_name": "main",
                        "schema_name": "default",
                        "table_name_prefix": "my_feature_2_off",
                    },
                    "cron_schedule_trigger": {"mode": "DERIVED"},
                },
            ],
        },
    )

    return resources
