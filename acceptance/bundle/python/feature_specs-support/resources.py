from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_feature_spec(
        "my_feature_spec_2",
        {
            "name": "main.default.my_feature_spec_2",
            "features": [
                {"declarative_feature": {"feature": "main.default.my_feature"}},
            ],
            "extra_pip_requirements": {"requirements": ["numpy==2.1.0"]},
        },
    )

    return resources
