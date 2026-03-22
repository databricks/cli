from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.jobs._models.alert_task_subscriber import (
    AlertTaskSubscriber,
    AlertTaskSubscriberParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AlertTask:
    """"""

    alert_id: VariableOrOptional[str] = None
    """
    The alert_id is the canonical identifier of the alert.
    """

    subscribers: VariableOrList[AlertTaskSubscriber] = field(default_factory=list)
    """
    The subscribers receive alert evaluation result notifications after the alert task is completed.
    The number of subscriptions is limited to 100.
    """

    warehouse_id: VariableOrOptional[str] = None
    """
    The warehouse_id identifies the warehouse settings used by the alert task.
    """

    workspace_path: VariableOrOptional[str] = None
    """
    The workspace_path is the path to the alert file in the workspace. The path:
    * must start with "/Workspace"
    * must be a normalized path.
    User has to select only one of alert_id or workspace_path to identify the alert.
    """

    @classmethod
    def from_dict(cls, value: "AlertTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AlertTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class AlertTaskDict(TypedDict, total=False):
    """"""

    alert_id: VariableOrOptional[str]
    """
    The alert_id is the canonical identifier of the alert.
    """

    subscribers: VariableOrList[AlertTaskSubscriberParam]
    """
    The subscribers receive alert evaluation result notifications after the alert task is completed.
    The number of subscriptions is limited to 100.
    """

    warehouse_id: VariableOrOptional[str]
    """
    The warehouse_id identifies the warehouse settings used by the alert task.
    """

    workspace_path: VariableOrOptional[str]
    """
    The workspace_path is the path to the alert file in the workspace. The path:
    * must start with "/Workspace"
    * must be a normalized path.
    User has to select only one of alert_id or workspace_path to identify the alert.
    """


AlertTaskParam = AlertTaskDict | AlertTask
