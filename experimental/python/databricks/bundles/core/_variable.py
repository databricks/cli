import typing
from dataclasses import dataclass, fields, is_dataclass
from typing import (
    Generic,
    Optional,
    Type,
    TypeVar,
    Union,
)

__all__ = [
    "Variable",
    "VariableOr",
    "VariableOrOptional",
    "VariableOrDict",
    "VariableOrList",
    "variables",
]

_T = TypeVar("_T", covariant=True)


@dataclass(kw_only=True)
class Variable(Generic[_T]):
    """
    Reference to a bundle variable.

    See: `Databricks Asset Bundles configuration <https://docs.databricks.com/en/dev-tools/bundles/settings.html>`_
    """

    path: str
    """
    Path to the variable, e.g. "var.my_variable".
    """

    type: Type[_T]
    """
    Type of the variable.
    """

    def __str__(self):
        return self.value

    def __repr__(self):
        return self.value

    @property
    def value(self) -> str:
        """
        Returns the variable path in the format of "${path}"
        """

        return "${" + self.path + "}"


VariableOr = Union[Variable[_T], _T]
VariableOrOptional = Union[Variable[_T], Optional[_T]]

# - 1. variable: ${var.my_list}
# - 2. regular list: [{"name": "abc"}, ...]
# - 3. list of variables: ["${var.my_item}", ...]
# - 4. list with a mix of (3) and (4) as elements
VariableOrList = VariableOr[list[VariableOr[_T]]]

# - 1. variable: ${var.my_list}
# - 2. regular dict: {"key": "value", ...}
# - 3. dict with variables (but not as keys): {"value": "${var.my_item}", ...}
# - 4. dict with a mix of (3) and (4) as values
VariableOrDict = VariableOr[dict[str, VariableOr[_T]]]


def variables(cls: type[_T]) -> type[_T]:
    """
    A decorator that initializes each annotated attribute in a class
    with :class:`~Variable` type. Variables are initialized with a path
    that corresponds to the attribute name. Variables should specify their
    type, or else they will be treated as :class:`~Any`. Complex types
    like data classes, lists or dictionaries are supported.

    For example, if your databricks.yml file contains:

    .. code-block:: yaml

        variables:
          warehouse_id:
            description: Warehouse ID for SQL tasks
            default: ...

    You can define a class with a `warehouse_id` attribute:

    .. code-block:: python

        @variables
        class MyVariables:
          warehouse_id: Variable[str] # ${var.warehouse_id}

    And later use it in your code as `MyVariables.warehouse_id`.

    For accessing bundle variable values, see :meth:`Bundle.resolve_variable`.
    """

    # making class a dataclass, solves a lot of problems, because we can just use 'fields'
    cls = dataclass(cls)
    assert is_dataclass(cls)

    # don't get type hints unless needed
    hints = None

    for field in fields(cls):
        field_type = field.type

        if isinstance(field_type, typing.ForwardRef) or isinstance(field_type, str):
            if hints is None:
                hints = typing.get_type_hints(cls)

            field_type = hints.get(field.name, field.type)

        origin = typing.get_origin(field_type) or field_type

        if origin != Variable:
            raise ValueError(
                f"Only 'Variable' type is allowed in classes annotated with @variables, got {field_type}"
            )

        args = typing.get_args(field_type)

        if not args:
            raise ValueError(
                f"Variable type must be specified for '{field.name}', e.g. Variable[str]"
            )
        else:
            variable_type = args[0]

        variable = Variable(path=f"var.{field.name}", type=variable_type)

        setattr(cls, field.name, variable)

    return cls
