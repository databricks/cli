from databricks.bundles.core import (
    Bundle,
    Resources,
    load_resources_from_current_package_module,
)


def load_resources(bundle: Bundle) -> Resources:
    """
    'load_resources' function is referenced in databricks.yml and is responsible for loading
    bundle resources defined in Python code. This function is called by Databricks CLI during
    bundle deployment. After deployment, this function is not used.
    """

    # the default implementation loads all Python files in 'resources' directory
    return load_resources_from_current_package_module()
