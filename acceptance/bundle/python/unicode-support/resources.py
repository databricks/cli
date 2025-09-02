from databricks.bundles.core import Resources, Diagnostics


def load_resources() -> Resources:
    resources = Resources()

    resources.add_job("job_2", {"name": "🔥🔥🔥"})

    resources.add_diagnostics(
        Diagnostics.create_warning("This is a warning message with unicode characters: 🔥🔥🔥"),
    )

    return resources
