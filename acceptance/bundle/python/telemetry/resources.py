from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()
    resources.add_job("python_job", {"name": "Python Job"})
    resources.add_pipeline("python_pipeline", {"name": "Python Pipeline"})

    return resources
