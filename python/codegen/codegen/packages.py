import os


def get_all_model_class_names(namespace: str, models_dir: str) -> list[str]:
    """
    Enumerate all generated model files in the _models/ directory for a namespace,
    extract the top-level class names, and return them as a list.
    """
    class_names = []
    for filename in os.listdir(models_dir):
        if not filename.endswith(".py") or filename == "__init__.py":
            continue
        file_path = os.path.join(models_dir, filename)
        with open(file_path, "r") as f:
            for line in f:
                line = line.strip()
                if line.startswith("class "):
                    # e.g. class ModelServingEndpoint(BaseModel):
                    class_name = line.split()[1].split("(")[0]
                    class_names.append(class_name)
    return class_names


import re
from typing import Optional

# All supported resource types and their namespace
RESOURCE_NAMESPACE = {
    "resources.Job": "jobs",
    "resources.Pipeline": "pipelines",
    "resources.Schema": "schemas",
    "resources.Volume": "volumes",
    "resources.ModelServingEndpoint": "serving",
    "resources.ModelServingEndpointPermission": "model_serving_endpoints",
    "resources.ModelServingEndpointPermissionLevel": "model_serving_endpoints",
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
    "model_serving_endpoints",
    "serving",
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
