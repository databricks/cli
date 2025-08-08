from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Grant:
    """"""

    principal: VariableOr[str]
    """
    The name of the principal that will be granted privileges
    """

    privileges: VariableOrList[str] = field(default_factory=list)
    """
    The privileges to grant to the specified entity
    """

    @classmethod
    def from_dict(cls, value: "GrantDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "GrantDict":
        return _transform_to_json_value(self)  # type:ignore


class GrantDict(TypedDict, total=False):
    """"""

    principal: VariableOr[str]
    """
    The name of the principal that will be granted privileges
    """

    privileges: VariableOrList[str]
    """
    The privileges to grant to the specified entity
    """


GrantParam = GrantDict | Grant
