from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_feature(
        "my_feature_2",
        {
            "full_name": "main.default.my_feature_2",
            "description": "My Feature (2)",
            "source": {
                "delta_table_source": {"full_name": "main.default.my_source_2"},
            },
            "function": {
                "aggregation_function": {
                    "count_function": {"input": "value"},
                    "time_window": {
                        "sliding": {
                            "window_duration": "604800s",
                            "slide_duration": "86400s",
                        },
                    },
                },
            },
        },
    )

    return resources
