from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SqlTaskSubscription:
    """"""

    destination_id: VariableOrOptional[str] = None
    """
    The canonical identifier of the destination to receive email notification. This parameter is mutually exclusive with user_name. You cannot set both destination_id and user_name for subscription notifications.
    """

    user_name: VariableOrOptional[str] = None
    """
    The user name to receive the subscription email. This parameter is mutually exclusive with destination_id. You cannot set both destination_id and user_name for subscription notifications.
    """

    @classmethod
    def from_dict(cls, value: "SqlTaskSubscriptionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SqlTaskSubscriptionDict":
        return _transform_to_json_value(self)  # type:ignore


class SqlTaskSubscriptionDict(TypedDict, total=False):
    """"""

    destination_id: VariableOrOptional[str]
    """
    The canonical identifier of the destination to receive email notification. This parameter is mutually exclusive with user_name. You cannot set both destination_id and user_name for subscription notifications.
    """

    user_name: VariableOrOptional[str]
    """
    The user name to receive the subscription email. This parameter is mutually exclusive with destination_id. You cannot set both destination_id and user_name for subscription notifications.
    """


SqlTaskSubscriptionParam = SqlTaskSubscriptionDict | SqlTaskSubscription
