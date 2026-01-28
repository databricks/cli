from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList
from databricks.bundles.model_serving_endpoints._models.route import (
    Route,
    RouteParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class TrafficConfig:
    """"""

    routes: VariableOrList[Route] = field(default_factory=list)
    """
    The list of routes that define traffic to each served entity.
    """

    @classmethod
    def from_dict(cls, value: "TrafficConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TrafficConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class TrafficConfigDict(TypedDict, total=False):
    """"""

    routes: VariableOrList[RouteParam]
    """
    The list of routes that define traffic to each served entity.
    """


TrafficConfigParam = TrafficConfigDict | TrafficConfig
