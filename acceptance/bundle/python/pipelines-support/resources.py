from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_pipeline("my_pipeline_1", {"name": "My Pipeline 1"})

    return resources
