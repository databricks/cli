import json
import re
from pathlib import Path
from typing import Optional

# Resources with a field type the generator can't model yet. Excluded until
# support for that type is added.
RESOURCE_DENYLIST = {
    "resources.ClusterPolicy",  # interface{}
    "resources.Dashboard",  # interface{}
    "resources.GenieSpace",  # interface{}
    "resources.Secret",  # time.Time
}

# Only GA and public-preview resources are generated; later stages may still change.
_EXCLUDED_RESOURCE_STAGES = {"PUBLIC_BETA", "PRIVATE_PREVIEW"}


def _resource_stage(config: dict, type_name: str) -> Optional[str]:
    node = config.get(type_name, {})
    if "x-databricks-launch-stage" in node:
        return node["x-databricks-launch-stage"]
    for option in node.get("oneOf", []):
        if "x-databricks-launch-stage" in option:
            return option["x-databricks-launch-stage"]
    return None


def _load_resource_namespace() -> dict[str, str]:
    """Map each generated resource type to its bundle section (plural) name.

    Derived from the Resources struct in the bundle schema so it stays in sync
    with the Go source, minus denylisted and non-public resources.
    """
    path = Path(__file__).parent / ".." / ".." / ".." / "bundle/schema/jsonschema.json"
    bundle = json.load(path.open())["$defs"]["github.com"]["databricks"]["cli"][
        "bundle"
    ]
    config = bundle["config"]
    properties = bundle["config.Resources"]["oneOf"][0]["properties"]

    namespace = {}
    for plural, prop in properties.items():
        ref = prop.get("$ref")
        if ref is None:
            options = prop.get("oneOf", []) + prop.get("anyOf", [])
            ref = next(o["$ref"] for o in options if o.get("$ref"))
        type_name = ref.split("/")[-1]

        if type_name in RESOURCE_DENYLIST:
            continue
        if _resource_stage(config, type_name) in _EXCLUDED_RESOURCE_STAGES:
            continue

        namespace[type_name] = plural

    return namespace


# All supported resource types and their namespace.
RESOURCE_NAMESPACE = _load_resource_namespace()

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


def to_snake_case(name: str) -> str:
    # "VectorSearchIndex" -> "vector_search_index"
    return re.sub(r"(?<!^)(?=[A-Z])", "_", name).lower()


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

    [_, name] = full_name.split(".")
    package_name = re.sub(r"(?<!^)(?=[A-Z])", "_", name).lower()

    return f"{get_root_package(namespace)}._models.{package_name}"
