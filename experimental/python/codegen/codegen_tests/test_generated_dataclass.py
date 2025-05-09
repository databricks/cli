import codegen.packages as packages
from codegen.generated_dataclass import (
    GeneratedDataclass,
    GeneratedField,
    dict_type,
    generate_dataclass,
    generate_type,
    GeneratedType,
    str_type,
    variable_or_type,
)
from codegen.jsonschema import Property, Schema, SchemaType


def test_generate_type_string():
    generated_type = generate_type(
        "#/$defs/string",
        is_param=False,
    )

    assert generated_type == GeneratedType(
        name="str",
        package=None,
        parameters=[],
    )


def test_generate_type_dict():
    generated_type = generate_type(
        "#/$defs/map/string",
        is_param=False,
    )

    assert generated_type == dict_type()


def test_generate_dataclass():
    generated = generate_dataclass(
        schema_name="jobs.Task",
        schema=Schema(
            type=SchemaType.OBJECT,
            description="task description",
            properties={
                "task_key": Property(
                    description="task_key description",
                    ref="#/$defs/string",
                ),
            },
            required=["task_key"],
        ),
    )

    assert generated == GeneratedDataclass(
        class_name="Task",
        package="databricks.bundles.jobs._models.task",
        description="task description",
        extends=[],
        fields=[
            GeneratedField(
                create_func_default=None,
                create_func_type_name=variable_or_type(str_type(), is_required=True),
                default=None,
                default_factory=None,
                description="task_key description",
                field_name="task_key",
                param_type_name=variable_or_type(str_type(), is_required=True),
                type_name=variable_or_type(str_type(), is_required=True),
                experimental=False,
            ),
        ],
        experimental=False,
    )


def test_get_class_name_string():
    class_name = packages.get_class_name("#/$defs/string")

    assert class_name == "str"
