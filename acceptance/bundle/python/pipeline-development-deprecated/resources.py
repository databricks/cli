from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_pipeline("with_development", {"name": "With development", "development": True})
    resources.add_pipeline("without_development", {"name": "Without development"})

    return resources
