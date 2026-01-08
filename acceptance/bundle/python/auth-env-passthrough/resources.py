import os

from databricks.bundles.core import Bundle, Resources


def load_resources(bundle: Bundle) -> Resources:
    host = os.environ.get("DATABRICKS_HOST", "NOT_SET")

    if host == "NOT_SET":
        raise ValueError("DATABRICKS_HOST was not passed to Python subprocess")

    if not host.startswith("http://") and not host.startswith("https://"):
        raise ValueError(f"DATABRICKS_HOST has invalid format: {host}")

    home = os.environ.get("HOME", os.environ.get("USERPROFILE", "NOT_SET"))
    if home == "NOT_SET":
        raise ValueError("HOME/USERPROFILE was not passed to Python subprocess")

    profile = os.environ.get("DATABRICKS_CONFIG_PROFILE")
    if profile is not None:
        raise ValueError(
            f"DATABRICKS_CONFIG_PROFILE should have been removed but was: {profile}. "
            "Conflicting auth env vars must be cleaned to prevent auth issues."
        )

    resources = Resources()
    resources.add_job(
        resource_name="test_job",
        job={"name": "auth_env_test_job"},
    )

    return resources
