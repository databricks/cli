import re
from typing import Optional

# All supported resource types and their namespace
RESOURCE_NAMESPACE = {
    "resources.Job": "jobs",
    "resources.Pipeline": "pipelines",
    "resources.Catalog": "catalogs",
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


def get_schema_key(ref: str) -> str:
    """
    Get the schema dict key from a full $ref string, handling generic types.
    E.g., '#/$defs/.../resources.Permission[.../jobs.JobPermissionLevel]' -> 'resources.JobPermission'
    """
    if "[" not in ref or not ref.endswith("]"):
        return ref.split("/")[-1]

    bracket_pos = ref.index("[")
    before = ref[:bracket_pos]  # e.g., '#/$defs/.../resources.Permission'
    type_param = ref[bracket_pos + 1 : -1]  # e.g., '.../jobs.JobPermissionLevel'

    type_ns = before.split("/")[-1].split(".")[0]  # 'resources'
    param_class = type_param.split("/")[-1].split(".")[-1]  # 'JobPermissionLevel'

    if param_class.endswith("PermissionLevel"):
        class_name = param_class[: -len("Level")]  # 'JobPermission'
        return f"{type_ns}.{class_name}"

    return ref.split("/")[-1]


def get_class_name(ref: str) -> str:
    name = ref.split("/")[-1]

    # Generic type: last segment is the type parameter like "jobs.JobPermissionLevel]"
    if name.endswith("]"):
        name = name[:-1]  # strip "]"
        param_class = name.split(".")[-1]  # 'JobPermissionLevel'
        if param_class.endswith("PermissionLevel"):
            name = param_class[: -len("Level")]  # 'JobPermission'
        else:
            name = param_class
    else:
        name = name.split(".")[-1]

    return RENAMES.get(name, name)


def is_resource(ref: str) -> bool:
    return ref in RESOURCE_TYPES


def should_load_ref(ref: str) -> bool:
    name = ref.split("/")[-1]

    # Skip Go generic type fragments (their names contain '[' from embedded package paths)
    if "[" in name:
        return False

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

    # Generic type: last segment is the type parameter like "jobs.JobPermissionLevel]"
    if full_name.endswith("]"):
        full_name = full_name[:-1]  # strip "]"
        if full_name in PRIMITIVES:
            return None
        param_ns = full_name.split(".")[0]  # e.g., 'jobs'
        param_class = full_name.split(".")[-1]  # e.g., 'JobPermissionLevel'
        if param_class.endswith("PermissionLevel"):
            class_name = param_class[: -len("Level")]  # 'JobPermission'
        else:
            class_name = param_class
        package_name = re.sub(r"(?<!^)(?=[A-Z])", "_", class_name).lower()
        return f"{get_root_package(param_ns)}._models.{package_name}"

    if full_name in PRIMITIVES:
        return None

    [_, name] = full_name.split(".")
    package_name = re.sub(r"(?<!^)(?=[A-Z])", "_", name).lower()

    return f"{get_root_package(namespace)}._models.{package_name}"
