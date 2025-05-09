from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SubscriptionSubscriber:
    """"""

    destination_id: VariableOrOptional[str] = None

    user_name: VariableOrOptional[str] = None

    @classmethod
    def from_dict(cls, value: "SubscriptionSubscriberDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SubscriptionSubscriberDict":
        return _transform_to_json_value(self)  # type:ignore


class SubscriptionSubscriberDict(TypedDict, total=False):
    """"""

    destination_id: VariableOrOptional[str]

    user_name: VariableOrOptional[str]


SubscriptionSubscriberParam = SubscriptionSubscriberDict | SubscriptionSubscriber
