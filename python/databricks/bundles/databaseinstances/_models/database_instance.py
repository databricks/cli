from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.databaseinstances._models.custom_tag import (
    CustomTag,
    CustomTagParam,
)
from databricks.bundles.databaseinstances._models.database_instance_permission import (
    DatabaseInstancePermission,
    DatabaseInstancePermissionParam,
)
from databricks.bundles.databaseinstances._models.database_instance_ref import (
    DatabaseInstanceRef,
    DatabaseInstanceRefParam,
)
from databricks.bundles.databaseinstances._models.database_instance_state import (
    DatabaseInstanceState,
    DatabaseInstanceStateParam,
)
from databricks.bundles.databaseinstances._models.lifecycle import (
    Lifecycle,
    LifecycleParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DatabaseInstance(Resource):
    """
    A DatabaseInstance represents a logical Postgres instance, comprised of both compute and storage.
    """

    name: VariableOr[str]
    """
    The name of the instance. This is the unique identifier for the instance.
    """

    capacity: VariableOrOptional[str] = None
    """
    The sku of the instance. Valid values are "CU_1", "CU_2", "CU_4", "CU_8".
    """

    child_instance_refs: VariableOrList[DatabaseInstanceRef] = field(
        default_factory=list
    )
    """
    The refs of the child instances. This is only available if the instance is
    parent instance.
    """

    creation_time: VariableOrOptional[str] = None
    """
    The timestamp when the instance was created.
    """

    creator: VariableOrOptional[str] = None
    """
    The email of the creator of the instance.
    """

    custom_tags: VariableOrList[CustomTag] = field(default_factory=list)
    """
    Custom tags associated with the instance. This field is only included on create and update responses.
    """

    effective_capacity: VariableOrOptional[str] = None
    """
    [DEPRECATED] 
    """

    effective_custom_tags: VariableOrList[CustomTag] = field(default_factory=list)
    """
    The recorded custom tags associated with the instance.
    """

    effective_enable_pg_native_login: VariableOrOptional[bool] = None
    """
    Whether the instance has PG native password login enabled.
    """

    effective_enable_readable_secondaries: VariableOrOptional[bool] = None
    """
    Whether secondaries serving read-only traffic are enabled. Defaults to false.
    """

    effective_node_count: VariableOrOptional[int] = None
    """
    The number of nodes in the instance, composed of 1 primary and 0 or more secondaries. Defaults to
    1 primary and 0 secondaries.
    """

    effective_retention_window_in_days: VariableOrOptional[int] = None
    """
    The retention window for the instance. This is the time window in days
    for which the historical data is retained.
    """

    effective_stopped: VariableOrOptional[bool] = None
    """
    Whether the instance is stopped.
    """

    effective_usage_policy_id: VariableOrOptional[str] = None
    """
    The policy that is applied to the instance.
    """

    enable_pg_native_login: VariableOrOptional[bool] = None
    """
    Whether to enable PG native password login on the instance. Defaults to false.
    """

    enable_readable_secondaries: VariableOrOptional[bool] = None
    """
    Whether to enable secondaries to serve read-only traffic. Defaults to false.
    """

    lifecycle: VariableOrOptional[Lifecycle] = None
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    node_count: VariableOrOptional[int] = None
    """
    The number of nodes in the instance, composed of 1 primary and 0 or more secondaries. Defaults to
    1 primary and 0 secondaries. This field is input only, see effective_node_count for the output.
    """

    parent_instance_ref: VariableOrOptional[DatabaseInstanceRef] = None
    """
    The ref of the parent instance. This is only available if the instance is
    child instance.
    Input: For specifying the parent instance to create a child instance. Optional.
    Output: Only populated if provided as input to create a child instance.
    """

    permissions: VariableOrList[DatabaseInstancePermission] = field(
        default_factory=list
    )

    pg_version: VariableOrOptional[str] = None
    """
    The version of Postgres running on the instance.
    """

    read_only_dns: VariableOrOptional[str] = None
    """
    The DNS endpoint to connect to the instance for read only access. This is only available if
    enable_readable_secondaries is true.
    """

    read_write_dns: VariableOrOptional[str] = None
    """
    The DNS endpoint to connect to the instance for read+write access.
    """

    retention_window_in_days: VariableOrOptional[int] = None
    """
    The retention window for the instance. This is the time window in days
    for which the historical data is retained. The default value is 7 days.
    Valid values are 2 to 35 days.
    """

    state: VariableOrOptional[DatabaseInstanceState] = None
    """
    The current state of the instance.
    """

    stopped: VariableOrOptional[bool] = None
    """
    Whether to stop the instance. An input only param, see effective_stopped for the output.
    """

    uid: VariableOrOptional[str] = None
    """
    An immutable UUID identifier for the instance.
    """

    usage_policy_id: VariableOrOptional[str] = None
    """
    The desired usage policy to associate with the instance.
    """

    @classmethod
    def from_dict(cls, value: "DatabaseInstanceDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DatabaseInstanceDict":
        return _transform_to_json_value(self)  # type:ignore


class DatabaseInstanceDict(TypedDict, total=False):
    """"""

    name: VariableOr[str]
    """
    The name of the instance. This is the unique identifier for the instance.
    """

    capacity: VariableOrOptional[str]
    """
    The sku of the instance. Valid values are "CU_1", "CU_2", "CU_4", "CU_8".
    """

    child_instance_refs: VariableOrList[DatabaseInstanceRefParam]
    """
    The refs of the child instances. This is only available if the instance is
    parent instance.
    """

    creation_time: VariableOrOptional[str]
    """
    The timestamp when the instance was created.
    """

    creator: VariableOrOptional[str]
    """
    The email of the creator of the instance.
    """

    custom_tags: VariableOrList[CustomTagParam]
    """
    Custom tags associated with the instance. This field is only included on create and update responses.
    """

    effective_capacity: VariableOrOptional[str]
    """
    [DEPRECATED] 
    """

    effective_custom_tags: VariableOrList[CustomTagParam]
    """
    The recorded custom tags associated with the instance.
    """

    effective_enable_pg_native_login: VariableOrOptional[bool]
    """
    Whether the instance has PG native password login enabled.
    """

    effective_enable_readable_secondaries: VariableOrOptional[bool]
    """
    Whether secondaries serving read-only traffic are enabled. Defaults to false.
    """

    effective_node_count: VariableOrOptional[int]
    """
    The number of nodes in the instance, composed of 1 primary and 0 or more secondaries. Defaults to
    1 primary and 0 secondaries.
    """

    effective_retention_window_in_days: VariableOrOptional[int]
    """
    The retention window for the instance. This is the time window in days
    for which the historical data is retained.
    """

    effective_stopped: VariableOrOptional[bool]
    """
    Whether the instance is stopped.
    """

    effective_usage_policy_id: VariableOrOptional[str]
    """
    The policy that is applied to the instance.
    """

    enable_pg_native_login: VariableOrOptional[bool]
    """
    Whether to enable PG native password login on the instance. Defaults to false.
    """

    enable_readable_secondaries: VariableOrOptional[bool]
    """
    Whether to enable secondaries to serve read-only traffic. Defaults to false.
    """

    lifecycle: VariableOrOptional[LifecycleParam]
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    node_count: VariableOrOptional[int]
    """
    The number of nodes in the instance, composed of 1 primary and 0 or more secondaries. Defaults to
    1 primary and 0 secondaries. This field is input only, see effective_node_count for the output.
    """

    parent_instance_ref: VariableOrOptional[DatabaseInstanceRefParam]
    """
    The ref of the parent instance. This is only available if the instance is
    child instance.
    Input: For specifying the parent instance to create a child instance. Optional.
    Output: Only populated if provided as input to create a child instance.
    """

    permissions: VariableOrList[DatabaseInstancePermissionParam]

    pg_version: VariableOrOptional[str]
    """
    The version of Postgres running on the instance.
    """

    read_only_dns: VariableOrOptional[str]
    """
    The DNS endpoint to connect to the instance for read only access. This is only available if
    enable_readable_secondaries is true.
    """

    read_write_dns: VariableOrOptional[str]
    """
    The DNS endpoint to connect to the instance for read+write access.
    """

    retention_window_in_days: VariableOrOptional[int]
    """
    The retention window for the instance. This is the time window in days
    for which the historical data is retained. The default value is 7 days.
    Valid values are 2 to 35 days.
    """

    state: VariableOrOptional[DatabaseInstanceStateParam]
    """
    The current state of the instance.
    """

    stopped: VariableOrOptional[bool]
    """
    Whether to stop the instance. An input only param, see effective_stopped for the output.
    """

    uid: VariableOrOptional[str]
    """
    An immutable UUID identifier for the instance.
    """

    usage_policy_id: VariableOrOptional[str]
    """
    The desired usage policy to associate with the instance.
    """


DatabaseInstanceParam = DatabaseInstanceDict | DatabaseInstance
