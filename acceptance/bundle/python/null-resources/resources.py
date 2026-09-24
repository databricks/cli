from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()
    resources.add_job("python_job", {"name": "Python Job", "tasks": []})
    return resources
