from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AlertTaskSubscriber:
    """
    Represents a subscriber that will receive alert notifications.
    A subscriber can be either a user (via email) or a notification destination (via destination_id).
    """

    destination_id: VariableOrOptional[str] = None

    user_name: VariableOrOptional[str] = None
    """
    A valid workspace email address.
    """

    @classmethod
    def from_dict(cls, value: "AlertTaskSubscriberDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AlertTaskSubscriberDict":
        return _transform_to_json_value(self)  # type:ignore


class AlertTaskSubscriberDict(TypedDict, total=False):
    """"""

    destination_id: VariableOrOptional[str]

    user_name: VariableOrOptional[str]
    """
    A valid workspace email address.
    """


AlertTaskSubscriberParam = AlertTaskSubscriberDict | AlertTaskSubscriber
