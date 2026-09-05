import re
from typing import Optional

# All supported resource types and their namespace
RESOURCE_NAMESPACE = {
    "resources.Job": "jobs",
    "resources.JobRun": "job_runs",
    "resources.Pipeline": "pipelines",
    "resources.Catalog": "catalogs",
    "resources.Schema": "schemas",
    "resources.Volume": "volumes",
    "resources.Alert": "alerts",
}

RESOURCE_TYPES = list(RESOURCE_NAMESPACE.keys())

RENAMES = {
    "string": "str",
    "boolean": "bool",
    "integer": "int",
    "number": "float",
    "int64": "int",
    "float64": "float",
}

PRIMITIVES = [
    "string",
    "boolean",
    "integer",
    "number",
    "bool",
    "int",
    "int64",
    "float64",
]


def get_class_name(ref: str) -> str:
    name = ref.split("/")[-1]
    name = name.split(".")[-1]

    return RENAMES.get(name, name)


def get_snake_case_name(ref: str) -> str:
    class_name = get_class_name(ref)
    return re.sub(r"(?<!^)(?=[A-Z])", "_", class_name).lower()


def is_resource(ref: str) -> bool:
    return ref in RESOURCE_TYPES


def get_root_package(namespace: str) -> str:
    return f"databricks.bundles.{namespace}"


def get_package(namespace: str, ref: str) -> Optional[str]:
    """
    Returns Python package for a given OpenAPI ref.
    Returns None for builtin types.
    """

    full_name = ref.split("/")[-1]

    if full_name in PRIMITIVES:
        return None

    package_name = get_snake_case_name(ref)

    return f"{get_root_package(namespace)}._models.{package_name}"
