import re
from typing import Optional

RESOURCE_NAMESPACE_OVERRIDE = {
    "resources.Job": "jobs",
    "resources.Pipeline": "pipelines",
    "resources.JobPermission": "jobs",
    "resources.JobPermissionLevel": "jobs",
    "resources.PipelinePermission": "pipelines",
    "resources.PipelinePermissionLevel": "pipelines",
}

# All supported resource types
RESOURCE_TYPES = [
    "resources.Job",
    "resources.Pipeline",
]

# Namespaces to load from OpenAPI spec.
#
# We can't load all types because of errors while loading some of them.
LOADED_NAMESPACES = [
    "compute",
    "jobs",
    "pipelines",
    "resources",
]

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


def is_resource(ref: str) -> bool:
    return ref in RESOURCE_TYPES


def should_load_ref(ref: str) -> bool:
    name = ref.split("/")[-1]

    # FIXME doesn't work, looks like enum, but doesn't have any values specified
    if name == "compute.Kind":
        return False

    for namespace in LOADED_NAMESPACES:
        if name.startswith(f"{namespace}."):
            return True

    return name in PRIMITIVES


def get_package(ref: str) -> Optional[str]:
    """
    Returns Python package for a given OpenAPI ref.
    Returns None for builtin types.
    """

    full_name = ref.split("/")[-1]

    if full_name in PRIMITIVES:
        return None

    [namespace, name] = full_name.split(".")

    if override := RESOURCE_NAMESPACE_OVERRIDE.get(full_name):
        namespace = override

    package_name = re.sub(r"(?<!^)(?=[A-Z])", "_", name).lower()

    return f"databricks.bundles.{namespace}._models.{package_name}"
