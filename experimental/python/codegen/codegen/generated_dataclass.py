from dataclasses import dataclass
from typing import Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from typing_extensions import Self

import codegen.packages as packages
from codegen.code_builder import CodeBuilder
from codegen.jsonschema import Property, Schema, Stage
from codegen.packages import is_resource


@dataclass
class GeneratedType:
    """
    GeneratedType is a type that can be used in GeneratedField.

    GeneratedType is self-recursive, so it can represent complex types like lists of dataclasses.
    """

    name: str
    """
    The name of the type, e.g., "Task"
    """

    package: Optional[str]
    """
    The package of the type, e.g., "databricks.bundles.jobs._models.task".

    If type is builtin, package is None.
    """

    parameters: list["Self"]
    """
    Parameters of the type, e.g., for list[str]:

        GeneratedType(
            name="list",
            parameters=[
                GeneratedType(name="str"),
            ],
        )

    """


@dataclass
class GeneratedField:
    """
    GeneratedField is a field in GeneratedDataclass.
    """

    field_name: str
    """
    The name of the field, e.g., "task_key"
    """

    type_name: GeneratedType
    """
    The type of the field, e.g., GeneratedType(name="Task", ...)
    """

    param_type_name: GeneratedType
    """
    The type of the field in TypedDict, e.g., GeneratedType(name="TaskParam", ...)
    """

    create_func_type_name: GeneratedType
    """
    Type type of the field in static "create" function, e.g., GeneratedType(name="TaskParam", ...)

    It can be different from param_type_name because lists are made optional in "create" function
    to avoid problems with mutable default arguments.
    """

    description: Optional[str]
    """
    The description of the field to be included into a docstring.
    """

    default: Optional[str]
    """
    The default value of the field, e.g., "None"
    """

    create_func_default: Optional[str]
    """
    The default value of the field in "create" function.

    It can be different from default because lists are made optional in "create" function
    to avoid problems with mutable default arguments.
    """

    default_factory: Optional[str]
    """
    Factory method for creating a default value, used for lists and dicts.
    """

    experimental: bool
    """
    If true, the field is experimental and should not be indexed in docs, and
    be marked as experimental in docstring.
    """

    def __post_init__(self):
        if self.default_factory is not None and self.default is not None:
            raise ValueError("Can't have both default and default_factory", self)


@dataclass
class GeneratedDataclass:
    """
    GeneratedDataclass represents a dataclass to be generated.
    """

    class_name: str
    """
    The name of the dataclass, e.g., "Task".
    """

    package: str
    """
    Package of the dataclass, e.g., "databricks.bundles.jobs._models.task".
    """

    description: Optional[str]
    """
    The description of the dataclass to be included into a docstring.
    """

    fields: list[GeneratedField]
    extends: list[GeneratedType]
    experimental: bool


def generate_field(
    field_name: str,
    prop: Property,
    is_required: bool,
) -> GeneratedField:
    field_type = generate_type(prop.ref, is_param=False)
    param_type = generate_type(prop.ref, is_param=True)

    field_type = variable_or_type(field_type, is_required=is_required)
    param_type = variable_or_type(param_type, is_required=is_required)

    if field_type.name == "VariableOrDict":
        return GeneratedField(
            field_name=field_name,
            type_name=field_type,
            param_type_name=param_type,
            create_func_type_name=optional_type(param_type),
            description=prop.description,
            default=None,
            default_factory="dict",
            create_func_default="None",
            experimental=prop.stage == Stage.PRIVATE,
        )
    elif field_type.name == "VariableOrList":
        return GeneratedField(
            field_name=field_name,
            type_name=field_type,
            param_type_name=param_type,
            create_func_type_name=optional_type(param_type),
            description=prop.description,
            default=None,
            default_factory="list",
            create_func_default="None",
            experimental=prop.stage == Stage.PRIVATE,
        )
    elif is_required:
        return GeneratedField(
            field_name=field_name,
            type_name=field_type,
            param_type_name=param_type,
            create_func_type_name=param_type,
            description=prop.description,
            default=None,
            default_factory=None,
            create_func_default=None,
            experimental=prop.stage == Stage.PRIVATE,
        )
    else:
        return GeneratedField(
            field_name=field_name,
            type_name=field_type,
            param_type_name=param_type,
            create_func_type_name=param_type,
            description=prop.description,
            default="None",
            default_factory=None,
            create_func_default="None",
            experimental=prop.stage == Stage.PRIVATE,
        )


def optional_type(generated: GeneratedType) -> GeneratedType:
    return GeneratedType(
        name="Optional",
        package="typing",
        parameters=[generated],
    )


def str_type() -> GeneratedType:
    return GeneratedType(
        name="str",
        package=None,
        parameters=[],
    )


def dict_type() -> GeneratedType:
    return GeneratedType(
        name="dict",
        package=None,
        parameters=[str_type(), str_type()],
    )


def variable_or_type(type: GeneratedType, is_required: bool) -> GeneratedType:
    if type.name == "list":
        [param] = type.parameters

        return variable_or_list_type(param)
    elif type.name == "dict":
        [key_param, value_param] = type.parameters

        assert key_param.name == "str"

        return variable_or_dict_type(value_param)
    else:
        name = "VariableOr" if is_required else "VariableOrOptional"

        return GeneratedType(
            name=name,
            package="databricks.bundles.core",
            parameters=[type],
        )


def variable_or_list_type(element_type: GeneratedType) -> GeneratedType:
    return GeneratedType(
        name="VariableOrList",
        package="databricks.bundles.core",
        parameters=[element_type],
    )


def variable_or_dict_type(element_type: GeneratedType) -> GeneratedType:
    return GeneratedType(
        name="VariableOrDict",
        package="databricks.bundles.core",
        parameters=[element_type],
    )


def generate_type(ref: str, is_param: bool) -> GeneratedType:
    if ref.startswith("#/$defs/slice/"):
        element_ref = ref.replace("#/$defs/slice/", "#/$defs/")
        element_type = generate_type(
            ref=element_ref,
            is_param=is_param,
        )

        return GeneratedType(
            name="list",
            package=None,
            parameters=[element_type],
        )

    if ref == "#/$defs/map/string":
        return dict_type()

    class_name = packages.get_class_name(ref)
    package = packages.get_package(ref)

    if is_param and package:
        class_name += "Param"

    return GeneratedType(
        name=class_name,
        package=package,
        parameters=[],
    )


def resource_type() -> GeneratedType:
    return GeneratedType(
        name="Resource",
        package="databricks.bundles.core",
        parameters=[],
    )


def generate_dataclass(schema_name: str, schema: Schema) -> GeneratedDataclass:
    print(f"Generating dataclass for {schema_name}")

    fields = list[GeneratedField]()
    class_name = packages.get_class_name(schema_name)

    for name, prop in schema.properties.items():
        is_required = name in schema.required
        field = generate_field(name, prop, is_required=is_required)

        fields.append(field)

    extends = []
    package = packages.get_package(schema_name)

    assert package

    if is_resource(schema_name):
        extends.append(resource_type())

    return GeneratedDataclass(
        class_name=class_name,
        package=package,
        description=schema.description,
        fields=fields,
        extends=extends,
        experimental=schema.stage == Stage.PRIVATE,
    )


def _get_type_code(generated: GeneratedType, quote: bool = True) -> str:
    if generated.parameters:
        parameters = ", ".join(
            map(lambda x: _get_type_code(x, quote), generated.parameters)
        )

        return f"{generated.name}[{parameters}]"
    else:
        if quote:
            return '"' + generated.name + '"'
        else:
            return generated.name


def _append_dataclass(b: CodeBuilder, generated: GeneratedDataclass):
    # Example:
    #
    # @dataclass
    # class Job(Resource):
    #     """docstring"""

    b.append("@dataclass(kw_only=True)")

    b.newline()
    b.append("class ", generated.class_name)

    if generated.extends:
        b.append("(")
        b.append_list(
            [_get_type_code(extend, quote=False) for extend in generated.extends]
        )
        b.append(")")

    b.append(":").newline()

    # FIXME should contain class docstring
    if not generated.description and not generated.experimental:
        b.indent().append_triple_quote().append_triple_quote().newline().newline()
    else:
        _append_description(b, generated.description, generated.experimental)


def _append_field(b: CodeBuilder, field: GeneratedField):
    # Example:
    #
    #     foo: list[str] = field(default_factory=list)

    b.indent().append(field.field_name).append(": ")

    # don't quote types because it breaks reflection
    b.append(_get_type_code(field.type_name, quote=False))

    if field.default_factory:
        b.append(" = field(")
        b.append_dict({"default_factory": field.default_factory})
        b.append(")")
    elif field.default:
        b.append(" = ")
        b.append(field.default)

    b.newline()


def _append_from_dict(b: CodeBuilder, generated: GeneratedDataclass):
    # Example:
    #
    #   @classmethod
    #   def from_dict(cls, value: 'JobDict') -> 'Job':
    #       return _transform(cls, value)

    b.indent().append("@classmethod").newline()

    (
        b.indent()
        .append("def from_dict(cls, value: ")
        .append("'")
        .append(generated.class_name + "Dict")
        .append("'")
        .append(") -> 'Self':")
        .newline()
    )

    b.indent().indent().append("return _transform(cls, value)").newline()
    b.newline()


def _append_as_dict(b: CodeBuilder, generated: GeneratedDataclass):
    # Example:
    #
    #   def as_dict(self) -> 'JobDict':
    #       return _transform_to_json_value(self) # type:ignore
    #

    b.indent().append("def as_dict(self) -> '").append(generated.class_name).append(
        "Dict':"
    ).newline()
    b.indent().indent().append(
        "return _transform_to_json_value(self) # type:ignore",
    ).newline()
    b.newline()


def _append_typed_dict(b: CodeBuilder, generated: GeneratedDataclass):
    # Example:
    #
    # class JobDict(TypedDict, total=False):
    #     """docstring"""
    #

    b.append("class ").append(generated.class_name).append(
        "Dict(TypedDict, total=False):"
    ).newline()

    # FIXME should contain class description
    b.indent().append_triple_quote().append_triple_quote().newline().newline()


def _append_description(b: CodeBuilder, description: Optional[str], experimental: bool):
    if description or experimental:
        b.indent().append_triple_quote().newline()
        if experimental:
            b.indent().append(":meta private: [EXPERIMENTAL]").newline()
            if description:
                b.indent().newline()
        if description:
            for line in description.split("\n"):
                b.indent().append(line).newline()
        b.indent().append_triple_quote().newline()


def _append_typed_dict_field(b: CodeBuilder, field: GeneratedField):
    b.indent().append(field.field_name).append(": ")
    b.append(_get_type_code(field.param_type_name, quote=False))
    b.newline()


def get_code(generated: GeneratedDataclass) -> str:
    b = CodeBuilder()

    _append_dataclass(b, generated)

    for field in generated.fields:
        _append_field(b, field)
        _append_description(b, field.description, field.experimental)

        b.newline()

    _append_from_dict(b, generated)
    _append_as_dict(b, generated)

    b.newline().newline()

    _append_typed_dict(b, generated)

    for field in generated.fields:
        _append_typed_dict_field(b, field)
        _append_description(b, field.description, field.experimental)

        b.newline()

    # Example: FooParam = FooDict | Foo

    b.newline()
    b.append(generated.class_name).append("Param")
    b.append(" = ")
    b.append(generated.class_name).append("Dict")
    b.append(" | ")
    b.append(generated.class_name)
    b.newline()

    return b.build()
