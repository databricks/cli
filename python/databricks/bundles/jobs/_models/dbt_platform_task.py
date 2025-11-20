from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DbtPlatformTask:
    """
    :meta private: [EXPERIMENTAL]
    """

    connection_resource_name: VariableOrOptional[str] = None
    """
    The resource name of the UC connection that authenticates the dbt platform for this task
    """

    dbt_platform_job_id: VariableOrOptional[str] = None
    """
    Id of the dbt platform job to be triggered. Specified as a string for maximum compatibility with clients.
    """

    @classmethod
    def from_dict(cls, value: "DbtPlatformTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DbtPlatformTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class DbtPlatformTaskDict(TypedDict, total=False):
    """"""

    connection_resource_name: VariableOrOptional[str]
    """
    The resource name of the UC connection that authenticates the dbt platform for this task
    """

    dbt_platform_job_id: VariableOrOptional[str]
    """
    Id of the dbt platform job to be triggered. Specified as a string for maximum compatibility with clients.
    """


DbtPlatformTaskParam = DbtPlatformTaskDict | DbtPlatformTask
