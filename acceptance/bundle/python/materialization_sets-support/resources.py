from databricks.bundles.core import Resources
from databricks.bundles.materialization_sets import (
    ExternalTrigger,
    MaterializationSet,
    MaterializedFeature,
    OfflineStoreConfig,
    OnlineStoreConfig,
    StreamingMode,
    TableTrigger,
)


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

    resources.add_materialization_set(
        "python_streaming_mode",
        MaterializationSet(
            materialized_features=[
                MaterializedFeature(
                    feature_name="main.default.python_streaming_feature",
                    online_store_config=OnlineStoreConfig(
                        online_store_name="my_online_store",
                        catalog_name="main",
                        schema_name="default",
                        table_name_prefix="python_streaming_feature_on",
                    ),
                    streaming_mode=StreamingMode(),
                ),
            ],
        ),
    )

    resources.add_materialization_set(
        "python_table_trigger",
        MaterializationSet(
            materialized_features=[
                MaterializedFeature(
                    feature_name="main.default.python_table_feature",
                    offline_store_config=OfflineStoreConfig(
                        catalog_name="main",
                        schema_name="default",
                        table_name_prefix="python_table_feature_off",
                    ),
                    table_trigger=TableTrigger(),
                ),
            ],
        ),
    )

    resources.add_materialization_set(
        "python_external_trigger",
        MaterializationSet(
            materialized_features=[
                MaterializedFeature(
                    feature_name="main.default.python_external_feature",
                    offline_store_config=OfflineStoreConfig(
                        catalog_name="main",
                        schema_name="default",
                        table_name_prefix="python_external_feature_off",
                    ),
                    external_trigger=ExternalTrigger(),
                ),
            ],
        ),
    )

    return resources
