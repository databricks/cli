from dataclasses import replace

import pytest
from codegen.jsonschema import get_schemas
from codegen.jsonschema_patch import (
    DROP_FIELD_DESCRIPTIONS,
    DROP_SCHEMA_DESCRIPTIONS,
    drop_descriptions,
)


def test_drop_descriptions():
    schemas = get_schemas()

    output = drop_descriptions(schemas)

    for schema_name in DROP_SCHEMA_DESCRIPTIONS:
        schema = schemas[schema_name]
        assert schema.description
        assert output[schema_name] == replace(schema, description=None)

    for schema_name, field_names in DROP_FIELD_DESCRIPTIONS.items():
        for field_name in field_names:
            prop = schemas[schema_name].properties[field_name]
            assert prop.description
            assert output[schema_name].properties[field_name] == replace(
                prop, description=None
            )


def test_drop_descriptions_unknown_schema():
    schemas = get_schemas()
    schemas.pop("jobs.SqlConditionTriggerMode")

    with pytest.raises(ValueError, match="unknown schemas"):
        drop_descriptions(schemas)


@pytest.mark.parametrize("description", [None, ""])
def test_drop_descriptions_empty_schema_description(description):
    schemas = get_schemas()
    schemas["jobs.SqlConditionTriggerMode"] = replace(
        schemas["jobs.SqlConditionTriggerMode"], description=description
    )

    with pytest.raises(ValueError, match=r"Schema description drop .* is a no-op"):
        drop_descriptions(schemas)
