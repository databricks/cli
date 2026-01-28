from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.model_serving_endpoints._models.auto_capture_config_input import (
    AutoCaptureConfigInput,
    AutoCaptureConfigInputParam,
)
from databricks.bundles.model_serving_endpoints._models.served_entity_input import (
    ServedEntityInput,
    ServedEntityInputParam,
)
from databricks.bundles.model_serving_endpoints._models.served_model_input import (
    ServedModelInput,
    ServedModelInputParam,
)
from databricks.bundles.model_serving_endpoints._models.traffic_config import (
    TrafficConfig,
    TrafficConfigParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class EndpointCoreConfigInput:
    """"""

    auto_capture_config: VariableOrOptional[AutoCaptureConfigInput] = None
    """
    Configuration for Inference Tables which automatically logs requests and responses to Unity Catalog.
    Note: this field is deprecated for creating new provisioned throughput endpoints,
    or updating existing provisioned throughput endpoints that never have inference table configured;
    in these cases please use AI Gateway to manage inference tables.
    """

    served_entities: VariableOrList[ServedEntityInput] = field(default_factory=list)
    """
    The list of served entities under the serving endpoint config.
    """

    served_models: VariableOrList[ServedModelInput] = field(default_factory=list)
    """
    (Deprecated, use served_entities instead) The list of served models under the serving endpoint config.
    """

    traffic_config: VariableOrOptional[TrafficConfig] = None
    """
    The traffic configuration associated with the serving endpoint config.
    """

    @classmethod
    def from_dict(cls, value: "EndpointCoreConfigInputDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "EndpointCoreConfigInputDict":
        return _transform_to_json_value(self)  # type:ignore


class EndpointCoreConfigInputDict(TypedDict, total=False):
    """"""

    auto_capture_config: VariableOrOptional[AutoCaptureConfigInputParam]
    """
    Configuration for Inference Tables which automatically logs requests and responses to Unity Catalog.
    Note: this field is deprecated for creating new provisioned throughput endpoints,
    or updating existing provisioned throughput endpoints that never have inference table configured;
    in these cases please use AI Gateway to manage inference tables.
    """

    served_entities: VariableOrList[ServedEntityInputParam]
    """
    The list of served entities under the serving endpoint config.
    """

    served_models: VariableOrList[ServedModelInputParam]
    """
    (Deprecated, use served_entities instead) The list of served models under the serving endpoint config.
    """

    traffic_config: VariableOrOptional[TrafficConfigParam]
    """
    The traffic configuration associated with the serving endpoint config.
    """


EndpointCoreConfigInputParam = EndpointCoreConfigInputDict | EndpointCoreConfigInput
