from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class NodeTypeFlexibility:
    """
    Configuration for flexible node types, allowing fallback to alternate node types during cluster launch and upscale.
    """

    alternate_node_type_ids: VariableOrList[str] = field(default_factory=list)
    """
    A list of node type IDs to use as fallbacks when the primary node type is unavailable.
    """

    @classmethod
    def from_dict(cls, value: "NodeTypeFlexibilityDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "NodeTypeFlexibilityDict":
        return _transform_to_json_value(self)  # type:ignore


class NodeTypeFlexibilityDict(TypedDict, total=False):
    """"""

    alternate_node_type_ids: VariableOrList[str]
    """
    A list of node type IDs to use as fallbacks when the primary node type is unavailable.
    """


NodeTypeFlexibilityParam = NodeTypeFlexibilityDict | NodeTypeFlexibility
