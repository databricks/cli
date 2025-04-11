from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_pipeline("my_pipeline_2", {"name": "My Pipeline 2"})

    return resources
