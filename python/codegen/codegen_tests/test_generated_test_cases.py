"""Tests for generated_test_cases.py, the codegen that builds example resource
values for the generated pydabs test suite.

For the unfamiliar: the generator reads a resource's JSON schema and synthesizes
one placeholder "value tree", then renders it two ways -- as a Python dict literal
and as a dataclass constructor call. Downstream tests use the pair to check that
converting the dict form yields the dataclass form. These tests cover the pieces
of that pipeline: parsing schema refs, synthesizing the value tree (and the rules
for which fields it includes), the two renderers, collecting the imports the
rendered code needs, and the source of the collector module that gathers it all.
"""

import codegen.jsonschema as openapi
import pytest
from codegen.generated_test_cases import (
    _collect_imports,
    _collector_code,
    _Enum,
    _is_composite,
    _List,
    _Map,
    _Object,
    _ref_name,
    _render_dataclass,
    _render_dict,
    _Scalar,
    _synth_object,
    _synth_ref,
    _synth_scalar,
)
from codegen.generated_wiring import _WiredResource
from codegen.jsonschema import Property, Schema, SchemaType

# Full $refs as they appear in jsonschema.json; only the last segment is
# significant to the code under test, but keeping the SDK prefix makes the
# fixtures read like the real spec.
_SDK = "#/$defs/github.com/databricks/databricks-sdk-go/service"
_COND_REF = f"{_SDK}/sql.AlertCondition"
_OP_REF = f"{_SDK}/sql.ComparisonOperator"


# _ref_name pulls the type name (the last path segment) out of a schema $ref.
def test_ref_name():
    assert _ref_name(_COND_REF) == "sql.AlertCondition"
    assert _ref_name("#/$defs/string") == "string"


# _is_composite separates refs that need recursive synthesis (list/map/object/enum)
# from plain scalar refs (string, int, ...).
def test_is_composite():
    assert _is_composite("#/$defs/slice/string")
    assert _is_composite("#/$defs/map/string")
    assert _is_composite(_COND_REF)
    assert not _is_composite("#/$defs/string")
    assert not _is_composite("#/$defs/int64")


# Each primitive type maps to a fixed placeholder value (a string uses the field
# name; numbers/bools use 0 / 0.0 / True).
@pytest.mark.parametrize(
    "name,expected",
    [
        ("string", _Scalar('"hint"', '"hint"')),
        ("integer", _Scalar("0", "0")),
        ("int", _Scalar("0", "0")),
        ("int64", _Scalar("0", "0")),
        ("number", _Scalar("0.0", "0.0")),
        ("float64", _Scalar("0.0", "0.0")),
        ("boolean", _Scalar("True", "True")),
        ("bool", _Scalar("True", "True")),
    ],
)
def test_synth_scalar(name, expected):
    assert _synth_scalar(name, "hint") == expected


# An unrecognized primitive means the schema has a type the generator doesn't
# model, so it fails loudly rather than emitting a bad value.
def test_synth_scalar_unknown_raises():
    with pytest.raises(ValueError, match="Unknown primitive: duration"):
        _synth_scalar("duration", "hint")


# A scalar ref uses the enclosing field's name as its string placeholder, so the
# generated example reads like "name" rather than a generic token.
def test_synth_ref_scalar_uses_field_name_as_hint():
    assert _synth_ref("jobs", "#/$defs/string", "name", {}, set()) == _Scalar(
        '"name"', '"name"'
    )


# A list ref becomes a one-element list whose single item is synthesized from the
# element type.
def test_synth_ref_list_recurses_on_element():
    assert _synth_ref("jobs", "#/$defs/slice/string", "tags", {}, set()) == _List(
        _Scalar('"tags"', '"tags"')
    )


# A map ref becomes one {"key": "value"} entry -- the generator only ever emits
# string-keyed, string-valued maps.
def test_synth_ref_map_is_always_string_keyed():
    assert _synth_ref("jobs", "#/$defs/map/string", "labels", {}, set()) == _Map(
        "key", _Scalar('"value"', '"value"')
    )


# Any other kind of map is never produced, so hitting one fails loudly.
def test_synth_ref_non_string_map_raises():
    with pytest.raises(ValueError, match="Unsupported map ref"):
        _synth_ref("jobs", "#/$defs/map/integer", "labels", {}, set())


# An enum ref becomes an _Enum node carrying the chosen value plus the class name,
# module, and member the generated code will reference.
def test_synth_ref_enum():
    schemas = {
        "sql.ComparisonOperator": Schema(type=SchemaType.STRING, enum=["greaterThan"]),
    }

    assert _synth_ref("alerts", _OP_REF, "op", schemas, set()) == _Enum(
        value="greaterThan",
        class_name="ComparisonOperator",
        module="databricks.bundles.alerts._models.comparison_operator",
        member="GREATER_THAN",
    )


# A type that (transitively) requires itself has no finite example, so synthesis
# must detect the cycle and stop instead of recursing forever.
def test_synth_ref_required_cycle_raises():
    schemas = {"sql.AlertCondition": Schema(type=SchemaType.OBJECT)}

    # The referenced object is already on the current path: a required cycle has
    # no finite value, so synthesis must fail rather than recurse forever.
    with pytest.raises(
        ValueError, match=r"Required-field cycle through 'sql.AlertCondition'"
    ):
        _synth_ref("alerts", _COND_REF, "condition", schemas, {"sql.AlertCondition"})


# Which properties land in a resource's example: the field-selection policy.
def test_synth_object_field_policy():
    # A top-level resource keeps: all required fields (scalar + composite), and
    # stable optional composite fields. It drops optional scalars.
    schemas = {
        "resources.Alert": Schema(
            type=SchemaType.OBJECT,
            properties={
                "display_name": Property(ref="#/$defs/string"),
                "condition": Property(ref=_COND_REF),
                "seconds_to_retrigger": Property(ref="#/$defs/int"),
                "tags": Property(ref="#/$defs/slice/string"),
            },
            required=["display_name", "condition"],
        ),
        "sql.AlertCondition": Schema(
            type=SchemaType.OBJECT,
            properties={
                "op": Property(ref="#/$defs/string"),
                "threshold": Property(ref="#/$defs/slice/string"),
            },
            required=["op"],
        ),
    }

    example = _synth_object(
        "alerts",
        "resources.Alert",
        schemas["resources.Alert"],
        schemas,
        set(),
        top_level=True,
    )

    assert example == _Object(
        class_name="Alert",
        module="databricks.bundles.alerts._models.alert",
        fields=[
            ("display_name", _Scalar('"display_name"', '"display_name"')),
            # Nested object contributes only its required field (op); the nested
            # optional composite (threshold) is dropped because top_level is False.
            (
                "condition",
                _Object(
                    class_name="AlertCondition",
                    module="databricks.bundles.alerts._models.alert_condition",
                    fields=[("op", _Scalar('"op"', '"op"'))],
                ),
            ),
            # seconds_to_retrigger (optional scalar) is dropped.
            ("tags", _List(_Scalar('"tags"', '"tags"'))),
        ],
    )


# An optional field is only included if it is "stable": deprecated fields and
# ones still in beta / private preview are left out (absent stage counts as GA).
@pytest.mark.parametrize(
    "deprecated,stage,kept",
    [
        (None, None, True),
        (None, openapi.LaunchStage.PUBLIC_PREVIEW, True),
        (True, None, False),
        (None, openapi.LaunchStage.PUBLIC_BETA, False),
        (None, openapi.LaunchStage.PRIVATE_PREVIEW, False),
    ],
)
def test_synth_object_optional_composite_stability(deprecated, stage, kept):
    schemas = {
        "resources.Alert": Schema(
            type=SchemaType.OBJECT,
            properties={
                "tags": Property(
                    ref="#/$defs/slice/string", deprecated=deprecated, stage=stage
                ),
            },
            required=[],
        ),
    }

    example = _synth_object(
        "alerts",
        "resources.Alert",
        schemas["resources.Alert"],
        schemas,
        set(),
        top_level=True,
    )

    assert bool(example.fields) == kept


# Inside a nested object (anything that isn't the resource itself) only required
# fields are kept, which keeps examples bounded.
def test_synth_object_nested_drops_all_optional():
    schemas = {
        "sql.AlertCondition": Schema(
            type=SchemaType.OBJECT,
            properties={
                "op": Property(ref="#/$defs/string"),
                "operand": Property(ref="#/$defs/slice/string"),
            },
            required=["op"],
        ),
    }

    example = _synth_object(
        "alerts",
        "sql.AlertCondition",
        schemas["sql.AlertCondition"],
        schemas,
        set(),
        top_level=False,
    )

    assert [name for name, _ in example.fields] == ["op"]


# --- rendering -------------------------------------------------------------

_VALUE_TREE = _Object(
    class_name="Alert",
    module="databricks.bundles.alerts._models.alert",
    fields=[
        ("display_name", _Scalar('"display_name"', '"display_name"')),
        (
            "op",
            _Enum(
                value="greaterThan",
                class_name="ComparisonOperator",
                module="databricks.bundles.alerts._models.comparison_operator",
                member="GREATER_THAN",
            ),
        ),
        ("tags", _List(_Scalar('"tags"', '"tags"'))),
        ("labels", _Map("key", _Scalar('"value"', '"value"'))),
    ],
)


# Rendering a value tree as a Python dict-literal string (the "dict_example" form).
def test_render_dict():
    assert _render_dict(_VALUE_TREE) == (
        '{"display_name": "display_name", "op": "greaterThan", '
        '"tags": ["tags"], "labels": {"key": "value"}}'
    )


# Rendering the same tree as a dataclass-constructor string (the "dataclass_example"
# form); the two renderers must agree on structure but differ on enums.
def test_render_dataclass():
    # Enums render as a class member reference, unlike the dict form's raw string.
    assert _render_dataclass(_VALUE_TREE) == (
        'Alert(display_name="display_name", op=ComparisonOperator.GREATER_THAN, '
        'tags=["tags"], labels={"key": "value"})'
    )


# The dataclass example references object and enum classes; this collects the
# (module, class) imports it needs, reaching into lists and maps to find them.
def test_collect_imports_gathers_objects_and_enums_through_containers():
    out: set[tuple[str, str]] = set()
    _collect_imports(_VALUE_TREE, out)

    assert out == {
        ("databricks.bundles.alerts._models.alert", "Alert"),
        ("databricks.bundles.alerts._models.comparison_operator", "ComparisonOperator"),
    }


# A tree of only primitives references no classes, so it needs no imports.
def test_collect_imports_scalar_only_tree_is_empty():
    out: set[tuple[str, str]] = set()
    _collect_imports(_Scalar('"x"', '"x"'), out)

    assert out == set()


# Source of the _generated/__init__.py that imports each resource's module and
# gathers their test cases into a single `test_cases` list.
def test_collector_code():
    resources = [
        _WiredResource(
            class_name="Alert",
            singular_name="alert",
            plural_name="alerts",
            model_module="databricks.bundles.alerts._models.alert",
        ),
        _WiredResource(
            class_name="Job",
            singular_name="job",
            plural_name="jobs",
            model_module="databricks.bundles.jobs._models.job",
        ),
    ]

    assert _collector_code(resources) == (
        "from databricks_tests.core._generated import (\n"
        "    alerts,\n"
        "    jobs,\n"
        ")\n"
        "\n"
        '__all__ = ["test_cases"]\n'
        "\n"
        "test_cases = [\n"
        "    alerts._test_case(),\n"
        "    jobs._test_case(),\n"
        "]\n"
    )
