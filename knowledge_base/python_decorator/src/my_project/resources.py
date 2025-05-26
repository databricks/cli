from databricks.bundles.core import Resources, load_resources_from_package_module

import my_project.jobs


def load_resources() -> Resources:
    return load_resources_from_package_module(my_project.jobs)
