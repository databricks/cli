import re
from dataclasses import dataclass
from typing import Optional

import codegen.packages as packages
from codegen.code_builder import CodeBuilder
from codegen.generated_dataclass import _append_description
from codegen.jsonschema import Schema, Stage


@dataclass(kw_only=True)
class GeneratedEnum:
    class_name: str
    package: str
    values: dict[str, str]
    description: Optional[str]
    experimental: bool


def generate_enum(schema_name: str, schema: Schema) -> GeneratedEnum:
    assert schema.enum

    class_name = packages.get_class_name(schema_name)
    package = packages.get_package(schema_name)
    values = {}

    assert package

    for value in schema.enum:
        values[_camel_to_upper_snake(value)] = value

    return GeneratedEnum(
        class_name=class_name,
        package=package,
        values=values,
        description=schema.description,
        experimental=schema.stage == Stage.PRIVATE,
    )


def get_code(generated: GeneratedEnum) -> str:
    b = CodeBuilder()

    # Example:
    #
    # class Color(Enum):
    #
    b.append(f"class {generated.class_name}(Enum):")
    b.newline()

    _append_description(b, generated.description, generated.experimental)

    # Example:
    #
    #   RED = "RED"
    #
    for key, value in generated.values.items():
        b.indent().append(f'{key} = "{value}"')
        b.newline()

    b.newline()

    # Example:
    #
    #  ColorParam = Literal["RED", "GREEN", "BLUE"] | Color

    b.append(generated.class_name).append('Param = Literal["')
    b.append_list(list(generated.values.values()), sep='", "')
    b.append('"] | ', generated.class_name)
    b.newline()

    return b.build()


def _camel_to_upper_snake(value):
    s1 = re.sub("(.)([A-Z][a-z]+)", r"\1_\2", value)

    return re.sub("([a-z0-9])([A-Z])", r"\1_\2", s1).upper()
