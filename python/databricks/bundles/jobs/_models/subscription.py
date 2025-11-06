from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.jobs._models.subscription_subscriber import (
    SubscriptionSubscriber,
    SubscriptionSubscriberParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Subscription:
    """"""

    custom_subject: VariableOrOptional[str] = None
    """
    Optional: Allows users to specify a custom subject line on the email sent
    to subscribers.
    """

    paused: VariableOrOptional[bool] = None
    """
    When true, the subscription will not send emails.
    """

    subscribers: VariableOrList[SubscriptionSubscriber] = field(default_factory=list)

    @classmethod
    def from_dict(cls, value: "SubscriptionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SubscriptionDict":
        return _transform_to_json_value(self)  # type:ignore


class SubscriptionDict(TypedDict, total=False):
    """"""

    custom_subject: VariableOrOptional[str]
    """
    Optional: Allows users to specify a custom subject line on the email sent
    to subscribers.
    """

    paused: VariableOrOptional[bool]
    """
    When true, the subscription will not send emails.
    """

    subscribers: VariableOrList[SubscriptionSubscriberParam]


SubscriptionParam = SubscriptionDict | Subscription
