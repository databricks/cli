import json
from pathlib import Path
from dataclasses import dataclass, field
from enum import Enum
from typing import Optional

import codegen.packages as packages


class Stage:
    PRIVATE = "PRIVATE"


@dataclass
class Property:
    ref: str
    description: Optional[str] = None
    deprecated: Optional[bool] = None
    stage: Optional[str] = None


class SchemaType(Enum):
    OBJECT = "object"
    STRING = "string"


@dataclass
class Schema:
    type: SchemaType
    enum: list[str] = field(default_factory=list)
    properties: dict[str, Property] = field(default_factory=dict)
    required: list[str] = field(default_factory=list)
    description: Optional[str] = None
    deprecated: Optional[bool] = None
    stage: Optional[str] = None

    def __post_init__(self):
        match self.type:
            case SchemaType.OBJECT:
                assert not self.enum

            case SchemaType.STRING:
                assert not self.properties
                assert not self.required
                assert self.enum
            case _:
                raise ValueError(f"Unknown type: {self.type}")

        for item in self.enum:
            assert isinstance(item, str)

        for item in self.required:
            assert isinstance(item, str)


@dataclass
class Spec:
    schemas: dict[str, Schema]


def _unwrap_variable(schema: dict):
    # we assume that each field can be a variable

    if anyOf := schema.get("anyOf") or schema.get("oneOf"):
        if len(anyOf) != 2:
            return None

        [primary, variable] = anyOf

        pattern = variable.get("pattern", "")
        type = variable.get("type", "")

        if (
            type == "string"
            and pattern.startswith("\\$\\{")
            and pattern.endswith("\\}")
        ):
            return primary

    return None


def _parse_schema(schema: dict) -> Schema:
    schema = _unwrap_variable(schema) or schema
    properties = {}

    def _parse_bool(value) -> Optional[bool]:
        assert value is None or isinstance(value, bool)

        return value

    for k, v in schema.get("properties", {}).items():
        assert v.get("type") is None
        assert v.get("anyOf") is None
        assert v.get("properties") is None
        assert v.get("items") is None

        assert v.get("$ref")

        prop = Property(
            ref=v["$ref"],
            description=v.get("description"),
            deprecated=_parse_bool(v.get("deprecated")),
            stage=v.get("x-databricks-preview"),
        )

        properties[k] = prop

    assert schema.get("type") in [
        "object",
        "string",
    ], f"{schema} type not in ['object', 'string']"

    return Schema(
        type=SchemaType(schema["type"]),
        enum=schema.get("enum", []),
        properties=properties,
        required=schema.get("required", []),
        description=schema.get("description"),
        deprecated=_parse_bool(schema.get("deprecated")),
        stage=schema.get("x-databricks-preview"),
    )


def _load_spec() -> dict:
    path = (
        Path(__file__).parent  # ./experimental/python/codegen/codegen
        / ".."  # ./experimental/python/codegen
        / ".."  # ./experimental/python/
        / ".."  # ./experimental
        / ".."  # ./
        / "./bundle/schema/jsonschema.json"
    )

    return json.load(path.open())


def get_schemas():
    output = dict[str, Schema]()
    spec = _load_spec()

    sdk_types_spec = _get_spec_path(
        spec,
        ["$defs", "github.com", "databricks", "databricks-sdk-go", "service"],
    )
    resource_types_spec = _get_spec_path(
        spec,
        ["$defs", "github.com", "databricks", "cli", "bundle", "config"],
    )

    # we don't need all spec, only get supported types
    flat_spec = {**sdk_types_spec, **resource_types_spec}
    flat_spec = {
        key: value for key, value in flat_spec.items() if packages.should_load_ref(key)
    }

    for name, schema in flat_spec.items():
        try:
            output[name] = _parse_schema(schema)
        except Exception as e:
            raise ValueError(f"Failed to parse schema for {name}") from e

    return output


def _get_spec_path(spec: dict, path: list[str]) -> dict:
    for key in path:
        spec = spec[key]

    return spec
