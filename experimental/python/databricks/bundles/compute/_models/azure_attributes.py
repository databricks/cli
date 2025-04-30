from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.azure_availability import (
    AzureAvailability,
    AzureAvailabilityParam,
)
from databricks.bundles.compute._models.log_analytics_info import (
    LogAnalyticsInfo,
    LogAnalyticsInfoParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AzureAttributes:
    """
    Attributes set during cluster creation which are related to Microsoft Azure.
    """

    availability: VariableOrOptional[AzureAvailability] = None

    first_on_demand: VariableOrOptional[int] = None
    """
    The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
    This value should be greater than 0, to make sure the cluster driver node is placed on an
    on-demand instance. If this value is greater than or equal to the current cluster size, all
    nodes will be placed on on-demand instances. If this value is less than the current cluster
    size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
    be placed on `availability` instances. Note that this value does not affect
    cluster size and cannot currently be mutated over the lifetime of a cluster.
    """

    log_analytics_info: VariableOrOptional[LogAnalyticsInfo] = None
    """
    Defines values necessary to configure and run Azure Log Analytics agent
    """

    spot_bid_max_price: VariableOrOptional[float] = None
    """
    The max bid price to be used for Azure spot instances.
    The Max price for the bid cannot be higher than the on-demand price of the instance.
    If not specified, the default value is -1, which specifies that the instance cannot be evicted
    on the basis of price, and only on the basis of availability. Further, the value should > 0 or -1.
    """

    @classmethod
    def from_dict(cls, value: "AzureAttributesDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AzureAttributesDict":
        return _transform_to_json_value(self)  # type:ignore


class AzureAttributesDict(TypedDict, total=False):
    """"""

    availability: VariableOrOptional[AzureAvailabilityParam]

    first_on_demand: VariableOrOptional[int]
    """
    The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
    This value should be greater than 0, to make sure the cluster driver node is placed on an
    on-demand instance. If this value is greater than or equal to the current cluster size, all
    nodes will be placed on on-demand instances. If this value is less than the current cluster
    size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
    be placed on `availability` instances. Note that this value does not affect
    cluster size and cannot currently be mutated over the lifetime of a cluster.
    """

    log_analytics_info: VariableOrOptional[LogAnalyticsInfoParam]
    """
    Defines values necessary to configure and run Azure Log Analytics agent
    """

    spot_bid_max_price: VariableOrOptional[float]
    """
    The max bid price to be used for Azure spot instances.
    The Max price for the bid cannot be higher than the on-demand price of the instance.
    If not specified, the default value is -1, which specifies that the instance cannot be evicted
    on the basis of price, and only on the basis of availability. Further, the value should > 0 or -1.
    """


AzureAttributesParam = AzureAttributesDict | AzureAttributes
