from codegen.generated_enum import GeneratedEnum, generate_enum
from codegen.jsonschema import Schema, SchemaType


def test_generate_enum():
    generated = generate_enum(
        schema_name="jobs.MyEnum",
        schema=Schema(
            enum=["myEnumValue"],
            type=SchemaType.STRING,
            description="enum description",
        ),
    )

    assert generated == GeneratedEnum(
        class_name="MyEnum",
        package="databricks.bundles.jobs._models.my_enum",
        values={"MY_ENUM_VALUE": "myEnumValue"},
        description="enum description",
        experimental=False,
    )
