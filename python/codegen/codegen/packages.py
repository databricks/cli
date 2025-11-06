import re
from typing import Optional

# All supported resource types and their namespace
RESOURCE_NAMESPACE = {
    "resources.Job": "jobs",
    "resources.Pipeline": "pipelines",
    "resources.Schema": "schemas",
    "resources.Volume": "volumes",
}

RESOURCE_TYPES = list(RESOURCE_NAMESPACE.keys())

# Namespaces to load from OpenAPI spec.
#
# We can't load all types because of errors while loading some of them.
LOADED_NAMESPACES = [
    "compute",
    "jobs",
    "pipelines",
    "resources",
    "catalog",
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

    for namespace in LOADED_NAMESPACES:
        if name.startswith(f"{namespace}."):
            return True

    return name in PRIMITIVES


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

    [_, name] = full_name.split(".")
    package_name = re.sub(r"(?<!^)(?=[A-Z])", "_", name).lower()

    return f"{get_root_package(namespace)}._models.{package_name}"
