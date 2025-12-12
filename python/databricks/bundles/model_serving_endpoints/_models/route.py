from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Route:
    """"""

    traffic_percentage: VariableOr[int]
    """
    The percentage of endpoint traffic to send to this route. It must be an integer between 0 and 100 inclusive.
    """

    served_entity_name: VariableOrOptional[str] = None

    served_model_name: VariableOrOptional[str] = None
    """
    The name of the served model this route configures traffic for.
    """

    @classmethod
    def from_dict(cls, value: "RouteDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "RouteDict":
        return _transform_to_json_value(self)  # type:ignore


class RouteDict(TypedDict, total=False):
    """"""

    traffic_percentage: VariableOr[int]
    """
    The percentage of endpoint traffic to send to this route. It must be an integer between 0 and 100 inclusive.
    """

    served_entity_name: VariableOrOptional[str]

    served_model_name: VariableOrOptional[str]
    """
    The name of the served model this route configures traffic for.
    """


RouteParam = RouteDict | Route
