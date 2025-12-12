from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class EmailNotifications:
    """"""

    on_update_failure: VariableOrList[str] = field(default_factory=list)
    """
    A list of email addresses to be notified when an endpoint fails to update its configuration or state.
    """

    on_update_success: VariableOrList[str] = field(default_factory=list)
    """
    A list of email addresses to be notified when an endpoint successfully updates its configuration or state.
    """

    @classmethod
    def from_dict(cls, value: "EmailNotificationsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "EmailNotificationsDict":
        return _transform_to_json_value(self)  # type:ignore


class EmailNotificationsDict(TypedDict, total=False):
    """"""

    on_update_failure: VariableOrList[str]
    """
    A list of email addresses to be notified when an endpoint fails to update its configuration or state.
    """

    on_update_success: VariableOrList[str]
    """
    A list of email addresses to be notified when an endpoint successfully updates its configuration or state.
    """


EmailNotificationsParam = EmailNotificationsDict | EmailNotifications
