import re
from dataclasses import dataclass
from typing import Optional

import codegen.packages as packages
from codegen.code_builder import CodeBuilder
from codegen.generated_dataclass import _append_description
from codegen.jsonschema import Schema, is_experimental_stage

# Enums whose platform value set grows faster than the schema. They keep their
# members for completion, but unknown strings are accepted instead of rejected
# (see https://github.com/databricks/cli/issues/6770).
OPEN_ENUMS = {"catalog.Privilege"}


@dataclass(kw_only=True)
class GeneratedEnum:
    class_name: str
    package: str
    values: dict[str, str]
    description: Optional[str]
    experimental: bool
    deprecated: bool
    open: bool = False


def generate_enum(namespace: str, schema_name: str, schema: Schema) -> GeneratedEnum:
    assert schema.enum

    class_name = packages.get_class_name(schema_name)
    package = packages.get_package(namespace, schema_name)
    values = {}

    assert package

    for value in schema.enum:
        name = _camel_to_upper_snake(value)
        # Distinct values must not collapse to the same member (e.g. "a-b" and "a_b").
        if name in values:
            raise ValueError(
                f"{schema_name}: enum values {values[name]!r} and {value!r} both map to member {name!r}"
            )
        values[name] = value

    return GeneratedEnum(
        class_name=class_name,
        package=package,
        values=values,
        description=schema.description,
        experimental=is_experimental_stage(schema.stage),
        deprecated=schema.deprecated or False,
        open=schema_name in OPEN_ENUMS,
    )


def get_code(generated: GeneratedEnum) -> str:
    b = CodeBuilder()

    # Example:
    #
    # class Color(Enum):
    #
    b.append(f"class {generated.class_name}(Enum):")
    b.newline()

    _append_description(
        b,
        generated.description,
        experimental=generated.experimental,
        deprecated=generated.deprecated,
    )

    # Example:
    #
    #   RED = "RED"
    #
    for key, value in generated.values.items():
        b.indent().append(f'{key} = "{value}"')
        b.newline()

    if generated.open:
        b.newline()
        _append_missing(b, generated.class_name)

    b.newline()

    # Example:
    #
    #  ColorParam = Literal["RED", "GREEN", "BLUE"] | Color

    b.append(generated.class_name).append('Param = Literal["')
    b.append_list(list(generated.values.values()), sep='", "')
    b.append('"] | ', generated.class_name)

    if generated.open:
        b.append(" | str")

    b.newline()

    return b.build()


def _append_missing(b: CodeBuilder, class_name: str):
    # Example:
    #
    #   @classmethod
    #   def _missing_(cls, value: object) -> "Color | None":
    #       ...
    #
    # Unknown strings become pseudo-members, the same way stdlib Flag builds
    # them. They are cached in _value2member_map_ so repeated lookups return
    # the same object and dataclass equality keeps working.
    b.indent().append("@classmethod").newline()
    b.indent().append(
        f'def _missing_(cls, value: object) -> "{class_name} | None":'
    ).newline()
    b.indent().indent().append("if not isinstance(value, str):").newline()
    b.indent().indent().indent().append("return None").newline()
    b.newline()
    b.indent().indent().append("member = object.__new__(cls)").newline()
    b.indent().indent().append("member._name_ = value").newline()
    b.indent().indent().append("member._value_ = value").newline()
    b.newline()
    b.indent().indent().append(
        "return cls._value2member_map_.setdefault(value, member)  # type:ignore"
    ).newline()


def _camel_to_upper_snake(value):
    s1 = re.sub("(.)([A-Z][a-z]+)", r"\1_\2", value)
    s1 = re.sub("([a-z0-9])([A-Z])", r"\1_\2", s1)
    # Non-identifier chars (e.g. "-" in "amazon-bedrock") become "_".
    return re.sub(r"[^0-9a-zA-Z]+", "_", s1).upper()
