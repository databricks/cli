from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DbtCloudTask:
    """
    :meta private: [EXPERIMENTAL]
    """

    connection_resource_name: VariableOrOptional[str] = None
    """
    The resource name of the UC connection that authenticates the dbt Cloud for this task
    """

    dbt_cloud_job_id: VariableOrOptional[int] = None
    """
    Id of the dbt Cloud job to be triggered
    """

    @classmethod
    def from_dict(cls, value: "DbtCloudTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DbtCloudTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class DbtCloudTaskDict(TypedDict, total=False):
    """"""

    connection_resource_name: VariableOrOptional[str]
    """
    The resource name of the UC connection that authenticates the dbt Cloud for this task
    """

    dbt_cloud_job_id: VariableOrOptional[int]
    """
    Id of the dbt Cloud job to be triggered
    """


DbtCloudTaskParam = DbtCloudTaskDict | DbtCloudTask
