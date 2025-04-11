from codegen.jsonschema import (
    Property,
    Schema,
    SchemaType,
    _parse_schema,
    _unwrap_variable,
    get_schemas,
)


def test_get_all_schemas():
    schemas = get_schemas()

    assert schemas


def test_unwrap_variable():
    out = _unwrap_variable(
        {
            "oneOf": [
                {"type": "object"},
                {
                    "type": "string",
                    "pattern": "\\$\\{(var(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)+)\\}",
                },
            ]
        }
    )

    assert out == {"type": "object"}


def test_parse_schema():
    spec = {
        "anyOf": [
            {
                "type": "object",
                "properties": {
                    "base_parameters": {
                        "description": "base_parameters description",
                        "$ref": "#/$defs/map/string",
                    },
                    "notebook_path": {"$ref": "#/$defs/string"},
                    "source": {
                        "$ref": "#/$defs/github.com/databricks/databricks-sdk-go/service/jobs.Source"
                    },
                    "warehouse_id": {"$ref": "#/$defs/string"},
                },
                "additionalProperties": False,
                "required": ["notebook_path"],
            },
            {
                "type": "string",
                "pattern": "\\$\\{(var(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)+)\\}",
            },
        ]
    }

    expected = Schema(
        type=SchemaType.OBJECT,
        properties={
            "base_parameters": Property(
                ref="#/$defs/map/string",
                description="base_parameters description",
            ),
            "notebook_path": Property(
                ref="#/$defs/string",
            ),
            "source": Property(
                ref="#/$defs/github.com/databricks/databricks-sdk-go/service/jobs.Source",
            ),
            "warehouse_id": Property(
                ref="#/$defs/string",
            ),
        },
        required=["notebook_path"],
    )

    assert _parse_schema(spec) == expected
