from dataclasses import dataclass, field
from typing import Any, TypeVar, Union, get_origin

from databricks.bundles.core._variable import Variable, VariableOr, VariableOrList

__all__ = [
    "Bundle",
]

_T = TypeVar("_T")

_VAR_PREFIX = "var"


@dataclass(frozen=True, kw_only=True)
class Bundle:
    """
    Bundle contains information about a bundle accessible in functions
    loading and mutating resources.
    """

    target: str
    """
    Selected target where the bundle is being loaded. E.g.: 'development', 'staging', or 'production'.
    """

    variables: dict[str, Any] = field(default_factory=dict)
    """
    Values of bundle variables resolved for selected target. Bundle variables are defined in databricks.yml.
    For accessing variables as structured data, use :meth:`resolve_variable`.

    Example:

    .. code-block:: yaml

        variables:
          default_dbr_version:
            description: Default version of Databricks Runtime
            default: "14.3.x-scala2.12"
    """

    def resolve_variable(self, variable: VariableOr[_T]) -> _T:
        """
        Resolve a variable to its value.

        If the value is a variable, it will be resolved and returned.
        Otherwise, the value will be returned as is.
        """
        if not isinstance(variable, Variable):
            return variable

        if not variable.path.startswith(_VAR_PREFIX + "."):
            raise ValueError(
                "You can only get values of variables starting with 'var.*'"
            )
        else:
            variable_name = variable.path[len(_VAR_PREFIX + ".") :]

        if variable_name not in self.variables:
            raise ValueError(
                f"Can't find '{variable_name}' variable. Did you define it in databricks.yml?"
            )

        value = self.variables.get(variable_name)

        # avoid circular import
        from databricks.bundles.core._transform import (
            _display_type,
            _find_union_arg,
            _transform,
            _unwrap_variable_path,
        )

        if nested := _unwrap_variable_path(value):
            can_be_variable = get_origin(variable.type) == Union and _find_union_arg(
                nested, variable.type
            )
            can_be_variable = can_be_variable or get_origin(variable.type) == Variable

            if not can_be_variable:
                display_type = _display_type(variable.type)

                raise ValueError(
                    f"Failed to resolve '{variable_name}' because refers to another "
                    f"variable '{nested}'. Change variable type to "
                    f"Variable[VariableOr[{display_type}]]"
                )

        try:
            return _transform(variable.type, value)
        except Exception as e:
            raise ValueError(f"Failed to read '{variable_name}' variable value") from e

    def resolve_variable_list(self, variable: VariableOrList[_T]) -> list[_T]:
        """
        Resolve a list variable to its value.

        If the value is a variable, or the list item is a variable, it will be resolved and returned.
        Otherwise, the value will be returned as is.
        """

        return [self.resolve_variable(item) for item in self.resolve_variable(variable)]
