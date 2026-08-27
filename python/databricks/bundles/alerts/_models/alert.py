from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.alerts._models.alert_statement_parameter import (
    AlertStatementParameter,
    AlertStatementParameterParam,
)
from databricks.bundles.alerts._models.alert_v2_evaluation import (
    AlertV2Evaluation,
    AlertV2EvaluationParam,
)
from databricks.bundles.alerts._models.alert_v2_run_as import (
    AlertV2RunAs,
    AlertV2RunAsParam,
)
from databricks.bundles.alerts._models.cron_schedule import (
    CronSchedule,
    CronScheduleParam,
)
from databricks.bundles.alerts._models.lifecycle import Lifecycle, LifecycleParam
from databricks.bundles.alerts._models.permission import Permission, PermissionParam
from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Alert(Resource):
    """"""

    display_name: VariableOr[str]
    """
    The display name of the alert.
    """

    evaluation: VariableOr[AlertV2Evaluation]

    query_text: VariableOr[str]
    """
    Text of the query to be run.
    """

    schedule: VariableOr[CronSchedule]

    warehouse_id: VariableOr[str]
    """
    ID of the SQL warehouse attached to the alert.
    """

    custom_description: VariableOrOptional[str] = None
    """
    Custom description for the alert. support mustache template.
    """

    custom_summary: VariableOrOptional[str] = None
    """
    Custom summary for the alert. support mustache template.
    """

    file_path: VariableOrOptional[str] = None

    lifecycle: VariableOrOptional[Lifecycle] = None
    """
    Settings that control the deployment lifecycle of the resource, such as preventing it from being destroyed.
    """

    parameters: VariableOrList[AlertStatementParameter] = field(default_factory=list)
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Query parameters bound when executing the alert query, referenced in the
    query text with `:name` syntax. Static values only.
    """

    parent_path: VariableOrOptional[str] = None
    """
    The workspace path of the folder containing the alert. Can only be set on create, and cannot be updated.
    """

    permissions: VariableOrList[Permission] = field(default_factory=list)
    """
    The permissions to apply to this resource.
    """

    run_as: VariableOrOptional[AlertV2RunAs] = None
    """
    Specifies the identity that will be used to run the alert.
    This field allows you to configure alerts to run as a specific user or service principal.
    - For user identity: Set `user_name` to the email of an active workspace user. Users can only set this to their own email.
    - For service principal: Set `service_principal_name` to the application ID. Requires the `servicePrincipal/user` role.
    If not specified, the alert will run as the request user.
    """

    run_as_user_name: VariableOrOptional[str] = None
    """
    [DEPRECATED] The run as username or application ID of service principal.
    On Create and Update, this field can be set to application ID of an active service principal. Setting this field requires the servicePrincipal/user role.
    Deprecated: Use `run_as` field instead. This field will be removed in a future release.
    """

    @classmethod
    def from_dict(cls, value: "AlertDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AlertDict":
        return _transform_to_json_value(self)  # type:ignore


class AlertDict(TypedDict, total=False):
    """"""

    display_name: VariableOr[str]
    """
    The display name of the alert.
    """

    evaluation: VariableOr[AlertV2EvaluationParam]

    query_text: VariableOr[str]
    """
    Text of the query to be run.
    """

    schedule: VariableOr[CronScheduleParam]

    warehouse_id: VariableOr[str]
    """
    ID of the SQL warehouse attached to the alert.
    """

    custom_description: VariableOrOptional[str]
    """
    Custom description for the alert. support mustache template.
    """

    custom_summary: VariableOrOptional[str]
    """
    Custom summary for the alert. support mustache template.
    """

    file_path: VariableOrOptional[str]

    lifecycle: VariableOrOptional[LifecycleParam]
    """
    Settings that control the deployment lifecycle of the resource, such as preventing it from being destroyed.
    """

    parameters: VariableOrList[AlertStatementParameterParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Query parameters bound when executing the alert query, referenced in the
    query text with `:name` syntax. Static values only.
    """

    parent_path: VariableOrOptional[str]
    """
    The workspace path of the folder containing the alert. Can only be set on create, and cannot be updated.
    """

    permissions: VariableOrList[PermissionParam]
    """
    The permissions to apply to this resource.
    """

    run_as: VariableOrOptional[AlertV2RunAsParam]
    """
    Specifies the identity that will be used to run the alert.
    This field allows you to configure alerts to run as a specific user or service principal.
    - For user identity: Set `user_name` to the email of an active workspace user. Users can only set this to their own email.
    - For service principal: Set `service_principal_name` to the application ID. Requires the `servicePrincipal/user` role.
    If not specified, the alert will run as the request user.
    """

    run_as_user_name: VariableOrOptional[str]
    """
    [DEPRECATED] The run as username or application ID of service principal.
    On Create and Update, this field can be set to application ID of an active service principal. Setting this field requires the servicePrincipal/user role.
    Deprecated: Use `run_as` field instead. This field will be removed in a future release.
    """


AlertParam = AlertDict | Alert
