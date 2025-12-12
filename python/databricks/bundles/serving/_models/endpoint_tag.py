from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class EndpointTag:
    """"""

    key: VariableOr[str]
    """
    Key field for a serving endpoint tag.
    """

    value: VariableOrOptional[str] = None
    """
    Optional value field for a serving endpoint tag.
    """

    @classmethod
    def from_dict(cls, value: "EndpointTagDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "EndpointTagDict":
        return _transform_to_json_value(self)  # type:ignore


class EndpointTagDict(TypedDict, total=False):
    """"""

    key: VariableOr[str]
    """
    Key field for a serving endpoint tag.
    """

    value: VariableOrOptional[str]
    """
    Optional value field for a serving endpoint tag.
    """


EndpointTagParam = EndpointTagDict | EndpointTag
