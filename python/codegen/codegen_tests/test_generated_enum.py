from codegen.generated_enum import GeneratedEnum, generate_enum, get_code
from codegen.jsonschema import Schema, SchemaType


def test_generate_enum():
    generated = generate_enum(
        namespace="bananas",
        schema_name="jobs.MyEnum",
        schema=Schema(
            enum=["myEnumValue"],
            type=SchemaType.STRING,
            description="enum description",
        ),
    )

    assert generated == GeneratedEnum(
        class_name="MyEnum",
        package="databricks.bundles.bananas._models.my_enum",
        values={"MY_ENUM_VALUE": "myEnumValue"},
        description="enum description",
        experimental=False,
        deprecated=False,
        open=False,
    )


def test_generate_open_enum():
    generated = generate_enum(
        namespace="catalogs",
        schema_name="catalog.Privilege",
        schema=Schema(enum=["SELECT"], type=SchemaType.STRING),
    )

    assert generated.open


def test_get_code_open_enum():
    code = get_code(
        GeneratedEnum(
            class_name="Privilege",
            package="databricks.bundles.catalogs._models.privilege",
            values={"SELECT": "SELECT"},
            description=None,
            experimental=False,
            deprecated=False,
            open=True,
        )
    )

    assert "def _missing_(cls, value: object)" in code
    assert 'PrivilegeParam = Literal["SELECT"] | Privilege | str' in code


def test_get_code_closed_enum():
    code = get_code(
        GeneratedEnum(
            class_name="Color",
            package="databricks.bundles.bananas._models.color",
            values={"RED": "RED"},
            description=None,
            experimental=False,
            deprecated=False,
        )
    )

    assert "_missing_" not in code
    assert 'ColorParam = Literal["RED"] | Color\n' in code
