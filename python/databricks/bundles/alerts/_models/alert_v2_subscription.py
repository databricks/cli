from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AlertV2Subscription:
    """"""

    destination_id: VariableOrOptional[str] = None

    user_email: VariableOrOptional[str] = None

    @classmethod
    def from_dict(cls, value: "AlertV2SubscriptionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AlertV2SubscriptionDict":
        return _transform_to_json_value(self)  # type:ignore


class AlertV2SubscriptionDict(TypedDict, total=False):
    """"""

    destination_id: VariableOrOptional[str]

    user_email: VariableOrOptional[str]


AlertV2SubscriptionParam = AlertV2SubscriptionDict | AlertV2Subscription
